// Package commons describes the free photos fetched from Wikimedia Commons
// for development data (docs/SEED_PHOTOS_BRIEF.md): the manifest cmd/commons
// writes and cmd/seed reads. Nothing here is used by the running site.
package commons

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// ManifestFile is the manifest's name inside the cache directory.
const ManifestFile = "manifest.json"

// Photo is one downloaded file with its credit.
type Photo struct {
	File       string     `json:"file"`     // name inside the cache directory
	Title      string     `json:"title"`    // Commons title, "File:…"
	PageURL    string     `json:"page_url"` // its Commons page
	Place      string     `json:"place"`    // the search place it was found for
	Lat        float64    `json:"lat"`
	Lng        float64    `json:"lng"`
	Author     string     `json:"author"`
	Licence    string     `json:"licence"` // e.g. "CC BY-SA 4.0"
	LicenceURL string     `json:"licence_url,omitempty"`
	Taken      *time.Time `json:"taken,omitempty"` // as Commons gives it; nil if unknown
}

// Manifest lists the photos in the order they were found: by place, nearest first.
type Manifest struct {
	Photos []Photo `json:"photos"`
}

// licence matches the short names of CC0, CC BY and CC BY-SA, not the NC or ND variants.
var licence = regexp.MustCompile(`(?i)^(cc0|cc by(-sa)?( |$))`)

// AllowedLicence reports whether short names a licence the development data may use.
func AllowedLicence(short string) bool { return licence.MatchString(short) }

var fileName = regexp.MustCompile(`^[A-Za-z0-9_-]+\.jpg$`)

// Thailand's bounding box, to catch a mistyped coordinate.
const (
	minLat, maxLat = 5.5, 20.6
	minLng, maxLng = 97.3, 105.7
)

// Validate checks one photo: a plain file name, an allowed licence and a position in Thailand.
func (p Photo) Validate() error {
	switch {
	case !fileName.MatchString(p.File):
		return fmt.Errorf("file name %q", p.File)
	case !AllowedLicence(p.Licence):
		return fmt.Errorf("%s: licence %q", p.File, p.Licence)
	case p.Author == "" || p.PageURL == "":
		return fmt.Errorf("%s: no credit", p.File)
	case p.Lat < minLat || p.Lat > maxLat || p.Lng < minLng || p.Lng > maxLng:
		return fmt.Errorf("%s: position %.5f, %.5f is outside Thailand", p.File, p.Lat, p.Lng)
	}
	return nil
}

// Write stores m as the manifest in dir.
func Write(dir string, m *Manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ManifestFile), append(b, '\n'), 0o644)
}

// Read loads and validates the manifest in dir; every listed file must exist.
func Read(dir string) (*Manifest, error) {
	b, err := os.ReadFile(filepath.Join(dir, ManifestFile))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("commons: %s: %w", ManifestFile, err)
	}
	seen := map[string]bool{}
	for _, p := range m.Photos {
		if err := p.Validate(); err != nil {
			return nil, fmt.Errorf("commons: %w", err)
		}
		if seen[p.File] {
			return nil, fmt.Errorf("commons: %s listed twice", p.File)
		}
		seen[p.File] = true
		if fi, err := os.Stat(filepath.Join(dir, p.File)); err != nil || fi.Size() == 0 {
			return nil, fmt.Errorf("commons: %s is missing or empty; run cmd/commons again", p.File)
		}
	}
	if len(m.Photos) == 0 {
		return nil, fmt.Errorf("commons: %s lists no photos", ManifestFile)
	}
	return &m, nil
}
