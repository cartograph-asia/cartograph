package countries_test

import (
	"testing"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// Brunei's settings on tag sets as the malaysia-singapore-brunei-latest.osm.pbf
// of 2026-11-01 holds them (docs/countries/brunei.md).
func TestBruneiNames(t *testing.T) {
	bn, ok := countries.ImportOf("bn")
	if !ok || bn.Hold != "" {
		t.Fatalf("Brunei: settings %v, hold %q", ok, bn.Hold)
	}
	if bn.LocalLang != "ms" || !bn.SlugSuffix || bn.Province.Count != [2]int{4, 4} {
		t.Errorf("Brunei: lang %q, suffix %v, count %v", bn.LocalLang, bn.SlugSuffix, bn.Province.Count)
	}
	// No district level (decision 6): the zero Level matches nothing.
	if len(bn.District.AdminLevels) != 0 || osmfile.MatchLevel(mbsRel(4197775, "boundary", "administrative", "admin_level", "6"), bn.District) {
		t.Errorf("Brunei: district level %v, want none", bn.District.AdminLevels)
	}
	checkMBSLevelNames(t, bn, []mbsLevelCase{
		{"district", mbsTags("name", "Belait", "name:en", "Belait District", "name:ms", "Daerah Belait"),
			bn.Province, "Belait", "", "district", "belait"},
		{"district with a hyphen", mbsTags("name", "Brunei-Muara", "name:en", "Brunei-Muara District", "name:ms", "Daerah Brunei-Muara"),
			bn.Province, "Brunei-Muara", "", "district", "brunei-muara"},
		{"name only", mbsTags("name", "Temburong"),
			bn.Province, "Temburong", "", "district", "temburong"},
	})
	// The word differs from "province", so kind_name is written: "District in Brunei".
	if k := osmfile.KindName("Tutong District", "Daerah Tutong", bn.Province, "province"); k != "district" {
		t.Errorf("kind_name %q, want district", k)
	}
	for name, c := range map[string]struct {
		id   int64
		iso  string
		want bool
	}{
		"Tutong": {3853886, "BN-TU", true},
		"Sabah":  {3879783, "MY-12", false},
	} {
		r := mbsRel(c.id, "boundary", "administrative", "admin_level", "4", "ISO3166-2", c.iso)
		if got := osmfile.MatchLevel(r, bn.Province); got != c.want {
			t.Errorf("province level, %s: %v, want %v", name, got, c.want)
		}
	}
	en, local := osmfile.PlaceNames(mbsTags("name", "Kampong Ayer"), bn)
	if en != "Kampong Ayer" || local != "" {
		t.Errorf("place: got %q %q", en, local)
	}
}
