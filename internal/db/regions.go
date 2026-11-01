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

// plotNumberLock is the first key of the transaction-level advisory lock
// that serialises number assignment within a region (the second key is the
// region id), shared by the import and by claims.
const plotNumberLock = 0x706c6f74 // "plot"

// AssignPlotNumbers gives each cell in cells that has no number yet the next
// free number in region, in the order given (the importer passes them
// sorted along a Hilbert curve). Existing numbers are never changed, so a
// re-import only appends. It returns how many numbers were added.
func (d *DB) AssignPlotNumbers(ctx context.Context, regionID int, cells []string) (int, error) {
	var added int64
	err := pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock($1, $2)`, plotNumberLock, regionID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `create temp table new_cells (cell text, ord int) on commit drop`); err != nil {
			return err
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"new_cells"}, []string{"cell", "ord"},
			pgx.CopyFromSlice(len(cells), func(i int) ([]any, error) { return []any{cells[i], i}, nil })); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			insert into plot_numbers (cell, region_id, number)
			select n.cell, $1, m.top + row_number() over (order by n.ord)
			from (select cell, ord from new_cells c
			      where not exists (select 1 from plot_numbers p where p.cell = c.cell)) n,
			     (select coalesce(max(number), 0) as top from plot_numbers where region_id = $1) m`, regionID)
		added = tag.RowsAffected()
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("db: assign plot numbers: %w", err)
	}
	return int(added), nil
}

// numberPlot gives cell the next free number in region unless it has one;
// called inside the claim transaction for plots outside the numbered land
// cells (e.g. a beach cut off by the coastline).
func numberPlot(ctx context.Context, tx pgx.Tx, regionID int, cell string) error {
	var has bool
	if err := tx.QueryRow(ctx, `select exists (select 1 from plot_numbers where cell = $1)`, cell).Scan(&has); err != nil || has {
		return err
	}
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock($1, $2)`, plotNumberLock, regionID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		insert into plot_numbers (cell, region_id, number)
		select $2, $1, coalesce(max(number), 0) + 1 from plot_numbers where region_id = $1
		on conflict (cell) do nothing`, regionID, cell)
	return err
}

// NumberPlots gives every surveyed plot with a region but no number the next
// free number in its region, oldest claim first (plots claimed before the
// import, or outside the land cells). It returns how many were numbered.
func (d *DB) NumberPlots(ctx context.Context) (int, error) {
	rows, err := d.Pool.Query(ctx, `
		select pl.cell, pl.region_id from plots pl
		where pl.region_id is not null and not exists (select 1 from plot_numbers n where n.cell = pl.cell)
		order by pl.claimed_at, pl.cell`)
	if err != nil {
		return 0, fmt.Errorf("db: number plots: %w", err)
	}
	type todo struct {
		cell   string
		region int
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (todo, error) {
		var t todo
		return t, r.Scan(&t.cell, &t.region)
	})
	if err != nil {
		return 0, fmt.Errorf("db: number plots: %w", err)
	}
	for _, t := range list {
		err := pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error { return numberPlot(ctx, tx, t.region, t.cell) })
		if err != nil {
			return 0, fmt.Errorf("db: number plot %s: %w", t.cell, err)
		}
	}
	return len(list), nil
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
	Near     bool    // PlaceEn is an OSM place, not the region fallback
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
		       case when pl.name_en is null then r.name_th else pl.name_th end,
		       pl.name_en is not null
		from regions r left join lateral (`+nearestPlaceSQL("$2::float8", "$3::float8")+`) pl on true
		where r.id = $1`, id, lat, lng,
	).Scan(&loc.Region, &loc.PlaceEn, &loc.PlaceTh, &loc.Near)
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
// place names are only filled, never replaced). Only a real nearby place is
// written: the region fallback would otherwise stick before cmd/places has
// run, and the pages fall back to the region name anyway. Plots outside
// every region are left alone and counted in outside.
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
		var placeEn *string
		placeTh := loc.PlaceTh
		if loc.Near {
			placeEn = &loc.PlaceEn
		} else {
			placeTh = nil
		}
		_, err = d.Pool.Exec(ctx, `
			update plots set region_id = coalesce(region_id, $2),
			  place_name_th = case when place_name is null then $4 else place_name_th end,
			  place_name = coalesce(place_name, $3)
			where cell = $1`, cell, loc.RegionID, placeEn, placeTh)
		if err != nil {
			return updated, outside, fmt.Errorf("db: backfill plot %s: %w", cell, err)
		}
		updated++
	}
	return updated, outside, nil
}

// PlaceRadiusKm is the distance within which a village, town or city names
// a plot; FinePlaceRadiusKm the one for neighbourhoods, quarters and suburbs.
const (
	PlaceRadiusKm     = 5.0
	FinePlaceRadiusKm = 2.0
)

// nearestPlaceSQL returns a subquery selecting name_en, name_th of the place
// that names a plot at (lat, lng): the nearest neighbourhood within
// FinePlaceRadiusKm, else the nearest quarter, else suburb (finest first);
// else the nearest village or town within PlaceRadiusKm; else the nearest
// city within PlaceRadiusKm. It uses the (lat, lng) index for a box
// prefilter and an equirectangular distance.
func nearestPlaceSQL(lat, lng string) string {
	const kmPerDeg = 111.195
	r := PlaceRadiusKm / kmPerDeg
	fine := FinePlaceRadiusKm / kmPerDeg
	return fmt.Sprintf(`
		select name_en, name_th from (
			select name_en, name_th, id, kind,
			       (lat - %[1]s)^2 + ((lng - %[2]s) * cos(radians(%[1]s)))^2 as d2
			from places
			where lat between %[1]s - %[3]g and %[1]s + %[3]g
			  and lng between %[2]s - %[3]g / cos(radians(%[1]s)) and %[2]s + %[3]g / cos(radians(%[1]s))
		) p
		where (kind in ('neighbourhood', 'quarter', 'suburb') and d2 <= %[5]g)
		   or (kind in ('village', 'town', 'city') and d2 <= %[4]g)
		order by case kind when 'neighbourhood' then 1 when 'quarter' then 2 when 'suburb' then 3
		                   when 'village' then 4 when 'town' then 4 else 5 end,
		         d2, id
		limit 1`, lat, lng, r, r*r, fine*fine)
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
