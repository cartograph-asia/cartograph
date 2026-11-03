// Command regions imports one country's provinces and districts, read by
// its settings in internal/countries, from an OSM extract into the regions
// table: simplified boundary, bbox, English and local names, slug, kind word
// and land_cells (res-8 cells whose centre is inside the province and not in
// a water polygon, numbered along a Hilbert curve). It checks first and
// refuses when a check fails; it touches only the country's own rows, is
// idempotent (upsert by osm_id) and afterwards backfills the country's plots
// without a region and recomputes claimed_cells and last_surveyed_at.
// DATABASE_URL comes from the environment; a real run applies the
// migrations first, -dry-run reads only and refuses while one is pending.
// cmd/import runs regions and places together for several countries.
//
//	regions -country th -data map/data -dry-run
//	regions -pbf map/data/thailand-latest.osm.pbf -water map/data/sources/water-polygons-split-3857.zip
//	regions -pbf testdata/pattaya.osm.gz -water testdata/pattaya-water.shp -bbox 100.82,12.82,100.98,13.02
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/importer"
)

func main() {
	var o importer.RegionOptions
	flag.StringVar(&o.Country, "country", "th", "ISO 3166-1 code of the country (its settings in internal/countries)")
	flag.BoolVar(&o.DryRun, "dry-run", false, "read only and report what the import would change")
	flag.StringVar(&o.PBF, "pbf", "", "OSM extract (.osm.pbf, .osm or .osm.gz); default: the cut or extract in -data")
	flag.StringVar(&o.Data, "data", "", "directory of the extracts, their cuts (scan/) and the water polygons ("+importer.WaterFile+")")
	flag.StringVar(&o.Water, "water", "", "water polygons (.shp or the water-polygons-split-3857 .zip); default: -data's, else none (sea counts as land)")
	flag.StringVar(&o.BBox, "bbox", "", "optional clip minLng,minLat,maxLng,maxLat: only regions intersecting it, land cells inside it")
	flag.Float64Var(&o.Tolerance, "tolerance", geo.SimplifyTolerance, "boundary simplification in degrees")
	flag.Parse()
	if err := run(o); err != nil {
		slog.Error("regions", "err", err)
		os.Exit(1)
	}
}

func run(o importer.RegionOptions) error {
	ctx := context.Background()
	d, err := importer.Connect(ctx, os.Getenv("DATABASE_URL"), o.DryRun)
	if err != nil {
		return err
	}
	defer d.Close()
	o.Report = os.Stdout
	_, err = importer.Regions(ctx, d, o)
	return err
}
