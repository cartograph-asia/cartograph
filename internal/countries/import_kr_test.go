package countries_test

import (
	"testing"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

func southKorea(t *testing.T) countries.Import {
	t.Helper()
	kr, ok := countries.ImportOf("kr")
	if !ok || kr.Hold != "" {
		t.Fatalf("South Korea: settings %v, hold %q", ok, kr.Hold)
	}
	return kr
}

func krTags(kv ...string) osm.Tags {
	var t osm.Tags
	for i := 0; i < len(kv); i += 2 {
		t = append(t, osm.Tag{Key: kv[i], Value: kv[i+1]})
	}
	return t
}

// South Korea's settings on tag sets as the south-korea-latest.osm.pbf of
// 2026-11-01 holds them (docs/countries/south-korea.md).
func TestSouthKoreaNames(t *testing.T) {
	kr := southKorea(t)
	if kr.LocalLang != "ko" || !kr.SlugSuffix || kr.Province.Count != [2]int{16, 16} {
		t.Errorf("South Korea: lang %q, suffix %v, count %v", kr.LocalLang, kr.SlugSuffix, kr.Province.Count)
	}
	cases := []struct {
		name            string
		tags            osm.Tags
		level           countries.Level
		en, local, kind string
		slug            string
	}{
		// Provinces: the designation is the kind word, stripped from the Korean name.
		{"special city", krTags("name", "서울특별시", "name:en", "Seoul", "name:ko", "서울특별시", "name:ko-Hani", "首爾特別市"),
			kr.Province, "Seoul", "서울", "special city", "seoul"},
		{"metropolitan city", krTags("name", "부산광역시", "name:en", "Busan", "name:ko", "부산광역시", "name:ko-Latn", "Busan"),
			kr.Province, "Busan", "부산", "metropolitan city", "busan"},
		{"special self-governing city", krTags("name", "세종특별자치시", "name:en", "Sejong", "name:ko", "세종특별자치시"),
			kr.Province, "Sejong", "세종", "special self-governing city", "sejong"},
		{"province keeps its do", krTags("name", "경기도", "name:en", "Gyeonggi", "name:ko", "경기도"),
			kr.Province, "Gyeonggi", "경기도", "province", "gyeonggi"},
		{"province with a direction", krTags("name", "경상북도", "name:en", "North Gyeongsang", "name:ko", "경상북도"),
			kr.Province, "North Gyeongsang", "경상북도", "province", "north-gyeongsang"},
		{"state", krTags("name", "강원특별자치도", "name:en", "Gangwon State", "name:ko", "강원특별자치도"),
			kr.Province, "Gangwon", "강원", "special self-governing province", "gangwon"},
		{"-do", krTags("name", "전북특별자치도", "name:en", "Jeonbuk-do", "name:ko", "전북특별자치도"),
			kr.Province, "Jeonbuk", "전북", "special self-governing province", "jeonbuk"},
		{"Jeju", krTags("name", "제주특별자치도", "name:en", "Jeju", "name:ko", "제주특별자치도"),
			kr.Province, "Jeju", "제주", "special self-governing province", "jeju"},
		{"Jeonnam-Gwangju, no ISO code", krTags("name", "전남광주통합특별시", "name:en", "Jeonnam-Gwangju Special Metropolitan City", "name:ko", "전남광주통합특별시"),
			kr.Province, "Jeonnam-Gwangju", "전남광주", "special metropolitan city", "jeonnam-gwangju"},
		{"Jeonnam-Gwangju without the Korean name", krTags("name:en", "Jeonnam-Gwangju Special Metropolitan City"),
			kr.Province, "Jeonnam-Gwangju", "", "special metropolitan city", "jeonnam-gwangju"},

		// Districts: -si city, -gun county, -gu district.
		{"si", krTags("name", "안동시", "name:en", "Andong-si", "name:ko", "안동시"),
			kr.District, "Andong", "안동", "city", "andong"},
		{"gun", krTags("name", "철원군", "name:en", "Cheorwon-gun", "name:ko", "철원군", "name:ko-Latn", "Cheorwon"),
			kr.District, "Cheorwon", "철원", "county", "cheorwon"},
		{"gu", krTags("name", "종로구", "name:en", "Jongno-gu", "name:ko", "종로구"),
			kr.District, "Jongno", "종로", "district", "jongno"},
		{"gu that is a direction", krTags("name", "중구", "name:en", "Jung-gu", "name:ko", "중구"),
			kr.District, "Jung", "중", "district", "jung"},
		// KindWord looks at both ends of a name: these start with 구 or 군.
		{"Guri-si starts with gu", krTags("name", "구리시", "name:en", "Guri-si", "name:ko", "구리시"),
			kr.District, "Guri", "구리", "city", "guri"},
		{"Gumi-si", krTags("name", "구미시", "name:en", "Gumi-si", "name:ko", "구미시"),
			kr.District, "Gumi", "구미", "city", "gumi"},
		{"Gunpo-si starts with gun", krTags("name", "군포시", "name:en", "Gunpo-si", "name:ko", "군포시"),
			kr.District, "Gunpo", "군포", "city", "gunpo"},
		{"Gunsan-si", krTags("name", "군산시", "name:en", "Gunsan-si", "name:ko", "군산시"),
			kr.District, "Gunsan", "군산", "city", "gunsan"},
		{"Gurye-gun starts with gu", krTags("name", "구례군", "name:en", "Gurye-gun", "name:ko", "구례군"),
			kr.District, "Gurye", "구례", "county", "gurye"},
		{"Yanggu-gun ends with gu and gun", krTags("name", "양구군", "name:en", "Yanggu-gun", "name:ko", "양구군"),
			kr.District, "Yanggu", "양구", "county", "yanggu"},
		{"Gunwi-gun", krTags("name", "군위군", "name:en", "Gunwi-gun", "name:ko", "군위군"),
			kr.District, "Gunwi", "군위", "county", "gunwi"},
		// Ten names have no English suffix; the Korean one decides the word.
		{"si with a bare English name", krTags("name", "안산시", "name:en", "Ansan", "name:ko", "안산시"),
			kr.District, "Ansan", "안산", "city", "ansan"},
		{"gun with a bare English name", krTags("name", "신안군", "name:en", "Sinan", "name:ko", "신안군"),
			kr.District, "Sinan", "신안", "county", "sinan"},
		{"Jeju city", krTags("name", "제주시", "name:en", "Jeju", "name:ko", "제주시"),
			kr.District, "Jeju", "제주", "city", "jeju"},
		// Only a Hangul name: no English name (Incheon's new Geomdan-gu).
		{"no English name", krTags("name", "검단구", "name:ko", "검단구"),
			kr.District, "", "검단", "district", ""},
	}
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, kr, tc.level)
		if n.En != tc.en || n.Local != tc.local || n.Kind != tc.kind {
			t.Errorf("%s: got %q %q %q, want %q %q %q", tc.name, n.En, n.Local, n.Kind, tc.en, tc.local, tc.kind)
		}
		if got := osmfile.Slug(n.En); got != tc.slug {
			t.Errorf("%s: slug %q, want %q", tc.name, got, tc.slug)
		}
	}
	if got := osmfile.Slug("Seoul") + "-south-korea"; got != "seoul-south-korea" {
		t.Errorf("province slug %q", got)
	}

	// Kind words that equal the structural kind are not stored (kind_name NULL).
	if k := osmfile.KindName("Gyeonggi", "경기도", kr.Province, "province"); k != "" {
		t.Errorf("Gyeonggi: kind_name %q, want none", k)
	}
	if k := osmfile.KindName("Gangnam-gu", "강남구", kr.District, "district"); k != "" {
		t.Errorf("Gangnam-gu: kind_name %q, want none", k)
	}
	if k := osmfile.KindName("Busan", "부산광역시", kr.Province, "province"); k != "metropolitan city" {
		t.Errorf("Busan: kind_name %q", k)
	}
	if k := osmfile.KindName("Andong-si", "안동시", kr.District, "district"); k != "city" {
		t.Errorf("Andong-si: kind_name %q", k)
	}

	// Places: English from name:en or a Latin name, never one in Hangul, Han,
	// kana or Cyrillic (villages in North Korea, China, Japan and Russia
	// beyond the border); local from name:ko, else a Hangul name.
	places := []struct {
		tags      osm.Tags
		en, local string
	}{
		{krTags("name", "연남동", "name:en", "Yeonnam-dong", "name:ko", "연남동", "name:ko-Latn", "Yeonnam-dong"), "Yeonnam-dong", "연남동"},
		{krTags("name", "연남동", "name:ko", "연남동", "name:ko-Latn", "Yeonnam-dong"), "Yeonnam-dong", "연남동"},
		{krTags("name", "연남동"), "", "연남동"},
		{krTags("name", "Itaewon-dong"), "Itaewon-dong", ""},
		{krTags("name", "平壌", "name:ko", "평양"), "", "평양"},
		{krTags("name", "対馬"), "", ""},
		{krTags("name", "Владивосток"), "", ""},
	}
	for _, tc := range places {
		en, local := osmfile.PlaceNames(tc.tags, kr)
		if en != tc.en || local != tc.local {
			t.Errorf("place %v: got %q %q, want %q %q", tc.tags, en, local, tc.en, tc.local)
		}
	}
}

// The sixteenth level-4 relation has no ISO code and comes in through
// Include; North Korea's level-4 relations in the same extract do not.
func TestSouthKoreaProvinceRelations(t *testing.T) {
	kr := southKorea(t)
	rel := func(id int64, kv ...string) *osm.Relation {
		return &osm.Relation{ID: osm.RelationID(id), Tags: krTags(kv...)}
	}
	level4 := func(id int64, iso string) *osm.Relation {
		if iso == "" {
			return rel(id, "boundary", "administrative", "admin_level", "4")
		}
		return rel(id, "boundary", "administrative", "admin_level", "4", "ISO3166-2", iso)
	}
	for _, tc := range []struct {
		r    *osm.Relation
		want bool
	}{
		{level4(2297418, "KR-11"), true},
		{level4(2355168, "KR-45"), true},
		{level4(21039457, ""), true},
		{level4(356533, "KP-05"), false},
		{level4(5455268, "KP-15"), false},
		{level4(999, ""), false},
		{rel(2419946, "boundary", "administrative", "admin_level", "6"), false},
	} {
		if got := osmfile.MatchLevel(tc.r, kr.Province); got != tc.want {
			t.Errorf("r%d %v: province %v, want %v", tc.r.ID, tc.r.Tags, got, tc.want)
		}
	}
	if !osmfile.MatchLevel(rel(2419946, "boundary", "administrative", "admin_level", "6"), kr.District) {
		t.Error("a level-6 relation is not a district")
	}
}
