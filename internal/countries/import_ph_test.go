package countries_test

import (
	"testing"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// The Philippines' settings on tag sets as the philippines-latest.osm.pbf of
// 2026-11-01 holds them (docs/countries/philippines.md).
func TestPhilippinesNames(t *testing.T) {
	ph, ok := countries.ImportOf("ph")
	if !ok || ph.Hold != "" {
		t.Fatalf("Philippines: settings %v, hold %q", ok, ph.Hold)
	}
	if ph.LocalLang != "fil" || !ph.SlugSuffix || ph.Province.Count != [2]int{103, 103} || len(ph.District.AdminLevels) != 0 {
		t.Errorf("Philippines: lang %q, suffix %v, count %v, district levels %v",
			ph.LocalLang, ph.SlugSuffix, ph.Province.Count, ph.District.AdminLevels)
	}
	tags := func(kv ...string) osm.Tags {
		var t osm.Tags
		for i := 0; i < len(kv); i += 2 {
			t = append(t, osm.Tag{Key: kv[i], Value: kv[i+1]})
		}
		return t
	}
	rel := func(id int64, kv ...string) *osm.Relation {
		return &osm.Relation{ID: osm.RelationID(id), Tags: tags(kv...)}
	}

	// Which relations are provinces.
	levels := []struct {
		name string
		r    *osm.Relation
		want bool
	}{
		{"province with ISO", rel(1506936, "boundary", "administrative", "admin_level", "4", "ISO3166-2", "PH-CEB", "name", "Cebu"), true},
		{"Special Geographic Area", rel(21710245, "boundary", "administrative", "admin_level", "4", "border_type", "special_geographic_area", "name", "Special Geographic Area"), true},
		// Metro Manila (level 3) and the 19 cities that lie outside every
		// level-4 relation (level 6) come in through Include.
		{"Metro Manila at level 3", rel(147488, "boundary", "administrative", "admin_level", "3", "ISO3166-2", "PH-00", "name", "Metro Manila"), true},
		{"Cebu City at level 6", rel(12455830, "boundary", "administrative", "admin_level", "6", "border_type", "highly_urbanized_city", "name", "Cebu City", "name:en", "Cebu City"), true},
		{"Davao City at level 6", rel(3936841, "boundary", "administrative", "admin_level", "6", "border_type", "highly_urbanized_city", "name", "Davao City"), true},
		// Not included: a region at level 3, a city of Metro Manila, a city
		// inside its province's relation, and a level-6 municipality.
		{"Calabarzon at level 3", rel(2002020, "boundary", "administrative", "admin_level", "3", "ISO3166-2", "PH-40", "name", "Calabarzon"), false},
		{"Pasig inside Metro Manila", rel(131703, "boundary", "administrative", "admin_level", "6", "border_type", "highly_urbanized_city", "name", "Pasig"), false},
		{"Ormoc inside Leyte", rel(5426241, "boundary", "administrative", "admin_level", "6", "border_type", "independent_component_city", "name", "Ormoc"), false},
		{"municipality at level 6", rel(1, "boundary", "administrative", "admin_level", "6", "name", "Bacoor"), false},
		{"Sabah in the extract", rel(3879783, "boundary", "administrative", "admin_level", "4", "ISO3166-2", "MY-12", "name", "Sabah"), false},
		{"Kaohsiung in the extract", rel(2127079, "boundary", "administrative", "admin_level", "4", "ISO3166-2", "TW-KHH", "name", "高雄市"), false},
	}
	for _, tc := range levels {
		if got := osmfile.MatchLevel(tc.r, ph.Province); got != tc.want {
			t.Errorf("%s: province %v, want %v", tc.name, got, tc.want)
		}
		if osmfile.MatchLevel(tc.r, ph.District) {
			t.Errorf("%s: matches the district level, which the Philippines do not have", tc.name)
		}
	}

	// 82 provinces + the Special Geographic Area + Metro Manila + 19 cities.
	if n := len(ph.Province.Include); n != 21 {
		t.Errorf("Include holds %d relations, want 21 (SGA, Metro Manila, 19 cities)", n)
	}
	seenID := map[int64]bool{}
	for _, id := range ph.Province.Include {
		if seenID[id] {
			t.Errorf("r%d is in Include twice", id)
		}
		seenID[id] = true
	}

	cases := []struct {
		name            string
		tags            osm.Tags
		en, local, kind string
		slug            string
	}{
		// Most provinces: name, name:en and name:tl are the same; local is NULL.
		{"names equal", tags("name", "Cebu", "name:en", "Cebu", "name:tl", "Cebu", "ISO3166-2", "PH-CEB"),
			"Cebu", "", "province", "cebu-philippines"},
		// name:tl is the Filipino name where it differs.
		{"Filipino name in name:tl", tags("name", "Cavite", "name:en", "Cavite", "name:tl", "Kabite", "ISO3166-2", "PH-CAV"),
			"Cavite", "Kabite", "province", "cavite-philippines"},
		{"Filipino direction word", tags("name", "Northern Samar", "name:en", "Northern Samar", "name:tl", "Hilagang Samar"),
			"Northern Samar", "Hilagang Samar", "province", "northern-samar-philippines"},
		// "Province" is part of this province's name and is not stripped.
		{"Mountain Province", tags("name", "Mountain Province", "name:en", "Mountain Province", "name:tl", "Lalawigang Bulubundukin"),
			"Mountain Province", "Lalawigang Bulubundukin", "province", "mountain-province-philippines"},
		// The one name:fil, without name:en.
		{"name:fil without name:en", tags("name", "South Cotabato", "name:fil", "Timog Kutabato", "ISO3166-2", "PH-SCO"),
			"South Cotabato", "Timog Kutabato", "province", "south-cotabato-philippines"},
		{"name only", tags("name", "Sarangani", "ISO3166-2", "PH-SAR"),
			"Sarangani", "", "province", "sarangani-philippines"},
		// The Special Geographic Area: no ISO code, no name:en, its own word.
		{"Special Geographic Area", tags("name", "Special Geographic Area", "short_name", "SGA", "alt_name", "Ligawasan Province", "border_type", "special_geographic_area"),
			"Special Geographic Area", "", "special geographic area", "special-geographic-area-philippines"},
		// Metro Manila and the cities: "region" and "city", the word " City"
		// stays in the name. Tag sets of the extract of 2026-11-01.
		{"Metro Manila", tags("name", "Metro Manila", "name:en", "Metro Manila", "name:tl", "Kalakhang Maynila", "official_name:en", "National Capital Region", "ISO3166-2", "PH-00"),
			"Metro Manila", "Kalakhang Maynila", "region", "metro-manila-philippines"},
		{"Davao City: name:tl equals name:en", tags("name", "Davao City", "name:en", "Davao City", "name:tl", "Davao City", "official_name", "City of Davao"),
			"Davao City", "", "city", "davao-city-philippines"},
		{"Cebu City: name:tl is the short name", tags("name", "Cebu City", "name:en", "Cebu City", "name:tl", "Cebu", "official_name:tl", "Lungsod ng Cebu"),
			"Cebu City", "Cebu", "city", "cebu-city-philippines"},
		{"Naga City: name only", tags("name", "Naga City", "official_name", "City of Naga", "border_type", "independent_component_city"),
			"Naga City", "", "city", "naga-city-philippines"},
		{"Cotabato City", tags("name", "Cotabato City", "name:en", "Cotabato City", "short_name", "Cotabato"),
			"Cotabato City", "", "city", "cotabato-city-philippines"},
		{"Iligan: whole name", tags("name", "Iligan", "name:en", "Iligan", "name:tl", "Iligan", "official_name", "City of Iligan"),
			"Iligan", "", "city", "iligan-philippines"},
		{"General Santos: whole name", tags("name", "General Santos", "name:en", "General Santos", "official_name", "City of General Santos"),
			"General Santos", "", "city", "general-santos-philippines"},
		// Provinces whose names are the cities' stay provinces.
		{"Cotabato province", tags("name", "Cotabato", "name:en", "Cotabato", "ISO3166-2", "PH-NCO"),
			"Cotabato", "", "province", "cotabato-philippines"},
		{"Iloilo province", tags("name", "Iloilo", "name:en", "Iloilo", "ISO3166-2", "PH-ILI"),
			"Iloilo", "", "province", "iloilo-philippines"},
	}
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, ph, ph.Province)
		if n.En != tc.en || n.Local != tc.local || n.Kind != tc.kind {
			t.Errorf("%s: got %q %q %q, want %q %q %q", tc.name, n.En, n.Local, n.Kind, tc.en, tc.local, tc.kind)
		}
		if got := osmfile.Slug(n.En) + "-philippines"; got != tc.slug {
			t.Errorf("%s: slug %q, want %q", tc.name, got, tc.slug)
		}
	}
	if k := osmfile.KindName("Cebu", "", ph.Province, "province"); k != "" {
		t.Errorf("province: kind_name %q, want NULL", k)
	}
	if k := osmfile.KindName("Special Geographic Area", "", ph.Province, "province"); k != "special geographic area" {
		t.Errorf("Special Geographic Area: kind_name %q", k)
	}

	// Places: English from name:en, else name; the Filipino name from
	// name:tl, else a name that differs from name:en; Han names from the
	// Spratlys are not English.
	places := []struct {
		tags      osm.Tags
		en, local string
	}{
		{tags("name", "Lahug"), "Lahug", ""},
		{tags("name", "Poblacion", "name:es", "Población"), "Poblacion", ""},
		{tags("name", "Manila", "name:en", "Manila", "name:tl", "Maynila"), "Manila", "Maynila"},
		{tags("name", "Quezon City", "name:en", "Quezon City", "name:tl", "Lungsod Quezon"), "Quezon City", "Lungsod Quezon"},
		{tags("name", "Banaba Kanluran", "name:en", "Banaba West"), "Banaba West", "Banaba Kanluran"},
		{tags("name", "Kaunlaran I Subdivision", "name:tl", "Bahayang Kaunlaran I"), "Kaunlaran I Subdivision", "Bahayang Kaunlaran I"},
		{tags("name", "渚碧岛", "name:en", "Zhubi"), "Zhubi", ""},
		{tags("name", "南沙区"), "", ""},
	}
	for _, tc := range places {
		en, local := osmfile.PlaceNames(tc.tags, ph)
		if en != tc.en || local != tc.local {
			t.Errorf("place %v: got %q %q, want %q %q", tc.tags, en, local, tc.en, tc.local)
		}
	}
}
