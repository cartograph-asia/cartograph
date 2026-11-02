// Package countryscan reads an OSM extract and reports what it holds for one
// country: the administrative levels, how well they tile the country's land,
// the names in them, the places and the number of plots. It is stage 1 of
// docs/COUNTRY_IMPORT_BRIEF.md. A scan is read-only: it writes no database
// row, and the same extract always gives the same report.
package countryscan

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/paulmach/orb"
	orbgeo "github.com/paulmach/orb/geo"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
	"github.com/cartograph-asia/cartograph/internal/h3"
)

const (
	// MinLevel and MaxLevel are the admin_levels the scan looks at.
	MinLevel = 2
	MaxLevel = 10
	// DefaultGeometryCap is the most relations of one level the scan builds
	// geometry for; a level with more is counted and read for its names only.
	DefaultGeometryCap = 8000
	// PlotKm2 is the mean area of a resolution-8 cell.
	PlotKm2 = 0.737
	// ThailandPlots, ThailandSeconds and ThailandMiB are what the Thailand
	// import measured (598,404 land plots), the base of the stage 3 estimate.
	ThailandPlots   = 598404
	ThailandSeconds = 25
	ThailandMiB     = 550
)

// Options configures Scan.
type Options struct {
	Country     countries.Country
	PBF         string // the extract, or the file the scan reads directly
	Water       string // water polygons (.shp or .zip); empty counts sea as land
	Res         int    // H3 resolution of the land set, default 8
	GeometryCap int    // default DefaultGeometryCap
	Cache       string // directory for the land cell list used by Overlaps; empty writes none
	Workers     int    // default GOMAXPROCS via the caller; 0 means 8
}

// Scan reads opts.PBF (already cut, see Cut) and returns the findings.
func Scan(ctx context.Context, o Options) (*Result, error) {
	if o.Res == 0 {
		o.Res = h3.PlotRes
	}
	if o.GeometryCap == 0 {
		o.GeometryCap = DefaultGeometryCap
	}
	if o.Workers <= 0 {
		o.Workers = 8
	}
	s := &scanner{o: o, cc: strings.ToUpper(o.Country.Code), lang: langSubtag(o.Country.Lang), recs: map[int][]shapeRec{}}
	r := &Result{Country: o.Country, Extract: filepath.Base(o.PBF), Res: o.Res, Workers: o.Workers}
	s.r = r

	t := time.Now()
	if err := s.collect(); err != nil {
		return nil, err
	}
	slog.Info("read administrative relations", "country", o.Country.Code, "relations", s.total, "took", time.Since(t).Round(time.Millisecond))

	if err := s.countryShape(); err != nil {
		return nil, err
	}
	if err := s.landCells(ctx); err != nil {
		return nil, err
	}
	for lv := MinLevel; lv <= MaxLevel; lv++ {
		if err := s.level(lv); err != nil {
			return nil, err
		}
	}
	s.propose()
	s.names()
	if err := s.places(); err != nil {
		return nil, err
	}
	s.size()
	if o.Cache != "" && o.Res == h3.PlotRes {
		if err := WriteCells(CellsFile(o.Cache, o.Country.Code), s.land); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// lrel is an administrative relation reduced to the tags the scan reads.
type lrel struct {
	id    int64
	level int
	tags  tagSet
}

// shapeRec is one in-country relation of a geometry level.
type shapeRec struct {
	rel    *lrel
	geom   orb.MultiPolygon // simplified as the import would store it
	centre orb.Point
	land   int // land plots whose centre lies inside
	areaKm float64
}

type scanner struct {
	o     Options
	cc    string // upper case ISO code
	lang  string // primary language subtag of the country
	r     *Result
	total int
	byLvl map[int][]*lrel

	countryRel *lrel
	shape      orb.MultiPolygon
	inCountry  []uint64 // sorted cells whose centre is in the country's shape
	land       []uint64 // sorted subset not in water
	recs       map[int][]shapeRec
}

// keepKey reports whether a relation tag is kept.
func keepKey(k string) bool {
	switch k {
	case "boundary", "admin_level", "name", "int_name", "ISO3166-1", "ISO3166-1:alpha2", "ISO3166-2", "wikidata":
		return true
	}
	return strings.HasPrefix(k, "name:")
}

func (s *scanner) collect() error {
	s.byLvl = map[int][]*lrel{}
	keep := func(r *osm.Relation) bool {
		if r.Tags.Find("boundary") != "administrative" {
			return false
		}
		l, err := strconv.Atoi(r.Tags.Find("admin_level"))
		return err == nil && l >= MinLevel && l <= MaxLevel
	}
	err := osmfile.Relations(s.o.PBF, keep, func(r *osm.Relation) {
		l, _ := strconv.Atoi(r.Tags.Find("admin_level"))
		t := tagSet{}
		for _, tg := range r.Tags {
			if keepKey(tg.Key) {
				t[tg.Key] = tg.Value
			}
		}
		s.byLvl[l] = append(s.byLvl[l], &lrel{id: int64(r.ID), level: l, tags: t})
		s.total++
	})
	if err != nil {
		return err
	}
	for l := range s.byLvl {
		sort.Slice(s.byLvl[l], func(i, j int) bool { return s.byLvl[l][i].id < s.byLvl[l][j].id })
	}
	for _, rl := range s.byLvl[2] {
		if hasCode(rl.tags["ISO3166-1:alpha2"], s.cc) || hasCode(rl.tags["ISO3166-1"], s.cc) {
			s.countryRel = rl
			break
		}
	}
	return nil
}

// hasCode reports whether the ';'-separated list v holds code, ignoring case.
func hasCode(v, code string) bool {
	for _, x := range strings.Split(v, ";") {
		if strings.EqualFold(strings.TrimSpace(x), code) {
			return true
		}
	}
	return false
}

// isoState says what a relation's ISO3166-2 tag tells about the country.
type isoState int

const (
	isoNone  isoState = iota // no ISO3166-2 tag
	isoHere                  // a value with this country's prefix
	isoOther                 // values, none with this country's prefix
)

func (s *scanner) iso(t tagSet) isoState {
	v := t["ISO3166-2"]
	if strings.TrimSpace(v) == "" {
		return isoNone
	}
	for _, x := range strings.Split(v, ";") {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(x)), s.cc+"-") {
			return isoHere
		}
	}
	return isoOther
}

// build reads and assembles the boundaries of the relations keep accepts. A
// relation with a member way that is not in the file (an extract cuts a
// relation at its edge) is returned in skipped, not as the partial polygon
// the assembly would make of the ways that are there (osmfile.Assemble with
// no relation allowed to be partial).
func (s *scanner) build(keep func(*osm.Relation) bool) (map[int64]osmfile.Boundary, []int64, error) {
	raw, err := osmfile.ReadBoundaryData(s.o.PBF, keep)
	if err != nil {
		return nil, nil, err
	}
	a, err := osmfile.Assemble(raw, nil)
	if err != nil {
		return nil, nil, err
	}
	skipped := append(a.NotWhole, a.NotClosed...)
	out := make(map[int64]osmfile.Boundary, len(a.Built))
	for _, b := range a.Built {
		out[b.OSMID] = b
	}
	return out, skipped, nil
}

// countryShape finds the country's outline: its admin_level 2 relation when
// the extract holds it whole, else the union of the relations of the lowest
// level whose ISO3166-2 tags name the country.
func (s *scanner) countryShape() error {
	sh := &s.r.Shape
	if s.countryRel != nil {
		id := osm.RelationID(s.countryRel.id)
		bs, _, err := s.build(func(r *osm.Relation) bool { return r.ID == id })
		if err != nil {
			return err
		}
		if b, ok := bs[s.countryRel.id]; ok && len(b.Geometry) > 0 {
			s.shape = geo.Simplify(b.Geometry, geo.SimplifyTolerance)
			sh.Source = fmt.Sprintf("the country relation r%d (admin_level 2)", b.OSMID)
			sh.RelationID = b.OSMID
			sh.AreaKm2 = orbgeo.Area(b.Geometry) / 1e6
			sh.Polygons = len(s.shape)
			return nil
		}
	}
	// The level whose relations with this country's ISO code add up to the
	// largest area: the lowest level alone may be a few special regions
	// (China's level 3 is Hong Kong and Macau).
	var best orb.MultiPolygon
	bestArea, bestLevel, bestIDs, bestBuilt := 0.0, 0, 0, 0
	for l := 3; l <= 6; l++ {
		var ids []*lrel
		for _, rl := range s.byLvl[l] {
			if s.iso(rl.tags) == isoHere {
				ids = append(ids, rl)
			}
		}
		if len(ids) == 0 || len(ids) > s.o.GeometryCap {
			continue
		}
		want := map[osm.RelationID]bool{}
		for _, rl := range ids {
			want[osm.RelationID(rl.id)] = true
		}
		bs, _, err := s.build(func(r *osm.Relation) bool { return want[r.ID] })
		if err != nil {
			return err
		}
		var mp orb.MultiPolygon
		area := 0.0
		for _, rl := range ids {
			if b, ok := bs[rl.id]; ok {
				mp = append(mp, b.Geometry...)
				area += orbgeo.Area(b.Geometry) / 1e6
			}
		}
		if area > bestArea*1.0001 {
			best, bestArea, bestLevel, bestIDs, bestBuilt = mp, area, l, len(ids), len(bs)
		}
	}
	if len(best) > 0 {
		s.shape = geo.Simplify(best, geo.SimplifyTolerance)
		sh.Source = fmt.Sprintf("the union of %d admin_level %d relations with ISO 3166-2 %s-xx (the country relation is not whole in the extract)", bestBuilt, bestLevel, s.cc)
		if bestBuilt < bestIDs {
			sh.Source += fmt.Sprintf("; %d more of them are not whole either and are missing from it, so the outline and the land plots are too small", bestIDs-bestBuilt)
		}
		sh.AreaKm2, sh.Polygons = bestArea, len(s.shape)
		return nil
	}
	return fmt.Errorf("countryscan: no outline of %s in %s: no admin_level 2 relation with ISO3166-1 %s and no relation with ISO3166-2 %s-xx",
		s.o.Country.NameEn, filepath.Base(s.o.PBF), s.cc, s.cc)
}

// parallelFor runs fn(0..n-1) on workers goroutines.
func parallelFor(n, workers int, fn func(i int)) {
	var wg sync.WaitGroup
	next := make(chan int)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				fn(i)
			}
		}()
	}
	for i := 0; i < n; i++ {
		next <- i
	}
	close(next)
	wg.Wait()
}

// polyCells returns the sorted cells at res whose centre lies inside mp.
func polyCells(mp orb.MultiPolygon, res, workers int) ([]uint64, error) {
	parts := make([][]uint64, len(mp))
	errs := make([]error, len(mp))
	parallelFor(len(mp), workers, func(i int) {
		p := mp[i]
		if len(p) == 0 || len(p[0]) < 4 {
			return
		}
		holes := make([][][2]float64, 0, len(p)-1)
		for _, h := range p[1:] {
			holes = append(holes, ringPairs(h))
		}
		parts[i], errs[i] = h3.PolygonCellIDs(ringPairs(p[0]), holes, res)
	})
	var out []uint64
	for i, e := range errs {
		if e != nil {
			return nil, e
		}
		out = append(out, parts[i]...)
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

func ringPairs(r orb.Ring) [][2]float64 {
	out := make([][2]float64, len(r))
	for i, p := range r {
		out[i] = [2]float64{p[0], p[1]}
	}
	return out
}

// landCells fills inCountry and land, and reads the water polygons.
func (s *scanner) landCells(ctx context.Context) error {
	t := time.Now()
	cells, err := polyCells(s.shape, s.o.Res, s.o.Workers)
	if err != nil {
		return err
	}
	s.inCountry = cells
	s.r.CountryCells = len(cells)
	slog.Info("filled the country outline", "country", s.o.Country.Code, "cells", len(cells), "took", time.Since(t).Round(time.Millisecond))

	var w *geo.Water
	if s.o.Water != "" {
		t = time.Now()
		if w, err = geo.ReadWater(s.o.Water, s.shape.Bound()); err != nil {
			return err
		}
		slog.Info("read water polygons", "polygons", w.Len(), "took", time.Since(t).Round(time.Millisecond))
	}
	const chunk = 50000
	n := (len(cells) + chunk - 1) / chunk
	kept := make([][]uint64, n)
	parallelFor(n, s.o.Workers, func(i int) {
		hi := min((i+1)*chunk, len(cells))
		for _, c := range cells[i*chunk : hi] {
			lat, lng, err := h3.IDCentre(c)
			if err == nil && !w.Contains(orb.Point{lng, lat}) {
				kept[i] = append(kept[i], c)
			}
		}
	})
	s.land = make([]uint64, 0, len(cells))
	for _, k := range kept {
		s.land = append(s.land, k...)
	}
	s.r.LandCells = len(s.land)
	s.r.Water = s.o.Water != ""
	slog.Info("masked water", "country", s.o.Country.Code, "land", len(s.land), "took", time.Since(t).Round(time.Millisecond))
	return ctx.Err()
}

// cellIn reports whether id is in the sorted list.
func cellIn(list []uint64, id uint64) bool {
	_, ok := slices.BinarySearch(list, id)
	return ok
}

// centreOf returns a point inside mp near its centroid.
func centreOf(mp orb.MultiPolygon) orb.Point {
	c, _ := planar.CentroidArea(mp)
	if !planar.MultiPolygonContains(mp, c) {
		if len(mp) > 0 && len(mp[0]) > 0 && len(mp[0][0]) > 0 {
			return mp[0][0][0]
		}
	}
	return c
}

// WriteCells writes a sorted cell list as little-endian uint64s.
func WriteCells(name string, cells []uint64) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return fmt.Errorf("countryscan: %w", err)
	}
	buf := make([]byte, 8*len(cells))
	for i, c := range cells {
		binary.LittleEndian.PutUint64(buf[8*i:], c)
	}
	if err := os.WriteFile(name, buf, 0o644); err != nil {
		return fmt.Errorf("countryscan: %w", err)
	}
	return nil
}

// ReadCells reads a list written by WriteCells.
func ReadCells(name string) ([]uint64, error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("countryscan: %w", err)
	}
	if len(b)%8 != 0 {
		return nil, fmt.Errorf("countryscan: %s: not a cell list", name)
	}
	out := make([]uint64, len(b)/8)
	for i := range out {
		out[i] = binary.LittleEndian.Uint64(b[8*i:])
	}
	return out, nil
}

// CellsFile is where the scan of the country with the given code keeps its
// land cells (resolution 8) under cache.
func CellsFile(cache, code string) string {
	return filepath.Join(cache, strings.ToLower(code)+".land8")
}
