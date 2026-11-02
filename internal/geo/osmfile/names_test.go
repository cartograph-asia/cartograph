package osmfile

import (
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"unicode"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
)

// The legacy functions are the Thai rules as they were before the settings
// existed, copied verbatim; Thailand's settings must give the same results.

func legacyNames(t osm.Tags) (en, th string) {
	for _, k := range []string{"name:en", "int_name", "name:th-Latn", "name"} {
		if v := strings.TrimSpace(t.Find(k)); v != "" && !legacyHasThai(v) {
			en = v
			break
		}
	}
	th = strings.TrimSpace(t.Find("name:th"))
	if th == "" && legacyHasThai(t.Find("name")) {
		th = strings.TrimSpace(t.Find("name"))
	}
	return en, th
}

func legacyHasThai(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Thai, r) {
			return true
		}
	}
	return false
}

func legacyRegionName(en, th string) (string, string) {
	en = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(en), " Province"))
	th = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(th), "จังหวัด"))
	return en, th
}

func legacyDistrictName(en, th string) (string, string) {
	en = strings.TrimSpace(en)
	en = strings.TrimSpace(strings.TrimSuffix(en, " District"))
	for _, p := range []string{"Khet ", "Amphoe "} {
		en = strings.TrimSpace(strings.TrimPrefix(en, p))
	}
	th = strings.TrimSpace(th)
	for _, p := range []string{"เขต", "อำเภอ"} {
		th = strings.TrimSpace(strings.TrimPrefix(th, p))
	}
	return en, th
}

func legacyIsProvince(r *osm.Relation, country string) bool {
	t := r.Tags
	return t.Find("boundary") == "administrative" && t.Find("admin_level") == "4" &&
		strings.HasPrefix(strings.ToUpper(t.Find("ISO3166-2")), strings.ToUpper(country)+"-")
}

func legacyIsDistrict(r *osm.Relation) bool {
	t := r.Tags
	return t.Find("boundary") == "administrative" && t.Find("admin_level") == "6"
}

func legacySlug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		default:
			dash = true
		}
	}
	return b.String()
}

func tags(kv ...string) osm.Tags {
	var t osm.Tags
	for i := 0; i+1 < len(kv); i += 2 {
		t = append(t, osm.Tag{Key: kv[i], Value: kv[i+1]})
	}
	return t
}

func TestThailandSettingsReproduceOldNames(t *testing.T) {
	th := countries.Imports["th"]
	for _, tg := range []osm.Tags{
		tags("name", "ชลบุรี"),
		tags("name", "จังหวัดชลบุรี", "name:en", "Chon Buri Province", "name:th", "จังหวัดชลบุรี"),
		tags("name", "จังหวัดชลบุรี", "name:en", "ชลบุรี"),
		tags("name", "จังหวัดชลบุรี", "int_name", "Chon Buri"),
		tags("name", "Pattaya", "name:th-Latn", "Phatthaya"),
		tags("name", "Pattaya"),
		tags("name", " Chon Buri Province ", "name:th", " จังหวัดชลบุรี "),
		tags("name", "เขตบางกะปิ", "name:en", "Bang Kapi District"),
		tags("name", "เขตวัฒนา", "name:en", "Khet Watthana"),
		tags("name", "อำเภอบ้านโคก", "name:en", "Amphoe Ban Khok"),
		tags("name", "อำเภอเมืองร้อยเอ็ด", "name:en", "Mueang Roi Et District"),
		tags("name", "Bangkok กรุงเทพ"),
		tags("name", "อำเภอ", "name:en", " District"),
		tags("name:th", "Pattaya", "name:en", "Pattaya"),
		tags("name:en", "", "name", "  "),
		tags(),
	} {
		wantEn, wantTh := legacyNames(tg)
		en, local := NamesFor(tg, th)
		if en != wantEn || local != wantTh {
			t.Errorf("NamesFor(%v) = %q, %q, want %q, %q", tg, en, local, wantEn, wantTh)
		}
		// Places: names as they are, a Latin name:th equal to the English
		// one included (LatinLocal is false for Thai).
		if en, local := PlaceNames(tg, th); en != wantEn || local != wantTh {
			t.Errorf("PlaceNames(%v) = %q, %q, want %q, %q", tg, en, local, wantEn, wantTh)
		}
		pe, pl := legacyRegionName(wantEn, wantTh)
		if e, l := StripName(en, local, th.Province); e != pe || l != pl {
			t.Errorf("province StripName(%q, %q) = %q, %q, want %q, %q", en, local, e, l, pe, pl)
		}
		if n := LevelNames(tg, th, th.Province); n.En != pe || n.Local != pl {
			t.Errorf("province LevelNames(%v) = %q, %q, want %q, %q", tg, n.En, n.Local, pe, pl)
		}
		de, dl := legacyDistrictName(wantEn, wantTh)
		if e, l := StripName(en, local, th.District); e != de || l != dl {
			t.Errorf("district StripName(%q, %q) = %q, %q, want %q, %q", en, local, e, l, de, dl)
		}
		if n := LevelNames(tg, th, th.District); n.En != de || n.Local != dl {
			t.Errorf("district LevelNames(%v) = %q, %q, want %q, %q", tg, n.En, n.Local, de, dl)
		}
		// Thailand's kind words follow from regions.kind, so kind_name stays NULL.
		if k := KindName(en, local, th.Province, "province"); k != "" {
			t.Errorf("province kind name %q", k)
		}
		if k := KindName(en, local, th.District, "district"); k != "" {
			t.Errorf("district kind name %q", k)
		}
	}
}

func TestThailandSettingsReproduceOldLevels(t *testing.T) {
	th := countries.Imports["th"]
	for _, tg := range []osm.Tags{
		tags("boundary", "administrative", "admin_level", "4", "ISO3166-2", "TH-20"),
		tags("boundary", "administrative", "admin_level", "4", "ISO3166-2", "th-20"),
		tags("boundary", "administrative", "admin_level", "4", "ISO3166-2", "TH20"),
		tags("boundary", "administrative", "admin_level", "4", "ISO3166-2", "LA-01"),
		tags("boundary", "administrative", "admin_level", "4"),
		tags("boundary", "administrative", "admin_level", "04", "ISO3166-2", "TH-20"),
		tags("boundary", "administrative", "admin_level", "6"),
		tags("boundary", "administrative", "admin_level", "6", "ISO3166-2", "TH-2001"),
		tags("boundary", "political", "admin_level", "6"),
		tags("boundary", "administrative", "admin_level", "8"),
		tags("admin_level", "4", "ISO3166-2", "TH-20"),
	} {
		r := &osm.Relation{ID: 1, Tags: tg}
		if got, want := MatchLevel(r, th.Province), legacyIsProvince(r, "th"); got != want {
			t.Errorf("MatchLevel(%v, province) = %v, want %v", tg, got, want)
		}
		if got, want := MatchLevel(r, th.District), legacyIsDistrict(r); got != want {
			t.Errorf("MatchLevel(%v, district) = %v, want %v", tg, got, want)
		}
	}
	if got := PlaceKindsFor(th); !reflect.DeepEqual(got, PlaceKinds) {
		t.Errorf("Thailand's place kinds %v", got)
	}
}

func TestMatchLevel(t *testing.T) {
	admin := func(level, iso string) *osm.Relation {
		tg := tags("boundary", "administrative", "admin_level", level)
		if iso != "" {
			tg = append(tg, osm.Tag{Key: "ISO3166-2", Value: iso})
		}
		return &osm.Relation{ID: 7, Tags: tg}
	}
	taiwan := countries.Level{AdminLevels: []int{7, 8}}
	for level, want := range map[string]bool{"6": false, "7": true, "8": true, "9": false} {
		if got := MatchLevel(admin(level, ""), taiwan); got != want {
			t.Errorf("Taiwan districts, level %s: %v", level, got)
		}
	}
	// Singapore: no ISO code; the parent relation is the caller's geometric check.
	if !MatchLevel(admin("5", ""), countries.Level{AdminLevels: []int{5}, Parent: 536780}) {
		t.Error("a level without ISO prefix needs none")
	}
	// Philippines: a relation named in Include counts without the ISO code.
	ph := countries.Level{AdminLevels: []int{4}, ISOPrefix: "PH", Include: []int64{7}}
	if !MatchLevel(&osm.Relation{ID: 7, Tags: tags("boundary", "administrative", "admin_level", "3")}, ph) {
		t.Error("Include did not count")
	}
	if MatchLevel(&osm.Relation{ID: 8, Tags: tags("boundary", "administrative", "admin_level", "4")}, ph) {
		t.Error("a relation without the ISO code counted")
	}
	// Vietnam: a relation named in Exclude never counts, and Excluded tells it.
	vn := countries.Level{AdminLevels: []int{6}, Exclude: []int64{7}}
	if MatchLevel(admin("6", ""), vn) || !Excluded(admin("6", ""), vn) {
		t.Error("Exclude did not exclude")
	}
	if Excluded(admin("5", ""), vn) || Excluded(&osm.Relation{ID: 8, Tags: tags("boundary", "administrative", "admin_level", "6")}, vn) {
		t.Error("Excluded counted a relation outside the rule or not listed")
	}
	// Brunei: no district level.
	if MatchLevel(admin("6", ""), countries.Level{}) {
		t.Error("the zero level matched")
	}
}

// vietnam, japan and indonesia are made-up settings for the tests; the real
// ones come with each country's PR.
var vietnam = countries.Import{
	Province: countries.Level{AdminLevels: []int{4}, ISOPrefix: "VN", Kind: "province",
		KindByAffix: map[string]string{"Thành phố": "city"},
		Strip:       countries.Affixes{EnSuffix: []string{" Province", " City"}, LocalPrefix: []string{"Tỉnh ", "Thành phố "}}},
	District: countries.Level{AdminLevels: []int{6}, Kind: "commune",
		KindByAffix: map[string]string{"Phường": "ward", "Ward": "ward"},
		Strip: countries.Affixes{EnPrefix: []string{"Xã ", "Phường "}, EnSuffix: []string{" Commune", " Ward"},
			LocalPrefix: []string{"Xã ", "Phường "}}},
	English: []string{"name:en", "int_name", "name"}, EnglishNot: []string{"Han"},
	Local: []string{"name:vi", "name"}, LocalScripts: []string{"Latin"},
	SlugSuffix: true,
}

var japan = countries.Import{
	Province: countries.Level{AdminLevels: []int{4}, ISOPrefix: "JP", Kind: "prefecture",
		Strip: countries.Affixes{EnSuffix: []string{" Prefecture"}}},
	District: countries.Level{AdminLevels: []int{7}, Kind: "municipality",
		KindByAffix: map[string]string{"市": "city", "町": "town", "村": "village"}},
	English: []string{"name:en", "name:ja-Latn", "name:ja_rm", "name"}, EnglishNot: []string{"Han", "Hiragana", "Katakana"},
	Local: []string{"name:ja", "name"}, LocalScripts: []string{"Han", "Hiragana", "Katakana"},
	SlugSuffix: true,
}

func TestVietnameseNames(t *testing.T) {
	n := LevelNames(tags("name", "Tỉnh Cao Bằng", "name:en", "Cao Bằng Province"), vietnam, vietnam.Province)
	if n.En != "Cao Bằng" || n.Local != "" || n.Kind != "province" {
		t.Fatalf("Cao Bằng: %+v", n)
	}
	if s := Slug(n.En) + "-vietnam"; s != "cao-bang-vietnam" {
		t.Errorf("slug %q", s)
	}
	n = LevelNames(tags("name", "Thành phố Cần Thơ", "name:en", "Can Tho City"), vietnam, vietnam.Province)
	if n.En != "Can Tho" || n.Local != "Cần Thơ" || n.Kind != "city" {
		t.Errorf("Cần Thơ: %+v", n)
	}
	// A commune that has only name: the English name is the Vietnamese one,
	// so the second name is NULL.
	n = LevelNames(tags("name", "Xã Tân Thành"), vietnam, vietnam.District)
	if n.En != "Tân Thành" || n.Local != "" || n.RawEn != "Xã Tân Thành" || n.Kind != "commune" {
		t.Errorf("commune: %+v", n)
	}
	n = LevelNames(tags("name", "Phường Tân Thành"), vietnam, vietnam.District)
	if n.En != "Tân Thành" || n.Kind != "ward" {
		t.Errorf("ward: %+v", n)
	}
	// A name in Han script is no English name, and not a Vietnamese one.
	if en, local := NamesFor(tags("name", "河内"), vietnam); en != "" || local != "" {
		t.Errorf("Han name: %q %q", en, local)
	}
}

func TestJapaneseNames(t *testing.T) {
	n := LevelNames(tags("name", "京丹後市", "name:en", "Kyōtango", "name:ja", "京丹後市"), japan, japan.District)
	if n.En != "Kyōtango" || n.Local != "京丹後市" || n.Kind != "city" || Slug(n.En) != "kyotango" {
		t.Fatalf("Kyōtango: %+v", n)
	}
	// LocalLang is empty, so the language is the country's.
	if c, _ := countries.ByCode("jp"); japan.LocalLang != "" || c.Lang != "ja" {
		t.Errorf("language %q / %q", japan.LocalLang, c.Lang)
	}
	// A Latin name is no Japanese name.
	if en, local := NamesFor(tags("name", "Kyoto"), japan); en != "Kyoto" || local != "" {
		t.Errorf("Latin name: %q %q", en, local)
	}
	// Kana counts as Japanese, and a name in it is no English name.
	if en, local := NamesFor(tags("name", "つくば市"), japan); en != "" || local != "つくば市" {
		t.Errorf("kana name: %q %q", en, local)
	}
}

func TestSlugFolds(t *testing.T) {
	for in, want := range map[string]string{
		"Cần Thơ":           "can-tho",
		"Đồng Tháp":         "dong-thap",
		"Ürümqi":            "urumqi",
		"Higashiōsaka":      "higashiosaka",
		"Ōmihachiman":       "omihachiman",
		"Hà Nội":            "ha-noi",
		"Liquiçá":           "liquica",
		"P'yŏngyang":        "p-yongyang",
		"Chon Buri":         "chon-buri",
		" Bueng  Kan ":      "bueng-kan",
		"Ko-Chang!":         "ko-chang",
		"ชลบุรี":            "",
		"Phường Tân Thành":  "phuong-tan-thanh",
		"Thành phố Hà Nội ": "thanh-pho-ha-noi",
	} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

// thaiProvinces are the English names of Thailand's 77 provinces.
var thaiProvinces = []string{
	"Amnat Charoen", "Ang Thong", "Bangkok", "Bueng Kan", "Buriram", "Chachoengsao", "Chai Nat",
	"Chaiyaphum", "Chanthaburi", "Chiang Mai", "Chiang Rai", "Chon Buri", "Chumphon", "Kalasin",
	"Kamphaeng Phet", "Kanchanaburi", "Khon Kaen", "Krabi", "Lampang", "Lamphun", "Loei", "Lopburi",
	"Mae Hong Son", "Maha Sarakham", "Mukdahan", "Nakhon Nayok", "Nakhon Pathom", "Nakhon Phanom",
	"Nakhon Ratchasima", "Nakhon Sawan", "Nakhon Si Thammarat", "Nan", "Narathiwat", "Nong Bua Lamphu",
	"Nong Khai", "Nonthaburi", "Pathum Thani", "Pattani", "Phang Nga", "Phatthalung", "Phayao",
	"Phetchabun", "Phetchaburi", "Phichit", "Phitsanulok", "Phra Nakhon Si Ayutthaya", "Phrae", "Phuket",
	"Prachinburi", "Prachuap Khiri Khan", "Ranong", "Ratchaburi", "Rayong", "Roi Et", "Sa Kaeo",
	"Sakon Nakhon", "Samut Prakan", "Samut Sakhon", "Samut Songkhram", "Saraburi", "Satun", "Sing Buri",
	"Sisaket", "Songkhla", "Sukhothai", "Suphan Buri", "Surat Thani", "Surin", "Tak", "Trang", "Trat",
	"Ubon Ratchathani", "Udon Thani", "Uthai Thani", "Uttaradit", "Yala", "Yasothon",
}

// Folding leaves ASCII alone, so the slugs Thailand already stores stay.
func TestSlugKeepsASCII(t *testing.T) {
	if len(thaiProvinces) != 77 {
		t.Fatalf("%d provinces", len(thaiProvinces))
	}
	for _, p := range thaiProvinces {
		for _, name := range []string{p, p + " Province"} {
			if got, want := Slug(name), legacySlug(name); got != want {
				t.Errorf("Slug(%q) = %q, was %q", name, got, want)
			}
		}
	}
	ascii := func(b []byte) bool {
		s := make([]byte, len(b))
		for i, c := range b {
			s[i] = c & 0x7f
		}
		return Slug(string(s)) == legacySlug(string(s))
	}
	if err := quick.Check(ascii, nil); err != nil {
		t.Error(err)
	}
}

func TestKindWord(t *testing.T) {
	taiwan := countries.Level{Kind: "district", KindByAffix: map[string]string{"市": "city", "縣": "county"}}
	malaysia := countries.Level{Kind: "state", KindByAffix: map[string]string{"Federal Territory": "federal territory", "Territory": "territory"}}
	for _, c := range []struct {
		en, local string
		l         countries.Level
		want      string
	}{
		{"Hsinchu", "新竹市", taiwan, "city"},
		{"Hsinchu County", "新竹縣", taiwan, "county"},
		{"Keelung", "", taiwan, "district"},
		{"", "市川市", taiwan, "city"},
		{"Federal Territory of Labuan", "", malaysia, "federal territory"}, // the longer affix first
		{"Northern Territory", "", malaysia, "territory"},
		{"Territorye", "", malaysia, "state"}, // no word boundary
		{"Selangor", "", malaysia, "state"},
		{"Statesville", "", countries.Level{Kind: "region", KindByAffix: map[string]string{"State": "state"}}, "region"},
		{"Kachin State", "", countries.Level{Kind: "region", KindByAffix: map[string]string{"State": "state"}}, "state"},
		{"수원시", "", countries.Level{Kind: "district", KindByAffix: map[string]string{"시": "city"}}, "city"},
	} {
		if got := KindWord(c.en, c.local, c.l); got != c.want {
			t.Errorf("KindWord(%q, %q) = %q, want %q", c.en, c.local, got, c.want)
		}
	}
	if k := KindName("Tokyo", "東京都", japan.Province, "province"); k != "prefecture" {
		t.Errorf("KindName = %q", k)
	}
	if k := KindName("Kachin State", "", countries.Level{Kind: "province", KindByAffix: map[string]string{"State": "province"}}, "province"); k != "" {
		t.Errorf("KindName equal to kind = %q", k)
	}
}

func sibling(id int64, imp countries.Import, l countries.Level, kv ...string) Sibling {
	return Sibling{OSMID: id, Names: LevelNames(tags(kv...), imp, l)}
}

func TestResolveSlugs(t *testing.T) {
	district := func(base string) string { return "ho-chi-minh-vietnam-" + base }

	// Siblings that collide only after stripping keep their affixes.
	sibs := []Sibling{
		sibling(1, vietnam, vietnam.District, "name", "Xã Tân Thành"),
		sibling(2, vietnam, vietnam.District, "name", "Phường Tân Thành"),
		sibling(3, vietnam, vietnam.District, "name", "Xã An Phú"),
	}
	ResolveSlugs(sibs, district)
	for i, want := range []struct{ slug, en string }{
		{"ho-chi-minh-vietnam-xa-tan-thanh", "Xã Tân Thành"},
		{"ho-chi-minh-vietnam-phuong-tan-thanh", "Phường Tân Thành"},
		{"ho-chi-minh-vietnam-an-phu", "An Phú"},
	} {
		if sibs[i].Slug != want.slug || sibs[i].Names.En != want.en || sibs[i].Names.Local != "" {
			t.Errorf("sibling %d: %q %+v, want %q %q", i, sibs[i].Slug, sibs[i].Names, want.slug, want.en)
		}
	}

	// Siblings that still collide get their OSM id; the local names keep
	// their affixes too. The id goes before a province's country suffix.
	indonesia := countries.Import{English: []string{"name:en", "name"}, Local: []string{"name:id", "name"}}
	regency := countries.Level{AdminLevels: []int{5}, Strip: countries.Affixes{LocalPrefix: []string{"Kota ", "Kabupaten "}}}
	sibs = []Sibling{
		sibling(5449460, indonesia, regency, "name:en", "Bogor", "name:id", "Kota Bogor"),
		sibling(5449461, indonesia, regency, "name:en", "Bogor", "name:id", "Kabupaten Bogor"),
	}
	ResolveSlugs(sibs, func(base string) string { return base + "-indonesia" })
	if sibs[0].Slug != "bogor-5449460-indonesia" || sibs[1].Slug != "bogor-5449461-indonesia" {
		t.Errorf("Bogor: %q %q", sibs[0].Slug, sibs[1].Slug)
	}
	if sibs[0].Names.Local != "Kota Bogor" || sibs[1].Names.Local != "Kabupaten Bogor" {
		t.Errorf("Bogor names: %+v %+v", sibs[0].Names, sibs[1].Names)
	}

	// A stored slug never changes: not when the relation is renamed, and not
	// when a new sibling collides with it; the new one gives way.
	sibs = []Sibling{
		sibling(1, vietnam, vietnam.District, "name", "Xã Tân Hòa"),
		sibling(2, vietnam, vietnam.District, "name", "Phường Tân Thành"),
		sibling(3, vietnam, vietnam.District, "name", "Xã Tân Thành"),
	}
	sibs[0].Stored = "ho-chi-minh-vietnam-an-phu"
	sibs[1].Stored = "ho-chi-minh-vietnam-tan-thanh"
	ResolveSlugs(sibs, district)
	for i, want := range []string{"ho-chi-minh-vietnam-an-phu", "ho-chi-minh-vietnam-tan-thanh", "ho-chi-minh-vietnam-xa-tan-thanh"} {
		if sibs[i].Slug != want {
			t.Errorf("sibling %d: %q, want %q", i, sibs[i].Slug, want)
		}
	}

	// A new sibling that collides with a stored one keeps its affix; the
	// stored one keeps its slug.
	sibs = []Sibling{
		sibling(1, vietnam, vietnam.District, "name", "Xã Tân Thành"),
		sibling(2, vietnam, vietnam.District, "name", "Xã Tân Thành"),
	}
	sibs[0].Stored = "ho-chi-minh-vietnam-tan-thanh"
	ResolveSlugs(sibs, district)
	if sibs[0].Slug != "ho-chi-minh-vietnam-tan-thanh" || sibs[1].Slug != "ho-chi-minh-vietnam-xa-tan-thanh" {
		t.Errorf("stored and new: %q %q", sibs[0].Slug, sibs[1].Slug)
	}
}

func TestPlaceKindsFor(t *testing.T) {
	got := PlaceKindsFor(countries.Import{PlaceKinds: []string{"city", "town", "village", "suburb", "quarter", "neighbourhood", "hamlet"}})
	if len(got) != 7 || !got["hamlet"] || PlaceKinds["hamlet"] {
		t.Errorf("place kinds %v, default %v", got, PlaceKinds)
	}
}
