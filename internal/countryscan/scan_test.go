package countryscan

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/countries"
)

var (
	fixture      = filepath.Join("..", "..", "testdata", "pattaya.osm.gz")
	fixtureWater = filepath.Join("..", "..", "testdata", "pattaya-water.shp")
)

func scanFixture(t *testing.T) *Result {
	t.Helper()
	c, _ := countries.ByCode("th")
	r, err := Scan(context.Background(), Options{Country: c, PBF: fixture, Water: fixtureWater, Workers: 4})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestFixtureLevels(t *testing.T) {
	r := scanFixture(t)
	t.Log("\n" + r.Report())
	if r.Province.Level != 4 {
		t.Fatalf("province level %d: %s", r.Province.Level, r.Province.Reason)
	}
	l4 := r.levelOf(4)
	if l4 == nil || l4.N != 1 || l4.ISOHere != 1 || l4.Prefixes["TH"] != 1 || l4.English != 1 {
		t.Fatalf("level 4: %+v", l4)
	}
	if l6 := r.levelOf(6); l6 == nil || l6.N != 3 || l6.Geometry == false {
		t.Fatalf("level 6: %+v", l6)
	}
	// The fixture holds three of Chon Buri's districts: they do not tile the land.
	if r.District.Level != 0 {
		t.Fatalf("district level %d: %s", r.District.Level, r.District.Reason)
	}
	if len(r.Names) != 1 || r.Names[0].N != 1 || r.Names[0].NoEnglish != 0 {
		t.Fatalf("names %+v", r.Names)
	}
	// 7 place nodes lie inside the outline: village, neighbourhood, city, town.
	if r.Places.Nodes != 7 {
		t.Fatalf("places %+v", r.Places)
	}
}

func TestReportIsDeterministic(t *testing.T) {
	a, b := scanFixture(t).Report(), scanFixture(t).Report()
	if a != b {
		t.Fatal("two scans of the same file give different reports")
	}
}

func TestFoldSlug(t *testing.T) {
	for in, want := range map[string]string{
		"Hà Nội":                "ha-noi",
		"Thành phố Hồ Chí Minh": "thanh-pho-ho-chi-minh",
		"Đà Nẵng":               "da-nang",
		"Chon Buri":             "chon-buri",
		"ชลบุรี":                "",
	} {
		if got := foldSlug(in); got != want {
			t.Errorf("foldSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScript(t *testing.T) {
	for in, want := range map[string]string{
		"Hà Nội": scriptLatin, "ชลบุรี": scriptThai, "東京都": scriptHan, "東京とうきょう": scriptKana,
		"서울특별시": scriptHangul, "Улаанбаатар": scriptCyrillic, "12": "",
	} {
		if got := scriptOf(in); got != want {
			t.Errorf("scriptOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAffixes(t *testing.T) {
	names := []string{"Tỉnh A B", "Tỉnh C D", "Tỉnh E F", "Tỉnh G", "Hà Nam"}
	pre, _ := affixes(names)
	if len(pre) == 0 || pre[0].Text != "Tỉnh" || pre[0].Count != 4 {
		t.Fatalf("prefixes %+v", pre)
	}
	_, suf := affixes([]string{"江苏省", "浙江省", "广东省", "山东省", "北京市"})
	if len(suf) == 0 || suf[0].Text != "省" || suf[0].Count != 4 {
		t.Fatalf("suffixes %+v", suf)
	}
}

func TestCellsRoundTrip(t *testing.T) {
	name := filepath.Join(t.TempDir(), "x.land8")
	in := []uint64{1, 5, 1 << 60}
	if err := WriteCells(name, in); err != nil {
		t.Fatal(err)
	}
	out, err := ReadCells(name)
	if err != nil || len(out) != 3 || out[2] != 1<<60 {
		t.Fatalf("%v %v", out, err)
	}
	if !strings.HasSuffix(CellsFile("d", "TH"), "th.land8") {
		t.Fatal(CellsFile("d", "TH"))
	}
}

func TestOverlaps(t *testing.T) {
	cache := t.TempDir()
	id := func(lat, lng float64) uint64 {
		c, err := cellID(lat, lng, 8)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	a, b, c := id(14.0, 102.5), id(14.0, 102.6), id(14.0, 102.7)
	sorted := func(x ...uint64) []uint64 { slices.Sort(x); return x }
	if err := WriteCells(CellsFile(cache, "th"), sorted(a, b)); err != nil {
		t.Fatal(err)
	}
	if err := WriteCells(CellsFile(cache, "la"), sorted(b, c)); err != nil {
		t.Fatal(err)
	}
	md, err := Overlaps(cache)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "| Laos / Thailand | 1 |") || !strings.Contains(md, "Not scanned yet") || strings.Contains(md, "Cambodia /") {
		t.Fatalf("overlaps.md:\n%s", md)
	}
	if again, _ := Overlaps(cache); again != md {
		t.Fatal("overlaps.md is not deterministic")
	}
}
