package osmfile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/paulmach/osm"
)

// Two neighbouring extracts. A holds relation 1 (version 1) and relation 3
// with only way 10 of relation 1; B holds relation 1 in version 2, both of
// its ways (way 10 also in a later version, node 2 moved) and relation 3's
// only way 30, but not relation 3 itself. Relation 4 is in B and not kept.
const multiA = `<?xml version="1.0" encoding="UTF-8"?>
<osm version="0.6">
  <node id="1" version="1" lat="10.00" lon="100.00"/>
  <node id="2" version="1" lat="10.00" lon="100.10"/>
  <node id="3" version="1" lat="10.10" lon="100.10"/>
  <way id="10" version="1"><nd ref="1"/><nd ref="2"/><nd ref="3"/></way>
  <relation id="1" version="1">
    <member type="way" ref="10" role="outer"/>
    <member type="way" ref="11" role="outer"/>
    <tag k="type" v="boundary"/><tag k="boundary" v="administrative"/><tag k="name" v="old"/>
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
  <way id="10" version="2"><nd ref="1"/><nd ref="2"/><nd ref="3"/></way>
  <way id="11" version="1"><nd ref="3"/><nd ref="4"/><nd ref="1"/></way>
  <way id="30" version="1"><nd ref="5"/><nd ref="6"/><nd ref="7"/><nd ref="5"/></way>
  <way id="40" version="1"><nd ref="8"/><nd ref="5"/></way>
  <relation id="1" version="2">
    <member type="way" ref="10" role="outer"/>
    <member type="way" ref="11" role="outer"/>
    <tag k="type" v="boundary"/><tag k="boundary" v="administrative"/><tag k="name" v="new"/>
  </relation>
  <relation id="4" version="1">
    <member type="way" ref="40" role="outer"/>
    <tag k="type" v="boundary"/><tag k="boundary" v="administrative"/><tag k="name" v="not kept"/>
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

func TestReadBoundaryDataFiles(t *testing.T) {
	dir := t.TempDir()
	a := writeOSM(t, dir, "a.osm", multiA)
	b := writeOSM(t, dir, "b.osm", multiB)
	keep := func(r *osm.Relation) bool { return r.ID != 4 }

	o, err := ReadBoundaryDataFiles([]string{a, b}, keep)
	if err != nil {
		t.Fatal(err)
	}
	var rels, ways, nodes []int64
	for _, r := range o.Relations {
		rels = append(rels, int64(r.ID))
	}
	for _, w := range o.Ways {
		ways = append(ways, int64(w.ID))
	}
	for _, n := range o.Nodes {
		nodes = append(nodes, int64(n.ID))
	}
	if !reflect.DeepEqual(rels, []int64{1, 3}) || !reflect.DeepEqual(ways, []int64{10, 11, 30}) || !reflect.DeepEqual(nodes, []int64{1, 2, 3, 4, 5, 6, 7}) {
		t.Fatalf("relations %v, ways %v, nodes %v", rels, ways, nodes)
	}
	if o.Relations[0].Tags.Find("name") != "new" || o.Ways[0].Version != 2 || o.Nodes[1].Lat != 9.99 {
		t.Fatalf("not the highest versions: %v, way 10 v%d, node 2 lat %v", o.Relations[0].Tags, o.Ways[0].Version, o.Nodes[1].Lat)
	}

	// Either order gives the same data, and both relations assemble whole.
	o2, err := ReadBoundaryDataFiles([]string{b, a}, keep)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(o, o2) {
		t.Fatal("the order of the files changes the result")
	}
	as, err := Assemble(o, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(builtIDs(as), []int64{1, 3}) || len(as.NotWhole)+len(as.NotClosed) != 0 {
		t.Fatalf("%+v", as)
	}

	// A alone: relation 1 lacks way 11 and relation 3 its only way.
	one, err := ReadBoundaryDataFiles([]string{a}, keep)
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
}

func TestReadBoundaryDataFilesMissing(t *testing.T) {
	if _, err := ReadBoundaryDataFiles([]string{filepath.Join(t.TempDir(), "none.osm")}, func(*osm.Relation) bool { return true }); err == nil {
		t.Fatal("no error for a missing file")
	}
}
