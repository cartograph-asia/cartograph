package web

import (
	"fmt"
	"html/template"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/storage"
	"github.com/cartograph-asia/cartograph/internal/views"
)

// Image variant sizes (longest side) as stored by internal/photos.
const (
	thumbPx  = 480
	mediumPx = 1200
	largePx  = 2000
)

func (rd *Renderer) funcs() template.FuncMap {
	return template.FuncMap{
		"asset":       rd.assets.url,
		"abs":         rd.abs,
		"img":         rd.img,
		"absimg":      rd.absImg,
		"srcset":      rd.srcset,
		"minimap":     func() string { return rd.cfg.MinimapScript },
		"when":        when,
		"date":        date,
		"iso":         iso,
		"num":         num,
		"count":       count,
		"surveyed":    surveyed,
		"pct":         pct,
		"bar":         bar,
		"pageURL":     pageURL,
		"dims":        dims,
		"grid":        grid,
		"heading":     heading,
		"join":        strings.Join,
		"photoCounts": photoCounts,
		"photosFor":   photosFor,
		"plotSVG":     plotSVG,
		"cellsSVG":    cellsSVG,
		"mapHash":     mapHash,
		"nearbyURL":   nearbyURL,
		"bounds":      boundsAttr,
		"plotCells":   func(p *views.Plot) []string { return neighbourCells(p.Plot.Cell, p.Neighbours) },
		// pathsFor is the link of each cell for data-paths, parallel to data-cells.
		"pathsFor": func(cells []string, paths map[string]string) string {
			out := make([]string, len(cells))
			for i, c := range cells {
				if out[i] = paths[c]; out[i] == "" {
					out[i] = "/plot/" + c
				}
			}
			return strings.Join(out, ",")
		},
		"pathEsc":       url.PathEscape,
		"adminSections": func() []adminSection { return adminSections },
		"featuredRow": func(f views.AdminFeatured, page *views.AdminFeaturedList, state string) featuredRow {
			return featuredRow{F: f, Page: page, State: state}
		},
		"overrideActions": func() []string {
			return []string{views.OverrideUndecided, views.OverrideKeep, views.OverrideReplace}
		},
		"stamp":      stamp,
		"localTime":  func(t time.Time) string { return t.In(zone).Format("2006-01-02T15:04") },
		"pageQuery":  pageQuery,
		"reasonMax":  func() int { return views.ReasonMax },
		"noteMax":    func() int { return views.InviteNoteMax },
		"textMax":    func() int { return views.FeaturedTextMax },
		"csrfField":  func() string { return views.CSRFField },
		"bioMax":     func() int { return views.BioMaxLen },
		"displayMax": func() int { return views.DisplayNameMax },
		"nameMin":    func() int { return views.NameMinLen },
		"nameMax":    func() int { return views.NameMaxLen },
		"namePat":    namePattern,
		"activeDays": func() int {
			return views.ActiveDays
		},
	}
}

// abs makes a site path absolute with BaseURL.
func (rd *Renderer) abs(p string) string {
	if strings.Contains(p, "://") {
		return p
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return rd.cfg.BaseURL + p
}

// img returns the public URL of an object key.
func (rd *Renderer) img(key string) string { return storage.PublicURL(rd.cfg.ImgBaseURL, key) }

// absImg is img made absolute, for Open Graph and feeds.
func (rd *Renderer) absImg(key string) string { return rd.abs(rd.img(key)) }

// srcset lists the variants of keys up to maxPx with their widths. Widths are
// the longest side, which is exact for landscape photos and conservative
// (smaller than true) for portrait ones.
func (rd *Renderer) srcset(k views.ImageKeys, maxPx int) string {
	parts := []string{rd.img(k.Thumb) + " " + strconv.Itoa(thumbPx) + "w"}
	if k.Medium != "" && maxPx >= mediumPx {
		parts = append(parts, rd.img(k.Medium)+" "+strconv.Itoa(mediumPx)+"w")
	}
	if k.Large != "" && (maxPx >= largePx || (k.Medium == "" && maxPx >= mediumPx)) {
		parts = append(parts, rd.img(k.Large)+" "+strconv.Itoa(largePx)+"w")
	}
	return strings.Join(parts, ", ")
}

// size is a width and height in pixels.
type size struct{ W, H int }

// dims scales a photo's stored size to longest side px; unknown sizes
// (prototype photos) are taken as 4:3 landscape.
func dims(p views.PhotoItem, px int) size {
	w, h := p.Width, p.Height
	if w <= 0 || h <= 0 {
		w, h = 4, 3
	}
	if w >= h {
		return size{px, int(math.Round(float64(px) * float64(h) / float64(w)))}
	}
	return size{int(math.Round(float64(px) * float64(w) / float64(h))), px}
}

// gridData is the input of the "grid" partial.
type gridData struct {
	Now    time.Time
	Photos []views.PhotoItem
	Eager  bool // the first photo is the page's first image: fetchpriority=high
	// Of names what every photo in the grid shares, so the captions leave it
	// out: "plot" on a plot page, "surveyor" on a Surveyor's page.
	Of string
}

// grid is the "grid" partial's input; an optional fourth argument is Of.
func grid(now time.Time, photos []views.PhotoItem, eager bool, of ...string) gridData {
	g := gridData{Now: now, Photos: photos, Eager: eager}
	if len(of) > 0 {
		g.Of = of[0]
	}
	return g
}

// heading sets a plot name as a page heading: the dot stays with the last
// word before it, so a long name breaks after the dot (Chrome still breaks
// before a no-wrap span that starts with the space, so the span holds the
// word as well), and the province and number are one inline block that
// moves to the next line whole unless it is wider than the line itself
// ("Pattaya, Bang Lamung ·" / "Chon Buri 2417"). The name itself comes from
// PlotRef.Name.
func heading(p views.PlotRef) template.HTML {
	local, province, ok := strings.Cut(p.Name(), " · ")
	if !ok {
		return template.HTML(template.HTMLEscapeString(local))
	}
	head, last := "", local
	if i := strings.LastIndexByte(local, ' '); i >= 0 {
		head, last = local[:i+1], local[i+1:]
	}
	esc := template.HTMLEscapeString
	return template.HTML(esc(head) + `<span class="nb">` + esc(last) + ` ·</span> <span class="pv">` + esc(province) + `</span>`)
}

func toTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, !t.IsZero()
	case *time.Time:
		if t == nil {
			return time.Time{}, false
		}
		return *t, !t.IsZero()
	}
	return time.Time{}, false
}

// date formats a time as "1 November 2026" in Thai time.
func date(v any) string {
	t, ok := toTime(v)
	if !ok {
		return ""
	}
	return t.In(zone).Format("2 January 2006")
}

func iso(v any) string {
	t, ok := toTime(v)
	if !ok {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// ago describes t relative to now: "just now", "3 hours ago", "yesterday",
// "5 days ago", "2 weeks ago", then the date.
func ago(now, t time.Time) string {
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}
	n := func(v int, unit string) string {
		if v == 1 {
			return "1 " + unit + " ago"
		}
		return strconv.Itoa(v) + " " + unit + "s ago"
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return n(int(d/time.Minute), "minute")
	case d < 24*time.Hour:
		return n(int(d/time.Hour), "hour")
	}
	days := calendarDays(now, t)
	switch {
	case days <= 1:
		return "yesterday"
	case days < 14:
		return strconv.Itoa(days) + " days ago"
	case days < 56:
		return n(days/7, "week")
	}
	return date(t)
}

// calendarDays counts calendar days in Thai time between t and now.
func calendarDays(now, t time.Time) int {
	a := now.In(zone)
	b := t.In(zone)
	da := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	db := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(da.Sub(db).Hours() / 24)
}

// when renders a <time> element with the relative time and the date as title.
func when(now time.Time, v any) template.HTML {
	t, ok := toTime(v)
	if !ok {
		return ""
	}
	return template.HTML(fmt.Sprintf(`<time datetime="%s" title="%s">%s</time>`,
		iso(t), template.HTMLEscapeString(date(t)), template.HTMLEscapeString(ago(now, t))))
}

// num formats an integer with thousands separators: 5921 -> "5,921".
func num(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// count is "1 photo" / "12 photos".
func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return num(n) + " " + many
}

// surveyed is what follows the number of plots a Surveyor surveyed, with
// the plots they claimed, in the same words on the profile and the list:
// "plots surveyed, 5 of them first". No claimed plots, no tail. The tail
// does not break, so a narrow caption wraps after the comma.
func surveyed(n, first int) template.HTML {
	s := "plots surveyed"
	if n == 1 {
		s = "plot surveyed"
	}
	switch {
	case first == 0:
		return template.HTML(s)
	case n == 1:
		return template.HTML(s + `, <span class="nb">the first</span>`)
	}
	return template.HTML(s + `, <span class="nb">` + num(first) + ` of them first</span>`)
}

// pct formats a coverage share: "0%", "0.22%", "4.5%", "37%".
func pct(c views.Coverage) string {
	p := c.Percent()
	switch {
	case p == 0:
		return "0%"
	case p < 0.01:
		return "<0.01%"
	case p < 1:
		return strconv.FormatFloat(p, 'f', 2, 64) + "%"
	case p < 10:
		return strconv.FormatFloat(p, 'f', 1, 64) + "%"
	}
	return strconv.FormatFloat(p, 'f', 0, 64) + "%"
}

// bar is the length of a coverage bar in viewBox units (0-100).
func bar(c views.Coverage) string {
	// At least 0.6 of 100 units, so a single surveyed plot stays visible.
	return strconv.FormatFloat(math.Max(0.6, math.Min(100, c.Percent())), 'f', 2, 64)
}

// pageURL returns canonical's path with ?page=n (no query for page 1).
// adminSection is one link of the admin navigation line.
type adminSection struct{ Key, Path, Label string }

var adminSections = []adminSection{
	{views.AdminOverviewSection, "/admin", "Overview"},
	{views.AdminPhotosSection, "/admin/photos", "Photos"},
	{views.AdminInvitesSection, "/admin/invites", "Invitations"},
	{views.AdminFeaturedSection, "/admin/featured", "Plot of the week"},
	{views.AdminSurveyorsSection, "/admin/surveyors", "Surveyors"},
	{views.AdminPlacesSection, "/admin/places", "Place names"},
	{views.AdminLogSection, "/admin/log", "Log"},
}

// featuredRow is one plot-of-the-week entry with its page (for the CSRF
// field) and whether it is current, scheduled or past.
type featuredRow struct {
	F     views.AdminFeatured
	Page  *views.AdminFeaturedList
	State string
}

// stamp is a date with the time of day in Bangkok, for the admin area:
// "1 November 2026, 16:05".
func stamp(v any) string {
	t, ok := toTime(v)
	if !ok {
		return ""
	}
	return t.In(zone).Format("2 January 2006, 15:04")
}

// pageQuery is pageURL that keeps the other query parameters of canonical
// (the admin lists' filters).
func pageQuery(canonical string, n int) string {
	p, q, _ := strings.Cut(canonical, "?")
	vs, _ := url.ParseQuery(q)
	vs.Del("page")
	vs.Del("done")
	if n > 1 {
		vs.Set("page", strconv.Itoa(n))
	}
	if e := vs.Encode(); e != "" {
		return p + "?" + e
	}
	return p
}

func pageURL(canonical string, n int) string {
	p, _, _ := strings.Cut(canonical, "?")
	if n <= 1 {
		return p
	}
	return p + "?page=" + strconv.Itoa(n)
}

// photoCounts is "6,0,2" for the cells of a small map, the plot itself and
// then its neighbours: their photo counts, 0 for a blank plot.
func photoCounts(self int, ns []views.CellStatus) string {
	b := make([]string, 0, len(ns)+1)
	b = append(b, strconv.Itoa(self))
	for _, s := range ns {
		b = append(b, strconv.Itoa(s.Photos))
	}
	return strings.Join(b, ",")
}

// photosFor is the photo count of each cell for data-photos, parallel to data-cells.
func photosFor(cells []string, n map[string]int) string {
	b := make([]string, len(cells))
	for i, c := range cells {
		b[i] = strconv.Itoa(n[c])
	}
	return strings.Join(b, ",")
}

// neighbourCells is the plot itself followed by its neighbours.
func neighbourCells(cell string, ns []views.CellStatus) []string {
	out := []string{cell}
	for _, n := range ns {
		out = append(out, n.Cell)
	}
	return out
}

// boundsAttr is "minLng,minLat,maxLng,maxLat" for data attributes.
func boundsAttr(b views.Bounds) string {
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', 5, 64) }
	return strings.Join([]string{f(b.MinLng), f(b.MinLat), f(b.MaxLng), f(b.MaxLat)}, ",")
}

// mapHash is the /map URL showing a bounding box ("#zoom/lat/lng").
func mapHash(b views.Bounds) string {
	span := math.Max(b.MaxLng-b.MinLng, b.MaxLat-b.MinLat)
	z := 12.0
	if span > 0 {
		z = math.Max(5, math.Min(15, math.Floor(math.Log2(360/span))-1))
	}
	return fmt.Sprintf("/map#%.0f/%.4f/%.4f", z, (b.MinLat+b.MaxLat)/2, (b.MinLng+b.MaxLng)/2)
}

// nearbyURL is Wikidata's "Nearby" page for the centre of a plot: the items
// and images around it. It is the plot's centre, never the position of a
// photo, so the link tells no more than the plot page already does.
func nearbyURL(cell string) string {
	lat, lng, err := h3.Centre(cell)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("https://www.wikidata.org/wiki/Special:Nearby#/coord/%.4f,%.4f", lat, lng)
}

// namePattern is views.NamePattern for an HTML pattern attribute: anchors
// dropped (implied) and the hyphen escaped, which the "v" regexp flag that
// browsers use for pattern requires inside a class.
func namePattern() string {
	return strings.ReplaceAll(strings.Trim(views.NamePattern, "^$"), "9-]", `9\-]`)
}
