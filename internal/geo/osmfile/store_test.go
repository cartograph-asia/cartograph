package osmfile

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmgeojson"
)

// geojsonBoundaries is BuildBoundaries as it was before the compact store:
// osmgeojson.Convert on the whole data. polygon must build exactly what it
// built.
func geojsonBoundaries(t *testing.T, o *osm.OSM) (out []Boundary, skipped []int64) {
	t.Helper()
	fc, err := osmgeojson.Convert(o, osmgeojson.NoMeta(true), osmgeojson.NoRelationMembership(true))
	if err != nil {
		t.Fatal(err)
	}
	geom := map[int64]orb.MultiPolygon{}
	for _, f := range fc.Features {
		id, ok := f.ID.(string)
		if !ok || !strings.HasPrefix(id, "relation/") {
			continue
		}
		var rid int64
		fmt.Sscanf(id, "relation/%d", &rid)
		switch g := f.Geometry.(type) {
		case orb.Polygon:
			geom[rid] = orb.MultiPolygon{g}
		case orb.MultiPolygon:
			geom[rid] = g
		}
	}
	for _, r := range o.Relations {
		g, ok := geom[int64(r.ID)]
		if !ok || len(g) == 0 {
			skipped = append(skipped, int64(r.ID))
			continue
		}
		out = append(out, Boundary{OSMID: int64(r.ID), Tags: r.Tags, Geometry: g})
	}
	return out, skipped
}

func sameAsGeoJSON(t *testing.T, o *osm.OSM) {
	t.Helper()
	wantB, wantS := geojsonBoundaries(t, o)
	gotB, gotS, err := BuildBoundaries(o)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotB, wantB) || !reflect.DeepEqual(gotS, wantS) {
		t.Fatalf("BuildBoundaries differs from osmgeojson:\n got %v skipped %v\nwant %v skipped %v", gotB, gotS, wantB, wantS)
	}
	// Assemble builds whole relations the same way.
	a := mustAssemble(t, o, nil)
	byID := map[int64]Boundary{}
	for _, b := range wantB {
		byID[b.OSMID] = b
	}
	for _, b := range a.Built {
		if !reflect.DeepEqual(b, byID[b.OSMID]) {
			t.Fatalf("Assemble differs from osmgeojson for r%d", b.OSMID)
		}
	}
}

func TestPolygonCases(t *testing.T) {
	cases := map[string]func(s *synth){
		"one closed outer way": func(s *synth) {
			s.relation(1, outer(s.closed(10, 0, 0, 10, 10))...)
		},
		"one outer way, inner rings open and closed": func(s *synth) {
			ms := outer(s.closed(10, 0, 0, 30, 30))
			ms = append(ms, member{s.closed(11, 2, 2, 8, 8), "inner"},
				member{s.way(12, pt{12, 12}, pt{18, 12}, pt{18, 18}), "inner"},
				member{s.closed(13, 50, 50, 60, 60), "inner"}) // outside: kept on one outer way
			s.relation(1, ms...)
		},
		"one outer way, only uninteresting tags": func(s *synth) {
			s.relation(1, outer(s.closed(10, 0, 0, 10, 10))...)
			s.o.Relations[0].Tags = osm.Tags{{Key: "type", Value: "multipolygon"}, {Key: "source", Value: "survey"}}
		},
		"one outer way present, one missing": func(s *synth) {
			ms := outer(s.closed(10, 0, 0, 10, 10), 99)
			ms = append(ms, member{s.closed(11, 20, 20, 30, 30), "inner"}) // outside the outer ring
			s.relation(1, ms...)
		},
		"one open outer way": func(s *synth) {
			s.relation(1, outer(s.way(10, pt{0, 0}, pt{10, 0}, pt{10, 10}))...)
		},
		"squares split, reversed and shuffled": func(s *synth) {
			sq := s.square(10, 0, 0, 10, 10)
			wn := s.o.Ways[1].Nodes
			wn[0], wn[1] = wn[1], wn[0] // way 12 reversed
			ms := outer(sq[2], sq[0], sq[3], sq[1])
			ms = append(ms, outer(s.square(20, 20, 0, 30, 10)...)...)
			ms = append(ms, member{s.closed(30, 22, 2, 28, 8), "inner"}, member{s.closed(31, 2, 2, 4, 4), "inner"})
			s.relation(1, ms...)
		},
		"route, untyped and multipolygon": func(s *synth) {
			s.relation(1, outer(s.closed(10, 0, 0, 10, 10))...)
			s.relation(2, outer(10)...)
			s.relation(3, outer(10)...)
			s.o.Relations[0].Tags[0].Value = "route"
			s.o.Relations[1].Tags = s.o.Relations[1].Tags[1:]
			s.o.Relations[2].Tags[0].Value = "multipolygon"
		},
		"member orientations": func(s *synth) {
			ms := outer(s.square(10, 0, 0, 10, 10)...)
			ms = append(ms, member{s.closed(20, 2, 2, 8, 8), "inner"})
			s.relation(1, ms...)
			s.relation(2, member{s.closed(30, 20, 0, 30, 10), "outer"})
			m := s.o.Relations[0].Members
			m[0].Orientation, m[2].Orientation, m[4].Orientation = orb.CW, orb.CCW, orb.CCW
			s.o.Relations[1].Members[0].Orientation = orb.CW
		},
		"a way listed twice, a one-node way, a missing node": func(s *synth) {
			sq := s.square(10, 0, 0, 10, 10)
			ms := outer(sq...)
			ms = append(ms, outer(sq[0], s.way(20, pt{40, 40}))...)
			s.relation(1, ms...)
			s.o.Nodes = s.o.Nodes[1:] // node (0, 0)
		},
		"positions on the way and on the member": func(s *synth) {
			sq := s.square(10, 0, 0, 10, 10)
			s.relation(1, outer(sq...)...)
			for i := range s.o.Ways[0].Nodes {
				wn := &s.o.Ways[0].Nodes[i]
				wn.Lon, wn.Lat = 100.5, 10.5 // differs from the node
			}
			s.relation(2, outer(77)...)
			s.o.Relations[1].Members[0].Nodes = osm.WayNodes{
				{ID: nodeID(pt{0, 0})}, {ID: nodeID(pt{10, 0}), Lon: 101, Lat: 11}, {ID: nodeID(pt{10, 10})}, {ID: nodeID(pt{0, 0})},
			}
		},
		"unsorted and repeated nodes and ways": func(s *synth) {
			s.relation(1, outer(s.square(10, 0, 0, 10, 10)...)...)
			n := *s.o.Nodes[0]
			n.Lon += 0.5
			s.o.Nodes = append(s.o.Nodes, &n) // the last copy counts
			s.o.Nodes[0], s.o.Nodes[2] = s.o.Nodes[2], s.o.Nodes[0]
			w := *s.o.Ways[0]
			w.Nodes = osm.WayNodes{{ID: nodeID(pt{0, 0})}, {ID: nodeID(pt{5, -5})}, {ID: nodeID(pt{10, 0})}}
			s.way(99, pt{5, -5})
			s.o.Ways = append(s.o.Ways, &w)
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			s := newSynth()
			build(s)
			sameAsGeoJSON(t, s.o)
		})
	}
}

// TestPolygonRandom compares polygon with osmgeojson on random relations:
// rectangles (some touching at a corner) cut into ways at random, ways
// reversed, members shuffled, repeated, missing or given an orientation,
// nodes missing.
func TestPolygonRandom(t *testing.T) {
	rnd := rand.New(rand.NewSource(3))
	for run := 0; run < 300; run++ {
		s := newSynth()
		wayID := osm.WayID(1)
		orient := map[osm.WayID]orb.Orientation{}
		ring := func(x0, y0, x1, y1 int, role string) []member {
			var ps []pt
			for x := x0; x < x1; x++ {
				ps = append(ps, pt{x, y0})
			}
			for y := y0; y < y1; y++ {
				ps = append(ps, pt{x1, y})
			}
			for x := x1; x > x0; x-- {
				ps = append(ps, pt{x, y1})
			}
			for y := y1; y > y0; y-- {
				ps = append(ps, pt{x0, y})
			}
			ps = append(ps, ps[0])
			var ms []member
			for start := 0; start < len(ps)-1; {
				end := start + 1 + rnd.Intn(len(ps)-1-start)
				piece := append([]pt(nil), ps[start:end+1]...)
				if rnd.Intn(2) == 0 {
					for i, j := 0, len(piece)-1; i < j; i, j = i+1, j-1 {
						piece[i], piece[j] = piece[j], piece[i]
					}
				}
				m := member{s.way(wayID, piece...), role}
				if rnd.Intn(8) == 0 {
					orient[wayID] = orb.Orientation(1 - 2*rnd.Intn(2))
				}
				wayID++
				ms = append(ms, m)
				start = end
			}
			if rnd.Intn(6) == 0 { // a piece missing
				i := rnd.Intn(len(ms))
				ms = append(ms[:i], ms[i+1:]...)
			}
			return ms
		}
		for rel := 1; rel <= 4; rel++ {
			var ms []member
			x, y, w, h := 0, 0, 0, 0
			for k := rnd.Intn(3) + 1; k > 0; k-- {
				if rnd.Intn(2) == 0 { // touching the last one at a corner
					x, y = x+w, y+h
				} else {
					x, y = rnd.Intn(40), rnd.Intn(40)
				}
				w, h = 2+rnd.Intn(12), 2+rnd.Intn(12)
				ms = append(ms, ring(x, y, x+w, y+h, "outer")...)
				if rnd.Intn(2) == 0 && w > 2 && h > 2 {
					ms = append(ms, ring(x+1, y+1, x+w-1, y+h-1, "inner")...)
				}
			}
			if rnd.Intn(5) == 0 {
				ms = append(ms, ms[rnd.Intn(len(ms))]) // repeated member
			}
			if rnd.Intn(5) == 0 {
				ms = append(ms, member{9000 + wayID, "outer"}) // absent way
			}
			rnd.Shuffle(len(ms), func(i, j int) { ms[i], ms[j] = ms[j], ms[i] })
			s.relation(osm.RelationID(rel), ms...)
		}
		for _, r := range s.o.Relations {
			for i := range r.Members {
				r.Members[i].Orientation = orient[osm.WayID(r.Members[i].Ref)]
			}
		}
		if rnd.Intn(4) == 0 && len(s.o.Nodes) > 0 {
			i := rnd.Intn(len(s.o.Nodes))
			s.o.Nodes = append(s.o.Nodes[:i], s.o.Nodes[i+1:]...)
		}
		rnd.Shuffle(len(s.o.Nodes), func(i, j int) { s.o.Nodes[i], s.o.Nodes[j] = s.o.Nodes[j], s.o.Nodes[i] })
		t.Run(fmt.Sprint(run), func(t *testing.T) { sameAsGeoJSON(t, s.o) })
	}
}

// A node missing from both files, for TestReadBoundaries.
const multiC = `<?xml version="1.0" encoding="UTF-8"?>
<osm version="0.6">
  <node id="20" version="1" lat="13.00" lon="103.00"/>
  <node id="21" version="1" lat="13.00" lon="103.10"/>
  <node id="22" version="1" lat="13.10" lon="103.10"/>
  <way id="50" version="3"><nd ref="99"/><nd ref="20"/><nd ref="21"/><nd ref="22"/><nd ref="20"/></way>
  <relation id="2" version="1">
    <member type="way" ref="50" role="outer"/>
    <tag k="type" v="boundary"/><tag k="boundary" v="administrative"/><tag k="name" v="far"/>
  </relation>
</osm>`

func TestReadBoundaries(t *testing.T) {
	dir := t.TempDir()
	a := writeOSM(t, dir, "a.osm", multiA)
	b := writeOSM(t, dir, "b.osm", multiB)
	c := writeOSM(t, dir, "c.osm", multiC)
	all := func(*osm.Relation) bool { return true }

	d, err := ReadBoundaries(a, []string{b}, all)
	if err != nil {
		t.Fatal(err)
	}
	if d.Ways() != 3 || d.Nodes() != 7 || len(d.Relations) != 2 || d.Relations[0].ID != 1 || d.Relations[1].ID != 3 {
		t.Fatalf("relations %d, ways %d, nodes %d", len(d.Relations), d.Ways(), d.Nodes())
	}
	if !d.HasWay(11) || d.HasWay(40) {
		t.Fatal("ways 11 and 40")
	}
	if p, ok := d.Point(2); !ok || p != (orb.Point{100.10, 9.99}) {
		t.Fatalf("node 2 at %v %v, want the later version", p, ok)
	}
	if _, ok := d.Point(8); ok {
		t.Fatal("node 8 of the neighbour's own relation was read")
	}
	if p, ok := d.FirstPoint(d.Relations[1]); !ok || p != (orb.Point{101, 11}) {
		t.Fatalf("first point of r3 %v %v", p, ok)
	}

	// The compact assembly and the raw form agree.
	got, err := d.Assemble(nil)
	if err != nil {
		t.Fatal(err)
	}
	o, err := ReadBoundaryDataWith(a, []string{b}, all)
	if err != nil {
		t.Fatal(err)
	}
	if want := mustAssemble(t, o, nil); !reflect.DeepEqual(got, want) {
		t.Fatalf("BoundaryData.Assemble %+v, Assemble %+v", got, want)
	}
	if !reflect.DeepEqual(d.OSM(), o) {
		t.Fatal("OSM() differs from ReadBoundaryDataWith")
	}

	// A node in no file is left out; the way keeps its id.
	d, err = ReadBoundaries(c, nil, all)
	if err != nil {
		t.Fatal(err)
	}
	if d.Nodes() != 3 {
		t.Fatalf("nodes %d", d.Nodes())
	}
	if _, ok := d.Point(99); ok {
		t.Fatal("node 99 found")
	}
	if p, ok := d.FirstPoint(d.Relations[0]); !ok || p != (orb.Point{103, 13}) {
		t.Fatalf("first point %v %v: node 99 is not in the data, node 20 is", p, ok)
	}
	o = d.OSM()
	if len(o.Ways) != 1 || o.Ways[0].Version != 3 || len(o.Ways[0].Nodes) != 5 || len(o.Nodes) != 3 {
		t.Fatalf("raw %+v", o)
	}
	as, err := d.Assemble(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(as.Built) != 1 || len(as.Built[0].Geometry[0][0]) != 4 {
		t.Fatalf("%+v", as)
	}
	if _, ok := (&BoundaryData{}).FirstPoint(&osm.Relation{Members: osm.Members{{Type: osm.TypeWay, Ref: 1}}}); ok {
		t.Fatal("first point in empty data")
	}
}

// fromOSM searches nodes in place when they are sorted, else copies them;
// both give the same positions, the last copy of a repeated node winning.
func TestFromOSMNodes(t *testing.T) {
	n := func(id osm.NodeID, lon float64) *osm.Node { return &osm.Node{ID: id, Lon: lon, Lat: 1} }
	sorted := fromOSM(&osm.OSM{Nodes: osm.Nodes{n(1, 1), n(2, 2), n(5, 5)}})
	if sorted.nodes == nil {
		t.Fatal("sorted nodes copied")
	}
	unsorted := fromOSM(&osm.OSM{Nodes: osm.Nodes{n(5, 5), n(2, 9), nil, n(1, 1), n(2, 2)}})
	if unsorted.nodes != nil || unsorted.Nodes() != 3 {
		t.Fatalf("unsorted nodes: %d", unsorted.Nodes())
	}
	for _, d := range []*BoundaryData{sorted, unsorted} {
		for id, lon := range map[osm.NodeID]float64{1: 1, 2: 2, 5: 5} {
			if p, ok := d.Point(id); !ok || p != (orb.Point{lon, 1}) {
				t.Fatalf("node %d at %v %v", id, p, ok)
			}
		}
		if _, ok := d.Point(3); ok {
			t.Fatal("node 3 found")
		}
	}
}
