package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"math/rand/v2"
	"time"
)

// synthPhoto draws a deterministic 1600×1200 "landscape" (sky, sun, hills,
// ground) from seed and returns it as a JPEG carrying EXIF GPS (lat, lng)
// and DateTimeOriginal (taken, written in Bangkok time with its offset).
func synthPhoto(seed uint64, lat, lng float64, taken time.Time) ([]byte, error) {
	const w, h = 1600, 1200
	r := rand.New(rand.NewPCG(seed, 0x5eed))
	sky := hsv(190+r.Float64()*40, 0.35+r.Float64()*0.3, 0.85+r.Float64()*0.15)
	horizon := hsv(30+r.Float64()*30, 0.2+r.Float64()*0.2, 0.95)
	ground := hsv(80+r.Float64()*80, 0.35+r.Float64()*0.3, 0.35+r.Float64()*0.3)
	hill := hsv(100+r.Float64()*60, 0.4+r.Float64()*0.3, 0.25+r.Float64()*0.2)
	sunX, sunY, sunR := 200+r.Float64()*1200, 150+r.Float64()*300, 50+r.Float64()*60
	horizonY := 650 + r.Float64()*200
	amp, freq, phase := 40+r.Float64()*80, 0.002+r.Float64()*0.006, r.Float64()*math.Pi*2

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			fx, fy := float64(x), float64(y)
			hillY := horizonY - amp*(1+math.Sin(fx*freq+phase))
			var c color.RGBA
			switch {
			case fy > horizonY:
				c = mix(ground, hill, (fy-horizonY)/(h-horizonY)*0.5)
			case fy > hillY:
				c = hill
			default:
				c = mix(sky, horizon, fy/horizonY)
				if d := math.Hypot(fx-sunX, fy-sunY); d < sunR {
					c = color.RGBA{255, 244, 214, 255}
				}
			}
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return withExif(buf.Bytes(), lat, lng, taken)
}

func hsv(hue, s, v float64) color.RGBA {
	hue = math.Mod(hue, 360) / 60
	c := v * s
	x := c * (1 - math.Abs(math.Mod(hue, 2)-1))
	var r, g, b float64
	switch int(hue) {
	case 0:
		r, g = c, x
	case 1:
		r, g = x, c
	case 2:
		g, b = c, x
	case 3:
		g, b = x, c
	case 4:
		r, b = x, c
	default:
		r, b = c, x
	}
	m := v - c
	return color.RGBA{uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255), 255}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	l := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

// TIFF types and tags used in the EXIF block.
const (
	tByte     = 1
	tASCII    = 2
	tLong     = 4
	tRational = 5
)

type ifdEntry struct {
	tag, typ uint16
	count    uint32
	data     []byte // little-endian value; inlined when ≤ 4 bytes
}

// writeIFD appends an IFD at the current end of b (offsets relative to the
// TIFF header) and returns b. next is the next-IFD offset (0 = none).
func writeIFD(b []byte, entries []ifdEntry) []byte {
	start := len(b)
	dataOff := start + 2 + 12*len(entries) + 4
	var data []byte
	b = binary.LittleEndian.AppendUint16(b, uint16(len(entries)))
	for _, e := range entries {
		b = binary.LittleEndian.AppendUint16(b, e.tag)
		b = binary.LittleEndian.AppendUint16(b, e.typ)
		b = binary.LittleEndian.AppendUint32(b, e.count)
		if len(e.data) <= 4 {
			v := make([]byte, 4)
			copy(v, e.data)
			b = append(b, v...)
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(dataOff+len(data)))
		data = append(data, e.data...)
		if len(data)%2 == 1 {
			data = append(data, 0)
		}
	}
	b = binary.LittleEndian.AppendUint32(b, 0)
	return append(b, data...)
}

func ascii(s string) ifdEntry {
	return ifdEntry{typ: tASCII, count: uint32(len(s) + 1), data: append([]byte(s), 0)}
}

func long(v uint32) []byte { return binary.LittleEndian.AppendUint32(nil, v) }

// dms encodes a non-negative coordinate as degrees, minutes, seconds rationals.
func dms(v float64) ifdEntry {
	deg := math.Floor(v)
	minF := (v - deg) * 60
	mins := math.Floor(minF)
	sec := (minF - mins) * 60
	var b []byte
	for _, r := range [][2]uint32{{uint32(deg), 1}, {uint32(mins), 1}, {uint32(math.Round(sec * 10000)), 10000}} {
		b = binary.LittleEndian.AppendUint32(b, r[0])
		b = binary.LittleEndian.AppendUint32(b, r[1])
	}
	return ifdEntry{typ: tRational, count: 3, data: b}
}

var bangkok = time.FixedZone("ICT", 7*3600)

// withExif inserts an APP1 EXIF segment after the SOI marker of a JPEG.
func withExif(j []byte, lat, lng float64, taken time.Time) ([]byte, error) {
	if len(j) < 2 || j[0] != 0xFF || j[1] != 0xD8 {
		return nil, fmt.Errorf("seed: not a JPEG")
	}
	latRef, lngRef := "N", "E"
	if lat < 0 {
		latRef, lat = "S", -lat
	}
	if lng < 0 {
		lngRef, lng = "W", -lng
	}
	t := taken.In(bangkok)

	// Layout: header, IFD0 (2 entries), Exif IFD, GPS IFD. The sizes of the
	// first two are fixed, so the sub-IFD offsets are known up front.
	exifEntries := func() []ifdEntry {
		dt := ascii(t.Format("2006:01:02 15:04:05"))
		dt.tag = 0x9003 // DateTimeOriginal
		off := ascii("+07:00")
		off.tag = 0x9011 // OffsetTimeOriginal
		return []ifdEntry{dt, off}
	}
	ifdSize := func(es []ifdEntry) int {
		n := 2 + 12*len(es) + 4
		for _, e := range es {
			if len(e.data) > 4 {
				n += len(e.data) + len(e.data)%2
			}
		}
		return n
	}
	exifOff := 8 + ifdSize(make([]ifdEntry, 2))
	gpsOff := exifOff + ifdSize(exifEntries())

	b := []byte{'I', 'I', 42, 0, 8, 0, 0, 0}
	b = writeIFD(b, []ifdEntry{
		{tag: 0x8769, typ: tLong, count: 1, data: long(uint32(exifOff))}, // ExifIFDPointer
		{tag: 0x8825, typ: tLong, count: 1, data: long(uint32(gpsOff))},  // GPSInfo
	})
	b = writeIFD(b, exifEntries())
	latE, lngE := dms(lat), dms(lng)
	latE.tag, lngE.tag = 0x0002, 0x0004
	latRefE, lngRefE := ascii(latRef), ascii(lngRef)
	latRefE.tag, lngRefE.tag = 0x0001, 0x0003
	b = writeIFD(b, []ifdEntry{
		{tag: 0x0000, typ: tByte, count: 4, data: []byte{2, 3, 0, 0}}, // GPSVersionID
		latRefE, latE, lngRefE, lngE,
	})
	if len(b) != gpsOff+ifdSize([]ifdEntry{{data: []byte{2, 3, 0, 0}}, latRefE, latE, lngRefE, lngE}) {
		return nil, fmt.Errorf("seed: exif layout mismatch")
	}

	seg := append([]byte("Exif\x00\x00"), b...)
	out := make([]byte, 0, len(j)+len(seg)+4)
	out = append(out, 0xFF, 0xD8, 0xFF, 0xE1)
	out = binary.BigEndian.AppendUint16(out, uint16(len(seg)+2))
	out = append(out, seg...)
	return append(out, j[2:]...), nil
}
