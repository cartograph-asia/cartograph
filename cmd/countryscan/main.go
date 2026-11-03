// Command countryscan reads an OSM extract and writes a report on what it
// holds for one country: administrative levels, names, places, the number of
// plots (docs/countries/<slug>.md). It is stage 1 of
// docs/COUNTRY_IMPORT_BRIEF.md and writes nothing to the database.
//
//	countryscan -country vn
//	countryscan -all
//	countryscan -overlaps
//
// The extract is cut down to its administrative boundaries and place nodes
// with osmium first (on the host, else in a container) and the result kept in
// map/data/scan/. -pbf and -no-cut read a file as it is, for the fixture.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/countryscan"
)

// extracts maps a country to its Geofabrik extract in map/data (map/download.sh).
var extracts = map[string]string{
	"bn": "malaysia-singapore-brunei", "kh": "cambodia", "id": "indonesia", "jp": "japan",
	"la": "laos", "my": "malaysia-singapore-brunei", "mm": "myanmar",
	"ph": "philippines", "sg": "malaysia-singapore-brunei", "kr": "south-korea", "tw": "taiwan",
	"th": "thailand", "tl": "east-timor", "vn": "vietnam",
}

func main() {
	var (
		code     = flag.String("country", "", "ISO 3166-1 code of the country to scan, e.g. vn")
		all      = flag.Bool("all", false, "scan every country in internal/countries, one after the other")
		overlaps = flag.Bool("overlaps", false, "compare the land plots the scans left behind and write overlaps.md")
		pbf      = flag.String("pbf", "", "extract to read instead of map/data/<name>-latest.osm.pbf")
		noCut    = flag.Bool("no-cut", false, "read -pbf as it is, without cutting it with osmium")
		docker   = flag.Bool("docker", false, "run osmium in a container even if it is installed")
		water    = flag.String("water", "map/data/sources/water-polygons-split-3857.zip", "water polygons (.shp or .zip); empty counts sea as land")
		cache    = flag.String("cache", "map/data/scan", "directory for the cut extracts and the land cell lists")
		out      = flag.String("out", "docs/countries", "directory for the reports")
		res      = flag.Int("res", 8, "H3 resolution of the land set (8 = plots; 7 is faster and estimates)")
		capN     = flag.Int("geometry-cap", countryscan.DefaultGeometryCap, "most relations of one level to build geometry for")
		workers  = flag.Int("workers", runtime.GOMAXPROCS(0), "parallel workers")
	)
	flag.Parse()
	ctx := context.Background()
	var err error
	switch {
	case *overlaps:
		err = writeOverlaps(*cache, *out)
	case *all:
		for _, c := range countries.All {
			if e := scanOne(ctx, c, *pbf, *noCut, *docker, *water, *cache, *out, *res, *capN, *workers); e != nil {
				slog.Error("scan failed", "country", c.Code, "err", e)
				err = fmt.Errorf("some scans failed")
			}
		}
	case *code != "":
		c, ok := countries.ByCode(*code)
		if !ok {
			err = fmt.Errorf("country %q is not in internal/countries", *code)
			break
		}
		err = scanOne(ctx, c, *pbf, *noCut, *docker, *water, *cache, *out, *res, *capN, *workers)
	default:
		err = fmt.Errorf("one of -country, -all or -overlaps is required")
	}
	if err != nil {
		slog.Error("countryscan", "err", err)
		os.Exit(1)
	}
}

func scanOne(ctx context.Context, c countries.Country, pbf string, noCut, docker bool, water, cache, out string, res, capN, workers int) error {
	if pbf == "" {
		name, ok := extracts[c.Code]
		if !ok {
			return fmt.Errorf("no extract known for %s", c.Code)
		}
		pbf = filepath.Join("map", "data", name+"-latest.osm.pbf")
	}
	read := pbf
	if !noCut {
		read = filepath.Join(cache, filepath.Base(pbf)+".cut.osm.pbf")
		slog.Info("cutting", "extract", pbf)
		if err := countryscan.Cut(ctx, pbf, read, docker); err != nil {
			return err
		}
	}
	slog.Info("scanning", "country", c.Code, "file", read)
	r, err := countryscan.Scan(ctx, countryscan.Options{Country: c, PBF: read, Water: water, Res: res, GeometryCap: capN, Cache: cache, Workers: workers})
	if err != nil {
		return err
	}
	r.Extract = filepath.Base(pbf)
	return write(filepath.Join(out, c.Slug+".md"), r.Report())
}

func writeOverlaps(cache, out string) error {
	s, err := countryscan.Overlaps(cache)
	if err != nil {
		return err
	}
	return write(filepath.Join(out, "overlaps.md"), s)
}

func write(name, s string) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	slog.Info("wrote", "file", name)
	return os.WriteFile(name, []byte(s), 0o644)
}
