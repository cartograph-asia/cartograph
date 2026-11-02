package db_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// profileWorld is newWorld with taken dates on adrian's four photos:
//
//	w.photos[0] beach    uploaded 2025-12-31 17:30 UTC, which is 2026-01-01 in Bangkok, taken 2025-06-10
//	w.photos[1] walking  uploaded 5 days ago, taken 2025-12-31 18:00 UTC, which is 2026-01-01 on the clock there
//	w.photos[4] beach    uploaded 1 hour ago, taken 2026-10-31
//	w.photos[5] jomtien  uploaded 30 minutes ago, no taken date
func profileWorld(t *testing.T) *world {
	t.Helper()
	w := newWorld(t)
	set := func(i int, utc string, local *string) {
		t.Helper()
		_, err := w.d.Pool.Exec(context.Background(),
			`update photos set taken_at = $2::timestamptz, taken_at_local = $3::timestamp where id = $1`, w.photos[i], utc, local)
		if err != nil {
			t.Fatal(err)
		}
	}
	local := "2026-01-01 01:00"
	set(0, "2025-06-10 05:00+00", nil)
	set(1, "2025-12-31 18:00+00", &local)
	set(4, "2026-10-31 08:00+00", nil)
	if _, err := w.d.Pool.Exec(context.Background(),
		`update photos set uploaded_at = '2025-12-31 17:30+00' where id = $1`, w.photos[0]); err != nil {
		t.Fatal(err)
	}
	return w
}

func ids(w *world, ps []views.PhotoItem) []string {
	var out []string
	for _, p := range ps {
		for i, id := range w.photos {
			if id.String() == p.ID {
				out = append(out, string(rune('0'+i)))
			}
		}
	}
	return out
}

func TestProfileOrdersAndFilters(t *testing.T) {
	w := profileWorld(t)
	ctx := context.Background()
	v := w.d.Views().WithNow(func() time.Time { return t0 })

	s, err := v.Surveyor(ctx, "adrian", views.SurveyorQuery{Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	if s.Stats.Photos != 4 || s.Stats.Surveyed != 3 || s.Stats.Plots != 3 || s.Stats.Regions != 1 {
		t.Fatalf("stats %+v", s.Stats)
	}
	// Three taken dates: the local date of the second photo is 2026-01-01.
	if s.Stats.Days != 3 {
		t.Fatalf("days %d", s.Stats.Days)
	}
	if len(s.Years) != 2 || s.Years[0] != (views.SurveyorYear{Year: 2026, Photos: 2}) || s.Years[1] != (views.SurveyorYear{Year: 2025, Photos: 1}) {
		t.Fatalf("years %+v", s.Years)
	}
	// Upload years are counted on Bangkok's clock, as the pages show dates.
	if len(s.UploadYears) != 1 || s.UploadYears[0] != (views.SurveyorYear{Year: 2026, Photos: 4}) {
		t.Fatalf("upload years %+v", s.UploadYears)
	}
	if len(s.Regions) != 1 || s.Regions[0].Plots != 3 || s.Regions[0].Photos != 4 || s.Regions[0].Land < 200 {
		t.Fatalf("regions %+v", s.Regions)
	}
	slug := s.Regions[0].Region.Slug
	if s.NoIndex || s.Canonical != "/surveyor/adrian" {
		t.Fatalf("plain view: noindex %v canonical %q", s.NoIndex, s.Canonical)
	}

	for _, c := range []struct {
		name string
		q    views.SurveyorQuery
		want string // photo numbers, in order
		path string
	}{
		{"newest", views.SurveyorQuery{}, "5410", "/surveyor/adrian"},
		{"oldest", views.SurveyorQuery{Order: views.OrderOldest}, "0145", "/surveyor/adrian?order=oldest"},
		{"taken", views.SurveyorQuery{Order: views.OrderTaken}, "4105", "/surveyor/adrian?order=taken"},
		{"year by local date", views.SurveyorQuery{Year: 2026}, "41", "/surveyor/adrian?year=2026"},
		{"year 2025", views.SurveyorQuery{Year: 2025}, "0", "/surveyor/adrian?year=2025"},
		{"year without photos", views.SurveyorQuery{Year: 2024}, "", "/surveyor/adrian?year=2024"},
		{"uploaded by Bangkok date", views.SurveyorQuery{Uploaded: 2026}, "5410", "/surveyor/adrian?uploaded=2026"},
		{"uploaded 2025", views.SurveyorQuery{Uploaded: 2025}, "", "/surveyor/adrian?uploaded=2025"},
		{"taken and uploaded", views.SurveyorQuery{Year: 2025, Uploaded: 2026}, "0", "/surveyor/adrian?uploaded=2026&year=2025"},
		{"region", views.SurveyorQuery{Region: slug}, "5410", "/surveyor/adrian?region=" + slug},
		{"region in capitals", views.SurveyorQuery{Region: strings.ToUpper(slug)}, "5410", "/surveyor/adrian?region=" + slug},
		{"one per plot", views.SurveyorQuery{OnePerPlot: true}, "541", "/surveyor/adrian?one=plot"},
		{"one per plot, oldest", views.SurveyorQuery{OnePerPlot: true, Order: views.OrderOldest}, "145", "/surveyor/adrian?one=plot&order=oldest"},
		// The filter is applied before the plot's newest photo is picked: in 2025 the beach shows photo 0.
		{"one per plot in a year", views.SurveyorQuery{OnePerPlot: true, Year: 2025}, "0", "/surveyor/adrian?one=plot&year=2025"},
	} {
		s, err := v.Surveyor(ctx, "adrian", c.q)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := strings.Join(ids(w, s.Photos), ""); got != c.want {
			t.Errorf("%s: photos %q, want %q", c.name, got, c.want)
		}
		if s.Pagination.Total != len(c.want) {
			t.Errorf("%s: total %d", c.name, s.Pagination.Total)
		}
		if s.Canonical != c.path || s.NoIndex != (c.name != "newest") || s.Feed != "/surveyor/adrian.rss" {
			t.Errorf("%s: canonical %q noindex %v feed %q", c.name, s.Canonical, s.NoIndex, s.Feed)
		}
	}

	// The breakdowns do not depend on the view.
	f, err := v.Surveyor(ctx, "adrian", views.SurveyorQuery{Year: 2025})
	if err != nil || f.FilterRegion != nil || len(f.Years) != 2 || f.Stats.Photos != 4 {
		t.Fatalf("filtered: %+v %v", f, err)
	}
	f, err = v.Surveyor(ctx, "adrian", views.SurveyorQuery{Region: slug})
	if err != nil || f.FilterRegion == nil || f.FilterRegion.Slug != slug {
		t.Fatalf("region filter: %+v %v", f.FilterRegion, err)
	}

	for _, q := range []views.SurveyorQuery{
		{Order: "random"}, {Region: "atlantis"}, {Page: 2}, {Page: 2, Year: 2025},
	} {
		if _, err := v.Surveyor(ctx, "adrian", q); !errors.Is(err, views.ErrNotFound) {
			t.Errorf("%+v: want ErrNotFound, got %v", q, err)
		}
	}

	// Somebody without photos has an empty, quiet profile.
	n, err := v.Surveyor(ctx, "no-photos", views.SurveyorQuery{})
	if err != nil || n.Stats.Photos != 0 || n.Stats.Days != 0 || len(n.Regions) != 0 || len(n.Years) != 0 || len(n.UploadYears) != 0 || n.HomeCoverage != nil {
		t.Fatalf("no photos: %+v %v", n, err)
	}
}

func TestProfileHiddenPhotosAndHomeCoverage(t *testing.T) {
	w := profileWorld(t)
	ctx := context.Background()
	v := w.d.Views().WithNow(func() time.Time { return t0 })

	// Home region: their own surveyed plots of it, not the region's claimed total.
	if _, err := w.d.Pool.Exec(ctx, `update surveyors set home_region_id = (select id from regions where kind = 'province') where name = 'adrian'`); err != nil {
		t.Fatal(err)
	}
	s, err := v.Surveyor(ctx, "adrian", views.SurveyorQuery{})
	if err != nil || s.HomeCoverage == nil || s.HomeCoverage.Claimed != 3 || s.HomeCoverage.Land < 200 {
		t.Fatalf("home coverage %+v %v", s.HomeCoverage, err)
	}
	if s.Profile.HomeRegion == nil {
		t.Fatal("home region missing")
	}

	// Hide the newest beach photo and the only jomtien photo.
	if _, err := w.d.Pool.Exec(ctx, `update photos set hidden_at = now(), hidden_reason = 'test' where id = any($1)`, w.photos[4:6]); err != nil {
		t.Fatal(err)
	}
	s, err = v.Surveyor(ctx, "adrian", views.SurveyorQuery{OnePerPlot: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(ids(w, s.Photos), ""); got != "10" {
		t.Errorf("one per plot after hiding: %q", got)
	}
	if s.Stats.Photos != 2 || s.Stats.Surveyed != 2 || s.HomeCoverage.Claimed != 2 || s.Years[0].Year != 2026 || s.Years[0].Photos != 1 {
		t.Errorf("after hiding: %+v %+v %+v", s.Stats, s.HomeCoverage, s.Years)
	}

	l, err := v.SurveyorPlots(ctx, "ADRIAN")
	if err != nil {
		t.Fatal(err)
	}
	if l.Surveyor.Name != "adrian" || len(l.Plots) != 2 {
		t.Fatalf("plots %+v", l)
	}
	for _, p := range l.Plots {
		if p.Plot.Cell == "" || p.Plot.Name() == "" || p.Lat < 12 || p.Lat > 13.5 || p.Lng < 100.5 || p.Lng > 101.2 || p.Photos != 1 || !p.First {
			t.Errorf("plot %+v", p)
		}
	}
	if _, err := v.SurveyorPlots(ctx, "nobody"); !errors.Is(err, views.ErrNotFound) {
		t.Errorf("unknown surveyor: %v", err)
	}
}
