package countryscan

import (
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/paulmach/orb"
	orbgeo "github.com/paulmach/orb/geo"
	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/geo"
)

// tagStats counts the tags of rels.
func (s *scanner) tagStats(rels []*lrel) TagStats {
	ts := TagStats{N: len(rels), Prefixes: map[string]int{}, Scripts: map[string]int{}, NameKeys: map[string]int{}}
	type named struct{ sort, show string }
	var names []named
	for _, rl := range rels {
		t := rl.tags
		if v := strings.TrimSpace(t["ISO3166-2"]); v != "" {
			ts.ISO++
			if s.iso(t) == isoHere {
				ts.ISOHere++
			}
			seen := map[string]bool{}
			for _, x := range strings.Split(v, ";") {
				p, _, _ := strings.Cut(strings.TrimSpace(x), "-")
				if p = strings.ToUpper(p); p != "" && !seen[p] {
					seen[p] = true
					ts.Prefixes[p]++
				}
			}
		}
		if strings.TrimSpace(t["name:en"]) != "" {
			ts.NameEn++
		}
		if strings.TrimSpace(t["int_name"]) != "" {
			ts.IntName++
		}
		if n := strings.TrimSpace(t["name"]); n != "" {
			ts.HasName++
			if isLatin(n) {
				ts.LatinName++
			}
			if sc := scriptOf(n); sc != "" {
				ts.Scripts[sc]++
			}
		}
		for k := range t {
			if strings.HasPrefix(k, "name:") && strings.TrimSpace(t[k]) != "" {
				ts.NameKeys[k]++
			}
		}
		en, _ := englishName(t, s.lang)
		loc, _ := localName(t, s.lang)
		if en != "" {
			ts.English++
		}
		show := en
		switch {
		case en == "":
			show = loc
		case loc != "" && loc != en:
			show = en + " (" + loc + ")"
		}
		if show != "" {
			names = append(names, named{strings.ToLower(show) + "\x00" + strconv.FormatInt(rl.id, 10), show})
		}
	}
	sort.Slice(names, func(i, j int) bool { return names[i].sort < names[j].sort })
	const want = 10
	if len(names) <= want {
		for _, n := range names {
			ts.Examples = append(ts.Examples, n.show)
		}
	} else {
		for i := 0; i < want; i++ {
			ts.Examples = append(ts.Examples, names[i*(len(names)-1)/(want-1)].show)
		}
	}
	return ts
}

// level reads one admin_level: its tags, and for a level small enough also
// its geometry, how it covers the country's land plots and where the
// relations lie.
func (s *scanner) level(lv int) error {
	rs := s.byLvl[lv]
	if len(rs) == 0 {
		return nil
	}
	L := &Level{Level: lv, Extract: len(rs), Scope: "extract"}
	s.r.Levels = append(s.r.Levels, L)

	// Level 2 holds the countries of the extract; its geometry is the outline.
	if lv == MinLevel || len(rs) > s.o.GeometryCap {
		L.CapExceeded = lv != MinLevel
		var scope []*lrel
		for _, rl := range rs {
			if s.iso(rl.tags) != isoOther {
				scope = append(scope, rl)
			} else {
				L.OtherISO++
			}
		}
		L.TagStats = s.tagStats(scope)
		return nil
	}

	t := time.Now()
	want := make(map[osm.RelationID]bool, len(rs))
	for _, rl := range rs {
		want[osm.RelationID(rl.id)] = true
	}
	bs, skipped, err := s.build(func(r *osm.Relation) bool { return want[r.ID] })
	if err != nil {
		return err
	}
	L.SkippedAny = len(skipped)
	skippedSet := map[int64]bool{}
	for _, id := range skipped {
		skippedSet[id] = true
	}

	// Decide, relation by relation, whether it belongs to the country.
	var scope []*lrel
	var recs []shapeRec
	for _, rl := range rs {
		st := s.iso(rl.tags)
		if st == isoOther {
			L.OtherISO++
			continue
		}
		b, built := bs[rl.id]
		if !built {
			if st == isoHere {
				scope = append(scope, rl)
				L.Skipped++
				if len(L.SkippedList) < 10 {
					n, _ := englishName(rl.tags, s.lang)
					L.SkippedList = append(L.SkippedList, strings.TrimSpace(fmt.Sprintf("r%d %s", rl.id, n)))
				}
			} else {
				L.Outside++ // cannot be placed; counted with the outside
			}
			continue
		}
		simple := geo.Simplify(b.Geometry, geo.SimplifyTolerance)
		if len(simple) == 0 {
			if st == isoHere {
				scope = append(scope, rl)
				L.Skipped++
			}
			continue
		}
		c := centreOf(simple)
		if st != isoHere {
			id, err := cellOf(c, s.o.Res)
			if err != nil || !cellIn(s.inCountry, id) {
				L.Outside++
				continue
			}
		}
		scope = append(scope, rl)
		recs = append(recs, shapeRec{rel: rl, geom: simple, centre: c, areaKm: orbgeo.Area(b.Geometry) / 1e6})
	}
	L.Scope, L.Geometry = "country", true
	L.TagStats = s.tagStats(scope)

	// Which land plots each relation holds, in parallel.
	lists := make([][]uint32, len(recs))
	errs := make([]error, len(recs))
	parallelFor(len(recs), s.o.Workers, func(i int) {
		ids, err := polyCells(recs[i].geom, s.o.Res, 1)
		if err != nil {
			errs[i] = err
			return
		}
		for _, id := range ids {
			if j, ok := slices.BinarySearch(s.land, id); ok {
				lists[i] = append(lists[i], uint32(j))
			}
		}
	})
	cover := make([]uint8, len(s.land))
	for i := range recs {
		if errs[i] != nil {
			L.FillErrors++
			slog.Warn("could not fill a boundary", "relation", recs[i].rel.id, "err", errs[i])
			continue
		}
		recs[i].land = len(lists[i])
		for _, j := range lists[i] {
			if cover[j] < 255 {
				cover[j]++
			}
		}
		L.AreaKm2 += recs[i].areaKm
	}
	for _, c := range cover {
		switch {
		case c == 0:
			L.Gaps++
		case c == 1:
			L.Covered++
		default:
			L.Covered++
			L.Overlaps++
		}
	}
	if n := len(s.land); n > 0 {
		L.CoverPct = 100 * float64(L.Covered) / float64(n)
		L.GapPct = 100 * float64(L.Gaps) / float64(n)
		L.OverlapPct = 100 * float64(L.Overlaps) / float64(n)
	}
	if a := s.r.Shape.AreaKm2; a > 0 {
		L.AreaPct = 100 * L.AreaKm2 / a
	}
	var lands []int
	for _, r := range recs {
		lands = append(lands, r.land)
		if r.land == 0 {
			L.NoLand++
		}
	}
	if len(lands) > 0 {
		sort.Ints(lands)
		L.LandMin, L.LandMedian, L.LandMax = lands[0], lands[len(lands)/2], lands[len(lands)-1]
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].rel.id < recs[j].rel.id })
	s.recs[lv] = recs
	slog.Info("read level", "country", s.o.Country.Code, "level", lv, "relations", len(rs), "in_country", len(scope),
		"cover_pct", fmt.Sprintf("%.1f", L.CoverPct), "took", time.Since(t).Round(time.Millisecond))
	return nil
}

// cellOf returns the cell containing p.
func cellOf(p orb.Point, res int) (uint64, error) { return cellID(p[1], p[0], res) }

// levelOf returns the scanned level lv, or nil.
func (r *Result) levelOf(lv int) *Level {
	for _, l := range r.Levels {
		if l.Level == lv {
			return l
		}
	}
	return nil
}
