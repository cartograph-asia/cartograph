// Package importer loads the OSM-derived tables of one country at a time:
// regions (cmd/regions), places (cmd/places), or both (cmd/import), read by
// the country's settings (internal/countries). A run touches only its own
// country's rows (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, stage 3); it checks
// first and refuses when a check fails, and a dry run only reads and
// reports. Both imports are idempotent and end by backfilling plots that
// lack a region or place name.
package importer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/h3"
)

// FixtureBBox is the area of testdata/pattaya.osm.gz and
// testdata/pattaya-water.shp (see testdata/README.md).
const FixtureBBox = "100.82,12.82,100.98,13.02"

// Fixture returns the paths of the fixture extract and water polygons in
// the source tree (for tests; it relies on the source location).
func Fixture() (osmPath, waterPath string) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "..", "testdata")
	return filepath.Join(dir, "pattaya.osm.gz"), filepath.Join(dir, "pattaya-water.shp")
}

// RegionStats reports what Regions did (or, in a dry run, would do).
type RegionStats struct {
	Regions         int // provinces
	LandCells       int
	Districts       int
	NewNumbers      int // plot numbers added to land cells (dry run: that would be added)
	BorderSkipped   int // land cells inside the run's provinces that another country holds
	PlotsNumbered   int // surveyed plots numbered after the land cells (outside them, or before the import)
	PlotsBackfilled int
	PlotsOutside    int
	Checks          []Check
}

// ErrChecks is returned (wrapped) when a check fails; nothing is written.
var ErrChecks = errors.New("checks failed")

// regionPlan is a run's regions with everything computed from the extract
// and the stored rows; Regions writes it unless it is a dry run.
type regionPlan struct {
	c       country
	o       Options
	in      Input
	nb      neighbours // the neighbours' cuts read for the boundaries
	clip    orb.Bound
	set     *regionSet
	stored  []db.StoredRegion
	byOSM   map[int64]*db.StoredRegion
	byID    map[int]*db.StoredRegion
	hold    bool
	checks  []Check
	applied bool

	// the combined index: stored provinces by id, this run's in place of
	// their stored rows, new ones appended (see landCells)
	idx       *geo.RegionIndex
	own       map[int]*province // index id → this run's province
	countryOf map[int]string    // index id → country, for the other entries
}

// Regions imports the provinces and districts of one country into d:
// checks, upsert by osm_id, land cells, plot numbers, plot backfill,
// claimed_cells/last_surveyed_at recomputed for the country. d must be
// migrated (a dry run refuses when it is not). It returns ErrChecks when a
// check fails, before anything is written.
func Regions(ctx context.Context, d *db.DB, o RegionOptions) (*RegionStats, error) {
	pl, err := planRegions(ctx, d, o)
	if err != nil {
		return nil, err
	}
	return pl.run(ctx, d)
}

// run reports pl, refuses on failed checks, and applies it unless dry.
func (pl *regionPlan) run(ctx context.Context, d *db.DB) (*RegionStats, error) {
	st := pl.stats()
	if pl.o.Report != nil {
		pl.writeReport(pl.o.Report)
	}
	if err := checksError(pl.c, pl.checks); err != nil {
		return st, err
	}
	if pl.o.DryRun {
		return st, nil
	}
	if err := pl.apply(ctx, d, st); err != nil {
		return st, err
	}
	return st, nil
}

// planRegions reads the extract and the stored rows and computes the run;
// it writes nothing.
func planRegions(ctx context.Context, d *db.DB, o Options) (*regionPlan, error) {
	c, err := resolveCountry(o)
	if err != nil {
		return nil, err
	}
	if o.Tolerance == 0 {
		o.Tolerance = geo.SimplifyTolerance
	}
	pl := &regionPlan{c: c, o: o}
	if c.Imp.Hold != "" {
		// Check 1 refuses before anything is read.
		pl.hold = true
		pl.checks = holdChecks(c)
		return pl, nil
	}
	if pl.clip, err = geo.ParseBound(o.BBox); err != nil {
		return nil, err
	}
	if pl.in, err = input(o, c); err != nil {
		return nil, err
	}
	if pl.in.Note != "" {
		slog.Warn("input", "country", c.Code, "file", pl.in.Path, "note", pl.in.Note)
	}
	pl.nb = neighbourInputs(o, c)
	for _, in := range pl.nb.read {
		if in.Note != "" {
			slog.Warn("neighbour input", "country", c.Code, "file", in.Path, "note", in.Note)
		}
	}
	for _, m := range pl.nb.missing {
		slog.Error("neighbour cut missing", "country", c.Code, "file", m)
	}
	if pl.nb.skipped != "" {
		slog.Warn("neighbours not read", "country", c.Code, "neighbours", c.Imp.Neighbours, "why", pl.nb.skipped)
	}
	if o.DryRun {
		if err := checkMigrated(ctx, d); err != nil {
			return nil, err
		}
	}
	start := time.Now()
	if pl.set, err = readRegions(pl.in.Path, pl.nb.paths(), c, pl.clip, o.Tolerance); err != nil {
		return nil, err
	}
	// The boundary store was the peak so far (millions of node positions);
	// hand its pages back before the land cells need theirs.
	debug.FreeOSMemory()
	if err := pl.loadStored(ctx, d); err != nil {
		return nil, err
	}
	if err := pl.landCells(ctx, d); err != nil {
		return nil, err
	}
	keepDistricts(pl.set, c, pl.byOSM)
	assignParents(pl.set, c, pl.byOSM, pl.byID)
	resolveSlugs(pl.set, c, pl.byOSM)
	pl.districtCover()
	pl.checks = pl.runChecks()
	slog.Info("planned regions", "country", c.Code, "provinces", len(pl.set.provinces), "districts", len(pl.set.districts),
		"took", time.Since(start).Round(time.Millisecond))
	return pl, nil
}

// loadStored reads the stored regions.
func (pl *regionPlan) loadStored(ctx context.Context, d *db.DB) error {
	var err error
	if pl.stored, err = d.StoredRegions(ctx); err != nil {
		return err
	}
	pl.byOSM = make(map[int64]*db.StoredRegion, len(pl.stored))
	pl.byID = make(map[int]*db.StoredRegion, len(pl.stored))
	for i := range pl.stored {
		s := &pl.stored[i]
		pl.byOSM[s.OSMID] = s
		pl.byID[s.ID] = s
	}
	return nil
}

// combinedIndex builds the index that decides land cells and places: every
// stored province in id order (the api's order), with this run's own
// country's rows replaced in place by the run's shapes (matched by osm_id),
// and the run's new provinces appended in OSM id order. A cell counts for
// the run only when the index gives it one of the run's provinces, so the
// country imported first keeps a border plot, also when it is re-imported
// later. New provinces get negative ids until they are stored.
func (pl *regionPlan) combinedIndex() {
	run := make(map[int64]*province, len(pl.set.provinces))
	for _, p := range pl.set.provinces {
		run[p.osmID] = p
		p.indexID = 0
	}
	pl.own = map[int]*province{}
	pl.countryOf = map[int]string{}
	var shapes []geo.RegionShape
	for i := range pl.stored {
		s := &pl.stored[i]
		if s.Kind != "province" {
			continue
		}
		if p := run[s.OSMID]; p != nil && s.Country == pl.c.Code {
			p.indexID = s.ID
			pl.own[s.ID] = p
			shapes = append(shapes, geo.RegionShape{ID: s.ID, Geometry: p.shape})
			continue
		}
		pl.countryOf[s.ID] = s.Country
		shapes = append(shapes, geo.RegionShape{ID: s.ID, Geometry: s.Geometry})
	}
	for i, p := range pl.set.provinces {
		if p.indexID == 0 {
			p.indexID = -(i + 1)
			pl.own[p.indexID] = p
			shapes = append(shapes, geo.RegionShape{ID: p.indexID, Geometry: p.shape})
		}
	}
	pl.idx = geo.NewRegionIndex(shapes)
}

// landCells computes every province's land cells in a bounded pool
// (GOMAXPROCS): cells, Hilbert sort, district cover and the cells without a
// number; each province keeps only the raw cell ids.
func (pl *regionPlan) landCells(ctx context.Context, d *db.DB) error {
	pl.combinedIndex()
	ps := pl.set.provinces
	if len(ps) == 0 {
		return nil
	}
	waterBound := pl.clip
	if geo.NoBound(waterBound) {
		waterBound = ps[0].shape.Bound()
		for _, p := range ps[1:] {
			waterBound = waterBound.Union(p.shape.Bound())
		}
		// The districts reaching into the provinces count their land
		// outside them too (districtLand); points inside the provinces'
		// bound see the same water polygons as before.
		pb := waterBound
		for _, d := range pl.set.outside {
			if d.bound.Intersects(pb) {
				waterBound = waterBound.Union(d.bound)
			}
		}
		for _, d := range pl.set.districts {
			waterBound = waterBound.Union(d.bound)
		}
	}
	wp, err := waterPath(pl.o)
	if err != nil {
		return err
	}
	var w *geo.Water
	if wp != "" {
		t := time.Now()
		if w, err = geo.ReadWater(wp, waterBound); err != nil {
			return err
		}
		slog.Info("read water polygons", "polygons", w.Len(), "took", time.Since(t).Round(time.Millisecond))
	} else {
		slog.Warn("no water polygons: sea counts as land")
	}
	debug.FreeOSMemory()
	held := func(id int) bool {
		_, mine := pl.own[id]
		return !mine && pl.countryOf[id] != pl.c.Code
	}
	t := time.Now()
	errs := make([]error, len(ps))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i, p := range ps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			errs[i] = pl.provinceCells(ctx, d, p, w, held)
		}()
	}
	wg.Wait()
	land := 0
	for i, err := range errs {
		if err != nil {
			return fmt.Errorf("land cells of r%d %s: %w", ps[i].osmID, ps[i].names.En, err)
		}
		land += ps[i].land
	}
	for _, d := range pl.candidates() {
		d.share = map[*province]int{}
	}
	for _, p := range ps {
		for d, n := range p.dcells.one {
			d.share[p] += n
		}
		for _, x := range p.dcells.more {
			for _, d := range x.ds {
				d.share[p] += x.n
			}
		}
	}
	slog.Info("counted land cells", "provinces", len(ps), "cells", land, "took", time.Since(t).Round(time.Millisecond))
	t = time.Now()
	if err := pl.districtLand(w); err != nil {
		return err
	}
	slog.Info("counted districts' land cells", "districts", len(pl.set.districts)+len(pl.set.outside), "took", time.Since(t).Round(time.Millisecond))
	return nil
}

// candidates are the districts read, whose centre a province holds or not,
// until keepDistricts.
func (pl *regionPlan) candidates() []*district {
	return slices.Concat(pl.set.districts, pl.set.outside)
}

// districtLand counts every district's land cells for keepDistricts: in
// the run's provinces (its share of their land cells), and the others,
// besides the run's land cells, by the province the combined index gives
// them (of another country, or a stored one not in the run) or none. A
// cell the index gives to one of the run's provinces that the province's
// own land cells missed (h3 and planar containment disagreeing on an edge)
// counts as in them. A district with no share whose centre no province
// holds is left out, since it cannot be kept.
func (pl *regionPlan) districtLand(w *geo.Water) error {
	for _, p := range pl.set.provinces {
		p.bound = p.shape.Bound()
	}
	var ds []*district
	for _, d := range pl.candidates() {
		d.in, d.other, d.none = 0, 0, 0
		for _, n := range d.share {
			d.in += n
		}
		if d.in > 0 || d.centred != nil {
			ds = append(ds, d)
		}
	}
	errs := make([]error, len(ds))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i, d := range ds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			errs[i] = geo.LandCellsBesides(d.shape, pl.landCell, w, pl.clip, func(c orb.Point) {
				switch id, ok := pl.idx.Contains(c); {
				case !ok:
					d.none++
				case pl.own[id] != nil:
					d.in++
				default:
					d.other++
				}
			})
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			return fmt.Errorf("land cells of district r%d %s: %w", ds[i].osmID, ds[i].names.En, err)
		}
	}
	return nil
}

// provinceCells computes one province's land cells.
func (pl *regionPlan) provinceCells(ctx context.Context, d *db.DB, p *province, w *geo.Water, held func(int) bool) error {
	cells, n, err := geo.LandCellsOf(geo.RegionShape{ID: p.indexID, Geometry: p.shape}, pl.idx, w, pl.clip, held)
	if err != nil {
		return err
	}
	// Number along a curve over the whole province (not the clip), so the
	// order does not depend on what was imported first.
	p.land, p.held = len(cells), n
	p.dcells = countDistrictCells(cells, pl.candidates(), p.shape.Bound())
	// Only the raw ids are kept, sorted by id (8 bytes a cell); the order
	// along the Hilbert curve is computed from them when they are numbered
	// (numberingOrder), the dry run does not need it.
	p.cells = make([]uint64, len(cells))
	for i, c := range cells {
		p.cells[i] = c.ID
	}
	cells = nil
	slices.Sort(p.cells)
	// What AssignPlotNumbers would add; a real run replaces it by what it added.
	p.newNumbers, err = d.CountUnnumbered(ctx, p.cells)
	return err
}

// numberingOrder returns p's land cells in the order they are numbered:
// along a Hilbert curve over the whole province (not the clip), so the
// order does not depend on what was imported first; cells in the same
// square of the curve's grid by their index.
func numberingOrder(p *province) ([]uint64, error) {
	cells := make([]geo.LandCell, len(p.cells))
	for i, id := range p.cells {
		lat, lng, err := h3.IDCentre(id)
		if err != nil {
			return nil, err
		}
		cells[i] = geo.LandCell{ID: id, Centre: orb.Point{lng, lat}}
	}
	geo.HilbertSort(cells, p.shape.Bound())
	out := make([]uint64, len(cells))
	for i, c := range cells {
		out[i] = c.ID
	}
	return out, nil
}

// landCell reports whether id is one of the land cells the run's provinces
// gave (their cells, sorted by id): only a province whose bound holds the
// cell's centre can have it.
func (pl *regionPlan) landCell(id uint64, centre orb.Point) bool {
	for _, p := range pl.set.provinces {
		if p.bound.Contains(centre) {
			if _, ok := slices.BinarySearch(p.cells, id); ok {
				return true
			}
		}
	}
	return false
}

// districtCells counts one province's land cells by the districts whose
// boundary holds their centre: once for each district, where districts
// overlap once for each set of them, so that the district cover counts a
// cell once.
type districtCells struct {
	one  map[*district]int     // cells in exactly one district
	more map[string]*cellShare // cells in several, by their OSM ids
}

type cellShare struct {
	ds []*district
	n  int
}

// countDistrictCells counts cells (one province's land cells, inside bound)
// by the districts of ds that hold their centre, whichever province they
// belong to.
func countDistrictCells(cells []geo.LandCell, ds []*district, bound orb.Bound) districtCells {
	var near []*district
	for _, d := range ds {
		if d.bound.Intersects(bound) {
			near = append(near, d)
		}
	}
	out := districtCells{one: map[*district]int{}}
	var in []*district
	for _, c := range cells {
		in = in[:0]
		for _, d := range near {
			if d.bound.Contains(c.Centre) && planar.MultiPolygonContains(d.shape, c.Centre) {
				in = append(in, d)
			}
		}
		switch len(in) {
		case 0:
		case 1:
			out.one[in[0]]++
		default:
			key := ""
			for _, d := range in {
				key += strconv.FormatInt(d.osmID, 10) + ","
			}
			if out.more == nil {
				out.more = map[string]*cellShare{}
			}
			x := out.more[key]
			if x == nil {
				x = &cellShare{ds: slices.Clone(in)}
				out.more[key] = x
			}
			x.n++
		}
	}
	return out
}

// districtCover counts every province's land cells that lie in one of its
// districts (the report's district cover), once the districts have their
// parents and those stored for another country are left out, and drops
// the counts.
func (pl *regionPlan) districtCover() {
	kept := make(map[*district]bool, len(pl.set.districts))
	for _, d := range pl.set.districts {
		kept[d] = true
	}
	mine := func(d *district, p *province) bool { return kept[d] && d.province == p }
	for _, p := range pl.set.provinces {
		p.inDistrict = 0
		for d, n := range p.dcells.one {
			if mine(d, p) {
				p.inDistrict += n
			}
		}
		for _, x := range p.dcells.more {
			if slices.ContainsFunc(x.ds, func(d *district) bool { return mine(d, p) }) {
				p.inDistrict += x.n
			}
		}
		p.dcells = districtCells{}
	}
}

func (pl *regionPlan) stats() *RegionStats {
	st := &RegionStats{Checks: pl.checks}
	if pl.set == nil {
		return st
	}
	st.Regions, st.Districts = len(pl.set.provinces), len(pl.set.districts)
	for _, p := range pl.set.provinces {
		st.LandCells += p.land
		st.BorderSkipped += p.held
		st.NewNumbers += p.newNumbers
	}
	return st
}

// rows returns the province rows to upsert, in OSM id order.
func (pl *regionPlan) provinceRows() []db.Region {
	rows := make([]db.Region, len(pl.set.provinces))
	for i, p := range pl.set.provinces {
		rows[i] = db.Region{OSMID: p.osmID, Kind: "province", KindName: p.kindName, AdminLevel: p.level, Country: pl.c.Code,
			Slug: p.slug, NameEn: p.names.En, NameLocal: p.names.Local, Boundary: p.shape, LandCells: p.land}
	}
	return rows
}

// districtRows returns the district rows to upsert, in OSM id order; the
// provinces must have their ids.
func (pl *regionPlan) districtRows() []db.Region {
	rows := make([]db.Region, len(pl.set.districts))
	for i, d := range pl.set.districts {
		parent := d.province.id
		rows[i] = db.Region{OSMID: d.osmID, Kind: "district", KindName: d.kindName, AdminLevel: d.level, Country: pl.c.Code,
			ParentID: &parent, Slug: d.slug, NameEn: d.names.En, NameLocal: d.names.Local, Boundary: d.shape}
	}
	return rows
}

// apply writes the plan: regions first, then the plot numbers province by
// province in a bounded pool, each province's cells dropped once numbered.
// A run that fails half-way is completed by running it again (numbers only
// append).
func (pl *regionPlan) apply(ctx context.Context, d *db.DB, st *RegionStats) error {
	start := time.Now()
	ids, err := d.UpsertRegions(ctx, pl.provinceRows())
	if err != nil {
		return err
	}
	for i, p := range pl.set.provinces {
		p.id = ids[i]
	}
	if _, err := d.UpsertRegions(ctx, pl.districtRows()); err != nil {
		return err
	}
	if err := d.MarkUrbanDistrictsOf(ctx, pl.c.Code); err != nil {
		return err
	}
	t := time.Now()
	ps := pl.set.provinces
	errs := make([]error, len(ps))
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i, p := range ps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			order, err := numberingOrder(p)
			if err != nil {
				errs[i] = err
				return
			}
			p.cells = nil
			p.newNumbers, errs[i] = d.AssignPlotNumbers(ctx, p.id, order)
		}()
	}
	wg.Wait()
	numbered := 0
	for i, err := range errs {
		if err != nil {
			return fmt.Errorf("plot numbers of %s: %w", ps[i].slug, err)
		}
		numbered += ps[i].newNumbers
	}
	st.NewNumbers = numbered
	slog.Info("numbered plots", "added", numbered, "took", time.Since(t).Round(time.Millisecond))
	pl.applied = true
	if err := d.ReloadRegions(ctx); err != nil {
		return err
	}
	if st.PlotsBackfilled, st.PlotsOutside, err = d.BackfillPlotsOf(ctx, pl.c.Code); err != nil {
		return err
	}
	if st.PlotsNumbered, err = d.NumberPlotsOf(ctx, pl.c.Code); err != nil {
		return err
	}
	if err := d.RefreshRegionCountsOf(ctx, pl.c.Code); err != nil {
		return err
	}
	slog.Info("imported regions", "country", pl.c.Code, "regions", st.Regions, "districts", st.Districts, "land_cells", st.LandCells,
		"border_skipped", st.BorderSkipped, "plots_backfilled", st.PlotsBackfilled, "plots_outside", st.PlotsOutside,
		"took", time.Since(start).Round(time.Millisecond))
	return nil
}
