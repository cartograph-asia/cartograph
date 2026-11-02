package importer

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/migrations"
)

// Options configures an import of one country: Regions, Places, or both
// through Import. RegionOptions and PlaceOptions are the same type, so the
// callers of the two imports keep their names.
type Options struct {
	Country   string            // ISO 3166-1 code, default "th"
	Settings  *countries.Import // settings to use instead of countries.ImportOf(Country) (tests: a synthetic country)
	PBF       string            // OSM file; empty: the cut or extract in Data (see InputFor)
	Water     string            // water polygons (.shp or .zip); empty: Data's, or none (sea counts as land)
	BBox      string            // optional clip "minLng,minLat,maxLng,maxLat"; the count and size checks are skipped
	Data      string            // directory of the extracts (map/data), for the defaults of PBF and Water
	Tolerance float64           // simplification in degrees, default geo.SimplifyTolerance
	DryRun    bool              // read only, report what a run would change
	Report    io.Writer         // where the checks and the report go; nil: none
}

// RegionOptions configures Regions.
type RegionOptions = Options

// PlaceOptions configures Places.
type PlaceOptions = Options

// WaterFile is where the water polygons lie inside the data directory.
const WaterFile = "sources/water-polygons-split-3857.zip"

// country is the country of a run with its settings.
type country struct {
	Code string // lower case
	Slug string // the suffix of new province slugs (SlugSuffix): countries.Country.Slug, else the code
	Name string
	Imp  countries.Import
}

// resolveCountry returns the country of o. A code that is not in
// internal/countries is refused unless the settings are injected.
func resolveCountry(o Options) (country, error) {
	code := strings.ToLower(o.Country)
	if code == "" {
		code = "th"
	}
	c := country{Code: code, Slug: code, Name: code}
	listed, ok := countries.ByCode(code)
	if ok {
		c.Slug, c.Name = listed.Slug, listed.NameEn
	}
	if o.Settings != nil {
		c.Imp = *o.Settings
		return c, nil
	}
	// Regions of a country that has no page (internal/countries) would be
	// invisible; adding the country to the list comes first.
	if !ok {
		return c, fmt.Errorf("importer: country %q is not in internal/countries; add it there first", code)
	}
	imp, ok := countries.ImportOf(code)
	if !ok {
		return c, fmt.Errorf("importer: country %q has no import settings in internal/countries", code)
	}
	c.Imp = imp
	return c, nil
}

// Input is the OSM file an import reads.
type Input struct {
	Path string
	Cut  bool   // the osmium cut cmd/countryscan (or map/cut.sh) leaves
	Note string // why the full extract is read, or what could not be checked
}

// InputFor returns the file to read for a Geofabrik extract in data: the
// cut data/scan/<extract>-latest.osm.pbf.cut.osm.pbf (boundaries with their
// ways and nodes, place nodes) when it exists and is newer than the extract
// data/<extract>-latest.osm.pbf, else the extract itself with a note.
func InputFor(data, extract string) (Input, error) {
	full := filepath.Join(data, extract+"-latest.osm.pbf")
	cut := filepath.Join(data, "scan", extract+"-latest.osm.pbf.cut.osm.pbf")
	fi, ferr := os.Stat(full)
	ci, cerr := os.Stat(cut)
	switch {
	case cerr == nil && ferr == nil && ci.ModTime().After(fi.ModTime()):
		return Input{Path: cut, Cut: true}, nil
	case cerr == nil && ferr != nil:
		return Input{Path: cut, Cut: true, Note: "the extract is not there, so the cut's age cannot be checked"}, nil
	case ferr == nil && cerr == nil:
		return Input{Path: full, Note: fmt.Sprintf("the cut is older than the extract (%s < %s); reading the full extract",
			ci.ModTime().UTC().Format(time.DateTime), fi.ModTime().UTC().Format(time.DateTime))}, nil
	case ferr == nil:
		return Input{Path: full, Note: "no cut (cmd/countryscan or map/cut.sh makes one); reading the full extract"}, nil
	}
	return Input{}, fmt.Errorf("importer: neither %s nor %s exists", full, cut)
}

// input returns the OSM file of o for c.
func input(o Options, c country) (Input, error) {
	if o.PBF != "" {
		return Input{Path: o.PBF}, nil
	}
	if o.Data == "" {
		return Input{}, fmt.Errorf("importer: no OSM file given (-pbf, or -data with the extract)")
	}
	if c.Imp.Extract == "" {
		return Input{}, fmt.Errorf("importer: the settings of %q name no extract", c.Code)
	}
	return InputFor(o.Data, c.Imp.Extract)
}

// neighbourInputFor finds a neighbour's file: InputFor, replaced in tests.
var neighbourInputFor = InputFor

// neighbours is what a run reads to complete its boundaries
// (countries.Import.Neighbours).
type neighbours struct {
	read    []Input  // the neighbours' files, in the settings' order
	missing []string // cut paths of neighbours with neither cut nor extract (check 6)
	skipped string   // why none were looked for, "" when they were
}

// paths returns the files to read.
func (n neighbours) paths() []string {
	out := make([]string, len(n.read))
	for i, in := range n.read {
		out[i] = in.Path
	}
	return out
}

// neighbourInputs returns the files that complete c's boundaries: for each
// of the settings' Neighbours its cut in Data, by the rules of InputFor
// (the cut when it is newer than the extract, else the extract with a
// note). A neighbour with neither is listed in missing, and check 6 fails.
// With -pbf and no -data the neighbours are not looked for (the file given
// may be a merged cut, map/merge-cut.sh). Places never read these files.
func neighbourInputs(o Options, c country) neighbours {
	if len(c.Imp.Neighbours) == 0 {
		return neighbours{}
	}
	if o.Data == "" {
		return neighbours{skipped: "-pbf without -data"}
	}
	var n neighbours
	for _, name := range c.Imp.Neighbours {
		in, err := neighbourInputFor(o.Data, name)
		if err != nil {
			n.missing = append(n.missing, filepath.Join(o.Data, "scan", name+"-latest.osm.pbf.cut.osm.pbf"))
			continue
		}
		n.read = append(n.read, in)
	}
	return n
}

// waterPath returns the water polygons of o: Water, else the file in Data,
// else "" (no water: sea counts as land, as for the fixture without it).
func waterPath(o Options) (string, error) {
	if o.Water != "" || o.Data == "" {
		return o.Water, nil
	}
	p := filepath.Join(o.Data, WaterFile)
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("importer: water polygons: %w (give -water)", err)
	}
	return p, nil
}

// Connect opens the database at url for an import. A real run migrates
// first, as the importers always did. A dry run does not: it opens the
// connection read-only (default_transaction_read_only) and refuses when a
// migration is pending, so it can be pointed at any copy of production.
func Connect(ctx context.Context, url string, dryRun bool) (*db.DB, error) {
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}
	if dryRun {
		var err error
		if url, err = readOnly(url); err != nil {
			return nil, err
		}
	}
	d, err := db.Connect(ctx, url, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if dryRun {
		err = checkMigrated(ctx, d)
	} else {
		err = d.Migrate(ctx, migrations.FS)
	}
	if err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

// readOnly adds default_transaction_read_only=on to a connection string
// (URL or key=value form).
func readOnly(s string) (string, error) {
	if !strings.Contains(s, "://") {
		return s + " default_transaction_read_only=on", nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("importer: DATABASE_URL: %w", err)
	}
	q := u.Query()
	q.Set("default_transaction_read_only", "on")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// checkMigrated refuses a database that lacks a migration: a dry run reads
// only and does not migrate, and its comparison needs the current schema.
func checkMigrated(ctx context.Context, d *db.DB) error {
	pending, err := d.PendingMigrations(ctx, migrations.FS)
	if err != nil {
		return err
	}
	if len(pending) > 0 {
		return fmt.Errorf("importer: dry run refused: %d migration(s) pending (%s); a dry run reads only and does not migrate",
			len(pending), strings.Join(pending, ", "))
	}
	return nil
}
