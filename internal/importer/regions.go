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
	"strconv"
	"sync"
	"time"

	"github.com/paulmach/orb"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/geo"
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
	if err := pl.loadStored(ctx, d); err != nil {
		return nil, err
	}
	resolveSlugs(pl.set, c, pl.byOSM)
	if err := pl.landCells(ctx, d); err != nil {
		return nil, err
	}
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
			return fmt.Errorf("land cells of %s: %w", ps[i].slug, err)
		}
		land += ps[i].land
	}
	slog.Info("counted land cells", "provinces", len(ps), "cells", land, "took", time.Since(t).Round(time.Millisecond))
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
	geo.HilbertSort(cells, p.shape.Bound())
	p.land, p.held = len(cells), n
	if len(p.districts) > 0 {
		shapes := make([]geo.RegionShape, len(p.districts))
		for i, d := range p.districts {
			shapes[i] = geo.RegionShape{ID: i, Geometry: d.shape}
		}
		didx := geo.NewRegionIndex(shapes)
		for _, c := range cells {
			if _, ok := didx.Contains(c.Centre); ok {
				p.inDistrict++
			}
		}
	}
	p.cells = make([]uint64, len(cells))
	for i, c := range cells {
		if p.cells[i], err = strconv.ParseUint(c.Cell, 16, 64); err != nil {
			return fmt.Errorf("cell %q: %w", c.Cell, err)
		}
	}
	// What AssignPlotNumbers would add; a real run replaces it by what it added.
	p.newNumbers, err = d.CountUnnumbered(ctx, cellStrings(p.cells))
	return err
}

// cellStrings turns raw cell ids back into the 15-character hex strings the
// database stores (h3-go formats a cell with strconv.FormatUint, base 16).
func cellStrings(ids []uint64) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = strconv.FormatUint(id, 16)
	}
	return out
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
			p.newNumbers, errs[i] = d.AssignPlotNumbers(ctx, p.id, cellStrings(p.cells))
			p.cells = nil
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
