package geo

import (
	"bytes"
	"math"
	"path/filepath"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/simplify"
)

var fixtureWater = filepath.Join("..", "..", "testdata", "pattaya-water.shp")

func TestMercatorRoundTrip(t *testing.T) {
	p := orb.Point{100.8826, 12.9346}
	q := MercatorToLngLat(LngLatToMercator(p))
	if math.Abs(p[0]-q[0]) > 1e-9 || math.Abs(p[1]-q[1]) > 1e-9 {
		t.Fatalf("%v -> %v", p, q)
	}
}

func TestShpRoundTrip(t *testing.T) {
	square := func(x, y, s float64) []orb.Point {
		return []orb.Point{{x, y}, {x, y + s}, {x + s, y + s}, {x + s, y}, {x, y}}
	}
	recs := []ShpRecord{
		{Box: [4]float64{0, 0, 10, 10}, Rings: [][]orb.Point{square(0, 0, 10), square(4, 4, 2)}},
		{Box: [4]float64{20, 20, 21, 21}, Rings: [][]orb.Point{square(20, 20, 1)}},
	}
	var buf bytes.Buffer
	if err := WriteShp(&buf, recs); err != nil {
		t.Fatal(err)
	}
	var got []ShpRecord
	if err := ReadShp(bytes.NewReader(buf.Bytes()), [4]float64{-1, -1, 11, 11}, func(r ShpRecord) error {
		got = append(got, r)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Rings) != 2 || got[0].Rings[1][2] != (orb.Point{6, 6}) {
		t.Fatalf("got %+v", got)
	}
}

func TestWaterFixture(t *testing.T) {
	w, err := ReadWater(fixtureWater, orb.Bound{Min: orb.Point{100.82, 12.82}, Max: orb.Point{100.98, 13.02}})
	if err != nil {
		t.Fatal(err)
	}
	if w.Len() == 0 {
		t.Fatal("no water polygons")
	}
	sea := []orb.Point{{100.85, 12.93}, {100.83, 12.85}}
	land := []orb.Point{{100.89, 12.93}, {100.95, 12.95}, {100.89, 12.88}}
	for _, p := range sea {
		if !w.Contains(p) {
			t.Errorf("%v should be water", p)
		}
	}
	for _, p := range land {
		if w.Contains(p) {
			t.Errorf("%v should be land", p)
		}
	}
}

func TestRegionIndex(t *testing.T) {
	sq := func(x, y, s float64) orb.MultiPolygon {
		return orb.MultiPolygon{{{{x, y}, {x + s, y}, {x + s, y + s}, {x, y + s}, {x, y}}}}
	}
	idx := NewRegionIndex([]RegionShape{{ID: 7, Geometry: sq(0, 0, 1)}, {ID: 9, Geometry: sq(1.001, 0, 1)}})
	cases := []struct {
		p      orb.Point
		id     int
		ok     bool
		inside bool
	}{
		{orb.Point{0.5, 0.5}, 7, true, true},
		{orb.Point{1.5, 0.5}, 9, true, true},
		{orb.Point{1.0004, 0.5}, 7, true, false}, // sliver between neighbours: nearest
		{orb.Point{1.0008, 0.5}, 9, true, false},
		{orb.Point{-0.005, 0.5}, 7, true, false}, // just off the coast
		{orb.Point{-0.5, 0.5}, 0, false, false},
	}
	for _, c := range cases {
		id, ok := idx.Locate(c.p)
		_, in := idx.Contains(c.p)
		if id != c.id || ok != c.ok || in != c.inside {
			t.Errorf("%v: got %d %v %v", c.p, id, ok, in)
		}
	}
	var none *RegionIndex
	if _, ok := none.Locate(orb.Point{}); ok || none.Len() != 0 {
		t.Fatal("nil index")
	}
	if _, _, ok := none.Nearest(orb.Point{}, 1, nil); ok {
		t.Fatal("nil index: Nearest")
	}
}

func TestRegionIndexContainsOfAndNearest(t *testing.T) {
	sq := func(x, y, s float64) orb.MultiPolygon {
		return orb.MultiPolygon{{{{x, y}, {x + s, y}, {x + s, y + s}, {x, y + s}, {x, y}}}}
	}
	// 3 and 4 overlap in 10.05–10.1 E; 5 lies far north, at 60° N.
	idx := NewRegionIndex([]RegionShape{{ID: 3, Geometry: sq(10, 0, 0.1)}, {ID: 4, Geometry: sq(10.05, 0, 0.1)}, {ID: 5, Geometry: sq(10, 60, 0.1)}})
	if id, ok := idx.ContainsOf(orb.Point{10.07, 0.05}, nil); !ok || id != 3 {
		t.Errorf("ContainsOf all: %d %v, want the first, 3", id, ok)
	}
	if id, ok := idx.ContainsOf(orb.Point{10.07, 0.05}, func(id int) bool { return id == 4 }); !ok || id != 4 {
		t.Errorf("ContainsOf 4 only: %d %v", id, ok)
	}
	if _, ok := idx.ContainsOf(orb.Point{10.02, 0.05}, func(id int) bool { return id == 4 }); ok {
		t.Error("ContainsOf 4 only outside 4")
	}

	near := func(id int) bool { return id != 4 }
	cases := []struct {
		p    orb.Point
		km   float64
		want func(int) bool
		id   int
		dist float64 // km, ±1 %
		ok   bool
	}{
		{orb.Point{10.05, 0.05}, 2, nil, 3, 0, true},                  // inside
		{orb.Point{10.05, 0.11}, 2, nil, 3, 1.112, true},              // 0.01° north of 3 and 4: the first
		{orb.Point{10.05, 0.11}, 1, nil, 0, 0, false},                 // beyond reach
		{orb.Point{10.16, 0.05}, 2, near, 3, 6.672, false},            // 4 is 1.1 km away but not wanted, 3 is 6.7 km
		{orb.Point{10.16, 0.05}, 2, nil, 4, 1.112, true},              // east of 4
		{orb.Point{10.12, 60.05}, 2, nil, 5, 1.112, true},             // 0.02° east at 60° N is 1.1 km
		{orb.Point{10.11, 0.11}, 2, nil, 4, 1.112, true},              // nearest is 4's edge, not 3's corner
		{orb.Point{9.99, -0.01}, 2, nil, 3, 1.112 * math.Sqrt2, true}, // a corner
	}
	for _, c := range cases {
		id, dist, ok := idx.Nearest(c.p, c.km, c.want)
		if ok != c.ok || (ok && (id != c.id || math.Abs(dist-c.dist) > 0.01*c.dist+1e-9)) {
			t.Errorf("Nearest(%v, %g): %d %.3f %v, want %d %.3f %v", c.p, c.km, id, dist, ok, c.id, c.dist, c.ok)
		}
	}
}

func TestGeometryRoundTrip(t *testing.T) {
	mp := orb.MultiPolygon{{{{0, 0}, {1, 0}, {1, 1}, {0, 0}}}}
	s, err := EncodeGeometry(mp)
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodeGeometry(s)
	if err != nil || len(back) != 1 || len(back[0][0]) != 4 {
		t.Fatalf("%v %v", back, err)
	}
	if _, err := DecodeGeometry(`{"type":"Point","coordinates":[1,2]}`); err == nil {
		t.Fatal("point accepted")
	}
}

func TestParseBound(t *testing.T) {
	b, err := ParseBound("100.82,12.82,100.98,13.02")
	if err != nil || b.Min[0] != 100.82 || b.Max[1] != 13.02 {
		t.Fatalf("%v %v", b, err)
	}
	if b, err := ParseBound(""); err != nil || !NoBound(b) {
		t.Fatalf("empty: %v %v", b, err)
	}
	for _, bad := range []string{"1,2,3", "3,3,1,1", "a,b,c,d"} {
		if _, err := ParseBound(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// Simplify gives the coordinates the simplifier gives, in rings that do not
// keep the unsimplified arrays alive, and drops collapsed rings.
func TestSimplifyCompactRingsSameCoordinates(t *testing.T) {
	var outer orb.Ring
	for i := 0; i <= 1000; i++ {
		outer = append(outer, orb.Point{float64(i) / 1000, 0.00001 * float64(i%2)})
	}
	outer = append(outer, orb.Point{1, 1}, orb.Point{0, 1}, outer[0])
	hole := orb.Ring{{0.4, 0.4}, {0.40001, 0.4}, {0.4, 0.40001}, {0.4, 0.4}} // collapses at 0.0005
	mp := orb.MultiPolygon{{outer, hole}, {{{5, 5}, {5, 5}, {5, 5}, {5, 5}}}}
	want := simplify.DouglasPeucker(0.0005).MultiPolygon(mp.Clone())
	got := Simplify(mp, 0.0005)
	if len(got) != 1 || len(got[0]) != 1 {
		t.Fatalf("polygons %d, rings %v", len(got), len(got[0]))
	}
	if !got[0][0].Equal(want[0][0]) {
		t.Fatalf("coordinates differ: %d vs %d points", len(got[0][0]), len(want[0][0]))
	}
	if cap(got[0][0]) != len(got[0][0]) {
		t.Errorf("ring keeps spare capacity: %d of %d", len(got[0][0]), cap(got[0][0]))
	}
}
