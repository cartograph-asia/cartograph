package photos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReadMetaWithGPS(t *testing.T) {
	m := ReadMeta(readFixture(t, "gps.jpg"))
	if !m.HasGPS {
		t.Fatal("expected GPS")
	}
	if math.Abs(m.Lat-12.851789) > 1e-5 || math.Abs(m.Lng-100.899881) > 1e-5 {
		t.Errorf("lat/lng = %f,%f", m.Lat, m.Lng)
	}
	// 2026:09:09 16:38:06 at +07:00.
	want := time.Date(2026, 9, 9, 9, 38, 6, 0, time.UTC)
	if m.TakenAt == nil || !m.TakenAt.Equal(want) {
		t.Errorf("TakenAt = %v, want %v", m.TakenAt, want)
	}
	// The wall clock is kept as is, so year and month stay local.
	wantLocal := time.Date(2026, 9, 9, 16, 38, 6, 0, time.UTC)
	if m.TakenLocal == nil || !m.TakenLocal.Equal(wantLocal) {
		t.Errorf("TakenLocal = %v, want %v", m.TakenLocal, wantLocal)
	}
	if m.TakenOffset == nil || *m.TakenOffset != 7*60 {
		t.Errorf("TakenOffset = %v, want 420", m.TakenOffset)
	}
}

func TestReadMetaWithoutGPS(t *testing.T) {
	m := ReadMeta(readFixture(t, "nogps.jpg"))
	if m.HasGPS {
		t.Fatalf("unexpected GPS %f,%f", m.Lat, m.Lng)
	}
	if m.TakenAt == nil {
		t.Error("expected TakenAt from DateTimeOriginal")
	}
}

func TestReadMetaNoExif(t *testing.T) {
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewGray(image.Rect(0, 0, 4, 4)))
	if m := ReadMeta(buf.Bytes()); m.HasGPS || m.TakenAt != nil {
		t.Errorf("unexpected meta %+v", m)
	}
	if m := ReadMeta([]byte("garbage")); m.HasGPS {
		t.Error("garbage has GPS")
	}
}

func TestSniff(t *testing.T) {
	heic := append([]byte{0, 0, 0, 0x18}, []byte("ftypheic\x00\x00\x00\x00mif1heic")...)
	cases := []struct {
		name string
		in   []byte
		want Format
		err  error
	}{
		{"jpeg", readFixture(t, "gps.jpg"), JPEG, nil},
		{"png", []byte("\x89PNG\r\n\x1a\n...."), PNG, nil},
		{"heic", heic, "", ErrHEIC},
		{"mp4", append([]byte{0, 0, 0, 0x18}, []byte("ftypisom")...), "", ErrUnsupported},
		{"garbage", []byte("hello world"), "", ErrUnsupported},
		{"empty", nil, "", ErrUnsupported},
	}
	for _, c := range cases {
		got, err := Sniff(c.in)
		if got != c.want || !errors.Is(err, c.err) {
			t.Errorf("%s: Sniff = %q, %v; want %q, %v", c.name, got, err, c.want, c.err)
		}
	}
}

func TestProcess(t *testing.T) {
	v, err := Process(readFixture(t, "gps.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	// The fixture is 640x480; Fit never upscales.
	if v.LargeWidth != 640 || v.LargeHeight != 480 {
		t.Errorf("large = %dx%d", v.LargeWidth, v.LargeHeight)
	}
	thumb, err := jpeg.DecodeConfig(bytes.NewReader(v.Thumb))
	if err != nil {
		t.Fatal(err)
	}
	if thumb.Width != ThumbSide || thumb.Height != 360 {
		t.Errorf("thumb = %dx%d", thumb.Width, thumb.Height)
	}
	if m := ReadMeta(v.Large); m.HasGPS || m.TakenAt != nil {
		t.Error("large variant still carries EXIF")
	}
}

func TestProcessAutoOrient(t *testing.T) {
	// Same pixels as gps.jpg but with Orientation=6 (rotate 90° CW).
	v, err := Process(readFixture(t, "rotated.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if v.LargeWidth != 480 || v.LargeHeight != 640 {
		t.Errorf("large = %dx%d, want 480x640", v.LargeWidth, v.LargeHeight)
	}
}

func TestProcessDownscalesAndFlattens(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3000, 1000)) // fully transparent
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	v, err := Process(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if v.LargeWidth != LargeSide || (v.LargeHeight < 666 || v.LargeHeight > 667) {
		t.Errorf("large = %dx%d", v.LargeWidth, v.LargeHeight)
	}
	medium, err := jpeg.DecodeConfig(bytes.NewReader(v.Medium))
	if err != nil {
		t.Fatal(err)
	}
	if medium.Width != MediumSide || medium.Height < 399 || medium.Height > 400 {
		t.Errorf("medium = %dx%d", medium.Width, medium.Height)
	}
	dec, err := jpeg.Decode(bytes.NewReader(v.Large))
	if err != nil {
		t.Fatal(err)
	}
	if r, g, b, _ := dec.At(10, 10).RGBA(); r < 0xf000 || g < 0xf000 || b < 0xf000 {
		t.Errorf("transparent area not flattened to white: %v", color.RGBA64Model.Convert(dec.At(10, 10)))
	}
}

func TestProcessRejects(t *testing.T) {
	if _, err := Process([]byte("not an image")); !errors.Is(err, ErrUnsupported) {
		t.Errorf("garbage: %v", err)
	}
	// Valid JPEG signature but truncated body.
	b := readFixture(t, "gps.jpg")[:200]
	if _, err := Process(b); !errors.Is(err, ErrInvalidImage) {
		t.Errorf("truncated jpeg: %v", err)
	}
}

// withSize returns a copy of a small encoded JPEG or PNG whose header claims
// w×h pixels: a decompression bomb in a few hundred bytes. The pixel data
// stays that of the small image.
func withSize(t *testing.T, b []byte, w, h int) []byte {
	t.Helper()
	b = bytes.Clone(b)
	if i := bytes.Index(b, []byte{0xFF, 0xC0}); b[0] == 0xFF && i > 0 { // JPEG SOF0
		binary.BigEndian.PutUint16(b[i+5:], uint16(h))
		binary.BigEndian.PutUint16(b[i+7:], uint16(w))
		return b
	}
	ihdr := b[8+4 : 8+4+4+13] // "IHDR" and its data, after the length
	if string(ihdr[:4]) != "IHDR" {
		t.Fatal("no IHDR")
	}
	binary.BigEndian.PutUint32(ihdr[4:], uint32(w))
	binary.BigEndian.PutUint32(ihdr[8:], uint32(h))
	binary.BigEndian.PutUint32(b[8+4+4+13:], crc32.ChecksumIEEE(ihdr))
	return b
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessRejectsBombs(t *testing.T) {
	jpg := readFixture(t, "nogps.jpg")
	png8 := encodePNG(t, image.NewNRGBA(image.Rect(0, 0, 4, 4)))
	png16 := encodePNG(t, image.NewNRGBA64(image.Rect(0, 0, 4, 4)))
	gray16 := encodePNG(t, image.NewGray16(image.Rect(0, 0, 4, 4)))
	for _, c := range []struct {
		name string
		b    []byte
	}{
		{"jpeg 65535x1000", withSize(t, jpg, 65535, 1000)},
		{"png 8-bit 6000x5001", withSize(t, png8, 6000, 5001)},
		{"png 16-bit 10000x9999", withSize(t, png16, 10000, 9999)}, // 757 KB of these decode to 908 MB
		{"png 16-bit 4000x3751", withSize(t, png16, 4000, 3751)},
		{"png gray 16-bit 4000x3751", withSize(t, gray16, 4000, 3751)},
		{"png 8-bit 1<<31 wide", withSize(t, png8, 1<<31-1, 1)},
	} {
		_, err := Process(c.b)
		if !errors.Is(err, ErrInvalidImage) || !strings.Contains(err.Error(), "megapixels") {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

func TestMaxPixels(t *testing.T) {
	for _, c := range []struct {
		f    Format
		m    color.Model
		want int64
	}{
		{JPEG, color.YCbCrModel, 60_000_000},
		{JPEG, color.CMYKModel, 60_000_000},
		{PNG, color.NRGBAModel, 30_000_000},
		{PNG, color.RGBAModel, 30_000_000},
		{PNG, color.GrayModel, 30_000_000},
		{PNG, color.NRGBA64Model, 15_000_000},
		{PNG, color.RGBA64Model, 15_000_000},
		{PNG, color.Gray16Model, 15_000_000},
	} {
		if got := maxPixels(c.f, c.m); got != c.want {
			t.Errorf("%s %T: %d, want %d", c.f, c.m, got, c.want)
		}
	}
	// A 16-bit PNG within its limit is processed as usual.
	img := image.NewNRGBA64(image.Rect(0, 0, 300, 200))
	for i := range img.Pix {
		img.Pix[i] = 0xFF
	}
	if v, err := Process(encodePNG(t, img)); err != nil || v.LargeWidth != 300 {
		t.Fatalf("16-bit png: %+v %v", v, err)
	}
}
