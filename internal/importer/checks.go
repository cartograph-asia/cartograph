package importer

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cartograph-asia/cartograph/internal/geo"
)

// Check is one of the seven checks every run makes before it writes
// (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, Dry run).
type Check struct {
	Num     int
	Name    string
	Status  string // "ok", "FAILED", "skipped" (with -bbox) or "not run" (after a hold)
	Details []string
}

// Failed reports whether the check failed.
func (c Check) Failed() bool { return c.Status == "FAILED" }

var checkNames = [...]string{
	1: "hold",
	2: "province count",
	3: "slugs",
	4: "land plots",
	5: "provinces without land plots",
	6: "province relations that do not assemble",
	7: "provinces stored for another country",
}

func check(n int, failed bool, details ...string) Check {
	st := "ok"
	if failed {
		st = "FAILED"
	}
	return Check{Num: n, Name: checkNames[n], Status: st, Details: details}
}

func skipped(n int, why string) Check {
	return Check{Num: n, Name: checkNames[n], Status: "skipped", Details: []string{why}}
}

// holdChecks are the checks of a country whose settings carry a Hold: check
// 1 fails, the others are not run.
func holdChecks(c country) []Check {
	out := []Check{check(1, true, c.Imp.Hold)}
	for n := 2; n <= 7; n++ {
		out = append(out, Check{Num: n, Name: checkNames[n], Status: "not run"})
	}
	return out
}

// validSlug is the regions.slug check constraint.
var validSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// runChecks makes checks 1 to 7 on a plan.
func (pl *regionPlan) runChecks() []Check {
	imp, ps := pl.c.Imp, pl.set.provinces
	clipped := !geo.NoBound(pl.clip)
	const withBBox = "with -bbox"
	out := []Check{check(1, false)}

	// 2: province count.
	switch n, lim := len(ps), imp.Province.Count; {
	case n == 0:
		out = append(out, check(2, true, "no provinces found"))
	case clipped:
		out = append(out, skipped(2, withBBox))
	case lim[1] == 0:
		out = append(out, check(2, true, fmt.Sprintf("%d provinces; the settings give no Count", n)))
	default:
		out = append(out, check(2, n < lim[0] || n > lim[1], fmt.Sprintf("%d provinces, settings %d–%d", n, lim[0], lim[1])))
	}

	// 3: slugs. A new slug must be valid, unique in the run and not used by
	// a stored row of another relation; a stored slug is kept.
	var bad []string
	storedSlug := make(map[string]int64, len(pl.stored))
	for _, s := range pl.stored {
		storedSlug[s.Slug] = s.OSMID
	}
	seen := map[string]int64{}
	slugCheck := func(kind string, osmID int64, slug string, isNew bool) {
		if other, dup := seen[slug]; dup {
			bad = append(bad, fmt.Sprintf("%s r%d: slug %q also given to r%d", kind, osmID, slug, other))
		}
		seen[slug] = osmID
		if !isNew {
			return
		}
		if !validSlug.MatchString(slug) {
			bad = append(bad, fmt.Sprintf("%s r%d: slug %q is not valid (no English name?)", kind, osmID, slug))
		}
		if other, ok := storedSlug[slug]; ok && other != osmID {
			bad = append(bad, fmt.Sprintf("%s r%d: slug %q is stored for r%d", kind, osmID, slug, other))
		}
	}
	for _, p := range ps {
		slugCheck("province", p.osmID, p.slug, p.stored == nil || p.stored.Country != pl.c.Code)
	}
	for _, d := range pl.set.districts {
		slugCheck("district", d.osmID, d.slug, d.stored == nil)
	}
	out = append(out, check(3, len(bad) > 0, bad...))

	// 4: land plots against the scan's figure.
	land := 0
	for _, p := range ps {
		land += p.land
	}
	switch {
	case clipped:
		out = append(out, skipped(4, withBBox))
	case imp.LandPlots <= 0:
		out = append(out, check(4, true, fmt.Sprintf("%d land plots; the settings give no LandPlots", land)))
	default:
		off := float64(land-imp.LandPlots) / float64(imp.LandPlots)
		out = append(out, check(4, math.Abs(off) > 0.10, fmt.Sprintf("%d land plots, settings %d (%+.2f %%)", land, imp.LandPlots, 100*off)))
	}

	// 5: provinces without land plots.
	if clipped {
		out = append(out, skipped(5, withBBox))
	} else {
		var empty []string
		for _, p := range ps {
			if p.land == 0 {
				empty = append(empty, fmt.Sprintf("r%d %s", p.osmID, p.slug))
			}
		}
		out = append(out, check(5, len(empty) > 0, empty...))
	}

	// 6: province-level relations in scope that do not assemble, and the
	// neighbours' cuts that should complete them but are missing.
	var na []string
	for _, m := range pl.nb.missing {
		name := strings.TrimSuffix(filepath.Base(m), "-latest.osm.pbf.cut.osm.pbf")
		na = append(na, fmt.Sprintf("neighbour cut %s is missing (Neighbours %q; make it with map/cut.sh %s)", m, name, name))
	}
	for _, id := range pl.set.notAssembled {
		na = append(na, fmt.Sprintf("r%d (not in Partial)", id))
		if containsID(imp.Partial, id) {
			na[len(na)-1] = fmt.Sprintf("r%d (in Partial, but no ring closes)", id)
		}
	}
	out = append(out, check(6, len(na) > 0, na...))

	// 7: province relations stored for another country.
	var other []string
	for _, p := range ps {
		if p.stored != nil && p.stored.Country != pl.c.Code {
			other = append(other, fmt.Sprintf("r%d %s is stored for %q as %s", p.osmID, p.names.En, p.stored.Country, p.stored.Slug))
		}
	}
	out = append(out, check(7, len(other) > 0, other...))
	return out
}

func containsID(ids []int64, id int64) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// checksError returns ErrChecks naming the failed checks, or nil.
func checksError(c country, checks []Check) error {
	var failed []string
	for _, ch := range checks {
		if ch.Failed() {
			s := fmt.Sprintf("%d %s", ch.Num, ch.Name)
			if len(ch.Details) > 0 {
				d := ch.Details
				if len(d) > 3 {
					d = append(d[:3:3], fmt.Sprintf("and %d more", len(ch.Details)-3))
				}
				s += " (" + strings.Join(d, "; ") + ")"
			}
			failed = append(failed, s)
		}
	}
	if len(failed) == 0 {
		return nil
	}
	sort.Strings(failed)
	return fmt.Errorf("importer: %s: %w: %s", c.Code, ErrChecks, strings.Join(failed, ", "))
}
