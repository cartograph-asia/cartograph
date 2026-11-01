// Command regions imports a country's provinces (OSM admin_level 4) from an
// OSM extract into the regions table: simplified boundary, bbox, English and
// Thai names, slug and land_cells (res-8 cells whose centre is inside the
// province and not in a water polygon). It is idempotent (upsert by osm_id)
// and afterwards backfills plots without a region and recomputes
// claimed_cells and last_surveyed_at. DATABASE_URL comes from the
// environment; migrations are applied first.
//
//	regions -pbf map/data/thailand-latest.osm.pbf -water map/data/sources/water-polygons-split-3857.zip
//	regions -pbf testdata/pattaya.osm.gz -water testdata/pattaya-water.shp -bbox 100.82,12.82,100.98,13.02
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/importer"
	"github.com/cartograph-asia/cartograph/migrations"
)

func main() {
	var o importer.RegionOptions
	flag.StringVar(&o.PBF, "pbf", "", "OSM extract (.osm.pbf, .osm or .osm.gz)")
	flag.StringVar(&o.Water, "water", "", "water polygons (.shp or the water-polygons-split-3857 .zip); empty counts sea as land")
	flag.StringVar(&o.BBox, "bbox", "", "optional clip minLng,minLat,maxLng,maxLat: only regions intersecting it, land cells inside it")
	flag.StringVar(&o.Country, "country", "th", "ISO 3166-1 code of the provinces (ISO3166-2 prefix)")
	flag.Float64Var(&o.Tolerance, "tolerance", geo.SimplifyTolerance, "boundary simplification in degrees")
	flag.Parse()
	if err := run(o); err != nil {
		slog.Error("regions", "err", err)
		os.Exit(1)
	}
}

func run(o importer.RegionOptions) error {
	if o.PBF == "" {
		return fmt.Errorf("-pbf is required")
	}
	ctx := context.Background()
	d, err := connect(ctx)
	if err != nil {
		return err
	}
	defer d.Close()
	_, err = importer.Regions(ctx, d, o)
	return err
}

func connect(ctx context.Context) (*db.DB, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}
	d, err := db.Connect(ctx, url, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if err := d.Migrate(ctx, migrations.FS); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}
