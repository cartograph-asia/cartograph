package countries_test

import (
	"testing"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// Laos's settings on tag sets as the laos-latest.osm.pbf of 2026-11-01
// holds them (docs/countries/laos.md).
func TestLaosNames(t *testing.T) {
	la, ok := countries.ImportOf("la")
	if !ok || la.Hold != "" {
		t.Fatalf("Laos: settings %v, hold %q", ok, la.Hold)
	}
	tags := func(kv ...string) osm.Tags {
		var t osm.Tags
		for i := 0; i < len(kv); i += 2 {
			t = append(t, osm.Tag{Key: kv[i], Value: kv[i+1]})
		}
		return t
	}
	cases := []struct {
		name               string
		tags               osm.Tags
		level              countries.Level
		en, local, kind    string
		slugPart, wantSlug string
	}{
		{"province with suffix", tags("name", "ວຽງຈັນ", "name:en", "Vientiane Province", "name:lo", "ວຽງຈັນ", "name:lo-Latn", "Viangchan"),
			la.Province, "Vientiane", "ວຽງຈັນ", "province", "vientiane", "vientiane-laos"},
		{"province without suffix", tags("name", "ອັດຕະປື", "name:en", "Attapeu", "name:lo", "ອັດຕະປື"),
			la.Province, "Attapeu", "ອັດຕະປື", "province", "attapeu", "attapeu-laos"},
		{"the capital keeps its word", tags("name", "ນະຄອນຫຼວງວຽງຈັນ", "name:en", "Vientiane Prefecture", "name:lo", "ນະຄອນຫຼວງວຽງຈັນ"),
			la.Province, "Vientiane Prefecture", "ນະຄອນຫຼວງວຽງຈັນ", "prefecture", "vientiane-prefecture", "vientiane-prefecture-laos"},
		{"district with muang and accent", tags("name", "ເມືອງປາກເຊ", "name:en", "Pak Sé District", "name:lo", "ເມືອງປາກເຊ"),
			la.District, "Pak Sé", "ປາກເຊ", "district", "pak-se", ""},
		{"district without name:lo", tags("name", "ເມືອງຫົງສາ", "name:en", "Hongsa District"),
			la.District, "Hongsa", "ຫົງສາ", "district", "hongsa", ""},
		{"district whose name is Latin", tags("name", "Sisattanāk", "name:en", "Sisattanak District", "name:lo", "ສີສັດຕະນາກ"),
			la.District, "Sisattanak", "ສີສັດຕະນາກ", "district", "sisattanak", ""},
	}
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, la, tc.level)
		if n.En != tc.en || n.Local != tc.local || n.Kind != tc.kind {
			t.Errorf("%s: got %q %q %q, want %q %q %q", tc.name, n.En, n.Local, n.Kind, tc.en, tc.local, tc.kind)
		}
		if got := osmfile.Slug(n.En); got != tc.slugPart {
			t.Errorf("%s: slug %q, want %q", tc.name, got, tc.slugPart)
		}
		if tc.wantSlug != "" {
			if got := osmfile.Slug(n.En) + "-laos"; got != tc.wantSlug || !la.SlugSuffix {
				t.Errorf("%s: province slug %q (suffix %v), want %q", tc.name, got, la.SlugSuffix, tc.wantSlug)
			}
		}
	}

	// Places beyond the border: a name in a neighbour's script is not English,
	// a Latin one is; the Lao name comes from name:lo or a Lao name only.
	places := []struct {
		tags      osm.Tags
		en, local string
	}{
		{tags("name", "ບ້ານນາ", "name:en", "Ban Na"), "Ban Na", "ບ້ານນາ"},
		{tags("name", "ບ້ານນາ"), "", "ບ້ານນາ"},
		{tags("name", "บ้านสองคอน", "name:en", "Ban Song Khon"), "Ban Song Khon", ""},
		{tags("name", "บ้านสองคอน"), "", ""},
		{tags("name", "口岸"), "", ""},
		{tags("name", "Tiên Sơn"), "Tiên Sơn", ""},
	}
	for _, tc := range places {
		en, local := osmfile.PlaceNames(tc.tags, la)
		if en != tc.en || local != tc.local {
			t.Errorf("place %v: got %q %q, want %q %q", tc.tags, en, local, tc.en, tc.local)
		}
	}
}
