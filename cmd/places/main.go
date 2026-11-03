// Command places imports one country's OSM place nodes (the place kinds of
// its settings, English and local names, province) into the places table:
// it upserts them and deletes the country's places that are gone, and fills
// the place names of the country's plots that have none. A place inside
// another country's province is left to that country; other countries'
// places are never changed. Run it after cmd/regions (cmd/import runs both).
// DATABASE_URL comes from the environment; a real run applies the
// migrations first, -dry-run reads only and refuses while one is pending.
//
//	places -country th -data map/data -dry-run
//	places -pbf map/data/thailand-latest.osm.pbf
//	places -pbf testdata/pattaya.osm.gz -bbox 100.82,12.82,100.98,13.02
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/cartograph-asia/cartograph/internal/importer"
)

func main() {
	var o importer.PlaceOptions
	flag.StringVar(&o.Country, "country", "th", "ISO 3166-1 code of the country (its settings in internal/countries)")
	flag.BoolVar(&o.DryRun, "dry-run", false, "read only and report what the import would change")
	flag.StringVar(&o.PBF, "pbf", "", "OSM extract (.osm.pbf, .osm or .osm.gz); default: the cut or extract in -data")
	flag.StringVar(&o.Data, "data", "", "directory of the extracts and their cuts (scan/)")
	flag.StringVar(&o.BBox, "bbox", "", "optional clip minLng,minLat,maxLng,maxLat")
	flag.Parse()
	if err := run(o); err != nil {
		slog.Error("places", "err", err)
		os.Exit(1)
	}
}

func run(o importer.PlaceOptions) error {
	ctx := context.Background()
	d, err := importer.Connect(ctx, os.Getenv("DATABASE_URL"), o.DryRun)
	if err != nil {
		return err
	}
	defer d.Close()
	o.Report = os.Stdout
	_, err = importer.Places(ctx, d, o)
	return err
}
