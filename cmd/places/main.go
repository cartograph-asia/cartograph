// Command places imports OSM city, town and village nodes (English and Thai
// names, region) into the places table, replacing its contents, and fills
// the place names of plots that have none. Run it after cmd/regions.
// DATABASE_URL comes from the environment; migrations are applied first.
//
//	places -pbf map/data/thailand-latest.osm.pbf
//	places -pbf testdata/pattaya.osm.gz -bbox 100.82,12.82,100.98,13.02
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/importer"
	"github.com/cartograph-asia/cartograph/migrations"
)

func main() {
	var o importer.PlaceOptions
	flag.StringVar(&o.PBF, "pbf", "", "OSM extract (.osm.pbf, .osm or .osm.gz)")
	flag.StringVar(&o.BBox, "bbox", "", "optional clip minLng,minLat,maxLng,maxLat")
	flag.Parse()
	if err := run(o); err != nil {
		slog.Error("places", "err", err)
		os.Exit(1)
	}
}

func run(o importer.PlaceOptions) error {
	if o.PBF == "" {
		return fmt.Errorf("-pbf is required")
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	ctx := context.Background()
	d, err := db.Connect(ctx, url, 30*time.Second)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Migrate(ctx, migrations.FS); err != nil {
		return err
	}
	_, err = importer.Places(ctx, d, o)
	return err
}
