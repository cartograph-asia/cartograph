package countryscan

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
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

// The accent line compares the folding slug the import uses with the slug
// before stage 2.
func TestAccentSlugs(t *testing.T) {
	for in, want := range map[string][2]string{
		"Hà Nội":                {"h-n-i", "ha-noi"},
		"Thành phố Hồ Chí Minh": {"th-nh-ph-h-ch-minh", "thanh-pho-ho-chi-minh"},
		"Đà Nẵng":               {"n-ng", "da-nang"},
		"Chon Buri":             {"chon-buri", "chon-buri"},
		"ชลบุรี":                {"", ""},
	} {
		if got := [2]string{plainSlug(in), osmfile.Slug(in)}; got != want {
			t.Errorf("slugs of %q = %q, want %q", in, got, want)
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

func TestMdText(t *testing.T) {
	for in, want := range map[string]string{
		"Chon Buri":             "Chon Buri",
		"a|b":                   `a\|b`,
		"<b>x</b>":              "&lt;b&gt;x&lt;/b&gt;",
		"[x](http://e.example)": `\[x\](http://e.example)`,
		"a`b":                   "a’b",
		"a\nb\tc\r\nd":          "a b c d",
		"snake_case *star*":     "snake_case *star*",
		"ชลบุรี · Hà Nội":       "ชลบุรี · Hà Nội",
	} {
		if got := mdText(in); got != want {
			t.Errorf("mdText(%q) = %q, want %q", in, got, want)
		}
	}
	if got := quoted([]string{"a`b", `c"d`}); got != `[]string{"a’b", "c\"d"}` {
		t.Errorf("quoted = %s", got)
	}
}

// unescapedPipes counts the table separators of a line.
func unescapedPipes(line string) int {
	n := 0
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '|':
			n++
		}
	}
	return n
}

func TestReportEscapesOSMText(t *testing.T) {
	r := scanFixture(t)
	const evil = "x|y <img src=a onerror=b> [c](http://e.example) `d`\ne"
	for _, l := range r.Levels {
		l.Examples = append(l.Examples, evil)
		l.SkippedList = append(l.SkippedList, "r1 "+evil)
		l.Prefixes[evil] = 1
		l.NameKeys[evil] = 1
	}
	for _, n := range r.Names {
		n.AccentEx = append(n.AccentEx, evil)
		n.Duplicates = append(n.Duplicates, evil)
		n.EnglishKeys[evil] = 1
		n.Prefixes = append(n.Prefixes, Affix{Text: evil, Count: n.N})
		n.LocalSuf = append(n.LocalSuf, Affix{Text: evil, Count: n.N})
		n.AccentLoss = 1
	}
	r.Places.Kinds = append(r.Places.Kinds, KindStat{Kind: evil, Total: 1})
	r.Places.Cities = append(r.Places.Cities, CityStat{Name: evil, Population: 1, Kinds: []KindStat{{Kind: evil, Total: 1}}})
	md := r.Report()

	// The fenced Go sketch shows text literally, so look at the prose only.
	prose, sketch, _ := strings.Cut(md, "```go")
	if strings.Contains(prose, "<img") {
		t.Fatalf("raw HTML in the report:\n%s", prose)
	}
	if strings.Contains(prose, "[c]") {
		t.Fatal("unescaped link text in the report")
	}
	for _, line := range strings.Split(prose, "\n") {
		if strings.Contains(line, "&lt;img") && strings.HasPrefix(line, "| ") {
			// the place table has four columns, five separators
			if got := unescapedPipes(line); got != 5 {
				t.Errorf("table row has %d separators: %s", got, line)
			}
		}
		if strings.Contains(line, "<") && !strings.HasPrefix(line, "```") {
			t.Errorf("raw < in line: %s", line)
		}
		if strings.Contains(line, "x|y") {
			t.Errorf("unescaped pipe in line: %s", line)
		}
		// A code span must not be opened or closed by OSM text: backticks come in pairs.
		if !strings.HasPrefix(line, "```") && strings.Count(line, "`")%2 != 0 {
			t.Errorf("odd number of backticks: %s", line)
		}
	}
	if !strings.Contains(prose, `x\|y &lt;img src=a onerror=b&gt; \[c\](http://e.example) ’d’ e`) {
		t.Fatalf("escaped text not found:\n%s", prose)
	}
	// The Go sketch: the Quote of an affix carries no backtick.
	if strings.Count(sketch, "```") != 1 || strings.Contains(sketch, "`d`") {
		t.Fatalf("code block broken:\n%s", sketch)
	}
}

// A scan with neighbours' cuts reads their ways and nodes only: with the
// fixture as its own neighbour every figure stays, and the report differs
// only by the sentence naming the neighbours.
func TestNeighboursOnlyNamedInReport(t *testing.T) {
	plain := scanFixture(t).Report()
	c, _ := countries.ByCode("th")
	r, err := Scan(context.Background(), Options{Country: c, PBF: fixture, Neighbours: []string{fixture}, Water: fixtureWater, Workers: 4})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.Neighbours, []string{"pattaya.osm.gz"}) {
		t.Errorf("neighbours %v", r.Neighbours)
	}
	sentence := "Boundaries completed with the ways and nodes of the neighbours' cuts `pattaya.osm.gz` (relations and places from the own file only). "
	with := r.Report()
	if !strings.Contains(with, sentence) || strings.Replace(with, sentence, "", 1) != plain {
		t.Errorf("report with neighbours is not the plain report plus one sentence:\n%s", with)
	}
	if strings.Contains(plain, "neighbours") {
		t.Error("the plain report mentions neighbours")
	}
}
