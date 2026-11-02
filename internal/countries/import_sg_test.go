package countries_test

import (
	"testing"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// Singapore's settings on tag sets as the malaysia-singapore-brunei-latest.osm.pbf
// of 2026-11-01 holds them (docs/countries/singapore.md).
func TestSingaporeNames(t *testing.T) {
	sg, ok := countries.ImportOf("sg")
	if !ok || sg.Hold != "" {
		t.Fatalf("Singapore: settings %v, hold %q", ok, sg.Hold)
	}
	if sg.LocalLang != "en" || !sg.SlugSuffix || sg.Province.Count != [2]int{5, 5} {
		t.Errorf("Singapore: lang %q, suffix %v, count %v", sg.LocalLang, sg.SlugSuffix, sg.Province.Count)
	}
	// Provinces have no ISO code: they are the level-5 relations inside the
	// country relation (Parent, checked by the importer on the geometry).
	if sg.Province.ISOPrefix != "" || sg.Province.Parent != 536780 {
		t.Errorf("Singapore: ISO prefix %q, parent %d", sg.Province.ISOPrefix, sg.Province.Parent)
	}
	region := mbsRel(19910183, "boundary", "administrative", "admin_level", "5", "name", "Central Region", "name:en", "Central Region")
	if !osmfile.MatchLevel(region, sg.Province) {
		t.Error("a level-5 region does not match the province level")
	}
	if osmfile.MatchLevel(mbsRel(4447619, "boundary", "administrative", "admin_level", "6"), sg.Province) {
		t.Error("a level-6 relation matches the province level")
	}
	checkMBSLevelNames(t, sg, []mbsLevelCase{
		// The word stays in the name; English is local, so local is NULL.
		{"region", mbsTags("name", "Central Region", "name:en", "Central Region"),
			sg.Province, "Central Region", "", "region", "central-region"},
		{"region with a hyphen", mbsTags("name", "North-east Region", "name:en", "North-east Region"),
			sg.Province, "North-east Region", "", "region", "north-east-region"},
		// Planning areas: name only, the Chinese and Tamil names are not read.
		{"planning area", mbsTags("name", "Ang Mo Kio", "name:zh", "宏茂桥", "name:ta", "அங் மோ கியோ"),
			sg.District, "Ang Mo Kio", "", "planning area", "ang-mo-kio"},
		{"planning area without name:zh", mbsTags("name", "Bukit Merah"),
			sg.District, "Bukit Merah", "", "planning area", "bukit-merah"},
	})
	if k := osmfile.KindName("Central Region", "", sg.Province, "province"); k != "region" {
		t.Errorf("region: kind_name %q", k)
	}
	if k := osmfile.KindName("Bedok", "", sg.District, "district"); k != "planning area" {
		t.Errorf("planning area: kind_name %q", k)
	}
	// Places: never a Chinese or Tamil name as English, never a local name.
	for _, tc := range []struct {
		name      string
		kv        []string
		en, local string
	}{
		{"neighbourhood", []string{"name", "Tiong Bahru"}, "Tiong Bahru", ""},
		{"name:en differs", []string{"name", "Pulau Ubin", "name:en", "Ubin Island"}, "Ubin Island", ""},
		{"Chinese name", []string{"name", "牛车水", "name:en", "Chinatown"}, "Chinatown", ""},
		{"Chinese only", []string{"name", "牛车水"}, "", ""},
		{"Tamil only", []string{"name", "சிங்கப்பூர்"}, "", ""},
	} {
		en, local := osmfile.PlaceNames(mbsTags(tc.kv...), sg)
		if en != tc.en || local != tc.local {
			t.Errorf("place %s: got %q %q, want %q %q", tc.name, en, local, tc.en, tc.local)
		}
	}
}
