package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config configures an S3-compatible store.
type S3Config struct {
	Endpoint  string // host:port, no scheme
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
	PathStyle bool   // true for MinIO/Garage; false uses virtual-hosted-style bucket lookup
	BaseURL   string // public base URL for objects (IMG_BASE_URL)
}

// S3 is a Store backed by any S3-compatible object store.
type S3 struct {
	client  *minio.Client
	bucket  string
	baseURL string
}

// NewS3 creates an S3 store. It does not contact the server.
func NewS3(cfg S3Config) (*S3, error) {
	lookup := minio.BucketLookupDNS
	if cfg.PathStyle {
		lookup = minio.BucketLookupPath
	}
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.UseSSL,
		Region:       cfg.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: new client: %w", err)
	}
	return &S3{client: client, bucket: cfg.Bucket, baseURL: cfg.BaseURL}, nil
}

// BucketExists reports whether the configured bucket exists.
func (s *S3) BucketExists(ctx context.Context) (bool, error) {
	return s.client.BucketExists(ctx, s.bucket)
}

// Put uploads r under key. If r exposes its length (bytes.Reader,
// bytes.Buffer, strings.Reader) the upload is a single PUT.
func (s *S3) Put(ctx context.Context, key, contentType string, r io.Reader) error {
	size := int64(-1)
	if l, ok := r.(interface{ Len() int }); ok {
		size = int64(l.Len())
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
		// Keys are unique per photo and never overwritten.
		CacheControl: "public, max-age=31536000, immutable",
	})
	if err != nil {
		return fmt.Errorf("storage: put %s: %w", key, err)
	}
	return nil
}

// Delete removes key. S3 answers a missing key with success, so deleting
// twice is fine.
func (s *S3) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("storage: delete %s: %w", key, err)
	}
	return nil
}

// URL returns the public URL of key.
func (s *S3) URL(key string) string {
	return PublicURL(s.baseURL, key)
}
