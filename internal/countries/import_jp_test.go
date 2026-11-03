package countries_test

import (
	"testing"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

func japan(t *testing.T) countries.Import {
	t.Helper()
	jp, ok := countries.ImportOf("jp")
	if !ok || jp.Hold != "" {
		t.Fatalf("Japan: settings %v, hold %q", ok, jp.Hold)
	}
	return jp
}

func jpTags(kv ...string) osm.Tags {
	var t osm.Tags
	for i := 0; i < len(kv); i += 2 {
		t = append(t, osm.Tag{Key: kv[i], Value: kv[i+1]})
	}
	return t
}

// Japan's settings on tag sets as the japan-latest.osm.pbf of 2026-11-01
// holds them (docs/countries/japan.md).
func TestJapanNames(t *testing.T) {
	jp := japan(t)
	if jp.LocalLang != "ja" || !jp.SlugSuffix || jp.Province.Count != [2]int{47, 47} {
		t.Errorf("Japan: lang %q, suffix %v, count %v", jp.LocalLang, jp.SlugSuffix, jp.Province.Count)
	}
	cases := []struct {
		name            string
		tags            osm.Tags
		level           countries.Level
		en, local, kind string
		slug            string
	}{
		// Prefectures: " Prefecture" goes, the Japanese designation stays.
		{"prefecture", jpTags("name", "愛知県", "name:en", "Aichi Prefecture", "name:ja", "愛知県", "name:ja-Latn", "Aichi-ken"),
			jp.Province, "Aichi", "愛知県", "prefecture", "aichi"},
		{"Tokyo: name holds 都, name:ja does not", jpTags("name", "東京都", "name:en", "Tokyo", "name:ja", "東京", "name:ja-Latn", "Tōkyō"),
			jp.Province, "Tokyo", "東京都", "metropolis", "tokyo"},
		{"Hokkaido keeps its 道", jpTags("name", "北海道", "name:en", "Hokkaido Prefecture", "name:ja", "北海道", "name:ja-Latn", "Hokkaidō"),
			jp.Province, "Hokkaido", "北海道", "prefecture", "hokkaido"},
		{"Kyoto", jpTags("name", "京都府", "name:en", "Kyoto Prefecture", "name:ja", "京都府", "name:ja-Latn", "Kyōto-fu"),
			jp.Province, "Kyoto", "京都府", "prefecture", "kyoto"},
		{"Osaka", jpTags("name", "大阪府", "name:en", "Osaka Prefecture", "name:ja", "大阪府", "name:ja-Latn", "Ōsaka-fu"),
			jp.Province, "Osaka", "大阪府", "prefecture", "osaka"},
		{"Hiroshima has no name:ja", jpTags("name", "広島県", "name:en", "Hiroshima Prefecture", "name:ja-Latn", "Hiroshima-ken"),
			jp.Province, "Hiroshima", "広島県", "prefecture", "hiroshima"},

		// Municipalities: the kind from the Japanese suffix, which stays.
		{"city", jpTags("name", "網走市", "name:en", "Abashiri", "name:ja", "網走市", "name:ja-Latn", "Abashiri-shi"),
			jp.District, "Abashiri", "網走市", "city", "abashiri"},
		{"town", jpTags("name", "大豊町", "name:en", "Otoyo", "name:ja", "大豊町", "name:ja-Latn", "Ōtoyo-chō"),
			jp.District, "Otoyo", "大豊町", "town", "otoyo"},
		{"village", jpTags("name", "生坂村", "name:en", "Ikusaka", "name:ja", "生坂村", "name:ja-Latn", "Ikusaka-mura"),
			jp.District, "Ikusaka", "生坂村", "village", "ikusaka"},
		{"special ward", jpTags("name", "新宿区", "name:en", "Shinjuku", "name:ja", "新宿区", "name:ja-Latn", "Shinjuku-ku"),
			jp.District, "Shinjuku", "新宿区", "special ward", "shinjuku"},
		{"special ward of one character", jpTags("name", "北区", "name:en", "Kita", "name:ja", "北区", "name:ja-Latn", "Kita-ku"),
			jp.District, "Kita", "北区", "special ward", "kita"},
		{"Okinawa City: name holds 市, name:ja does not", jpTags("name", "沖縄市", "name:en", "Okinawa", "name:ja", "沖縄", "name:ja-Latn", "Okinawa-shi"),
			jp.District, "Okinawa", "沖縄市", "city", "okinawa"},
		{"Ojiya City", jpTags("name", "小千谷市", "name:en", "Ojiya City", "name:ja", "小千谷市", "name:ja-Latn", "Ojiya-shi"),
			jp.District, "Ojiya", "小千谷市", "city", "ojiya"},
		// Accent folding (macrons).
		{"Kyōtango", jpTags("name", "京丹後市", "name:en", "Kyōtango", "name:ja", "京丹後市"),
			jp.District, "Kyōtango", "京丹後市", "city", "kyotango"},
		{"Higashiōsaka", jpTags("name", "東大阪市", "name:en", "Higashiōsaka", "name:ja", "東大阪市"),
			jp.District, "Higashiōsaka", "東大阪市", "city", "higashiosaka"},
		{"Ōmihachiman", jpTags("name", "近江八幡市", "name:en", "Ōmihachiman", "name:ja", "近江八幡市"),
			jp.District, "Ōmihachiman", "近江八幡市", "city", "omihachiman"},
		{"Bunkyō", jpTags("name", "文京区", "name:en", "Bunkyō", "name:ja", "文京区"),
			jp.District, "Bunkyō", "文京区", "special ward", "bunkyo"},
		// A name that starts with one kind character and ends with another.
		{"Murayama starts with 村, a city", jpTags("name", "村山市", "name:en", "Murayama", "name:ja", "村山市"),
			jp.District, "Murayama", "村山市", "city", "murayama"},
		{"Murakami", jpTags("name", "村上市", "name:en", "Murakami", "name:ja", "村上市"),
			jp.District, "Murakami", "村上市", "city", "murakami"},
		{"Machida starts with 町, a city", jpTags("name", "町田市", "name:en", "Machida", "name:ja", "町田市"),
			jp.District, "Machida", "町田市", "city", "machida"},
		{"Murata starts with 村, a town", jpTags("name", "村田町", "name:en", "Murata", "name:ja", "村田町"),
			jp.District, "Murata", "村田町", "town", "murata"},
		{"Ichikawa (Hyōgo) starts with 市, a town", jpTags("name", "市川町", "name:en", "Ichikawa", "name:ja", "市川町"),
			jp.District, "Ichikawa", "市川町", "town", "ichikawa"},
		{"Ichikai", jpTags("name", "市貝町", "name:en", "Ichikai", "name:ja", "市貝町"),
			jp.District, "Ichikai", "市貝町", "town", "ichikai"},
		{"Ichikawamisato", jpTags("name", "市川三郷町", "name:en", "Ichikawamisato", "name:ja", "市川三郷町"),
			jp.District, "Ichikawamisato", "市川三郷町", "town", "ichikawamisato"},
		{"Ichikawa (Chiba) is a city", jpTags("name", "市川市", "name:en", "Ichikawa", "name:ja", "市川市"),
			jp.District, "Ichikawa", "市川市", "city", "ichikawa"},
		{"Ichihara", jpTags("name", "市原市", "name:en", "Ichihara", "name:ja", "市原市"),
			jp.District, "Ichihara", "市原市", "city", "ichihara"},
		// Equal English names within one prefecture: the local name tells them apart.
		{"Kushiro City", jpTags("name", "釧路市", "name:en", "Kushiro", "name:ja", "釧路市"),
			jp.District, "Kushiro", "釧路市", "city", "kushiro"},
		{"Kushiro Town", jpTags("name", "釧路町", "name:en", "Kushiro", "name:ja", "釧路町"),
			jp.District, "Kushiro", "釧路町", "town", "kushiro"},
		// Only a Japanese name: no English name.
		{"no English name", jpTags("name", "所属未定地"),
			jp.District, "", "所属未定地", "municipality", ""},
	}
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, jp, tc.level)
		if n.En != tc.en || n.Local != tc.local || n.Kind != tc.kind {
			t.Errorf("%s: got %q %q %q, want %q %q %q", tc.name, n.En, n.Local, n.Kind, tc.en, tc.local, tc.kind)
		}
		if got := osmfile.Slug(n.En); got != tc.slug {
			t.Errorf("%s: slug %q, want %q", tc.name, got, tc.slug)
		}
	}

	// Every prefecture and every municipality differs from the structural
	// kind, so kind_name is always set.
	if k := osmfile.KindName("Aichi Prefecture", "愛知県", jp.Province, "province"); k != "prefecture" {
		t.Errorf("Aichi: kind_name %q", k)
	}
	if k := osmfile.KindName("Tokyo", "東京都", jp.Province, "province"); k != "metropolis" {
		t.Errorf("Tokyo: kind_name %q", k)
	}
	if k := osmfile.KindName("Shinjuku", "新宿区", jp.District, "district"); k != "special ward" {
		t.Errorf("Shinjuku: kind_name %q", k)
	}

	// Places: English from name:en, a Latin romanisation or a Latin name,
	// never one in Han, kana, Hangul or Cyrillic (Korean and Russian places
	// in the extract); local from a Japanese name, else name:ja.
	places := []struct {
		tags      osm.Tags
		en, local string
	}{
		{jpTags("place", "quarter", "name", "広尾", "name:en", "Hiroo", "name:ja", "広尾"), "Hiroo", "広尾"},
		{jpTags("place", "neighbourhood", "name", "桜堤3", "name:en", "3-chome", "name:ja", "三丁目"), "3-chome", "桜堤3"},
		{jpTags("place", "quarter", "name", "東旭川町日ノ出", "name:ja", "東旭川町日の出", "name:ja-Latn", "Higashiasahikawachō Hinode"), "Higashiasahikawachō Hinode", "東旭川町日ノ出"},
		{jpTags("place", "neighbourhood", "name", "山下", "name:ja_rm", "Yamashita"), "Yamashita", "山下"},
		{jpTags("place", "neighbourhood", "name", "山下"), "", "山下"},
		{jpTags("place", "village", "name", "Niseko Village", "name:ja", "ニセコビレッジ"), "Niseko Village", "ニセコビレッジ"},
		{jpTags("place", "village", "name", "울릉읍"), "", ""},
		{jpTags("place", "town", "name", "Южно-Курильск", "name:ja", "ユジノクリリスク"), "", "ユジノクリリスク"},
	}
	for _, tc := range places {
		en, local := osmfile.PlaceNames(tc.tags, jp)
		if en != tc.en || local != tc.local {
			t.Errorf("place %v: got %q %q, want %q %q", tc.tags, en, local, tc.en, tc.local)
		}
	}
}

// The 47 prefectures by ISO prefix; Busan, South Gyeongsang and Sakhalin in
// the same extract do not; the municipalities are level 7, the wards of the
// designated cities (level 8) are not, and 所属未定地 is excluded.
func TestJapanRelations(t *testing.T) {
	jp := japan(t)
	rel := func(id int64, kv ...string) *osm.Relation {
		return &osm.Relation{ID: osm.RelationID(id), Tags: jpTags(kv...)}
	}
	adm := func(id int64, level, iso string) *osm.Relation {
		if iso == "" {
			return rel(id, "boundary", "administrative", "admin_level", level)
		}
		return rel(id, "boundary", "administrative", "admin_level", level, "ISO3166-2", iso)
	}
	for _, tc := range []struct {
		r    *osm.Relation
		want bool
	}{
		{adm(1543125, "4", "JP-13"), true},
		{adm(3795658, "4", "JP-01"), true},
		{adm(2396450, "4", "KR-26"), false},
		{adm(2393403, "4", "KR-48"), false},
		{adm(394235, "4", "RU-SAK"), false},
		{adm(1758858, "7", ""), false},
	} {
		if got := osmfile.MatchLevel(tc.r, jp.Province); got != tc.want {
			t.Errorf("r%d %v: province %v, want %v", tc.r.ID, tc.r.Tags, got, tc.want)
		}
	}
	for _, tc := range []struct {
		r    *osm.Relation
		want bool
	}{
		{adm(1758858, "7", ""), true},  // 新宿区, a special ward
		{adm(4134838, "7", ""), true},  // 村山市
		{adm(358681, "8", ""), false},  // 北区 of Osaka City, a ward of a designated city
		{adm(2679919, "7", ""), false}, // 所属未定地
	} {
		if got := osmfile.MatchLevel(tc.r, jp.District); got != tc.want {
			t.Errorf("r%d %v: district %v, want %v", tc.r.ID, tc.r.Tags, got, tc.want)
		}
	}
	if !osmfile.Excluded(adm(2679919, "7", ""), jp.District) {
		t.Error("所属未定地 is not reported as excluded")
	}
}
