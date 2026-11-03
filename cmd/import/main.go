// Command import imports countries, one after the other in the order given,
// in one process: for each, the checks, then its regions (provinces,
// districts, land cells, plot numbers), then its places
// (docs/COUNTRY_IMPORT_STAGE2_BRIEF.md, stage 3). Each country is read by
// its settings in internal/countries, from the osmium cut
// <data>/scan/<extract>-latest.osm.pbf.cut.osm.pbf when it is newer than
// the extract <data>/<extract>-latest.osm.pbf, else from the extract. A
// country that fails is logged and the next one still runs; a summary
// table follows, and the exit code is 1 if any country failed.
//
// A real run migrates the database first. -dry-run reads only (a read-only
// connection, no migration: it refuses while one is pending) and prints
// what a run would change. DATABASE_URL comes from the environment.
//
//	import -countries la -dry-run
//	import -countries la,kh -data /srv/map/data
//	import -countries th -pbf testdata/pattaya.osm.gz -water testdata/pattaya-water.shp -bbox 100.82,12.82,100.98,13.02
//
// -pbf, -bbox and an explicit -water other than the data directory's are
// overrides for a single country (the fixture).
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/importer"
)

func main() {
	var list string
	var o importer.Options
	flag.StringVar(&list, "countries", "th", "comma-separated ISO 3166-1 codes, imported in this order")
	flag.BoolVar(&o.DryRun, "dry-run", false, "read only and report what the import would change")
	flag.StringVar(&o.Data, "data", "map/data", "directory of the extracts, their cuts (scan/) and the water polygons ("+importer.WaterFile+")")
	flag.StringVar(&o.Water, "water", "", "water polygons (.shp or the water-polygons-split-3857 .zip); default: the data directory's")
	flag.StringVar(&o.PBF, "pbf", "", "OSM file instead of the data directory's (single country only)")
	flag.StringVar(&o.BBox, "bbox", "", "clip minLng,minLat,maxLng,maxLat (single country only; skips the count and size checks)")
	flag.Float64Var(&o.Tolerance, "tolerance", geo.SimplifyTolerance, "boundary simplification in degrees")
	flag.Parse()
	// The live data of a big country is most of the peak; collect earlier
	// rather than let the heap double (GOGC from the environment wins).
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(40)
	}
	codes := split(list)
	if err := validate(codes, o); err != nil {
		slog.Error("import", "err", err)
		os.Exit(2)
	}
	ctx := context.Background()
	d, err := importer.Connect(ctx, os.Getenv("DATABASE_URL"), o.DryRun)
	if err != nil {
		slog.Error("import", "err", err)
		os.Exit(1)
	}
	defer d.Close()
	o.Report = os.Stdout
	var rows []*importer.Summary
	failed := false
	for _, code := range codes {
		o.Country = code
		s, err := importer.Import(ctx, d, o)
		if err != nil {
			failed = true
			slog.Error("import", "country", code, "err", err)
		}
		rows = append(rows, s)
	}
	fmt.Println()
	importer.WriteSummary(os.Stdout, rows)
	if failed {
		d.Close()
		os.Exit(1)
	}
}

// split returns the codes of a comma-separated list, lower case.
func split(list string) []string {
	var out []string
	for _, c := range strings.Split(list, ",") {
		if c = strings.ToLower(strings.TrimSpace(c)); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// validate refuses an empty list, a code given twice and the
// single-country overrides with several countries.
func validate(codes []string, o importer.Options) error {
	if len(codes) == 0 {
		return fmt.Errorf("-countries is empty")
	}
	seen := map[string]bool{}
	for _, c := range codes {
		if seen[c] {
			return fmt.Errorf("-countries names %q twice", c)
		}
		seen[c] = true
	}
	if len(codes) > 1 && (o.PBF != "" || o.BBox != "") {
		return fmt.Errorf("-pbf and -bbox are for a single country")
	}
	return nil
}
