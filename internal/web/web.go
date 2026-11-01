// Package web renders Cartograph's server-side pages: the front page, the
// lists, profiles, plots, photos and regions, their RSS feeds, the account
// forms that internal/auth serves, and the static assets they need.
//
// Data comes from a views.Source; nothing here talks to the database or the
// object store. Image URLs are built from Config.ImgBaseURL + object key.
package web

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cartograph-asia/cartograph/internal/views"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

//go:embed content/principles.md
var principlesMD string

//go:embed content/about.html
var aboutHTML template.HTML

//go:embed content/licence.html
var licenceHTML template.HTML

// Config configures the renderer.
type Config struct {
	// BaseURL is the public origin of the site without a trailing slash, e.g.
	// "https://cartograph.asia" (BASE_URL). Canonical links, Open Graph and
	// feeds use it for absolute URLs. Required.
	BaseURL string
	// ImgBaseURL is the public base of photo objects (IMG_BASE_URL); an image
	// URL is ImgBaseURL + "/" + key. Required.
	ImgBaseURL string
	// MinimapScript is the URL of the lazily imported small-map module from
	// the web/ build. Defaults to DefaultMinimapScript.
	MinimapScript string
	// CSRF returns the request's CSRF token for POST forms; nil or "" leaves
	// Page.CSRF as the handler set it. Set by the accounts middleware.
	CSRF func(*http.Request) string
	// Now is the clock for Page.Now when the Source left it zero; time.Now if nil.
	Now func() time.Time
	// Logger receives render errors; slog.Default() if nil.
	Logger *slog.Logger
}

// DefaultMinimapScript is where the web/ build (base /map/) puts the
// small-map entry under a stable name.
const DefaultMinimapScript = "/map/assets/minimap.js"

// StaticPrefix is the URL prefix of the embedded assets.
const StaticPrefix = "/static/"

// zone is the display time zone. Thailand has no daylight saving time, so a
// fixed zone needs no tzdata in the distroless image.
var zone = time.FixedZone("ICT", 7*60*60)

// Renderer renders pages. Its Render method is a views.RenderFunc.
type Renderer struct {
	cfg    Config
	log    *slog.Logger
	pages  map[string]*template.Template
	assets *assets
	static map[string]template.HTML // rendered bodies of the static pages
}

// Static pages that are not views.Page* constants.
const (
	pageAbout      = "about"
	pagePrinciples = "principles"
	pageLicence    = "licence"
)

// NewRenderer parses the embedded templates.
func NewRenderer(cfg Config) (*Renderer, error) {
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if cfg.BaseURL == "" || !strings.Contains(cfg.BaseURL, "://") {
		return nil, errors.New("web: BaseURL must be an absolute URL such as https://cartograph.asia")
	}
	if strings.TrimSpace(cfg.ImgBaseURL) == "" {
		return nil, errors.New("web: ImgBaseURL is required")
	}
	if cfg.MinimapScript == "" {
		cfg.MinimapScript = DefaultMinimapScript
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	rd := &Renderer{cfg: cfg, log: cfg.Logger, pages: map[string]*template.Template{}}
	if rd.log == nil {
		rd.log = slog.Default()
	}
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}
	if rd.assets, err = loadAssets(sub); err != nil {
		return nil, err
	}

	base, err := template.New("").Funcs(rd.funcs()).ParseFS(templateFS, "templates/layout.html", "templates/partials.html")
	if err != nil {
		return nil, fmt.Errorf("web: parse layout: %w", err)
	}
	names := []string{
		views.PageHome, views.PageLatest, views.PageSurveyors, views.PageRegions, views.PageSurveyor,
		views.PagePlot, views.PagePhoto, views.PageRegion, views.PageSignIn, views.PageCheckEmail,
		views.PageChooseName, views.PageEditProfile, views.PageError, "static",
	}
	for _, name := range names {
		t, err := base.Clone()
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(templateFS, "templates/"+name+".html"); err != nil {
			return nil, fmt.Errorf("web: parse %s: %w", name, err)
		}
		rd.pages[name] = t
	}
	rd.static = map[string]template.HTML{
		pageAbout:      aboutHTML,
		pagePrinciples: markdown(principlesMD),
		pageLicence:    licenceHTML,
	}
	return rd, nil
}

// RenderFunc returns Render as a views.RenderFunc for handlers in other packages.
func (rd *Renderer) RenderFunc() views.RenderFunc { return rd.Render }

// Render writes page with data and status. data is the page struct named by
// page (see the views.Page* constants). Viewer comes from the request
// context, CSRF from Config.CSRF, Now from the clock when the Source left it
// zero. A template error becomes a plain 500.
func (rd *Renderer) Render(w http.ResponseWriter, r *http.Request, status int, page string, data any) {
	if b, ok := data.(views.Based); ok {
		rd.fill(r, b.Base())
	}
	var buf bytes.Buffer
	var err error
	ctype := "text/html; charset=utf-8"
	switch {
	case page == views.PageFeed:
		f, ok := data.(*views.Feed)
		if !ok {
			err = fmt.Errorf("feed data is %T", data)
			break
		}
		ctype = "application/rss+xml; charset=utf-8"
		err = rd.writeFeed(&buf, f)
	case rd.pages[page] != nil:
		err = rd.pages[page].ExecuteTemplate(&buf, "layout", data)
	default:
		err = fmt.Errorf("unknown page %q", page)
	}
	if err != nil {
		rd.log.Error("render", "page", page, "path", r.URL.Path, "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", ctype)
	h.Set("X-Content-Type-Options", "nosniff")
	if p, ok := data.(views.Based); ok && p.Base().Viewer != nil {
		h.Set("Cache-Control", "private, no-cache")
	}
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(buf.Bytes())
	}
}

func (rd *Renderer) fill(r *http.Request, p *views.Page) {
	if v := views.ViewerFrom(r.Context()); v != nil {
		p.Viewer = v
	}
	if rd.cfg.CSRF != nil {
		if t := rd.cfg.CSRF(r); t != "" {
			p.CSRF = t
		}
	}
	if p.Now.IsZero() {
		p.Now = rd.cfg.Now()
	}
}

// staticPage is a page whose body is fixed text (About, Principles, licence).
type staticPage struct {
	views.Page
	Body template.HTML
}

func (rd *Renderer) renderStatic(w http.ResponseWriter, r *http.Request, name, title, desc string) {
	p := &staticPage{Page: views.Page{Title: title, Canonical: "/" + name, Description: desc}, Body: rd.static[name]}
	rd.Render(w, r, http.StatusOK, "static", p)
}

// renderError renders the error page with status and message.
func (rd *Renderer) renderError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	title := http.StatusText(status)
	if status == http.StatusNotFound {
		title = "Not found"
	}
	rd.Render(w, r, status, views.PageError, &views.Error{
		Page:   views.Page{Title: title, NoIndex: true},
		Status: status, Message: msg,
	})
}
