// Package storage is the only place that talks to the object store.
// Everything else uses the small Store interface and object keys.
package storage

import (
	"context"
	"io"
	"strings"
)

// Store puts objects and builds their public URLs.
type Store interface {
	// Put stores the content of r under key.
	Put(ctx context.Context, key, contentType string, r io.Reader) error
	// URL returns the public URL for key.
	URL(key string) string
	// Delete removes the object under key; a missing object is not an error.
	Delete(ctx context.Context, key string) error
}

// PublicURL joins base and key with exactly one slash.
func PublicURL(base, key string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(key, "/")
}
