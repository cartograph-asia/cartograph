package importer

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/db"
)

// keepDistricts: more than half of the land cells in the run's provinces
// keeps a district, wherever its centre lies; without a land cell the
// centre decides; a stored district stays.
func TestKeepDistricts(t *testing.T) {
	a := &province{osmID: 1}
	set := &regionSet{provinces: []*province{a}}
	dist := func(id int64, centred *province, in, other, none int) *district {
		d := &district{osmID: id, centred: centred, province: centred, in: in, other: other, none: none}
		if centred != nil {
			set.districts = append(set.districts, d)
		} else {
			set.outside = append(set.outside, d)
			set.dropped.Outside++
		}
		return d
	}
	crescent := dist(10, nil, 118, 0, 34) // centre outside, most land inside: rescued
	half := dist(11, a, 50, 50, 0)        // exactly half is not more than half
	sliver := dist(12, a, 3, 97, 0)       // a neighbour's district reaching in
	unstored := dist(13, nil, 3, 0, 97)   // centre and land outside
	island := dist(14, a, 0, 0, 0)        // no land cell: the centre keeps it
	sea := dist(15, nil, 0, 0, 0)         // no land cell, centre outside
	stored := dist(16, a, 10, 90, 0)      // stored for zz: stays
	other := dist(17, a, 10, 90, 0)       // stored for another country
	plain := dist(18, a, 100, 0, 0)
	st := map[int64]*db.StoredRegion{16: {OSMID: 16, Country: "zz"}, 17: {OSMID: 17, Country: "yy"}}
	keepDistricts(set, country{Code: "zz"}, st)

	var kept []int64
	for _, d := range set.districts {
		kept = append(kept, d.osmID)
	}
	if want := []int64{10, 14, 16, 18}; !slices.Equal(kept, want) {
		t.Errorf("kept %v, want %v", kept, want)
	}
	if !crescent.rescued || island.rescued || plain.rescued {
		t.Errorf("rescued: crescent %v, island %v, plain %v", crescent.rescued, island.rescued, plain.rescued)
	}
	if !slices.Equal(set.landDropped, []*district{half, sliver, other}) || !slices.Equal(set.storedKept, []*district{stored}) {
		t.Errorf("land dropped %v, stored kept %v", set.landDropped, set.storedKept)
	}
	// 13 and 15 were counted outside when read, 10 left that count; 11, 12
	// and 17 joined it.
	if set.dropped.Outside != 5 || set.outside != nil {
		t.Errorf("outside %d, %v", set.dropped.Outside, set.outside)
	}
	_, _ = unstored, sea
}

// The synthetic layout of TestDistrictsKeptByLand: zz's province A
// (102.00–102.30 E, 10.00–10.30 N, no water polygons there, so all land)
// and S (in the sea off Pattaya, inside the fixture's water polygons); yy's
// province Y east of A (102.30–102.60 E), imported first.
func keepLayout(t *testing.T) string {
	t.Helper()
	d := func(id int64, name string, outer ...rect) testRel {
		return testRel{id: id, tags: []string{"admin_level", "6", "name", name}, outer: outer}
	}
	return writeTestOSM(t, []testRel{
		{id: 900000301, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-01", "name", "Zed A"}, outer: []rect{{102.00, 10.00, 102.30, 10.30}}},
		{id: 900000302, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-02", "name", "Zed S"}, outer: []rect{{100.83, 12.88, 100.86, 12.97}}},
		{id: 900000309, tags: []string{"admin_level", "4", "ISO3166-2", "YY-01", "name", "Why"}, outer: []rect{{102.30, 10.00, 102.60, 10.30}}},
		// A crescent (Taiwan's Dajia): a U inside A and a strip west of A,
		// in no province. Its centroid lies in the U's gap, outside it, and
		// the first vertex is the strip's (the assembly puts the last
		// member way's polygon first): no province holds its centre. About
		// 118 land cells in A, 34 in none.
		d(900000311, "Crescent",
			rect{102.00, 10.05, 102.02, 10.20}, rect{102.02, 10.05, 102.10, 10.07}, rect{102.08, 10.07, 102.10, 10.20},
			rect{101.985, 10.05, 101.999, 10.20}),
		// 70 % in A, 30 % in Y, centre in A.
		d(900000312, "Straddle", rect{102.244, 10.02, 102.324, 10.07}),
		// 70 % in Y and a long arm in A, 30 %: its centroid lies outside it,
		// and the first vertex, the arm's, in A.
		d(900000313, "Reach", rect{102.30, 10.08, 102.3655, 10.28}, rect{102.02, 10.26, 102.30, 10.28}),
		// 30 % in A, 70 % in Y, centre in Y.
		d(900000314, "Neighbour", rect{102.276, 10.10, 102.356, 10.15}),
		// Smaller than a plot: no land cell; in A, and in no province.
		d(900000315, "Speck", rect{102.15, 10.25, 102.1512, 10.2512}),
		d(900000316, "Speck Out", rect{101.95, 10.25, 101.9512, 10.2512}),
		// Sea only, in S: no land cell, its centre in S.
		d(900000317, "Sea", rect{100.835, 12.90, 100.85, 12.95}),
	}, nil)
}

func keepRun(ctx context.Context, t *testing.T, d *db.DB, code string, imp countries.Import, pbf string, dry bool) string {
	t.Helper()
	_, waterPath := Fixture()
	var report bytes.Buffer
	o := Options{Country: code, Settings: &imp, PBF: pbf, Water: waterPath, BBox: "100.8,9.9,102.7,13.1", DryRun: dry, Report: &report}
	if _, err := Regions(ctx, d, o); err != nil {
		t.Fatalf("%s: %v\n%s", code, err, report.String())
	}
	return report.String()
}

// A district is kept by the majority of its land cells: the crescent whose
// centre lies outside is rescued, a district reaching into A by 30 % is
// dropped, one 70 % in A is kept; without a land cell the centre decides.
// A re-import changes nothing; a stored district the rule drops stays.
func TestDistrictsKeptByLand(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	pbf := keepLayout(t)
	yy := zzN("")
	yy.Province.ISOPrefix = "YY"
	keepRun(ctx, t, d, "yy", yy, pbf, false)

	zz := zzSettings()
	zz.Province.Count = [2]int{2, 2}
	report := keepRun(ctx, t, d, "zz", zz, pbf, true)
	wantLines(t, "fresh", report,
		"Districts: 4 (4 added, 0 updated, 0 unchanged",
		"outside the provinces (most land cells, else the centre) 3,",
		"Districts kept by their land cells, their centre outside the provinces: 1\n  - r900000311 zed-a-zz-crescent \"Crescent\", in r900000301 Zed A: ",
		"Districts dropped by their land cells, their centre in a province: 1\n  - r900000313 \"Reach\", centre in r900000301 Zed A: ",
		"Stored districts kept, although a fresh import drops them by their land cells (information only): 0")
	// The splits: the crescent mostly in A, its strip in none; Reach 30/70.
	for _, l := range strings.Split(report, "\n") {
		var in, all, other, none int
		switch {
		case !strings.Contains(l, "land plots in the run's provinces"):
		case strings.Contains(l, "r900000311 zed-a-zz-crescent"):
			parseSplit(t, l, &in, &all, &other, &none)
			if other != 0 || none == 0 || 2*in <= all || in+other+none != all {
				t.Errorf("crescent: %s", l)
			}
		case strings.Contains(l, "r900000313 \"Reach\""):
			parseSplit(t, l, &in, &all, &other, &none)
			if none != 0 || f(in, all) < 0.25 || f(in, all) > 0.35 || in+other != all {
				t.Errorf("reach: %s", l)
			}
		}
	}

	// Speck and Sea have no land cell (the centre kept them), Neighbour's
	// land lies mostly in Y.
	_, waterPath := Fixture()
	pl, err := planRegions(ctx, d, Options{Country: "zz", Settings: &zz, PBF: pbf, Water: waterPath, BBox: "100.8,9.9,102.7,13.1", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(pl.set.districts, func(x *district) bool { return x.osmID == 900000317 }) {
		t.Errorf("Sea is not kept")
	}
	for _, x := range append(slices.Clone(pl.set.districts), pl.set.landDropped...) {
		land := x.in + x.other + x.none
		switch x.osmID {
		case 900000315, 900000317:
			if land != 0 {
				t.Errorf("r%d %s: %d land cells, want none", x.osmID, x.names.En, land)
			}
		case 900000312:
			if r := f(x.in, land); r < 0.65 || r > 0.75 || x.none != 0 {
				t.Errorf("Straddle: %d in, %d other, %d none", x.in, x.other, x.none)
			}
		}
	}

	keepRun(ctx, t, d, "zz", zz, pbf, false)
	want := map[int64]string{
		900000311: "zed-a-zz-crescent Crescent NULL in r900000301",
		900000312: "zed-a-zz-straddle Straddle NULL in r900000301",
		900000315: "zed-a-zz-speck Speck NULL in r900000301",
		900000317: "zed-s-zz-sea Sea NULL in r900000302",
	}
	if got := storedDistricts(ctx, t, d); !mapsEqual(got, want) {
		t.Errorf("stored: %v, want %v", got, want)
	}

	// A re-import changes nothing.
	all := sums(t, d, "")
	report = keepRun(ctx, t, d, "zz", zz, pbf, true)
	wantLines(t, "re-run", report, "Districts: 4 (0 added, 0 updated, 4 unchanged")
	keepRun(ctx, t, d, "zz", zz, pbf, false)
	sameSums(t, "a re-import", all, sums(t, d, ""))

	// Reach stored for zz (as the centre rule stored it): it stays, under
	// its stored parent, and the report says a fresh import drops it.
	if _, err := d.Pool.Exec(ctx, `insert into regions (osm_id, admin_level, country, slug, name_en, boundary,
		min_lat, min_lng, max_lat, max_lng, land_cells, kind, parent_id)
		select 900000313, admin_level, country, 'zed-a-zz-reach', 'Reach', boundary, min_lat, min_lng, max_lat, max_lng, 0, kind, parent_id
		from regions where osm_id = 900000312`); err != nil {
		t.Fatal(err)
	}
	report = keepRun(ctx, t, d, "zz", zz, pbf, true)
	wantLines(t, "stored", report,
		"Districts: 5 (0 added, 1 updated, 4 unchanged",
		"r900000313 zed-a-zz-reach: boundary",
		"Districts dropped by their land cells, their centre in a province: 0",
		"Stored districts kept, although a fresh import drops them by their land cells (information only): 1\n  - r900000313 \"Reach\", centre in r900000301 Zed A: ")
	keepRun(ctx, t, d, "zz", zz, pbf, false)
	want[900000313] = "zed-a-zz-reach Reach NULL in r900000301"
	if got := storedDistricts(ctx, t, d); !mapsEqual(got, want) {
		t.Errorf("stored with Reach: %v, want %v", got, want)
	}
	all = sums(t, d, "")
	report = keepRun(ctx, t, d, "zz", zz, pbf, true)
	wantLines(t, "stored, again", report, "Districts: 5 (0 added, 0 updated, 5 unchanged")
	keepRun(ctx, t, d, "zz", zz, pbf, false)
	sameSums(t, "a re-import with Reach stored", all, sums(t, d, ""))
}

func f(a, b int) float64 { return float64(a) / float64(b) }

// parseSplit reads "… N of M land plots in the run's provinces, O in
// another country's, X in none" from a report line.
func parseSplit(t *testing.T, line string, in, all, other, none *int) {
	t.Helper()
	i := strings.LastIndex(line, ": ")
	if _, err := fmt.Sscanf(line[i+2:], "%d of %d land plots in the run's provinces, %d in another country's, %d in none", in, all, other, none); err != nil {
		t.Fatalf("%q: %v", line, err)
	}
}
