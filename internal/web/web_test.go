package web

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/countries"
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
	// The mark is lower-case "cartograph.asia"; handles like cartograph-asia
	// keep their spelling, and running text says "Cartograph".
	badMark = regexp.MustCompile(`(?i:\bCartograph\s+Asia\b)|\bCartograph\.asia\b`)
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
	// /about/plots explains the grid itself: it names hexagons and squares
	// and shows an example index on purpose.
	if name != "/about/plots" {
		if m := wordsBad.FindString(text); m != "" {
			t.Errorf("%s: forbidden word %q in visible text", name, m)
		}
		if m := cellRe.FindString(text); m != "" {
			t.Errorf("%s: bare cell id %s in visible text", name, m)
		}
	}
	if m := badMark.FindString(text); m != "" {
		t.Errorf("%s: %q instead of the wordmark cartograph.asia", name, m)
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
			`<b>13</b> plots surveyed`, `href="/surveyors"`, `href="/regions"`, `<abbr class="first" title="First survey of this plot">first</abbr>`, "Chon Buri", `class="lede"`, `<svg class="bar"`,
			`title="Pattaya, Bang Lamung · Chon Buri 2417"><span class="pl">Pattaya</span><span class="pn">&#8288; · Chon Buri 2417</span></a>`,
			`<span lang="th" class="th">ชลบุรี</span>`, `href="/latest">More surveys`, "Welcome",
		}},
		{"latest", views.PageLatest, viewstest.Latest(10), []string{"Latest surveys", `rel="next" href="/latest?page=2"`, "Page 1 of 3"}},
		{"surveyors", views.PageSurveyors, viewstest.Surveyors(), []string{"ploy88", "beach-walker", `8 plots surveyed, <span class="nb">5 of them first</span>`, `12 photos · <span class="nb">joined`}},
		{"regions", views.PageRegions, viewstest.Regions(), []string{"Rayong", "Bangkok", "0 of 4,818 plots",
			"<h2 id=\"countries-h\">Countries</h2>", `<a href="/thailand">Thailand</a> <span lang="th" class="th">ประเทศไทย</span>`,
			`<a href="/vietnam">Vietnam</a>`, "Not open yet"}},
		{"country", views.PageCountry, viewstest.Country(), []string{
			"<h1>Thailand <span lang=\"th\" class=\"th\">ประเทศไทย</span></h1>", `<p class="kicker">Country</p>`, "13 of 12,868 plots",
			`data-minimap="coverage"`, `data-bounds="100.33000,12.50000,101.84000,13.96000"`, "<h2 id=\"provinces-h\">Provinces</h2>",
			`<a href="/region/chon-buri">Chon Buri</a>`, `<b>2</b> of 3 provinces surveyed`, "Latest surveys", "Most active Surveyors",
			`href="/thailand.rss"`, "Open Thailand on the map",
		}},
		{"country not open", views.PageCountry, viewstest.CountryNotOpen(), []string{
			"<h1>Vietnam <span lang=\"vi\"", "Cartograph does not cover Vietnam yet.", `name="robots" content="noindex"`,
		}},
		{"surveyor", views.PageSurveyor, viewstest.Surveyor(), []string{
			"<h1>adrian</h1>", "Walking Pattaya", "home region", `data-minimap="plots"`, `<svg class="cells"`,
			`href="/surveyor/adrian.rss"`, "Edit your profile", `<b>8</b> plots surveyed, <span class="nb">5 of them first</span>`, "8 plots surveyed by adrian",
		}},
		{"placeholder", views.PageSurveyor, viewstest.PlaceholderSurveyor(), []string{"<h1>beach-walker</h1>"}},
		{"plot", views.PagePlot, viewstest.Plot(), []string{
			"<h1 class=\"name\">Pattaya, Bang <span class=\"nb\">Lamung ·</span> <span class=\"pv\">Chon Buri 2417</span>", "first surveyed by", "H3 index <code>88658bd11dfffff</code>", `href="/about/plots">What is a plot?`, `data-paths="/plot/chon-buri/2417,/plot/chon-buri/2418,`, "beach-walker", `href="/map?cell=88658bd11dfffff"`,
			"Survey this plot", `data-focus="88658bd11dfffff"`, `data-photos="6,`, `class="s s3 f"`,
		}},
		{"blank plot", views.PagePlot, viewstest.BlankPlot(), []string{"Nobody has surveyed this plot yet", `name="robots" content="noindex"`, "Survey this plot"}},
		{"photo", views.PagePhoto, viewstest.Photo(), []string{`<h1><a href="/plot/chon-buri/2417">Pattaya, Bang <span class="nb">Lamung ·</span> <span class="pv">Chon Buri 2417</span></a></h1>`, "CC BY-SA 4.0", `rel="prev"`, `rel="next"`, "Photo 3 of", "Full size"}},
		{"prototype photo", views.PagePhoto, viewstest.PrototypePhoto(), []string{"first survey of this plot", `width="2000" height="1500"`}},
		{"region", views.PageRegion, viewstest.Region(), []string{
			"<h1>Chon Buri", "0.20%", "5,921 plots", `data-minimap="coverage"`, `data-bounds="100.68000,12.50000,101.73000,13.59000"`,
			`<p class="kicker">Province in <a href="/thailand">Thailand</a></p>`, "Most active Surveyors", "Last 30 days", "All time", "coverage, 12 of 5,921 plots", `href="/region/chon-buri.rss"`, "Open Chon Buri on the map",
		}},
		{"signin", views.PageSignIn, viewstest.SignIn(), []string{`action="/signin"`, `name="csrf" value="tok-123"`, `name="next" value="/map"`, `type="email"`, `value="adrian@example"`, "That does not look", `aria-invalid="true" aria-describedby="email-error"`}},
		{"check-email", views.PageCheckEmail, viewstest.CheckEmail(), []string{"adrian@example.org"}},
		{"choose-name", views.PageChooseName, viewstest.ChooseName(), []string{`action="/signin/name"`, `pattern="[A-Za-z0-9\-]{3,30}"`, `maxlength="30"`, "That name is taken",
			`<p id="name-error" class="error" role="alert">`, `aria-describedby="name-error name-hint" autofocus aria-invalid="true"`}},
		{"edit-profile", views.PageEditProfile, viewstest.EditProfile(), []string{`action="/settings"`, "Saved.", `<option value="chon-buri" selected>`, `maxlength="500"`, "Walking Pattaya"}},
		{"confirm-signin", views.PageConfirmSignIn, &views.ConfirmSignIn{Page: views.Page{Title: "Sign in", NoIndex: true}, Token: "abc&def"},
			[]string{`action="/signin/link"`, `name="token" value="abc&amp;def"`, `name="csrf" value="tok-123"`, ">Sign in</button>"}},
		{"error", views.PageError, viewstest.NotFound(), []string{"There is nothing at this address."}},
		{"signin-again", views.PageSignInAgain, viewstest.SignInAgain(), []string{`action="/signout"`, `name="next" value="/signin?next=%2Fadmin"`}},
	}
	for _, p := range pages {
		t.Run(p.name, func(t *testing.T) {
			body := render(t, p.page, p.data, ctx)
			checkHTML(t, p.name, body)
			must(t, body, p.wants...)
			must(t, body, `href="/map">Start surveying`, `href="/surveyor/adrian">adrian</a>`, `action="/signout"`,
				`<nav class="site" aria-label="Sections"><a href="/latest">Latest</a>`, `<a href="/regions">Regions</a>`, `<a href="/surveyors">Surveyors</a>`, `<a href="/map">Map</a></nav>`,
				`href="/principles"`, `href="/about/plots">About the plots`, "hello@cartograph.asia", "@cartographasia", "@cartograph.asia", "ODbL")
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

// TestWordmark checks that the site stands as cartograph.asia, lower case,
// wherever it is the mark: masthead, footer, title, og:site_name and feed
// titles.
func TestWordmark(t *testing.T) {
	const mark = `cartograph<span class="tld">.asia</span>`
	body := render(t, views.PageHome, viewstest.Home())
	must(t, body, `<span class="wordmark">`+mark+`</span>`, `<p class="foot-mark"><a href="/">`+mark+`</a></p>`,
		"<title>cartograph.asia: a photo for every plot of land</title>",
		`<meta property="og:site_name" content="cartograph.asia">`,
		`title="cartograph.asia: latest surveys"`)
	body = render(t, views.PageSurveyors, viewstest.Surveyors())
	must(t, body, "<title>Surveyors · cartograph.asia</title>")
	h := newServer(t, 0)
	for _, path := range []string{"/latest.rss", "/region/chon-buri.rss", "/surveyor/adrian.rss"} {
		if r := get(t, h, path); !strings.Contains(r.body, "<title>cartograph.asia: ") {
			t.Errorf("%s: channel title is not cartograph.asia: …", path)
		}
	}
}

func TestRoutes(t *testing.T) {
	h := newServer(t, 0)
	for _, path := range []string{
		"/", "/latest", "/surveyors", "/regions", "/surveyor/adrian", "/surveyor/beach-walker",
		"/plot/88658bd11dfffff", "/plot/88658bd157fffff", "/photo/" + viewstest.FeaturedPhotoID,
		"/region/chon-buri", "/region/bangkok", "/region/rayong", "/thailand", "/vietnam", "/about", "/about/plots", "/principles", "/licence",
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
	if !strings.Contains(d.Channel.Items[0].Description, `alt="Photo at Ban &#34;Nong&#34; &lt;Khai&gt; &amp; Co, Bang Lamung · Chon Buri 2412 by`) {
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

func TestAdminPagesRender(t *testing.T) {
	ctx := views.WithViewer(context.Background(), viewstest.AdminViewer())
	pages := []struct {
		name  string
		page  string
		data  any
		wants []string
	}{
		{"overview", views.PageAdminOverview, viewstest.AdminOverview(), []string{"<h1>Overview</h1>", "<b>1</b> invitation pending", "4 links sent and 2 redeemed",
			"no name yet", `<code>surveyor.block</code>`, `href="/admin/surveyor/spam-bot"`, `<span class="tag">hidden</span>`, `aria-current="page">Overview`}},
		{"photos", views.PageAdminPhotos, viewstest.AdminPhotos(), []string{"Hidden photos", "Show all photos", `href="/admin/photo/` + viewstest.HiddenPhotoID}},
		{"hidden photo", views.PageAdminPhoto, viewstest.AdminPhoto(true), []string{"Shows a face close up.", "/unhide", "Unhide the photo", "12.93201, 100.88713", "adrian@example.org"}},
		{"photo", views.PageAdminPhoto, viewstest.AdminPhoto(false), []string{"Feature this photo", "/hide", "Plot of the week", "until replaced"}},
		{"confirm", views.PageAdminConfirm, viewstest.AdminConfirm(), []string{"Hide this photo", "passes to ploy88", `name="reason"`, `maxlength="200"`, `aria-invalid="true"`,
			`name="hide_photos" type="checkbox" value="1"`, `class="danger" type="submit">Hide the photo`, "Back, change nothing"}},
		{"featured", views.PageAdminFeatured, viewstest.AdminFeatured(), []string{"End now", "Delete this scheduled entry", `type="datetime-local"`, "Paste a photo", "Last week."}},
		{"surveyors", views.PageAdminSurveyors, viewstest.AdminSurveyors(), []string{"spam-bot", `<span class="tag">blocked</span>`, "bot@example.org"}},
		{"surveyor", views.PageAdminSurveyor, viewstest.AdminSurveyor(false), []string{"Block ploy88", "Clear the bio", "Sign out everywhere", "Jomtien every morning."}},
		{"blocked surveyor", views.PageAdminSurveyor, viewstest.AdminSurveyor(true), []string{"Blocked ", "Uploads screenshots.", "Lift the block", "Also unhide the 1 photo hidden with the block"}},
		{"places", views.PageAdminPlaces, viewstest.AdminPlaces(), []string{"Ban Bang Krabueang", `action="/admin/places/th/Ban%20Bang%20Krabueang"`, `value="replace" checked`, "Seri Village", "List it"}},
		{"invites", views.PageAdminInvites, viewstest.AdminInvites(), []string{"Sign-up is by invitation", "already a Surveyor: ploy88", "pending", "accepted as", "Send a new link", "Revoke", `maxlength="300"`}},
		{"log", views.PageAdminLog, viewstest.AdminLog(), []string{"<h1>Log</h1>", "Before and after", `<code>invite.create</code>`}},
	}
	for _, p := range pages {
		t.Run(p.name, func(t *testing.T) {
			body := render(t, p.page, p.data, ctx)
			checkHTML(t, "admin "+p.name, body)
			must(t, body, p.wants...)
			must(t, body, `<meta name="robots" content="noindex">`, `<nav class="admin-nav" aria-label="Admin">`, `href="/admin">Admin</a>`)
		})
	}
}

func TestAdminLinksOnlyForAdmins(t *testing.T) {
	admin := views.WithViewer(context.Background(), viewstest.AdminViewer())
	member := views.WithViewer(context.Background(), viewstest.Viewer())
	must(t, render(t, views.PagePhoto, viewstest.Photo(), admin), `href="/admin">Admin</a>`, `<p class="admin-line">Admin:`)
	for _, ctx := range []context.Context{member, context.Background()} {
		mustNot(t, render(t, views.PagePhoto, viewstest.Photo(), ctx), `href="/admin"`, "admin-line")
	}
	si := viewstest.SignIn()
	si.InviteOnly = true
	must(t, render(t, views.PageSignIn, si), "Cartograph is invitation-only for now.")
	mustNot(t, render(t, views.PageSignIn, viewstest.SignIn()), "invitation-only")
}

func TestHiddenPhotoIsGone(t *testing.T) {
	h := newServer(t, 0)
	r := get(t, h, "/photo/"+viewstest.GonePhotoID)
	if r.status != http.StatusGone {
		t.Fatalf("hidden photo: %d", r.status)
	}
	must(t, r.body, "This photo was removed.", `content="noindex"`, "Error 410")
}

func TestNoInlineStyles(t *testing.T) {
	// Every page is styled from style.css only (docs/DESIGN_BRIEF.md).
	for _, name := range []string{"home", "latest", "surveyors", "regions", "surveyor", "plot", "photo", "region", "country",
		"signin", "check-email", "choose-name", "edit-profile", "confirm-signin", "error", "static", "layout", "partials",
		"admin-overview", "admin-photos", "admin-photo", "admin-confirm", "admin-featured", "admin-surveyors", "admin-surveyor",
		"admin-places", "admin-invites", "admin-log", "signin-again"} {
		b, err := templateFS.ReadFile("templates/" + name + ".html")
		if err != nil {
			t.Fatal(err)
		}
		if m := regexp.MustCompile(`\sstyle=|<style`).Find(b); m != nil {
			t.Errorf("templates/%s.html: inline style %q", name, m)
		}
	}
}

func TestStylesheetURLs(t *testing.T) {
	// url(...) in a stylesheet is rewritten to the referenced asset's hashed
	// URL; a reference to a missing asset is an error.
	a, err := loadAssets(fstest.MapFS{
		"style.css":     {Data: []byte(`@font-face { src: url(fonts/f.woff2) format("woff2"); } .x { background: url("dot.svg"); }`)},
		"fonts/f.woff2": {Data: []byte("font")},
		"dot.svg":       {Data: []byte("<svg/>")},
	})
	if err != nil {
		t.Fatal(err)
	}
	css := string(a.byHashed[strings.TrimPrefix(a.url("style.css"), StaticPrefix)].data)
	for _, f := range []string{"fonts/f.woff2", "dot.svg"} {
		if !strings.Contains(css, "url("+a.url(f)+")") {
			t.Errorf("style.css does not refer to %s as %s: %s", f, a.url(f), css)
		}
	}
	if _, err := loadAssets(fstest.MapFS{"style.css": {Data: []byte(`.x { background: url(missing.png); }`)}}); err == nil {
		t.Error("a reference to a missing asset loaded")
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

func TestStaticPages(t *testing.T) {
	// Every Markdown file in docs/pages/ has a route.
	files, err := filepath.Glob("../../docs/pages/*.md")
	if err != nil || len(files) == 0 {
		t.Fatalf("docs/pages: %v %v", files, err)
	}
	routed := map[string]bool{}
	for _, sp := range StaticPages {
		routed[sp.File] = true
	}
	for _, f := range files {
		if !routed["pages/"+filepath.Base(f)] {
			t.Errorf("%s has no entry in StaticPages", f)
		}
	}

	h := newServer(t, 0)
	body := get(t, h, "/principles").body
	must(t, body, "<h1>Principles</h1>", "<ol>", "<strong>People first, map second.</strong>", "Early-web spirit")
	if strings.Count(body, "<li>") != 8 {
		t.Errorf("want 8 principles")
	}
	body = get(t, h, "/about/plots").body
	must(t, body, "<title>About the plots · cartograph.asia</title>", "<h1>About the plots</h1>", `<p class="lede">Cartograph divides`,
		`<meta name="description" content="Cartograph divides the map into plots:`, "<h2>Why hexagons</h2>",
		"<table>", "<th>Cells worldwide</th><td>691,776,122</td>", "<code>88658bd337fffff</code>", `<a href="https://h3geo.org/">H3</a>`,
		`<link rel="canonical" href="https://cartograph.asia/about/plots">`)
	mustNot(t, body, "<thead>", "**", "`")
	body = get(t, h, "/licence").body
	must(t, body, "<h1>Data &amp; licence</h1>", "<ul>", `<a href="/about/plots">About the plots</a>`, `href="mailto:hello@cartograph.asia"`)
	body = get(t, h, "/about").body
	must(t, body, "<h1>About</h1>", `<p class="lede">Cartograph is a photo survey`, "<ol>", `<a href="/map">map</a>`)
	// The legal pages are drafts in docs/drafts/ until they are filled in
	// (docs/DEPLOY.md, Going live): not served and not linked.
	mustNot(t, body, `href="/legal-notice"`, `href="/privacy"`, `href="/impressum"`)
	for _, u := range []string{"/legal-notice", "/privacy", "/impressum"} {
		if r := get(t, h, u); r.status != http.StatusNotFound {
			t.Errorf("%s: %d, want 404 while the page is a draft", u, r.status)
		}
	}
}

func TestMarkdown(t *testing.T) {
	d := markdown("# T **x**\n\nFirst `a<b>` and [l](javascript:void).\n\n| h | i |\n|---|---|\n| 1 | 2 |\n\n- one\n- two\n1. three\n")
	if d.Title != "T x" || d.Summary != "First a<b> and l." {
		t.Fatalf("title %q summary %q", d.Title, d.Summary)
	}
	body := string(d.Body)
	for _, want := range []string{
		`<p class="lede">First <code>a&lt;b&gt;</code> and l.</p>`,
		"<thead><tr><th>h</th><th>i</th></tr></thead>", "<tr><th>1</th><td>2</td></tr>",
		"<ul>\n<li>one</li>\n<li>two</li>\n</ul>\n<ol>\n<li>three</li>\n</ol>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in\n%s", want, body)
		}
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
	must(t, r.body, "<h1 class=\"name\">Pattaya, Bang <span class=\"nb\">Lamung ·</span> <span class=\"pv\">Chon Buri 2417</span>", `<link rel="canonical" href="https://cartograph.asia/plot/88658bd11dfffff">`)
	if r := get(t, h, "/plot/CHON-BURI/2417"); r.status != http.StatusOK {
		t.Errorf("slug case: status %d", r.status)
	}
	for _, path := range []string{"/plot/chon-buri/1", "/plot/chon-buri/02417", "/plot/chon-buri/0", "/plot/chon-buri/x", "/plot/rayong/2417"} {
		if r := get(t, h, path); r.status != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", path, r.status)
		}
	}
}

// TestCaptionsLeaveOutWhatIsShared checks that a plot page's grid names the
// Surveyor instead of the plot, and a Surveyor's page leaves out the name.
func TestCaptionsLeaveOutWhatIsShared(t *testing.T) {
	body := render(t, views.PagePlot, viewstest.Plot())
	must(t, body, `<p class="cap"><a class="who" href="/surveyor/beach-walker">beach-walker</a>`)
	mustNot(t, body, `<p class="cap"><a class="plot"`)
	body = render(t, views.PageSurveyor, viewstest.Surveyor())
	must(t, body, `<p class="cap"><a class="plot"`)
	mustNot(t, body, `<span class="lt"><a href="/surveyor/adrian">`)
	body = render(t, views.PageLatest, viewstest.Latest(10))
	must(t, body, `<span class="line"><span class="lt"><a href="/surveyor/`)
}

// TestFirstTagOutsideTheCut checks that the "first" tag follows the part of
// the caption line that may be cut, so it always shows.
func TestFirstTagOutsideTheCut(t *testing.T) {
	body := render(t, views.PageHome, viewstest.Home())
	must(t, body, `</time></span> <abbr class="first" title="First survey of this plot">first</abbr></span>`)
	mustNot(t, body, `</time> <abbr class="first"`)
}

func TestSurveyed(t *testing.T) {
	for _, c := range []struct {
		n, first int
		want     string
	}{
		{0, 0, "plots surveyed"},
		{1, 0, "plot surveyed"},
		{1, 1, `plot surveyed, <span class="nb">the first</span>`},
		{8, 0, "plots surveyed"},
		{8, 5, `plots surveyed, <span class="nb">5 of them first</span>`},
		{2500, 1200, `plots surveyed, <span class="nb">1,200 of them first</span>`},
	} {
		if got := string(surveyed(c.n, c.first)); got != c.want {
			t.Errorf("surveyed(%d, %d) = %q, want %q", c.n, c.first, got, c.want)
		}
	}
}

func TestHeading(t *testing.T) {
	p := views.PlotRef{Place: "A & B", District: "C", Region: views.RegionRef{NameEn: "D", Slug: "d"}, Number: 7}
	if got, want := string(heading(p)), `A &amp; B, <span class="nb">C ·</span> <span class="pv">D 7</span>`; got != want {
		t.Errorf("heading = %q, want %q", got, want)
	}
	if got := string(heading(views.PlotRef{Region: views.RegionRef{NameEn: "D"}, Number: 7})); got != "D 7" {
		t.Errorf("heading without a place = %q", got)
	}
}

func TestDisplayNames(t *testing.T) {
	// somchai-k is "Somchai Kittisak", ploy88 "พลอย ชลบุรี"; adrian has none.
	h := newServer(t, 0)
	home := get(t, h, "/").body
	must(t, home, `<a href="/surveyor/ploy88">พลอย ชลบุรี</a>`)
	mustNot(t, home, `>ploy88<`)

	prof := get(t, h, "/surveyor/somchai-k").body
	must(t, prof, "<h1>Somchai Kittisak</h1>", `<p class="meta">@somchai-k · Surveyor since`, "<title>Somchai Kittisak · cartograph.asia</title>",
		`href="/surveyor/somchai-k.rss"`, `<link rel="canonical" href="https://cartograph.asia/surveyor/somchai-k">`)
	mustNot(t, get(t, h, "/surveyor/adrian").body, "@adrian")

	feed := get(t, h, "/surveyor/somchai-k.rss").body
	must(t, feed, `xmlns:dc="http://purl.org/dc/elements/1.1/"`, "<dc:creator>Somchai Kittisak</dc:creator>", "by Somchai Kittisak</title>")

	// A photo by a Surveyor with a display name: label, handle, licence.
	src := viewstest.NewFake()
	l, _ := src.Latest(context.Background(), 1)
	var id string
	for _, p := range l.Photos {
		if p.Surveyor.Name == "somchai-k" {
			id = p.ID
			break
		}
	}
	ph := get(t, h, "/photo/"+id).body
	must(t, ph, `by <a href="/surveyor/somchai-k">Somchai Kittisak</a> @somchai-k`,
		`© <a href="/surveyor/somchai-k">Somchai Kittisak (@somchai-k)</a>, <a rel="license"`)
}

func TestDisplayNameIsTextNotURL(t *testing.T) {
	// A display name with markup characters is escaped in text and never
	// becomes part of a URL.
	p := viewstest.Photo()
	p.Photo.Surveyor = views.SurveyorRef{Name: "ada", Display: `Ada "<b>" Lovelace`}
	v := viewstest.Viewer()
	v.Display = `Ada "<b>" Lovelace`
	body := render(t, views.PagePhoto, p, views.WithViewer(context.Background(), v))
	must(t, body, `Ada &#34;&lt;b&gt;&#34; Lovelace`, `href="/surveyor/ada"`)
	mustNot(t, body, `<b>"`, `/surveyor/Ada`, `Lovelace"`+">", `%3cb%3e`)
	for _, m := range regexp.MustCompile(`href="[^"]*"`).FindAllString(body, -1) {
		if strings.Contains(m, "Lovelace") {
			t.Errorf("display name in a URL: %s", m)
		}
	}
}

func TestDisplayNameForms(t *testing.T) {
	cn := viewstest.ChooseName()
	cn.Error, cn.ErrorField, cn.Display = "That is another Surveyor's name.", "display", "somchai-k"
	body := render(t, views.PageChooseName, cn)
	must(t, body, `name="display_name"`, `value="somchai-k"`, `maxlength="50"`, `aria-describedby="name-error display-hint" autofocus aria-invalid="true"`)
	mustNot(t, body, `aria-describedby="name-error name-hint"`)
	ep := viewstest.EditProfile()
	ep.Display = "Adrian Altner"
	must(t, render(t, views.PageEditProfile, ep), `id="display_name" name="display_name" type="text" value="Adrian Altner"`, "Optional; you can change it any time.")
}

func TestCountryPages(t *testing.T) {
	h := newServer(t, 10)

	th := get(t, h, "/thailand")
	if th.status != http.StatusOK {
		t.Fatalf("/thailand: %d", th.status)
	}
	must(t, th.body, `<link rel="canonical" href="https://cartograph.asia/thailand">`,
		`<link rel="alternate" type="application/rss+xml" title="cartograph.asia: Thailand" href="https://cartograph.asia/thailand.rss">`,
		`rel="next" href="/thailand?page=2"`, "Page 1 of 3")
	mustNot(t, th.body, `name="robots"`)
	if strings.Count(th.body, `<li><a class="ph"`) != 10 {
		t.Error("/thailand does not show 10 photos on its first page")
	}
	p2 := get(t, h, "/thailand?page=2")
	must(t, p2.body, `rel="prev" href="/thailand"`, `href="https://cartograph.asia/thailand?page=2"`)

	// In the list, not imported: a short page that says so, not indexed, no feed.
	vn := get(t, h, "/vietnam")
	if vn.status != http.StatusOK {
		t.Fatalf("/vietnam: %d", vn.status)
	}
	must(t, vn.body, `name="robots" content="noindex"`, "Cartograph does not cover Vietnam yet.", `<p class="kicker">Country</p>`)
	mustNot(t, vn.body, "vietnam.rss", `rel="alternate"`, "data-minimap", `class="figures`, "Latest surveys")

	for from, to := range map[string]string{
		"/Thailand":         "/thailand",
		"/THAILAND?page=2":  "/thailand?page=2",
		"/thailand/":        "/thailand",
		"/Vietnam/":         "/vietnam",
		"/thailand?page=1":  "/thailand",
		"/THAILAND.rss":     "/thailand.rss",
		"/South-Korea":      "/south-korea",
		"/thailand.rss?x=1": "",
	} {
		r := get(t, h, from)
		if to == "" {
			if r.status != http.StatusOK {
				t.Errorf("%s: %d, want 200", from, r.status)
			}
			continue
		}
		if r.status != http.StatusMovedPermanently || r.header.Get("Location") != to {
			t.Errorf("%s: %d -> %q, want 301 -> %q", from, r.status, r.header.Get("Location"), to)
		}
	}

	// Nothing is guessed: codes, misspellings and unknown names are plain 404s.
	for _, path := range []string{"/th", "/tailand", "/thailand-2", "/atlantis", "/atlantis/", "/atlantis.rss", "/thailand?page=x",
		"/thailand?page=0", "/thailand?page=4", "/vietnam?page=2", "/vietnam.rss", "/.rss", "/thailand.rss.rss", "/th.rss"} {
		r := get(t, h, path)
		if r.status != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", path, r.status)
		}
	}
}

func TestCountryFeed(t *testing.T) {
	r := get(t, newServer(t, 10), "/thailand.rss")
	if r.status != http.StatusOK || !strings.HasPrefix(r.header.Get("Content-Type"), "application/rss+xml") {
		t.Fatalf("/thailand.rss: %d %q", r.status, r.header.Get("Content-Type"))
	}
	var f feedDoc
	if err := xml.Unmarshal([]byte(r.body), &f); err != nil {
		t.Fatal(err)
	}
	if f.Channel.Title != "cartograph.asia: Thailand" || len(f.Channel.Items) != 30 {
		t.Errorf("feed: title %q, %d items", f.Channel.Title, len(f.Channel.Items))
	}
	if rss, _ := links(f.Channel.Links); rss != "https://cartograph.asia/thailand" {
		t.Errorf("feed link %q", rss)
	}
}

// Static routes win over /{country}: every page of the site still answers.
func TestCountryRouteLeavesPagesAlone(t *testing.T) {
	h := newServer(t, 0)
	for _, sp := range StaticPages {
		if r := get(t, h, sp.Path); r.status != http.StatusOK {
			t.Errorf("%s: %d", sp.Path, r.status)
		}
	}
	for _, path := range []string{"/latest", "/latest.rss", "/regions", "/surveyors"} {
		if r := get(t, h, path); r.status != http.StatusOK {
			t.Errorf("%s: %d", path, r.status)
		}
	}
	// And a country slug is never a first segment of a route this package registers.
	r := chi.NewRouter()
	NewHandlers(viewstest.NewFake(), newRenderer(t)).Routes(r)
	_ = chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		first, _, _ := strings.Cut(strings.TrimPrefix(route, "/"), "/")
		if _, ok := countries.BySlug(first); ok {
			t.Errorf("route %s %s starts with a country's slug", method, route)
		}
		return nil
	})
}
