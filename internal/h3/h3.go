// Package h3 is a thin wrapper around github.com/uber/h3-go so that the rest
// of the code base works with plain string cell indexes and never imports
// h3-go directly.
package h3

import (
	"errors"
	"fmt"
	"strings"

	h3go "github.com/uber/h3-go/v4"
)

// PlotRes is the H3 resolution of a plot.
const PlotRes = 8

// ErrInvalidCell is returned for strings that are not a valid H3 cell index.
var ErrInvalidCell = errors.New("invalid H3 cell")

// CellFromLatLng returns the cell at resolution res that contains the point.
func CellFromLatLng(lat, lng float64, res int) (string, error) {
	c, err := h3go.LatLngToCell(h3go.NewLatLng(lat, lng), res)
	if err != nil {
		return "", fmt.Errorf("h3: latlng to cell: %w", err)
	}
	return c.String(), nil
}

// Parse validates s as an H3 cell index and returns its canonical (lower-case)
// string form and resolution.
func Parse(s string) (string, int, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) != 15 {
		return "", 0, ErrInvalidCell
	}
	c := h3go.CellFromString(s)
	if !c.IsValid() || c.String() != s {
		return "", 0, ErrInvalidCell
	}
	return s, c.Resolution(), nil
}

// Parent returns the parent of cell at resolution res.
func Parent(cell string, res int) (string, error) {
	c, err := parse(cell)
	if err != nil {
		return "", err
	}
	if res < 0 || res > c.Resolution() {
		return "", fmt.Errorf("h3: parent resolution %d out of range for cell at resolution %d", res, c.Resolution())
	}
	p, err := c.Parent(res)
	if err != nil {
		return "", fmt.Errorf("h3: cell to parent: %w", err)
	}
	return p.String(), nil
}

// ChildrenCount returns the number of descendants of cell at childRes
// (H3 cellToChildrenSize). Pentagons have fewer children than hexagons.
func ChildrenCount(cell string, childRes int) (int64, error) {
	c, err := parse(cell)
	if err != nil {
		return 0, err
	}
	res := c.Resolution()
	if childRes < res || childRes > 15 {
		return 0, fmt.Errorf("h3: child resolution %d out of range for cell at resolution %d", childRes, res)
	}
	n := int64(1)
	for range childRes - res {
		n *= 7
	}
	if c.IsPentagon() {
		// One centre child plus five hexagonal sub-trees per level:
		// 1 + 5 * (7^n - 1) / 6.
		return 1 + 5*(n-1)/6, nil
	}
	return n, nil
}

// LatLng is a point in degrees.
type LatLng struct{ Lat, Lng float64 }

// Boundary returns the vertices of cell in degrees, counter-clockwise, not closed.
func Boundary(cell string) ([]LatLng, error) {
	c, err := parse(cell)
	if err != nil {
		return nil, err
	}
	b, err := c.Boundary()
	if err != nil {
		return nil, fmt.Errorf("h3: cell to boundary: %w", err)
	}
	out := make([]LatLng, len(b))
	for i, v := range b {
		out[i] = LatLng{v.Lat, v.Lng}
	}
	return out, nil
}

func parse(cell string) (h3go.Cell, error) {
	c := h3go.CellFromString(cell)
	if !c.IsValid() {
		return 0, ErrInvalidCell
	}
	return c, nil
}

// Centre returns the centre of cell as lat, lng.
func Centre(cell string) (lat, lng float64, err error) {
	c, err := parse(cell)
	if err != nil {
		return 0, 0, err
	}
	ll, err := c.LatLng()
	if err != nil {
		return 0, 0, fmt.Errorf("h3: cell to latlng: %w", err)
	}
	return ll.Lat, ll.Lng, nil
}

// Ring returns the cells at grid distance exactly 1 to k from cell
// (gridDisk without the cell itself).
func Ring(cell string, k int) ([]string, error) {
	c, err := parse(cell)
	if err != nil {
		return nil, err
	}
	disk, err := c.GridDisk(k)
	if err != nil {
		return nil, fmt.Errorf("h3: grid disk: %w", err)
	}
	out := make([]string, 0, len(disk)-1)
	for _, d := range disk {
		if d != c {
			out = append(out, d.String())
		}
	}
	return out, nil
}

// CellCentre is a cell with its centre.
type CellCentre struct {
	Cell     string // 15-char hex index
	Lng, Lat float64
}

// PolygonCells returns the cells at res whose centre lies inside the polygon
// (outer ring and holes as lng, lat pairs), each with its centre.
func PolygonCells(outer [][2]float64, holes [][][2]float64, res int) ([]CellCentre, error) {
	cs, err := polygonToCells(outer, holes, res)
	if err != nil {
		return nil, err
	}
	out := make([]CellCentre, 0, len(cs))
	for _, c := range cs {
		ll, err := c.LatLng()
		if err != nil {
			return nil, fmt.Errorf("h3: cell to latlng: %w", err)
		}
		out = append(out, CellCentre{Cell: c.String(), Lng: ll.Lng, Lat: ll.Lat})
	}
	return out, nil
}

// PolygonCentres returns the centres (lng, lat) of the cells at res whose
// centre lies inside the polygon. outer and holes are rings of lng, lat pairs.
func PolygonCentres(outer [][2]float64, holes [][][2]float64, res int) ([][2]float64, error) {
	cells, err := polygonToCells(outer, holes, res)
	if err != nil {
		return nil, err
	}
	out := make([][2]float64, 0, len(cells))
	for _, c := range cells {
		ll, err := c.LatLng()
		if err != nil {
			return nil, fmt.Errorf("h3: cell to latlng: %w", err)
		}
		out = append(out, [2]float64{ll.Lng, ll.Lat})
	}
	return out, nil
}

func polygonToCells(outer [][2]float64, holes [][][2]float64, res int) ([]h3go.Cell, error) {
	loop := func(r [][2]float64) h3go.GeoLoop {
		l := make(h3go.GeoLoop, len(r))
		for i, p := range r {
			l[i] = h3go.NewLatLng(p[1], p[0])
		}
		return l
	}
	poly := h3go.GeoPolygon{GeoLoop: loop(outer)}
	for _, h := range holes {
		poly.Holes = append(poly.Holes, loop(h))
	}
	cells, err := h3go.PolygonToCells(poly, res)
	if err != nil {
		return nil, fmt.Errorf("h3: polygon to cells: %w", err)
	}
	return cells, nil
}
