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
