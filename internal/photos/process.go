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
	LargeSide     = 2000
	LargeQuality  = 85
	MediumSide    = 1200
	MediumQuality = 85
	ThumbSide     = 480
	ThumbQuality  = 80

	// The pixel limits guard against decompression bombs: a file of a few
	// hundred KB can declare an image that decodes to gigabytes, so the
	// header is checked before anything is decoded. A 48 MP phone photo is
	// about 49 million pixels; photos are rarely PNG. Each limit keeps the
	// decoded image at or below about 240 MB.
	MaxJPEGPixels = 60_000_000
	MaxPNGPixels  = 30_000_000
	// PNGs with 16 bits per channel decode to twice the memory per pixel.
	MaxPNG16Pixels = MaxPNGPixels / 2
)

// ErrInvalidImage is returned when a file claims to be JPEG/PNG but cannot be
// decoded, or is too large in pixels.
var ErrInvalidImage = errors.New("the file could not be read as an image")

// Variants are the encoded images that get stored for one photo.
type Variants struct {
	Large       []byte
	Medium      []byte // photo pages and share images
	Thumb       []byte
	LargeWidth  int // stored in the DB; medium's size is derived from it
	LargeHeight int
}

// Process decodes b (JPEG or PNG), applies the EXIF orientation and returns
// the large, medium and thumb JPEG variants. Re-encoding drops all metadata.
func Process(b []byte) (*Variants, error) {
	format, err := Sniff(b)
	if err != nil {
		return nil, err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, ErrInvalidImage
	}
	if limit := maxPixels(format, cfg.ColorModel); cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > limit {
		return nil, fmt.Errorf("%w: image dimensions %dx%d exceed the limit of %d megapixels for this kind of file",
			ErrInvalidImage, cfg.Width, cfg.Height, limit/1_000_000)
	}

	img, err := imaging.Decode(bytes.NewReader(b), imaging.AutoOrientation(true))
	if err != nil {
		return nil, ErrInvalidImage
	}

	// Resize large from the original, medium from large and thumb from
	// medium: decoding once and never holding more than one full-size image
	// keeps memory in check.
	large := flatten(imaging.Fit(img, LargeSide, LargeSide, imaging.Lanczos))
	img = nil
	medium := imaging.Fit(large, MediumSide, MediumSide, imaging.Lanczos)
	thumb := imaging.Fit(medium, ThumbSide, ThumbSide, imaging.Lanczos)

	v := &Variants{LargeWidth: large.Bounds().Dx(), LargeHeight: large.Bounds().Dy()}
	if v.Large, err = encode(large, LargeQuality); err != nil {
		return nil, err
	}
	if v.Medium, err = encode(medium, MediumQuality); err != nil {
		return nil, err
	}
	if v.Thumb, err = encode(thumb, ThumbQuality); err != nil {
		return nil, err
	}
	return v, nil
}

// maxPixels is the largest image Process decodes for a format and colour
// model (as image.DecodeConfig reports them).
func maxPixels(f Format, m color.Model) int64 {
	if f == JPEG {
		return MaxJPEGPixels
	}
	switch m {
	case color.Gray16Model, color.RGBA64Model, color.NRGBA64Model:
		return MaxPNG16Pixels
	}
	return MaxPNGPixels
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
