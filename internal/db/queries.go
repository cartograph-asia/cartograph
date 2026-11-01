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
	ID         uuid.UUID
	Cell       string // res-8 plot
	Parent5    string
	Parent6    string
	Surveyor   string
	KeyLarge   string
	KeyThumb   string
	Lat, Lng   float64
	TakenAt    *time.Time
	UploadedAt time.Time
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
	Photos        []PlotPhoto
}

// ClaimPhoto records a survey in one transaction: it upserts the plot
// (first_photo, first_surveyor and claimed_at are only set when the plot row
// is new; photo_count is incremented) and inserts the photo. The plot is
// written first because photos.cell references plots(cell). claimed reports
// whether this photo created the plot.
func (d *DB) ClaimPhoto(ctx context.Context, p NewPhoto) (claimed bool, err error) {
	err = pgx.BeginFunc(ctx, d.Pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			insert into plots (cell, first_photo, first_surveyor, claimed_at, photo_count, parent5, parent6)
			values ($1, $2, $3, $4, 1, $5, $6)
			on conflict (cell) do update set photo_count = plots.photo_count + 1
			returning (xmax = 0)`,
			p.Cell, p.ID, p.Surveyor, p.UploadedAt, p.Parent5, p.Parent6,
		).Scan(&claimed)
		if err != nil {
			return fmt.Errorf("upsert plot: %w", err)
		}
		_, err = tx.Exec(ctx, `
			insert into photos (id, cell, surveyor, key_large, key_thumb, lat, lng, taken_at, uploaded_at)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			p.ID, p.Cell, p.Surveyor, p.KeyLarge, p.KeyThumb, p.Lat, p.Lng, p.TakenAt, p.UploadedAt,
		)
		if err != nil {
			return fmt.Errorf("insert photo: %w", err)
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
		select cell, photo_count, coalesce(first_surveyor, ''), claimed_at
		from plots where cell = $1 and photo_count > 0`, cell,
	).Scan(&p.Cell, &p.PhotoCount, &p.FirstSurveyor, &p.ClaimedAt)
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
