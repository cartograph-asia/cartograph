package countries_test

import (
	"slices"
	"testing"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

func idTags(kv ...string) osm.Tags {
	var t osm.Tags
	for i := 0; i < len(kv); i += 2 {
		t = append(t, osm.Tag{Key: kv[i], Value: kv[i+1]})
	}
	return t
}

func indonesia(t *testing.T) countries.Import {
	t.Helper()
	id, ok := countries.ImportOf("id")
	if !ok || id.Hold != "" {
		t.Fatalf("Indonesia: settings %v, hold %q", ok, id.Hold)
	}
	return id
}

func TestIndonesiaSettings(t *testing.T) {
	id := indonesia(t)
	if id.Extract != "indonesia" || id.Province.Count != [2]int{38, 38} || id.Province.ISOPrefix != "ID" ||
		!slices.Equal(id.Province.AdminLevels, []int{4}) || !slices.Equal(id.District.AdminLevels, []int{5}) ||
		id.LocalLang != "id" || !id.SlugSuffix || len(id.Partial) != 0 || id.LegacyPlaces || id.LegacyDuplicateSlug {
		t.Errorf("Indonesia's settings: %+v", id)
	}
	// Riau Islands and North Kalimantan close only with the Malaysian cut.
	if !slices.Equal(id.Neighbours, []string{"malaysia-singapore-brunei"}) {
		t.Errorf("Indonesia's neighbours: %v", id.Neighbours)
	}
}

// Indonesia's settings on tag sets as indonesia-latest.osm.pbf of 2026-11-01
// holds them (docs/countries/indonesia.md): provinces with their special
// status in the name, regencies and cities with or without their word.
func TestIndonesiaNames(t *testing.T) {
	id := indonesia(t)
	cases := []struct {
		name            string
		tags            osm.Tags
		level           countries.Level
		en, local, kind string
		slug            string // province: the full slug; district: the name part
	}{
		{"province, equal names", idTags("name", "Bali", "name:en", "Bali", "name:id", "Bali", "official_name", "Provinsi Bali"),
			id.Province, "Bali", "", "province", "bali-indonesia"},
		{"province, translated", idTags("name", "Jawa Barat", "name:en", "West Java", "name:id", "Jawa Barat"),
			id.Province, "West Java", "Jawa Barat", "province", "west-java-indonesia"},
		{"province with a hyphen", idTags("name", "Kepulauan Bangka Belitung", "name:en", "Bangka-Belitung Islands", "name:id", "Kepulauan Bangka Belitung"),
			id.Province, "Bangka-Belitung Islands", "Kepulauan Bangka Belitung", "province", "bangka-belitung-islands-indonesia"},
		{"Riau Islands beside Riau", idTags("name", "Kepulauan Riau", "name:en", "Riau Islands", "name:id", "Kepulauan Riau"),
			id.Province, "Riau Islands", "Kepulauan Riau", "province", "riau-islands-indonesia"},
		{"special region by its words", idTags("name", "Daerah Istimewa Yogyakarta", "name:en", "Special Region of Yogyakarta", "name:id", "Daerah Istimewa Yogyakarta"),
			id.Province, "Yogyakarta", "", "special region", "yogyakarta-indonesia"},
		{"special capital region", idTags("name", "Daerah Khusus Ibukota Jakarta", "name:en", "Special Capital Region of Jakarta", "name:id", "Daerah Khusus Ibukota Jakarta"),
			id.Province, "Jakarta", "", "special capital region", "jakarta-indonesia"},
		{"Aceh by its whole name", idTags("name", "Aceh", "name:en", "Aceh", "name:id", "Aceh"),
			id.Province, "Aceh", "", "special region", "aceh-indonesia"},
		{"Papua's new provinces", idTags("name", "Papua Pegunungan", "name:en", "Highland Papua", "name:id", "Papua Pegunungan"),
			id.Province, "Highland Papua", "Papua Pegunungan", "province", "highland-papua-indonesia"},

		{"regency, bare name", idTags("name", "Agam", "official_name", "Kabupaten Agam"),
			id.District, "Agam", "", "regency", "agam"},
		{"regency, Kabupaten in name", idTags("name", "Kabupaten Karo", "official_name", "Kabupaten Karo"),
			id.District, "Karo", "", "regency", "karo"},
		{"regency, Kab in name", idTags("name", "Kab Serang", "official_name", "Kabupaten Serang"),
			id.District, "Serang", "", "regency", "serang"},
		{"regency, Regency in name:en", idTags("name", "Klungkung", "name:en", "Klungkung Regency", "name:id", "Kabupaten Klungkung"),
			id.District, "Klungkung", "", "regency", "klungkung"},
		{"regency, Regency in name", idTags("name", "Nias Regency", "official_name", "Kabupaten Nias"),
			id.District, "Nias", "", "regency", "nias"},
		{"regency, translated", idTags("name", "Kepulauan Seribu", "name:en", "Thousand Islands", "official_name", "Kabupaten Administrasi Kepulauan Seribu"),
			id.District, "Thousand Islands", "Kepulauan Seribu", "regency", "thousand-islands"},
		{"Kota at the end is the name", idTags("name", "Lima Puluh Kota", "official_name", "Kabupaten Lima Puluh Kota"),
			id.District, "Lima Puluh Kota", "", "regency", "lima-puluh-kota"},
		{"Kota without a space is the name", idTags("name", "Kotabaru", "name:en", "Kotabaru", "name:id", "Kotabaru", "official_name", "Kabupaten Kotabaru"),
			id.District, "Kotabaru", "", "regency", "kotabaru"},
		{"city, Kota and City of", idTags("name", "Kota Medan", "name:en", "City of Medan"),
			id.District, "Medan", "", "city", "medan"},
		{"city, City in name:en", idTags("name", "Dumai", "name:en", "Dumai City", "name:id", "Dumai", "official_name", "Kota Dumai"),
			id.District, "Dumai", "", "city", "dumai"},
		{"city, Kota in name", idTags("name", "Kota Yogyakarta"),
			id.District, "Yogyakarta", "", "city", "yogyakarta"},
		// A city whose word is only in official_name, which the name rules
		// do not read: its kind word is the default.
		{"city without its word", idTags("name", "Kendari", "official_name", "Kota Kendari"),
			id.District, "Kendari", "", "regency", "kendari"},
		{"Jakarta's administrative city", idTags("name", "Jakarta Selatan", "name:en", "South Jakarta", "official_name", "Kota Administrasi Jakarta Selatan"),
			id.District, "South Jakarta", "Jakarta Selatan", "regency", "south-jakarta"},
	}
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, id, tc.level)
		if n.En != tc.en || n.Local != tc.local || n.Kind != tc.kind {
			t.Errorf("%s: got %q %q %q, want %q %q %q", tc.name, n.En, n.Local, n.Kind, tc.en, tc.local, tc.kind)
		}
		slug := osmfile.Slug(n.En)
		if tc.level.ISOPrefix != "" {
			slug += "-indonesia"
		}
		if slug != tc.slug {
			t.Errorf("%s: slug %q, want %q", tc.name, slug, tc.slug)
		}
	}

	// Places: Indonesian names are Latin, so an English name equal to the
	// local one leaves the local NULL; a name in Han or Arabic script is not
	// English.
	places := []struct {
		tags      osm.Tags
		en, local string
	}{
		{idTags("name", "Jakarta", "name:en", "Jakarta"), "Jakarta", ""},
		{idTags("name", "Tretes"), "Tretes", ""},
		{idTags("name", "Ballaʼ Parang"), "Ballaʼ Parang", ""},
		{idTags("name", "Kampong Metal 美達村", "name:en", "Metal Village"), "Metal Village", "Kampong Metal 美達村"},
		{idTags("name", "美達村"), "", ""},
		{idTags("name", "باچوق"), "", ""},
		{idTags("name", "Kelurahan Menteng", "name:en", "Menteng", "official_name", "Kelurahan Menteng"), "Menteng", "Kelurahan Menteng"},
	}
	for _, tc := range places {
		en, local := osmfile.PlaceNames(tc.tags, id)
		if en != tc.en || local != tc.local {
			t.Errorf("place %v: got %q %q, want %q %q", tc.tags, en, local, tc.en, tc.local)
		}
	}
}

// Regencies and cities of one name in one province (decision 7 of
// docs/COUNTRY_IMPORT_STAGE2_BRIEF.md): where OSM writes the word in a name
// read, the pair keeps it; where both names are bare, the OSM id tells them
// apart. Pairs from West Java and East Java as OSM holds them.
func TestIndonesiaDuplicateDistricts(t *testing.T) {
	id := indonesia(t)
	sib := func(osmID int64, kv ...string) osmfile.Sibling {
		return osmfile.Sibling{OSMID: osmID, Names: osmfile.LevelNames(idTags(kv...), id, id.District)}
	}
	sibs := []osmfile.Sibling{
		sib(14745927, "name", "Bogor", "name:en", "Bogor", "name:id", "Bogor", "official_name", "Kota Bogor"),
		sib(14762112, "name", "Kab Bogor", "name:en", "Bogor", "name:id", "Bogor", "official_name", "Kabupaten Bogor"),
		sib(14509733, "name", "Bekasi", "name:en", "Bekasi", "name:id", "Bekasi", "official_name", "Kota Bekasi"),
		sib(14765575, "name", "Kab Bekasi", "name:en", "Kab Bekasi", "name:id", "Kab Bekasi", "official_name", "Kabupaten Bekasi"),
		sib(14935959, "name", "Kabupaten Bandung", "name:en", "Kabupaten Bandung", "name:id", "Kabupaten Bandung"),
		sib(13290062, "name", "Kota Bandung", "name:en", "Bandung City", "name:id", "Kota Bandung"),
		sib(9676374, "name", "Kabupaten Kediri", "name:en", "Kediri Regency", "official_name", "Kabupaten Kediri"),
		sib(9676375, "name", "Kota Kediri", "name:en", "Kediri City", "official_name", "Kota Kediri"),
		sib(9674152, "name", "Kabupaten Malang", "official_name", "Kabupaten Malang"),
		sib(9674151, "name", "Kota Malang", "official_name", "Kota Malang"),
		sib(1, "name", "Kota Batu", "official_name", "Kota Batu"),
		sib(2, "name", "Batu Bara", "official_name", "Kabupaten Batu Bara"),
	}
	osmfile.ResolveSlugs(sibs, func(base string) string { return "p-" + base })
	want := []struct{ slug, en, local, kind string }{
		{"p-bogor-14745927", "Bogor", "", "regency"},
		{"p-bogor-14762112", "Bogor", "", "regency"},
		{"p-bekasi", "Bekasi", "", "regency"},
		{"p-kab-bekasi", "Kab Bekasi", "", "regency"},
		{"p-kabupaten-bandung", "Kabupaten Bandung", "", "regency"},
		{"p-bandung-city", "Bandung City", "Kota Bandung", "city"},
		{"p-kediri-regency", "Kediri Regency", "Kabupaten Kediri", "regency"},
		{"p-kediri-city", "Kediri City", "Kota Kediri", "city"},
		{"p-kabupaten-malang", "Kabupaten Malang", "", "regency"},
		{"p-kota-malang", "Kota Malang", "", "city"},
		{"p-batu", "Batu", "", "city"},
		{"p-batu-bara", "Batu Bara", "", "regency"},
	}
	for i, w := range want {
		s := sibs[i]
		if s.Slug != w.slug || s.Names.En != w.en || s.Names.Local != w.local || s.Names.Kind != w.kind {
			t.Errorf("r%d: got %q %q %q %q, want %q %q %q %q", s.OSMID, s.Slug, s.Names.En, s.Names.Local, s.Names.Kind,
				w.slug, w.en, w.local, w.kind)
		}
	}
}
