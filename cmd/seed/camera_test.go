package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/importer"
	"github.com/cartograph-asia/cartograph/internal/photos"
	"github.com/cartograph-asia/cartograph/internal/storage"
)

func TestFakeCameras(t *testing.T) {
	labels := map[string]bool{}
	for _, fc := range fakeCameras {
		label := photos.Label(fc.make, fc.model)
		if label == "" || labels[label] {
			t.Errorf("label %q of %s %s is empty or twice", label, fc.make, fc.model)
		}
		labels[label] = true
		for _, m := range fc.modes {
			if m.focalMM <= 0 || m.focalMM >= 5000 || m.equiv <= 0 || m.fNumber <= 0 || m.fNumber >= 100 || len(m.exposures) == 0 {
				t.Errorf("%s: mode %+v", label, m)
			}
		}
	}
	// The label cases that make statistics hard are there on purpose.
	for _, want := range []string{"Apple iPhone 17 Pro", "samsung SM-S921B", "NIKON Z 6_3", "RICOH GR III", "OLYMPUS OM-1", "Canon EOS R6m2"} {
		if !labels[want] {
			t.Errorf("no fake camera labelled %q", want)
		}
	}

	// A photo always gets the same data; a Surveyor mostly uses one camera.
	id := uuid.MustParse("6f1c2a52-3b8e-4c55-9d2e-7a0c6f2b9e11")
	a, b := cameraFor("mali", id), cameraFor("mali", id)
	if *a.Flash != *b.Flash || a.Label != b.Label || a.ISO != b.ISO || a.ExposureS != b.ExposureS || a.FocalMM != b.FocalMM {
		t.Errorf("not deterministic: %+v %+v", a, b)
	}
	own, other := fakeCameras[primary("mali")], 0
	for i := range 1000 {
		c := cameraFor("mali", uuid.NewSHA1(namespace, []byte(fmt.Sprint(i))))
		if c.Label != photos.Label(own.make, own.model) {
			other++
		}
		if c.ISO <= 0 || c.ExposureS <= 0 || c.ExposureS > 1 || c.FNumber <= 0 {
			t.Fatalf("values %+v", c)
		}
	}
	if other < 100 || other > 300 {
		t.Errorf("%d of 1000 photos with another camera than the Surveyor's own, want about 200", other)
	}
}

func TestSeedCameras(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	osmPath, waterPath := importer.Fixture()
	if _, err := importer.Regions(ctx, d, importer.RegionOptions{PBF: osmPath, Water: waterPath, BBox: importer.FixtureBBox}); err != nil {
		t.Fatal(err)
	}
	if _, err := importer.Places(ctx, d, importer.PlaceOptions{PBF: osmPath, BBox: importer.FixtureBBox}); err != nil {
		t.Fatal(err)
	}
	store := storage.NewMemory("http://img.test")
	now := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	if _, err := Seed(ctx, d, store, now, 30); err != nil {
		t.Fatal(err)
	}

	// A real account's photo is not touched; one that already has data keeps it.
	real := uuid.New()
	if _, err := d.Pool.Exec(ctx, `insert into surveyors (id, name, email) values ($1, 'real-person', 'person@example.org')`, real); err != nil {
		t.Fatal(err)
	}
	cell, _ := firstCell(t, d)
	realPhoto := uuid.New()
	if _, err := d.ClaimPhoto(ctx, db.NewPhoto{ID: realPhoto, Cell: cell, Parent5: "85", Parent6: "86", Surveyor: "real-person", SurveyorID: real,
		KeyLarge: "l", KeyThumb: "t", Lat: 12.9, Lng: 100.9, UploadedAt: now}); err != nil {
		t.Fatal(err)
	}
	var had uuid.UUID
	if err := d.Pool.QueryRow(ctx, `select id from photos where surveyor = 'noi-pattaya' order by id limit 1`).Scan(&had); err != nil {
		t.Fatal(err)
	}
	keep := &photos.Camera{Label: "Kept Camera", ISO: 100}
	if err := d.AddCamera(ctx, had, keep); err != nil {
		t.Fatal(err)
	}

	n, err := SeedCameras(ctx, d)
	if err != nil || n != 29 {
		t.Fatalf("first run filled %d: %v", n, err)
	}
	if n, err = SeedCameras(ctx, d); err != nil || n != 0 {
		t.Fatalf("second run filled %d: %v", n, err)
	}
	var rows int
	if err := d.Pool.QueryRow(ctx, `select count(*) from photo_camera`).Scan(&rows); err != nil || rows != 30 {
		t.Fatalf("%d rows, want 30 (30 seeded photos, the real one has none): %v", rows, err)
	}
	var label string
	if err := d.Pool.QueryRow(ctx, `select camera from photo_camera where photo_id = $1`, had).Scan(&label); err != nil || label != "Kept Camera" {
		t.Errorf("existing data replaced: %q %v", label, err)
	}
	if err := d.Pool.QueryRow(ctx, `select count(*) from photo_camera where photo_id = $1`, realPhoto).Scan(&rows); err != nil || rows != 0 {
		t.Errorf("a real account's photo got camera data: %d %v", rows, err)
	}

	// The photo page shows it.
	pg, err := d.Views().WithNow(func() time.Time { return now }).Photo(ctx, had.String())
	if err != nil || pg.Camera == nil || pg.Camera.Label != "Kept Camera" {
		t.Errorf("page camera %+v %v", pg.Camera, err)
	}
}

// firstCell is a plot of the fixture region.
func firstCell(t *testing.T, d *db.DB) (string, int) {
	t.Helper()
	var cell string
	var n int
	if err := d.Pool.QueryRow(context.Background(), `select cell, number from plot_numbers order by number limit 1`).Scan(&cell, &n); err != nil {
		t.Fatal(err)
	}
	return cell, n
}
