// Command api is the Cartograph backend: the server-rendered pages, magic-link
// sign-in and the JSON API for plots, photos and coverage. It is configured entirely through environment variables
// (see deploy/.env.example).
//
// "api -healthcheck" probes GET /healthz on API_ADDR and exits 0 or 1, so the
// distroless image (no shell, no curl) can still have a container healthcheck.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/api"
	"github.com/cartograph-asia/cartograph/internal/auth"
	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/storage"
	"github.com/cartograph-asia/cartograph/internal/web"
	"github.com/cartograph-asia/cartograph/migrations"
)

type config struct {
	addr        string
	databaseURL string
	auth        auth.Config
	s3          storage.S3Config
	maxUploadMB int
	tilesURL    string // TILES_URL, see api.Config.TilesURL
}

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe GET /healthz on API_ADDR and exit 0 if healthy")
	flag.Parse()
	if *healthcheck {
		if err := probe(envOr("API_ADDR", ":8000")); err != nil {
			fmt.Fprintln(os.Stderr, "unhealthy:", err)
			os.Exit(1)
		}
		return
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := db.Connect(ctx, cfg.databaseURL, 30*time.Second)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := database.Migrate(ctx, migrations.FS); err != nil {
		return err
	}

	store, err := storage.NewS3(cfg.s3)
	if err != nil {
		return err
	}
	checkBucket(ctx, store, cfg.s3.Bucket)

	handler, err := routes(cfg, database, store)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute, // large uploads on slow mobile connections
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.addr)
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down")
		sctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(sctx)
	}
	return nil
}

// routes puts the pages, the sign-in routes and the JSON API on one router.
// The auth middleware wraps all of them, so the upload handler sees the
// signed-in Surveyor and every page can show "Sign in" or the name.
func routes(cfg config, database *db.DB, store storage.Store) (http.Handler, error) {
	acfg := cfg.auth
	rd, err := web.NewRenderer(web.Config{BaseURL: acfg.BaseURL, ImgBaseURL: cfg.s3.BaseURL})
	if err != nil {
		return nil, err
	}
	a := auth.New(acfg, auth.NewPGStore(database.Pool), auth.NewMailer(acfg.SMTP), rd.RenderFunc(), database.Views())
	pages := web.NewHandlers(database.Views(), rd)
	apiH := api.New(database, store, api.Config{MaxUploadMB: cfg.maxUploadMB, TilesURL: cfg.tilesURL}).Handler()

	r := chi.NewRouter()
	r.Use(a.Middleware)
	// The JSON API keeps its own router, logging and JSON errors; chi
	// prefers the static /api/me from the auth routes over this wildcard.
	r.Handle("/api/*", apiH)
	r.Handle("/healthz", apiH)
	r.Group(func(r chi.Router) {
		r.Use(api.Observe)
		a.Routes(r)
		pages.Routes(r)
	})
	r.NotFound(api.Observe(http.HandlerFunc(pages.NotFound)).ServeHTTP)
	return r, nil
}

// probe requests /healthz from the server listening on addr.
func probe(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("API_ADDR %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + net.JoinHostPort(host, port) + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET /healthz: %s", resp.Status)
	}
	return nil
}

// checkBucket logs whether the bucket exists. It never creates it; the
// deployment does that (garage-init in compose).
func checkBucket(ctx context.Context, store *storage.S3, bucket string) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ok, err := store.BucketExists(cctx)
	switch {
	case err != nil:
		slog.Warn("could not check photo bucket; uploads will fail until storage is reachable", "bucket", bucket, "err", err)
	case !ok:
		slog.Warn("photo bucket does not exist; uploads will fail until it is created", "bucket", bucket)
	default:
		slog.Info("photo bucket ok", "bucket", bucket)
	}
}

func loadConfig() (config, error) {
	var missing []string
	need := func(name string) string {
		v := strings.TrimSpace(os.Getenv(name))
		if v == "" {
			missing = append(missing, name)
		}
		return v
	}
	c := config{
		addr:        envOr("API_ADDR", ":8000"),
		databaseURL: need("DATABASE_URL"),
		s3: storage.S3Config{
			Endpoint:  need("S3_ENDPOINT"),
			Bucket:    need("S3_BUCKET"),
			AccessKey: need("S3_ACCESS_KEY"),
			SecretKey: need("S3_SECRET_KEY"),
			Region:    envOr("S3_REGION", "us-east-1"),
			BaseURL:   need("IMG_BASE_URL"),
		},
	}
	var errs []error
	if len(missing) > 0 {
		errs = append(errs, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", ")))
	}
	var err error
	// Checked here, before the database is touched: a public site refuses
	// to start without SMTP rather than log sign-in links.
	if c.auth, err = auth.ConfigFromEnv(os.Getenv); err != nil {
		errs = append(errs, err)
	}
	if c.s3.PathStyle, err = envBool("S3_USE_PATH_STYLE", false); err != nil {
		errs = append(errs, err)
	}
	if c.s3.UseSSL, err = envBool("S3_USE_SSL", false); err != nil {
		errs = append(errs, err)
	}
	if c.maxUploadMB, err = strconv.Atoi(envOr("MAX_UPLOAD_MB", "25")); err != nil || c.maxUploadMB <= 0 {
		errs = append(errs, errors.New("MAX_UPLOAD_MB must be a positive integer"))
	}
	c.tilesURL = envOr("TILES_URL", api.DefaultTilesURL)
	if !strings.HasPrefix(c.tilesURL, "/") && !strings.HasPrefix(c.tilesURL, "https://") && !strings.HasPrefix(c.tilesURL, "http://") {
		errs = append(errs, fmt.Errorf("TILES_URL must be a path starting with / or an http(s) URL, got %q", c.tilesURL))
	}
	if strings.Contains(c.s3.Endpoint, "://") {
		errs = append(errs, errors.New("S3_ENDPOINT must be host:port without a scheme; use S3_USE_SSL for https"))
	}
	return c, errors.Join(errs...)
}

func envOr(name, def string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return def
}

func envBool(name string, def bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false, got %q", name, v)
	}
	return b, nil
}
