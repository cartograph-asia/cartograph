package importer

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// maxLines is how many changes of one kind the report lists by name.
const maxLines = 40

// list collects the lines of one part of the report, keeping the first
// maxLines and counting the rest.
type list struct {
	n     int
	lines []string
}

func (l *list) add(format string, args ...any) {
	l.n++
	if len(l.lines) < maxLines {
		l.lines = append(l.lines, fmt.Sprintf(format, args...))
	}
}

func (l *list) write(w io.Writer, title string) {
	fmt.Fprintf(w, "%s: %d\n", title, l.n)
	for _, s := range l.lines {
		fmt.Fprintf(w, "  - %s\n", s)
	}
	if l.n > len(l.lines) {
		fmt.Fprintf(w, "  - … and %d more\n", l.n-len(l.lines))
	}
}

func q(s string) string {
	if s == "" {
		return "NULL"
	}
	return fmt.Sprintf("%q", s)
}

// regionDiff lists what a run changes in one stored region row.
func regionDiff(s *db.StoredRegion, kind, kindName string, level int, names osmfile.Named, parentOSM int64, storedParentOSM int64,
	land int, boundary string) []string {
	var ch []string
	if s.Kind != kind {
		ch = append(ch, fmt.Sprintf("kind %s → %s", s.Kind, kind))
	}
	if s.NameEn != names.En {
		ch = append(ch, fmt.Sprintf("name_en %q → %q", s.NameEn, names.En))
	}
	if s.NameLocal != names.Local {
		ch = append(ch, fmt.Sprintf("name_local %s → %s", q(s.NameLocal), q(names.Local)))
	}
	if s.KindName != kindName {
		ch = append(ch, fmt.Sprintf("kind_name %s → %s", q(s.KindName), q(kindName)))
	}
	if s.AdminLevel != level {
		ch = append(ch, fmt.Sprintf("admin_level %d → %d", s.AdminLevel, level))
	}
	if parentOSM != storedParentOSM {
		ch = append(ch, fmt.Sprintf("province r%d → r%d", storedParentOSM, parentOSM))
	}
	if s.LandCells != land {
		ch = append(ch, fmt.Sprintf("land_cells %d → %d", s.LandCells, land))
	}
	if s.Boundary != boundary {
		ch = append(ch, "boundary")
	}
	return ch
}

// writeReport writes the checks and what the run changes, compared with
// the stored rows.
func (pl *regionPlan) writeReport(w io.Writer) {
	mode := "Run"
	if pl.o.DryRun {
		mode = "Dry run"
	}
	fmt.Fprintf(w, "\n## %s: regions of %s (%s)\n\n", mode, pl.c.Name, pl.c.Code)
	if pl.in.Path != "" {
		kind := "file"
		if pl.in.Cut {
			kind = "cut"
		}
		fmt.Fprintf(w, "Input: %s (%s)\n", pl.in.Path, kind)
		if pl.in.Note != "" {
			fmt.Fprintf(w, "Input note: %s\n", pl.in.Note)
		}
	}
	// Only for a country with Neighbours, so other reports stay as they were.
	for _, in := range pl.nb.read {
		kind := "file"
		if in.Cut {
			kind = "cut"
		}
		fmt.Fprintf(w, "Neighbour read for boundaries only: %s (%s)\n", in.Path, kind)
		if in.Note != "" {
			fmt.Fprintf(w, "Neighbour note: %s\n", in.Note)
		}
	}
	for _, m := range pl.nb.missing {
		fmt.Fprintf(w, "Neighbour missing: %s (check 6)\n", m)
	}
	if pl.nb.skipped != "" {
		fmt.Fprintf(w, "Neighbours not read (%s): %s\n", pl.nb.skipped, strings.Join(pl.c.Imp.Neighbours, ", "))
	}
	if !geo.NoBound(pl.clip) {
		fmt.Fprintf(w, "Clip: %s\n", pl.o.BBox)
	}
	fmt.Fprintln(w, "\nChecks:")
	for _, c := range pl.checks {
		fmt.Fprintf(w, "  %d. %-42s %s\n", c.Num, c.Name, c.Status)
		for _, d := range c.Details {
			fmt.Fprintf(w, "       %s\n", d)
		}
	}
	if pl.hold {
		return
	}
	set := pl.set
	osmOf := func(id *int) int64 {
		if id == nil {
			return 0
		}
		if s := pl.byID[*id]; s != nil {
			return s.OSMID
		}
		return 0
	}

	var pAdded, pUpdated, dAdded, dUpdated, slugs, renamed, byLand, parentKept list
	pSame, dSame, landChanged := 0, 0, 0
	kinds := map[string]int{}
	land, held, newNumbers, inDistrict := 0, 0, 0, 0
	for _, p := range set.provinces {
		land, held, newNumbers, inDistrict = land+p.land, held+p.held, newNumbers+p.newNumbers, inDistrict+p.inDistrict
		kinds["province "+q(p.kindName)]++
		text, _ := geo.EncodeGeometry(p.shape)
		s := p.stored
		if s == nil || s.Country != pl.c.Code {
			pAdded.add("r%d %s %q (%s), %d land plots", p.osmID, p.slug, p.names.En, q(p.names.Local), p.land)
			slugs.add("province r%d: %s", p.osmID, p.slug)
			continue
		}
		if s.LandCells != p.land {
			landChanged++
		}
		if ch := regionDiff(s, "province", p.kindName, p.level, p.names, 0, 0, p.land, text); len(ch) > 0 {
			pUpdated.add("r%d %s: %s", p.osmID, p.slug, strings.Join(ch, ", "))
		} else {
			pSame++
		}
		if fresh := osmfile.Slug(p.names.En); s.Slug != fresh && !strings.HasPrefix(s.Slug, fresh) {
			renamed.add("province r%d keeps %s (its name now gives %s)", p.osmID, s.Slug, fresh)
		}
	}
	for _, d := range set.districts {
		kinds["district "+q(d.kindName)]++
		text, _ := geo.EncodeGeometry(d.shape)
		s := d.stored
		if d.majority != nil && d.centred != nil && d.majority != d.centred {
			byLand.add("r%d %s %q: land cells in r%d %s, centre in r%d %s", d.osmID, d.slug, d.names.En,
				d.majority.osmID, d.majority.names.En, d.centred.osmID, d.centred.names.En)
		}
		if s != nil && d.majority != nil && d.majority != d.province {
			parentKept.add("r%d %s %q: stays in r%d %s, a fresh import gives r%d %s", d.osmID, d.slug, d.names.En,
				d.province.osmID, d.province.names.En, d.majority.osmID, d.majority.names.En)
		}
		if s == nil {
			dAdded.add("r%d %s %q (%s)", d.osmID, d.slug, d.names.En, q(d.names.Local))
			slugs.add("district r%d: %s", d.osmID, d.slug)
			continue
		}
		parent := d.province.osmID
		if ch := regionDiff(s, "district", d.kindName, d.level, d.names, parent, osmOf(s.ParentID), 0, text); len(ch) > 0 {
			dUpdated.add("r%d %s: %s", d.osmID, d.slug, strings.Join(ch, ", "))
		} else {
			dSame++
		}
	}
	run := map[int64]bool{}
	for _, p := range set.provinces {
		run[p.osmID] = true
	}
	for _, d := range set.districts {
		run[d.osmID] = true
	}
	var gone list
	for _, s := range pl.stored {
		if s.Country == pl.c.Code && !run[s.OSMID] {
			gone.add("%s r%d %s", s.Kind, s.OSMID, s.Slug)
		}
	}

	fmt.Fprintf(w, "\nProvinces: %d (%d added, %d updated, %d unchanged)\n", len(set.provinces), pAdded.n, pUpdated.n, pSame)
	pAdded.write(w, "Provinces added")
	pUpdated.write(w, "Provinces updated")
	fmt.Fprintf(w, "Districts: %d (%d added, %d updated, %d unchanged; settings count %v, reported only)\n",
		len(set.districts), dAdded.n, dUpdated.n, dSame, pl.c.Imp.District.Count)
	dAdded.write(w, "Districts added")
	dUpdated.write(w, "Districts updated")
	dr := set.dropped
	fmt.Fprintf(w, "Districts dropped: no English name %d, centre outside the provinces %d, not assembled %d, stored for another country %d, outside the parent %d\n",
		dr.NoEnglish, dr.Outside, dr.NotAssembled, dr.OtherCountry, dr.OutsideParent)
	if len(pl.c.Imp.Province.Exclude) > 0 || len(pl.c.Imp.District.Exclude) > 0 {
		fmt.Fprintf(w, "Relations excluded: %d, by settings (Level.Exclude) %v\n", len(set.excluded), set.excluded)
	}
	byLand.write(w, "Districts placed by their land cells, not their centre")
	parentKept.write(w, "Districts whose parent would change (information only: a stored district keeps its parent and slug)")
	gone.write(w, "Stored rows of this country not in this run (kept as they are)")
	slugs.write(w, "New slugs")
	renamed.write(w, "Renamed relations keeping their stored slug")
	var ks []string
	for k, n := range kinds {
		ks = append(ks, fmt.Sprintf("%s %d", k, n))
	}
	sort.Strings(ks)
	fmt.Fprintf(w, "Kind words (kind kind_name count): %s\n", strings.Join(ks, ", "))
	fmt.Fprintf(w, "Land plots: %d (settings %d); provinces whose land_cells change: %d\n", land, pl.c.Imp.LandPlots, landChanged)
	fmt.Fprintf(w, "New plot numbers: %d\n", newNumbers)
	fmt.Fprintf(w, "Border cells skipped (held by another country): %d\n", held)
	if land > 0 {
		fmt.Fprintf(w, "District cover: %.1f %% of the land plots lie in a district\n", 100*float64(inDistrict)/float64(land))
	}
	if len(set.partial) > 0 {
		ids := make([]int64, 0, len(set.partial))
		for id := range set.partial {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		fmt.Fprintln(w, "Built from their closed rings only (Partial):")
		for _, id := range ids {
			x := set.partial[id]
			fmt.Fprintf(w, "  - r%d: %d rings built, %d fragments dropped, %d member ways missing\n", id, x.RingsBuilt, x.FragmentsDropped, x.MissingWays)
		}
	}
	if set.outsideParent > 0 || len(set.scopeUnknown) > 0 {
		fmt.Fprintf(w, "Province-level relations outside the parent: %d; without geometry and of unknown scope: %v\n", set.outsideParent, set.scopeUnknown)
	}
	if len(set.vanished) > 0 {
		fmt.Fprintf(w, "Relations that vanished when simplified: %v\n", set.vanished)
	}
}
