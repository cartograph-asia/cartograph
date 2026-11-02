package countries_test

import (
	"testing"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

func taiwan(t *testing.T) countries.Import {
	t.Helper()
	tw, ok := countries.ImportOf("tw")
	if !ok || tw.Hold != "" {
		t.Fatalf("Taiwan: settings %v, hold %q", ok, tw.Hold)
	}
	return tw
}

func twTags(kv ...string) osm.Tags {
	var t osm.Tags
	for i := 0; i < len(kv); i += 2 {
		t = append(t, osm.Tag{Key: kv[i], Value: kv[i+1]})
	}
	return t
}

// Taiwan's settings on tag sets as the taiwan-latest.osm.pbf of 2026-11-01
// holds them (docs/countries/taiwan.md).
func TestTaiwanNames(t *testing.T) {
	tw := taiwan(t)
	if tw.LocalLang != "zh-Hant" || !tw.SlugSuffix || tw.Province.Count != [2]int{22, 22} {
		t.Errorf("Taiwan: lang %q, suffix %v, count %v", tw.LocalLang, tw.SlugSuffix, tw.Province.Count)
	}
	cases := []struct {
		name            string
		tags            osm.Tags
		level           countries.Level
		en, local, kind string
		slug            string
	}{
		// Provinces: 市 city, 縣 county, from the Chinese name; the English
		// names carry the word unevenly.
		{"special municipality, bare English name", twTags("name", "臺北市", "name:en", "Taipei", "name:zh", "臺北市", "ISO3166-2", "TW-TPE"),
			tw.Province, "Taipei", "臺北", "city", "taipei"},
		{"provincial city", twTags("name", "基隆市", "name:en", "Keelung", "name:zh", "基隆市"),
			tw.Province, "Keelung", "基隆", "city", "keelung"},
		{"the one city with City", twTags("name", "桃園市", "name:en", "Taoyuan City", "name:zh", "桃園市"),
			tw.Province, "Taoyuan", "桃園", "city", "taoyuan"},
		{"county", twTags("name", "彰化縣", "name:en", "Changhua County", "name:zh", "彰化縣"),
			tw.Province, "Changhua", "彰化", "county", "changhua"},
		{"county without County", twTags("name", "金門縣", "name:en", "Kinmen", "name:zh", "金門縣"),
			tw.Province, "Kinmen", "金門", "county", "kinmen"},
		{"Lienchiang (Matsu)", twTags("name", "連江縣", "name:en", "Lienchiang County", "name:zh", "連江縣"),
			tw.Province, "Lienchiang", "連江", "county", "lienchiang"},

		// Districts at level 7: 區, " District" (162 of 170) or bare.
		{"district", twTags("name", "淡水區", "name:en", "Tamsui District", "name:zh", "淡水區"),
			tw.District, "Tamsui", "淡水", "district", "tamsui"},
		{"district with a bare English name", twTags("name", "北投區", "name:en", "Beitou", "name:zh", "北投區"),
			tw.District, "Beitou", "北投", "district", "beitou"},
		{"鎮 inside a district name", twTags("name", "前鎮區", "name:en", "Qianzhen District", "name:zh", "前鎮區"),
			tw.District, "Qianzhen", "前鎮", "district", "qianzhen"},
		{"平鎮區", twTags("name", "平鎮區", "name:en", "Pingzhen District", "name:zh", "平鎮區"),
			tw.District, "Pingzhen", "平鎮", "district", "pingzhen"},
		{"市 inside a district name", twTags("name", "新市區", "name:en", "Xinshi District", "name:zh", "新市區"),
			tw.District, "Xinshi", "新市", "district", "xinshi"},
		{"a direction", twTags("name", "北區", "name:en", "North District", "name:zh", "北區"),
			tw.District, "North", "北", "district", "north"},
		{"apostrophe", twTags("name", "仁愛區", "name:en", "Ren'ai", "name:zh", "仁愛區"),
			tw.District, "Ren'ai", "仁愛", "district", "renai"},
		{"OSM's Iamtiaⁿ", twTags("name", "鹽埕區", "name:en", "Iamtiaⁿ District", "name:zh", "鹽埕區"),
			tw.District, "Iamtiaⁿ", "鹽埕", "district", "iamtia"},

		// Districts at level 8: 鄉 township, 鎮 town, 市 city.
		{"township", twTags("name", "阿里山鄉", "name:en", "Alishan Township", "name:zh", "阿里山鄉"),
			tw.District, "Alishan", "阿里山", "township", "alishan"},
		{"township with a bare English name", twTags("name", "九如鄉", "name:en", "Jiuru", "name:zh", "九如鄉"),
			tw.District, "Jiuru", "九如", "township", "jiuru"},
		{"town", twTags("name", "竹南鎮", "name:en", "Zhunan Town", "name:zh", "竹南鎮"),
			tw.District, "Zhunan", "竹南", "town", "zhunan"},
		{"town with a bare English name", twTags("name", "恆春鎮", "name:en", "Hengchun", "name:zh", "恆春鎮"),
			tw.District, "Hengchun", "恆春", "town", "hengchun"},
		{"a township called Town in English", twTags("name", "東引鄉", "name:en", "Dongyin Town", "name:zh", "東引鄉"),
			tw.District, "Dongyin", "東引", "township", "dongyin"},
		{"county-administered city", twTags("name", "屏東市", "name:en", "Pingtung City", "name:zh", "屏東市", "name:zh-Hant", "屏東市"),
			tw.District, "Pingtung", "屏東", "city", "pingtung"},
		{"county-administered city, bare", twTags("name", "頭份市", "name:en", "Toufen", "name:zh", "頭份市"),
			tw.District, "Toufen", "頭份", "city", "toufen"},
		{"name with the indigenous name", twTags("name", "蘭嶼鄉 Ponso no Tao", "name:en", "Orchid Island", "name:zh", "蘭嶼鄉"),
			tw.District, "Orchid Island", "蘭嶼", "township", "orchid-island"},
		{"no English name", twTags("name", "大坵嶼", "name:zh-Hant", "大坵嶼"),
			tw.District, "", "大坵嶼", "district", ""},
	}
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, tw, tc.level)
		if n.En != tc.en || n.Local != tc.local || n.Kind != tc.kind {
			t.Errorf("%s: got %q %q %q, want %q %q %q", tc.name, n.En, n.Local, n.Kind, tc.en, tc.local, tc.kind)
		}
		if got := osmfile.Slug(n.En); got != tc.slug {
			t.Errorf("%s: slug %q, want %q", tc.name, got, tc.slug)
		}
	}

	// kind_name: every province has a word of its own; a 區 district keeps NULL.
	if k := osmfile.KindName("Taipei", "臺北市", tw.Province, "province"); k != "city" {
		t.Errorf("Taipei: kind_name %q", k)
	}
	if k := osmfile.KindName("Penghu", "澎湖縣", tw.Province, "province"); k != "county" {
		t.Errorf("Penghu: kind_name %q", k)
	}
	if k := osmfile.KindName("Tamsui District", "淡水區", tw.District, "district"); k != "" {
		t.Errorf("Tamsui: kind_name %q, want none", k)
	}
	if k := osmfile.KindName("Alishan Township", "阿里山鄉", tw.District, "district"); k != "township" {
		t.Errorf("Alishan: kind_name %q", k)
	}

	// Places: English from name:en or a Latin name, never Han or pinyin from
	// name:zh-Latn-pinyin; local from name:zh-Hant, name:zh, else a Han name.
	places := []struct {
		tags      osm.Tags
		en, local string
	}{
		{twTags("name", "霧社 Alang Paran", "name:en", "Wushe", "name:zh", "霧社", "name:trv", "Alang Paran"), "Wushe", "霧社"},
		{twTags("name", "烏來區 Ulay", "name:en", "Wulai", "name:nan-Hant", "烏來區"), "Wulai", "烏來區 Ulay"},
		{twTags("name", "車埕", "name:zh", "車埕", "name:zh-Latn-pinyin", "Chēchéng"), "", "車埕"},
		{twTags("name", "莿桐巷", "name:zh-Latn-pinyin", "citongxiang"), "", "莿桐巷"},
		{twTags("name", "子龍廟"), "", "子龍廟"},
		{twTags("name", "Lesan"), "Lesan", ""},
		{twTags("name", "塩埕埔", "name:zh", "鹽埕埔", "name:zh-Hant", "鹽埕埔"), "", "鹽埕埔"},
		{twTags("name", "ヤミ"), "", ""},
	}
	for _, tc := range places {
		en, local := osmfile.PlaceNames(tc.tags, tw)
		if en != tc.en || local != tc.local {
			t.Errorf("place %v: got %q %q, want %q %q", tc.tags, en, local, tc.en, tc.local)
		}
	}
}

// Chiayi and Hsinchu are each a county and a city: after stripping they
// collide, and by decision 7 both keep their words.
func TestTaiwanCountyAndCityOfOneName(t *testing.T) {
	tw := taiwan(t)
	sibs := []osmfile.Sibling{
		{OSMID: 2790418, Names: osmfile.LevelNames(twTags("name", "嘉義市", "name:en", "Chiayi", "name:zh", "嘉義市"), tw, tw.Province)},
		{OSMID: 2908173, Names: osmfile.LevelNames(twTags("name", "嘉義縣", "name:en", "Chiayi County", "name:zh", "嘉義縣"), tw, tw.Province)},
		{OSMID: 2849488, Names: osmfile.LevelNames(twTags("name", "新竹市", "name:en", "Hsinchu", "name:zh", "新竹市"), tw, tw.Province)},
		{OSMID: 2912613, Names: osmfile.LevelNames(twTags("name", "新竹縣", "name:en", "Hsinchu County", "name:zh", "新竹縣"), tw, tw.Province)},
		{OSMID: 2917549, Names: osmfile.LevelNames(twTags("name", "彰化縣", "name:en", "Changhua County", "name:zh", "彰化縣"), tw, tw.Province)},
	}
	osmfile.ResolveSlugs(sibs, func(base string) string { return base + "-taiwan" })
	want := []struct{ slug, en, local string }{
		{"chiayi-taiwan", "Chiayi", "嘉義市"},
		{"chiayi-county-taiwan", "Chiayi County", "嘉義縣"},
		{"hsinchu-taiwan", "Hsinchu", "新竹市"},
		{"hsinchu-county-taiwan", "Hsinchu County", "新竹縣"},
		{"changhua-taiwan", "Changhua", "彰化"},
	}
	for i, w := range want {
		if s := sibs[i]; s.Slug != w.slug || s.Names.En != w.en || s.Names.Local != w.local {
			t.Errorf("r%d: got %q %q %q, want %q %q %q", s.OSMID, s.Slug, s.Names.En, s.Names.Local, w.slug, w.en, w.local)
		}
	}
}

// Provinces are the level-4 relations with TW codes (Fujian, CN-FJ, is in
// the extract too); districts are levels 7 and 8, never the villages at 9.
func TestTaiwanLevels(t *testing.T) {
	tw := taiwan(t)
	rel := func(id int64, kv ...string) *osm.Relation {
		return &osm.Relation{ID: osm.RelationID(id), Tags: twTags(append([]string{"boundary", "administrative"}, kv...)...)}
	}
	for _, tc := range []struct {
		r                  *osm.Relation
		province, district bool
	}{
		{rel(1293250, "admin_level", "4", "ISO3166-2", "TW-TPE"), true, false},
		{rel(3339695, "admin_level", "4", "ISO3166-2", "TW-KIN"), true, false},
		{rel(553303, "admin_level", "4", "ISO3166-2", "CN-FJ"), false, false},
		{rel(1296139, "admin_level", "7"), false, true},
		{rel(2127210, "admin_level", "8"), false, true},
		{rel(11952574, "admin_level", "8"), false, true}, // a Xiamen subdistrict: matched, then dropped by its centre
		{rel(1, "admin_level", "9"), false, false},
		{rel(2, "admin_level", "6"), false, false},
	} {
		if got := osmfile.MatchLevel(tc.r, tw.Province); got != tc.province {
			t.Errorf("r%d %v: province %v, want %v", tc.r.ID, tc.r.Tags, got, tc.province)
		}
		if got := osmfile.MatchLevel(tc.r, tw.District); got != tc.district {
			t.Errorf("r%d %v: district %v, want %v", tc.r.ID, tc.r.Tags, got, tc.district)
		}
	}
	if len(tw.Neighbours) != 2 || tw.Neighbours[0] != "china" || tw.Neighbours[1] != "philippines" || len(tw.Partial) != 0 {
		t.Errorf("Taiwan: neighbours %v, partial %v", tw.Neighbours, tw.Partial)
	}
	if len(tw.PlaceKinds) != 0 {
		t.Errorf("Taiwan: place kinds %v, want the default six (hamlets undecided)", tw.PlaceKinds)
	}
}
