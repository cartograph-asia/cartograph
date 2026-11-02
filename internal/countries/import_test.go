package countries

import (
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
	}
	if got := Imports["th"]; !reflect.DeepEqual(got, want) {
		t.Errorf("Thailand's settings\n got %+v\nwant %+v", got, want)
	}
}
