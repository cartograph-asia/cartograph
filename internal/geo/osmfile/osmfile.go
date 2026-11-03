// Package osmfile reads OSM extracts (.pbf, .osm, .osm.gz) for the importers
// and the fixture generator. It is separate from internal/geo because the
// PBF decoder links the system zlib through cgo (github.com/DataDog/czlib);
// the api binary must not depend on it.
package osmfile

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"github.com/paulmach/osm/osmxml"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/geo"
)

// scanOpts selects what a pass over an OSM file reads.
type scanOpts struct {
	nodes, ways, relations bool
	filterNode             func(*osm.Node) bool
	filterWay              func(*osm.Way) bool
	filterRelation         func(*osm.Relation) bool
}

// scan reads an OSM file (.pbf, .osm or .osm.gz) and calls fn for every
// object that passes opts. PBF passes skip unwanted blocks cheaply.
func scan(name string, opts scanOpts, fn func(osm.Object)) error {
	f, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("geo: open %s: %w", name, err)
	}
	defer f.Close()
	ctx := context.Background()
	var s osm.Scanner
	if strings.HasSuffix(name, ".pbf") {
		ps := osmpbf.New(ctx, f, runtime.GOMAXPROCS(0))
		ps.SkipNodes, ps.SkipWays, ps.SkipRelations = !opts.nodes, !opts.ways, !opts.relations
		ps.FilterNode, ps.FilterWay, ps.FilterRelation = opts.filterNode, opts.filterWay, opts.filterRelation
		s = ps
	} else {
		var r io.Reader = f
		if strings.HasSuffix(name, ".gz") {
			gz, err := gzip.NewReader(f)
			if err != nil {
				return fmt.Errorf("geo: open %s: %w", name, err)
			}
			defer gz.Close()
			r = gz
		}
		s = osmxml.New(ctx, r)
	}
	defer s.Close()
	for s.Scan() {
		switch o := s.Object().(type) {
		case *osm.Node:
			if opts.nodes && (opts.filterNode == nil || opts.filterNode(o)) {
				fn(o)
			}
		case *osm.Way:
			if opts.ways && (opts.filterWay == nil || opts.filterWay(o)) {
				fn(o)
			}
		case *osm.Relation:
			if opts.relations && (opts.filterRelation == nil || opts.filterRelation(o)) {
				fn(o)
			}
		}
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("geo: read %s: %w", name, err)
	}
	return nil
}

// ReadBoundaryData reads the relations accepted by keep together with all
// their member ways and those ways' nodes, in three passes. The result is
// raw OSM data for BuildBoundaries, Assemble or a fixture, made from
// ReadBoundaries: the relations as read, ways and nodes with their ids and
// positions only (no tags, no metadata), each sorted by id. Callers that
// need no raw objects should use ReadBoundaries itself, which holds a
// fraction of the memory.
func ReadBoundaryData(name string, keep func(*osm.Relation) bool) (*osm.OSM, error) {
	d, err := ReadBoundaries(name, nil, keep)
	if err != nil {
		return nil, err
	}
	return d.OSM(), nil
}

// Boundary is one administrative boundary relation with its geometry.
type Boundary struct {
	OSMID    int64
	Tags     osm.Tags
	Geometry orb.MultiPolygon
}

// BuildBoundaries assembles the multipolygon of every relation in o, in
// o's order, as osmgeojson built it (polygon). Relations whose rings cannot
// be closed are returned in skipped.
func BuildBoundaries(o *osm.OSM) (out []Boundary, skipped []int64, err error) {
	d := fromOSM(o)
	for _, r := range o.Relations {
		g := d.polygon(r)
		if len(g) == 0 {
			skipped = append(skipped, int64(r.ID))
			continue
		}
		out = append(out, Boundary{OSMID: int64(r.ID), Tags: r.Tags, Geometry: g})
	}
	return out, skipped, nil
}

// Place is an OSM place node of one of a country's place kinds.
type Place struct {
	OSMType   string // "n", "w" or "r"
	OSMID     int64
	Kind      string // a key of PlaceKinds
	NameEn    string
	NameLocal string
	Point     orb.Point
}

// PlaceKinds are the place=* values imported as plot names unless a
// country's settings name others (PlaceKindsFor). Only nodes are read;
// places mapped as areas alone are not imported.
var PlaceKinds = map[string]bool{
	"city": true, "town": true, "village": true,
	"suburb": true, "quarter": true, "neighbourhood": true,
}

// ReadPlaces reads the place nodes of imp's kinds (PlaceKindsFor) inside
// bound (whole file if geo.NoBound(bound)) that have an English name, with
// their names by imp's rules (PlaceNames).
func ReadPlaces(name string, bound orb.Bound, imp countries.Import) ([]Place, error) {
	var out []Place
	kinds := PlaceKindsFor(imp)
	keep := func(n *osm.Node) bool {
		if !kinds[n.Tags.Find("place")] {
			return false
		}
		return geo.NoBound(bound) || bound.Contains(n.Point())
	}
	err := scan(name, scanOpts{nodes: true, filterNode: keep}, func(obj osm.Object) {
		n := obj.(*osm.Node)
		en, local := PlaceNames(n.Tags, imp)
		if en == "" {
			return
		}
		out = append(out, Place{OSMType: "n", OSMID: int64(n.ID), Kind: n.Tags.Find("place"), NameEn: en, NameLocal: local, Point: n.Point()})
	})
	return out, err
}

// ReadPlacesRaw returns the raw place nodes of kinds inside bound.
func ReadPlacesRaw(name string, bound orb.Bound, kinds map[string]bool) ([]*osm.Node, error) {
	var out []*osm.Node
	keep := func(n *osm.Node) bool {
		return kinds[n.Tags.Find("place")] && (geo.NoBound(bound) || bound.Contains(n.Point()))
	}
	err := scan(name, scanOpts{nodes: true, filterNode: keep}, func(obj osm.Object) {
		out = append(out, obj.(*osm.Node))
	})
	return out, err
}

// Relations calls fn for every relation of the file that keep accepts. keep
// runs on several goroutines and must not write shared state; fn is called
// one object at a time.
func Relations(name string, keep func(*osm.Relation) bool, fn func(*osm.Relation)) error {
	return scan(name, scanOpts{relations: true, filterRelation: keep}, func(o osm.Object) { fn(o.(*osm.Relation)) })
}

// Nodes calls fn for every node of the file that keep accepts, with the same
// rules as Relations.
func Nodes(name string, keep func(*osm.Node) bool, fn func(*osm.Node)) error {
	return scan(name, scanOpts{nodes: true, filterNode: keep}, func(o osm.Object) { fn(o.(*osm.Node)) })
}
