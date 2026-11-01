package geo

import (
	"sort"

	"github.com/paulmach/orb"
)

// hilbertOrder is the curve's order: a 65,536 × 65,536 grid, about 3 m per
// step even for the largest province, far finer than a plot.
const hilbertOrder = 16

// HilbertSort orders cells along a Hilbert curve laid over the square that
// contains b, so cells next to each other on the ground get nearby positions.
// Cells in the same grid square keep their order by cell index.
func HilbertSort(cells []LandCell, b orb.Bound) {
	side := max(b.Max[0]-b.Min[0], b.Max[1]-b.Min[1])
	if side <= 0 {
		side = 1
	}
	const n = 1 << hilbertOrder
	scale := func(v, lo float64) uint32 {
		f := (v - lo) / side * (n - 1)
		return uint32(max(0, min(f, n-1)))
	}
	keys := make(map[string]uint64, len(cells))
	for _, c := range cells {
		keys[c.Cell] = hilbertIndex(scale(c.Centre[0], b.Min[0]), scale(c.Centre[1], b.Min[1]))
	}
	sort.Slice(cells, func(i, j int) bool {
		ki, kj := keys[cells[i].Cell], keys[cells[j].Cell]
		if ki != kj {
			return ki < kj
		}
		return cells[i].Cell < cells[j].Cell
	})
}

// hilbertIndex maps (x, y) on the 2^hilbertOrder grid to its distance along
// the Hilbert curve (the classic xy2d).
func hilbertIndex(x, y uint32) uint64 {
	const n = uint32(1) << hilbertOrder
	var d uint64
	for s := n / 2; s > 0; s /= 2 {
		var rx, ry uint32
		if x&s != 0 {
			rx = 1
		}
		if y&s != 0 {
			ry = 1
		}
		d += uint64(s) * uint64(s) * uint64((3*rx)^ry)
		// Rotate the quadrant so the curve stays continuous.
		if ry == 0 {
			if rx == 1 {
				x, y = n-1-x, n-1-y
			}
			x, y = y, x
		}
	}
	return d
}
