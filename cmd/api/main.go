// Command api is the Cartograph backend: a JSON API for plots, photos and
// coverage. It is configured entirely through environment variables
// (see deploy/.env.example).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cartograph-asia/cartograph/internal/api"
	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/storage"
	"github.com/cartograph-asia/cartograph/migrations"
)

type config struct {
	addr        string
	databaseURL string
	s3          storage.S3Config
	maxUploadMB int
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := db.Connect(ctx, cfg.databaseURL, 30*time.Second)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := database.Migrate(ctx, migrations.FS); err != nil {
		return err
	}

	store, err := storage.NewS3(cfg.s3)
	if err != nil {
		return err
	}
	checkBucket(ctx, store, cfg.s3.Bucket)

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           api.New(database, store, api.Config{MaxUploadMB: cfg.maxUploadMB}).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute, // large uploads on slow mobile connections
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.addr)
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down")
		sctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(sctx)
	}
	return nil
}

// checkBucket logs whether the bucket exists. It never creates it; the
// deployment does that (the mc container in compose).
func checkBucket(ctx context.Context, store *storage.S3, bucket string) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ok, err := store.BucketExists(cctx)
	switch {
	case err != nil:
		slog.Warn("could not check photo bucket; uploads will fail until storage is reachable", "bucket", bucket, "err", err)
	case !ok:
		slog.Warn("photo bucket does not exist; uploads will fail until it is created", "bucket", bucket)
	default:
		slog.Info("photo bucket ok", "bucket", bucket)
	}
}

func loadConfig() (config, error) {
	var missing []string
	need := func(name string) string {
		v := strings.TrimSpace(os.Getenv(name))
		if v == "" {
			missing = append(missing, name)
		}
		return v
	}
	c := config{
		addr:        envOr("API_ADDR", ":8000"),
		databaseURL: need("DATABASE_URL"),
		s3: storage.S3Config{
			Endpoint:  need("S3_ENDPOINT"),
			Bucket:    need("S3_BUCKET"),
			AccessKey: need("S3_ACCESS_KEY"),
			SecretKey: need("S3_SECRET_KEY"),
			Region:    envOr("S3_REGION", "us-east-1"),
			BaseURL:   need("IMG_BASE_URL"),
		},
	}
	var errs []error
	if len(missing) > 0 {
		errs = append(errs, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", ")))
	}
	var err error
	if c.s3.PathStyle, err = envBool("S3_USE_PATH_STYLE", false); err != nil {
		errs = append(errs, err)
	}
	if c.s3.UseSSL, err = envBool("S3_USE_SSL", false); err != nil {
		errs = append(errs, err)
	}
	if c.maxUploadMB, err = strconv.Atoi(envOr("MAX_UPLOAD_MB", "25")); err != nil || c.maxUploadMB <= 0 {
		errs = append(errs, errors.New("MAX_UPLOAD_MB must be a positive integer"))
	}
	if strings.Contains(c.s3.Endpoint, "://") {
		errs = append(errs, errors.New("S3_ENDPOINT must be host:port without a scheme; use S3_USE_SSL for https"))
	}
	return c, errors.Join(errs...)
}

func envOr(name, def string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return def
}

func envBool(name string, def bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false, got %q", name, v)
	}
	return b, nil
}
