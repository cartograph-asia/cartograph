package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// NewPhoto is a photo to be inserted by ClaimPhoto.
type NewPhoto struct {
	ID             uuid.UUID
	Cell           string // res-8 plot
	Parent5        string
	Parent6        string
	Surveyor       string    // name as typed or the account's name; kept in photos.surveyor
	SurveyorID     uuid.UUID // account; uuid.Nil stores null
	KeyLarge       string
	KeyMedium      string // empty stores null
	KeyThumb       string
	Width          int // of the large variant; 0 stores null
	Height         int
	Lat, Lng       float64
	TakenAt        *time.Time
	TakenLocal     *time.Time // camera wall clock (location UTC, not converted)
	TakenUTCOffset *int       // minutes east of UTC; nil if unknown
	UploadedAt     time.Time
}

func nullUUID(u uuid.UUID) *uuid.UUID {
	if u == uuid.Nil {
		return nil
	}
	return &u
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullInt(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}

// PlotSummary is one surveyed plot for the map layer.
type PlotSummary struct {
	Cell          string
	PhotoCount    int
	FirstSurveyor string
	ClaimedAt     time.Time
	ThumbKey      string // thumb of the first photo; may be empty
}

// PlotPhoto is one photo of a plot.
type PlotPhoto struct {
	ID         uuid.UUID
	Surveyor   string
	TakenAt    *time.Time
	UploadedAt time.Time
	KeyLarge   string
	KeyThumb   string
}

// PlotDetail is a surveyed plot with all its photos (oldest first).
type PlotDetail struct {
	Cell          string
	PhotoCount    int
	FirstSurveyor string
	ClaimedAt     time.Time
	Place         string // nearest place, else the region name; empty outside every region
	Region        string // region name_en; empty outside every region
	RegionSlug    string
	District      string // district name_en; empty where none is imported
	Number        int    // running number in the region; 0 if none
	Photos        []PlotPhoto
}

// ClaimPhoto records a survey in one transaction: it upserts the plot
// (first_photo, first_surveyor(_id), claimed_at, region and place name are
// only set when the plot row is new; photo_count is incremented), inserts the
// photo and updates the region's claimed_cells and last_surveyed_at. The plot
// is written first because photos.cell references plots(cell). claimed
// reports whether this photo created the plot. A plot whose centre is outside
// every region is rejected with ErrOutsideRegions.
func (d *DB) ClaimPhoto(ctx context.Context, p NewPhoto) (claimed bool, err error) {
	loc, err := d.locatePlot(ctx, d.Pool, p.Cell)
	if err != nil {
		if errors.Is(err, ErrOutsideRegions) {
			return false, err
		}
		return false, fmt.Errorf("db: claim photo: %w", err)
	}
	err = pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			insert into plots (cell, first_photo, first_surveyor, first_surveyor_id, claimed_at, photo_count,
			                   parent5, parent6, region_id, place_name, place_name_th, district_id)
			values ($1, $2, $3, $4, $5, 1, $6, $7, $8, $9, $10, $11)
			on conflict (cell) do update set photo_count = plots.photo_count + 1
			returning (xmax = 0)`,
			p.Cell, p.ID, p.Surveyor, nullUUID(p.SurveyorID), p.UploadedAt, p.Parent5, p.Parent6,
			loc.RegionID, loc.PlaceEn, loc.PlaceTh, loc.DistrictID,
		).Scan(&claimed)
		if err != nil {
			return fmt.Errorf("upsert plot: %w", err)
		}
		if claimed {
			if err := numberPlot(ctx, tx, loc.RegionID, p.Cell); err != nil {
				return fmt.Errorf("number plot: %w", err)
			}
		}
		_, err = tx.Exec(ctx, `
			insert into photos (id, cell, surveyor, surveyor_id, key_large, key_medium, key_thumb,
			                    width, height, lat, lng, taken_at, taken_at_local, taken_utc_offset, uploaded_at)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
			p.ID, p.Cell, p.Surveyor, nullUUID(p.SurveyorID), p.KeyLarge, nullString(p.KeyMedium), p.KeyThumb,
			nullInt(p.Width), nullInt(p.Height), p.Lat, p.Lng, p.TakenAt, p.TakenLocal, p.TakenUTCOffset, p.UploadedAt,
		)
		if err != nil {
			return fmt.Errorf("insert photo: %w", err)
		}
		// The plot's stored region: for a plot claimed before the import it
		// may be null until cmd/regions backfills it.
		_, err = tx.Exec(ctx, `
			update regions set
			  claimed_cells = claimed_cells + $2::int,
			  last_surveyed_at = greatest(last_surveyed_at, $3)
			where id = (select region_id from plots where cell = $1)`,
			p.Cell, map[bool]int{true: 1}[claimed], p.UploadedAt)
		if err != nil {
			return fmt.Errorf("update region: %w", err)
		}
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("db: claim photo: %w", err)
	}
	return claimed, nil
}

// Plots returns the surveyed plots among cells.
func (d *DB) Plots(ctx context.Context, cells []string) ([]PlotSummary, error) {
	rows, err := d.Pool.Query(ctx, `
		select p.cell, p.photo_count, coalesce(p.first_surveyor, ''), p.claimed_at, coalesce(ph.key_thumb, '')
		from plots p
		left join photos ph on ph.id = p.first_photo
		where p.cell = any($1) and p.photo_count > 0
		order by p.cell`, cells)
	if err != nil {
		return nil, fmt.Errorf("db: plots: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (PlotSummary, error) {
		var s PlotSummary
		err := r.Scan(&s.Cell, &s.PhotoCount, &s.FirstSurveyor, &s.ClaimedAt, &s.ThumbKey)
		return s, err
	})
	if err != nil {
		return nil, fmt.Errorf("db: plots: %w", err)
	}
	return out, nil
}

// Plot returns one surveyed plot with its photos, or ErrNotFound.
func (d *DB) Plot(ctx context.Context, cell string) (*PlotDetail, error) {
	var p PlotDetail
	err := d.Pool.QueryRow(ctx, `
		select pl.cell, pl.photo_count, coalesce(pl.first_surveyor, ''), pl.claimed_at,
		       coalesce(pl.place_name, r.name_en, ''), coalesce(r.name_en, ''), coalesce(r.slug, ''), coalesce(n.number, 0),
		       coalesce(dr.name_en, '')
		from plots pl left join regions r on r.id = pl.region_id
		left join regions dr on dr.id = pl.district_id
		left join plot_numbers n on n.cell = pl.cell
		where pl.cell = $1 and pl.photo_count > 0`, cell,
	).Scan(&p.Cell, &p.PhotoCount, &p.FirstSurveyor, &p.ClaimedAt, &p.Place, &p.Region, &p.RegionSlug, &p.Number, &p.District)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("db: plot: %w", err)
	}
	rows, err := d.Pool.Query(ctx, `
		select id, surveyor, taken_at, uploaded_at, key_large, key_thumb
		from photos where cell = $1
		order by uploaded_at, id`, cell)
	if err != nil {
		return nil, fmt.Errorf("db: plot photos: %w", err)
	}
	p.Photos, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (PlotPhoto, error) {
		var ph PlotPhoto
		err := r.Scan(&ph.ID, &ph.Surveyor, &ph.TakenAt, &ph.UploadedAt, &ph.KeyLarge, &ph.KeyThumb)
		return ph, err
	})
	if err != nil {
		return nil, fmt.Errorf("db: plot photos: %w", err)
	}
	return &p, nil
}

// Coverage returns the number of surveyed plots per parent cell for the
// given parents at res (5 or 6). Parents without surveyed plots are absent.
func (d *DB) Coverage(ctx context.Context, res int, parents []string) (map[string]int64, error) {
	var q string
	switch res {
	case 5:
		q = `select parent5, count(*) from plots where parent5 = any($1) and photo_count > 0 group by parent5`
	case 6:
		q = `select parent6, count(*) from plots where parent6 = any($1) and photo_count > 0 group by parent6`
	default:
		return nil, fmt.Errorf("db: coverage: unsupported resolution %d", res)
	}
	rows, err := d.Pool.Query(ctx, q, parents)
	if err != nil {
		return nil, fmt.Errorf("db: coverage: %w", err)
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var cell string
		var n int64
		if err := rows.Scan(&cell, &n); err != nil {
			return nil, fmt.Errorf("db: coverage: %w", err)
		}
		out[cell] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: coverage: %w", err)
	}
	return out, nil
}
