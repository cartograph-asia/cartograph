// Package geo does the planar geometry of regions and water: a minimal
// shapefile reader for OSM water polygons, region boundaries, point-in-region
// lookup and land-cell counting. Coordinates are WGS 84 as orb.Point{lng, lat}
// unless stated otherwise. It is pure Go apart from internal/h3.
package geo

import (
	"archive/zip"
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"strings"

	"github.com/paulmach/orb"
)

// This file is a minimal ESRI shapefile (.shp) reader and writer for polygon
// layers such as OSM's water-polygons-split-3857. Only the geometry file is
// read: the water layer's .dbf holds nothing but an id. The reader streams,
// so the 1.3 GB Asia-wide file inside its zip is never held in memory.

const (
	shpFileCode    = 9994
	shpVersion     = 1000
	shpTypePolygon = 5
	shpHeaderLen   = 100
)

// ShpRecord is one polygon record in the file's own coordinates (EPSG:3857
// for the water layer): Box is minX, minY, maxX, maxY; Rings are closed.
type ShpRecord struct {
	Box   [4]float64
	Rings [][]orb.Point
}

// OpenShp opens a .shp file, or the first *.shp entry inside a .zip.
func OpenShp(name string) (io.ReadCloser, error) {
	if !strings.EqualFold(path.Ext(name), ".zip") {
		return os.Open(name)
	}
	zr, err := zip.OpenReader(name)
	if err != nil {
		return nil, fmt.Errorf("geo: open %s: %w", name, err)
	}
	for _, f := range zr.File {
		if strings.EqualFold(path.Ext(f.Name), ".shp") {
			rc, err := f.Open()
			if err != nil {
				zr.Close()
				return nil, fmt.Errorf("geo: open %s in %s: %w", f.Name, name, err)
			}
			return multiCloser{rc, rc, zr}, nil
		}
	}
	zr.Close()
	return nil, fmt.Errorf("geo: no .shp file in %s", name)
}

type multiCloser struct {
	io.Reader
	a, b io.Closer
}

func (m multiCloser) Close() error { return errors.Join(m.a.Close(), m.b.Close()) }

// ReadShp calls fn for every polygon record whose box intersects box
// (minX, minY, maxX, maxY in file coordinates). Records of other shape types
// and null shapes are skipped.
func ReadShp(r io.Reader, box [4]float64, fn func(ShpRecord) error) error {
	br := bufio.NewReaderSize(r, 1<<20)
	var hdr [shpHeaderLen]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return fmt.Errorf("geo: read shp header: %w", err)
	}
	if binary.BigEndian.Uint32(hdr[0:]) != shpFileCode || binary.LittleEndian.Uint32(hdr[28:]) != shpVersion {
		return errors.New("geo: not a shapefile")
	}
	var rh [8]byte
	var buf []byte
	for {
		if _, err := io.ReadFull(br, rh[:]); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("geo: read shp record header: %w", err)
		}
		n := int(binary.BigEndian.Uint32(rh[4:])) * 2 // content length in 16-bit words
		if n < 4 {
			return errors.New("geo: corrupt shp record")
		}
		if n < 44 { // null shape or too short for a polygon
			if _, err := br.Discard(n); err != nil {
				return fmt.Errorf("geo: read shp record: %w", err)
			}
			continue
		}
		head, err := br.Peek(36)
		if err != nil {
			return fmt.Errorf("geo: read shp record: %w", err)
		}
		typ := binary.LittleEndian.Uint32(head[0:])
		var rb [4]float64
		for i := range rb {
			rb[i] = math.Float64frombits(binary.LittleEndian.Uint64(head[4+8*i:]))
		}
		if typ != shpTypePolygon || rb[2] < box[0] || rb[0] > box[2] || rb[3] < box[1] || rb[1] > box[3] {
			if _, err := br.Discard(n); err != nil {
				return fmt.Errorf("geo: read shp record: %w", err)
			}
			continue
		}
		if cap(buf) < n {
			buf = make([]byte, n)
		}
		buf = buf[:n]
		if _, err := io.ReadFull(br, buf); err != nil {
			return fmt.Errorf("geo: read shp record: %w", err)
		}
		rec, err := decodePolygon(buf, rb)
		if err != nil {
			return err
		}
		if err := fn(rec); err != nil {
			return err
		}
	}
}

func decodePolygon(b []byte, box [4]float64) (ShpRecord, error) {
	numParts := int(binary.LittleEndian.Uint32(b[36:]))
	numPoints := int(binary.LittleEndian.Uint32(b[40:]))
	if numParts < 0 || numPoints < 0 || 44+4*numParts+16*numPoints > len(b) {
		return ShpRecord{}, errors.New("geo: corrupt shp polygon")
	}
	parts := make([]int, numParts+1)
	for i := range numParts {
		parts[i] = int(binary.LittleEndian.Uint32(b[44+4*i:]))
	}
	parts[numParts] = numPoints
	pts := b[44+4*numParts:]
	rec := ShpRecord{Box: box, Rings: make([][]orb.Point, 0, numParts)}
	for i := range numParts {
		lo, hi := parts[i], parts[i+1]
		if lo < 0 || hi > numPoints || lo > hi {
			return ShpRecord{}, errors.New("geo: corrupt shp polygon parts")
		}
		ring := make([]orb.Point, hi-lo)
		for j := range ring {
			o := 16 * (lo + j)
			ring[j] = orb.Point{
				math.Float64frombits(binary.LittleEndian.Uint64(pts[o:])),
				math.Float64frombits(binary.LittleEndian.Uint64(pts[o+8:])),
			}
		}
		rec.Rings = append(rec.Rings, ring)
	}
	return rec, nil
}

// WriteShp writes recs as a polygon shapefile (.shp only; no .shx or .dbf,
// which ReadShp does not need).
func WriteShp(w io.Writer, recs []ShpRecord) error {
	box := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	total := shpHeaderLen
	for _, r := range recs {
		box[0], box[1] = min(box[0], r.Box[0]), min(box[1], r.Box[1])
		box[2], box[3] = max(box[2], r.Box[2]), max(box[3], r.Box[3])
		total += 8 + recordLen(r)
	}
	if len(recs) == 0 {
		box = [4]float64{}
	}
	bw := bufio.NewWriter(w)
	hdr := make([]byte, shpHeaderLen)
	binary.BigEndian.PutUint32(hdr[0:], shpFileCode)
	binary.BigEndian.PutUint32(hdr[24:], uint32(total/2))
	binary.LittleEndian.PutUint32(hdr[28:], shpVersion)
	binary.LittleEndian.PutUint32(hdr[32:], shpTypePolygon)
	for i, v := range box {
		binary.LittleEndian.PutUint64(hdr[36+8*i:], math.Float64bits(v))
	}
	bw.Write(hdr)
	for i, r := range recs {
		n := recordLen(r)
		b := make([]byte, 8+n)
		binary.BigEndian.PutUint32(b[0:], uint32(i+1))
		binary.BigEndian.PutUint32(b[4:], uint32(n/2))
		c := b[8:]
		binary.LittleEndian.PutUint32(c[0:], shpTypePolygon)
		for j, v := range r.Box {
			binary.LittleEndian.PutUint64(c[4+8*j:], math.Float64bits(v))
		}
		points := 0
		for _, ring := range r.Rings {
			points += len(ring)
		}
		binary.LittleEndian.PutUint32(c[36:], uint32(len(r.Rings)))
		binary.LittleEndian.PutUint32(c[40:], uint32(points))
		o, start := 44+4*len(r.Rings), 0
		for j, ring := range r.Rings {
			binary.LittleEndian.PutUint32(c[44+4*j:], uint32(start))
			for _, p := range ring {
				binary.LittleEndian.PutUint64(c[o:], math.Float64bits(p[0]))
				binary.LittleEndian.PutUint64(c[o+8:], math.Float64bits(p[1]))
				o += 16
			}
			start += len(ring)
		}
		bw.Write(b)
	}
	return bw.Flush()
}

func recordLen(r ShpRecord) int {
	n := 44 + 4*len(r.Rings)
	for _, ring := range r.Rings {
		n += 16 * len(ring)
	}
	return n
}

// Web Mercator (EPSG:3857) conversion.
const earthRadius = 6378137.0

// MercatorToLngLat converts EPSG:3857 metres to WGS 84 degrees.
func MercatorToLngLat(p orb.Point) orb.Point {
	return orb.Point{
		p[0] / earthRadius * 180 / math.Pi,
		math.Atan(math.Sinh(p[1]/earthRadius)) * 180 / math.Pi,
	}
}

// LngLatToMercator converts WGS 84 degrees to EPSG:3857 metres.
func LngLatToMercator(p orb.Point) orb.Point {
	lat := p[1] * math.Pi / 180
	return orb.Point{
		p[0] * math.Pi / 180 * earthRadius,
		math.Log(math.Tan(math.Pi/4+lat/2)) * earthRadius,
	}
}

// MercatorBox returns the EPSG:3857 box of a WGS 84 bound.
func MercatorBox(b orb.Bound) [4]float64 {
	lo, hi := LngLatToMercator(b.Min), LngLatToMercator(b.Max)
	return [4]float64{lo[0], lo[1], hi[0], hi[1]}
}
