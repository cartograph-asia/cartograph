package photos

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // register decoders for image.DecodeConfig
	_ "image/png"

	"github.com/disintegration/imaging"
)

const (
	LargeSide    = 2000
	LargeQuality = 85
	ThumbSide    = 480
	ThumbQuality = 80

	// MaxPixels guards against decompression bombs. A 48 MP phone photo is
	// about 49 million pixels.
	MaxPixels = 100_000_000
)

// ErrInvalidImage is returned when a file claims to be JPEG/PNG but cannot be
// decoded, or is too large in pixels.
var ErrInvalidImage = errors.New("the file could not be read as an image")

// Variants are the encoded images that get stored for one photo.
type Variants struct {
	Large       []byte
	Thumb       []byte
	LargeWidth  int
	LargeHeight int
}

// Process decodes b (JPEG or PNG), applies the EXIF orientation and returns
// the large and thumb JPEG variants. Re-encoding drops all metadata.
func Process(b []byte) (*Variants, error) {
	if _, err := Sniff(b); err != nil {
		return nil, err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, ErrInvalidImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return nil, fmt.Errorf("%w: image dimensions %dx%d exceed the limit of %d megapixels",
			ErrInvalidImage, cfg.Width, cfg.Height, MaxPixels/1_000_000)
	}

	img, err := imaging.Decode(bytes.NewReader(b), imaging.AutoOrientation(true))
	if err != nil {
		return nil, ErrInvalidImage
	}

	// Resize large from the original, thumb from large: decoding once and
	// never holding more than one full-size image keeps memory in check.
	large := flatten(imaging.Fit(img, LargeSide, LargeSide, imaging.Lanczos))
	img = nil
	thumb := imaging.Fit(large, ThumbSide, ThumbSide, imaging.Lanczos)

	v := &Variants{LargeWidth: large.Bounds().Dx(), LargeHeight: large.Bounds().Dy()}
	if v.Large, err = encode(large, LargeQuality); err != nil {
		return nil, err
	}
	if v.Thumb, err = encode(thumb, ThumbQuality); err != nil {
		return nil, err
	}
	return v, nil
}

// flatten composites images with transparency onto white, since JPEG has no
// alpha channel.
func flatten(img *image.NRGBA) *image.NRGBA {
	if img.Opaque() {
		return img
	}
	bg := imaging.New(img.Bounds().Dx(), img.Bounds().Dy(), color.White)
	return imaging.Overlay(bg, img, image.Point{}, 1.0)
}

func encode(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(quality)); err != nil {
		return nil, fmt.Errorf("photos: encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}
