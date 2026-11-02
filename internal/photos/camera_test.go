package photos

import (
	"os"
	"strings"
	"testing"
)

// Adrian's own iPhone photo carries the camera tags.
func TestReadMetaCamera(t *testing.T) {
	b, err := os.ReadFile("../../testdata/IMG_1911.jpeg")
	if err != nil {
		t.Fatal(err)
	}
	c := ReadMeta(b).Camera
	if c == nil {
		t.Fatal("no camera")
	}
	if c.Make != "Apple" || c.Model != "iPhone 17 Pro" || c.Label != "Apple iPhone 17 Pro" ||
		!strings.HasPrefix(c.Lens, "iPhone 17 Pro back triple camera") ||
		c.FocalMM != 6.8 || c.Focal35MM != 48 || c.FNumber != 1.8 || c.ISO != 64 {
		t.Errorf("camera %+v", c)
	}
	if c.ExposureS <= 0 || c.ExposureS > 0.0001 {
		t.Errorf("exposure %v", c.ExposureS)
	}
	if c.Flash == nil || *c.Flash {
		t.Errorf("flash %v", c.Flash)
	}
}

func TestReadMetaWithoutCamera(t *testing.T) {
	for _, f := range []string{"gps.jpg", "nogps.jpg", "rotated.jpg"} {
		if c := ReadMeta(readFixture(t, f)).Camera; c != nil {
			t.Errorf("%s: %+v", f, c)
		}
	}
	if c := ReadMeta([]byte("not a jpeg")).Camera; c != nil {
		t.Errorf("garbage: %+v", c)
	}
}

func TestLabel(t *testing.T) {
	for _, c := range []struct{ make, model, want string }{
		{"Apple", "iPhone 17 Pro", "Apple iPhone 17 Pro"},
		{"Canon", "Canon EOS R5", "Canon EOS R5"},
		{"NIKON CORPORATION", "NIKON D850", "NIKON D850"},
		{"OLYMPUS IMAGING CORP.", "E-M1MarkII", "OLYMPUS E-M1MarkII"},
		{"SONY", "ILCE-7M4", "SONY ILCE-7M4"},
		{"FUJIFILM", "X100V", "FUJIFILM X100V"},
		{"", "Pixel 9", "Pixel 9"},
		{"Samsung", "", "Samsung"},
		{"", "", ""},
		{"  Canon \x00 ", "\tEOS   R5\n", "Canon EOS R5"},
	} {
		if got := Label(c.make, c.model); got != c.want {
			t.Errorf("Label(%q, %q) = %q, want %q", c.make, c.model, got, c.want)
		}
	}
}

func TestCleanText(t *testing.T) {
	if got := cleanText("a\x00b‮ c​\td\n\ne"); got != "a b c d e" {
		t.Errorf("control and invisible characters: %q", got)
	}
	long := cleanText(strings.Repeat("ü", 300))
	if n := len([]rune(long)); n != maxText {
		t.Errorf("cut at %d characters, got %d", maxText, n)
	}
	if got := cleanText("<script>alert(1)</script>"); got != "<script>alert(1)</script>" {
		t.Errorf("text is kept as text: %q", got)
	}
}
