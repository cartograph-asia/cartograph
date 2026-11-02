package web

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/views"
	"github.com/cartograph-asia/cartograph/internal/views/viewstest"
)

func TestProfileQuery(t *testing.T) {
	h := newServer(t, 4)

	for _, bad := range []string{
		"?order=random", "?order=", "?order=oldest&order=taken", "?region=atlantis", "?region=../x", "?region=",
		"?year=1999", "?year=24", "?year=abcd", "?year=3000", "?one=all", "?one=", "?page=0", "?page=99",
	} {
		if r := get(t, h, "/surveyor/adrian"+bad); r.status != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", bad, r.status)
		}
	}

	// The same view spelled differently is redirected to its canonical form.
	for from, to := range map[string]string{
		"/surveyor/adrian?page=1":                       "/surveyor/adrian",
		"/surveyor/adrian?one=plot&order=oldest":        "/surveyor/adrian?one=plot&order=oldest",
		"/surveyor/adrian?region=Chon-Buri&order=taken": "/surveyor/adrian?order=taken&region=chon-buri",
		"/surveyor/ADRIAN?year=2026&page=1":             "/surveyor/adrian?year=2026",
	} {
		r := get(t, h, from)
		switch {
		case from == to:
			if r.status != http.StatusOK {
				t.Errorf("%s: status %d", from, r.status)
			}
		case r.status != http.StatusMovedPermanently || r.header.Get("Location") != to:
			t.Errorf("%s: %d -> %q, want 301 -> %q", from, r.status, r.header.Get("Location"), to)
		}
	}
	// Parameters of other kinds are left alone.
	if r := get(t, h, "/surveyor/adrian?utm_source=x"); r.status != http.StatusOK {
		t.Errorf("utm_source: %d", r.status)
	}

	// The plain profile is indexed; any other view is not, and paging keeps the view.
	plain := get(t, h, "/surveyor/adrian")
	mustNot(t, plain.body, `name="robots"`)
	must(t, plain.body, `rel="next" href="/surveyor/adrian?page=2"`, `<a href="/surveyor/adrian?order=oldest">Oldest uploaded</a>`,
		`<a href="/surveyor/adrian?order=taken">Newest taken</a>`, `<a href="/surveyor/adrian?one=plot">One photo per plot</a>`,
		`aria-current="true">Newest uploaded</a>`, `aria-current="true">All photos</a>`, "Older →")
	old := get(t, h, "/surveyor/adrian?order=oldest")
	must(t, old.body, `name="robots" content="noindex"`, `rel="next" href="/surveyor/adrian?order=oldest&amp;page=2"`,
		`aria-current="true">Oldest uploaded</a>`, "Oldest uploaded first.", "Next →")
	mustNot(t, old.body, "Older →")
	old2 := get(t, h, "/surveyor/adrian?order=oldest&page=2")
	must(t, old2.body, `rel="prev" href="/surveyor/adrian?order=oldest"`)

	one := get(t, h, "/surveyor/adrian?one=plot")
	must(t, one.body, `aria-current="true">One photo per plot</a>`)
	if n := strings.Count(one.body, `<li><a class="ph"`); n == 0 || n > 8 {
		t.Errorf("one photo per plot shows %d photos, adrian has 8 plots", n)
	}
}

func TestProfileSections(t *testing.T) {
	// The fixture Surveyor: figures, home region line, exports, the breakdown by region.
	s := viewstest.Surveyor()
	body := render(t, views.PageSurveyor, s)
	must(t, body, `<b>8</b> plots surveyed, <span class="nb">5 of them first</span>`, `days with photos</li>`,
		`href="/surveyor/adrian.gpx">GPX of the plots</a>`, `href="/surveyor/adrian.geojson">GeoJSON of the plots</a>`,
		`Newest uploaded</a>`)
	if len(s.Regions) > 1 {
		must(t, body, "Where and when", `<h3 class="kicker">Where</h3>`, `class="tally"`, `href="/surveyor/adrian?region=`)
	}

	// Breakdowns appear with two regions or two years, home coverage with plots in the home region.
	s = viewstest.Surveyor()
	s.Regions = []views.SurveyorRegion{
		{Region: viewstest.ChonBuri, Plots: 6, Photos: 9, Land: 5921},
		{Region: viewstest.Rayong, Plots: 2, Photos: 2, Land: 4818},
	}
	s.Years = []views.SurveyorYear{{Year: 2026, Photos: 7}, {Year: 2025, Photos: 1}}
	s.Profile.HomeRegion = &viewstest.ChonBuri
	s.HomeCoverage = &views.Coverage{Claimed: 6, Land: 5921}
	body = render(t, views.PageSurveyor, s)
	must(t, body, `<h3 class="kicker">Where</h3>`, `<h3 class="kicker">When</h3>`,
		`<a href="/surveyor/adrian?region=chon-buri">Chon Buri</a> <span class="n">6 plots of 5,921 · 9 photos</span>`,
		`<a href="/surveyor/adrian?region=rayong">Rayong</a>`,
		`<a href="/surveyor/adrian?year=2026">2026</a> <span class="n">7 photos</span>`, `<a href="/surveyor/adrian?year=2025">2025</a>`,
		`6 of 5,921 plots of <a href="/region/chon-buri">Chon Buri</a> surveyed by adrian`, `<svg class="bar"`)

	// With one region, one year and nothing in the home region the page stays as short as before.
	s = viewstest.Surveyor()
	s.Regions, s.Years, s.Stats.Days = s.Regions[:1], s.Years[:min(1, len(s.Years))], 0
	s.HomeCoverage = &views.Coverage{Land: 5921}
	body = render(t, views.PageSurveyor, s)
	mustNot(t, body, "Where and when", `class="tally"`, "days with photos", `<div class="cov big">`)

	// A filtered view says what it shows and how to leave it.
	s, err := viewstest.NewFake().Surveyor(context.Background(), "adrian", views.SurveyorQuery{Region: "rayong", Year: 2026})
	if err != nil {
		t.Fatal(err)
	}
	body = render(t, views.PageSurveyor, s)
	must(t, body, "Showing 0 photos in Rayong taken in 2026.", `<a href="/surveyor/adrian">Show all</a>`, "No photos of adrian match this view.")
	mustNot(t, body, "has not uploaded a photo yet")

	// Somebody without photos has no controls.
	s = viewstest.Surveyor()
	s.Photos, s.PlotCells, s.Regions, s.Years, s.Stats = nil, nil, nil, nil, views.SurveyorStats{}
	body = render(t, views.PageSurveyor, s)
	must(t, body, "has not uploaded a photo yet")
	mustNot(t, body, "Oldest uploaded", ".gpx", "Show all")
}

var inlineStyle = regexp.MustCompile(`\sstyle=`)

func TestProfileViewsAreClean(t *testing.T) {
	h := newServer(t, 4)
	for _, p := range []string{
		"/surveyor/adrian", "/surveyor/adrian?order=taken", "/surveyor/adrian?one=plot", "/surveyor/adrian?region=chon-buri&year=2026",
	} {
		r := get(t, h, p)
		if r.status != http.StatusOK {
			t.Fatalf("%s: %d", p, r.status)
		}
		if inlineStyle.MatchString(r.body) {
			t.Errorf("%s: inline style", p)
		}
		if wordsBad.MatchString(tagRe.ReplaceAllString(r.body, " ")) {
			t.Errorf("%s: %q in the text", p, wordsBad.FindString(tagRe.ReplaceAllString(r.body, " ")))
		}
		if r.header.Get("Set-Cookie") != "" {
			t.Errorf("%s: sets a cookie", p)
		}
	}
}

type gpxDoc struct {
	XMLName xml.Name `xml:"gpx"`
	Version string   `xml:"version,attr"`
	Meta    struct {
		Name string `xml:"name"`
	} `xml:"metadata"`
	Points []struct {
		Lat  string `xml:"lat,attr"`
		Lon  string `xml:"lon,attr"`
		Name string `xml:"name"`
		Desc string `xml:"desc"`
		Link struct {
			Href string `xml:"href,attr"`
		} `xml:"link"`
	} `xml:"wpt"`
}

func TestProfileExports(t *testing.T) {
	h := newServer(t, 10)
	want, err := viewstest.NewFake().SurveyorPlots(context.Background(), "adrian")
	if err != nil || len(want.Plots) != 8 {
		t.Fatalf("fixture plots: %+v %v", want, err)
	}

	r := get(t, h, "/surveyor/adrian.gpx")
	if r.status != http.StatusOK || r.header.Get("Content-Type") != "application/gpx+xml; charset=utf-8" ||
		r.header.Get("Content-Disposition") != `inline; filename="adrian.gpx"` || r.header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("gpx: %d %v", r.status, r.header)
	}
	var g gpxDoc
	if err := xml.Unmarshal([]byte(r.body), &g); err != nil {
		t.Fatalf("gpx does not parse: %v\n%s", err, r.body)
	}
	if g.Version != "1.1" || g.Meta.Name != "cartograph.asia: plots of adrian" || len(g.Points) != 8 {
		t.Fatalf("gpx: %+v", g)
	}
	first := g.Points[0]
	if first.Lat == "" || first.Lon == "" || first.Name != want.Plots[0].Plot.Name() ||
		!strings.HasPrefix(first.Link.Href, testBase+"/plot/") || !strings.Contains(first.Desc, "photo") {
		t.Errorf("first waypoint %+v", first)
	}

	r = get(t, h, "/surveyor/adrian.geojson")
	if r.status != http.StatusOK || r.header.Get("Content-Type") != "application/geo+json" {
		t.Fatalf("geojson: %d %v", r.status, r.header)
	}
	var gj struct {
		Type     string `json:"type"`
		Features []struct {
			Geometry struct {
				Type        string    `json:"type"`
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties map[string]any `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal([]byte(r.body), &gj); err != nil {
		t.Fatalf("geojson does not parse: %v", err)
	}
	if gj.Type != "FeatureCollection" || len(gj.Features) != 8 {
		t.Fatalf("geojson: %s", r.body)
	}
	f := gj.Features[0]
	if f.Geometry.Type != "Point" || len(f.Geometry.Coordinates) != 2 || f.Geometry.Coordinates[0] < 100 || f.Geometry.Coordinates[1] > 20 {
		t.Errorf("coordinates are [lng, lat]: %+v", f.Geometry)
	}
	if len(f.Properties) != 4 || f.Properties["name"] != want.Plots[0].Plot.Name() {
		t.Errorf("properties %v", f.Properties)
	}

	// Only plots: no e-mail address, no time, no photo.
	for _, body := range []string{get(t, h, "/surveyor/adrian.gpx").body, r.body} {
		mustNot(t, body, "@example", "<time", "uploaded", "/photo/", "_thumb")
	}

	// The name's spelling is canonical, as for the feeds; unknown names are 404; HEAD has no body.
	if r := get(t, h, "/surveyor/ADRIAN.gpx"); r.status != http.StatusMovedPermanently || r.header.Get("Location") != "/surveyor/adrian.gpx" {
		t.Errorf("ADRIAN.gpx: %d %q", r.status, r.header.Get("Location"))
	}
	if r := get(t, h, "/surveyor/nobody.geojson"); r.status != http.StatusNotFound {
		t.Errorf("nobody.geojson: %d", r.status)
	}
}

// TestProfileExportEscapes holds a display name that looks like markup to the
// encoders, not to a browser.
func TestProfileExportEscapes(t *testing.T) {
	rd := newRenderer(t)
	l := &views.SurveyorPlotList{
		Surveyor: views.SurveyorRef{Name: "ada", Display: `Ada "<b>" & Lovelace`},
		Plots: []views.SurveyorPlot{{
			Plot: views.PlotRef{Cell: "88658bd11dfffff", Place: `Bang <Lamung> & "co"`, Region: viewstest.ChonBuri, Number: 7},
			Lat:  12.9319, Lng: 100.8867, Photos: 1,
		}},
	}
	var gpx, js strings.Builder
	if err := rd.writeGPX(&gpx, l); err != nil {
		t.Fatal(err)
	}
	if err := rd.writeGeoJSON(&js, l); err != nil {
		t.Fatal(err)
	}
	var g gpxDoc
	if err := xml.Unmarshal([]byte(gpx.String()), &g); err != nil {
		t.Fatalf("gpx: %v\n%s", err, gpx.String())
	}
	if g.Meta.Name != `cartograph.asia: plots of Ada "<b>" & Lovelace` || !strings.Contains(g.Points[0].Name, `<Lamung> & "co"`) {
		t.Errorf("round trip: %q / %q", g.Meta.Name, g.Points[0].Name)
	}
	mustNot(t, gpx.String(), "<b>", "<Lamung>")
	if !json.Valid([]byte(js.String())) {
		t.Errorf("geojson invalid: %s", js.String())
	}
}
