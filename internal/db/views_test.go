package db_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/importer"
	"github.com/cartograph-asia/cartograph/internal/views"
)

var t0 = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

// fixtureDB is a database with the Pattaya fixture imported.
func fixtureDB(t *testing.T) *db.DB {
	t.Helper()
	d := db.NewTestDB(t)
	ctx := context.Background()
	osmPath, waterPath := importer.Fixture()
	st, err := importer.Regions(ctx, d, importer.RegionOptions{PBF: osmPath, Water: waterPath, BBox: importer.FixtureBBox})
	if err != nil {
		t.Fatal(err)
	}
	if st.Regions != 1 || st.LandCells < 200 || st.LandCells > 2000 {
		t.Fatalf("regions: %+v", st)
	}
	ps, err := importer.Places(ctx, d, importer.PlaceOptions{PBF: osmPath, BBox: importer.FixtureBBox})
	if err != nil || ps.Places < 3 {
		t.Fatalf("places: %+v %v", ps, err)
	}
	return d
}

type world struct {
	d         *db.DB
	surveyors map[string]uuid.UUID
	photos    []uuid.UUID // in upload order
}

func (w *world) surveyor(t *testing.T, name string, joined time.Time) {
	t.Helper()
	id := uuid.New()
	_, err := w.d.Pool.Exec(context.Background(),
		`insert into surveyors (id, name, email, joined_at) values ($1, $2, $3, $4)`, id, name, strings.ToLower(name)+"@example.test", joined)
	if err != nil {
		t.Fatal(err)
	}
	w.surveyors[name] = id
}

func (w *world) claim(t *testing.T, name string, lat, lng float64, at time.Time) (string, bool) {
	t.Helper()
	cell, err := h3.CellFromLatLng(lat, lng, h3.PlotRes)
	if err != nil {
		t.Fatal(err)
	}
	p5, _ := h3.Parent(cell, 5)
	p6, _ := h3.Parent(cell, 6)
	id := uuid.New()
	key := fmt.Sprintf("photos/%s/%s", at.Format("2006/01"), id)
	claimed, err := w.d.ClaimPhoto(context.Background(), db.NewPhoto{
		ID: id, Cell: cell, Parent5: p5, Parent6: p6, Surveyor: name, SurveyorID: w.surveyors[name],
		KeyLarge: key + "_large.jpg", KeyMedium: key + "_medium.jpg", KeyThumb: key + "_thumb.jpg",
		Width: 2000, Height: 1500, Lat: lat, Lng: lng, UploadedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.photos = append(w.photos, id)
	return cell, claimed
}

// Points in the fixture: Pattaya Beach, Walking Street, Nong Prue, Jomtien.
var (
	beach   = [2]float64{12.9346, 100.8840}
	walking = [2]float64{12.9270, 100.8720}
	nongPru = [2]float64{12.9442, 100.9397}
	jomtien = [2]float64{12.8870, 100.8760}
)

func newWorld(t *testing.T) *world {
	w := &world{d: fixtureDB(t), surveyors: map[string]uuid.UUID{}}
	w.surveyor(t, "adrian", t0.Add(-40*24*time.Hour))
	w.surveyor(t, "Somchai-K", t0.Add(-10*24*time.Hour))
	w.surveyor(t, "no-photos", t0.Add(-time.Hour))
	w.claim(t, "adrian", beach[0], beach[1], t0.Add(-35*24*time.Hour))    // 0 claims beach, outside 30 days
	w.claim(t, "adrian", walking[0], walking[1], t0.Add(-5*24*time.Hour)) // 1 claims walking street
	w.claim(t, "Somchai-K", beach[0], beach[1], t0.Add(-3*24*time.Hour))  // 2
	w.claim(t, "Somchai-K", nongPru[0], nongPru[1], t0.Add(-2*time.Hour)) // 3 claims nong prue
	w.claim(t, "adrian", beach[0], beach[1], t0.Add(-time.Hour))          // 4
	w.claim(t, "adrian", jomtien[0], jomtien[1], t0.Add(-30*time.Minute)) // 5 claims jomtien
	return w
}

func TestClaimSetsLocation(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	var region, claimed int
	var place string
	var last time.Time
	cell, _ := h3.CellFromLatLng(beach[0], beach[1], h3.PlotRes)
	err := w.d.Pool.QueryRow(ctx, `select pl.region_id, pl.place_name, r.claimed_cells, r.last_surveyed_at
		from plots pl join regions r on r.id = pl.region_id where pl.cell = $1`, cell).Scan(&region, &place, &claimed, &last)
	if err != nil {
		t.Fatal(err)
	}
	if place != "Pattaya" || claimed != 4 || !last.Equal(t0.Add(-30*time.Minute)) {
		t.Fatalf("place %q claimed %d last %v", place, claimed, last)
	}
	var firstID uuid.UUID
	if err := w.d.Pool.QueryRow(ctx, `select first_surveyor_id from plots where cell = $1`, cell).Scan(&firstID); err != nil || firstID != w.surveyors["adrian"] {
		t.Fatalf("first surveyor %v %v", firstID, err)
	}

	// Bangkok is outside the fixture's regions.
	_, err = w.d.ClaimPhoto(ctx, db.NewPhoto{ID: uuid.New(), Cell: "8864a4b161fffff", Surveyor: "adrian", KeyLarge: "l", KeyThumb: "t", UploadedAt: t0})
	if !errors.Is(err, db.ErrOutsideRegions) {
		t.Fatalf("outside: %v", err)
	}

	// A re-import keeps the counters right.
	osmPath, waterPath := importer.Fixture()
	if _, err := importer.Regions(ctx, w.d, importer.RegionOptions{PBF: osmPath, Water: waterPath, BBox: importer.FixtureBBox}); err != nil {
		t.Fatal(err)
	}
	if err := w.d.Pool.QueryRow(ctx, `select count(*), sum(claimed_cells) from regions where kind = 'province'`).Scan(&region, &claimed); err != nil || region != 1 || claimed != 4 {
		t.Fatalf("after re-import: %d regions, %d claimed, %v", region, claimed, err)
	}
	// The fixture's districts sit below Chon Buri, and the claimed plot has one.
	var districts int
	var district string
	if err := w.d.Pool.QueryRow(ctx, `select count(*) from regions where kind = 'district' and parent_id = $1`, region0(t, w)).Scan(&districts); err != nil || districts != 3 {
		t.Fatalf("districts: %d %v", districts, err)
	}
	if err := w.d.Pool.QueryRow(ctx, `select r.name_en from plots p join regions r on r.id = p.district_id where p.cell = $1`, cell).Scan(&district); err != nil || district != "Bang Lamung" {
		t.Fatalf("plot district %q %v", district, err)
	}
}

func TestBackfill(t *testing.T) {
	d := db.NewTestDB(t)
	ctx := context.Background()
	// A prototype plot claimed before any import.
	cell, _ := h3.CellFromLatLng(beach[0], beach[1], h3.PlotRes)
	id := uuid.New()
	if _, err := d.Pool.Exec(ctx, `insert into plots (cell, first_photo, first_surveyor, claimed_at, photo_count) values ($1, $2, 'proto', $3, 1)`, cell, id, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Pool.Exec(ctx, `insert into photos (id, cell, surveyor, key_large, key_thumb, lat, lng, uploaded_at) values ($1, $2, 'proto', 'l', 't', $3, $4, $5)`, id, cell, beach[0], beach[1], t0); err != nil {
		t.Fatal(err)
	}
	osmPath, waterPath := importer.Fixture()
	st, err := importer.Regions(ctx, d, importer.RegionOptions{PBF: osmPath, Water: waterPath, BBox: importer.FixtureBBox})
	if err != nil || st.PlotsBackfilled != 1 {
		t.Fatalf("regions %+v %v", st, err)
	}
	var place *string
	d.Pool.QueryRow(ctx, `select place_name from plots where cell = $1`, cell).Scan(&place)
	if place != nil { // no places yet: left empty, the pages show the province
		t.Fatalf("place before places import: %q", *place)
	}
	if _, err := importer.Places(ctx, d, importer.PlaceOptions{PBF: osmPath, BBox: importer.FixtureBBox}); err != nil {
		t.Fatal(err)
	}
	d.Pool.QueryRow(ctx, `select place_name from plots where cell = $1`, cell).Scan(&place)
	if place == nil || *place != "Pattaya" {
		t.Fatalf("place after places import: %v", deref(place))
	}
	var claimed int
	var last time.Time
	d.Pool.QueryRow(ctx, `select claimed_cells, last_surveyed_at from regions`).Scan(&claimed, &last)
	if claimed != 1 || !last.Equal(t0) {
		t.Fatalf("counts %d %v", claimed, last)
	}
}

func TestViews(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	v := w.d.Views().WithNow(func() time.Time { return t0 })

	h, err := v.Home(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if h.Counters != (views.Counters{Plots: 4, Photos: 6, Surveyors: 3, Regions: 1}) {
		t.Fatalf("counters %+v", h.Counters)
	}
	if len(h.Latest) != 6 || h.Latest[0].ID != w.photos[5].String() || !h.Latest[0].FirstSurvey || h.Latest[1].FirstSurvey {
		t.Fatalf("latest %+v", h.Latest)
	}
	it := h.Latest[0]
	if it.Plot.Place == "" || it.Plot.Region != (views.RegionRef{Slug: "chon-buri", NameEn: "Chon Buri", NameLocal: "ชลบุรี"}) ||
		it.Surveyor.Name != "adrian" || it.Keys.Medium == "" || it.Width != 2000 || !it.UploadedAt.Equal(t0.Add(-30*time.Minute)) {
		t.Fatalf("item %+v", it)
	}
	// No featured row: fallback to the newest first survey.
	if h.Featured == nil || h.Featured.Editorial || h.Featured.Photo.ID != w.photos[5].String() {
		t.Fatalf("featured fallback %+v", h.Featured)
	}
	if len(h.Regions) != 1 || h.Regions[0].Coverage.Claimed != 4 || h.Regions[0].Coverage.Land < 200 {
		t.Fatalf("regions %+v", h.Regions)
	}
	top := h.Regions[0].TopSurveyors // last 30 days: adrian 3, Somchai-K 2
	if len(top) != 2 || top[0].Surveyor.Name != "adrian" || top[0].Photos != 3 || top[0].Plots != 2 || top[1].Photos != 2 {
		t.Fatalf("top %+v", top)
	}
	if len(h.NewSurveyors) != 2 || h.NewSurveyors[0].Surveyor.Name != "Somchai-K" || h.NewSurveyors[0].FirstPhoto.ID != w.photos[2].String() {
		t.Fatalf("new surveyors %+v", h.NewSurveyors)
	}

	// Editorial featured photo.
	if _, err := w.d.Pool.Exec(ctx, `insert into featured (photo_id, body, starts_at) values ($1, 'Second Road at six.', $2)`, w.photos[2], t0.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := w.d.Pool.Exec(ctx, `insert into featured (photo_id, body, starts_at) values ($1, 'future', $2)`, w.photos[0], t0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if h, err = v.Home(ctx); err != nil || !h.Featured.Editorial || h.Featured.Photo.ID != w.photos[2].String() || h.Featured.Text != "Second Road at six." {
		t.Fatalf("featured %+v %v", h.Featured, err)
	}

	l, err := v.Latest(ctx, 0)
	if err != nil || len(l.Photos) != 6 || l.Pagination.Page != 1 || l.Canonical != "/latest" {
		t.Fatalf("latest %+v %v", l, err)
	}
	if _, err := v.Latest(ctx, 2); !errors.Is(err, views.ErrNotFound) {
		t.Fatalf("latest page 2: %v", err)
	}

	sl, err := v.Surveyors(ctx, 1)
	if err != nil || len(sl.Surveyors) != 3 || sl.Surveyors[0].Surveyor.Name != "no-photos" || sl.Surveyors[0].LastPhoto != nil ||
		sl.Surveyors[2].Surveyed != 3 || sl.Surveyors[2].Plots != 3 || sl.Surveyors[2].Photos != 4 || sl.Surveyors[2].LastPhoto.ID != w.photos[5].String() {
		t.Fatalf("surveyors %+v %v", sl, err)
	}

	// A country row is reachable by slug but never listed as a province.
	if _, err := w.d.UpsertRegions(ctx, []db.Region{{OSMID: 2067731, Kind: "country", AdminLevel: 2, Country: "th", Slug: "thailand",
		NameEn: "Thailand", Boundary: orb.MultiPolygon{{{{97, 5}, {106, 5}, {106, 21}, {97, 21}, {97, 5}}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := w.d.ReloadRegions(ctx); err != nil {
		t.Fatal(err)
	}
	rl, err := v.Regions(ctx)
	if err != nil || len(rl.Regions) != 1 || rl.Regions[0].Region.Slug != "chon-buri" {
		t.Fatalf("regions %+v %v", rl, err)
	}
	if c, err := v.Region(ctx, "thailand", 1); err != nil || c.Region.NameEn != "Thailand" || c.Kind != "country" {
		t.Fatalf("country page %+v %v", c, err)
	}

	s, err := v.Surveyor(ctx, "ADRIAN", views.SurveyorQuery{Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	if s.Profile.Name != "adrian" || s.Stats != (views.SurveyorStats{Surveyed: 3, Plots: 3, Photos: 4, Regions: 1}) || len(s.PlotCells) != 3 ||
		s.Bounds == nil || len(s.Photos) != 4 || s.Canonical != "/surveyor/adrian" || s.Feed != "/surveyor/adrian.rss" {
		t.Fatalf("surveyor %+v", s)
	}
	if _, err := v.Surveyor(ctx, "nobody", views.SurveyorQuery{Page: 1}); !errors.Is(err, views.ErrNotFound) {
		t.Fatalf("unknown surveyor: %v", err)
	}

	beachCell, _ := h3.CellFromLatLng(beach[0], beach[1], h3.PlotRes)
	p, err := v.Plot(ctx, strings.ToUpper(beachCell))
	if err != nil {
		t.Fatal(err)
	}
	if !p.Surveyed || p.PhotoCount != 3 || len(p.Photos) != 3 || p.Photos[0].ID != w.photos[0].String() ||
		p.FirstSurveyor.Name != "adrian" || p.Plot.Place != "Pattaya" || len(p.Neighbours) != 6 || p.Lat == 0 || p.NoIndex {
		t.Fatalf("plot %+v", p)
	}
	if p.Title != fmt.Sprintf("Pattaya, Bang Lamung · Chon Buri %d", p.Plot.Number) || p.Plot.Number == 0 || strings.Contains(p.Description, beachCell) {
		t.Fatalf("plot meta %q %q", p.Title, p.Description)
	}
	blank, err := v.Plot(ctx, p.Neighbours[0].Cell)
	if err != nil || blank.Surveyed || !blank.NoIndex || blank.Plot.Place == "" || blank.Plot.Region.Slug != "chon-buri" {
		t.Fatalf("blank plot %+v %v", blank, err)
	}
	for _, bad := range []string{"nonsense", "8864a4b161fffff" /* Bangkok, no region */, "85658bd3fffffff" /* res 5 */} {
		if _, err := v.Plot(ctx, bad); !errors.Is(err, views.ErrNotFound) {
			t.Errorf("plot %s: %v", bad, err)
		}
	}

	ph, err := v.Photo(ctx, w.photos[2].String())
	if err != nil {
		t.Fatal(err)
	}
	if ph.Position != 2 || ph.Count != 3 || ph.Prev.ID != w.photos[0].String() || ph.Next.ID != w.photos[4].String() ||
		ph.Licence != views.CCBYSA4 || ph.OG.ImageKey != ph.Photo.Keys.Medium || ph.OG.ImageWidth != 1200 || ph.OG.ImageHeight != 900 {
		t.Fatalf("photo %+v", ph)
	}
	if first, err := v.Photo(ctx, w.photos[0].String()); err != nil || first.Prev != nil || !first.Photo.FirstSurvey {
		t.Fatalf("first photo %+v %v", first, err)
	}
	if _, err := v.Photo(ctx, "not-a-uuid"); !errors.Is(err, views.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := v.Photo(ctx, uuid.NewString()); !errors.Is(err, views.ErrNotFound) {
		t.Fatal(err)
	}

	r, err := v.Region(ctx, "Chon-Buri", 1)
	if err != nil {
		t.Fatal(err)
	}
	if r.Kind != "province" || r.Counters.Plots != 4 || r.Counters.Photos != 6 || r.Counters.Surveyors != 2 || len(r.Latest) != 6 ||
		len(r.ActiveRecent) != 2 || r.ActiveAllTime[0].Photos != 4 || r.LastSurveyedAt == nil || r.Bounds.MinLat == 0 || r.Feed != "/region/chon-buri.rss" {
		t.Fatalf("region %+v", r)
	}
	if _, err := v.Region(ctx, "atlantis", 1); !errors.Is(err, views.ErrNotFound) {
		t.Fatal(err)
	}

	refs, err := v.RegionRefs(ctx)
	if err != nil || len(refs) != 1 || refs[0].Slug != "chon-buri" {
		t.Fatalf("refs %+v %v", refs, err)
	}

	for name, f := range map[string]func() (*views.Feed, error){
		"latest":   func() (*views.Feed, error) { return v.LatestFeed(ctx) },
		"region":   func() (*views.Feed, error) { return v.RegionFeed(ctx, "chon-buri") },
		"surveyor": func() (*views.Feed, error) { return v.SurveyorFeed(ctx, "somchai-k") },
	} {
		fd, err := f()
		if err != nil || len(fd.Items) == 0 || !fd.Updated.Equal(fd.Items[0].UploadedAt) || fd.Title == "" {
			t.Errorf("%s feed %+v %v", name, fd, err)
		}
	}
	if _, err := v.SurveyorFeed(ctx, "nobody"); !errors.Is(err, views.ErrNotFound) {
		t.Fatal(err)
	}
}

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// A Surveyor's bounds come from plot boundaries, never from the exact photo
// positions: two Surveyors with one photo each at different spots in the
// same plot get the same bounds, centred in the plot.
func TestSurveyorBoundsFromPlots(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	cell, _ := h3.CellFromLatLng(walking[0], walking[1], h3.PlotRes)
	lat, lng, err := h3.Centre(cell)
	if err != nil {
		t.Fatal(err)
	}
	spots := map[string][2]float64{"balcony-a": {lat + 0.0012, lng - 0.0009}, "balcony-b": {lat - 0.0011, lng + 0.0013}}
	v := w.d.Views().WithNow(func() time.Time { return t0 })
	var got []*views.Bounds
	for name, p := range spots {
		w.surveyor(t, name, t0.Add(-time.Hour))
		if c, _ := w.claim(t, name, p[0], p[1], t0); c != cell {
			t.Fatalf("%s: spot %v is in %s, not %s", name, p, c, cell)
		}
		s, err := v.Surveyor(ctx, name, views.SurveyorQuery{Page: 1})
		if err != nil || s.Bounds == nil {
			t.Fatalf("%s: %+v %v", name, s, err)
		}
		b := s.Bounds
		for _, x := range []float64{b.MinLat, b.MaxLat} {
			if x == p[0] {
				t.Fatalf("%s: bounds %+v contain the photo latitude", name, b)
			}
		}
		mid, _ := h3.CellFromLatLng((b.MinLat+b.MaxLat)/2, (b.MinLng+b.MaxLng)/2, h3.PlotRes)
		if mid != cell {
			t.Fatalf("%s: bounds %+v centred outside the plot", name, b)
		}
		got = append(got, b)
	}
	if *got[0] != *got[1] {
		t.Fatalf("bounds depend on the photo position: %+v %+v", got[0], got[1])
	}
}

func TestPlotNumbers(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	var n, min, max, distinct int
	if err := w.d.Pool.QueryRow(ctx, `select count(*), min(number), max(number), count(distinct number) from plot_numbers`).
		Scan(&n, &min, &max, &distinct); err != nil {
		t.Fatal(err)
	}
	// Every land cell of the fixture is numbered 1..n without gaps (claims
	// on land cells reuse their numbers).
	if n < 200 || min != 1 || max != n || distinct != n {
		t.Fatalf("numbers: %d rows, %d..%d, %d distinct", n, min, max, distinct)
	}

	// Neighbouring plots get nearby numbers: the median gap between a cell
	// and its numbered neighbours is small compared with the province.
	rows, err := w.d.Pool.Query(ctx, `select cell, number from plot_numbers`)
	if err != nil {
		t.Fatal(err)
	}
	num := map[string]int{}
	for rows.Next() {
		var c string
		var k int
		if err := rows.Scan(&c, &k); err != nil {
			t.Fatal(err)
		}
		num[c] = k
	}
	near := 0
	total := 0
	for c, k := range num {
		ring, _ := h3.Ring(c, 1)
		for _, r := range ring {
			if j, ok := num[r]; ok && r != c {
				total++
				if abs(j-k) <= 20 {
					near++
				}
			}
		}
	}
	if near*2 < total {
		t.Fatalf("only %d of %d neighbour pairs are within 20 numbers", near, total)
	}

	// A re-import keeps every number and adds none.
	osmPath, waterPath := importer.Fixture()
	if _, err := importer.Regions(ctx, w.d, importer.RegionOptions{PBF: osmPath, Water: waterPath, BBox: importer.FixtureBBox}); err != nil {
		t.Fatal(err)
	}
	same := 0
	if err := w.d.Pool.QueryRow(ctx, `select count(*) from plot_numbers`).Scan(&same); err != nil || same != n {
		t.Fatalf("after re-import: %d numbers, want %d (%v)", same, n, err)
	}
	beachCell, _ := h3.CellFromLatLng(beach[0], beach[1], h3.PlotRes)
	var k int
	if err := w.d.Pool.QueryRow(ctx, `select number from plot_numbers where cell = $1`, beachCell).Scan(&k); err != nil || k != num[beachCell] {
		t.Fatalf("beach number changed: %d → %d (%v)", num[beachCell], k, err)
	}

	// A surveyed plot without a number (here: its number removed, as for a
	// cell outside the land cells) gets the next free one on the next import.
	if _, err := w.d.Pool.Exec(ctx, `delete from plot_numbers where cell = $1`, beachCell); err != nil {
		t.Fatal(err)
	}
	if _, err := importer.Regions(ctx, w.d, importer.RegionOptions{PBF: osmPath, Water: waterPath, BBox: importer.FixtureBBox}); err != nil {
		t.Fatal(err)
	}
	if err := w.d.Pool.QueryRow(ctx, `select number from plot_numbers where cell = $1`, beachCell).Scan(&k); err != nil || k != n+1 {
		t.Fatalf("renumbered beach: %d, want %d (%v)", k, n+1, err)
	}

	// The friendly path resolves, regardless of case.
	cell, err := w.d.Views().PlotCell(ctx, "Chon-Buri", n+1)
	if err != nil || cell != beachCell {
		t.Fatalf("PlotCell: %q %v", cell, err)
	}
	if _, err := w.d.Views().PlotCell(ctx, "chon-buri", n+2); err != views.ErrNotFound {
		t.Fatalf("unknown number: %v", err)
	}
	p, err := w.d.Views().Plot(ctx, beachCell)
	if err != nil || p.Plot.Number != n+1 || p.Plot.Name() != fmt.Sprintf("Pattaya, Bang Lamung · Chon Buri %d", n+1) {
		t.Fatalf("plot page: %+v %v", p, err)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func TestPlaceTiers(t *testing.T) {
	d := db.NewTestDB(t)
	ctx := context.Background()
	osmPath, waterPath := importer.Fixture()
	if _, err := importer.Regions(ctx, d, importer.RegionOptions{PBF: osmPath, Water: waterPath, BBox: importer.FixtureBBox}); err != nil {
		t.Fatal(err)
	}
	// A plot in the countryside of the fixture: no place nearby yet, and its
	// district (an amphoe) is not urban.
	cell, _ := h3.CellFromLatLng(12.99, 100.95, h3.PlotRes)
	lat, lng, _ := h3.Centre(cell)
	at := func(km float64) (float64, float64) { return lat + km/111.195, lng }
	place := func(id int64, kind, name string, km float64) {
		la, ln := at(km)
		if _, err := d.Pool.Exec(ctx, `insert into places (osm_type, osm_id, kind, name_en, lat, lng, country) values ('n', $1, $2, $3, $4, $5, 'th')`,
			id, kind, name, la, ln); err != nil {
			t.Fatal(err)
		}
	}
	name := func() string {
		loc, err := d.LocatePlot(ctx, cell)
		if err != nil {
			t.Fatal(err)
		}
		if loc.District == "" {
			t.Fatal("no district for the plot")
		}
		return loc.PlaceEn
	}
	steps := []struct {
		id         int64
		kind, name string
		km         float64
		want       string
	}{
		{1, "neighbourhood", "Estate", 0.2, "Chon Buri"}, // outside a city: no quarters, estates or suburbs
		{2, "town", "Some Town", 4.8, "Some Town"},       // town within 5 km
		{3, "village", "Ban Near", 3, "Ban Near"},        // nearest of village/town
	}
	for _, s := range steps {
		place(s.id, s.kind, s.name, s.km)
		if got := name(); got != s.want {
			t.Fatalf("after adding %s %q at %.2f km: name %q, want %q", s.kind, s.name, s.km, got, s.want)
		}
	}
	reconfigure := func(sql string) {
		t.Helper()
		if _, err := d.Pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
		if err := d.MarkUrbanDistricts(ctx); err != nil {
			t.Fatal(err)
		}
		if err := d.ReloadRegions(ctx); err != nil {
			t.Fatal(err)
		}
	}
	// A province listed in urban_regions makes all its districts urban.
	reconfigure(`update place_naming set urban_regions = '{chon-buri}' where country = 'th'`)
	if got := name(); got != "Estate" {
		t.Fatalf("urban province: name %q, want Estate", got)
	}
	reconfigure(`update place_naming set urban_regions = '{bangkok}' where country = 'th'`)
	if got := name(); got != "Ban Near" {
		t.Fatalf("rural again: name %q, want Ban Near", got)
	}
	steps = steps[:0]
	steps = append(steps, []struct {
		id         int64
		kind, name string
		km         float64
		want       string
	}{
		{4, "city", "Big City", 4.9, "Estate"},               // a city within 5 km: urban
		{5, "suburb", "Khet Name", 2.9, "Estate"},            // a neighbourhood beats a suburb
		{6, "quarter", "Far Quarter", 3.2, "Estate"},         // quarters count within 3 km only
		{7, "quarter", "Khwaeng Near", 2.95, "Khwaeng Near"}, // a quarter beats both
	}...)
	for _, s := range steps {
		place(s.id, s.kind, s.name, s.km)
		if got := name(); got != s.want {
			t.Fatalf("after adding %s %q at %.2f km: name %q, want %q", s.kind, s.name, s.km, got, s.want)
		}
	}
	// The order is data: suburbs first if the table says so.
	reconfigure(`update place_naming set urban_kinds = '{suburb,quarter,neighbourhood}' where country = 'th'`)
	if got := name(); got != "Khet Name" {
		t.Fatalf("suburbs first: name %q, want Khet Name", got)
	}
}

// region0 is the id of the fixture's only province.
func region0(t *testing.T, w *world) int {
	t.Helper()
	var id int
	if err := w.d.Pool.QueryRow(context.Background(), `select id from regions where kind = 'province'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCountry(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	v := w.d.Views().WithNow(func() time.Time { return t0 })

	c, err := v.Country(ctx, "thailand", 1)
	if err != nil {
		t.Fatal(err)
	}
	// The country's figures are the sums over its provinces: here the one.
	prov, err := v.Region(ctx, "chon-buri", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Open || c.Coverage != prov.Coverage || c.Coverage.Claimed != 4 || c.Bounds != prov.Bounds || !c.LastSurveyedAt.Equal(*prov.LastSurveyedAt) {
		t.Fatalf("country %+v, province %+v", c, prov)
	}
	if c.Counters != (views.Counters{Plots: 4, Photos: 6, Surveyors: 2, Regions: 1}) {
		t.Fatalf("counters %+v", c.Counters)
	}
	if c.Canonical != "/thailand" || c.Feed != "/thailand.rss" || c.NoIndex || c.Country.NameEn != "Thailand" || c.Country.Lang != "th" {
		t.Fatalf("page %+v", c.Page)
	}
	if len(c.Provinces) != 1 || c.Provinces[0].Region.Slug != "chon-buri" || len(c.Latest) != 6 || c.Latest[0].ID != w.photos[5].String() {
		t.Fatalf("provinces %+v, latest %d", c.Provinces, len(c.Latest))
	}
	if len(c.ActiveRecent) != 2 || c.ActiveRecent[0].Surveyor.Name != "adrian" || c.ActiveRecent[0].Photos != 3 || len(c.ActiveAllTime) != 2 || c.ActiveAllTime[0].Photos != 4 {
		t.Fatalf("active %+v / %+v", c.ActiveRecent, c.ActiveAllTime)
	}
	if prov.Country.Slug != "thailand" || prov.Country.NameEn != "Thailand" {
		t.Fatalf("province page does not link its country: %+v", prov.Country)
	}

	// Case-insensitive; a page past the last one and unknown slugs are not found.
	if c, err := v.Country(ctx, "THAILAND", 1); err != nil || c.Canonical != "/thailand" {
		t.Fatalf("upper case: %+v %v", c, err)
	}
	for _, slug := range []string{"th", "atlantis", "chon-buri"} {
		if _, err := v.Country(ctx, slug, 1); !errors.Is(err, views.ErrNotFound) {
			t.Errorf("Country(%q): %v", slug, err)
		}
	}
	if _, err := v.Country(ctx, "thailand", 2); !errors.Is(err, views.ErrNotFound) {
		t.Errorf("page 2: %v", err)
	}

	// In the list, no regions: a short page, not indexed, no feed.
	vn, err := v.Country(ctx, "vietnam", 1)
	if err != nil || vn.Open || !vn.NoIndex || vn.Feed != "" || vn.Canonical != "/vietnam" || vn.Latest != nil || vn.Provinces != nil {
		t.Fatalf("vietnam %+v %v", vn, err)
	}
	if _, err := v.Country(ctx, "vietnam", 2); !errors.Is(err, views.ErrNotFound) {
		t.Errorf("vietnam page 2: %v", err)
	}
	if _, err := v.CountryFeed(ctx, "vietnam"); !errors.Is(err, views.ErrNotFound) {
		t.Errorf("vietnam feed: %v", err)
	}

	// The feed holds this country's photos only: add a province elsewhere.
	if _, err := w.d.UpsertRegions(ctx, []db.Region{{OSMID: 900001, Kind: "province", AdminLevel: 4, Country: "vn", Slug: "ha-noi", NameEn: "Ha Noi",
		Boundary: orb.MultiPolygon{{{{105.5, 20.8}, {106.0, 20.8}, {106.0, 21.3}, {105.5, 21.3}, {105.5, 20.8}}}}, LandCells: 50}}); err != nil {
		t.Fatal(err)
	}
	if err := w.d.ReloadRegions(ctx); err != nil {
		t.Fatal(err)
	}
	w.surveyor(t, "ha-noi-walker", t0.Add(-time.Hour))
	if _, claimed := w.claim(t, "ha-noi-walker", 21.03, 105.85, t0); !claimed {
		t.Fatal("the plot in Ha Noi was not claimed")
	}
	f, err := v.CountryFeed(ctx, "thailand")
	if err != nil || len(f.Items) != 6 || f.Canonical != "/thailand" {
		t.Fatalf("thailand feed: %d items, %+v, %v", len(f.Items), f.Page, err)
	}
	if f, err = v.CountryFeed(ctx, "vietnam"); err != nil || len(f.Items) != 1 || f.Items[0].Surveyor.Name != "ha-noi-walker" {
		t.Fatalf("vietnam feed: %+v %v", f, err)
	}
	if vn, err = v.Country(ctx, "vietnam", 1); err != nil || !vn.Open || vn.Coverage.Claimed != 1 || vn.Coverage.Land != 50 || vn.Counters.Photos != 1 {
		t.Fatalf("vietnam %+v %v", vn, err)
	}
	if c, err = v.Country(ctx, "thailand", 1); err != nil || c.Counters.Photos != 6 || c.Coverage.Claimed != 4 {
		t.Fatalf("thailand after Vietnam: %+v %v", c.Counters, err)
	}

	// /regions: every country in the list, the open ones with their figures.
	rl, err := v.Regions(ctx)
	if err != nil || len(rl.Countries) < 17 || len(rl.Regions) != 2 {
		t.Fatalf("regions %+v %v", rl, err)
	}
	open := map[string]views.CountrySummary{}
	for _, s := range rl.Countries {
		if s.Open {
			open[s.Country.Slug] = s
		}
	}
	if len(open) != 2 || open["thailand"].Coverage.Claimed != 4 || open["vietnam"].Coverage.Claimed != 1 ||
		len(open["thailand"].TopSurveyors) != 2 || open["thailand"].TopSurveyors[0].Surveyor.Name != "adrian" {
		t.Fatalf("open countries %+v", open)
	}
}
