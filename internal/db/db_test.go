package db

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cartograph-asia/cartograph/migrations"
)

// testDB connects to TEST_DATABASE_URL in a fresh schema and applies the
// migrations. The test is skipped when the variable is not set.
func testDB(t *testing.T) *DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())

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
	d := &DB{Pool: pool}
	t.Cleanup(d.Close)
	if err := d.Migrate(ctx, migrations.FS); err != nil {
		t.Fatal(err)
	}
	// Running again is a no-op.
	if err := d.Migrate(ctx, migrations.FS); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	return d
}

func newPhoto(cell, surveyor string, at time.Time) NewPhoto {
	id := uuid.New()
	return NewPhoto{
		ID: id, Cell: cell, Parent5: "85" + cell[2:5] + "p5", Parent6: "86" + cell[2:5] + "p6",
		Surveyor: surveyor, KeyLarge: id.String() + "_large.jpg", KeyThumb: id.String() + "_thumb.jpg",
		Lat: 12.85, Lng: 100.9, UploadedAt: at,
	}
}

func TestClaimAndRead(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	t0 := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	cell := "88651a2b2dfffff"

	p1 := newPhoto(cell, "Ada", t0)
	taken := t0.Add(-time.Hour)
	p1.TakenAt = &taken
	claimed, err := d.ClaimPhoto(ctx, p1)
	if err != nil || !claimed {
		t.Fatalf("first claim: %v %v", claimed, err)
	}
	claimed, err = d.ClaimPhoto(ctx, newPhoto(cell, "Grace", t0.Add(time.Minute)))
	if err != nil || claimed {
		t.Fatalf("second claim: %v %v", claimed, err)
	}

	p, err := d.Plot(ctx, cell)
	if err != nil {
		t.Fatal(err)
	}
	if p.PhotoCount != 2 || p.FirstSurveyor != "Ada" || !p.ClaimedAt.Equal(t0) || len(p.Photos) != 2 {
		t.Fatalf("plot: %+v", p)
	}
	if p.Photos[0].ID != p1.ID || p.Photos[0].TakenAt == nil || !p.Photos[0].TakenAt.Equal(taken) || p.Photos[1].TakenAt != nil {
		t.Fatalf("photos: %+v", p.Photos)
	}

	if _, err := d.Plot(ctx, "88651a2b2bfffff"); err != ErrNotFound {
		t.Fatalf("blank plot: %v", err)
	}

	list, err := d.Plots(ctx, []string{cell, "88651a2b2bfffff"})
	if err != nil || len(list) != 1 || list[0].PhotoCount != 2 || list[0].ThumbKey != p1.KeyThumb {
		t.Fatalf("plots: %+v %v", list, err)
	}

	cov, err := d.Coverage(ctx, 6, []string{p1.Parent6, "other"})
	if err != nil || cov[p1.Parent6] != 1 || cov["other"] != 0 {
		t.Fatalf("coverage: %v %v", cov, err)
	}
}

func TestConcurrentClaimsSinglePlot(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	cell := "88651a2b2dfffff"
	const n = 10
	var wg sync.WaitGroup
	results := make(chan bool, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := d.ClaimPhoto(ctx, newPhoto(cell, fmt.Sprint("s", i), time.Now()))
			if err != nil {
				t.Error(err)
			}
			results <- claimed
		}()
	}
	wg.Wait()
	close(results)
	claims := 0
	for c := range results {
		if c {
			claims++
		}
	}
	p, err := d.Plot(ctx, cell)
	if err != nil {
		t.Fatal(err)
	}
	if claims != 1 || p.PhotoCount != n || len(p.Photos) != n {
		t.Fatalf("claims=%d photo_count=%d photos=%d", claims, p.PhotoCount, len(p.Photos))
	}
}
