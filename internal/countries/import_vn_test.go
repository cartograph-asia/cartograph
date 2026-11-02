package countries_test

import (
	"slices"
	"testing"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

func vnTags(kv ...string) osm.Tags {
	var t osm.Tags
	for i := 0; i < len(kv); i += 2 {
		t = append(t, osm.Tag{Key: kv[i], Value: kv[i+1]})
	}
	return t
}

func vietnam(t *testing.T) countries.Import {
	t.Helper()
	vn, ok := countries.ImportOf("vn")
	if !ok || vn.Hold != "" {
		t.Fatalf("Vietnam: settings %v, hold %q", ok, vn.Hold)
	}
	return vn
}

func TestVietnamSettings(t *testing.T) {
	vn := vietnam(t)
	// China's Nansha District is never a Khánh Hòa commune; Trường Sa is not excluded.
	if !slices.Equal(vn.District.Exclude, []int64{6753263}) || len(vn.District.Include) != 0 || len(vn.Province.Exclude) != 0 {
		t.Errorf("Vietnam's Exclude/Include: %+v / %+v", vn.District, vn.Province)
	}
	if vn.Extract != "vietnam" || vn.Province.Count != [2]int{34, 34} || vn.Province.ISOPrefix != "VN" ||
		vn.LocalLang != "vi" || !vn.SlugSuffix || !slices.Equal(vn.Neighbours, []string{"philippines"}) || len(vn.Partial) != 0 {
		t.Errorf("Vietnam's settings: %+v", vn)
	}
}

// Vietnam's settings on tag sets as vietnam-latest.osm.pbf of 2026-11-01
// holds them (docs/countries/vietnam.md): provinces after the 2025 reform,
// communes, wards and special zones, accents folded in the slugs.
func TestVietnamNames(t *testing.T) {
	vn := vietnam(t)
	cases := []struct {
		name            string
		tags            osm.Tags
		level           countries.Level
		en, local, kind string
		slug            string // province: the full slug; district: the name part
	}{
		{"province, name:en with Province", vnTags("name", "Tỉnh Cao Bằng", "name:en", "Cao Bằng Province"),
			vn.Province, "Cao Bằng", "", "province", "cao-bang-vietnam"},
		{"name:vi without prefix", vnTags("name", "Tỉnh Khánh Hòa", "name:en", "Khánh Hòa Province", "name:vi", "Khánh Hòa"),
			vn.Province, "Khánh Hòa", "", "province", "khanh-hoa-vietnam"},
		{"capital: the prefix of name makes the city", vnTags("name", "Thành phố Hà Nội", "name:en", "Hà Nội", "name:vi", "Hà Nội"),
			vn.Province, "Hà Nội", "", "city", "ha-noi-vietnam"},
		{"City stays in the English name", vnTags("name", "Thành phố Hồ Chí Minh", "name:en", "Ho Chi Minh City", "name:vi", "Thành phố Hồ Chí Minh"),
			vn.Province, "Ho Chi Minh City", "Hồ Chí Minh", "city", "ho-chi-minh-city-vietnam"},
		{"English without accents keeps the local name", vnTags("name", "Thành phố Quảng Ninh", "name:en", "Quang Ninh City"),
			vn.Province, "Quang Ninh City", "Quảng Ninh", "city", "quang-ninh-city-vietnam"},
		{"folding: Cần Thơ", vnTags("name", "Thành phố Cần Thơ", "name:en", "Cần Thơ", "name:vi", "Thành phố Cần Thơ"),
			vn.Province, "Cần Thơ", "", "city", "can-tho-vietnam"},
		{"folding: Đồng Tháp", vnTags("name", "Tỉnh Đồng Tháp", "name:en", "Đồng Tháp Province"),
			vn.Province, "Đồng Tháp", "", "province", "dong-thap-vietnam"},
		{"folding: Đắk Lắk", vnTags("name", "Tỉnh Đắk Lắk", "name:en", "Đắk Lắk Province"),
			vn.Province, "Đắk Lắk", "", "province", "dak-lak-vietnam"},

		{"commune with name:en", vnTags("name", "Xã A Dơi", "name:en", "A Dơi Commune", "name:vi", "Xã A Dơi"),
			vn.District, "A Dơi", "", "commune", "a-doi"},
		{"commune with name only", vnTags("name", "Xã Hàm Tân"),
			vn.District, "Hàm Tân", "", "commune", "ham-tan"},
		{"ward", vnTags("name", "Phường Nam Triệu", "name:en", "Nam Triệu Ward"),
			vn.District, "Nam Triệu", "", "ward", "nam-trieu"},
		{"ward with name:en without accents", vnTags("name", "Phường Hoàn Kiếm", "name:en", "Hoan Kiem"),
			vn.District, "Hoan Kiem", "Hoàn Kiếm", "ward", "hoan-kiem"},
		{"commune with name:en without accents or word", vnTags("name", "Xã Hòa Long", "name:en", "Hoa Long"),
			vn.District, "Hoa Long", "Hòa Long", "commune", "hoa-long"},
		{"special zone with name only", vnTags("name", "Đặc khu Phú Quốc"),
			vn.District, "Phú Quốc", "", "special zone", "phu-quoc"},
		{"special zone, Administrative", vnTags("name", "Đặc khu Vân Đồn", "name:en", "Vân Đồn Special Administrative Region"),
			vn.District, "Vân Đồn", "", "special zone", "van-don"},
		{"special zone, Administration", vnTags("name", "Đặc khu Cát Hải", "name:en", "Cát Hải Special Administration Region", "name:vi", "Đặc khu Cát Hải"),
			vn.District, "Cát Hải", "", "special zone", "cat-hai"},
		{"special zone without prefix", vnTags("name", "Cồn Cỏ", "name:en", "Cồn Cỏ Special Administrative Region"),
			vn.District, "Cồn Cỏ", "", "special zone", "con-co"},
	}
	for _, tc := range cases {
		n := osmfile.LevelNames(tc.tags, vn, tc.level)
		if n.En != tc.en || n.Local != tc.local || n.Kind != tc.kind {
			t.Errorf("%s: got %q %q %q, want %q %q %q", tc.name, n.En, n.Local, n.Kind, tc.en, tc.local, tc.kind)
		}
		slug := osmfile.Slug(n.En)
		if tc.level.ISOPrefix != "" {
			slug += "-vietnam"
		}
		if slug != tc.slug {
			t.Errorf("%s: slug %q, want %q", tc.name, slug, tc.slug)
		}
	}

	// Places: Vietnamese names are Latin, so an English name equal to the
	// local one leaves the local NULL; names in a neighbour's script are
	// neither (villages the extract holds beyond the border).
	places := []struct {
		tags      osm.Tags
		en, local string
	}{
		{vnTags("name", "Hà Nội", "name:en", "Hanoi", "name:vi", "Hà Nội"), "Hanoi", "Hà Nội"},
		{vnTags("name", "Thôn 1"), "Thôn 1", ""},
		{vnTags("name", "口岸"), "", ""},
		{vnTags("name", "ບ້ານນາ"), "", ""},
		{vnTags("name", "ភូមិថ្មី"), "", ""},
		{vnTags("name", "东兴市", "name:vi", "Đông Hưng", "name:en", "Dongxing"), "Dongxing", "Đông Hưng"},
	}
	for _, tc := range places {
		en, local := osmfile.PlaceNames(tc.tags, vn)
		if en != tc.en || local != tc.local {
			t.Errorf("place %v: got %q %q, want %q %q", tc.tags, en, local, tc.en, tc.local)
		}
	}
}

// Siblings whose stripped names share a slug keep their words (decision 7
// of docs/COUNTRY_IMPORT_STAGE2_BRIEF.md); names that differ only in their
// tone marks fold to the same slug even then and get the OSM id. Pairs from
// Đồng Tháp and Hải Phòng.
func TestVietnamDuplicateDistricts(t *testing.T) {
	vn := vietnam(t)
	sib := func(id int64, kv ...string) osmfile.Sibling {
		return osmfile.Sibling{OSMID: id, Names: osmfile.LevelNames(vnTags(kv...), vn, vn.District)}
	}
	sibs := []osmfile.Sibling{
		sib(10088783, "name", "Phường Mỹ Tho"),
		sib(15586817, "name", "Xã Mỹ Thọ"),
		sib(15581288, "name", "Xã Tân Thành"),
		sib(15583784, "name", "Xã Tân Thạnh"),
		sib(19326581, "name", "Xã Cẩm Giang", "name:en", "Cẩm Giang Commune"),
		sib(19326582, "name", "Xã Cẩm Giàng", "name:en", "Cẩm Giàng Commune"),
		sib(1, "name", "Xã Hàm Tân"),
	}
	osmfile.ResolveSlugs(sibs, func(base string) string { return "p-" + base })
	want := []struct{ slug, en, local string }{
		{"p-phuong-my-tho", "Phường Mỹ Tho", ""},
		{"p-xa-my-tho", "Xã Mỹ Thọ", ""},
		{"p-xa-tan-thanh-15581288", "Xã Tân Thành", ""},
		{"p-xa-tan-thanh-15583784", "Xã Tân Thạnh", ""},
		{"p-cam-giang-commune-19326581", "Cẩm Giang Commune", "Xã Cẩm Giang"},
		{"p-cam-giang-commune-19326582", "Cẩm Giàng Commune", "Xã Cẩm Giàng"},
		{"p-ham-tan", "Hàm Tân", ""},
	}
	for i, w := range want {
		s := sibs[i]
		if s.Slug != w.slug || s.Names.En != w.en || s.Names.Local != w.local {
			t.Errorf("r%d: got %q %q %q, want %q %q %q", s.OSMID, s.Slug, s.Names.En, s.Names.Local, w.slug, w.en, w.local)
		}
	}
}
