package countries_test

import (
	"testing"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

func myanmar(t *testing.T) countries.Import {
	t.Helper()
	mm, ok := countries.ImportOf("mm")
	if !ok || mm.Hold != "" {
		t.Fatalf("Myanmar: settings %v, hold %q", ok, mm.Hold)
	}
	return mm
}

func mmTags(kv ...string) osm.Tags {
	var t osm.Tags
	for i := 0; i < len(kv); i += 2 {
		t = append(t, osm.Tag{Key: kv[i], Value: kv[i+1]})
	}
	return t
}

func TestMyanmarSettings(t *testing.T) {
	mm := myanmar(t)
	if mm.Extract != "myanmar" || mm.Province.Count != [2]int{15, 15} || mm.Province.ISOPrefix != "MM" ||
		len(mm.District.AdminLevels) != 1 || mm.District.AdminLevels[0] != 7 ||
		mm.LocalLang != "my" || !mm.SlugSuffix || len(mm.Neighbours) != 0 || len(mm.Partial) != 0 || mm.LandPlots != 804_966 {
		t.Errorf("Myanmar's settings: %+v", mm)
	}
}

// The 15 provinces as myanmar-latest.osm.pbf of 2026-11-01 tags them
// (name, name:en, name:my, int_name): every one gets its word, seven states,
// seven regions and the union territory, from the English name where it
// carries the word and from the Burmese one where it does not.
func TestMyanmarProvinces(t *testing.T) {
	mm := myanmar(t)
	cases := []struct {
		tags            osm.Tags
		en, local, kind string
		slug            string
	}{
		{mmTags("name", "ဧရာဝတီတိုင်းဒေသကြီး", "name:en", "Ayeyarwady", "name:my", "ဧ​ရာဝတီတိုင်း", "int_name", "Ayeyarwady Region"),
			"Ayeyarwady", "ဧ​ရာဝတီ", "region", "ayeyarwady-myanmar"}, // the zero-width space is OSM's
		{mmTags("name", "ပဲခူးတိုင်းဒေသကြီး", "name:en", "Bago Region", "name:my", "ပဲခူးတိုင်းဒေသကြီး", "int_name", "Bago Region"),
			"Bago", "ပဲခူး", "region", "bago-myanmar"},
		{mmTags("name", "ချင်းပြည်နယ်", "name:en", "Chin", "name:my", "ချင်းပြည်နယ်", "int_name", "Chin State"),
			"Chin", "ချင်း", "state", "chin-myanmar"},
		{mmTags("name", "ကချင်ပြည်နယ်", "name:en", "Kachin State", "name:my", "ကချင်ပြည်နယ်", "int_name", "Kachin State"),
			"Kachin", "ကချင်", "state", "kachin-myanmar"},
		{mmTags("name", "ကယားပြည်နယ်", "name:en", "Kayah State", "name:my", "ကယားပြည်နယ်", "int_name", "Kayah State"),
			"Kayah", "ကယား", "state", "kayah-myanmar"},
		{mmTags("name", "ကရင်ပြည်နယ်", "int_name", "Kayin State"), // no name:en, no name:my
			"Kayin", "ကရင်", "state", "kayin-myanmar"},
		{mmTags("name", "မကွေးတိုင်းဒေသကြီး", "name:en", "Magway", "name:my", "မကွေးတိုင်းဒေသကြီး", "int_name", "Magway Region"),
			"Magway", "မကွေး", "region", "magway-myanmar"},
		{mmTags("name", "မန္တလေးတိုင်း", "name:en", "Mandalay", "name:my", "မန္တလေးတိုင်း", "int_name", "Mandalay Region"),
			"Mandalay", "မန္တလေး", "region", "mandalay-myanmar"},
		{mmTags("name", "မွန်ပြည်နယ်", "name:en", "Mon State", "name:my", "မွန်ပြည်နယ်", "int_name", "Mon State"),
			"Mon", "မွန်", "state", "mon-myanmar"},
		{mmTags("name", "နေပြည်တော် ပြည်ထောင်စုနယ်မြေ", "name:en", "Naypyitaw Union Territory", "name:my", "နေပြည်တော် ပြည်ထောင်စုနယ်မြေ", "int_name", "Naypyidaw Union Territory"),
			"Naypyitaw", "နေပြည်တော်", "union territory", "naypyitaw-myanmar"},
		{mmTags("name", "ရခိုင်ပြည်နယ်", "name:en", "Rakhine", "name:my", "ရခိုင်ပြည်နယ်"),
			"Rakhine", "ရခိုင်", "state", "rakhine-myanmar"},
		{mmTags("name", "စစ်ကိုင်းတိုင်းဒေသကြီး", "name:en", "Sagaing Region", "name:my", "စစ်ကိုင်းတိုင်းဒေသကြီး", "int_name", "Sagaing Region"),
			"Sagaing", "စစ်ကိုင်း", "region", "sagaing-myanmar"},
		{mmTags("name", "ရှမ်းပြည်နယ်", "name:en", "Shan State", "name:my", "ရှမ်းပြည်နယ်", "int_name", "Shan State"),
			"Shan", "ရှမ်း", "state", "shan-myanmar"},
		{mmTags("name", "တနင်္သာရီတိုင်း", "name:en", "Tanintharyi Region", "name:my", "တနင်္သာရီတိုင်း", "int_name", "Tanintharyi Region"),
			"Tanintharyi", "တနင်္သာရီ", "region", "tanintharyi-myanmar"},
		{mmTags("name", "ရန်ကုန်တိုင်းဒေသကြီး", "name:en", "Yangon", "name:my", "ရန်ကုန်တိုင်းဒေသကြီး", "int_name", "Yangon Region"),
			"Yangon", "ရန်ကုန်", "region", "yangon-myanmar"},
	}
	words := map[string]int{}
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, mm, mm.Province)
		if n.En != tc.en || n.Local != tc.local || n.Kind != tc.kind {
			t.Errorf("%s: got %q %q %q, want %q %q %q", tc.slug, n.En, n.Local, n.Kind, tc.en, tc.local, tc.kind)
		}
		if got := osmfile.Slug(n.En) + "-myanmar"; got != tc.slug {
			t.Errorf("%s: slug %q", tc.slug, got)
		}
		if k := osmfile.KindName(n.RawEn, n.RawLocal, mm.Province, "province"); k != tc.kind {
			t.Errorf("%s: kind_name %q, want %q", tc.slug, k, tc.kind)
		}
		words[n.Kind]++
	}
	if words["state"] != 7 || words["region"] != 7 || words["union territory"] != 1 {
		t.Errorf("kind words %v, want 7 states, 7 regions, 1 union territory", words)
	}
}

// Only the level-4 relations with an MM code are provinces: not the two Wa
// areas inside Shan State (no ISO code), not the neighbours' provinces that
// the extract cuts. Townships are level 7.
func TestMyanmarRelations(t *testing.T) {
	mm := myanmar(t)
	rel := func(id int64, level, iso string) *osm.Relation {
		t := mmTags("boundary", "administrative", "admin_level", level)
		if iso != "" {
			t = append(t, osm.Tag{Key: "ISO3166-2", Value: iso})
		}
		return &osm.Relation{ID: osm.RelationID(id), Tags: t}
	}
	for _, tc := range []struct {
		r              *osm.Relation
		province, dist bool
	}{
		{rel(5996485, "4", "MM-17"), true, false},  // Shan State
		{rel(5996482, "4", "MM-18"), true, false},  // Naypyitaw
		{rel(9399861, "4", ""), false, false},      // Northern Wa State
		{rel(9399860, "4", ""), false, false},      // Wa State Southern Region
		{rel(913094, "4", "CN-YN"), false, false},  // Yunnan
		{rel(1908771, "4", "TH-50"), false, false}, // Chiang Mai
		{rel(2027869, "4", "IN-MN"), false, false}, // Manipur
		{rel(3824588, "4", "BD-B"), false, false},  // Chattogram Division
		{rel(5831654, "4", "LA-BK"), false, false}, // Bokeo
		{rel(5996605, "7", ""), false, true},       // Falam Township
		{rel(5996518, "6", ""), false, false},      // Loikaw District
	} {
		if got := osmfile.MatchLevel(tc.r, mm.Province); got != tc.province {
			t.Errorf("r%d: province %v, want %v", tc.r.ID, got, tc.province)
		}
		if got := osmfile.MatchLevel(tc.r, mm.District); got != tc.dist {
			t.Errorf("r%d: district %v, want %v", tc.r.ID, got, tc.dist)
		}
	}
}

// Townships as the extract tags them: with and without the word, Burmese
// names from name:my before name, and the slips the suffixes catch.
func TestMyanmarTownships(t *testing.T) {
	mm := myanmar(t)
	cases := []struct {
		name      string
		tags      osm.Tags
		en, local string
		slug      string
	}{
		{"word in both", mmTags("name", "ဖလမ်းမြို့နယ်", "name:en", "Falam Township", "name:my", "ဖလမ်းမြို့နယ်"),
			"Falam", "ဖလမ်း", "falam"},
		{"Yangon township without the word, name only", mmTags("name", "အလုံ", "name:en", "Ahlone"),
			"Ahlone", "အလုံ", "ahlone"},
		{"word only in name:my", mmTags("name", "ဂန့်ဂေါ", "name:en", "Gangaw Township", "name:my", "ဂန့်ဂေါမြို့နယ်"),
			"Gangaw", "ဂန့်ဂေါ", "gangaw"},
		{"Burmese word without its final asat", mmTags("name", "လက်ပတန်းမြို့နယ်", "name:en", "Letpadan Township", "name:my", "လက်ပတန်းမြို့နယ"),
			"Letpadan", "လက်ပတန်း", "letpadan"},
		{"zero-width non-joiner before the word, Chinese in name", mmTags("name", "ရှီးရှမ်း‌မြို့နယ် 西山区", "name:en", "Xishan Township", "name:my", "ရှီးရှမ်း‌မြို့နယ်"),
			"Xishan", "ရှီးရှမ်း", "xishan"},
		{"de facto", mmTags("name", "မိုင်းယောင်းမြို့နယ် (အမှန်တကယ်)", "name:en", "Mong Yawng Township (de facto)", "name:my", "မိုင်းယောင်းမြို့နယ် (အမှန်တကယ်)"),
			"Mong Yawng", "မိုင်းယောင်း", "mong-yawng"},
		{"Wa township: name in Wa and Chinese", mmTags("name", "Veng' Na' Vi' 纳威区", "name:en", "Navi Township", "name:my", "နာဝီးမြို့နယ်", "name:prk", "Veng' Na' Vi'"),
			"Navi", "နာဝီး", "navi"},
		{"Latin name, Burmese name:my", mmTags("name", "Mingaladon", "name:en", "Mingaladon", "name:my", "မင်္ဂလာဒုံ"),
			"Mingaladon", "မင်္ဂလာဒုံ", "mingaladon"},
		{"parentheses", mmTags("name", "လှိုင်သာယာ (အရှေ့ပိုင်း)", "name:en", "Hlaingtharya (East)"),
			"Hlaingtharya (East)", "လှိုင်သာယာ (အရှေ့ပိုင်း)", "hlaingtharya-east"},
		{"hyphen", mmTags("name", "ချောင်းဦးမြို့နယ်", "name:en", "Chaung-U Township", "name:my", "ချောင်းဦးမြို့နယ်"),
			"Chaung-U", "ချောင်းဦး", "chaung-u"},
	}
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, mm, mm.District)
		if n.En != tc.en || n.Local != tc.local || n.Kind != "township" {
			t.Errorf("%s: got %q %q %q, want %q %q township", tc.name, n.En, n.Local, n.Kind, tc.en, tc.local)
		}
		if got := osmfile.Slug(n.En); got != tc.slug {
			t.Errorf("%s: slug %q, want %q", tc.name, got, tc.slug)
		}
	}
	if k := osmfile.KindName("Falam Township", "ဖလမ်းမြို့နယ်", mm.District, "district"); k != "township" {
		t.Errorf("township: kind_name %q", k)
	}
}

// Places: English from name:en or a Latin name, never one in Myanmar script
// or a neighbour's; local from name:my, else a name in Myanmar script.
func TestMyanmarPlaces(t *testing.T) {
	mm := myanmar(t)
	for _, tc := range []struct {
		tags      osm.Tags
		en, local string
	}{
		{mmTags("name", "ရန်ကုန်", "name:en", "Yangon", "name:my", "ရန်ကုန်"), "Yangon", "ရန်ကုန်"},
		{mmTags("name", "အမှတ် (၃) ရပ်ကွက်", "name:en", "No. (3) Ward"), "No. (3) Ward", "အမှတ် (၃) ရပ်ကွက်"},
		{mmTags("name", "Yegyawgyi"), "Yegyawgyi", ""}, // a delta village named in Latin letters only (n263577284)
		{mmTags("name", "ထင်းရှုးခြံ"), "", "ထင်းရှုးခြံ"},
		{mmTags("name", "ချားဘော 查波"), "", "ချားဘော 查波"},
		{mmTags("name", "ဝၢၼ်ႈႁႆး", "name:en", "Wan Panghang"), "Wan Panghang", "ဝၢၼ်ႈႁႆး"}, // Shan, in Myanmar script
		{mmTags("int_name", "စည်ပင်သာယာ", "name", "စည်ပင်သာယာ"), "", "စည်ပင်သာယာ"},
		{mmTags("name", "小新寨"), "", ""},
		{mmTags("name", "บ้านปางม้า"), "", ""},
		{mmTags("name", "ຈ່ຽງແຂງ"), "", ""},
		{mmTags("name", "আলীখালী"), "", ""},
		{mmTags("name", "ေတာင္ညို", "name:en", "Taung Nyo"), "Taung Nyo", "ေတာင္ညို"}, // Zawgyi, kept as OSM has it
	} {
		en, local := osmfile.PlaceNames(tc.tags, mm)
		if en != tc.en || local != tc.local {
			t.Errorf("place %v: got %q %q, want %q %q", tc.tags, en, local, tc.en, tc.local)
		}
	}
}
