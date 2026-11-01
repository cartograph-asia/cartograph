package db

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/paulmach/orb"

	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/h3"
)

// Region is one row for UpsertRegions.
type Region struct {
	OSMID      int64
	Kind       string // "country", "province" or "city"
	AdminLevel int
	Country    string // ISO 3166-1 alpha-2, lower case
	Slug       string
	NameEn     string
	NameTh     string           // may be empty
	Boundary   orb.MultiPolygon // simplified, WGS 84
	LandCells  int
}

// UpsertRegions inserts or updates regions by osm_id in one transaction and
// returns their ids in order. claimed_cells and last_surveyed_at are left
// alone; call RefreshRegionCounts afterwards.
func (d *DB) UpsertRegions(ctx context.Context, rs []Region) ([]int, error) {
	ids := make([]int, len(rs))
	err := pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error {
		for i, r := range rs {
			text, err := geo.EncodeGeometry(r.Boundary)
			if err != nil {
				return err
			}
			b := r.Boundary.Bound()
			var th *string
			if r.NameTh != "" {
				th = &r.NameTh
			}
			err = tx.QueryRow(ctx, `
				insert into regions (osm_id, admin_level, country, slug, name_en, name_th, boundary,
				                     min_lat, min_lng, max_lat, max_lng, land_cells, kind)
				values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
				on conflict (osm_id) do update set
				  kind = excluded.kind, admin_level = excluded.admin_level, country = excluded.country, slug = excluded.slug,
				  name_en = excluded.name_en, name_th = excluded.name_th, boundary = excluded.boundary,
				  min_lat = excluded.min_lat, min_lng = excluded.min_lng,
				  max_lat = excluded.max_lat, max_lng = excluded.max_lng, land_cells = excluded.land_cells
				returning id`,
				r.OSMID, r.AdminLevel, r.Country, r.Slug, r.NameEn, th, text,
				b.Min[1], b.Min[0], b.Max[1], b.Max[0], r.LandCells, r.Kind,
			).Scan(&ids[i])
			if err != nil {
				return fmt.Errorf("upsert region %d (%s): %w", r.OSMID, r.Slug, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("db: upsert regions: %w", err)
	}
	return ids, nil
}

// RegionShapes returns every province's id and boundary, ordered by id.
// Plots belong to provinces, so countries and cities are not in the index.
func (d *DB) RegionShapes(ctx context.Context) ([]geo.RegionShape, error) {
	rows, err := d.Pool.Query(ctx, `select id, boundary from regions where kind = 'province' order by id`)
	if err != nil {
		return nil, fmt.Errorf("db: region shapes: %w", err)
	}
	type row struct {
		id   int
		text string
	}
	rs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		return x, r.Scan(&x.id, &x.text)
	})
	if err != nil {
		return nil, fmt.Errorf("db: region shapes: %w", err)
	}
	out := make([]geo.RegionShape, 0, len(rs))
	for _, r := range rs {
		g, err := geo.DecodeGeometry(r.text)
		if err != nil {
			return nil, fmt.Errorf("db: region %d: %w", r.id, err)
		}
		out = append(out, geo.RegionShape{ID: r.id, Geometry: g})
	}
	return out, nil
}

// regionCache holds the in-memory point-in-region index.
type regionCache struct {
	mu       sync.Mutex
	idx      *geo.RegionIndex
	loadedAt time.Time
}

// regionRetry is how long after a load an empty or failed lookup may
// trigger another load (the importers run as separate processes).
const regionRetry = time.Minute

// ReloadRegions loads the region boundaries into the in-memory index used by
// ClaimPhoto and the plot pages. The api calls it at start-up; lookups also
// reload lazily when the index is empty or a point is in no region and the
// last load is older than a minute, so an import is picked up without restart.
func (d *DB) ReloadRegions(ctx context.Context) error {
	shapes, err := d.RegionShapes(ctx)
	if err != nil {
		return err
	}
	idx := geo.NewRegionIndex(shapes)
	d.regions.mu.Lock()
	d.regions.idx, d.regions.loadedAt = idx, time.Now()
	d.regions.mu.Unlock()
	return nil
}

// LocateRegion returns the region of a point (lat, lng), or ok false when it
// lies outside every region.
func (d *DB) LocateRegion(ctx context.Context, lat, lng float64) (id int, ok bool, err error) {
	d.regions.mu.Lock()
	idx, stale := d.regions.idx, time.Since(d.regions.loadedAt) > regionRetry
	d.regions.mu.Unlock()
	p := orb.Point{lng, lat}
	if id, ok := idx.Locate(p); ok {
		return id, true, nil
	}
	if idx != nil && !stale {
		return 0, false, nil
	}
	if err := d.ReloadRegions(ctx); err != nil {
		return 0, false, err
	}
	d.regions.mu.Lock()
	idx = d.regions.idx
	d.regions.mu.Unlock()
	id, ok = idx.Locate(p)
	return id, ok, nil
}

// RefreshRegionCounts recomputes claimed_cells and last_surveyed_at of every
// region from plots and photos, so re-imports and backfills stay consistent.
func (d *DB) RefreshRegionCounts(ctx context.Context) error {
	_, err := d.Pool.Exec(ctx, `
		update regions r set
		  claimed_cells    = coalesce((select count(*) from plots p where p.region_id = r.id and p.photo_count > 0), 0),
		  last_surveyed_at = (select max(ph.uploaded_at) from photos ph join plots p on p.cell = ph.cell where p.region_id = r.id)`)
	if err != nil {
		return fmt.Errorf("db: refresh region counts: %w", err)
	}
	return nil
}

// ErrOutsideRegions is returned by ClaimPhoto when the plot's centre lies
// outside every imported region (or no regions are imported).
var ErrOutsideRegions = errors.New("the position is outside the surveyed area")

// querier is a pool or a transaction.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PlotLocation is where a plot is: its region and display names.
type PlotLocation struct {
	RegionID int
	Region   string  // region name_en
	PlaceEn  string  // nearest place within PlaceRadiusKm, else the region's name
	PlaceTh  *string // Thai name of the same; nil if unknown
	Lat, Lng float64 // cell centre
}

// LocatePlot finds the region containing the centre of cell (in memory) and
// the nearest place (in SQL), or returns ErrOutsideRegions.
func (d *DB) LocatePlot(ctx context.Context, cell string) (*PlotLocation, error) {
	return d.locatePlot(ctx, d.Pool, cell)
}

func (d *DB) locatePlot(ctx context.Context, q querier, cell string) (*PlotLocation, error) {
	lat, lng, err := h3.Centre(cell)
	if err != nil {
		return nil, err
	}
	id, ok, err := d.LocateRegion(ctx, lat, lng)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrOutsideRegions
	}
	loc := &PlotLocation{RegionID: id, Lat: lat, Lng: lng}
	err = q.QueryRow(ctx, `
		select r.name_en, coalesce(pl.name_en, r.name_en),
		       case when pl.name_en is null then r.name_th else pl.name_th end
		from regions r left join lateral (`+nearestPlaceSQL("$2::float8", "$3::float8")+`) pl on true
		where r.id = $1`, id, lat, lng,
	).Scan(&loc.Region, &loc.PlaceEn, &loc.PlaceTh)
	if errors.Is(err, pgx.ErrNoRows) {
		// The index is newer or older than the table; treat as outside.
		return nil, ErrOutsideRegions
	}
	if err != nil {
		return nil, fmt.Errorf("db: locate plot: %w", err)
	}
	return loc, nil
}

// BackfillPlots sets region_id, place_name and place_name_th on plots that
// lack them (claimed before the import, or before places were imported;
// place names are only filled, never replaced). Plots outside every region
// are left alone and counted in outside.
func (d *DB) BackfillPlots(ctx context.Context) (updated, outside int, err error) {
	rows, err := d.Pool.Query(ctx, `select cell from plots where region_id is null or place_name is null order by cell`)
	if err != nil {
		return 0, 0, fmt.Errorf("db: backfill plots: %w", err)
	}
	cells, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return 0, 0, fmt.Errorf("db: backfill plots: %w", err)
	}
	for _, cell := range cells {
		loc, err := d.LocatePlot(ctx, cell)
		if errors.Is(err, ErrOutsideRegions) {
			outside++
			continue
		}
		if err != nil {
			return updated, outside, err
		}
		_, err = d.Pool.Exec(ctx, `
			update plots set region_id = coalesce(region_id, $2),
			  place_name_th = case when place_name is null then $4 else place_name_th end,
			  place_name = coalesce(place_name, $3)
			where cell = $1`, cell, loc.RegionID, loc.PlaceEn, loc.PlaceTh)
		if err != nil {
			return updated, outside, fmt.Errorf("db: backfill plot %s: %w", cell, err)
		}
		updated++
	}
	return updated, outside, nil
}

// PlaceRadiusKm is the distance within which the nearest OSM place names a plot.
const PlaceRadiusKm = 5.0

// nearestPlaceSQL returns a subquery selecting name_en, name_th of the
// nearest place within PlaceRadiusKm of (lat, lng), using the (lat, lng)
// index for a box prefilter and an equirectangular distance.
func nearestPlaceSQL(lat, lng string) string {
	const kmPerDeg = 111.195
	dLat := PlaceRadiusKm / kmPerDeg
	return fmt.Sprintf(`
		select name_en, name_th from places
		where lat between %[1]s - %[3]g and %[1]s + %[3]g
		  and lng between %[2]s - %[3]g / cos(radians(%[1]s)) and %[2]s + %[3]g / cos(radians(%[1]s))
		  and (lat - %[1]s)^2 + ((lng - %[2]s) * cos(radians(%[1]s)))^2 <= %[4]g
		order by (lat - %[1]s)^2 + ((lng - %[2]s) * cos(radians(%[1]s)))^2, id
		limit 1`, lat, lng, dLat, dLat*dLat)
}

// Place is one row for ReplacePlaces.
type Place struct {
	OSMType  string
	OSMID    int64
	Kind     string
	NameEn   string
	NameTh   string
	Lat, Lng float64
	RegionID *int
}

// ReplacePlaces upserts places by (osm_type, osm_id) and deletes the places
// not in ps, in one transaction. Plots keep their copied names.
func (d *DB) ReplacePlaces(ctx context.Context, ps []Place) error {
	err := pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `create temp table new_places (
			osm_type char(1), osm_id bigint, kind text, name_en text, name_th text,
			lat double precision, lng double precision, region_id int) on commit drop`); err != nil {
			return err
		}
		rows := make([][]any, len(ps))
		for i, p := range ps {
			var th *string
			if p.NameTh != "" {
				th = &ps[i].NameTh
			}
			rows[i] = []any{p.OSMType, p.OSMID, p.Kind, p.NameEn, th, p.Lat, p.Lng, p.RegionID}
		}
		cols := []string{"osm_type", "osm_id", "kind", "name_en", "name_th", "lat", "lng", "region_id"}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"new_places"}, cols, pgx.CopyFromRows(rows)); err != nil {
			return fmt.Errorf("copy: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			insert into places (osm_type, osm_id, kind, name_en, name_th, lat, lng, region_id)
			select osm_type, osm_id, kind, name_en, name_th, lat, lng, region_id from new_places
			on conflict (osm_type, osm_id) do update set kind = excluded.kind, name_en = excluded.name_en,
			  name_th = excluded.name_th, lat = excluded.lat, lng = excluded.lng, region_id = excluded.region_id`); err != nil {
			return fmt.Errorf("upsert: %w", err)
		}
		_, err := tx.Exec(ctx, `delete from places p where not exists
			(select 1 from new_places n where n.osm_type = p.osm_type and n.osm_id = p.osm_id)`)
		return err
	})
	if err != nil {
		return fmt.Errorf("db: replace places: %w", err)
	}
	return nil
}
