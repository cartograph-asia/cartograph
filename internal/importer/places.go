package importer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/paulmach/orb"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// DefaultPlaceReachKm is the reach of a country's places when its settings
// leave PlaceReachKm at 0.
const DefaultPlaceReachKm = 2.0

// placeClass is how a place node relates to the run's provinces.
type placeClass int

const (
	placeStrong placeClass = iota // strictly inside one of the run's provinces (legacy: located in one)
	placeWeak                     // inside no province, within reach of one of the run's (legacy: all others)
	placeOther                    // inside another country's province
	placeFar                      // farther than the reach from every province of the run
)

// PlaceStats reports what Places did (or, in a dry run, would do).
type PlaceStats struct {
	Places          int // places stored for the country
	OutsideRegions  int // of them in no province (kept, region_id null)
	Strong          int // of them strictly inside one of the run's provinces (legacy: located in one)
	Weak            int // of them outside every province, within reach of one of the run's (legacy: all others)
	Added           int
	Updated         int
	Removed         int
	TakenOver       int // of Added and Updated: weak places of another country, strong for the run
	SkippedOther    int // inside another country's province
	SkippedStored   int // stored for another country inside its provinces (the guard)
	SkippedWeak     int // weak for the run and stored weak for another country, left to it
	SkippedFar      int // farther than the reach from every province of the run
	PlotsBackfilled int
}

// Places stores one country's place nodes (the settings' place kinds) of
// the extract (inside BBox if set), then fills place names of the country's
// plots that have none. Run it after Regions. A place strictly inside one
// of the country's provinces is stored with it ("strong"); one inside no
// province but within the settings' PlaceReachKm of one of the country's
// provinces is stored without a province ("weak"); one inside another
// country's province, or farther away, is not stored. A strong place takes
// over another country's weak one; a strong place of another country, and
// a weak one when the run's claim is weak too, are never changed or
// deleted. Thailand (LegacyPlaces) keeps the rule it was imported with:
// every place, located with geo.NearTolerance, none outside a province.
func Places(ctx context.Context, d *db.DB, o PlaceOptions) (*PlaceStats, error) {
	return places(ctx, d, o, nil)
}

// places is Places with the regions of the same run (rp, nil when the
// regions are not part of it): a dry run then locates places in the
// combined index of rp, since its provinces are not stored.
func places(ctx context.Context, d *db.DB, o Options, rp *regionPlan) (*PlaceStats, error) {
	c, err := resolveCountry(o)
	if err != nil {
		return nil, err
	}
	if c.Imp.Hold != "" {
		return nil, fmt.Errorf("importer: %s: %w: 1 hold (%s)", c.Code, ErrChecks, c.Imp.Hold)
	}
	clip, err := geo.ParseBound(o.BBox)
	if err != nil {
		return nil, err
	}
	in, err := input(o, c)
	if err != nil {
		return nil, err
	}
	if in.Note != "" && rp == nil {
		slog.Warn("input", "country", c.Code, "file", in.Path, "note", in.Note)
	}
	if o.DryRun {
		if err := checkMigrated(ctx, d); err != nil {
			return nil, err
		}
	}
	start := time.Now()
	ps, err := osmfile.ReadPlaces(in.Path, clip, c.Imp)
	if err != nil {
		return nil, err
	}

	// The index places are located in: the plan's while its provinces are
	// not stored (a dry run), else the stored provinces in id order, which
	// after Regions is the same index.
	if rp == nil || rp.applied || rp.hold {
		rp = &regionPlan{c: c, o: o, set: &regionSet{}}
		if err := rp.loadStored(ctx, d); err != nil {
			return nil, err
		}
		rp.combinedIndex()
	}
	countryOf := func(id int) string {
		if _, ok := rp.own[id]; ok {
			return c.Code
		}
		return rp.countryOf[id]
	}
	regionOSM := func(id int) int64 {
		if p, ok := rp.own[id]; ok {
			return p.osmID
		}
		if s := rp.byID[id]; s != nil {
			return s.OSMID
		}
		return 0
	}
	storedPlaces, err := d.StoredPlaces(ctx)
	if err != nil {
		return nil, err
	}
	type key struct {
		t  string
		id int64
	}
	stored := make(map[key]*db.StoredPlace, len(storedPlaces))
	for i := range storedPlaces {
		s := &storedPlaces[i]
		stored[key{s.OSMType, s.OSMID}] = s
	}

	reach := c.Imp.PlaceReachKm
	if reach == 0 {
		reach = DefaultPlaceReachKm
	}
	ownProvince := func(id int) bool { _, ok := rp.own[id]; return ok }
	// classify gives a place its class and, when strong, the index id of
	// its province.
	classify := func(pt orb.Point) (placeClass, int) {
		if c.Imp.LegacyPlaces {
			id, ok := rp.idx.Locate(pt)
			switch {
			case !ok:
				return placeWeak, 0
			case countryOf(id) != c.Code:
				return placeOther, 0
			}
			return placeStrong, id
		}
		if id, ok := rp.idx.Contains(pt); ok {
			if countryOf(id) != c.Code {
				return placeOther, 0
			}
			return placeStrong, id
		}
		if _, _, ok := rp.idx.Nearest(pt, reach, ownProvince); ok {
			return placeWeak, 0
		}
		return placeFar, 0
	}

	st := &PlaceStats{}
	var added, updated, removed, takenOver list
	rows := make([]db.Place, 0, len(ps))
	seen := make(map[key]bool, len(ps))
	for _, p := range ps {
		row := db.Place{OSMType: p.OSMType, OSMID: p.OSMID, Kind: p.Kind, NameEn: p.NameEn, NameLocal: p.NameLocal,
			Lat: p.Point[1], Lng: p.Point[0], Country: c.Code}
		class, id := classify(p.Point)
		switch class {
		case placeOther:
			st.SkippedOther++
			continue
		case placeFar:
			st.SkippedFar++
			continue
		}
		var osmRegion int64
		if class == placeStrong {
			osmRegion = regionOSM(id)
			if id > 0 {
				row.RegionID = &id
			}
		}
		k := key{p.OSMType, p.OSMID}
		s := stored[k]
		if s != nil && s.Country != c.Code {
			// Another country's place: a strong one is never taken, a weak
			// one only by a strong claim (the upsert's guard agrees).
			switch {
			case s.RegionID != nil:
				st.SkippedStored++
				continue
			case class == placeWeak:
				st.SkippedWeak++
				continue
			}
			takenOver.add("%s%d %q from %s", p.OSMType, p.OSMID, p.NameEn, s.Country)
		}
		seen[k] = true
		if osmRegion == 0 {
			st.OutsideRegions++
		}
		if class == placeStrong {
			st.Strong++
		} else {
			st.Weak++
		}
		rows = append(rows, row)
		switch {
		case s == nil:
			added.add("%s%d %s %q (%s)", p.OSMType, p.OSMID, p.Kind, p.NameEn, q(p.NameLocal))
		default:
			var storedRegion int64
			if s.RegionID != nil {
				if r := rp.byID[*s.RegionID]; r != nil {
					storedRegion = r.OSMID
				}
			}
			var ch []string
			if s.Kind != row.Kind {
				ch = append(ch, fmt.Sprintf("kind %s → %s", s.Kind, row.Kind))
			}
			if s.NameEn != row.NameEn {
				ch = append(ch, fmt.Sprintf("name_en %q → %q", s.NameEn, row.NameEn))
			}
			if s.NameLocal != row.NameLocal {
				ch = append(ch, fmt.Sprintf("name_local %s → %s", q(s.NameLocal), q(row.NameLocal)))
			}
			if s.Lat != row.Lat || s.Lng != row.Lng {
				ch = append(ch, fmt.Sprintf("position %.7f,%.7f → %.7f,%.7f", s.Lat, s.Lng, row.Lat, row.Lng))
			}
			if storedRegion != osmRegion {
				ch = append(ch, fmt.Sprintf("province r%d → r%d", storedRegion, osmRegion))
			}
			if s.Country != c.Code {
				ch = append(ch, fmt.Sprintf("country %s → %s", s.Country, c.Code))
			}
			if len(ch) > 0 {
				updated.add("%s%d %q: %v", p.OSMType, p.OSMID, p.NameEn, ch)
			}
		}
	}
	for _, s := range storedPlaces {
		if s.Country == c.Code && !seen[key{s.OSMType, s.OSMID}] {
			removed.add("%s%d %q", s.OSMType, s.OSMID, s.NameEn)
		}
	}
	st.Places, st.Added, st.Updated, st.Removed, st.TakenOver = len(rows), added.n, updated.n, removed.n, takenOver.n
	if o.Report != nil {
		writePlaces(o.Report, o.DryRun, c, len(ps), reach, st, &added, &updated, &removed, &takenOver)
	}
	if o.DryRun {
		return st, nil
	}
	if err := d.ReplacePlaces(ctx, c.Code, rows); err != nil {
		return st, err
	}
	if err := d.ReloadRegions(ctx); err != nil {
		return st, err
	}
	if st.PlotsBackfilled, _, err = d.BackfillPlotsOf(ctx, c.Code); err != nil {
		return st, err
	}
	slog.Info("imported places", "country", c.Code, "places", st.Places, "strong", st.Strong, "weak", st.Weak,
		"taken_over", st.TakenOver, "skipped_other_country", st.SkippedOther+st.SkippedStored+st.SkippedWeak,
		"skipped_far", st.SkippedFar, "plots_backfilled", st.PlotsBackfilled,
		"took", time.Since(start).Round(time.Millisecond))
	return st, nil
}

func writePlaces(w io.Writer, dry bool, c country, read int, reach float64, st *PlaceStats, added, updated, removed, takenOver *list) {
	mode := "Run"
	if dry {
		mode = "Dry run"
	}
	fmt.Fprintf(w, "\n## %s: places of %s (%s)\n\n", mode, c.Name, c.Code)
	if c.Imp.LegacyPlaces {
		fmt.Fprintf(w, "Places read: %d; stored for %s: %d (%d in or within %g° of one of its provinces, %d outside every province; legacy rule)\n",
			read, c.Code, st.Places, st.Strong, geo.NearTolerance, st.Weak)
		fmt.Fprintf(w, "Places not stored: %d inside another country's province, %d stored for another country inside its provinces, %d stored for another country outside every province\n",
			st.SkippedOther, st.SkippedStored, st.SkippedWeak)
	} else {
		fmt.Fprintf(w, "Places read: %d; stored for %s: %d (%d strong, inside one of its provinces; %d weak, outside every province and within %g km of one of its provinces)\n",
			read, c.Code, st.Places, st.Strong, st.Weak, reach)
		fmt.Fprintf(w, "Places not stored: %d inside another country's province, %d stored for another country inside its provinces, %d weak and stored weak for another country, %d farther than %g km\n",
			st.SkippedOther, st.SkippedStored, st.SkippedWeak, st.SkippedFar, reach)
	}
	takenOver.write(w, "Places taken over (weak for another country, strong for "+c.Code+")")
	added.write(w, "Places added")
	updated.write(w, "Places updated")
	removed.write(w, "Places removed")
}
