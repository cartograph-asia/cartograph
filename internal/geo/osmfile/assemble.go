package osmfile

import (
	"sort"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/osm"
)

// Assembled is what Assemble made of a set of boundary relations.
type Assembled struct {
	Built     []Boundary            // assembled boundaries, sorted by OSMID
	NotWhole  []int64               // relations with a member way missing from the data, not allowed to be partial (sorted)
	NotClosed []int64               // relations whose rings do not close (sorted)
	Partial   map[int64]PartialInfo // relations built although incomplete, because partial[id] was true
}

// PartialInfo describes what a relation built from its closed rings only
// left out.
type PartialInfo struct {
	RingsBuilt       int // closed rings that went into the geometry
	FragmentsDropped int // ring fragments (chains of ways that do not close) left out; an inner ring outside every outer ring counts here too
	MissingWays      int // member ways absent from the data
}

// Assemble builds the multipolygons of the relations in o. A relation with a
// member way absent from o is NotWhole (no geometry) unless partial[id] is
// true; then it is built from its closed rings only and described in Partial.
// A relation whose rings cannot be closed is NotClosed (unless partial[id],
// then the closed rings only, or NotClosed if none closes). o is not
// changed. It is (*BoundaryData).Assemble on o's data.
func Assemble(o *osm.OSM, partial map[int64]bool) (*Assembled, error) {
	if o == nil {
		return &Assembled{Partial: map[int64]PartialInfo{}}, nil
	}
	return fromOSM(o).Assemble(partial)
}

// Assemble builds the multipolygons of d's relations; see the function
// Assemble for the rules.
//
// Whole relations are assembled as osmgeojson assembled them before
// (polygon, the rules of BuildBoundaries): they count as closed when at
// least one outer ring closes. A relation listed in partial is assembled
// ring by ring (assembleRings) and appears in Partial only when something
// was left out.
func (d *BoundaryData) Assemble(partial map[int64]bool) (*Assembled, error) {
	a := &Assembled{Partial: map[int64]PartialInfo{}}
	for _, r := range d.Relations {
		id := int64(r.ID)
		missing := d.missingWays(r)
		switch {
		case partial[id]:
			g, info := d.assembleRings(r)
			info.MissingWays = missing
			if len(g) == 0 {
				a.NotClosed = append(a.NotClosed, id)
				continue
			}
			a.Built = append(a.Built, Boundary{OSMID: id, Tags: r.Tags, Geometry: g})
			if info.MissingWays > 0 || info.FragmentsDropped > 0 {
				a.Partial[id] = info
			}
		case missing > 0:
			a.NotWhole = append(a.NotWhole, id)
		default:
			g := d.polygon(r)
			if len(g) == 0 {
				a.NotClosed = append(a.NotClosed, id)
				continue
			}
			a.Built = append(a.Built, Boundary{OSMID: id, Tags: r.Tags, Geometry: g})
		}
	}
	sort.Slice(a.Built, func(i, j int) bool { return a.Built[i].OSMID < a.Built[j].OSMID })
	sortIDs(a.NotWhole)
	sortIDs(a.NotClosed)
	return a, nil
}

func sortIDs(s []int64) { sort.Slice(s, func(i, j int) bool { return s[i] < s[j] }) }

// missingWays counts the distinct member ways of r (any role) absent from d.
func (d *BoundaryData) missingWays(r *osm.Relation) int {
	seen := map[osm.WayID]bool{}
	for _, m := range r.Members {
		id := osm.WayID(m.Ref)
		if m.Type != osm.TypeWay || d.HasWay(id) || seen[id] {
			continue
		}
		seen[id] = true
	}
	return len(seen)
}

// assembleRings joins the outer and inner member ways of r that are present
// end to end, keeps the chains that close and attaches every inner ring to
// the smallest outer ring that holds it. Outer rings are counter-clockwise,
// inner rings clockwise, as osmgeojson makes them. The geometry is empty
// when no outer ring closes.
func (d *BoundaryData) assembleRings(r *osm.Relation) (orb.MultiPolygon, PartialInfo) {
	var outer, inner []orb.LineString
	seen := map[osm.WayID]bool{}
	for _, m := range r.Members {
		if m.Type != osm.TypeWay || (m.Role != "outer" && m.Role != "inner") {
			continue
		}
		id := osm.WayID(m.Ref)
		w, ok := d.ways[id]
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		ls := d.line(w)
		if len(ls) < 2 {
			continue
		}
		if m.Role == "outer" {
			outer = append(outer, ls)
		} else {
			inner = append(inner, ls)
		}
	}
	var info PartialInfo
	outerRings, dropped := joinRings(outer)
	info.FragmentsDropped += dropped
	if len(outerRings) == 0 {
		info.FragmentsDropped += len(inner)
		return nil, info
	}
	innerRings, dropped := joinRings(inner)
	info.FragmentsDropped += dropped

	mp := make(orb.MultiPolygon, len(outerRings))
	areas := make([]float64, len(outerRings))
	for i, ring := range outerRings {
		if ring.Orientation() != orb.CCW {
			ring.Reverse()
		}
		mp[i] = orb.Polygon{ring}
		areas[i] = planar.Area(ring)
		info.RingsBuilt++
	}
	for _, ring := range innerRings {
		best := -1
		for i := range mp {
			if ringInside(ring, mp[i][0]) && (best < 0 || areas[i] < areas[best]) {
				best = i
			}
		}
		if best < 0 {
			info.FragmentsDropped++
			continue
		}
		if ring.Orientation() != orb.CW {
			ring.Reverse()
		}
		mp[best] = append(mp[best], ring)
		info.RingsBuilt++
	}
	return mp, info
}

// joinRings joins lines end to end (reversing them where needed) in the
// order given and returns the chains that close as rings, and how many
// chains did not close.
func joinRings(lines []orb.LineString) (rings []orb.Ring, open int) {
	used := make([]bool, len(lines))
	for start := range lines {
		if used[start] {
			continue
		}
		used[start] = true
		chain := append(orb.LineString(nil), lines[start]...)
		for chain[0] != chain[len(chain)-1] {
			first, last := chain[0], chain[len(chain)-1]
			found := false
			for i, l := range lines {
				if used[i] {
					continue
				}
				switch {
				case l[0] == last:
					chain = append(chain, l[1:]...)
				case l[len(l)-1] == last:
					chain = append(chain, reversed(l)[1:]...)
				case l[len(l)-1] == first:
					chain = append(append(orb.LineString(nil), l[:len(l)-1]...), chain...)
				case l[0] == first:
					r := reversed(l)
					chain = append(append(orb.LineString(nil), r[:len(r)-1]...), chain...)
				default:
					continue
				}
				used[i], found = true, true
				break
			}
			if !found {
				break
			}
		}
		if len(chain) >= 4 && chain[0] == chain[len(chain)-1] {
			rings = append(rings, orb.Ring(chain))
		} else {
			open++
		}
	}
	return rings, open
}

func reversed(l orb.LineString) orb.LineString {
	r := make(orb.LineString, len(l))
	for i, p := range l {
		r[len(l)-1-i] = p
	}
	return r
}

// ringInside reports whether every vertex of inner is inside outer or on its
// boundary (inner rings often share a vertex with their outer ring).
func ringInside(inner, outer orb.Ring) bool {
	b, ib := outer.Bound(), inner.Bound()
	if !b.Contains(ib.Min) || !b.Contains(ib.Max) {
		return false
	}
	for _, p := range inner {
		if !planar.RingContains(outer, p) {
			return false
		}
	}
	return true
}
