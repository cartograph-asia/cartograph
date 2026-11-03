package importer

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo"
)

// readTest reads path with imp as country zz, no clip, no stored rows.
func readTest(t *testing.T, path string, imp countries.Import) *regionSet {
	t.Helper()
	set, err := readRegions(path, nil, country{Code: "zz", Slug: "zz", Name: "Zed", Imp: imp}, noClip, geo.SimplifyTolerance)
	if err != nil {
		t.Fatal(err)
	}
	resolveSlugs(set, country{Code: "zz", Slug: "zz", Imp: imp}, nil)
	return set
}

// noClip is the empty bound: no clipping.
var noClip, _ = geo.ParseBound("")

func provinceIDs(set *regionSet) []int64 {
	var out []int64
	for _, p := range set.provinces {
		out = append(out, p.osmID)
	}
	return out
}

func districtLevels(set *regionSet) map[int64]int {
	out := map[int64]int{}
	for _, d := range set.districts {
		out[d.osmID] = d.level
	}
	return out
}

func TestLevelsTwoDistrictLevels(t *testing.T) {
	// Taiwan: districts are levels 7 and 8 together.
	imp := zzSettings()
	imp.District = countries.Level{AdminLevels: []int{7, 8}, Kind: "district"}
	path := writeTestOSM(t, []testRel{
		{id: 1, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-01", "name", "One"}, outer: []rect{{120, 23, 121, 24}}},
		{id: 7, tags: []string{"admin_level", "7", "name", "Seven"}, outer: []rect{{120.1, 23.1, 120.4, 23.4}}},
		{id: 8, tags: []string{"admin_level", "8", "name", "Eight"}, outer: []rect{{120.5, 23.5, 120.8, 23.8}}},
		{id: 6, tags: []string{"admin_level", "6", "name", "Six"}, outer: []rect{{120.1, 23.5, 120.4, 23.8}}},
	}, nil)
	set := readTest(t, path, imp)
	if got := districtLevels(set); len(got) != 2 || got[7] != 7 || got[8] != 8 {
		t.Fatalf("districts %v", got)
	}
	for _, d := range set.districts {
		if !strings.HasPrefix(d.slug, "one-zz-") {
			t.Errorf("district slug %q", d.slug)
		}
	}
}

func TestLevelsParent(t *testing.T) {
	// Singapore: provinces without ISO codes, recognised by lying inside the
	// country relation.
	imp := zzSettings()
	imp.Province = countries.Level{AdminLevels: []int{5}, Parent: 500, Kind: "region", Count: [2]int{1, 1}}
	path := writeTestOSM(t, []testRel{
		{id: 500, tags: []string{"admin_level", "2", "name", "Zedland"}, outer: []rect{{103.6, 1.2, 104.1, 1.5}}},
		{id: 51, tags: []string{"admin_level", "5", "name", "Central Region"}, outer: []rect{{103.8, 1.25, 103.9, 1.35}}},
		{id: 52, tags: []string{"admin_level", "5", "name", "Johor Bit"}, outer: []rect{{103.5, 1.5, 103.7, 1.7}}},
	}, nil)
	set := readTest(t, path, imp)
	if got := provinceIDs(set); !slices.Equal(got, []int64{51}) || set.outsideParent != 1 {
		t.Fatalf("provinces %v, outside the parent %d", got, set.outsideParent)
	}
	if p := set.provinces[0]; p.level != 5 || p.slug != "central-region-zz" || p.kindName != "region" {
		t.Errorf("province %+v", p)
	}
}

func TestLevelsInclude(t *testing.T) {
	// Philippines: a relation without the ISO code counts when listed.
	imp := zzSettings()
	imp.Province.Include = []int64{80}
	path := writeTestOSM(t, []testRel{
		{id: 1, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-01", "name", "One"}, outer: []rect{{120, 5, 121, 6}}},
		{id: 80, tags: []string{"admin_level", "3", "name", "Special Area"}, outer: []rect{{122, 5, 123, 6}}},
		{id: 81, tags: []string{"admin_level", "4", "name", "No Code"}, outer: []rect{{124, 5, 125, 6}}},
	}, nil)
	set := readTest(t, path, imp)
	if got := provinceIDs(set); !slices.Equal(got, []int64{1, 80}) {
		t.Fatalf("provinces %v", got)
	}
	if set.provinces[1].level != 3 {
		t.Errorf("Include relation level %d", set.provinces[1].level)
	}
}

func TestLevelsNoDistrictLevel(t *testing.T) {
	// Brunei: no district level.
	imp := zzSettings()
	imp.District = countries.Level{}
	path := writeTestOSM(t, []testRel{
		{id: 1, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-01", "name", "One"}, outer: []rect{{114, 4, 115, 5}}},
		{id: 6, tags: []string{"admin_level", "6", "name", "Mukim"}, outer: []rect{{114.1, 4.1, 114.4, 4.4}}},
	}, nil)
	set := readTest(t, path, imp)
	if len(set.provinces) != 1 || len(set.districts) != 0 || set.dropped != (dropped{}) || set.relations != 1 {
		t.Fatalf("provinces %v districts %v dropped %+v relations read %d", provinceIDs(set), districtLevels(set), set.dropped, set.relations)
	}
}

func TestLevelsNamesKindsAndDuplicates(t *testing.T) {
	// Names by the settings: affix stripped, kind word by affix, a local
	// name equal to the English one (Latin script) is NULL; districts that
	// collide after stripping keep their affixes; a district without an
	// English name or outside every province is dropped.
	imp := zzSettings()
	imp.District.Strip = countries.Affixes{EnPrefix: []string{"Ward ", "Commune "}}
	path := writeTestOSM(t, []testRel{
		{id: 1, tags: []string{"admin_level", "4", "ISO3166-2", "ZZ-01", "name", "Zed One", "name:en", "Zed One Prefecture"}, outer: []rect{{100, 10, 101, 11}}},
		{id: 11, tags: []string{"admin_level", "6", "name", "Ward Tan"}, outer: []rect{{100.1, 10.1, 100.3, 10.3}}},
		{id: 12, tags: []string{"admin_level", "6", "name", "Commune Tan"}, outer: []rect{{100.5, 10.1, 100.7, 10.3}}},
		{id: 13, tags: []string{"admin_level", "6", "name:zz", "Zz only"}, outer: []rect{{100.1, 10.5, 100.3, 10.7}}},
		{id: 14, tags: []string{"admin_level", "6", "name", "Elsewhere"}, outer: []rect{{102.1, 10.5, 102.3, 10.7}}},
	}, nil)
	set := readTest(t, path, imp)
	p := set.provinces[0]
	if p.names.En != "Zed One" || p.names.Local != "" || p.kindName != "prefecture" || p.slug != "zed-one-zz" {
		t.Fatalf("province %+v %q", p.names, p.slug)
	}
	slugs := map[int64]string{}
	for _, d := range set.districts {
		slugs[d.osmID] = d.slug + " " + d.names.En
	}
	if slugs[11] != "zed-one-zz-ward-tan Ward Tan" || slugs[12] != "zed-one-zz-commune-tan Commune Tan" || len(slugs) != 2 {
		t.Errorf("districts %v", slugs)
	}
	// "Zz only" has only name:zz, which English does not read.
	if set.dropped.NoEnglish != 1 || set.dropped.Outside != 1 {
		t.Errorf("dropped %+v", set.dropped)
	}
}

func TestCellStringsRoundTrip(t *testing.T) {
	for _, c := range []string{"8865a8a5b1fffff", "88283082a3fffff"} {
		id, err := strconv.ParseUint(c, 16, 64)
		if err != nil || (geo.LandCell{ID: id}).Cell() != c {
			t.Errorf("%s: %v %v", c, (geo.LandCell{ID: id}).Cell(), err)
		}
	}
}

func TestInputFor(t *testing.T) {
	dir := t.TempDir()
	full := filepath.Join(dir, "zed-latest.osm.pbf")
	cut := filepath.Join(dir, "scan", "zed-latest.osm.pbf.cut.osm.pbf")
	if _, err := InputFor(dir, "zed"); err == nil {
		t.Fatal("no file: no error")
	}
	write := func(p string, at time.Time) {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	t0 := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	write(full, t0)
	if in, err := InputFor(dir, "zed"); err != nil || in.Path != full || in.Cut || in.Note == "" {
		t.Errorf("extract only: %+v %v", in, err)
	}
	write(cut, t0.Add(time.Hour))
	if in, err := InputFor(dir, "zed"); err != nil || in.Path != cut || !in.Cut || in.Note != "" {
		t.Errorf("newer cut: %+v %v", in, err)
	}
	write(full, t0.Add(2*time.Hour))
	if in, err := InputFor(dir, "zed"); err != nil || in.Path != full || !strings.Contains(in.Note, "older") {
		t.Errorf("older cut: %+v %v", in, err)
	}
	os.Remove(full)
	if in, err := InputFor(dir, "zed"); err != nil || in.Path != cut || in.Note == "" {
		t.Errorf("cut only: %+v %v", in, err)
	}
}

func TestReadOnlyURL(t *testing.T) {
	got, err := readOnly("postgres://u:p@h:5432/db?sslmode=disable")
	if err != nil || !strings.Contains(got, "default_transaction_read_only=on") || !strings.Contains(got, "sslmode=disable") {
		t.Errorf("url: %q %v", got, err)
	}
	if got, _ := readOnly("host=h dbname=db"); got != "host=h dbname=db default_transaction_read_only=on" {
		t.Errorf("dsn: %q", got)
	}
}
