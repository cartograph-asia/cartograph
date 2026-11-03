package osmfile

import (
	"testing"

	"github.com/cartograph-asia/cartograph/internal/countries"
)

// indonesiaOfficial is Indonesia's district level as the country branch
// writes it (import_id.go, 2026-11-03), with Level.OfficialName set;
// malaysiaOfficial Malaysia's states with the federal territories' word read
// from official_name. Made-up settings for the tests: the real ones come with
// each country's PR.
var indonesiaOfficial = countries.Import{
	Province: countries.Level{AdminLevels: []int{4}, ISOPrefix: "ID", Kind: "province"},
	District: countries.Level{AdminLevels: []int{5}, Kind: "regency",
		KindByAffix: map[string]string{"Kota ": "city", "City of ": "city", " City": "city",
			"Kabupaten ": "regency", "Kab ": "regency", " Regency": "regency"},
		Strip: countries.Affixes{
			EnSuffix:    []string{" Regency", " City"},
			EnPrefix:    []string{"City of ", "Kabupaten ", "Kota ", "Kab "},
			LocalSuffix: []string{" Regency"},
			LocalPrefix: []string{"Kabupaten ", "Kota ", "Kab "},
		},
		OfficialName: []string{"official_name:en", "official_name"}},
	English:      []string{"name:en", "int_name", "name:id-Latn", "name"},
	EnglishNot:   []string{"Arabic", "Han"},
	Local:        []string{"name:id", "name"},
	LocalScripts: []string{"Latin"},
	SlugSuffix:   true,
}

var malaysiaOfficial = countries.Import{
	Province: countries.Level{AdminLevels: []int{4}, ISOPrefix: "MY", Kind: "state",
		KindByAffix:  map[string]string{"Wilayah Persekutuan": "federal territory", "Federal Territory of": "federal territory"},
		OfficialName: []string{"official_name:en", "official_name"}},
	English:      []string{"name:en", "int_name", "name"},
	EnglishNot:   []string{"Arabic", "Han", "Tamil", "Thai"},
	Local:        []string{"name:ms", "name"},
	LocalScripts: []string{"Latin"},
	SlugSuffix:   true,
}

// withoutOfficial returns imp with no OfficialName on either level: today's
// behaviour.
func withoutOfficial(imp countries.Import) countries.Import {
	imp.Province.OfficialName, imp.District.OfficialName = nil, nil
	return imp
}

func TestOfficialNames(t *testing.T) {
	type want struct {
		en, local, kind, offEn, offLocal string
	}
	id, my := indonesiaOfficial, malaysiaOfficial
	for _, c := range []struct {
		name string
		imp  countries.Import
		l    countries.Level
		tags []string
		want want
	}{
		{"a city with its word in official_name only", id, id.District,
			[]string{"name", "Bogor", "name:en", "Bogor", "name:id", "Bogor", "official_name", "Kota Bogor"},
			want{"Bogor", "", "city", "Kota Bogor", "Kota Bogor"}},
		{"a regency named with an abbreviation, its words in official_name and official_name:en", id, id.District,
			[]string{"name", "Kab Bogor", "name:en", "Bogor", "name:id", "Bogor", "official_name", "Kabupaten Bogor",
				"official_name:en", "Bogor Regency", "official_name:id", "Kabupaten Bogor"},
			want{"Bogor", "", "regency", "Bogor Regency", "Kabupaten Bogor"}},
		{"official_name:en is English only, official_name serves the local name", id, id.District,
			[]string{"name", "Bogor", "name:en", "Bogor", "official_name", "Kota Bogor", "official_name:en", "Bogor City"},
			want{"Bogor", "", "city", "Bogor City", "Kota Bogor"}},
		{"Surabaya", id, id.District,
			[]string{"name", "Surabaya", "name:en", "Surabaya", "official_name", "Kota Surabaya"},
			want{"Surabaya", "", "city", "Kota Surabaya", "Kota Surabaya"}},
		{"a regency whose name ends in Kota", id, id.District,
			[]string{"name", "Lima Puluh Kota", "official_name", "Kabupaten Lima Puluh Kota"},
			want{"Lima Puluh Kota", "", "regency", "Kabupaten Lima Puluh Kota", "Kabupaten Lima Puluh Kota"}},
		{"Kotabaru has no word anywhere", id, id.District,
			[]string{"name", "Kotabaru"},
			want{"Kotabaru", "", "regency", "", ""}},
		{"a name with its word keeps it as the affixed form", id, id.District,
			[]string{"name", "Kabupaten Karo", "official_name", "Kabupaten Karo"},
			want{"Karo", "", "regency", "", ""}},
		{"the word in name:en wins over official_name", id, id.District,
			[]string{"name", "Kediri", "name:en", "Kediri City", "official_name", "Kabupaten Kediri"},
			want{"Kediri", "", "city", "", "Kabupaten Kediri"}},
		{"an official name that is not the name's form gives the kind word only", id, id.District,
			[]string{"name", "Jakarta Pusat", "name:en", "Central Jakarta", "official_name", "Kota Administrasi Jakarta Pusat"},
			want{"Central Jakarta", "Jakarta Pusat", "city", "", ""}},
		{"without the setting Surabaya is a regency", withoutOfficial(id), withoutOfficial(id).District,
			[]string{"name", "Surabaya", "name:en", "Surabaya", "official_name", "Kota Surabaya"},
			want{"Surabaya", "", "regency", "", ""}},
		{"Kuala Lumpur", my, my.Province,
			[]string{"name", "Kuala Lumpur", "name:en", "Kuala Lumpur", "official_name", "Wilayah Persekutuan Kuala Lumpur"},
			want{"Kuala Lumpur", "", "federal territory", "", ""}},
		{"Putrajaya", my, my.Province,
			[]string{"name", "Putrajaya", "official_name", "Wilayah Persekutuan Putrajaya", "official_name:en", "Federal Territory of Putrajaya"},
			want{"Putrajaya", "", "federal territory", "", ""}},
		{"Labuan", my, my.Province,
			[]string{"name", "Labuan", "official_name", "Wilayah Persekutuan Labuan"},
			want{"Labuan", "", "federal territory", "", ""}},
		{"a state", my, my.Province,
			[]string{"name", "Selangor", "official_name", "Negeri Selangor Darul Ehsan"},
			want{"Selangor", "", "state", "", ""}},
	} {
		t.Run(c.name, func(t *testing.T) {
			n := LevelNames(tags(c.tags...), c.imp, c.l)
			got := want{n.En, n.Local, n.Kind, n.OfficialEn, n.OfficialLocal}
			if got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

// Without the setting the official tags are never read: Thailand's names
// and kind words are those of the tags without them.
func TestOfficialNamesThailand(t *testing.T) {
	th := countries.Imports["th"]
	for _, kv := range [][]string{
		{"name", "อำเภอบางไทร", "name:en", "Bang Sai", "official_name", "อำเภอบางไทร", "official_name:en", "Bang Sai District"},
		{"name", "จังหวัดชลบุรี", "name:en", "Chon Buri", "official_name:en", "Chon Buri Province"},
	} {
		with := LevelNames(tags(kv...), th, th.District)
		without := LevelNames(tags(kv[:4]...), th, th.District)
		if with != without || with.OfficialEn != "" || with.OfficialLocal != "" {
			t.Errorf("%v: %+v, without the official tags %+v", kv, with, without)
		}
	}
}

func TestResolveSlugsOfficial(t *testing.T) {
	westJava := func(base string) string { return "west-java-indonesia-" + base }
	d := func(id int64, kv ...string) Sibling {
		return sibling(id, indonesiaOfficial, indonesiaOfficial.District, kv...)
	}
	type want struct{ slug, en, local string }
	for _, c := range []struct {
		name string
		sibs []Sibling
		want []want
	}{
		{"a bare pair takes the words of official_name", []Sibling{
			d(14745927, "name", "Bogor", "official_name", "Kota Bogor"),
			d(14762112, "name", "Bogor", "official_name", "Kabupaten Bogor"),
			d(3, "name", "Surabaya", "official_name", "Kota Surabaya"),
		}, []want{
			{"west-java-indonesia-kota-bogor", "Kota Bogor", ""},
			{"west-java-indonesia-kabupaten-bogor", "Kabupaten Bogor", ""},
			{"west-java-indonesia-surabaya", "Surabaya", ""},
		}},
		{"official_name:en for the English name, official_name:id or official_name for the local one", []Sibling{
			d(1, "name", "Bogor", "name:en", "Bogor", "official_name", "Kota Bogor", "official_name:en", "Bogor City"),
			d(2, "name", "Kab Bogor", "name:en", "Bogor", "name:id", "Bogor", "official_name", "Kabupaten Bogor",
				"official_name:en", "Bogor Regency", "official_name:id", "Kabupaten Bogor"),
		}, []want{
			{"west-java-indonesia-bogor-city", "Bogor City", "Kota Bogor"},
			{"west-java-indonesia-bogor-regency", "Bogor Regency", "Kabupaten Bogor"},
		}},
		{"one side with its word in name, the other in official_name", []Sibling{
			d(1, "name", "Kab Bekasi", "official_name", "Kabupaten Bekasi"),
			d(2, "name", "Bekasi", "official_name", "Kota Bekasi"),
		}, []want{
			{"west-java-indonesia-kab-bekasi", "Kab Bekasi", ""},
			{"west-java-indonesia-kota-bekasi", "Kota Bekasi", ""},
		}},
		{"a pair without official names still ends with the ids", []Sibling{
			d(1, "name", "Solok"),
			d(2, "name", "Solok"),
		}, []want{
			{"west-java-indonesia-solok-1", "Solok", ""},
			{"west-java-indonesia-solok-2", "Solok", ""},
		}},
		{"equal official names end with the ids, words and all", []Sibling{
			d(1, "name", "Solok", "official_name", "Kota Solok"),
			d(2, "name", "Solok", "official_name", "Kota Solok"),
		}, []want{
			{"west-java-indonesia-kota-solok-1", "Kota Solok", ""},
			{"west-java-indonesia-kota-solok-2", "Kota Solok", ""},
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			names := map[int64]Named{}
			for _, s := range c.sibs {
				names[s.OSMID] = s.Names
			}
			ResolveSlugs(c.sibs, westJava)
			for i, w := range c.want {
				s := c.sibs[i]
				if got := (want{s.Slug, s.Names.En, s.Names.Local}); got != w {
					t.Errorf("r%d: %+v, want %+v", s.OSMID, got, w)
				}
			}
			// A second run over what the first stored changes nothing.
			again := fresh(stored(c.sibs, nil), names)
			ResolveSlugs(again, westJava)
			for i, s := range again {
				if s.Slug != c.sibs[i].Slug || s.Names != c.sibs[i].Names {
					t.Errorf("second run, r%d: %q %+v, first %q %+v", s.OSMID, s.Slug, s.Names, c.sibs[i].Slug, c.sibs[i].Names)
				}
			}
		})
	}
}

// The setting never changes what is stored: a pair stored with bare names
// and ids (imported before the setting) keeps them; a relation renamed in
// OSM takes the form a fresh import gives it, never a new slug; a new
// sibling of a stored bare one takes its official words.
func TestResolveSlugsOfficialStable(t *testing.T) {
	westJava := func(base string) string { return "west-java-indonesia-" + base }
	before, after := withoutOfficial(indonesiaOfficial), indonesiaOfficial
	city := []string{"name", "Bogor", "official_name", "Kota Bogor"}
	regency := []string{"name", "Bogor", "official_name", "Kabupaten Bogor"}

	first := []Sibling{sibling(1, before, before.District, city...), sibling(2, before, before.District, regency...)}
	ResolveSlugs(first, westJava)
	if first[0].Slug != "west-java-indonesia-bogor-1" || first[1].Slug != "west-java-indonesia-bogor-2" {
		t.Fatalf("without the setting: %q %q", first[0].Slug, first[1].Slug)
	}
	second := fresh(stored(first, nil), map[int64]Named{
		1: LevelNames(tags(city...), after, after.District),
		2: LevelNames(tags(regency...), after, after.District),
	})
	ResolveSlugs(second, westJava)
	for i, s := range second {
		if s.Slug != first[i].Slug || s.Names.En != "Bogor" || s.Names.Local != "" {
			t.Errorf("stored before the setting, r%d: %q %+v", s.OSMID, s.Slug, s.Names)
		}
	}

	// Renamed in OSM: the stored slug stays, the name takes what a fresh
	// import of the pair gives (its official words).
	renamed := fresh(stored(first, nil), map[int64]Named{
		1: LevelNames(tags("name", "Bogor Raya", "official_name", "Kota Bogor Raya"), after, after.District),
		2: LevelNames(tags(regency...), after, after.District),
	})
	ResolveSlugs(renamed, westJava)
	if renamed[0].Slug != "west-java-indonesia-bogor-1" || renamed[0].Names.En != "Bogor Raya" || renamed[1].Names.En != "Bogor" {
		t.Errorf("renamed: %q %+v; %+v", renamed[0].Slug, renamed[0].Names, renamed[1].Names)
	}
	// Renamed so that the pair collides now: the renamed one takes its
	// official words under its stored slug, the stored bare one stays.
	apart := []Sibling{sibling(1, after, after.District, city...),
		sibling(2, after, after.District, "name", "Bogor Barat", "official_name", "Kabupaten Bogor Barat")}
	ResolveSlugs(apart, westJava)
	renamed = fresh(stored(apart, nil), map[int64]Named{
		1: LevelNames(tags(city...), after, after.District),
		2: LevelNames(tags(regency...), after, after.District),
	})
	ResolveSlugs(renamed, westJava)
	if renamed[0].Slug != "west-java-indonesia-bogor" || renamed[0].Names.En != "Bogor" ||
		renamed[1].Slug != "west-java-indonesia-bogor-barat" || renamed[1].Names.En != "Kabupaten Bogor" {
		t.Errorf("renamed into a collision: %q %+v; %q %+v", renamed[0].Slug, renamed[0].Names, renamed[1].Slug, renamed[1].Names)
	}

	// A stored bare Bogor alone, then its sibling appears: the stored one
	// keeps its slug and name, the new one takes its words.
	alone := []Sibling{sibling(1, after, after.District, city...)}
	ResolveSlugs(alone, westJava)
	if alone[0].Slug != "west-java-indonesia-bogor" || alone[0].Names.En != "Bogor" {
		t.Fatalf("alone: %q %+v", alone[0].Slug, alone[0].Names)
	}
	pair := append(stored(alone, nil), sibling(2, after, after.District, regency...))
	pair[0].Names = LevelNames(tags(city...), after, after.District)
	ResolveSlugs(pair, westJava)
	if pair[0].Slug != "west-java-indonesia-bogor" || pair[0].Names.En != "Bogor" ||
		pair[1].Slug != "west-java-indonesia-kabupaten-bogor" || pair[1].Names.En != "Kabupaten Bogor" {
		t.Errorf("new sibling: %q %+v; %q %+v", pair[0].Slug, pair[0].Names, pair[1].Slug, pair[1].Names)
	}
}
