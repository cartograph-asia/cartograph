package countryscan

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/h3"
)

// pairOverlap is the land plots two countries both claim.
type pairOverlap struct {
	a, b                           countries.Country
	cells                          int
	minLat, maxLat, minLng, maxLng float64
	sample                         []uint64
}

// intersect counts the cells two sorted lists share.
func intersect(a, b []uint64, keep func(uint64)) {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] < b[j]:
			i++
		case a[i] > b[j]:
			j++
		default:
			keep(a[i])
			i++
			j++
		}
	}
}

// Overlaps compares the land cells the scans left in cache and renders the
// plots that lie in the outline of two countries as Markdown: the border
// plots, and the disputed areas Adrian decides on (brief, question 5).
func Overlaps(cache string) (string, error) {
	type loaded struct {
		c     countries.Country
		cells []uint64
	}
	var have []loaded
	var missing []string
	for _, c := range countries.All {
		cells, err := ReadCells(CellsFile(cache, c.Code))
		if errors.Is(err, fs.ErrNotExist) {
			missing = append(missing, c.NameEn)
			continue
		}
		if err != nil {
			return "", err
		}
		have = append(have, loaded{c, cells})
	}
	var pairs []pairOverlap
	for i := range have {
		for j := i + 1; j < len(have); j++ {
			p := pairOverlap{a: have[i].c, b: have[j].c, minLat: 90, minLng: 180, maxLat: -90, maxLng: -180}
			intersect(have[i].cells, have[j].cells, func(id uint64) {
				p.cells++
				if len(p.sample) < 5 {
					p.sample = append(p.sample, id)
				}
				if lat, lng, err := h3.IDCentre(id); err == nil {
					p.minLat, p.maxLat = min(p.minLat, lat), max(p.maxLat, lat)
					p.minLng, p.maxLng = min(p.minLng, lng), max(p.maxLng, lng)
				}
			})
			if p.cells > 0 {
				pairs = append(pairs, p)
			}
		}
	}

	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	w("# Country scan: overlaps between countries\n\n")
	w("Written by `go run ./cmd/countryscan -overlaps` from the land plots (resolution 8) each country scan left behind. A plot is listed when its centre lies inside the outline of two countries and is not in water. ")
	w("OSM's de facto boundaries apply; each overlap is decided by Adrian and the decision gets a line in `docs/decisions.md` (`docs/COUNTRY_IMPORT_BRIEF.md`, question 5). ")
	w("A plot in both countries belongs to the country imported first and keeps its number.\n\n")
	if len(missing) > 0 {
		w("Not scanned yet, so not compared: %s.\n\n", strings.Join(missing, ", "))
	}
	if len(pairs) == 0 {
		w("No plot lies in two of the %d scanned countries.\n", len(have))
		return b.String(), nil
	}
	w("| Countries | Plots | About km² | Between (lat, lng) | Examples |\n|---|---:|---:|---|---|\n")
	for _, p := range pairs {
		var ex []string
		for _, id := range p.sample {
			if lat, lng, err := h3.IDCentre(id); err == nil {
				ex = append(ex, fmt.Sprintf("%.4f, %.4f", lat, lng))
			}
		}
		w("| %s / %s | %s | %s | %.2f, %.2f to %.2f, %.2f | %s |\n", p.a.NameEn, p.b.NameEn, num(p.cells), num(int(float64(p.cells)*PlotKm2)),
			p.minLat, p.minLng, p.maxLat, p.maxLng, strings.Join(ex, "; "))
	}
	return b.String(), nil
}
