package osmfile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/paulmach/osm"
)

// A country's cut and a neighbour's. A holds relations 1 and 3 but only way
// 10 of relation 1 and no way of relation 3; B holds both ways of relation 1
// (way 10 in a later version, node 2 moved), relation 3's only way 30, a
// later version of relation 1 itself and a relation 4 of its own.
const multiA = `<?xml version="1.0" encoding="UTF-8"?>
<osm version="0.6">
  <node id="1" version="1" lat="10.00" lon="100.00"/>
  <node id="2" version="1" lat="10.00" lon="100.10"/>
  <node id="3" version="1" lat="10.10" lon="100.10"/>
  <way id="10" version="1"><nd ref="1"/><nd ref="2"/><nd ref="3"/></way>
  <relation id="1" version="1">
    <member type="way" ref="10" role="outer"/>
    <member type="way" ref="11" role="outer"/>
    <tag k="type" v="boundary"/><tag k="boundary" v="administrative"/><tag k="name" v="own"/>
  </relation>
  <relation id="3" version="1">
    <member type="way" ref="30" role="outer"/>
    <tag k="type" v="boundary"/><tag k="boundary" v="administrative"/><tag k="name" v="island"/>
  </relation>
</osm>`

const multiB = `<?xml version="1.0" encoding="UTF-8"?>
<osm version="0.6">
  <node id="1" version="1" lat="10.00" lon="100.00"/>
  <node id="2" version="2" lat="9.99" lon="100.10"/>
  <node id="3" version="1" lat="10.10" lon="100.10"/>
  <node id="4" version="1" lat="10.10" lon="100.00"/>
  <node id="5" version="1" lat="11.00" lon="101.00"/>
  <node id="6" version="1" lat="11.00" lon="101.10"/>
  <node id="7" version="1" lat="11.10" lon="101.10"/>
  <node id="8" version="1" lat="12.00" lon="102.00"/>
  <node id="9" version="1" lat="12.10" lon="102.00"/>
  <node id="10" version="1" lat="12.10" lon="102.10"/>
  <way id="10" version="2"><nd ref="1"/><nd ref="2"/><nd ref="3"/></way>
  <way id="11" version="1"><nd ref="3"/><nd ref="4"/><nd ref="1"/></way>
  <way id="30" version="1"><nd ref="5"/><nd ref="6"/><nd ref="7"/><nd ref="5"/></way>
  <way id="40" version="1"><nd ref="8"/><nd ref="9"/><nd ref="10"/><nd ref="8"/></way>
  <relation id="1" version="2">
    <member type="way" ref="10" role="outer"/>
    <member type="way" ref="11" role="outer"/>
    <tag k="type" v="boundary"/><tag k="boundary" v="administrative"/><tag k="name" v="neighbour's"/>
  </relation>
  <relation id="4" version="1">
    <member type="way" ref="40" role="outer"/>
    <tag k="type" v="boundary"/><tag k="boundary" v="administrative"/><tag k="name" v="neighbour"/>
  </relation>
</osm>`

func writeOSM(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func ids(o *osm.OSM) (rels, ways, nodes []int64) {
	for _, r := range o.Relations {
		rels = append(rels, int64(r.ID))
	}
	for _, w := range o.Ways {
		ways = append(ways, int64(w.ID))
	}
	for _, n := range o.Nodes {
		nodes = append(nodes, int64(n.ID))
	}
	return rels, ways, nodes
}

func TestReadBoundaryDataWith(t *testing.T) {
	dir := t.TempDir()
	a := writeOSM(t, dir, "a.osm", multiA)
	b := writeOSM(t, dir, "b.osm", multiB)
	all := func(*osm.Relation) bool { return true }

	o, err := ReadBoundaryDataWith(a, []string{b}, all)
	if err != nil {
		t.Fatal(err)
	}
	rels, ways, nodes := ids(o)
	if !reflect.DeepEqual(rels, []int64{1, 3}) || !reflect.DeepEqual(ways, []int64{10, 11, 30}) || !reflect.DeepEqual(nodes, []int64{1, 2, 3, 4, 5, 6, 7}) {
		t.Fatalf("relations %v, ways %v, nodes %v", rels, ways, nodes)
	}
	// The relation is the country's own; ways and nodes the latest version.
	if o.Relations[0].Tags.Find("name") != "own" || o.Ways[0].Version != 2 || o.Nodes[1].Lat != 9.99 {
		t.Fatalf("relation %v, way 10 v%d, node 2 lat %v", o.Relations[0].Tags, o.Ways[0].Version, o.Nodes[1].Lat)
	}
	as, err := Assemble(o, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(builtIDs(as), []int64{1, 3}) || len(as.NotWhole)+len(as.NotClosed) != 0 {
		t.Fatalf("%+v", as)
	}

	// A neighbour listed twice changes nothing.
	o2, err := ReadBoundaryDataWith(a, []string{b, b}, all)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(o, o2) {
		t.Fatal("a repeated file changes the result")
	}

	// The country's cut alone: relation 1 lacks way 11, relation 3 its only way.
	one, err := ReadBoundaryDataWith(a, nil, all)
	if err != nil {
		t.Fatal(err)
	}
	as, err = Assemble(one, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(as.Built) != 0 || !reflect.DeepEqual(as.NotWhole, []int64{1, 3}) {
		t.Fatalf("%+v", as)
	}
	// It reads what ReadBoundaryData reads.
	plain, err := ReadBoundaryData(a, all)
	if err != nil {
		t.Fatal(err)
	}
	pr, pw, pn := ids(plain)
	or, ow, on := ids(one)
	if !reflect.DeepEqual([][]int64{pr, pw, pn}, [][]int64{or, ow, on}) {
		t.Fatalf("ReadBoundaryData %v %v %v, ReadBoundaryDataWith %v %v %v", pr, pw, pn, or, ow, on)
	}
}

func TestReadBoundaryDataWithMissingFile(t *testing.T) {
	dir := t.TempDir()
	a := writeOSM(t, dir, "a.osm", multiA)
	all := func(*osm.Relation) bool { return true }
	if _, err := ReadBoundaryDataWith(a, []string{filepath.Join(dir, "none.osm")}, all); err == nil {
		t.Fatal("no error for a missing neighbour")
	}
	if _, err := ReadBoundaryDataWith(filepath.Join(dir, "none.osm"), nil, all); err == nil {
		t.Fatal("no error for a missing cut")
	}
}
