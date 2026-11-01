// Command webpreview serves every server-rendered page from the fixture data
// in internal/views/viewstest, for checking templates and CSS in a browser
// without a database or object store. Photos are generated placeholder
// JPEGs; fixture keys exist in no bucket.
//
//	go run ./cmd/webpreview            # http://127.0.0.1:8093
//	(cd web && npx vite build --base /map/)   # optional: small maps
//
// /preview/as?viewer=adrian|new|none switches the signed-in viewer (cookie).
// Photos are crops of the JPEGs in -photos (testdata/ by default), so the
// pages look like pages with photos; without them, calm generated images.
// The account pages (/signin, /check-email, /signin/link, /signin/name,
// /settings) show their
// fixtures; their forms post to stubs. It never binds :8080.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/jpeg"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/views"
	"github.com/cartograph-asia/cartograph/internal/views/viewstest"
	"github.com/cartograph-asia/cartograph/internal/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8093", "listen address")
	dist := flag.String("dist", "web/dist", "web build (vite build --base /map/) for the small maps; skipped if missing")
	tiles := flag.String("tiles", "map/asia.pmtiles", "PMTiles archive for the small maps; skipped if missing")
	perPage := flag.Int("per-page", 10, "photos per page, low to show pagination with 30 fixture photos")
	photoDir := flag.String("photos", "testdata", "directory of JPEGs cropped into the fixture photos; generated images if empty")
	flag.Parse()
	if strings.HasSuffix(*addr, ":8080") {
		fmt.Fprintln(os.Stderr, "webpreview: :8080 belongs to the compose stack")
		os.Exit(2)
	}

	src := viewstest.NewFake()
	src.PerPage = *perPage
	base := "http://" + *addr
	rd, err := web.NewRenderer(web.Config{
		BaseURL:    base,
		ImgBaseURL: base + "/img",
	})
	if err != nil {
		slog.Error("renderer", "err", err)
		os.Exit(1)
	}
	pages := web.NewHandlers(src, rd)

	r := chi.NewRouter()
	r.Use(viewerFromCookie, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(views.WithCSRF(r.Context(), "preview-csrf")))
		})
	})
	pages.Routes(r)
	r.NotFound(pages.NotFound)
	accountPages(r, rd.RenderFunc())

	ph := newPlaceholders(src, *photoDir)
	r.Get("/img/*", ph.ServeHTTP)
	r.Get("/api/coverage", coverage(src))
	if fi, err := os.Stat(*dist); err == nil && fi.IsDir() {
		fs := http.FileServer(http.Dir(*dist))
		r.Handle("/map/*", http.StripPrefix("/map", fs))
		r.Handle("/fonts/*", fs) // web/style.json before the /map/ base path
	} else {
		slog.Warn("no web build; small maps keep their fallback", "dist", *dist)
	}
	if _, err := os.Stat(*tiles); err == nil {
		r.Get("/tiles/asia.pmtiles", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, *tiles) })
	} else {
		slog.Warn("no PMTiles archive; small maps have no base map", "tiles", *tiles)
	}

	slog.Info("preview", "url", base)
	srv := &http.Server{Addr: *addr, Handler: r, ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
}

// viewerFromCookie signs the preview in as a fixture viewer.
func viewerFromCookie(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("preview_viewer"); err == nil {
			switch c.Value {
			case "adrian":
				r = r.WithContext(views.WithViewer(r.Context(), viewstest.Viewer()))
			case "new":
				r = r.WithContext(views.WithViewer(r.Context(), viewstest.NewViewer()))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func accountPages(r chi.Router, render views.RenderFunc) {
	r.Get("/preview/as", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "preview_viewer", Value: r.URL.Query().Get("viewer"), Path: "/"})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	r.Get("/signin", func(w http.ResponseWriter, r *http.Request) {
		p := viewstest.SignIn()
		if r.URL.Query().Get("error") == "" {
			p.Error, p.Email = "", ""
		}
		render(w, r, http.StatusOK, views.PageSignIn, p)
	})
	r.Post("/signin", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/check-email", http.StatusSeeOther) })
	r.Get("/check-email", func(w http.ResponseWriter, r *http.Request) {
		render(w, r, http.StatusOK, views.PageCheckEmail, viewstest.CheckEmail())
	})
	r.Get("/signin/link", func(w http.ResponseWriter, r *http.Request) {
		render(w, r, http.StatusOK, views.PageConfirmSignIn, &views.ConfirmSignIn{
			Page: views.Page{Title: "Sign in", NoIndex: true}, Token: r.URL.Query().Get("token")})
	})
	r.Get("/signin/name", func(w http.ResponseWriter, r *http.Request) {
		render(w, r, http.StatusOK, views.PageChooseName, viewstest.ChooseName())
	})
	r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
		p := viewstest.EditProfile()
		p.Saved = r.URL.Query().Get("saved") == "1"
		render(w, r, http.StatusOK, views.PageEditProfile, p)
	})
	for _, p := range []string{"/signin/link", "/signin/name", "/settings", "/signout"} {
		r.Post(p, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/", http.StatusSeeOther) })
	}
}

// coverage answers GET /api/coverage from the fixture photos.
func coverage(src *viewstest.Fake) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, _ := strconv.Atoi(r.URL.Query().Get("res"))
		feed, _ := src.LatestFeed(r.Context())
		claimed := map[string]int{}
		seen := map[string]bool{}
		for _, p := range feed.Items {
			if seen[p.Plot.Cell] {
				continue
			}
			seen[p.Plot.Cell] = true
			if parent, err := h3.Parent(p.Plot.Cell, res); err == nil {
				claimed[parent]++
			}
		}
		var out []string
		for _, c := range strings.Split(r.URL.Query().Get("cells"), ",") {
			if n := claimed[c]; n > 0 {
				total, _ := h3.ChildrenCount(c, h3.PlotRes)
				out = append(out, fmt.Sprintf(`{"cell":%q,"surveyed":%d,"total":%d}`, c, n, total))
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"cells":[%s]}`, strings.Join(out, ","))
	}
}

// placeholders renders a calm two-tone JPEG per key, in the variant's size
// and the fixture photo's orientation.
type placeholders struct {
	mu     sync.Mutex
	cache  map[string][]byte
	shapes map[string][2]int // photo id -> width, height of the large variant
	photos []image.Image     // real photos to crop from; generated images if none
}

func newPlaceholders(src *viewstest.Fake, dir string) *placeholders {
	p := &placeholders{cache: map[string][]byte{}, shapes: map[string][2]int{}}
	feed, _ := src.LatestFeed(context.Background())
	for _, it := range feed.Items {
		p.shapes[it.ID] = [2]int{it.Width, it.Height}
	}
	if dir != "" {
		files, _ := filepath.Glob(filepath.Join(dir, "*.jp*g"))
		for _, f := range files {
			img, err := imaging.Open(f, imaging.AutoOrientation(true))
			if err != nil {
				slog.Warn("preview photo", "file", f, "err", err)
				continue
			}
			p.photos = append(p.photos, imaging.Fit(img, 2000, 2000, imaging.Lanczos))
		}
		slog.Info("preview photos", "dir", dir, "photos", len(p.photos))
	}
	return p
}

var keyRe = regexp.MustCompile(`([0-9a-f-]{36})_(thumb|medium|large)\.jpg$`)

func (p *placeholders) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/img/")
	m := keyRe.FindStringSubmatch(key)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	p.mu.Lock()
	data, ok := p.cache[key]
	if !ok {
		data = p.draw(m[1], m[2])
		p.cache[key] = data
	}
	p.mu.Unlock()
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeContent(w, r, filepath.Base(key), time.Time{}, bytes.NewReader(data))
}

func (p *placeholders) draw(id, variant string) []byte {
	long := map[string]int{"thumb": 480, "medium": 1200, "large": 2000}[variant]
	wh := p.shapes[id]
	w, h := long, long*3/4
	if wh[0] > 0 && wh[1] > wh[0] {
		w, h = long*wh[0]/wh[1], long
	}
	f := fnv.New32a()
	f.Write([]byte(id))
	s := f.Sum32()
	if len(p.photos) > 0 {
		// A different crop of one of the photos per id, mirrored for half of
		// them, so a grid of 30 does not look like three photos repeated.
		src := p.photos[s%uint32(len(p.photos))]
		anchors := []imaging.Anchor{imaging.Center, imaging.Left, imaging.Right, imaging.Top, imaging.Bottom, imaging.TopLeft, imaging.BottomRight}
		zoom := 1 + float64(s>>8%40)/100 // crop 60-100 % of the photo
		b := src.Bounds()
		cw, ch := int(float64(b.Dx())/zoom), int(float64(b.Dy())/zoom)
		img := imaging.Fill(imaging.CropAnchor(src, cw, ch, anchors[s>>4%uint32(len(anchors))]), w, h, imaging.Center, imaging.Lanczos)
		if s>>12%2 == 1 {
			img = imaging.FlipH(img)
		}
		var buf bytes.Buffer
		_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 82})
		return buf.Bytes()
	}
	sky := color.RGBA{uint8(150 + s%60), uint8(180 + s>>8%50), uint8(200 + s>>16%50), 255}
	ground := color.RGBA{uint8(60 + s>>4%80), uint8(100 + s>>12%70), uint8(60 + s>>20%50), 255}
	horizon := 0.45 + float64(s%20)/100
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		t := float64(y) / float64(h)
		c := sky
		if t > horizon {
			c = ground
		}
		shade := 1 - 0.25*t
		row := color.RGBA{uint8(float64(c.R) * shade), uint8(float64(c.G) * shade), uint8(float64(c.B) * shade), 255}
		for x := range w {
			img.SetRGBA(x, y, row)
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}
