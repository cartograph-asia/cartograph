package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/importer"
	"github.com/cartograph-asia/cartograph/internal/photos"
	"github.com/cartograph-asia/cartograph/internal/storage"
	"github.com/cartograph-asia/cartograph/migrations"
)

func TestSynthPhotoExif(t *testing.T) {
	taken := time.Date(2026, 10, 3, 23, 30, 0, 0, time.UTC)
	b, err := synthPhoto(1, 12.9346, 100.8901, taken)
	if err != nil {
		t.Fatal(err)
	}
	m := photos.ReadMeta(b)
	if !m.HasGPS || math.Abs(m.Lat-12.9346) > 1e-6 || math.Abs(m.Lng-100.8901) > 1e-6 {
		t.Fatalf("gps %+v", m)
	}
	if m.TakenAt == nil || !m.TakenAt.Equal(taken) {
		t.Fatalf("taken %v", m.TakenAt)
	}
	again, _ := synthPhoto(1, 12.9346, 100.8901, taken)
	if string(again) != string(b) {
		t.Fatal("not deterministic")
	}
	if _, err := photos.Process(b); err != nil {
		t.Fatal(err)
	}
}

// testDB opens a fresh, migrated schema in TEST_DATABASE_URL.
func testDB(t *testing.T) *db.DB {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("seed_%d", time.Now().UnixNano())
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "create schema "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "drop schema "+schema+" cascade")
		admin.Close()
	})
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	d := &db.DB{Pool: pool}
	t.Cleanup(d.Close)
	if err := d.Migrate(ctx, migrations.FS); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestSeedIdempotent(t *testing.T) {
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
	st, err := Seed(ctx, d, store, now, 30)
	if err != nil {
		t.Fatal(err)
	}
	if st.Surveyors != 5 || st.Photos != 30 || !st.Featured || len(store.Keys()) != 90 {
		t.Fatalf("first run %+v, %d keys", st, len(store.Keys()))
	}
	st, err = Seed(ctx, d, store, now.Add(time.Hour), 30)
	if err != nil {
		t.Fatal(err)
	}
	if st.Surveyors != 0 || st.Photos != 0 || st.Skipped != 30 || st.Featured || len(store.Keys()) != 90 {
		t.Fatalf("second run %+v", st)
	}

	h, err := d.Views().WithNow(func() time.Time { return now }).Home(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if h.Counters.Photos != 30 || h.Counters.Plots != 18 || h.Counters.Surveyors != 5 || h.Featured == nil || !h.Featured.Editorial ||
		len(h.Latest) != 24 || len(h.NewSurveyors) == 0 || len(h.Regions) != 1 {
		t.Fatalf("home %+v", h)
	}
	for _, p := range h.Latest {
		if p.Plot.Place == "" || p.Plot.Region.Slug != "chon-buri" || p.TakenAt == nil || p.Width == 0 {
			t.Fatalf("item %+v", p)
		}
	}
}
