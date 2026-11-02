package countries_test

import (
	"testing"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// mbsTags builds (shared by the Malaysia, Brunei and Singapore tests) tags from key, value pairs.
func mbsTags(kv ...string) osm.Tags {
	var t osm.Tags
	for i := 0; i < len(kv); i += 2 {
		t = append(t, osm.Tag{Key: kv[i], Value: kv[i+1]})
	}
	return t
}

// mbsRel is a relation with the given id and tags.
func mbsRel(id int64, kv ...string) *osm.Relation {
	return &osm.Relation{ID: osm.RelationID(id), Tags: mbsTags(kv...)}
}

// mbsLevelCase is one relation's tags and what the level's rules make of them.
type mbsLevelCase struct {
	name            string
	tags            osm.Tags
	level           countries.Level
	en, local, kind string
	slug            string
}

func checkMBSLevelNames(t *testing.T, imp countries.Import, cases []mbsLevelCase) {
	t.Helper()
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, imp, tc.level)
		if n.En != tc.en || n.Local != tc.local || n.Kind != tc.kind {
			t.Errorf("%s: got %q %q %q, want %q %q %q", tc.name, n.En, n.Local, n.Kind, tc.en, tc.local, tc.kind)
		}
		if got := osmfile.Slug(n.En); got != tc.slug {
			t.Errorf("%s: slug %q, want %q", tc.name, got, tc.slug)
		}
	}
}

// Malaysia's settings on tag sets as the malaysia-singapore-brunei-latest.osm.pbf
// of 2026-11-01 holds them (docs/countries/malaysia.md).
func TestMalaysiaNames(t *testing.T) {
	my, ok := countries.ImportOf("my")
	if !ok || my.Hold != "" {
		t.Fatalf("Malaysia: settings %v, hold %q", ok, my.Hold)
	}
	if my.LocalLang != "ms" || !my.SlugSuffix || my.Province.Count != [2]int{16, 16} {
		t.Errorf("Malaysia: lang %q, suffix %v, count %v", my.LocalLang, my.SlugSuffix, my.Province.Count)
	}
	if len(my.Neighbours) != 1 || my.Neighbours[0] != "philippines" {
		t.Errorf("Malaysia: neighbours %v, want [philippines] (Sabah)", my.Neighbours)
	}
	checkMBSLevelNames(t, my, []mbsLevelCase{
		// States: names equal, local NULL; the word is "state".
		{"state", mbsTags("name", "Kedah", "name:en", "Kedah", "name:ms", "Kedah"),
			my.Province, "Kedah", "", "state", "kedah"},
		{"state without name:en", mbsTags("name", "Selangor", "name:ms", "Selangor", "official_name", "Selangor Darul Ehsan"),
			my.Province, "Selangor", "", "state", "selangor"},
		{"state with an English name", mbsTags("name", "Pulau Pinang", "name:en", "Penang", "name:ms", "Pulau Pinang"),
			my.Province, "Penang", "Pulau Pinang", "state", "penang"},
		{"Malacca", mbsTags("name", "Melaka", "name:en", "Malacca", "name:ms", "Melaka"),
			my.Province, "Malacca", "Melaka", "state", "malacca"},
		// The federal territories, by their whole name.
		{"Kuala Lumpur", mbsTags("name", "Kuala Lumpur", "name:en", "Kuala Lumpur", "name:ms", "Kuala Lumpur", "official_name", "Wilayah Persekutuan Kuala Lumpur"),
			my.Province, "Kuala Lumpur", "", "federal territory", "kuala-lumpur"},
		{"Labuan", mbsTags("name", "Labuan", "name:ms", "Labuan"),
			my.Province, "Labuan", "", "federal territory", "labuan"},
		{"Putrajaya", mbsTags("name", "Putrajaya", "name:en", "Putrajaya", "name:ms", "Putrajaya"),
			my.Province, "Putrajaya", "", "federal territory", "putrajaya"},
		// Districts.
		{"district, name only", mbsTags("name", "Kuala Muda", "name:zh", "瓜拉姆达县"),
			my.District, "Kuala Muda", "", "district", "kuala-muda"},
		{"district with Daerah", mbsTags("name", "Sandakan District", "name:ms", "Daerah Sandakan"),
			my.District, "Sandakan", "", "district", "sandakan"},
		{"English with District", mbsTags("name", "Seberang Perai Utara", "name:en", "North Seberang Perai District"),
			my.District, "North Seberang Perai", "Seberang Perai Utara", "district", "north-seberang-perai"},
		{"Kelantan's jajahan, Jawi in name", mbsTags("name", "باچوق / Bachok", "name:en", "Bachok", "name:ms", "Jajahan Bachok", "int_name", "Bachok"),
			my.District, "Bachok", "", "district", "bachok"},
		{"Penang's translated names", mbsTags("name", "Barat Daya", "name:en", "South-West"),
			my.District, "South-West", "Barat Daya", "district", "south-west"},
		{"and in English", mbsTags("name", "Larut, Matang dan Selama", "name:en", "Larut, Matang and Selama", "name:ms", "Larut, Matang dan Selama"),
			my.District, "Larut, Matang and Selama", "Larut, Matang dan Selama", "district", "larut-matang-and-selama"},
	})

	// kind_name: "state" for every state, "federal territory" for three.
	if k := osmfile.KindName("Johor", "Johor", my.Province, "province"); k != "state" {
		t.Errorf("state: kind_name %q", k)
	}
	if k := osmfile.KindName("Kuala Lumpur", "Kuala Lumpur", my.Province, "province"); k != "federal territory" {
		t.Errorf("federal territory: kind_name %q", k)
	}
	if k := osmfile.KindName("Kuala Muda", "", my.District, "district"); k != "" {
		t.Errorf("district: kind_name %q, want NULL", k)
	}

	// Provinces are the MY- relations at level 4; Brunei's and Thailand's in
	// the same extract are not.
	for name, c := range map[string]struct {
		r    *osm.Relation
		want bool
	}{
		"Sabah":   {mbsRel(3879783, "boundary", "administrative", "admin_level", "4", "ISO3166-2", "MY-12"), true},
		"Belait":  {mbsRel(3853884, "boundary", "administrative", "admin_level", "4", "ISO3166-2", "BN-BE"), false},
		"Satun":   {mbsRel(2, "boundary", "administrative", "admin_level", "4", "ISO3166-2", "TH-91"), false},
		"no ISO":  {mbsRel(3, "boundary", "administrative", "admin_level", "4"), false},
		"Sarawak": {mbsRel(15997748, "boundary", "administrative", "admin_level", "5"), false},
	} {
		if got := osmfile.MatchLevel(c.r, my.Province); got != c.want {
			t.Errorf("province level, %s: %v, want %v", name, got, c.want)
		}
	}

	// Places: a Thai name beyond the border is not English; Malay names are
	// Latin and NULL when they repeat the English one.
	for _, tc := range []struct {
		tags      osm.Tags
		en, local string
	}{
		{mbsTags("name", "Bandar Baru Bangi", "name:ms", "Bandar Baru Bangi"), "Bandar Baru Bangi", ""},
		{mbsTags("name", "Pagoh"), "Pagoh", ""},
		{mbsTags("name", "Melaka", "name:en", "Malacca"), "Malacca", "Melaka"},
		{mbsTags("name", "ปาดังเบซาร์", "name:en", "Padang Besar", "name:ms", "Padang Besar"), "Padang Besar", ""},
		{mbsTags("name", "บ้านปะลุกา"), "", ""},
	} {
		en, local := osmfile.PlaceNames(tc.tags, my)
		if en != tc.en || local != tc.local {
			t.Errorf("place %v: got %q %q, want %q %q", tc.tags, en, local, tc.en, tc.local)
		}
	}
}
