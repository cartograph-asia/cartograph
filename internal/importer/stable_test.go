package importer

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/db"
)

// communes are zz's settings with Vietnam's district words, so that
// stripping makes siblings collide (decision 7).
func communes() countries.Import {
	imp := zzSettings()
	imp.District = countries.Level{AdminLevels: []int{6}, Kind: "commune",
		KindByAffix: map[string]string{"Phường": "ward"},
		Strip:       countries.Affixes{EnPrefix: []string{"Xã ", "Phường "}, LocalPrefix: []string{"Xã ", "Phường "}}}
	return imp
}

// regionsRun runs Regions for zz with imp over pbf; the land plots of a
// first dry run become imp.LandPlots, so check 4 passes.
func regionsRun(ctx context.Context, t *testing.T, d *db.DB, imp *countries.Import, pbf string, dry bool) (*RegionStats, string) {
	t.Helper()
	_, waterPath := Fixture()
	if imp.LandPlots == 0 {
		st, _ := Regions(ctx, d, Options{Country: "zz", Settings: imp, PBF: pbf, Water: waterPath, DryRun: true})
		if st == nil || st.LandCells == 0 {
			t.Fatalf("no land plots: %+v", st)
		}
		imp.LandPlots = st.LandCells
	}
	var report bytes.Buffer
	st, err := Regions(ctx, d, Options{Country: "zz", Settings: imp, PBF: pbf, Water: waterPath, DryRun: dry, Report: &report})
	if err != nil {
		t.Fatalf("%v\n%s", err, report.String())
	}
	return st, report.String()
}

func storedDistricts(ctx context.Context, t *testing.T, d *db.DB) map[int64]string {
	t.Helper()
	rows, err := d.Pool.Query(ctx, `select r.osm_id, r.slug || ' ' || r.name_en || ' ' || coalesce(r.name_local, 'NULL') || ' in r' || p.osm_id
		from regions r join regions p on p.id = r.parent_id where r.country = 'zz' and r.kind = 'district'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var s string
		if err := rows.Scan(&id, &s); err != nil {
			t.Fatal(err)
		}
		out[id] = s
	}
	return out
}

// A second import over the first changes nothing, also for districts that
// decision 7 gave their affixes or their ids; a new third sibling is
// resolved against the stored slugs and leaves the others as they are.
func TestReimportKeepsDuplicateSlugs(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	imp := communes()
	province := testRel{id: 900000101, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-01", "name", "Zed One"}, outer: []rect{{101.10, 12.90, 101.40, 12.98}}}
	commune := func(id int64, name string, x float64) testRel {
		return testRel{id: id, tags: []string{"admin_level", "6", "name", name}, outer: []rect{{x, 12.91, x + 0.04, 12.97}}}
	}
	rels := []testRel{province,
		commune(900000111, "Xã Tân Thành", 101.11),
		commune(900000112, "Phường Tân Thành", 101.16),
		commune(900000113, "Xã An Phú", 101.21),
		commune(900000114, "Xã Ân Phú", 101.26), // folds to the same slug, words and all
	}
	regionsRun(ctx, t, d, &imp, writeTestOSM(t, rels, nil), false)
	first := storedDistricts(ctx, t, d)
	want := map[int64]string{
		900000111: "zed-one-zz-xa-tan-thanh Xã Tân Thành NULL in r900000101",
		900000112: "zed-one-zz-phuong-tan-thanh Phường Tân Thành NULL in r900000101",
		900000113: "zed-one-zz-xa-an-phu-900000113 Xã An Phú NULL in r900000101",
		900000114: "zed-one-zz-xa-an-phu-900000114 Xã Ân Phú NULL in r900000101",
	}
	for id, w := range want {
		if first[id] != w {
			t.Errorf("first run, r%d: %q, want %q", id, first[id], w)
		}
	}

	_, report := regionsRun(ctx, t, d, &imp, writeTestOSM(t, rels, nil), true)
	for _, w := range []string{"Provinces: 1 (0 added, 0 updated, 1 unchanged)", "Districts: 4 (0 added, 0 updated, 4 unchanged", "New slugs: 0"} {
		if !strings.Contains(report, w) {
			t.Errorf("second run's report lacks %q:\n%s", w, report)
		}
	}
	all := sums(t, d, "")
	regionsRun(ctx, t, d, &imp, writeTestOSM(t, rels, nil), false)
	sameSums(t, "after the second run", all, sums(t, d, ""))

	// A third sibling: it folds to Tân Thành's slug with its word, so it
	// takes its id; the stored ones stay.
	rels = append(rels, commune(900000115, "Xã Tân Thạnh", 101.31))
	imp.LandPlots = 0
	_, report = regionsRun(ctx, t, d, &imp, writeTestOSM(t, rels, nil), true)
	for _, w := range []string{"Districts: 5 (1 added, 0 updated, 4 unchanged", "district r900000115: zed-one-zz-xa-tan-thanh-900000115"} {
		if !strings.Contains(report, w) {
			t.Errorf("third sibling's report lacks %q:\n%s", w, report)
		}
	}
	regionsRun(ctx, t, d, &imp, writeTestOSM(t, rels, nil), false)
	got := storedDistricts(ctx, t, d)
	want[900000115] = "zed-one-zz-xa-tan-thanh-900000115 Xã Tân Thạnh NULL in r900000101"
	for id, w := range want {
		if got[id] != w {
			t.Errorf("after the third sibling, r%d: %q, want %q", id, got[id], w)
		}
	}
	_, report = regionsRun(ctx, t, d, &imp, writeTestOSM(t, rels, nil), true)
	if !strings.Contains(report, "Districts: 5 (0 added, 0 updated, 5 unchanged") {
		t.Errorf("run after the third sibling:\n%s", report)
	}
}

// A district belongs to the province holding most of its land cells, not
// the one holding its centre; a stored district keeps its parent and slug,
// and the report names the parent a fresh import would give.
func TestDistrictParentByLandCells(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	imp := zzSettings()
	imp.Province.Count = [2]int{2, 2}
	imp.District.Exclude = []int64{900000209}
	rels := []testRel{
		{id: 900000201, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-01", "name", "Zed A"}, outer: []rect{{101.00, 12.50, 101.30, 13.10}}},
		{id: 900000202, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-02", "name", "Zed B"}, outer: []rect{{101.30, 12.50, 101.60, 13.10}}},
		// About 30 % of it in A, where its centroid lies (a thin arm), 70 %
		// in B (a narrow strip along the border).
		{id: 900000203, tags: []string{"admin_level", "6", "name", "Straddle"}, outer: []rect{
			{101.10, 12.78, 101.30, 12.82}, {101.30, 12.50, 101.33, 13.10}}},
		// Smaller than a plot: no land cell, so its centre decides.
		{id: 900000204, tags: []string{"admin_level", "6", "name", "Speck"}, outer: []rect{{101.05, 12.60, 101.0512, 12.6012}}},
		{id: 900000205, tags: []string{"admin_level", "6", "name", "Plain"}, outer: []rect{{101.40, 12.60, 101.50, 12.70}}},
		// Matches the level, excluded by the settings.
		{id: 900000209, tags: []string{"admin_level", "6", "name", "Elsewhere"}, outer: []rect{{101.40, 12.80, 101.50, 12.90}}},
	}
	pbf := writeTestOSM(t, rels, nil)
	_, report := regionsRun(ctx, t, d, &imp, pbf, true)
	for _, w := range []string{
		"r900000203 zed-b-zz-straddle \"Straddle\": land cells in r900000202 Zed B, centre in r900000201 Zed A",
		"Relations excluded: 1, by settings (Level.Exclude) [900000209]",
		"Districts whose parent would change (information only: a stored district keeps its parent and slug): 0",
	} {
		if !strings.Contains(report, w) {
			t.Errorf("fresh report lacks %q:\n%s", w, report)
		}
	}
	regionsRun(ctx, t, d, &imp, pbf, false)
	want := map[int64]string{
		900000203: "zed-b-zz-straddle Straddle NULL in r900000202",
		900000204: "zed-a-zz-speck Speck NULL in r900000201",
		900000205: "zed-b-zz-plain Plain NULL in r900000202",
	}
	if got := storedDistricts(ctx, t, d); !mapsEqual(got, want) {
		t.Errorf("fresh import: %v, want %v", got, want)
	}

	// As the centroid rule stored it: under A, with A's slug.
	if _, err := d.Pool.Exec(ctx, `update regions set slug = 'zed-a-zz-straddle',
		parent_id = (select id from regions where osm_id = 900000201) where osm_id = 900000203`); err != nil {
		t.Fatal(err)
	}
	all := sums(t, d, "")
	_, report = regionsRun(ctx, t, d, &imp, pbf, true)
	for _, w := range []string{
		"Districts: 3 (0 added, 0 updated, 3 unchanged",
		"Districts whose parent would change (information only: a stored district keeps its parent and slug): 1",
		"r900000203 zed-a-zz-straddle \"Straddle\": stays in r900000201 Zed A, a fresh import gives r900000202 Zed B",
	} {
		if !strings.Contains(report, w) {
			t.Errorf("stored report lacks %q:\n%s", w, report)
		}
	}
	regionsRun(ctx, t, d, &imp, pbf, false)
	sameSums(t, "a run over a district stored under its centre's province", all, sums(t, d, ""))
}

func mapsEqual(a, b map[int64]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// assignParents: the largest share wins, a tie goes to the lower OSM id, no
// share leaves the centre's province, a stored parent stays.
func TestAssignParents(t *testing.T) {
	a, b, c := &province{osmID: 3}, &province{osmID: 2}, &province{osmID: 1}
	set := &regionSet{provinces: []*province{a, b, c}}
	dist := func(id int64, centred *province, share map[*province]int) *district {
		dd := &district{osmID: id, centred: centred, province: centred, share: share}
		set.districts = append(set.districts, dd)
		return dd
	}
	split := dist(10, a, map[*province]int{a: 3, b: 4, c: 3}) // no majority: the largest share
	tie := dist(11, a, map[*province]int{a: 5, b: 5})         // a tie: the lower OSM id
	none := dist(12, c, nil)                                  // no land cell: the centre
	kept := dist(13, a, map[*province]int{a: 1, c: 9})        // stored under a
	other := dist(14, a, map[*province]int{b: 9})             // stored for another country
	gone := dist(15, a, map[*province]int{b: 9})              // stored under a province not in the run
	parent, goneParent := 1, 2
	byID := map[int]*db.StoredRegion{1: {ID: 1, OSMID: 3, Country: "zz"}, 2: {ID: 2, OSMID: 99, Country: "zz"}}
	stored := map[int64]*db.StoredRegion{
		13: {OSMID: 13, Country: "zz", ParentID: &parent},
		14: {OSMID: 14, Country: "yy", ParentID: &parent},
		15: {OSMID: 15, Country: "zz", ParentID: &goneParent},
	}
	assignParents(set, country{Code: "zz"}, stored, byID)
	for _, x := range []struct {
		d             *district
		parent, fresh *province
	}{{split, b, b}, {tie, b, b}, {none, c, c}, {kept, a, c}, {other, b, b}, {gone, b, b}} {
		if x.d.province != x.parent || x.d.majority != x.fresh {
			t.Errorf("r%d: parent r%d, majority r%d; want r%d, r%d", x.d.osmID, x.d.province.osmID, x.d.majority.osmID, x.parent.osmID, x.fresh.osmID)
		}
		if !slices.Contains(x.d.province.districts, x.d) {
			t.Errorf("r%d is not among its province's districts", x.d.osmID)
		}
	}
}

// A relation in Level.Exclude is never read, and the run counts it.
func TestLevelsExclude(t *testing.T) {
	imp := zzSettings()
	imp.District.Exclude = []int64{6753263}
	imp.Province.Exclude = []int64{2}
	path := writeTestOSM(t, []testRel{
		{id: 1, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-01", "name", "One"}, outer: []rect{{110, 8, 116, 12}}},
		{id: 2, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-02", "name", "Two"}, outer: []rect{{100, 8, 104, 12}}},
		{id: 6753263, tags: []string{"admin_level", "6", "name", "Nansha District"}, outer: []rect{{112, 9, 114, 11}}},
		{id: 7, tags: []string{"admin_level", "6", "name", "Seven"}, outer: []rect{{111, 9, 111.5, 9.5}}},
	}, nil)
	set := readTest(t, path, imp)
	if got := provinceIDs(set); !slices.Equal(got, []int64{1}) {
		t.Errorf("provinces %v", got)
	}
	if got := districtLevels(set); len(got) != 1 || got[7] != 6 {
		t.Errorf("districts %v", got)
	}
	if !slices.Equal(set.excluded, []int64{2, 6753263}) || set.relations != 2 {
		t.Errorf("excluded %v, relations read %d", set.excluded, set.relations)
	}
}
