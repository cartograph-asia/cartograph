package photos

import (
	"math"
	"math/big"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rwcarlsen/goexif/exif"
)

// Camera is what we keep of a photo's camera tags (docs/CAMERA_BRIEF.md). It
// holds no serial number, no owner name, no firmware or software and nothing
// from the MakerNote: none of those are read.
type Camera struct {
	Make, Model string // cleaned; may be empty
	Label       string // "Apple iPhone 17 Pro", never empty
	Lens        string
	FocalMM     float64 // 0 if unknown
	Focal35MM   int
	FNumber     float64
	ExposureS   float64 // seconds
	ISO         int
	Flash       *bool // nil if unknown; true if the flash fired
}

// maxText is the longest make, model or lens we keep, in characters.
const maxText = 100

// cameraFrom reads the camera tags of x; nil if the photo has none that
// parse. Values outside what a camera can have are left out, never an error.
func cameraFrom(x *exif.Exif) *Camera {
	c := &Camera{
		Make:  cleanText(str(x, exif.Make)),
		Model: cleanText(str(x, exif.Model)),
		Lens:  cleanText(str(x, exif.LensModel)),
	}
	if v, ok := rat(x, exif.FocalLength); ok && v > 0 && v < 5000 {
		c.FocalMM = round(v, 1)
	}
	if v, ok := integer(x, exif.FocalLengthIn35mmFilm); ok && v > 0 && v < 5000 {
		c.Focal35MM = v
	}
	if v, ok := rat(x, exif.FNumber); ok && v > 0 && v < 100 {
		c.FNumber = round(v, 1)
	}
	if v, ok := rat(x, exif.ExposureTime); ok && v > 0 && v <= 86400 {
		c.ExposureS = v
	}
	if v, ok := integer(x, exif.ISOSpeedRatings); ok && v > 0 && v < 5_000_000 {
		c.ISO = v
	}
	if v, ok := integer(x, exif.Flash); ok && v >= 0 {
		fired := v&1 == 1
		c.Flash = &fired
	}
	c.Label = Label(c.Make, c.Model)
	if c.Label == "" && c.Lens == "" && c.FocalMM == 0 && c.Focal35MM == 0 && c.FNumber == 0 && c.ExposureS == 0 && c.ISO == 0 {
		return nil
	}
	if c.Label == "" {
		c.Label = "Unknown camera"
	}
	return c
}

func str(x *exif.Exif, f exif.FieldName) string {
	t, err := x.Get(f)
	if err != nil {
		return ""
	}
	s, err := t.StringVal()
	if err != nil {
		return ""
	}
	return s
}

func rat(x *exif.Exif, f exif.FieldName) (float64, bool) {
	t, err := x.Get(f)
	if err != nil {
		return 0, false
	}
	r, err := t.Rat(0)
	if err != nil || r == nil || r.Denom().Sign() == 0 {
		return 0, false
	}
	v, _ := new(big.Rat).Set(r).Float64()
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

func integer(x *exif.Exif, f exif.FieldName) (int, bool) {
	t, err := x.Get(f)
	if err != nil {
		return 0, false
	}
	v, err := t.Int(0)
	return v, err == nil
}

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

var spaces = regexp.MustCompile(`\s+`)

// cleanText makes a string from a file fit to store and print: control and
// invisible characters removed, spaces collapsed, cut at maxText characters.
func cleanText(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == utf8.RuneError || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, s)
	s = strings.TrimSpace(spaces.ReplaceAllString(s, " "))
	if r := []rune(s); len(r) > maxText {
		s = strings.TrimSpace(string(r[:maxText]))
	}
	return s
}

var corporate = regexp.MustCompile(`(?i)[\s,]+(corporation|corp\.?|company|co\.?,? ?ltd\.?|ltd\.?|imaging corp\.?|inc\.?|electronics)$`)

// Label is the one name a statistics query groups by: the make without its
// corporate suffix, then the model, the make left out when the model starts
// with it ("Canon" + "Canon EOS R5" is "Canon EOS R5").
func Label(maker, model string) string {
	maker, model = cleanText(maker), cleanText(model)
	for {
		m := corporate.ReplaceAllString(maker, "")
		if m == maker {
			break
		}
		maker = m
	}
	switch {
	case model == "":
		return maker
	case maker == "":
		return model
	}
	first := strings.ToLower(strings.Fields(maker)[0])
	if strings.HasPrefix(strings.ToLower(model), first) {
		return model
	}
	return maker + " " + model
}
