package countries

import (
	"fmt"
	"reflect"
	"testing"
	"unicode"
)

func TestEveryCountryHasImportSettings(t *testing.T) {
	for _, c := range All {
		imp, ok := ImportOf(c.Code)
		if !ok {
			t.Errorf("%s: no entry in Imports", c.Code)
			continue
		}
		if imp.Extract == "" || imp.LandPlots <= 0 {
			t.Errorf("%s: extract %q, land plots %d", c.Code, imp.Extract, imp.LandPlots)
		}
		if imp.Hold == "" {
			// An open country needs the rules an import reads.
			if len(imp.Province.AdminLevels) == 0 || len(imp.English) == 0 || len(imp.Local) == 0 {
				t.Errorf("%s: not on hold, but province level or name tags are missing", c.Code)
			}
		}
		for _, l := range []Level{imp.Province, imp.District} {
			if l.Count[0] > l.Count[1] {
				t.Errorf("%s: count %v", c.Code, l.Count)
			}
		}
		for _, s := range append(append([]string{}, imp.EnglishNot...), imp.LocalScripts...) {
			if unicode.Scripts[s] == nil {
				t.Errorf("%s: %q is not a Unicode script name", c.Code, s)
			}
		}
	}
	if len(Imports) != len(All) {
		t.Errorf("Imports has %d entries for %d countries", len(Imports), len(All))
	}
	if _, ok := ImportOf("TH"); !ok {
		t.Error("ImportOf(TH) not found")
	}
	if _, ok := ImportOf("xx"); ok {
		t.Error("ImportOf(xx) found")
	}
}

func TestExtracts(t *testing.T) {
	for code, want := range map[string]string{
		"th": "thailand", "tl": "east-timor", "my": "malaysia-singapore-brunei",
		"bn": "malaysia-singapore-brunei", "sg": "malaysia-singapore-brunei", "kp": "north-korea",
	} {
		if got := Imports[code].Extract; got != want {
			t.Errorf("%s: extract %q, want %q", code, got, want)
		}
	}
}

// Neighbours name the extracts whose cuts complete a country's boundaries
// (map/data/scan/<name>-latest.osm.pbf.cut.osm.pbf), so each must be an
// extract the settings already know. An extract read for boundaries only
// and never imported (China's "bhutan", docs/countries/
// incomplete-relations.md) is added to boundaryOnly by the PR that needs it.
func TestNeighboursAreKnownExtracts(t *testing.T) {
	for _, e := range neighbourErrors(Imports, nil) {
		t.Error(e)
	}
	// The rule itself, on settings that break it.
	bad := map[string]Import{
		"aa": {Extract: "aa-land", Neighbours: []string{"bb-land", "nowhere", "aa-land", "bb-land", "bhutan"}},
		"bb": {Extract: "bb-land"},
	}
	got := neighbourErrors(bad, map[string]bool{"bhutan": true})
	if len(got) != 3 {
		t.Errorf("errors on bad settings: %q, want 3 (unknown, own extract, twice)", got)
	}
}

func neighbourErrors(imps map[string]Import, boundaryOnly map[string]bool) []string {
	known := map[string]bool{}
	for _, imp := range imps {
		known[imp.Extract] = true
	}
	var out []string
	for code, imp := range imps {
		seen := map[string]bool{}
		for _, n := range imp.Neighbours {
			switch {
			case !known[n] && !boundaryOnly[n]:
				out = append(out, code+": neighbour "+n+" is not the Extract of any country's settings")
			case n == imp.Extract:
				out = append(out, code+": neighbour "+n+" is the country's own extract")
			case seen[n]:
				out = append(out, code+": neighbour "+n+" listed twice")
			}
			seen[n] = true
		}
	}
	return out
}

// Only Thailand keeps the old duplicate rule for districts.
func TestLegacyDuplicateSlugThailandOnly(t *testing.T) {
	for code, imp := range Imports {
		if imp.LegacyDuplicateSlug != (code == "th") {
			t.Errorf("%s: LegacyDuplicateSlug %v", code, imp.LegacyDuplicateSlug)
		}
	}
}

// Only Thailand keeps the old place rule; no country sets a negative reach.
func TestLegacyPlacesThailandOnly(t *testing.T) {
	for code, imp := range Imports {
		if imp.LegacyPlaces != (code == "th") {
			t.Errorf("%s: LegacyPlaces %v", code, imp.LegacyPlaces)
		}
		if imp.PlaceReachKm < 0 {
			t.Errorf("%s: PlaceReachKm %v", code, imp.PlaceReachKm)
		}
	}
}

func TestChinaAndMongoliaHold(t *testing.T) {
	for _, code := range []string{"cn", "mn"} {
		if Imports[code].Hold == "" {
			t.Errorf("%s does not open in the alpha, but has no Hold", code)
		}
	}
}

// Thailand's settings are the ones docs/COUNTRY_IMPORT_STAGE2_BRIEF.md
// gives; internal/geo/osmfile's tests check them against the old functions.
func TestThailandSettings(t *testing.T) {
	want := Import{
		Extract: "thailand",
		Province: Level{AdminLevels: []int{4}, ISOPrefix: "TH", Count: [2]int{77, 77}, Kind: "province",
			Strip: Affixes{EnSuffix: []string{" Province"}, LocalPrefix: []string{"จังหวัด"}}},
		District: Level{AdminLevels: []int{6}, Kind: "district",
			Strip: Affixes{EnSuffix: []string{" District"}, EnPrefix: []string{"Khet ", "Amphoe "}, LocalPrefix: []string{"เขต", "อำเภอ"}}},
		English:      []string{"name:en", "int_name", "name:th-Latn", "name"},
		EnglishNot:   []string{"Thai"},
		Local:        []string{"name:th", "name"},
		LocalScripts: []string{"Thai"},
		LandPlots:    598411,

		LegacyDuplicateSlug: true,
		LegacyPlaces:        true,
	}
	if got := Imports["th"]; !reflect.DeepEqual(got, want) {
		t.Errorf("Thailand's settings\n got %+v\nwant %+v", got, want)
	}
}

// A relation is either included or excluded at a level, never both.
func TestExcludeNotIncluded(t *testing.T) {
	for _, e := range excludeErrors(Imports) {
		t.Error(e)
	}
	// The rule itself, on settings that break it.
	bad := map[string]Import{"aa": {
		Province: Level{AdminLevels: []int{4}, Include: []int64{1, 2}, Exclude: []int64{2}},
		District: Level{AdminLevels: []int{6}, Include: []int64{3}, Exclude: []int64{3, 4}},
	}}
	if got := excludeErrors(bad); len(got) != 2 {
		t.Errorf("errors on bad settings: %q, want 2", got)
	}
}

func excludeErrors(imps map[string]Import) []string {
	var out []string
	for code, imp := range imps {
		for name, l := range map[string]Level{"province": imp.Province, "district": imp.District} {
			for _, id := range l.Exclude {
				for _, in := range l.Include {
					if id == in {
						out = append(out, fmt.Sprintf("%s: %s relation r%d is in both Include and Exclude", code, name, id))
					}
				}
			}
		}
	}
	return out
}
