// Package photos holds the image pipeline: format detection, EXIF metadata
// extraction and resizing into the stored variants.
package photos

import (
	"bytes"
	"errors"
)

// Format is a supported input format.
type Format string

const (
	JPEG Format = "jpeg"
	PNG  Format = "png"
)

var (
	// ErrHEIC is returned for HEIC/HEIF files, which are not supported yet.
	ErrHEIC = errors.New("HEIC is not supported yet; please upload JPEG or PNG")
	// ErrUnsupported is returned for anything that is not JPEG or PNG.
	ErrUnsupported = errors.New("unsupported file type; please upload JPEG or PNG")
)

// heifBrands are ISO-BMFF major brands used by HEIC/HEIF files.
var heifBrands = map[string]bool{
	"heic": true, "heix": true, "hevc": true, "hevx": true,
	"heim": true, "heis": true, "hevm": true, "hevs": true,
	"mif1": true, "msf1": true,
}

// Sniff detects the format of an uploaded file from its first bytes.
func Sniff(b []byte) (Format, error) {
	switch {
	case len(b) >= 3 && bytes.Equal(b[:3], []byte{0xFF, 0xD8, 0xFF}):
		return JPEG, nil
	case len(b) >= 8 && bytes.Equal(b[:8], []byte("\x89PNG\r\n\x1a\n")):
		return PNG, nil
	case len(b) >= 12 && string(b[4:8]) == "ftyp" && heifBrands[string(b[8:12])]:
		return "", ErrHEIC
	}
	return "", ErrUnsupported
}
