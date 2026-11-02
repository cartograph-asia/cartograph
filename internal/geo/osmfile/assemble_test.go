package osmfile

import (
	"reflect"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/osm"
)

// synth builds small in-memory OSM data on a grid: the node at (x, y) has
// the id x*100+y+1 and lies at lon 100+x/100, lat 10+y/100.
type synth struct{ o *osm.OSM }

type pt [2]int

func newSynth() *synth { return &synth{o: &osm.OSM{}} }

func nodeID(p pt) osm.NodeID { return osm.NodeID(p[0]*100 + p[1] + 1) }

func gridPoint(x, y float64) orb.Point { return orb.Point{100 + x/100, 10 + y/100} }

// way adds a way through ps (and the nodes it needs) and returns its id.
func (s *synth) way(id osm.WayID, ps ...pt) osm.WayID {
	w := &osm.Way{ID: id}
	have := map[osm.NodeID]bool{}
	for _, n := range s.o.Nodes {
		have[n.ID] = true
	}
	for _, p := range ps {
		nid := nodeID(p)
		w.Nodes = append(w.Nodes, osm.WayNode{ID: nid})
		if !have[nid] {
			have[nid] = true
			g := gridPoint(float64(p[0]), float64(p[1]))
			s.o.Nodes = append(s.o.Nodes, &osm.Node{ID: nid, Lon: g[0], Lat: g[1], Visible: true, Version: 1})
		}
	}
	s.o.Ways = append(s.o.Ways, w)
	return id
}

// member is a way member of a relation; the way need not exist.
type member struct {
	way  osm.WayID
	role string
}

func (s *synth) relation(id osm.RelationID, ms ...member) {
	r := &osm.Relation{ID: id, Tags: osm.Tags{
		{Key: "type", Value: "boundary"},
		{Key: "boundary", Value: "administrative"},
		{Key: "admin_level", Value: "4"},
		{Key: "name", Value: "Test"},
	}}
	for _, m := range ms {
		r.Members = append(r.Members, osm.Member{Type: osm.TypeWay, Ref: int64(m.way), Role: m.role})
	}
	s.o.Relations = append(s.o.Relations, r)
}

func outer(ids ...osm.WayID) []member {
	var ms []member
	for _, id := range ids {
		ms = append(ms, member{id, "outer"})
	}
	return ms
}

// square adds the four sides of the square (x0,y0)-(x1,y1) as ways with ids
// base+1..base+4, counter-clockwise from the bottom side.
func (s *synth) square(base osm.WayID, x0, y0, x1, y1 int) []osm.WayID {
	return []osm.WayID{
		s.way(base+1, pt{x0, y0}, pt{x1, y0}),
		s.way(base+2, pt{x1, y0}, pt{x1, y1}),
		s.way(base+3, pt{x1, y1}, pt{x0, y1}),
		s.way(base+4, pt{x0, y1}, pt{x0, y0}),
	}
}

// closed adds the square (x0,y0)-(x1,y1) as one closed way.
func (s *synth) closed(id osm.WayID, x0, y0, x1, y1 int) osm.WayID {
	return s.way(id, pt{x0, y0}, pt{x1, y0}, pt{x1, y1}, pt{x0, y1}, pt{x0, y0})
}

func mustAssemble(t *testing.T, o *osm.OSM, partial map[int64]bool) *Assembled {
	t.Helper()
	a, err := Assemble(o, partial)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func builtIDs(a *Assembled) []int64 {
	var ids []int64
	for _, b := range a.Built {
		ids = append(ids, b.OSMID)
	}
	return ids
}

func TestAssembleWholeSquare(t *testing.T) {
	s := newSynth()
	s.relation(1, outer(s.square(10, 0, 0, 10, 10)...)...)
	a := mustAssemble(t, s.o, nil)
	if len(a.Built) != 1 || a.Built[0].OSMID != 1 || len(a.NotWhole)+len(a.NotClosed)+len(a.Partial) != 0 {
		t.Fatalf("%+v", a)
	}
	g := a.Built[0].Geometry
	if len(g) != 1 || len(g[0]) != 1 || !planar.MultiPolygonContains(g, gridPoint(5, 5)) || planar.MultiPolygonContains(g, gridPoint(15, 5)) {
		t.Fatalf("geometry %v", g)
	}
	if a.Built[0].Tags.Find("name") != "Test" {
		t.Fatal("tags lost")
	}
}

// threeRings is a relation 7 with two closed squares and a third square of
// three ways whose fourth (way 34) is missing from the data.
func threeRings() *osm.OSM {
	s := newSynth()
	ms := outer(s.square(10, 0, 0, 10, 10)...)
	ms = append(ms, outer(s.closed(20, 20, 0, 30, 10))...)
	c := s.square(30, 40, 0, 50, 10)
	s.o.Ways = s.o.Ways[:len(s.o.Ways)-1] // drop way 34
	ms = append(ms, outer(c...)...)
	s.relation(7, ms...)
	return s.o
}

func TestAssembleMissingWay(t *testing.T) {
	a := mustAssemble(t, threeRings(), nil)
	if len(a.Built) != 0 || !reflect.DeepEqual(a.NotWhole, []int64{7}) || len(a.NotClosed) != 0 || len(a.Partial) != 0 {
		t.Fatalf("%+v", a)
	}
	// Another relation's partial flag changes nothing.
	a = mustAssemble(t, threeRings(), map[int64]bool{8: true})
	if !reflect.DeepEqual(a.NotWhole, []int64{7}) {
		t.Fatalf("%+v", a)
	}
}

func TestAssemblePartial(t *testing.T) {
	a := mustAssemble(t, threeRings(), map[int64]bool{7: true})
	if len(a.NotWhole)+len(a.NotClosed) != 0 || !reflect.DeepEqual(builtIDs(a), []int64{7}) {
		t.Fatalf("%+v", a)
	}
	want := PartialInfo{RingsBuilt: 2, FragmentsDropped: 1, MissingWays: 1}
	if got := a.Partial[7]; got != want {
		t.Fatalf("partial %+v, want %+v", got, want)
	}
	g := a.Built[0].Geometry
	if len(g) != 2 {
		t.Fatalf("polygons %d", len(g))
	}
	for _, p := range []orb.Point{gridPoint(5, 5), gridPoint(25, 5)} {
		if !planar.MultiPolygonContains(g, p) {
			t.Fatalf("%v not in %v", p, g)
		}
	}
	if planar.MultiPolygonContains(g, gridPoint(45, 5)) {
		t.Fatal("the unclosed square was built")
	}
	for _, poly := range g {
		if poly[0].Orientation() != orb.CCW {
			t.Fatal("outer ring not counter-clockwise")
		}
	}
}

func TestAssembleNotClosed(t *testing.T) {
	s := newSynth()
	// r1: one open line; r2: a closed square and an open line, whole.
	s.relation(1, outer(s.way(10, pt{0, 0}, pt{10, 0}, pt{10, 10}))...)
	ms := outer(s.closed(20, 20, 0, 30, 10))
	ms = append(ms, outer(s.way(21, pt{40, 0}, pt{50, 0}, pt{50, 10}))...)
	s.relation(2, ms...)

	a := mustAssemble(t, s.o, nil)
	// r2 is whole and keeps BuildBoundaries' rule: built from its closed ring.
	if !reflect.DeepEqual(a.NotClosed, []int64{1}) || !reflect.DeepEqual(builtIDs(a), []int64{2}) || len(a.NotWhole)+len(a.Partial) != 0 {
		t.Fatalf("%+v", a)
	}

	a = mustAssemble(t, s.o, map[int64]bool{1: true, 2: true})
	if !reflect.DeepEqual(a.NotClosed, []int64{1}) || !reflect.DeepEqual(builtIDs(a), []int64{2}) {
		t.Fatalf("%+v", a)
	}
	if got, want := a.Partial[2], (PartialInfo{RingsBuilt: 1, FragmentsDropped: 1}); got != want {
		t.Fatalf("partial %+v, want %+v", got, want)
	}
	if _, ok := a.Partial[1]; ok {
		t.Fatal("a relation that was not built is in Partial")
	}
}

func TestAssembleInnerRings(t *testing.T) {
	s := newSynth()
	ms := outer(s.square(10, 0, 0, 30, 30)...) // big square
	ms = append(ms, outer(s.closed(20, 40, 0, 50, 10))...)
	ms = append(ms, outer(s.closed(21, 10, 10, 20, 20))...) // island in the lake
	ms = append(ms,
		member{s.closed(30, 2, 2, 28, 28), "inner"},   // lake in the big square
		member{s.closed(31, 12, 12, 18, 18), "inner"}, // pond on the island
		member{s.closed(32, 42, 2, 48, 8), "inner"},   // hole in the small square
		member{s.closed(33, 60, 0, 70, 10), "inner"},  // outside every outer ring
	)
	s.relation(5, ms...)

	check := func(a *Assembled) {
		t.Helper()
		if !reflect.DeepEqual(builtIDs(a), []int64{5}) {
			t.Fatalf("%+v", a)
		}
		g := a.Built[0].Geometry
		if len(g) != 3 {
			t.Fatalf("polygons %d: %v", len(g), g)
		}
		in := map[orb.Point]bool{
			gridPoint(1, 1):     true,  // big square, outside the lake
			gridPoint(5, 5):     false, // lake
			gridPoint(11, 11):   true,  // island
			gridPoint(15, 15):   false, // pond
			gridPoint(41, 1):    true,  // small square
			gridPoint(45, 5):    false, // its hole
			gridPoint(65, 5):    false, // the stray inner ring
			gridPoint(35, 5):    false,
			gridPoint(29.5, 15): true,
		}
		for p, want := range in {
			if planar.MultiPolygonContains(g, p) != want {
				t.Errorf("%v inside: want %v", p, want)
			}
		}
		for _, poly := range g {
			for i, r := range poly {
				want := orb.CW
				if i == 0 {
					want = orb.CCW
				}
				if r.Orientation() != want {
					t.Errorf("ring %d orientation %v", i, r.Orientation())
				}
			}
		}
	}
	whole := mustAssemble(t, s.o, nil)
	check(whole)
	own := mustAssemble(t, s.o, map[int64]bool{5: true})
	check(own)
	// Nothing missing, but the stray inner ring is left out.
	if got, want := own.Partial[5], (PartialInfo{RingsBuilt: 6, FragmentsDropped: 1}); got != want {
		t.Fatalf("partial %+v, want %+v", got, want)
	}
}

func TestAssemblePartialWholeIsNotPartial(t *testing.T) {
	s := newSynth()
	s.relation(1, outer(s.square(10, 0, 0, 10, 10)...)...)
	a := mustAssemble(t, s.o, map[int64]bool{1: true})
	if !reflect.DeepEqual(builtIDs(a), []int64{1}) || len(a.Partial) != 0 {
		t.Fatalf("%+v", a)
	}
}

func TestAssembleEmpty(t *testing.T) {
	for _, o := range []*osm.OSM{nil, {}} {
		a := mustAssemble(t, o, nil)
		if len(a.Built)+len(a.NotWhole)+len(a.NotClosed) != 0 || a.Partial == nil || len(a.Partial) != 0 {
			t.Fatalf("%+v", a)
		}
	}
}

func TestAssembleDeterministic(t *testing.T) {
	build := func(reverse bool) *osm.OSM {
		s := newSynth()
		s.relation(9, outer(s.square(10, 0, 0, 10, 10)...)...)
		s.relation(3, outer(s.closed(20, 20, 0, 30, 10))...)
		s.relation(6, outer(s.way(30, pt{40, 0}, pt{50, 0}))...)
		s.relation(4, member{99, "outer"})
		s.relation(2, outer(s.closed(40, 60, 0, 70, 10), 98)...)
		s.relation(8, member{97, "outer"}, member{96, "outer"})
		if reverse {
			rs := s.o.Relations
			for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
				rs[i], rs[j] = rs[j], rs[i]
			}
			ws := s.o.Ways
			for i, j := 0, len(ws)-1; i < j; i, j = i+1, j-1 {
				ws[i], ws[j] = ws[j], ws[i]
			}
		}
		return s.o
	}
	partial := map[int64]bool{2: true, 8: true}
	a := mustAssemble(t, build(false), partial)
	if !reflect.DeepEqual(builtIDs(a), []int64{2, 3, 9}) || !reflect.DeepEqual(a.NotWhole, []int64{4}) || !reflect.DeepEqual(a.NotClosed, []int64{6, 8}) {
		t.Fatalf("%+v", a)
	}
	if got, want := a.Partial[2], (PartialInfo{RingsBuilt: 1, MissingWays: 1}); got != want || len(a.Partial) != 1 {
		t.Fatalf("partial %+v", a.Partial)
	}
	for i := 0; i < 5; i++ {
		b := mustAssemble(t, build(i%2 == 1), partial)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("run %d differs:\n%+v\n%+v", i, a, b)
		}
	}
}
