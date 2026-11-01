package web

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/views"
	"github.com/cartograph-asia/cartograph/internal/views/viewstest"
)

const (
	testBase = "https://cartograph.asia"
	testImg  = "https://img.cartograph.asia"
)

func newRenderer(t *testing.T) *Renderer {
	t.Helper()
	rd, err := NewRenderer(Config{BaseURL: testBase + "/", ImgBaseURL: testImg})
	if err != nil {
		t.Fatal(err)
	}
	return rd
}

func newServer(t *testing.T, perPage int) http.Handler {
	t.Helper()
	rd := newRenderer(t)
	src := viewstest.NewFake()
	src.PerPage = perPage
	r := chi.NewRouter()
	h := NewHandlers(src, rd)
	h.Routes(r)
	r.NotFound(h.NotFound)
	return r
}

type resp struct {
	status int
	header http.Header
	body   string
}

func get(t *testing.T, h http.Handler, target string, ctx ...context.Context) resp {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if len(ctx) > 0 {
		req = req.WithContext(ctx[0])
	}
	req = req.WithContext(views.WithCSRF(req.Context(), "tok-123"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	b, _ := io.ReadAll(rec.Result().Body)
	return resp{rec.Code, rec.Header(), string(b)}
}

func render(t *testing.T, page string, data any, ctx ...context.Context) string {
	t.Helper()
	rd := newRenderer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if len(ctx) > 0 {
		req = req.WithContext(ctx[0])
	}
	req = req.WithContext(views.WithCSRF(req.Context(), "tok-123"))
	rec := httptest.NewRecorder()
	rd.Render(rec, req, http.StatusOK, page, data)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status %d: %s", page, rec.Code, rec.Body)
	}
	return rec.Body.String()
}

func must(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(body, w) {
			t.Errorf("missing %q", w)
		}
	}
}

func mustNot(t *testing.T, body string, nots ...string) {
	t.Helper()
	for _, w := range nots {
		if strings.Contains(body, w) {
			t.Errorf("unexpected %q", w)
		}
	}
}

var (
	imgTag    = regexp.MustCompile(`<img\b[^>]*>`)
	tagRe     = regexp.MustCompile(`(?s)<script.*?</script>|<[^>]+>`)
	cellRe    = regexp.MustCompile(`\b8[0-9a-f]{14}\b`)
	detailsRe = regexp.MustCompile(`(?s)<p class="details">.*?</p>`)
	wordsBad  = regexp.MustCompile(`(?i)\b(hex|hexagons?|squares?)\b`)
)

// checkHTML runs the structural rules every page must follow.
func checkHTML(t *testing.T, name, body string) {
	t.Helper()
	if !strings.HasPrefix(body, "<!doctype html>") || !strings.Contains(body, "</html>") {
		t.Errorf("%s: not a complete document", name)
	}
	high := 0
	for _, img := range imgTag.FindAllString(body, -1) {
		if !strings.Contains(img, ` width="`) || !strings.Contains(img, ` height="`) || !strings.Contains(img, ` alt=`) {
			t.Errorf("%s: img without width/height/alt: %s", name, img)
		}
		if strings.Contains(img, `fetchpriority="high"`) {
			high++
		} else if !strings.Contains(img, `loading="lazy"`) && !strings.Contains(img, `logo`) {
			t.Errorf("%s: img neither high priority nor lazy: %s", name, img)
		}
	}
	if high > 1 {
		t.Errorf("%s: %d images with fetchpriority=high", name, high)
	}
	// The plot page's details line is the one place a cell id may appear.
	text := tagRe.ReplaceAllString(detailsRe.ReplaceAllString(body, ""), " ")
	if m := wordsBad.FindString(text); m != "" {
		t.Errorf("%s: forbidden word %q in visible text", name, m)
	}
	if m := cellRe.FindString(text); m != "" {
		t.Errorf("%s: bare cell id %s in visible text", name, m)
	}
	if strings.Contains(body, "<no value>") || strings.Contains(body, "ZgotmplZ") {
		t.Errorf("%s: template artefact in output", name)
	}
}

func TestEveryPageRenders(t *testing.T) {
	ctx := views.WithViewer(context.Background(), viewstest.Viewer())
	pages := []struct {
		name  string
		page  string
		data  any
		wants []string
	}{
		{"home", views.PageHome, viewstest.Home(), []string{
			"Latest surveys", "Plot of the week", viewstest.FeaturedText, "New Surveyors", "Regions",
			`<b>13</b> plots surveyed`, `href="/surveyors"`, `href="/regions"`, "first survey", "Chon Buri",
			`<span lang="th" class="th">ชลบุรี</span>`, `href="/latest">More surveys`, "Welcome",
		}},
		{"latest", views.PageLatest, viewstest.Latest(10), []string{"Latest surveys", `rel="next" href="/latest?page=2"`, "Page 1 of 3"}},
		{"surveyors", views.PageSurveyors, viewstest.Surveyors(), []string{"ploy88", "beach-walker", "plots claimed"}},
		{"regions", views.PageRegions, viewstest.Regions(), []string{"Rayong", "Bangkok", "0 of 4,818 plots"}},
		{"surveyor", views.PageSurveyor, viewstest.Surveyor(), []string{
			"<h1>adrian</h1>", "Walking Pattaya", "home region", `data-minimap="plots"`, `<svg class="cells"`,
			`href="/surveyor/adrian.rss"`, "Edit your profile", "plots claimed",
		}},
		{"placeholder", views.PageSurveyor, viewstest.PlaceholderSurveyor(), []string{"<h1>beach-walker</h1>"}},
		{"plot", views.PagePlot, viewstest.Plot(), []string{
			"<h1>Pattaya · Chon Buri 2417", "first surveyed by", "H3 index <code>88658bd11dfffff</code>", `href="/plot/chon-buri/2417"`, "beach-walker", `href="/map?cell=88658bd11dfffff"`,
			"Survey this plot", `data-focus="88658bd11dfffff"`, `data-surveyed="1,`, `class="s f"`,
		}},
		{"blank plot", views.PagePlot, viewstest.BlankPlot(), []string{"Nobody has surveyed this plot yet", `name="robots" content="noindex"`, "Survey this plot"}},
		{"photo", views.PagePhoto, viewstest.Photo(), []string{">Pattaya · Chon Buri 2417</a></h1>", "CC BY-SA 4.0", `rel="prev"`, `rel="next"`, "Photo 3 of", "Full size"}},
		{"prototype photo", views.PagePhoto, viewstest.PrototypePhoto(), []string{"first survey of this plot", `width="2000" height="1500"`}},
		{"region", views.PageRegion, viewstest.Region(), []string{
			"<h1>Chon Buri", "0.20%", "5,921 plots", `data-minimap="coverage"`, `data-bounds="100.68000,12.50000,101.73000,13.59000"`,
			"Most active, last 30 days", "Most active, all time", `href="/region/chon-buri.rss"`, "Open Chon Buri on the map",
		}},
		{"signin", views.PageSignIn, viewstest.SignIn(), []string{`action="/signin"`, `name="csrf" value="tok-123"`, `name="next" value="/map"`, `type="email"`, `value="adrian@example"`, "That does not look"}},
		{"check-email", views.PageCheckEmail, viewstest.CheckEmail(), []string{"adrian@example.org"}},
		{"choose-name", views.PageChooseName, viewstest.ChooseName(), []string{`action="/signin/name"`, `pattern="[A-Za-z0-9\-]{3,30}"`, `maxlength="30"`, "That name is taken"}},
		{"edit-profile", views.PageEditProfile, viewstest.EditProfile(), []string{`action="/settings"`, "Saved.", `<option value="chon-buri" selected>`, `maxlength="500"`, "Walking Pattaya"}},
		{"confirm-signin", views.PageConfirmSignIn, &views.ConfirmSignIn{Page: views.Page{Title: "Sign in", NoIndex: true}, Token: "abc&def"},
			[]string{`action="/signin/link"`, `name="token" value="abc&amp;def"`, `name="csrf" value="tok-123"`, ">Sign in</button>"}},
		{"error", views.PageError, viewstest.NotFound(), []string{"There is nothing at this address."}},
	}
	for _, p := range pages {
		t.Run(p.name, func(t *testing.T) {
			body := render(t, p.page, p.data, ctx)
			checkHTML(t, p.name, body)
			must(t, body, p.wants...)
			must(t, body, `href="/map">Start surveying`, `href="/surveyor/adrian">adrian</a>`, `action="/signout"`,
				`href="/principles"`, "hello@cartograph.asia", "@cartographasia", "@cartograph.asia", "ODbL")
		})
	}
}

func TestAnonymousMasthead(t *testing.T) {
	body := render(t, views.PageHome, viewstest.Home())
	must(t, body, `<a href="/signin">Sign in</a>`)
	mustNot(t, body, "Sign out", "Edit your profile")
	body = render(t, views.PageSurveyor, viewstest.Surveyor())
	mustNot(t, body, "Edit your profile")
	// A viewer without a name is asked to choose one.
	body = render(t, views.PageChooseName, viewstest.ChooseName())
	must(t, body, `href="/signin/name">Choose your name`)
}

func TestRoutes(t *testing.T) {
	h := newServer(t, 0)
	for _, path := range []string{
		"/", "/latest", "/surveyors", "/regions", "/surveyor/adrian", "/surveyor/beach-walker",
		"/plot/88658bd11dfffff", "/plot/88658bd157fffff", "/photo/" + viewstest.FeaturedPhotoID,
		"/region/chon-buri", "/region/bangkok", "/region/rayong", "/about", "/principles", "/licence",
	} {
		r := get(t, h, path)
		if r.status != http.StatusOK {
			t.Errorf("%s: status %d", path, r.status)
			continue
		}
		if ct := r.header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("%s: content type %q", path, ct)
		}
		checkHTML(t, path, r.body)
	}
}

func TestNotFound(t *testing.T) {
	h := newServer(t, 10)
	for _, path := range []string{
		"/surveyor/nobody", "/surveyor/nobody.rss", "/plot/not-a-cell", "/plot/8928308280fffff",
		"/photo/nope", "/region/atlantis", "/region/atlantis.rss", "/latest?page=4", "/latest?page=abc",
		"/latest?page=0", "/latest?page=-1", "/surveyor/adrian?page=9", "/region/chon-buri?page=x", "/nowhere",
		"/static/style.00000000.css",
	} {
		r := get(t, h, path)
		if r.status != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", path, r.status)
		}
		if !strings.HasPrefix(path, "/static/") {
			must(t, r.body, "<h1>Not found</h1>", `content="noindex"`)
		}
	}
}

func TestPagination(t *testing.T) {
	h := newServer(t, 10)
	p1 := get(t, h, "/latest")
	must(t, p1.body, `rel="next" href="/latest?page=2"`, "Page 1 of 3")
	mustNot(t, p1.body, `rel="prev"`)
	p2 := get(t, h, "/latest?page=2")
	must(t, p2.body, `rel="prev" href="/latest"`, `rel="next" href="/latest?page=3"`, `href="https://cartograph.asia/latest?page=2"`)
	p3 := get(t, h, "/latest?page=3")
	must(t, p3.body, `rel="prev" href="/latest?page=2"`, "Page 3 of 3")
	mustNot(t, p3.body, `rel="next"`)
	if strings.Count(p2.body, `<li><a class="ph"`) != 10 {
		t.Error("page 2 does not have 10 photos")
	}
	s2 := get(t, h, "/surveyor/adrian?page=2")
	must(t, s2.body, `rel="prev" href="/surveyor/adrian"`)
	if strings.Contains(p1.body, "scroll") {
		t.Error("infinite scroll?")
	}
}

func TestCanonicalRedirects(t *testing.T) {
	h := newServer(t, 10)
	cases := map[string]string{
		"/surveyor/ADRIAN":        "/surveyor/adrian",
		"/surveyor/Adrian?page=2": "/surveyor/adrian?page=2",
		"/surveyor/mali":          "/surveyor/Mali",
		"/region/Chon-Buri":       "/region/chon-buri",
		"/latest?page=1":          "/latest",
		"/plot/88658BD11DFFFFF":   "/plot/88658bd11dfffff",
		"/surveyor/ADRIAN.rss":    "/surveyor/adrian.rss",
		"/region/CHON-BURI.rss":   "/region/chon-buri.rss",
	}
	for from, to := range cases {
		r := get(t, h, from)
		if r.status != http.StatusMovedPermanently || r.header.Get("Location") != to {
			t.Errorf("%s: %d -> %q, want 301 -> %q", from, r.status, r.header.Get("Location"), to)
		}
	}
	// Unrelated query parameters do not redirect.
	if r := get(t, h, "/latest?utm_source=x"); r.status != http.StatusOK {
		t.Errorf("/latest?utm_source: %d", r.status)
	}
}

type feedDoc struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel struct {
		Title         string     `xml:"title"`
		Links         []feedLink `xml:"link"` // the RSS link and the atom:link
		Description   string     `xml:"description"`
		LastBuildDate string     `xml:"lastBuildDate"`
		Items         []struct {
			Title string `xml:"title"`
			Link  string `xml:"link"`
			GUID  struct {
				IsPermaLink string `xml:"isPermaLink,attr"`
				Value       string `xml:",chardata"`
			} `xml:"guid"`
			PubDate     string `xml:"pubDate"`
			Description string `xml:"description"`
		} `xml:"item"`
	} `xml:"channel"`
}

type feedLink struct {
	XMLName xml.Name
	Href    string `xml:"href,attr"`
	Rel     string `xml:"rel,attr"`
	Type    string `xml:"type,attr"`
	Value   string `xml:",chardata"`
}

// links splits the channel's RSS <link> from its atom:link elements.
func links(ls []feedLink) (rss string, atom []feedLink) {
	for _, l := range ls {
		if l.XMLName.Space == "http://www.w3.org/2005/Atom" {
			atom = append(atom, l)
		} else {
			rss = l.Value
		}
	}
	return rss, atom
}

func TestFeeds(t *testing.T) {
	h := newServer(t, 0)
	cases := map[string]struct {
		link  string
		items int
	}{
		"/latest.rss":             {testBase + "/latest", 30}, // every fixture photo
		"/region/chon-buri.rss":   {testBase + "/region/chon-buri", 0},
		"/surveyor/adrian.rss":    {testBase + "/surveyor/adrian", 0},
		"/region/rayong.rss":      {testBase + "/region/rayong", 0},
		"/surveyor/ploy88.rss":    {testBase + "/surveyor/ploy88", 0},
		"/surveyor/somchai-k.rss": {testBase + "/surveyor/somchai-k", 0},
	}
	for path, want := range cases {
		r := get(t, h, path)
		if r.status != http.StatusOK {
			t.Errorf("%s: status %d", path, r.status)
			continue
		}
		if ct := r.header.Get("Content-Type"); ct != "application/rss+xml; charset=utf-8" {
			t.Errorf("%s: content type %q", path, ct)
		}
		if !strings.HasPrefix(r.body, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Errorf("%s: no XML declaration", path)
		}
		var d feedDoc
		if err := xml.Unmarshal([]byte(r.body), &d); err != nil {
			t.Errorf("%s: does not parse: %v", path, err)
			continue
		}
		c := d.Channel
		link, atom := links(c.Links)
		if d.Version != "2.0" || c.Title == "" || c.Description == "" || link != want.link {
			t.Errorf("%s: channel %q %q %q %q", path, d.Version, c.Title, c.Description, link)
		}
		if _, err := time.Parse(time.RFC1123Z, c.LastBuildDate); err != nil {
			t.Errorf("%s: lastBuildDate %q: %v", path, c.LastBuildDate, err)
		}
		if len(atom) != 1 || atom[0].Rel != "self" || atom[0].Href != testBase+path || atom[0].Type != "application/rss+xml" {
			t.Errorf("%s: atom:link %+v", path, atom)
		}
		if (len(c.Items) == 0 && path != "/region/rayong.rss") || len(c.Items) > views.FeedItems || (want.items > 0 && len(c.Items) != want.items) {
			t.Errorf("%s: %d items", path, len(c.Items))
		}
		var prev time.Time
		for i, it := range c.Items {
			pub, err := time.Parse(time.RFC1123Z, it.PubDate)
			if err != nil {
				t.Errorf("%s item %d: pubDate %q", path, i, it.PubDate)
			}
			if i > 0 && pub.After(prev) {
				t.Errorf("%s: items not newest first", path)
			}
			prev = pub
			if it.Title == "" || !strings.HasPrefix(it.Link, testBase+"/photo/") || it.GUID.Value != it.Link || it.GUID.IsPermaLink != "true" {
				t.Errorf("%s item %d: %+v", path, i, it)
			}
			if !strings.HasPrefix(it.Description, `<a href="`+it.Link+`"><img src="`+testImg+"/photos/") ||
				!strings.Contains(it.Description, `_thumb.jpg"`) || !strings.Contains(it.Description, ` width="`) {
				t.Errorf("%s item %d: description %q", path, i, it.Description)
			}
			if cellRe.MatchString(it.Title) {
				t.Errorf("%s item %d: cell id in title", path, i)
			}
		}
		// The thumbnail markup is escaped as text, never raw XML.
		if strings.Contains(r.body, "<img") {
			t.Errorf("%s: unescaped HTML in feed", path)
		}
	}
}

func TestFeedEscaping(t *testing.T) {
	f := viewstest.Feed()
	f.Title = `Tom & Jerry's <feed>`
	f.Items = f.Items[:1]
	f.Items[0].Plot.Place = `Ban "Nong" <Khai> & Co`
	body := render(t, views.PageFeed, f)
	var d feedDoc
	if err := xml.Unmarshal([]byte(body), &d); err != nil {
		t.Fatal(err)
	}
	if d.Channel.Title != f.Title || !strings.Contains(d.Channel.Items[0].Title, f.Items[0].Plot.Place) {
		t.Fatalf("round trip: %q / %q", d.Channel.Title, d.Channel.Items[0].Title)
	}
	if !strings.Contains(d.Channel.Items[0].Description, `alt="Photo at Ban &#34;Nong&#34; &lt;Khai&gt; &amp; Co · Chon Buri 2412 by`) {
		t.Fatalf("description not HTML-escaped: %q", d.Channel.Items[0].Description)
	}
}

func TestOpenGraphAndAlternates(t *testing.T) {
	h := newServer(t, 0)
	ph := viewstest.Photo()
	body := get(t, h, "/photo/"+ph.Photo.ID).body
	must(t, body,
		`<meta property="og:type" content="article">`,
		`<meta property="og:image" content="`+testImg+"/"+ph.Photo.Keys.Medium+`">`,
		`<meta property="og:image:width" content="1200">`,
		`<meta property="og:image:height" content="900">`,
		`<meta property="og:image:alt" content="Photo near Pattaya">`,
		`<meta property="og:url" content="`+testBase+"/photo/"+ph.Photo.ID+`">`,
		`<link rel="canonical" href="`+testBase+"/photo/"+ph.Photo.ID+`">`,
		`<meta name="twitter:card" content="summary_large_image">`,
	)
	// Prototype photo: share image is the large variant, size unknown.
	pp := viewstest.PrototypePhoto()
	body = get(t, h, "/photo/"+pp.Photo.ID).body
	must(t, body, `<meta property="og:image" content="`+testImg+"/"+pp.Photo.Keys.Large+`">`)
	mustNot(t, body, "og:image:width")
	for path, feed := range map[string]string{
		"/": "/latest.rss", "/latest": "/latest.rss", "/surveyor/adrian": "/surveyor/adrian.rss", "/region/chon-buri": "/region/chon-buri.rss",
	} {
		must(t, get(t, h, path).body, `<link rel="alternate" type="application/rss+xml"`, `href="`+testBase+feed+`"`)
	}
	mustNot(t, get(t, h, "/plot/88658bd11dfffff").body, `rel="alternate"`)
}

func TestImages(t *testing.T) {
	h := newServer(t, 0)
	ph := viewstest.Photo()
	body := get(t, h, "/photo/"+ph.Photo.ID).body
	must(t, body, `src="`+testImg+"/"+ph.Photo.Keys.Medium+`"`,
		`srcset="`+testImg+"/"+ph.Photo.Keys.Thumb+" 480w, "+testImg+"/"+ph.Photo.Keys.Medium+" 1200w, "+testImg+"/"+ph.Photo.Keys.Large+` 2000w"`,
		`fetchpriority="high"`)
	home := get(t, h, "/").body
	if strings.Count(home, `fetchpriority="high"`) != 1 {
		t.Error("home: want exactly one high-priority image")
	}
	first := imgTag.FindAllString(home, 2)[1] // [0] is the logo
	if !strings.Contains(first, `fetchpriority="high"`) {
		t.Errorf("home: first photo is not high priority: %s", first)
	}
	// Prototype photos have no medium: srcset falls back to the large variant.
	pp := viewstest.PrototypePhoto()
	must(t, get(t, h, "/photo/"+pp.Photo.ID).body, " 480w, "+testImg+"/"+pp.Photo.Keys.Large+" 2000w")
}

func TestNoScriptNeeded(t *testing.T) {
	h := newServer(t, 0)
	for _, path := range []string{"/", "/latest", "/surveyors", "/regions", "/photo/" + viewstest.FeaturedPhotoID, "/about"} {
		mustNot(t, get(t, h, path).body, "<script")
	}
	// Map pages carry one small module loader and a complete fallback.
	for _, path := range []string{"/plot/88658bd11dfffff", "/surveyor/adrian", "/region/chon-buri"} {
		body := get(t, h, path).body
		if strings.Count(body, "<script") != 1 {
			t.Errorf("%s: want one script", path)
		}
		must(t, body, `import("/map/assets/minimap.js")`, `IntersectionObserver`)
	}
	must(t, get(t, h, "/plot/88658bd11dfffff").body, `<svg class="cells"`, `aria-label="Outline of this plot and its 6 neighbours`)
}

func TestFrontPageWeight(t *testing.T) {
	body := render(t, views.PageHome, viewstest.Home())
	rd := newRenderer(t)
	css := rd.assets.size("style.css")
	total := len(body) + css
	t.Logf("front page: %d bytes HTML + %d bytes CSS = %d bytes", len(body), css, total)
	if total >= 100*1024 {
		t.Fatalf("front page is %d bytes, budget is 100 KB", total)
	}
}

func TestStaticAssets(t *testing.T) {
	h := newServer(t, 0)
	rd := newRenderer(t)
	for _, name := range []string{"style.css", "logo.svg", "apple-touch-icon.png"} {
		u := rd.assets.url(name)
		if !regexp.MustCompile(`^/static/[a-z-]+\.[0-9a-f]{8}\.[a-z]+$`).MatchString(u) {
			t.Errorf("%s: url %q has no hash", name, u)
		}
		r := get(t, h, u)
		if r.status != http.StatusOK || r.header.Get("Cache-Control") != "public, max-age=31536000, immutable" {
			t.Errorf("%s: %d %q", u, r.status, r.header.Get("Cache-Control"))
		}
	}
	if ct := get(t, h, rd.assets.url("style.css")).header.Get("Content-Type"); !strings.HasPrefix(ct, "text/css") {
		t.Errorf("css content type %q", ct)
	}
}

func TestPrinciplesCopyIsCurrent(t *testing.T) {
	doc, err := os.ReadFile("../../docs/PRINCIPLES.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(doc) != principlesMD {
		t.Fatal("internal/web/content/principles.md differs from docs/PRINCIPLES.md; copy it again")
	}
	body := get(t, newServer(t, 0), "/principles").body
	must(t, body, "<h1>Principles</h1>", "<ol>", "<strong>People first, map second.</strong>", "Early-web spirit")
	if strings.Count(body, "<li>") != 8 {
		t.Errorf("want 8 principles")
	}
}

func TestRenderSetsNowAndRejectsUnknownPage(t *testing.T) {
	rd := newRenderer(t)
	fixed := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	rd.cfg.Now = func() time.Time { return fixed }
	p := &views.Error{Message: "x"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rd.Render(rec, req.WithContext(views.WithCSRF(req.Context(), "tok-123")), http.StatusTeapot, views.PageError, p)
	if rec.Code != http.StatusTeapot || !p.Now.Equal(fixed) || p.CSRF != "tok-123" {
		t.Fatalf("status %d now %v csrf %q", rec.Code, p.Now, p.CSRF)
	}
	rec = httptest.NewRecorder()
	rd.Render(rec, httptest.NewRequest(http.MethodGet, "/", nil), http.StatusOK, "nope", p)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unknown page: %d", rec.Code)
	}
}

func TestNewRendererConfig(t *testing.T) {
	if _, err := NewRenderer(Config{ImgBaseURL: testImg}); err == nil {
		t.Error("missing BaseURL accepted")
	}
	if _, err := NewRenderer(Config{BaseURL: testBase}); err == nil {
		t.Error("missing ImgBaseURL accepted")
	}
	// A root-relative IMG_BASE_URL is made absolute for feeds and Open Graph.
	rd, err := NewRenderer(Config{BaseURL: testBase, ImgBaseURL: "/img"})
	if err != nil {
		t.Fatal(err)
	}
	if got := rd.absImg("photos/a.jpg"); got != testBase+"/img/photos/a.jpg" {
		t.Fatal(got)
	}
}

func TestPlotByNumber(t *testing.T) {
	h := newServer(t, 0)
	r := get(t, h, "/plot/chon-buri/2417")
	if r.status != http.StatusOK {
		t.Fatalf("status %d", r.status)
	}
	// Same page as /plot/{cell}, which stays the canonical link.
	must(t, r.body, "<h1>Pattaya · Chon Buri 2417", `<link rel="canonical" href="https://cartograph.asia/plot/88658bd11dfffff">`)
	if r := get(t, h, "/plot/CHON-BURI/2417"); r.status != http.StatusOK {
		t.Errorf("slug case: status %d", r.status)
	}
	for _, path := range []string{"/plot/chon-buri/1", "/plot/chon-buri/02417", "/plot/chon-buri/0", "/plot/chon-buri/x", "/plot/rayong/2417"} {
		if r := get(t, h, path); r.status != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", path, r.status)
		}
	}
}
