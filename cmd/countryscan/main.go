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
//
// The boundaries are completed with the cuts of the country's Neighbours
// (internal/countries), as the importer does, so that scan and import
// agree; -neighbours names others ("" for none). A neighbour's extract is
// cut like the country's; its relations and places are never read.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/countryscan"
)

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
		nb       neighbourFlag
	)
	flag.Var(&nb, "neighbours", "comma-separated Geofabrik names whose cuts complete the boundaries; default: the country's settings (Neighbours), \"\" for none")
	flag.Parse()
	ctx := context.Background()
	var err error
	switch {
	case *overlaps:
		err = writeOverlaps(*cache, *out)
	case *all:
		for _, c := range countries.All {
			if e := scanOne(ctx, c, *pbf, *noCut, *docker, *water, *cache, *out, *res, *capN, *workers, &nb); e != nil {
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
		err = scanOne(ctx, c, *pbf, *noCut, *docker, *water, *cache, *out, *res, *capN, *workers, &nb)
	default:
		err = fmt.Errorf("one of -country, -all or -overlaps is required")
	}
	if err != nil {
		slog.Error("countryscan", "err", err)
		os.Exit(1)
	}
}

func scanOne(ctx context.Context, c countries.Country, pbf string, noCut, docker bool, water, cache, out string, res, capN, workers int, nb *neighbourFlag) error {
	names := nb.names
	if !nb.set {
		imp, _ := countries.ImportOf(c.Code)
		names = imp.Neighbours
	}
	var more []string
	for _, n := range names {
		cut, err := neighbourCut(ctx, n, noCut, docker, cache)
		if err != nil {
			return err
		}
		more = append(more, cut)
	}
	if pbf == "" {
		// The extract's Geofabrik name in map/data (map/download.sh) comes
		// from the import settings, so scan and import read the same file.
		imp, ok := countries.ImportOf(c.Code)
		if !ok || imp.Extract == "" {
			return fmt.Errorf("no extract known for %s", c.Code)
		}
		pbf = filepath.Join("map", "data", imp.Extract+"-latest.osm.pbf")
	}
	read := pbf
	if !noCut {
		read = filepath.Join(cache, filepath.Base(pbf)+".cut.osm.pbf")
		slog.Info("cutting", "extract", pbf)
		if err := countryscan.Cut(ctx, pbf, read, docker); err != nil {
			return err
		}
	}
	slog.Info("scanning", "country", c.Code, "file", read, "neighbours", more)
	r, err := countryscan.Scan(ctx, countryscan.Options{Country: c, PBF: read, Neighbours: more, Water: water, Res: res, GeometryCap: capN, Cache: cache, Workers: workers})
	if err != nil {
		return err
	}
	r.Extract = filepath.Base(pbf)
	return write(filepath.Join(out, c.Slug+".md"), r.Report())
}

// neighbourCut returns the cut of the neighbour extract name in cache,
// cutting map/data/<name>-latest.osm.pbf first unless noCut (as for the
// country's own extract). Without the extract an existing cut is read as
// it is; with neither the scan fails.
func neighbourCut(ctx context.Context, name string, noCut, docker bool, cache string) (string, error) {
	extract := filepath.Join("map", "data", name+"-latest.osm.pbf")
	cut := filepath.Join(cache, name+"-latest.osm.pbf.cut.osm.pbf")
	if _, err := os.Stat(extract); err == nil && !noCut {
		slog.Info("cutting", "neighbour", extract)
		if err := countryscan.Cut(ctx, extract, cut, docker); err != nil {
			return "", err
		}
	}
	if _, err := os.Stat(cut); err != nil {
		return "", fmt.Errorf("neighbour cut %s is missing; make it with map/cut.sh %s (or download the extract: map/download.sh %s)", cut, name, name)
	}
	return cut, nil
}

// neighbourFlag is -neighbours: a comma-separated list, and whether it was
// given at all (not given: the country's settings).
type neighbourFlag struct {
	set   bool
	names []string
}

func (f *neighbourFlag) String() string { return strings.Join(f.names, ",") }

func (f *neighbourFlag) Set(s string) error {
	f.set, f.names = true, nil
	for _, n := range strings.Split(s, ",") {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if !validExtract.MatchString(n) {
			return fmt.Errorf("%q is not a Geofabrik extract name", n)
		}
		f.names = append(f.names, n)
	}
	return nil
}

// validExtract is a Geofabrik extract name, as map/merge-cut.sh checks it.
var validExtract = regexp.MustCompile(`^[a-z0-9-]+$`)

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
