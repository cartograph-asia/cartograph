package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// Handlers serves the public pages and feeds from a views.Source.
type Handlers struct {
	src views.Source
	rd  *Renderer
}

// NewHandlers returns the page handlers.
func NewHandlers(src views.Source, rd *Renderer) *Handlers {
	return &Handlers{src: src, rd: rd}
}

// Routes mounts every page, feed and the static assets on r:
//
//	/  /latest  /latest.rss  /surveyors  /regions
//	/surveyor/{name}[.rss]  /plot/{cell}  /plot/{slug}/{number}  /photo/{id}  /region/{slug}[.rss]
//	/about  /about/plots  /principles  /licence (StaticPages)  /static/*
//
// The account pages (/signin, /signin/link, /signin/name, /settings, /signout) belong to
// internal/auth, which renders through Renderer.RenderFunc.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/", h.home)
	r.Get("/latest", h.latest)
	r.Get("/latest.rss", h.latestFeed)
	r.Get("/surveyors", h.surveyors)
	r.Get("/regions", h.regions)
	r.Get("/surveyor/{name}", h.surveyor)
	r.Get("/plot/{cell}", h.plot)
	r.Get("/plot/{slug}/{number}", h.plotByNumber)
	r.Get("/photo/{id}", h.photo)
	r.Get("/region/{slug}", h.region)
	for _, sp := range StaticPages {
		r.Get(sp.Path, h.static(sp.Path))
	}
	r.Handle(StaticPrefix+"*", h.rd.assets)
}

// NotFound renders the 404 page; mount it with r.NotFound for page paths.
func (h *Handlers) NotFound(w http.ResponseWriter, r *http.Request) {
	h.rd.renderError(w, r, http.StatusNotFound, "There is nothing at this address.")
}

func (h *Handlers) static(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { h.rd.renderStatic(w, r, path) }
}

// fail renders ErrNotFound as 404, ErrGone as 410 and anything else as 500.
func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, views.ErrNotFound) {
		h.NotFound(w, r)
		return
	}
	if errors.Is(err, views.ErrGone) {
		h.rd.renderError(w, r, http.StatusGone, "This photo was removed.")
		return
	}
	h.rd.log.Error("page", "path", r.URL.Path, "err", err)
	h.rd.renderError(w, r, http.StatusInternalServerError, "Something went wrong on our side. Please try again in a moment.")
}

// page reads ?page=. Missing is 1; anything but a positive integer is not found.
func page(r *http.Request) (int, bool) {
	v, ok := r.URL.Query()["page"]
	if !ok {
		return 1, true
	}
	n, err := strconv.Atoi(v[0])
	if err != nil || n < 1 || len(v) > 1 {
		return 0, false
	}
	return n, true
}

// canonical redirects to canon (path with an optional ?page=N) when the
// request spells the path differently (case of a name, slug or id) or says
// ?page=1 explicitly. It reports whether it redirected.
func canonical(w http.ResponseWriter, r *http.Request, canon string) bool {
	if canon == "" {
		return false
	}
	path, query, _ := strings.Cut(canon, "?")
	got := r.URL.Query().Get("page")
	want := strings.TrimPrefix(query, "page=")
	if r.URL.EscapedPath() == path && got == want {
		return false
	}
	http.Redirect(w, r, canon, http.StatusMovedPermanently)
	return true
}

// show renders a page after the canonical-URL check.
func (h *Handlers) show(w http.ResponseWriter, r *http.Request, name string, data views.Based, err error) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if canonical(w, r, data.Base().Canonical) {
		return
	}
	h.rd.Render(w, r, http.StatusOK, name, data)
}

func (h *Handlers) home(w http.ResponseWriter, r *http.Request) {
	d, err := h.src.Home(r.Context())
	h.show(w, r, views.PageHome, d, err)
}

func (h *Handlers) latest(w http.ResponseWriter, r *http.Request) {
	n, ok := page(r)
	if !ok {
		h.NotFound(w, r)
		return
	}
	d, err := h.src.Latest(r.Context(), n)
	h.show(w, r, views.PageLatest, d, err)
}

func (h *Handlers) surveyors(w http.ResponseWriter, r *http.Request) {
	n, ok := page(r)
	if !ok {
		h.NotFound(w, r)
		return
	}
	d, err := h.src.Surveyors(r.Context(), n)
	h.show(w, r, views.PageSurveyors, d, err)
}

func (h *Handlers) regions(w http.ResponseWriter, r *http.Request) {
	d, err := h.src.Regions(r.Context())
	h.show(w, r, views.PageRegions, d, err)
}

func (h *Handlers) surveyor(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if base, ok := strings.CutSuffix(name, ".rss"); ok {
		h.feed(w, r, func() (*views.Feed, error) { return h.src.SurveyorFeed(r.Context(), base) })
		return
	}
	n, ok := page(r)
	if !ok {
		h.NotFound(w, r)
		return
	}
	d, err := h.src.Surveyor(r.Context(), name, n)
	h.show(w, r, views.PageSurveyor, d, err)
}

func (h *Handlers) plot(w http.ResponseWriter, r *http.Request) {
	d, err := h.src.Plot(r.Context(), chi.URLParam(r, "cell"))
	h.show(w, r, views.PagePlot, d, err)
}

// plotByNumber serves a plot under its friendly path /plot/{slug}/{number}.
// The page is the same as at /plot/{cell}, whose path stays the canonical
// link, so the friendly URL keeps showing in the address bar.
func (h *Handlers) plotByNumber(w http.ResponseWriter, r *http.Request) {
	raw := chi.URLParam(r, "number")
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || raw != strconv.Itoa(n) {
		h.NotFound(w, r)
		return
	}
	cell, err := h.src.PlotCell(r.Context(), chi.URLParam(r, "slug"), n)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	d, err := h.src.Plot(r.Context(), cell)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.rd.Render(w, r, http.StatusOK, views.PagePlot, d)
}

func (h *Handlers) photo(w http.ResponseWriter, r *http.Request) {
	d, err := h.src.Photo(r.Context(), chi.URLParam(r, "id"))
	h.show(w, r, views.PagePhoto, d, err)
}

func (h *Handlers) region(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if base, ok := strings.CutSuffix(slug, ".rss"); ok {
		h.feed(w, r, func() (*views.Feed, error) { return h.src.RegionFeed(r.Context(), base) })
		return
	}
	n, ok := page(r)
	if !ok {
		h.NotFound(w, r)
		return
	}
	d, err := h.src.Region(r.Context(), slug, n)
	h.show(w, r, views.PageRegion, d, err)
}

func (h *Handlers) latestFeed(w http.ResponseWriter, r *http.Request) {
	h.feed(w, r, func() (*views.Feed, error) { return h.src.LatestFeed(r.Context()) })
}

// feed renders a feed, redirecting to the canonical feed path first.
func (h *Handlers) feed(w http.ResponseWriter, r *http.Request, get func() (*views.Feed, error)) {
	f, err := get()
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if self := feedSelf(f); r.URL.EscapedPath() != self {
		http.Redirect(w, r, self, http.StatusMovedPermanently)
		return
	}
	h.rd.Render(w, r, http.StatusOK, views.PageFeed, f)
}
