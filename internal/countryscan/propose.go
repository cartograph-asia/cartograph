package countryscan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

const (
	// minCoverPct and maxOverlapPct are what a level must reach to be a
	// candidate for provinces or districts.
	minCoverPct   = 90.0
	maxOverlapPct = 2.0
	// maxISOOverlapPct is the overlap allowed for a province level that the
	// ISO 3166-2 codes confirm (at least isoShare of its relations carry one):
	// Myanmar's states and regions overlap by 2.9 %.
	maxISOOverlapPct = 5.0
	isoShare         = 0.8
)

// candidate reports whether l tiles the land well enough to be a province or
// district level.
func candidate(l *Level) bool {
	return l.Geometry && l.Level >= 3 && l.N >= 1 && l.CoverPct >= minCoverPct && l.OverlapPct <= maxOverlapPct
}

// propose picks the province level (the lowest level with at least three
// relations, at least 80 % of them with the country's ISO 3166-2 code and
// the cover and overlap limits, else the lowest candidate with at least three
// relations, else the lowest candidate) and the district level (the next candidate below it with more
// relations). Both are proposals; the table of levels is what to decide by.
func (s *scanner) propose() {
	r := s.r
	r.Province = Proposal{Role: "province"}
	r.District = Proposal{Role: "district"}
	for _, l := range r.Levels {
		if candidate(l) {
			r.Province.Candidates = append(r.Province.Candidates, l.Level)
		}
	}
	r.District.Candidates = r.Province.Candidates

	var prov *Level
	for _, l := range r.Levels {
		if l.Geometry && l.Level >= 3 && l.N >= 3 && l.CoverPct >= minCoverPct && l.OverlapPct <= maxISOOverlapPct &&
			float64(l.ISOHere) >= isoShare*float64(l.N) {
			prov = l
			break
		}
	}
	for _, l := range r.Levels {
		if prov == nil && candidate(l) && l.N >= 3 {
			prov = l
		}
	}
	if prov == nil && len(r.Province.Candidates) > 0 {
		prov = r.levelOf(r.Province.Candidates[0]) // a country with fewer than three provinces
	}
	if prov == nil {
		var best *Level
		for _, l := range r.Levels {
			if l.Geometry && l.Level >= 3 && (best == nil || l.CoverPct > best.CoverPct) {
				best = l
			}
		}
		r.Province.Reason = fmt.Sprintf("none: no level from 3 up covers %.0f %% of the land plots with at most %.0f %% overlap", minCoverPct, maxOverlapPct)
		if best != nil {
			r.Province.Reason += fmt.Sprintf(" (best: level %d, %s, %.1f %%)", best.Level, relations(best.N), best.CoverPct)
		}
		r.District.Reason = "none: no province level"
		return
	}
	r.Province.Level = prov.Level
	r.Province.Reason = s.reason(prov)

	for _, l := range r.Levels {
		if l.Level > prov.Level && candidate(l) && l.N > prov.N {
			r.District.Level = l.Level
			r.District.Reason = s.reason(l)
			break
		}
	}
	if r.District.Level == 0 {
		r.District.Reason = fmt.Sprintf("none: no level below %d covers %.0f %% of the land plots with more relations than the provinces", prov.Level, minCoverPct)
		return
	}
	s.children()
}

// reason is the sentence that backs a proposed level.
func (s *scanner) reason(l *Level) string {
	iso := ""
	switch {
	case l.ISOHere == l.N && l.N > 0:
		iso = fmt.Sprintf(", all with ISO 3166-2 %s-xx", s.cc)
	case l.ISOHere > 0:
		iso = fmt.Sprintf(", %d with ISO 3166-2 %s-xx", l.ISOHere, s.cc)
	default:
		iso = ", none with an ISO 3166-2 code"
	}
	return fmt.Sprintf("level %d: %s%s, covering %.1f %% of the country's land plots (%.2f %% overlap, %s plots in none)",
		l.Level, relations(l.N), iso, l.CoverPct, l.OverlapPct, num(l.Gaps))
}

// index builds a region index over the simplified shapes of recs, numbered
// by their position.
func index(recs []shapeRec) *geo.RegionIndex {
	shapes := make([]geo.RegionShape, len(recs))
	for i, r := range recs {
		shapes[i] = geo.RegionShape{ID: i, Geometry: r.geom}
	}
	return geo.NewRegionIndex(shapes)
}

// parents returns, for each district, the position of the province holding
// its centre, or -1.
func parents(prov, dist []shapeRec) []int {
	idx := index(prov)
	out := make([]int, len(dist))
	for i, d := range dist {
		if p, ok := idx.Contains(d.centre); ok {
			out[i] = p
		} else {
			out[i] = -1
		}
	}
	return out
}

// children counts the districts per province.
func (s *scanner) children() {
	prov, dist := s.recs[s.r.Province.Level], s.recs[s.r.District.Level]
	per := make([]int, len(prov))
	orphans := 0
	for _, p := range parents(prov, dist) {
		if p < 0 {
			orphans++
		} else {
			per[p]++
		}
	}
	d := &s.r.District
	d.Orphans = orphans
	if len(per) == 0 {
		return
	}
	sort.Ints(per)
	d.ChildMin, d.ChildMedian, d.ChildMax = per[0], per[len(per)/2], per[len(per)-1]
}

// names reads the names of the proposed levels.
func (s *scanner) names() {
	for _, p := range []Proposal{s.r.Province, s.r.District} {
		if p.Level == 0 {
			continue
		}
		recs := s.recs[p.Level]
		var par []int
		if p.Role == "district" {
			par = parents(s.recs[s.r.Province.Level], recs)
		}
		n := &NameReport{Role: p.Role, Level: p.Level, N: len(recs), EnglishKeys: map[string]int{}, LocalKeys: map[string]int{}}
		var ens, locs []string
		slugs := map[string]int{}
		for i, rec := range recs {
			en, ek := englishName(rec.rel.tags, s.lang)
			loc, lk := localName(rec.rel.tags, s.lang)
			if en == "" {
				n.NoEnglish++
			} else {
				n.EnglishKeys[ek]++
				ens = append(ens, en)
			}
			if loc == "" {
				n.NoLocal++
			} else {
				n.LocalKeys[lk]++
				locs = append(locs, loc)
			}
			if en != "" && en == loc {
				n.SameAsLocal++
			}
			slug := osmfile.Slug(en)
			if slug == "" {
				n.EmptySlug++
				continue
			}
			if plain := plainSlug(en); plain != slug {
				n.AccentLoss++
				if len(n.AccentEx) < 3 {
					n.AccentEx = append(n.AccentEx, fmt.Sprintf("%s: %s → %s", en, plain, slug))
				}
			}
			key := slug
			if par != nil {
				key = fmt.Sprintf("%d/%s", par[i], slug)
			}
			slugs[key]++
		}
		for k, c := range slugs {
			if c > 1 {
				n.Duplicates = append(n.Duplicates, fmt.Sprintf("%s (%d)", k, c))
			}
		}
		sort.Strings(n.Duplicates)
		n.Prefixes, n.Suffixes = affixes(ens)
		n.LocalPre, n.LocalSuf = affixes(locs)
		s.r.Names = append(s.r.Names, n)
	}
}

// size estimates stage 3 from the land plots and the Thailand measurements.
func (s *scanner) size() {
	r := s.r
	plots := r.LandCells
	if r.Res < 8 {
		plots *= pow7(8 - r.Res)
	}
	r.Size = Size{
		LandPlots:  plots,
		Estimated:  r.Res < 8,
		Seconds:    max(1, plots*ThailandSeconds/ThailandPlots),
		MiB:        max(100, plots*ThailandMiB/ThailandPlots),
		NumberRows: plots,
	}
}

func pow7(n int) int {
	out := 1
	for i := 0; i < n; i++ {
		out *= 7
	}
	return out
}

// relations is "1 relation", "3 relations".
func relations(n int) string {
	if n == 1 {
		return "1 relation"
	}
	return num(n) + " relations"
}

// join lists "a, b, c".
func join(ss []string) string { return strings.Join(ss, ", ") }
