package osmfile

// The multipolygon assembly of github.com/paulmach/osm/osmgeojson (v0.9.0,
// build_polygon.go, and internal/mputil), MIT licence, Copyright (c) 2013
// Paul Mach, reduced to the geometry of one relation and reading positions
// from BoundaryData instead of *osm.Node. It is kept step for step so that a
// boundary comes out exactly as osmgeojson.Convert built it before; the
// tests compare the two.

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
)

// typeTrue is osmgeojson's ignore set for a relation's own tags.
var typeTrue = map[string]string{"type": "true"}

// polygon returns the multipolygon osmgeojson builds for r, or nil where it
// builds none: r is not of type multipolygon or boundary, no outer ring
// closes, or r has one outer way and no tags of interest besides type (then
// osmgeojson names the feature after the way, and BuildBoundaries never
// found it). Member ways absent from d are left out; a member that carries
// its own nodes stands in for its way.
func (d *BoundaryData) polygon(r *osm.Relation) orb.MultiPolygon {
	if tt := r.Tags.Find("type"); tt != "multipolygon" && tt != "boundary" {
		return nil
	}
	var outer, inner []segment
	outerCount := 0
	for _, m := range r.Members {
		if m.Type != osm.TypeWay || (m.Role != "inner" && m.Role != "outer") {
			continue
		}
		if m.Role == "outer" {
			outerCount++
		}
		w, ok := d.ways[osm.WayID(m.Ref)]
		if !ok {
			if len(m.Nodes) == 0 {
				continue
			}
			w = compactWay(0, m.Nodes)
		}
		ls := d.line(w)
		if len(ls) == 0 {
			continue
		}
		s := segment{orientation: m.Orientation, line: ls}
		if m.Role == "outer" {
			if s.orientation == orb.CW {
				s.reverse()
			}
			outer = append(outer, s)
		} else {
			if s.orientation == orb.CCW {
				s.reverse()
			}
			inner = append(inner, s)
		}
	}
	if len(outer) == 0 {
		return nil
	}
	if len(outer) == 1 && outerCount == 1 {
		ring := multiSegment(outer).ring(orb.CCW)
		if len(ring) < 4 || !ring.Closed() {
			return nil
		}
		if !hasInterestingTags(r.Tags, typeTrue) {
			return nil
		}
		innerSections := join(inner)
		p := make(orb.Polygon, 0, len(inner)+1)
		p = append(p, ring)
		for _, is := range innerSections {
			p = append(p, is.ring(orb.CW))
		}
		return orb.MultiPolygon{p}
	}
	mp := make(orb.MultiPolygon, 0, len(outer))
	for _, os := range join(outer) {
		ring := os.ring(orb.CCW)
		if len(ring) < 4 || !ring.Closed() {
			continue
		}
		mp = append(mp, orb.Polygon{ring})
	}
	if len(mp) == 0 {
		return nil
	}
	for _, is := range join(inner) {
		ring := is.ring(orb.CW)
		for i := range mp {
			if polygonContains(mp[i][0], ring) {
				mp[i] = append(mp[i], ring)
				break
			}
		}
	}
	return mp
}

func hasInterestingTags(tags osm.Tags, ignore map[string]string) bool {
	for _, t := range tags {
		if !osm.UninterestingTags[t.Key] && (ignore == nil || !(ignore[t.Key] == "true" || ignore[t.Key] == t.Value)) {
			return true
		}
	}
	return false
}

// polygonContains reports whether any vertex of r lies inside outer.
func polygonContains(outer orb.Ring, r orb.Ring) bool {
	for _, p := range r {
		inside := false
		x, y := p[0], p[1]
		i, j := 0, len(outer)-1
		for i < len(outer) {
			xi, yi := outer[i][0], outer[i][1]
			xj, yj := outer[j][0], outer[j][1]
			if ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi)+xi) {
				inside = !inside
			}
			j = i
			i++
		}
		if inside {
			return true
		}
	}
	return false
}

// segment is a member way's line (mputil.Segment).
type segment struct {
	orientation orb.Orientation
	reversed    bool
	line        orb.LineString
}

func (s *segment) reverse() {
	s.reversed = !s.reversed
	s.line.Reverse()
}

func (s segment) first() orb.Point { return s.line[0] }
func (s segment) last() orb.Point  { return s.line[len(s.line)-1] }

// multiSegment is a chain of segments joined end to end.
type multiSegment []segment

func (ms multiSegment) first() orb.Point { return ms[0].line[0] }

func (ms multiSegment) last() orb.Point {
	l := ms[len(ms)-1].line
	return l[len(l)-1]
}

// ring makes the chain a ring of orientation o, using the members'
// orientations where they give one.
func (ms multiSegment) ring(o orb.Orientation) orb.Ring {
	n := 0
	for _, s := range ms {
		n += len(s.line)
	}
	ring := make(orb.Ring, 0, n)
	haveOrient, reversed := false, false
	for _, s := range ms {
		if s.orientation != 0 {
			haveOrient = true
			if (s.orientation == o) == s.reversed {
				reversed = true
			}
		}
		ring = append(ring, s.line...)
	}
	if (haveOrient && reversed) || (!haveOrient && ring.Orientation() != o) {
		ring.Reverse()
	}
	return ring
}

// join joins segments end to end into chains (mputil.Join), taking them
// from the end of the list, as osmgeojson does; the order decides where a
// ring starts.
func join(segments []segment) []multiSegment {
	lists := []multiSegment{}
	at := 0
	for _, s := range segments {
		if len(s.line) <= 1 {
			continue
		}
		segments[at] = s
		at++
	}
	segments = segments[:at]
	for len(segments) != 0 {
		current := multiSegment{segments[len(segments)-1]}
		segments = segments[:len(segments)-1]
		for len(segments) != 0 && !current.first().Equal(current.last()) {
			first, last := current.first(), current.last()
			foundAt := -1
			for i, s := range segments {
				if last.Equal(s.first()) {
					s.line = s.line[1:]
					current = append(current, s)
				} else if last.Equal(s.last()) {
					s.reverse()
					s.line = s.line[1:]
					current = append(current, s)
				} else if first.Equal(s.last()) {
					s.line = s.line[:len(s.line)-1]
					current = append(multiSegment{s}, current...)
				} else if first.Equal(s.first()) {
					s.reverse()
					s.line = s.line[:len(s.line)-1]
					current = append(multiSegment{s}, current...)
				} else {
					continue
				}
				foundAt = i
				break
			}
			if foundAt == -1 {
				break // a dangling way, a ring that does not close
			}
			if foundAt < len(segments)/2 {
				for i := foundAt; i > 0; i-- {
					segments[i] = segments[i-1]
				}
				segments = segments[1:]
			} else {
				for i := foundAt + 1; i < len(segments); i++ {
					segments[i-1] = segments[i]
				}
				segments = segments[:len(segments)-1]
			}
		}
		lists = append(lists, current)
	}
	return lists
}
