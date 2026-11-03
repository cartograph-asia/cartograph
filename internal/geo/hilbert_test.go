package geo

import (
	"testing"

	"github.com/paulmach/orb"
)

func TestHilbertIndexIsAContinuousCurve(t *testing.T) {
	// The 8×8 block at the origin is the start of the curve: it holds exactly
	// the positions 0..63, and each step moves to a neighbouring square.
	const k = 8
	at := make(map[uint64][2]int, k*k)
	for x := 0; x < k; x++ {
		for y := 0; y < k; y++ {
			at[hilbertIndex(uint32(x), uint32(y))] = [2]int{x, y}
		}
	}
	for d := uint64(0); d < k*k; d++ {
		p, ok := at[d]
		if !ok {
			t.Fatalf("position %d missing from the first %d×%d block", d, k, k)
		}
		if d == 0 {
			continue
		}
		q := at[d-1]
		if dx, dy := abs(p[0]-q[0]), abs(p[1]-q[1]); dx+dy != 1 {
			t.Fatalf("step %d→%d jumps from %v to %v", d-1, d, q, p)
		}
	}
}

func TestHilbertSortKeepsNeighboursClose(t *testing.T) {
	b := orb.Bound{Min: orb.Point{100, 12}, Max: orb.Point{101, 13}}
	cells := []LandCell{
		{ID: 4, Centre: orb.Point{100.95, 12.05}},
		{ID: 1, Centre: orb.Point{100.01, 12.01}},
		{ID: 3, Centre: orb.Point{100.95, 12.95}},
		{ID: 2, Centre: orb.Point{100.02, 12.01}},
	}
	HilbertSort(cells, b)
	got := []uint64{cells[0].ID, cells[1].ID}
	if got[0] != 1 || got[1] != 2 {
		t.Fatalf("order %v: the two neighbouring cells should come first", cells)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
