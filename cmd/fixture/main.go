// Command fixture cuts the test fixture out of the full inputs: every
// province boundary intersecting -bbox (complete relation, member ways and
// their nodes, without metadata) plus the city/town/village nodes inside
// -bbox, as gzipped OSM XML; and the water polygons clipped to -bbox as a
// .shp file. The defaults reproduce testdata/pattaya.osm.gz and
// testdata/pattaya-water.shp from the repository root:
//
//	go run ./cmd/fixture
package main

import (
	"compress/gzip"
	"encoding/xml"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/clip"
	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// PattayaBBox is the default fixture area: Pattaya, Naklua, Jomtien and Nong Prue.
const PattayaBBox = "100.82,12.82,100.98,13.02"

func main() {
	pbf := flag.String("pbf", "map/data/thailand-latest.osm.pbf", "full OSM extract")
	water := flag.String("water", "map/data/sources/water-polygons-split-3857.zip", "full water polygons")
	bbox := flag.String("bbox", PattayaBBox, "fixture area minLng,minLat,maxLng,maxLat")
	country := flag.String("country", "th", "country of the provinces")
	outOSM := flag.String("out-osm", "testdata/pattaya.osm.gz", "output OSM XML (gzipped)")
	outWater := flag.String("out-water", "testdata/pattaya-water.shp", "output water shapefile")
	flag.Parse()
	if err := run(*pbf, *water, *bbox, *country, *outOSM, *outWater); err != nil {
		slog.Error("fixture", "err", err)
		os.Exit(1)
	}
}

func run(pbf, water, bbox, country, outOSM, outWater string) error {
	b, err := geo.ParseBound(bbox)
	if err != nil {
		return err
	}
	if geo.NoBound(b) {
		return fmt.Errorf("-bbox is required")
	}
	raw, err := osmfile.ReadBoundaryData(pbf, func(r *osm.Relation) bool { return osmfile.IsProvince(r, country) })
	if err != nil {
		return err
	}
	bs, _, err := osmfile.BuildBoundaries(raw)
	if err != nil {
		return err
	}
	keepRel := map[osm.RelationID]bool{}
	for _, x := range bs {
		if x.Geometry.Bound().Intersects(b) {
			keepRel[osm.RelationID(x.OSMID)] = true
		}
	}
	out := &osm.OSM{Version: "0.6", Generator: "cartograph cmd/fixture"}
	ways, nodes := map[osm.WayID]bool{}, map[osm.NodeID]bool{}
	for _, r := range raw.Relations {
		if !keepRel[r.ID] {
			continue
		}
		rr := &osm.Relation{ID: r.ID, Visible: true, Version: 1, Tags: r.Tags}
		for _, m := range r.Members {
			if m.Type == osm.TypeWay {
				ways[osm.WayID(m.Ref)] = true
				rr.Members = append(rr.Members, osm.Member{Type: m.Type, Ref: m.Ref, Role: m.Role})
			}
		}
		out.Relations = append(out.Relations, rr)
	}
	for _, w := range raw.Ways {
		if !ways[w.ID] {
			continue
		}
		ww := &osm.Way{ID: w.ID, Visible: true, Version: 1}
		for _, n := range w.Nodes {
			nodes[n.ID] = true
			ww.Nodes = append(ww.Nodes, osm.WayNode{ID: n.ID})
		}
		out.Ways = append(out.Ways, ww)
	}
	for _, n := range raw.Nodes {
		if nodes[n.ID] {
			out.Nodes = append(out.Nodes, &osm.Node{ID: n.ID, Visible: true, Version: 1, Lat: n.Lat, Lon: n.Lon})
		}
	}
	places, err := osmfile.ReadPlacesRaw(pbf, b)
	if err != nil {
		return err
	}
	for _, n := range places {
		if nodes[n.ID] {
			continue // already present as a boundary node (no tags kept there)
		}
		var tags osm.Tags
		for _, t := range n.Tags {
			switch t.Key {
			case "place", "name", "name:en", "name:th", "int_name", "name:th-Latn":
				tags = append(tags, t)
			}
		}
		out.Nodes = append(out.Nodes, &osm.Node{ID: n.ID, Visible: true, Version: 1, Lat: n.Lat, Lon: n.Lon, Tags: tags})
	}
	sort.Slice(out.Nodes, func(i, j int) bool { return out.Nodes[i].ID < out.Nodes[j].ID })
	sort.Slice(out.Ways, func(i, j int) bool { return out.Ways[i].ID < out.Ways[j].ID })
	sort.Slice(out.Relations, func(i, j int) bool { return out.Relations[i].ID < out.Relations[j].ID })
	if err := writeOSM(outOSM, out); err != nil {
		return err
	}
	slog.Info("wrote osm", "file", outOSM, "relations", len(out.Relations), "ways", len(out.Ways), "nodes", len(out.Nodes), "places", len(places))

	recs, err := geo.ReadShpRecords(water, b)
	if err != nil {
		return err
	}
	mbox := geo.MercatorBox(b)
	box := orb.Bound{Min: orb.Point{mbox[0], mbox[1]}, Max: orb.Point{mbox[2], mbox[3]}}
	var clipped []geo.ShpRecord
	for _, r := range recs {
		var c geo.ShpRecord
		for _, ring := range r.Rings {
			cr := clip.Ring(box, orb.Ring(ring).Clone())
			if len(cr) < 4 {
				continue
			}
			c.Rings = append(c.Rings, cr)
		}
		if len(c.Rings) == 0 {
			continue
		}
		rb := orb.Ring(c.Rings[0]).Bound()
		for _, ring := range c.Rings[1:] {
			rb = rb.Union(orb.Ring(ring).Bound())
		}
		c.Box = [4]float64{rb.Min[0], rb.Min[1], rb.Max[0], rb.Max[1]}
		clipped = append(clipped, c)
	}
	f, err := os.Create(outWater)
	if err != nil {
		return err
	}
	if err := geo.WriteShp(f, clipped); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	slog.Info("wrote water", "file", outWater, "polygons", len(clipped))
	return nil
}

func writeOSM(name string, o *osm.OSM) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	gz, _ := gzip.NewWriterLevel(f, gzip.BestCompression)
	gz.Header.Name = "" // keep the output byte-identical between runs
	if _, err := gz.Write([]byte(xml.Header)); err != nil {
		return err
	}
	enc := xml.NewEncoder(gz)
	enc.Indent("", " ")
	if err := enc.Encode(o); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return f.Close()
}
