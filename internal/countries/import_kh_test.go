package countries_test

import (
	"testing"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// Cambodia's settings on tag sets as the cambodia-latest.osm.pbf of
// 2026-11-01 holds them (docs/countries/cambodia.md).
func TestCambodiaNames(t *testing.T) {
	kh, ok := countries.ImportOf("kh")
	if !ok || kh.Hold != "" {
		t.Fatalf("Cambodia: settings %v, hold %q", ok, kh.Hold)
	}
	tags := func(kv ...string) osm.Tags {
		var t osm.Tags
		for i := 0; i < len(kv); i += 2 {
			t = append(t, osm.Tag{Key: kv[i], Value: kv[i+1]})
		}
		return t
	}
	cases := []struct {
		name            string
		tags            osm.Tags
		level           countries.Level
		en, local, kind string
		slug            string
	}{
		{"province", tags("name", "ខេត្តកំពង់ចាម", "name:en", "Kampong Cham", "name:km-Latn", "Khêt Kâmpóng Cham"),
			kh.Province, "Kampong Cham", "កំពង់ចាម", "province", "kampong-cham-cambodia"},
		{"province with name:km", tags("name", "ខេត្តកំពង់ស្ពឺ", "name:en", "Kampong Speu", "name:km", "ខេត្តកំពុងស្ពឺ"),
			kh.Province, "Kampong Speu", "កំពុងស្ពឺ", "province", "kampong-speu-cambodia"},
		{"province whose English name carries khaet", tags("name", "ខេត្តព្រះសីហនុ", "name:en", "Khaet Preah Sihanouk", "name:km", "ខេត្តព្រះសីហនុ"),
			kh.Province, "Preah Sihanouk", "ព្រះសីហនុ", "province", "preah-sihanouk-cambodia"},
		{"the capital", tags("name", "រាជធានីភ្នំពេញ", "name:en", "Phnom Penh"),
			kh.Province, "Phnom Penh", "ភ្នំពេញ", "capital", "phnom-penh-cambodia"},
		{"district without affix", tags("name", "ព្រៃនប់", "name:en", "Prey Nob", "name:km", "ស្រុកព្រៃនប់"),
			kh.District, "Prey Nob", "ព្រៃនប់", "district", "prey-nob"},
		{"khan of Phnom Penh", tags("name", "ខណ្ឌចំការមន", "name:en", "Khan Chamkar Mon"),
			kh.District, "Chamkar Mon", "ចំការមន", "district", "chamkar-mon"},
		{"khan without ខណ្ឌ", tags("name", "ប្រាំពីរមករា", "name:en", "Khan Prampir Makara"),
			kh.District, "Prampir Makara", "ប្រាំពីរមករា", "district", "prampir-makara"},
		{"krong", tags("name", "ក្រុងប៉ោយប៉ែត", "name:en", "Paoy Paet", "name:km", "ក្រុងប៉ោយប៉ែត"),
			kh.District, "Paoy Paet", "ប៉ោយប៉ែត", "district", "paoy-paet"},
		{"commune tagged as a district stays as OSM has it", tags("name", "ឃុំទឹកថ្លា", "name:en", "Teuk Thla Commune", "name:km", "ឃុំទឹកថ្លា"),
			kh.District, "Teuk Thla Commune", "ឃុំទឹកថ្លា", "district", "teuk-thla-commune"},
	}
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, kh, tc.level)
		if n.En != tc.en || n.Local != tc.local || n.Kind != tc.kind {
			t.Errorf("%s: got %q %q %q, want %q %q %q", tc.name, n.En, n.Local, n.Kind, tc.en, tc.local, tc.kind)
		}
		slug := osmfile.Slug(n.En)
		if tc.level.Kind == "province" && kh.SlugSuffix {
			slug += "-cambodia"
		}
		if slug != tc.slug {
			t.Errorf("%s: slug %q, want %q", tc.name, slug, tc.slug)
		}
	}

	// Places beyond the border: a name in a neighbour's script is not English,
	// a Latin one is; the Khmer name comes from name:km or a Khmer name only.
	places := []struct {
		tags      osm.Tags
		en, local string
	}{
		{tags("name", "ភូមិអូរ", "name:en", "Phum Ou"), "Phum Ou", "ភូមិអូរ"},
		{tags("name", "ភូមិអូរ"), "", "ភូមិអូរ"},
		{tags("name", "ភូមិអូរ", "name:km-Latn", "Phum Ou"), "Phum Ou", "ភូមិអូរ"},
		{tags("name", "บ้านโคกสูง", "name:en", "Ban Khok Sung"), "Ban Khok Sung", ""},
		{tags("name", "บ้านโคกสูง"), "", ""},
		{tags("name", "ບ້ານນາ"), "", ""},
		{tags("name", "口岸"), "", ""},
		{tags("name", "Ấp Tân Thành"), "Ấp Tân Thành", ""},
	}
	for _, tc := range places {
		en, local := osmfile.PlaceNames(tc.tags, kh)
		if en != tc.en || local != tc.local {
			t.Errorf("place %v: got %q %q, want %q %q", tc.tags, en, local, tc.en, tc.local)
		}
	}
}

// Cambodia is opened with the figures of its scan.
func TestCambodiaSettings(t *testing.T) {
	kh := countries.Imports["kh"]
	if kh.Extract != "cambodia" || kh.LocalLang != "km" || !kh.SlugSuffix || kh.LandPlots != 213_169 {
		t.Errorf("settings %+v", kh)
	}
	if kh.Province.Count != [2]int{25, 25} || kh.Province.ISOPrefix != "KH" || kh.District.AdminLevels[0] != 6 {
		t.Errorf("levels %+v %+v", kh.Province, kh.District)
	}
}
