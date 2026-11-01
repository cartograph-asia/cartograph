package web

import (
	"fmt"
	"html/template"
	"math"
	"strconv"
	"strings"

	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/views"
)

// The small maps are drawn twice: as a server-side SVG of the plot outlines
// (complete without JavaScript), which the MapLibre module replaces with a
// real map once it scrolls into view.

// maxSVGCells caps the SVG fallback; a profile with more plots shows the link only.
const maxSVGCells = 2000

type svgCell struct {
	ring  []h3.LatLng
	class string // "s s1" … "s s3" surveyed (by photo count), "b" blank, plus " f" for the focused plot
}

// photoClass is a cell's class by its photo count, in the three steps of
// the /map page: 1 photo, 2–3, 4 or more.
func photoClass(n int) string {
	switch {
	case n <= 0:
		return "b"
	case n == 1:
		return "s s1"
	case n <= 3:
		return "s s2"
	}
	return "s s3"
}

// plotSVG draws a plot and its neighbours.
func plotSVG(p *views.Plot) template.HTML {
	cells := []svgCell{{class: photoClass(p.PhotoCount) + " f"}}
	cellIDs := []string{p.Plot.Cell}
	for _, n := range p.Neighbours {
		cells = append(cells, svgCell{class: photoClass(n.Photos)})
		cellIDs = append(cellIDs, n.Cell)
	}
	surveyed := 0
	for _, n := range p.Neighbours {
		if n.Surveyed {
			surveyed++
		}
	}
	label := "Outline of this plot."
	if len(p.Neighbours) > 0 {
		label = fmt.Sprintf("Outline of this plot and its %d neighbours, %d of them surveyed.", len(p.Neighbours), surveyed)
	}
	return drawSVG(cellIDs, cells, label)
}

// cellsSVG draws a set of surveyed plots, e.g. a Surveyor's, coloured by
// their photo counts.
func cellsSVG(ids []string, photos map[string]int, label string) template.HTML {
	if len(ids) == 0 || len(ids) > maxSVGCells {
		return ""
	}
	cells := make([]svgCell, len(ids))
	for i, id := range ids {
		cells[i].class = photoClass(max(photos[id], 1))
	}
	return drawSVG(ids, cells, label)
}

func drawSVG(ids []string, cells []svgCell, label string) template.HTML {
	minLat, minLng, maxLat, maxLng := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	kept := cells[:0]
	for i, id := range ids {
		ring, err := h3.Boundary(id)
		if err != nil {
			continue
		}
		c := cells[i]
		c.ring = ring
		kept = append(kept, c)
		for _, v := range ring {
			minLat, maxLat = math.Min(minLat, v.Lat), math.Max(maxLat, v.Lat)
			minLng, maxLng = math.Min(minLng, v.Lng), math.Max(maxLng, v.Lng)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	// Equirectangular projection scaled by cos(lat), fitted into a 600-wide box
	// with a margin; at least ~10 km across so a single plot is not huge.
	k := math.Cos((minLat + maxLat) / 2 * math.Pi / 180)
	w := (maxLng - minLng) * k
	h := maxLat - minLat
	minSpan := 0.03
	if w < minSpan {
		minLng -= (minSpan - w) / 2 / k
		w = minSpan
	}
	if h < minSpan*0.6 {
		minLat -= (minSpan*0.6 - h) / 2
		h = minSpan * 0.6
	}
	const vw = 600.0
	vh := math.Max(200, math.Min(600, vw*h/w))
	scale := math.Min((vw-40)/w, (vh-40)/h)
	ox := (vw - w*scale) / 2
	oy := (vh - h*scale) / 2
	top := minLat + h
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }

	paths := map[string]*strings.Builder{}
	var order []string
	for _, c := range kept {
		b := paths[c.class]
		if b == nil {
			b = &strings.Builder{}
			paths[c.class] = b
			order = append(order, c.class)
		}
		for i, v := range c.ring {
			x := ox + (v.Lng-minLng)*k*scale
			y := oy + (top-v.Lat)*scale
			if i == 0 {
				b.WriteString("M")
			} else {
				b.WriteString("L")
			}
			b.WriteString(f(x) + " " + f(y))
		}
		b.WriteString("Z")
	}
	var out strings.Builder
	fmt.Fprintf(&out, `<svg class="cells" viewBox="0 0 %s %s" role="img" aria-label="%s">`, f(vw), f(vh), template.HTMLEscapeString(label))
	// Blank first, focus last, so outlines sit on top.
	rank := func(c string) int {
		switch {
		case strings.Contains(c, "f"):
			return 2
		case c == "s":
			return 1
		}
		return 0
	}
	for r := 0; r <= 2; r++ {
		for _, c := range order {
			if rank(c) == r {
				fmt.Fprintf(&out, `<path class="%s" d="%s"/>`, c, paths[c].String())
			}
		}
	}
	out.WriteString(`</svg>`)
	return template.HTML(out.String())
}
