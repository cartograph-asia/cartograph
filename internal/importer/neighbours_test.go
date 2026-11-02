package importer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/geo"
)

// The province ZZ-01 (r100, 110.0–110.2 E, 10.0–10.1 N) is two ways: w11
// (south and east side) in the country's own file, w12 (north and west
// side) only in the neighbour's, as a border line a Geofabrik extract cuts.
// The own file also holds a town inside the province; the neighbour's file
// holds a village inside it, its own province relation (ZZ-02, r300) and
// that relation's way.
const (
	ownOSM = `<?xml version="1.0" encoding="UTF-8"?>
<osm version="0.6" generator="test">
 <node id="1" version="1" lat="10.0" lon="110.0"/>
 <node id="2" version="1" lat="10.0" lon="110.2"/>
 <node id="3" version="1" lat="10.1" lon="110.2"/>
 <node id="201" version="1" lat="10.05" lon="110.10"><tag k="place" v="town"/><tag k="name" v="Own Town"/></node>
 <way id="11" version="1"><nd ref="1"/><nd ref="2"/><nd ref="3"/></way>
 <relation id="100" version="1">
  <member type="way" ref="11" role="outer"/><member type="way" ref="12" role="outer"/>
  <tag k="boundary" v="administrative"/><tag k="type" v="boundary"/><tag k="admin_level" v="4"/>
  <tag k="ISO3166-2" v="ZZ-01"/><tag k="name" v="Border Prefecture"/>
 </relation>
</osm>
`
	neighbourOSM = `<?xml version="1.0" encoding="UTF-8"?>
<osm version="0.6" generator="test">
 <node id="1" version="1" lat="10.0" lon="110.0"/>
 <node id="3" version="1" lat="10.1" lon="110.2"/>
 <node id="4" version="1" lat="10.1" lon="110.0"/>
 <node id="31" version="1" lat="10.0" lon="110.3"/>
 <node id="32" version="1" lat="10.0" lon="110.4"/>
 <node id="33" version="1" lat="10.1" lon="110.4"/>
 <node id="34" version="1" lat="10.1" lon="110.3"/>
 <node id="202" version="1" lat="10.05" lon="110.05"><tag k="place" v="village"/><tag k="name" v="Neighbour Village"/></node>
 <way id="12" version="1"><nd ref="3"/><nd ref="4"/><nd ref="1"/></way>
 <way id="13" version="1"><nd ref="31"/><nd ref="32"/><nd ref="33"/><nd ref="34"/><nd ref="31"/></way>
 <relation id="300" version="1">
  <member type="way" ref="13" role="outer"/>
  <tag k="boundary" v="administrative"/><tag k="type" v="boundary"/><tag k="admin_level" v="4"/>
  <tag k="ISO3166-2" v="ZZ-02"/><tag k="name" v="Neighbour's Own"/>
 </relation>
</osm>
`
)

// neighbourFiles writes the own and the neighbour's file and points the
// neighbour "nb-land" at the latter for the test's duration.
func neighbourFiles(t *testing.T) (own, nb string) {
	t.Helper()
	dir := t.TempDir()
	own, nb = filepath.Join(dir, "own.osm"), filepath.Join(dir, "nb-land.osm")
	for p, s := range map[string]string{own: ownOSM, nb: neighbourOSM} {
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := neighbourInputFor
	neighbourInputFor = func(data, name string) (Input, error) {
		if name == "nb-land" {
			return Input{Path: nb, Cut: true}, nil
		}
		return old(data, name)
	}
	t.Cleanup(func() { neighbourInputFor = old })
	return own, nb
}

func TestNeighboursCompleteBoundaries(t *testing.T) {
	own, nb := neighbourFiles(t)
	imp := zzSettings()
	c := country{Code: "zz", Slug: "zz", Name: "Zed", Imp: imp}

	// The own file alone: r100 does not assemble (check 6).
	set, err := readRegions(own, nil, c, noClip, geo.SimplifyTolerance)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.provinces) != 0 || !slices.Equal(set.notAssembled, []int64{100}) {
		t.Fatalf("own file alone: provinces %v, not assembled %v", provinceIDs(set), set.notAssembled)
	}

	// With the neighbour's cut: r100 is whole, and the neighbour's own
	// province relation r300 is not read although it matches the level.
	set, err = readRegions(own, []string{nb}, c, noClip, geo.SimplifyTolerance)
	if err != nil {
		t.Fatal(err)
	}
	if got := provinceIDs(set); !slices.Equal(got, []int64{100}) || len(set.notAssembled) != 0 || set.relations != 1 {
		t.Fatalf("with the neighbour: provinces %v, not assembled %v, relations read %d", got, set.notAssembled, set.relations)
	}
}

func TestNeighbourInputs(t *testing.T) {
	imp := zzSettings()
	imp.Neighbours = []string{"nowhere"}
	c := country{Code: "zz", Slug: "zz", Name: "Zed", Imp: imp}
	data := t.TempDir()

	n := neighbourInputs(Options{Data: data}, c)
	want := filepath.Join(data, "scan", "nowhere-latest.osm.pbf.cut.osm.pbf")
	if len(n.read) != 0 || !slices.Equal(n.missing, []string{want}) {
		t.Fatalf("missing neighbour: %+v", n)
	}
	// A missing cut fails check 6 with the file and the way to make it.
	pl := &regionPlan{c: c, nb: n, set: &regionSet{provinces: []*province{{osmID: 1, slug: "one-zz", land: 1}}}}
	var six Check
	for _, ch := range pl.runChecks() {
		if ch.Num == 6 {
			six = ch
		}
	}
	if !six.Failed() || len(six.Details) != 1 || !strings.Contains(six.Details[0], want) || !strings.Contains(six.Details[0], "map/cut.sh nowhere") {
		t.Errorf("check 6: %+v", six)
	}

	// -pbf without -data: not looked for, and said so.
	if n := neighbourInputs(Options{PBF: "x.osm"}, c); n.skipped == "" || len(n.missing) != 0 || len(n.read) != 0 {
		t.Errorf("-pbf without -data: %+v", n)
	}
	// No neighbours: nothing at all.
	c.Imp.Neighbours = nil
	if n := neighbourInputs(Options{Data: data}, c); n.skipped != "" || n.missing != nil || n.read != nil {
		t.Errorf("no neighbours: %+v", n)
	}
}

// A run with a neighbour: its boundaries assemble from both cuts, its
// places come from the own cut only, and the dry run names the neighbour's
// file.
func TestNeighbourNeverFeedsPlaces(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	own, nb := neighbourFiles(t)
	_, waterPath := Fixture()
	zz := zzSettings()
	zz.Neighbours = []string{"nb-land"}
	var report strings.Builder
	o := Options{Country: "zz", Settings: &zz, PBF: own, Data: t.TempDir(), Water: waterPath, DryRun: true, Report: &report}
	sum, err := Import(ctx, d, o)
	if !errors.Is(err, ErrChecks) || !strings.Contains(err.Error(), "4 land plots") || sum.Provinces != 1 || sum.LandPlots == 0 {
		t.Fatalf("dry run without LandPlots: %+v %v", sum, err)
	}
	if !strings.Contains(report.String(), "Neighbour read for boundaries only: "+nb+" (cut)") {
		t.Errorf("the dry run does not name the neighbour's cut:\n%s", report.String())
	}
	zz.LandPlots = sum.LandPlots
	o.DryRun, o.Report = false, nil
	if sum, err = Import(ctx, d, o); err != nil {
		t.Fatalf("zz: %+v %v", sum, err)
	}
	if sum.Provinces != 1 || sum.Places != 1 {
		t.Errorf("summary %+v, want 1 province and 1 place", sum)
	}
	var names []string
	rows, err := d.Pool.Query(ctx, `select osm_id, name_en, country from places order by osm_id`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id int64
		var name, country string
		if err := rows.Scan(&id, &name, &country); err != nil {
			t.Fatal(err)
		}
		names = append(names, fmt.Sprintf("%d %s %s", id, name, country))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(names, []string{"201 Own Town zz"}) {
		t.Errorf("places %q, want only the own cut's town", names)
	}
	var n int
	if err := d.Pool.QueryRow(ctx, `select count(*) from regions where osm_id = 300`).Scan(&n); err != nil || n != 0 {
		t.Errorf("the neighbour's province stored %d times %v", n, err)
	}
}

func TestLegacyDuplicateSlug(t *testing.T) {
	// Two district relations whose stripped names collide.
	path := writeTestOSM(t, []testRel{
		{id: 1, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-01", "name", "Zed One"}, outer: []rect{{100, 10, 101, 11}}},
		{id: 22, tags: []string{"admin_level", "6", "name", "Tan District"}, outer: []rect{{100.5, 10.1, 100.7, 10.3}}},
		{id: 21, tags: []string{"admin_level", "6", "name", "Tan District"}, outer: []rect{{100.1, 10.1, 100.3, 10.3}}},
	}, nil)
	slugs := func(set *regionSet) map[int64]string {
		out := map[int64]string{}
		for _, d := range set.districts {
			out[d.osmID] = d.slug + " " + d.names.En
		}
		return out
	}
	want := func(set *regionSet, w map[int64]string) {
		t.Helper()
		got := slugs(set)
		if len(got) != len(w) {
			t.Errorf("districts %v, want %v", got, w)
		}
		for id, s := range w {
			if got[id] != s {
				t.Errorf("r%d: %q, want %q", id, got[id], s)
			}
		}
	}

	// Decision 7: both keep the affix, then both get their OSM id.
	imp := zzSettings()
	want(readTest(t, path, imp), map[int64]string{21: "zed-one-zz-tan-district-21 Tan District", 22: "zed-one-zz-tan-district-22 Tan District"})

	// The old rule: the first by OSM id keeps the plain slug, the second
	// gets its id, names stay stripped.
	imp.LegacyDuplicateSlug = true
	want(readTest(t, path, imp), map[int64]string{21: "zed-one-zz-tan Tan", 22: "zed-one-zz-tan-22 Tan"})

	// A stored slug is kept, and a new district gives way to it.
	c := country{Code: "zz", Slug: "zz", Name: "Zed", Imp: imp}
	set, err := readRegions(path, nil, c, noClip, geo.SimplifyTolerance)
	if err != nil {
		t.Fatal(err)
	}
	resolveSlugs(set, c, map[int64]*db.StoredRegion{22: {OSMID: 22, Country: "zz", Slug: "zed-one-zz-tan"}})
	want(set, map[int64]string{21: "zed-one-zz-tan-21 Tan", 22: "zed-one-zz-tan Tan"})
}
