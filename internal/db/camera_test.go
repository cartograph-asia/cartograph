package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/cartograph-asia/cartograph/internal/photos"
)

func yes() *bool { b := true; return &b }

func TestClaimStoresCamera(t *testing.T) {
	d := testDB(t)
	testRegion(t, d)
	ctx := context.Background()
	t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	cell := "88658bd11dfffff"

	with := newPhoto(cell, "Ada", t0)
	with.Camera = &photos.Camera{Make: "Apple", Model: "iPhone 17 Pro", Label: "Apple iPhone 17 Pro", Lens: "back camera",
		FocalMM: 6.8, Focal35MM: 48, FNumber: 1.8, ExposureS: 1.0 / 250, ISO: 64, Flash: yes()}
	partial := newPhoto(cell, "Ada", t0.Add(time.Minute))
	partial.Camera = &photos.Camera{Label: "Unknown camera", ISO: 200} // a make and model can be missing
	without := newPhoto(cell, "Ada", t0.Add(2*time.Minute))
	for _, p := range []NewPhoto{with, partial, without} {
		if _, err := d.ClaimPhoto(ctx, p); err != nil {
			t.Fatal(err)
		}
	}

	var n int
	if err := d.Pool.QueryRow(ctx, `select count(*) from photo_camera`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("rows %d %v", n, err)
	}
	var label, make, model string
	var focal, fnum, exp float64
	var f35 int
	var iso int
	var flash *bool
	if err := d.Pool.QueryRow(ctx, `select camera, make, model, focal_length_mm, focal_length_35mm, f_number, exposure_time_s, iso, flash_fired
		from photo_camera where photo_id = $1`, with.ID).Scan(&label, &make, &model, &focal, &f35, &fnum, &exp, &iso, &flash); err != nil {
		t.Fatal(err)
	}
	if label != "Apple iPhone 17 Pro" || make != "Apple" || model != "iPhone 17 Pro" || focal != 6.8 || f35 != 48 || fnum != 1.8 || exp != 1.0/250 || iso != 64 || flash == nil || !*flash {
		t.Errorf("stored %q %q %q %v %v %v %v %v %v", label, make, model, focal, f35, fnum, exp, iso, flash)
	}
	// Missing values are null, not zero.
	var nulls int
	if err := d.Pool.QueryRow(ctx, `select count(*) from photo_camera where photo_id = $1
		and make is null and model is null and lens is null and focal_length_mm is null and focal_length_35mm is null
		and f_number is null and exposure_time_s is null and flash_fired is null and iso = 200`, partial.ID).Scan(&nulls); err != nil || nulls != 1 {
		t.Errorf("partial row: %d %v", nulls, err)
	}

	// The page read returns it, and nil for a photo without.
	v := d.Views()
	pg, err := v.Photo(ctx, with.ID.String())
	if err != nil || pg.Camera == nil || pg.Camera.Line() != "Apple iPhone 17 Pro · 6.8 mm (48 mm equiv.) · f/1.8 · 1/250 s · ISO 64" {
		t.Fatalf("page camera %+v %v", pg.Camera, err)
	}
	if pg, err = v.Photo(ctx, without.ID.String()); err != nil || pg.Camera != nil {
		t.Errorf("no camera: %+v %v", pg.Camera, err)
	}

	// The row goes with the photo.
	if _, err := d.Pool.Exec(ctx, `delete from photos where id = $1`, with.ID); err != nil {
		t.Fatal(err)
	}
	if err := d.Pool.QueryRow(ctx, `select count(*) from photo_camera where photo_id = $1`, with.ID).Scan(&n); err != nil || n != 0 {
		t.Errorf("row left behind: %d %v", n, err)
	}
}

// The checks of the table refuse absurd numbers; the cleaning in
// internal/photos keeps them from ever arriving.
func TestCameraChecks(t *testing.T) {
	d := testDB(t)
	testRegion(t, d)
	ctx := context.Background()
	p := newPhoto("88658bd11dfffff", "Ada", time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC))
	if _, err := d.ClaimPhoto(ctx, p); err != nil {
		t.Fatal(err)
	}
	for _, col := range []string{"iso = -5", "f_number = 0", "exposure_time_s = 1e9", "focal_length_mm = 100000", "camera = ''"} {
		_, err := d.Pool.Exec(ctx, `insert into photo_camera (photo_id, camera) values ($1, 'x')`, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		_, err = d.Pool.Exec(ctx, `update photo_camera set `+col+` where photo_id = $1`, p.ID)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23514" {
			t.Errorf("%s: want a check violation, got %v", col, err)
		}
		if _, err := d.Pool.Exec(ctx, `delete from photo_camera where photo_id = $1`, p.ID); err != nil {
			t.Fatal(err)
		}
	}
}
