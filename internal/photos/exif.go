package photos

import (
	"bytes"
	"io"
	"math"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

// Meta is the metadata we keep from a photo's EXIF block.
type Meta struct {
	HasGPS  bool
	Lat     float64
	Lng     float64
	TakenAt *time.Time // nil if unknown
	// TakenLocal is DateTimeOriginal as the camera's wall clock (location
	// UTC, but not converted); nil if unknown.
	TakenLocal *time.Time
	// TakenOffset is OffsetTimeOriginal in minutes east of UTC; nil if the
	// photo has none.
	TakenOffset *int
}

const offsetTimeOriginal exif.FieldName = "OffsetTimeOriginal"

func init() {
	// goexif does not know the EXIF 2.31 offset tags; load OffsetTimeOriginal
	// (0x9011) from the EXIF sub-IFD so that DateTimeOriginal can be placed in
	// the right time zone.
	exif.RegisterParsers(offsetParser{})
}

type offsetParser struct{}

func (offsetParser) Parse(x *exif.Exif) error {
	tag, err := x.Get(exif.ExifIFDPointer)
	if err != nil {
		return nil
	}
	off, err := tag.Int64(0)
	if err != nil {
		return nil
	}
	r := bytes.NewReader(x.Raw)
	if _, err := r.Seek(off, 0); err != nil {
		return nil
	}
	dir, _, err := tiff.DecodeDir(r, x.Tiff.Order)
	if err != nil {
		return nil
	}
	x.LoadTags(dir, map[uint16]exif.FieldName{0x9011: offsetTimeOriginal}, false)
	return nil
}

// ReadMeta extracts GPS position and capture time from the EXIF block of b.
// It is best effort: missing or malformed EXIF yields an empty Meta.
func ReadMeta(b []byte) Meta { return ReadMetaFrom(bytes.NewReader(b)) }

// ReadMetaFrom is ReadMeta on a stream, such as an upload spooled to disk;
// it reads no further than the EXIF block.
func ReadMetaFrom(r io.Reader) Meta {
	var m Meta
	x, err := exif.Decode(r)
	if x == nil || (err != nil && exif.IsCriticalError(err)) {
		return m
	}
	if lat, lng, err := x.LatLong(); err == nil && validLatLng(lat, lng) && !(lat == 0 && lng == 0) {
		m.HasGPS, m.Lat, m.Lng = true, lat, lng
	}
	m.TakenAt, m.TakenLocal, m.TakenOffset = takenAt(x)
	return m
}

// takenAt parses DateTimeOriginal. EXIF stores local wall-clock time; if
// OffsetTimeOriginal is present it is applied, otherwise the time is taken
// as UTC (the true zone is unknown). It also returns the wall clock itself
// and the offset in minutes (nil without OffsetTimeOriginal).
func takenAt(x *exif.Exif) (utc, local *time.Time, offsetMin *int) {
	tag, err := x.Get(exif.DateTimeOriginal)
	if err != nil {
		return nil, nil, nil
	}
	s, err := tag.StringVal()
	if err != nil {
		return nil, nil, nil
	}
	s = strings.TrimRight(strings.TrimSpace(s), "\x00")
	wall, err := time.ParseInLocation("2006:01:02 15:04:05", s, time.UTC)
	if err != nil || wall.Year() < 1900 {
		return nil, nil, nil
	}
	t := wall
	if ot, err := x.Get(offsetTimeOriginal); err == nil {
		if offs, err := ot.StringVal(); err == nil {
			if z, err := time.Parse("-07:00", strings.TrimRight(strings.TrimSpace(offs), "\x00")); err == nil {
				_, secs := z.Zone()
				t = wall.Add(-time.Duration(secs) * time.Second)
				mins := secs / 60
				offsetMin = &mins
			}
		}
	}
	return &t, &wall, offsetMin
}

func validLatLng(lat, lng float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lng) && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// ValidLatLng reports whether lat/lng is a valid WGS84 position.
func ValidLatLng(lat, lng float64) bool { return validLatLng(lat, lng) }
