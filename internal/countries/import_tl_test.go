package countries_test

import (
	"testing"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// Timor-Leste's settings on tag sets as the east-timor-latest.osm.pbf of
// 2026-11-01 holds them (docs/countries/timor-leste.md).
func TestTimorLesteNames(t *testing.T) {
	tl, ok := countries.ImportOf("tl")
	if !ok || tl.Hold != "" {
		t.Fatalf("Timor-Leste: settings %v, hold %q", ok, tl.Hold)
	}
	if tl.LocalLang != "tet" || !tl.SlugSuffix || tl.Province.Count != [2]int{14, 14} {
		t.Errorf("Timor-Leste: lang %q, suffix %v, count %v", tl.LocalLang, tl.SlugSuffix, tl.Province.Count)
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
		// Tetum name differs from the English one.
		{"municipality in Tetum", tags("name", "Baukau", "name:en", "Baucau", "name:pt", "Baucau", "name:tet", "Baukau"),
			tl.Province, "Baucau", "Baukau", "municipality", "baucau"},
		// Equal names: local is NULL.
		{"municipality, names equal", tags("name", "Dili", "name:en", "Dili", "name:pt", "Díli", "name:tet", "Dili"),
			tl.Province, "Dili", "", "municipality", "dili"},
		// name:tet, not name:pt, is the local name; the accent is folded in the slug.
		{"municipality with accent", tags("name", "Likisá", "name:en", "Liquiçá", "name:pt", "Liquiçá", "name:tet", "Likisá"),
			tl.Province, "Liquiçá", "Likisá", "municipality", "liquica"},
		{"name is Portuguese", tags("name", "Lautém", "name:en", "Lautém", "name:pt", "Lautém", "name:tet", "Lautein"),
			tl.Province, "Lautém", "Lautein", "municipality", "lautem"},
		// Atauro has no ISO code and no name:pt.
		{"Atauro", tags("name", "Ataúru", "name:en", "Atauro", "name:tet", "Ataúru"),
			tl.Province, "Atauro", "Ataúru", "municipality", "atauro"},
		{"special administrative region", tags("name", "Oe-Cusse Ambeno", "name:en", "Oecussi-Ambeno", "name:pt", "Oecussi-Ambeno", "name:tet", "Oe-Cusse Ambeno"),
			tl.Province, "Oecussi-Ambeno", "Oe-Cusse Ambeno", "municipality", "oecussi-ambeno"},
		// Administrative posts: most have name and name:tet only.
		{"post without name:en", tags("name", "Remexio", "name:tet", "Remesiu"),
			tl.District, "Remexio", "Remesiu", "administrative post", "remexio"},
		{"post, names equal", tags("name", "Cristo Rei", "name:tet", "Cristo Rei"),
			tl.District, "Cristo Rei", "", "administrative post", "cristo-rei"},
		{"post with name:en", tags("name", "Kelikai", "name:en", "Quelicai", "name:tet", "Kelikai"),
			tl.District, "Quelicai", "Kelikai", "administrative post", "quelicai"},
		{"post with accent", tags("name", "Lautém", "name:tet", "Lautein"),
			tl.District, "Lautém", "Lautein", "administrative post", "lautem"},
		// An OSM slip stays as OSM has it: name:tet of Loré names Lospalos.
		{"post whose name:tet is another's", tags("name", "Loré", "name:en", "Lore", "name:tet", "Lospalos"),
			tl.District, "Lore", "Lospalos", "administrative post", "lore"},
	}
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, tl, tc.level)
		if n.En != tc.en || n.Local != tc.local || n.Kind != tc.kind {
			t.Errorf("%s: got %q %q %q, want %q %q %q", tc.name, n.En, n.Local, n.Kind, tc.en, tc.local, tc.kind)
		}
		if got := osmfile.Slug(n.En); got != tc.slug {
			t.Errorf("%s: slug %q, want %q", tc.name, got, tc.slug)
		}
	}

	// Both words differ from the structural kinds, so kind_name is written.
	if k := osmfile.KindName("Dili", "Dili", tl.Province, "province"); k != "municipality" {
		t.Errorf("municipality: kind_name %q", k)
	}
	if k := osmfile.KindName("Remexio", "Remesiu", tl.District, "district"); k != "administrative post" {
		t.Errorf("administrative post: kind_name %q", k)
	}

	// Province slugs carry the country: oecussi-ambeno-timor-leste.
	if got := osmfile.Slug("Cova Lima") + "-timor-leste"; got != "cova-lima-timor-leste" {
		t.Errorf("province slug %q", got)
	}

	// The Indonesian provinces of the extract are not Timor-Leste's, Atauro is.
	rel := func(id int64, kv ...string) *osm.Relation {
		return &osm.Relation{ID: osm.RelationID(id), Tags: tags(kv...)}
	}
	prov := map[string]struct {
		r    *osm.Relation
		want bool
	}{
		"Dili":     {rel(4631022, "boundary", "administrative", "admin_level", "4", "ISO3166-2", "TL-DI"), true},
		"Atauro":   {rel(12363798, "boundary", "administrative", "admin_level", "4"), true},
		"East NT":  {rel(2396778, "boundary", "administrative", "admin_level", "4", "ISO3166-2", "ID-NT"), false},
		"no ISO":   {rel(1, "boundary", "administrative", "admin_level", "4"), false},
		"post":     {rel(12363788, "boundary", "administrative", "admin_level", "5"), false},
		"not a TL": {rel(2, "boundary", "administrative", "admin_level", "4", "ISO3166-2", "TH-10"), false},
	}
	for name, c := range prov {
		if got := osmfile.MatchLevel(c.r, tl.Province); got != c.want {
			t.Errorf("province level, %s: %v, want %v", name, got, c.want)
		}
	}
	if !osmfile.MatchLevel(rel(12363788, "boundary", "administrative", "admin_level", "5"), tl.District) {
		t.Error("district level: a level-5 relation does not match")
	}

	// Places: the extract holds West Timor's villages beyond the border,
	// in Latin too; the Tetum name is name:tet, else a Latin name.
	places := []struct {
		tags      osm.Tags
		en, local string
	}{
		{tags("name", "Lospalos"), "Lospalos", ""},
		{tags("name", "Baukau", "name:en", "Baucau"), "Baucau", "Baukau"},
		{tags("name", "Díli", "name:en", "Dili", "name:tet", "Dili"), "Dili", ""},
		{tags("name", "Pante Macassar", "name:tet", "Pante-Makasár"), "Pante Macassar", "Pante-Makasár"},
	}
	for _, tc := range places {
		en, local := osmfile.PlaceNames(tc.tags, tl)
		if en != tc.en || local != tc.local {
			t.Errorf("place %v: got %q %q, want %q %q", tc.tags, en, local, tc.en, tc.local)
		}
	}
}
