package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
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
