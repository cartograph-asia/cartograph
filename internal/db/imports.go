package db

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/paulmach/orb"

	"github.com/cartograph-asia/cartograph/internal/geo"
)

// The functions in this file only read. The importer (internal/importer)
// compares what an OSM extract gives with what is stored through them, so
// its dry run can report every change without writing anything.

// StoredRegion is a stored row of regions as the importer compares it.
type StoredRegion struct {
	ID         int
	OSMID      int64
	Kind       string
	KindName   string // "" for NULL
	AdminLevel int
	Country    string
	Slug       string
	NameEn     string
	NameLocal  string // "" for NULL
	ParentID   *int
	LandCells  int
	Boundary   string           // GeoJSON text as stored
	Geometry   orb.MultiPolygon // decoded, provinces only
}

// StoredRegions returns every row of regions ordered by id; the boundaries
// of provinces are decoded (the importer's combined index), those of the
// other kinds only kept as text.
func (d *DB) StoredRegions(ctx context.Context) ([]StoredRegion, error) {
	rows, err := d.Pool.Query(ctx, `
		select id, osm_id, kind, coalesce(kind_name, ''), admin_level, country, slug, name_en,
		       coalesce(name_local, ''), parent_id, land_cells, boundary
		from regions order by id`)
	if err != nil {
		return nil, fmt.Errorf("db: stored regions: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (StoredRegion, error) {
		var s StoredRegion
		err := r.Scan(&s.ID, &s.OSMID, &s.Kind, &s.KindName, &s.AdminLevel, &s.Country, &s.Slug, &s.NameEn,
			&s.NameLocal, &s.ParentID, &s.LandCells, &s.Boundary)
		return s, err
	})
	if err != nil {
		return nil, fmt.Errorf("db: stored regions: %w", err)
	}
	for i := range out {
		if out[i].Kind != "province" {
			continue
		}
		if out[i].Geometry, err = geo.DecodeGeometry(out[i].Boundary); err != nil {
			return nil, fmt.Errorf("db: region %d: %w", out[i].ID, err)
		}
	}
	return out, nil
}

// unnumberedChunk is how many cells CountUnnumbered sends in one query.
const unnumberedChunk = 50_000

// CountUnnumbered returns how many of cells (raw 64-bit H3 indexes) have no
// plot number yet: the numbers AssignPlotNumbers would add. It sends them in
// chunks, so it never holds more than one chunk's strings.
func (d *DB) CountUnnumbered(ctx context.Context, cells []uint64) (int, error) {
	total := 0
	for len(cells) > 0 {
		k := min(len(cells), unnumberedChunk)
		chunk := make([]string, k)
		for i, c := range cells[:k] {
			chunk[i] = cellText(c)
		}
		cells = cells[k:]
		var n int
		err := d.Pool.QueryRow(ctx, `
			select count(*) from unnest($1::text[]) c(cell)
			where not exists (select 1 from plot_numbers p where p.cell = c.cell)`, chunk).Scan(&n)
		if err != nil {
			return 0, fmt.Errorf("db: count unnumbered: %w", err)
		}
		total += n
	}
	return total, nil
}

// cellText is a raw cell index as the 15-character hex string the tables
// store (h3-go formats a cell with strconv.FormatUint, base 16).
func cellText(id uint64) string { return strconv.FormatUint(id, 16) }

// StoredPlace is a stored row of places as the importer compares it.
type StoredPlace struct {
	OSMType   string
	OSMID     int64
	Kind      string
	NameEn    string
	NameLocal string // "" for NULL
	Lat, Lng  float64
	RegionID  *int
	Country   string
}

// StoredPlaces returns every row of places.
func (d *DB) StoredPlaces(ctx context.Context) ([]StoredPlace, error) {
	rows, err := d.Pool.Query(ctx, `
		select osm_type, osm_id, kind, name_en, coalesce(name_local, ''), lat, lng, region_id, country
		from places order by osm_type, osm_id`)
	if err != nil {
		return nil, fmt.Errorf("db: stored places: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (StoredPlace, error) {
		var p StoredPlace
		err := r.Scan(&p.OSMType, &p.OSMID, &p.Kind, &p.NameEn, &p.NameLocal, &p.Lat, &p.Lng, &p.RegionID, &p.Country)
		p.OSMType = strings.TrimSpace(p.OSMType)
		return p, err
	})
	if err != nil {
		return nil, fmt.Errorf("db: stored places: %w", err)
	}
	return out, nil
}

// PendingMigrations returns the migrations in fsys that schema_migrations
// does not record (all of them when the table does not exist), in the order
// Migrate would apply them. It only reads.
func (d *DB) PendingMigrations(ctx context.Context, fsys fs.FS) ([]string, error) {
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("migrate: list files: %w", err)
	}
	sort.Strings(names)
	applied := map[string]bool{}
	var exists bool
	if err := d.Pool.QueryRow(ctx, `select to_regclass('schema_migrations') is not null`).Scan(&exists); err != nil {
		return nil, fmt.Errorf("migrate: read schema_migrations: %w", err)
	}
	if exists {
		rows, err := d.Pool.Query(ctx, "select version from schema_migrations")
		if err != nil {
			return nil, fmt.Errorf("migrate: read schema_migrations: %w", err)
		}
		versions, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return nil, fmt.Errorf("migrate: read schema_migrations: %w", err)
		}
		for _, v := range versions {
			applied[v] = true
		}
	}
	var out []string
	for _, n := range names {
		if v := strings.TrimSuffix(n, ".sql"); !applied[v] {
			out = append(out, v)
		}
	}
	return out, nil
}
