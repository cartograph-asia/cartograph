package geo

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/orb/simplify"

	"github.com/cartograph-asia/cartograph/internal/h3"
)

// SimplifyTolerance is the Douglas-Peucker tolerance for stored region
// boundaries in degrees (about 55 m, well under a plot's 460 m edge).
const SimplifyTolerance = 0.0005

// NearTolerance is how far (degrees, about 1 km) outside every stored
// boundary a point may lie and still be given the nearest region. It covers
// slivers between simplified neighbours and beaches cut off by the coastline.
const NearTolerance = 0.01

// Simplify returns mp simplified for storage; rings that collapse are dropped.
func Simplify(mp orb.MultiPolygon, tol float64) orb.MultiPolygon {
	s := simplify.DouglasPeucker(tol).MultiPolygon(mp.Clone())
	var out orb.MultiPolygon
	for _, p := range s {
		if len(p) == 0 || len(p[0]) < 4 {
			continue
		}
		q := orb.Polygon{p[0]}
		for _, h := range p[1:] {
			if len(h) >= 4 {
				q = append(q, h)
			}
		}
		out = append(out, q)
	}
	return out
}

// EncodeGeometry returns mp as GeoJSON geometry text.
func EncodeGeometry(mp orb.MultiPolygon) (string, error) {
	b, err := json.Marshal(geojson.NewGeometry(mp))
	if err != nil {
		return "", fmt.Errorf("geo: encode geometry: %w", err)
	}
	return string(b), nil
}

// DecodeGeometry parses GeoJSON Polygon or MultiPolygon geometry text.
func DecodeGeometry(s string) (orb.MultiPolygon, error) {
	g, err := geojson.UnmarshalGeometry([]byte(s))
	if err != nil {
		return nil, fmt.Errorf("geo: decode geometry: %w", err)
	}
	switch t := g.Geometry().(type) {
	case orb.Polygon:
		return orb.MultiPolygon{t}, nil
	case orb.MultiPolygon:
		return t, nil
	}
	return nil, fmt.Errorf("geo: geometry is %s, not a polygon", g.Type)
}

// RegionShape is a region's id and stored boundary.
type RegionShape struct {
	ID       int
	Geometry orb.MultiPolygon
	bound    orb.Bound
}

// RegionIndex finds the region containing a point. It holds the provinces
// or the districts of a country (up to about a thousand polygons, scanned
// with a bounding-box prefilter) and is safe for concurrent reads.
type RegionIndex struct {
	shapes []RegionShape
	byID   map[int]int // region id -> index in shapes
}

// NewRegionIndex builds an index; on overlaps the earlier shape wins.
func NewRegionIndex(shapes []RegionShape) *RegionIndex {
	x := &RegionIndex{shapes: make([]RegionShape, len(shapes)), byID: make(map[int]int, len(shapes))}
	for i, s := range shapes {
		var g orb.MultiPolygon
		for _, p := range s.Geometry {
			if len(p) > 0 && len(p[0]) >= 4 {
				g = append(g, p)
			}
		}
		s.Geometry, s.bound = g, g.Bound()
		x.shapes[i] = s
		x.byID[s.ID] = i
	}
	return x
}

// In reports whether p lies inside the boundary of region id.
func (x *RegionIndex) In(id int, p orb.Point) bool {
	if x == nil {
		return false
	}
	i, ok := x.byID[id]
	if !ok {
		return false
	}
	s := &x.shapes[i]
	return s.bound.Contains(p) && planar.MultiPolygonContains(s.Geometry, p)
}

// Len returns the number of regions.
func (x *RegionIndex) Len() int {
	if x == nil {
		return 0
	}
	return len(x.shapes)
}

// Contains returns the region whose boundary contains p.
func (x *RegionIndex) Contains(p orb.Point) (int, bool) {
	if x == nil {
		return 0, false
	}
	for i := range x.shapes {
		s := &x.shapes[i]
		if len(s.Geometry) > 0 && s.bound.Contains(p) && planar.MultiPolygonContains(s.Geometry, p) {
			return s.ID, true
		}
	}
	return 0, false
}

// Locate returns the region containing p, else the region whose boundary is
// nearest to p within NearTolerance.
func (x *RegionIndex) Locate(p orb.Point) (int, bool) {
	if id, ok := x.Contains(p); ok {
		return id, true
	}
	if x == nil {
		return 0, false
	}
	best, bestD := 0, math.Inf(1)
	for i := range x.shapes {
		s := &x.shapes[i]
		if len(s.Geometry) == 0 || !s.bound.Pad(NearTolerance).Contains(p) {
			continue
		}
		if d := planar.DistanceFrom(s.Geometry, p); d < bestD {
			best, bestD = s.ID, d
		}
	}
	if bestD <= NearTolerance {
		return best, true
	}
	return 0, false
}

// LandCell is a res-8 cell that counts as land in a region.
type LandCell struct {
	Cell   string
	Centre orb.Point
}

// LandCells returns the res-8 cells whose centre lies inside shape (and is
// located to shape by idx, so overlapping neighbours never count a cell
// twice), is not in water, and lies inside clip unless clip is empty.
func LandCells(shape RegionShape, idx *RegionIndex, water *Water, clip orb.Bound) ([]LandCell, error) {
	var out []LandCell
	for _, poly := range shape.Geometry {
		holes := make([][][2]float64, 0, len(poly)-1)
		for _, h := range poly[1:] {
			holes = append(holes, ringPairs(h))
		}
		cells, err := h3.PolygonCells(ringPairs(poly[0]), holes, h3.PlotRes)
		if err != nil {
			return nil, err
		}
		for _, c := range cells {
			p := orb.Point{c.Lng, c.Lat}
			if !NoBound(clip) && !clip.Contains(p) {
				continue
			}
			if id, ok := idx.Contains(p); !ok || id != shape.ID {
				continue
			}
			if water.Contains(p) {
				continue
			}
			out = append(out, LandCell{Cell: c.Cell, Centre: p})
		}
	}
	return out, nil
}

func ringPairs(r orb.Ring) [][2]float64 {
	out := make([][2]float64, len(r))
	for i, p := range r {
		out[i] = [2]float64{p[0], p[1]}
	}
	return out
}

// NoBound reports whether b means "no clipping": empty or the zero value.
func NoBound(b orb.Bound) bool { return b.IsEmpty() || b.IsZero() }

// ParseBound parses "minLng,minLat,maxLng,maxLat"; empty yields an empty bound.
func ParseBound(s string) (orb.Bound, error) {
	if s == "" {
		return orb.Bound{Min: orb.Point{1, 1}, Max: orb.Point{0, 0}}, nil // IsEmpty
	}
	var b orb.Bound
	if _, err := fmt.Sscanf(s, "%g,%g,%g,%g", &b.Min[0], &b.Min[1], &b.Max[0], &b.Max[1]); err != nil {
		return b, fmt.Errorf("geo: bbox %q: want minLng,minLat,maxLng,maxLat", s)
	}
	if b.Min[0] >= b.Max[0] || b.Min[1] >= b.Max[1] {
		return b, fmt.Errorf("geo: bbox %q is empty", s)
	}
	return b, nil
}
