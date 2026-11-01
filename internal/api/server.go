// Package api holds the HTTP handlers and routing for the JSON API.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/storage"
)

// Repo is the subset of the database the handlers need; *db.DB implements it.
type Repo interface {
	ClaimPhoto(ctx context.Context, p db.NewPhoto) (claimed bool, err error)
	Plots(ctx context.Context, cells []string) ([]db.PlotSummary, error)
	Plot(ctx context.Context, cell string) (*db.PlotDetail, error)
	Coverage(ctx context.Context, res int, parents []string) (map[string]int64, error)
}

// Config holds the handler settings.
type Config struct {
	MaxUploadMB int // reject photo files larger than this
	// TilesURL is where the base map's PMTiles archive is served (TILES_URL):
	// a path on this site or a full URL, e.g. on a CDN. GET /api/config hands
	// it to the map page and the small maps, so moving the archive is a
	// change in .env, not in the map's build.
	TilesURL string
	// MaxConcurrentProcessing bounds how many uploads decode images at once;
	// a 48 MP photo needs a few hundred MB while being resized.
	MaxConcurrentProcessing int
	// UploadTimeout replaces the server's read and write timeouts for
	// POST /api/photos (default DefaultUploadTimeout).
	UploadTimeout time.Duration
	// SpoolDir holds uploads while they arrive ("" = os.TempDir).
	SpoolDir string
}

// DefaultUploadTimeout is long enough for a 20 MB photo at about 300 kbit/s,
// a slow mobile connection.
const DefaultUploadTimeout = 10 * time.Minute

// Server serves the API.
type Server struct {
	repo  Repo
	store storage.Store
	cfg   Config
	sem   chan struct{}
	now   func() time.Time

	mu        sync.Mutex
	uploading map[string]bool // Surveyor ids with an upload in progress
}

// New creates a Server.
func New(repo Repo, store storage.Store, cfg Config) *Server {
	if cfg.UploadTimeout <= 0 {
		cfg.UploadTimeout = DefaultUploadTimeout
	}
	if cfg.MaxUploadMB <= 0 {
		cfg.MaxUploadMB = 25
	}
	if cfg.MaxConcurrentProcessing <= 0 {
		cfg.MaxConcurrentProcessing = 2
	}
	if cfg.TilesURL == "" {
		cfg.TilesURL = DefaultTilesURL
	}
	return &Server{
		repo:  repo,
		store: store,
		cfg:   cfg,
		sem:   make(chan struct{}, cfg.MaxConcurrentProcessing),
		now:   time.Now,

		uploading: map[string]bool{},
	}
}

// Handler returns the HTTP handler with all routes and middleware.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(recoverer, requestLogger, cors)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	r.Route("/api", func(r chi.Router) {
		r.Get("/config", s.clientConfig)
		r.Post("/photos", s.createPhoto)
		r.Get("/plots", s.listPlots)
		r.Get("/plots/{cell}", s.getPlot)
		r.Get("/coverage", s.coverage)
	})
	return r
}

// DefaultTilesURL is the archive's path when TILES_URL is not set: Caddy
// serves map/ at /tiles/.
const DefaultTilesURL = "/tiles/asia.pmtiles"

// clientConfig is GET /api/config: what the map needs from the environment.
func (s *Server) clientConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, map[string]string{"tiles_url": s.cfg.TilesURL})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("write response", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			h.Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Observe wraps next in the same panic recovery and request logging as
// Handler. cmd/api uses it for the routes that live outside the JSON API
// (server-rendered pages, sign-in).
func Observe(next http.Handler) http.Handler { return recoverer(requestLogger(next)) }

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec)
			}
			slog.Error("panic in handler", "panic", rec, "path", r.URL.Path, "stack", string(debug.Stack()))
			writeError(w, http.StatusInternalServerError, "internal error")
		}()
		next.ServeHTTP(w, r)
	})
}
