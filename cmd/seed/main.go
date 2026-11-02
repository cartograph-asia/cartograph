// Command seed fills a database that has the regions and places imported
// with fixture content for local development: a handful of Surveyor
// accounts (name@example.test), synthetic geotagged JPEGs over Pattaya run
// through the real upload pipeline and stored via internal/storage, and one
// featured photo. It is idempotent: ids are deterministic and existing rows
// are left alone, so a second run does nothing.
//
// Environment as for the api: DATABASE_URL, S3_ENDPOINT, S3_BUCKET,
// S3_ACCESS_KEY, S3_SECRET_KEY, S3_REGION, S3_USE_SSL, S3_USE_PATH_STYLE,
// IMG_BASE_URL (optional here). Migrations are applied first.
//
//	regions -pbf testdata/pattaya.osm.gz -water testdata/pattaya-water.shp -bbox 100.82,12.82,100.98,13.02
//	places  -pbf testdata/pattaya.osm.gz -bbox 100.82,12.82,100.98,13.02
//	seed
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/storage"
	"github.com/cartograph-asia/cartograph/migrations"
)

func main() {
	n := flag.Int("photos", 30, "number of synthetic photos")
	flag.Parse()
	if err := run(*n); err != nil {
		slog.Error("seed", "err", err)
		os.Exit(1)
	}
}

func run(n int) error {
	var missing []string
	need := func(name string) string {
		v := strings.TrimSpace(os.Getenv(name))
		if v == "" {
			missing = append(missing, name)
		}
		return v
	}
	dbURL := need("DATABASE_URL")
	cfg := storage.S3Config{
		Endpoint:  need("S3_ENDPOINT"),
		Bucket:    need("S3_BUCKET"),
		AccessKey: need("S3_ACCESS_KEY"),
		SecretKey: need("S3_SECRET_KEY"),
		Region:    envOr("S3_REGION", "us-east-1"),
		BaseURL:   os.Getenv("IMG_BASE_URL"),
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	var err error
	if cfg.UseSSL, err = envBool("S3_USE_SSL"); err != nil {
		return err
	}
	if cfg.PathStyle, err = envBool("S3_USE_PATH_STYLE"); err != nil {
		return err
	}
	store, err := storage.NewS3(cfg)
	if err != nil {
		return err
	}
	ctx := context.Background()
	d, err := db.Connect(ctx, dbURL, 30*time.Second)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Migrate(ctx, migrations.FS); err != nil {
		return err
	}
	start := time.Now()
	st, err := Seed(ctx, d, store, time.Now(), n)
	if err != nil {
		return err
	}
	slog.Info("seeded", "surveyors_created", st.Surveyors, "photos_created", st.Photos, "photos_present", st.Skipped,
		"featured_set", st.Featured, "took", time.Since(start).Round(time.Millisecond))
	filled, err := SeedCameras(ctx, d)
	if err != nil {
		return err
	}
	slog.Info("seeded cameras", "photos_filled", filled)
	return nil
}

func envOr(name, def string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return def
}

func envBool(name string) (bool, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return false, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, errors.New(name + " must be true or false")
	}
	return b, nil
}
