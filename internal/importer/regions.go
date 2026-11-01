// Package importer loads the OSM-derived tables: regions (cmd/regions) and
// places (cmd/places). Both imports are idempotent and end by backfilling
// plots that lack a region or place name.
package importer

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
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

// RegionOptions configures Regions.
type RegionOptions struct {
	PBF       string  // OSM extract (.osm.pbf, .osm or .osm.gz)
	Water     string  // water polygons (.shp or .zip); empty counts sea as land
	BBox      string  // optional clip "minLng,minLat,maxLng,maxLat"
	Country   string  // ISO 3166-1 code, default "th"
	Tolerance float64 // simplification in degrees, default geo.SimplifyTolerance
}

// RegionStats reports what Regions did.
type RegionStats struct {
	Regions         int
	LandCells       int
	PlotsBackfilled int
	PlotsOutside    int
}

// Regions imports the provinces into d: upsert by osm_id, land cells, plot
// backfill, claimed_cells/last_surveyed_at recomputed. d must be migrated.
func Regions(ctx context.Context, d *db.DB, o RegionOptions) (*RegionStats, error) {
	if o.Country == "" {
		o.Country = "th"
	}
	if o.Tolerance == 0 {
		o.Tolerance = geo.SimplifyTolerance
	}
	pbf, waterPath, country, tol := o.PBF, o.Water, o.Country, o.Tolerance
	if pbf == "" {
		return nil, fmt.Errorf("importer: no OSM file given")
	}
	clip, err := geo.ParseBound(o.BBox)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	raw, err := osmfile.ReadBoundaryData(pbf, func(r *osm.Relation) bool { return osmfile.IsProvince(r, country) })
	if err != nil {
		return nil, err
	}
	bs, skipped, err := osmfile.BuildBoundaries(raw)
	if err != nil {
		return nil, err
	}
	for _, id := range skipped {
		slog.Warn("skipping boundary without closed rings", "relation", id)
	}
	slog.Info("read boundaries", "relations", len(raw.Relations), "ways", len(raw.Ways), "nodes", len(raw.Nodes), "took", time.Since(start).Round(time.Millisecond))
	raw = nil

	var rows []db.Region
	var shapes []geo.RegionShape
	all := orb.Bound{Min: orb.Point{1, 1}} // empty
	slugs := map[string]int64{}
	for _, b := range bs {
		simple := geo.Simplify(b.Geometry, tol)
		if len(simple) == 0 {
			slog.Warn("boundary vanished after simplification", "relation", b.OSMID)
			continue
		}
		if !geo.NoBound(clip) && !simple.Bound().Intersects(clip) {
			continue
		}
		en, th := osmfile.Names(b.Tags)
		en, th = osmfile.RegionName(en, th)
		slug := osmfile.Slug(en)
		if other, dup := slugs[slug]; dup || slug == "" {
			return nil, fmt.Errorf("relation %d: slug %q is empty or also used by relation %d", b.OSMID, slug, other)
		}
		slugs[slug] = b.OSMID
		rows = append(rows, db.Region{OSMID: b.OSMID, AdminLevel: 4, Country: country, Slug: slug, NameEn: en, NameTh: th, Boundary: simple})
		shapes = append(shapes, geo.RegionShape{ID: len(shapes), Geometry: simple})
		if all.IsEmpty() {
			all = simple.Bound()
		} else {
			all = all.Union(simple.Bound())
		}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no provinces found for country %q", country)
	}
	// Overlaps resolve to the lower osm_id, matching the api's index order
	// as long as regions are inserted in this order.
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].OSMID < rows[j].OSMID })
	for i := range rows {
		shapes[i] = geo.RegionShape{ID: i, Geometry: rows[i].Boundary}
	}

	waterBound := all
	if !geo.NoBound(clip) {
		waterBound = clip
	}
	var w *geo.Water
	if waterPath != "" {
		t := time.Now()
		if w, err = geo.ReadWater(waterPath, waterBound); err != nil {
			return nil, err
		}
		slog.Info("read water polygons", "polygons", w.Len(), "took", time.Since(t).Round(time.Millisecond))
	}

	t := time.Now()
	idx := geo.NewRegionIndex(shapes)
	var wg sync.WaitGroup
	errs := make([]error, len(rows))
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i := range rows {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			rows[i].LandCells, errs[i] = geo.LandCells(shapes[i], idx, w, clip)
		}()
	}
	wg.Wait()
	land := 0
	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("land cells of %s: %w", rows[i].Slug, err)
		}
		land += rows[i].LandCells
	}
	slog.Info("counted land cells", "regions", len(rows), "cells", land, "took", time.Since(t).Round(time.Millisecond))

	if _, err := d.UpsertRegions(ctx, rows); err != nil {
		return nil, err
	}
	if err := d.ReloadRegions(ctx); err != nil {
		return nil, err
	}
	st := &RegionStats{Regions: len(rows), LandCells: land}
	if st.PlotsBackfilled, st.PlotsOutside, err = d.BackfillPlots(ctx); err != nil {
		return nil, err
	}
	if err := d.RefreshRegionCounts(ctx); err != nil {
		return nil, err
	}
	slog.Info("imported regions", "regions", st.Regions, "land_cells", st.LandCells, "plots_backfilled", st.PlotsBackfilled,
		"plots_outside", st.PlotsOutside, "took", time.Since(start).Round(time.Millisecond))
	return st, nil
}
