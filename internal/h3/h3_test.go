package h3

import (
	"testing"

	h3go "github.com/uber/h3-go/v4"
)

// A point in Pattaya, taken from testdata/IMG_1911.jpeg.
const testLat, testLng = 12.8517888888889, 100.899880555556

func TestCellFromLatLng(t *testing.T) {
	cell, err := CellFromLatLng(testLat, testLng, PlotRes)
	if err != nil {
		t.Fatal(err)
	}
	if len(cell) != 15 || cell[:2] != "88" {
		t.Fatalf("unexpected res-8 cell %q", cell)
	}
	got, res, err := Parse(cell)
	if err != nil || got != cell || res != PlotRes {
		t.Fatalf("Parse(%q) = %q, %d, %v", cell, got, res, err)
	}
	// A nearby point (~10 m away) lands in the same plot.
	other, _ := CellFromLatLng(testLat+0.00005, testLng+0.00005, PlotRes)
	if other != cell {
		t.Fatalf("nearby point in different cell: %s vs %s", other, cell)
	}
}

func TestParse(t *testing.T) {
	cell, _ := CellFromLatLng(testLat, testLng, PlotRes)
	upper := ""
	for _, r := range cell {
		if r >= 'a' && r <= 'f' {
			r -= 'a' - 'A'
		}
		upper += string(r)
	}
	if got, _, err := Parse(upper); err != nil || got != cell {
		t.Fatalf("Parse(upper) = %q, %v", got, err)
	}
	for _, bad := range []string{"", "88", "zzzzzzzzzzzzzzz", "0x" + cell[2:], "800000000000000", cell + "0"} {
		if _, _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", bad)
		}
	}
}

func TestParent(t *testing.T) {
	cell, _ := CellFromLatLng(testLat, testLng, PlotRes)
	for _, res := range []int{5, 6} {
		p, err := Parent(cell, res)
		if err != nil {
			t.Fatal(err)
		}
		want, _ := CellFromLatLng(testLat, testLng, res)
		if p != want {
			t.Errorf("Parent(res %d) = %s, want %s", res, p, want)
		}
		if _, r, _ := Parse(p); r != res {
			t.Errorf("parent resolution = %d, want %d", r, res)
		}
	}
	if _, err := Parent(cell, 9); err == nil {
		t.Error("Parent to finer resolution succeeded, want error")
	}
	if _, err := Parent("nonsense", 5); err == nil {
		t.Error("Parent of invalid cell succeeded, want error")
	}
}

func TestChildrenCount(t *testing.T) {
	p5, _ := CellFromLatLng(testLat, testLng, 5)
	p6, _ := CellFromLatLng(testLat, testLng, 6)
	if n, err := ChildrenCount(p5, PlotRes); err != nil || n != 343 {
		t.Errorf("ChildrenCount(res5) = %d, %v; want 343", n, err)
	}
	if n, err := ChildrenCount(p6, PlotRes); err != nil || n != 49 {
		t.Errorf("ChildrenCount(res6) = %d, %v; want 49", n, err)
	}
	// Pentagons: compare against h3-go's own enumeration.
	pents, err := h3go.Pentagons(5)
	if err != nil {
		t.Fatal(err)
	}
	children, err := pents[0].Children(PlotRes)
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := ChildrenCount(pents[0].String(), PlotRes); n != int64(len(children)) {
		t.Errorf("pentagon ChildrenCount = %d, want %d", n, len(children))
	}
	if _, err := ChildrenCount(p6, 5); err == nil {
		t.Error("ChildrenCount to coarser resolution succeeded, want error")
	}
}

func TestBoundary(t *testing.T) {
	b, err := Boundary("88658bd11dfffff")
	if err != nil || len(b) != 6 {
		t.Fatalf("boundary: %v %v", b, err)
	}
	for _, v := range b {
		if v.Lat < 12.9 || v.Lat > 12.95 || v.Lng < 100.86 || v.Lng > 100.91 {
			t.Fatalf("vertex %+v far from Pattaya", v)
		}
	}
	if _, err := Boundary("nope"); err == nil {
		t.Fatal("want error for invalid cell")
	}
}
