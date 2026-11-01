package geo

import (
	"fmt"
	"math"

	"github.com/paulmach/orb"
)

// waterGrid is the bucket size of the water index in degrees.
const waterGrid = 0.05

// Water answers "is this point in the sea?" from OSM water polygons.
type Water struct {
	polys   []waterPoly
	buckets map[[2]int32][]int32
}

type waterPoly struct {
	bound orb.Bound
	rings []orb.Ring // WGS 84; even-odd rule across all rings of the record
}

// NewWater builds an index from shapefile records in EPSG:3857.
func NewWater(recs []ShpRecord) *Water {
	w := &Water{buckets: map[[2]int32][]int32{}}
	for _, r := range recs {
		w.add(r)
	}
	return w
}

// ReadWater reads the water polygons intersecting bound (WGS 84) from a .shp
// or .zip file.
func ReadWater(name string, bound orb.Bound) (*Water, error) {
	rc, err := OpenShp(name)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	w := &Water{buckets: map[[2]int32][]int32{}}
	if err := ReadShp(rc, MercatorBox(bound), func(r ShpRecord) error { w.add(r); return nil }); err != nil {
		return nil, fmt.Errorf("geo: read water %s: %w", name, err)
	}
	return w, nil
}

// ReadShpRecords returns the raw records intersecting bound (WGS 84).
func ReadShpRecords(name string, bound orb.Bound) ([]ShpRecord, error) {
	rc, err := OpenShp(name)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var out []ShpRecord
	err = ReadShp(rc, MercatorBox(bound), func(r ShpRecord) error { out = append(out, r); return nil })
	return out, err
}

func (w *Water) add(r ShpRecord) {
	p := waterPoly{bound: orb.Bound{Min: MercatorToLngLat(orb.Point{r.Box[0], r.Box[1]}), Max: MercatorToLngLat(orb.Point{r.Box[2], r.Box[3]})}}
	for _, ring := range r.Rings {
		rr := make(orb.Ring, len(ring))
		for i, pt := range ring {
			rr[i] = MercatorToLngLat(pt)
		}
		p.rings = append(p.rings, rr)
	}
	id := int32(len(w.polys))
	w.polys = append(w.polys, p)
	x0, y0 := bucket(p.bound.Min)
	x1, y1 := bucket(p.bound.Max)
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			k := [2]int32{x, y}
			w.buckets[k] = append(w.buckets[k], id)
		}
	}
}

func bucket(p orb.Point) (int32, int32) {
	return int32(math.Floor(p[0] / waterGrid)), int32(math.Floor(p[1] / waterGrid))
}

// Len returns the number of polygons in the index.
func (w *Water) Len() int { return len(w.polys) }

// Contains reports whether p (lng, lat) lies in a water polygon.
func (w *Water) Contains(p orb.Point) bool {
	if w == nil {
		return false
	}
	x, y := bucket(p)
	for _, id := range w.buckets[[2]int32{x, y}] {
		wp := &w.polys[id]
		if !wp.bound.Contains(p) {
			continue
		}
		inside := false
		for _, r := range wp.rings {
			if ringCrossings(r, p) {
				inside = !inside
			}
		}
		if inside {
			return true
		}
	}
	return false
}

// ringCrossings reports whether a ray from p to +x crosses r an odd number
// of times (point in ring, even-odd rule).
func ringCrossings(r orb.Ring, p orb.Point) bool {
	in := false
	n := len(r)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		a, b := r[i], r[j]
		if (a[1] > p[1]) != (b[1] > p[1]) &&
			p[0] < (b[0]-a[0])*(p[1]-a[1])/(b[1]-a[1])+a[0] {
			in = !in
		}
	}
	return in
}
