package osmfile

import (
	"slices"
	"sort"

	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
)

// BoundaryData is a set of boundary relations with their member ways and
// those ways' node positions, held compactly: a way is the ids of its nodes,
// a node its id and position (in two sorted slices), and the tags and
// metadata of ways and nodes are not kept. Thailand's provinces and
// districts (2.0 million nodes) take about 65 MB this way; read as
// *osm.Node and *osm.Way they took about 600 MB.
type BoundaryData struct {
	Relations []*osm.Relation // sorted by id when read by ReadBoundaries

	ways map[osm.WayID]cway
	ids  []osm.NodeID // sorted, unique; pts[i] is the position of ids[i]
	pts  []orb.Point
	// nodes replaces ids and pts for data made from an *osm.OSM whose
	// nodes are sorted by id without duplicates (fromOSM).
	nodes osm.Nodes
}

// cway is a way: its version, the ids of its nodes and, only when the data
// gives them on the way itself (an augmented file), their positions, {0, 0}
// where a node has none there.
type cway struct {
	version int
	ids     []osm.NodeID
	aug     []orb.Point
}

func compactWay(version int, wn osm.WayNodes) cway {
	w := cway{version: version, ids: make([]osm.NodeID, len(wn))}
	for i, n := range wn {
		w.ids[i] = n.ID
		if n.Lon != 0 || n.Lat != 0 {
			if w.aug == nil {
				w.aug = make([]orb.Point, len(wn))
			}
			w.aug[i] = orb.Point{n.Lon, n.Lat}
		}
	}
	return w
}

// ReadBoundaries reads the relations keep accepts from name, and their
// member ways and those ways' nodes from name and from every file in more
// (the neighbours' cuts, see ReadBoundaryDataWith), in three passes per
// file. Only what the boundaries need is kept: the wanted way and node ids
// are sorted slices searched by the decoder's filters, and a node is kept as
// its position alone. An object found more than once is kept in its highest
// version (the first one read when the versions are equal).
func ReadBoundaries(name string, more []string, keep func(*osm.Relation) bool) (*BoundaryData, error) {
	rels := map[osm.RelationID]*osm.Relation{}
	err := scan(name, scanOpts{relations: true, filterRelation: keep}, func(obj osm.Object) {
		r := obj.(*osm.Relation)
		if old := rels[r.ID]; old == nil || r.Version > old.Version {
			rels[r.ID] = r
		}
	})
	if err != nil {
		return nil, err
	}
	d := &BoundaryData{Relations: make([]*osm.Relation, 0, len(rels))}
	var wantWays []osm.WayID
	for _, r := range rels {
		d.Relations = append(d.Relations, r)
		for _, m := range r.Members {
			if m.Type == osm.TypeWay {
				wantWays = append(wantWays, osm.WayID(m.Ref))
			}
		}
	}
	rels = nil
	sort.Slice(d.Relations, func(i, j int) bool { return d.Relations[i].ID < d.Relations[j].ID })
	slices.Sort(wantWays)
	wantWays = slices.Compact(wantWays)

	files := append([]string{name}, more...)
	d.ways = make(map[osm.WayID]cway, len(wantWays))
	for _, f := range files {
		err := scan(f, scanOpts{ways: true, filterWay: func(w *osm.Way) bool {
			_, ok := slices.BinarySearch(wantWays, w.ID)
			return ok
		}}, func(obj osm.Object) {
			w := obj.(*osm.Way)
			if old, ok := d.ways[w.ID]; ok && w.Version <= old.version {
				return
			}
			d.ways[w.ID] = compactWay(w.Version, w.Nodes)
		})
		if err != nil {
			return nil, err
		}
	}
	wantWays = nil

	refs := 0
	for _, w := range d.ways {
		refs += len(w.ids)
	}
	ids := make([]osm.NodeID, 0, refs)
	for _, w := range d.ways {
		ids = append(ids, w.ids...)
	}
	slices.Sort(ids)
	ids = slices.Clone(slices.Compact(ids)) // drop the spare capacity
	pts := make([]orb.Point, len(ids))
	version := make([]int32, len(ids)) // -1: not read
	for i := range version {
		version[i] = -1
	}
	for _, f := range files {
		err := scan(f, scanOpts{nodes: true, filterNode: func(n *osm.Node) bool {
			_, ok := slices.BinarySearch(ids, n.ID)
			return ok
		}}, func(obj osm.Object) {
			n := obj.(*osm.Node)
			i, _ := slices.BinarySearch(ids, n.ID)
			if version[i] >= 0 && int32(n.Version) <= version[i] {
				return
			}
			version[i] = int32(n.Version)
			pts[i] = orb.Point{n.Lon, n.Lat}
		})
		if err != nil {
			return nil, err
		}
	}
	// Leave out the nodes that are in no file.
	k := 0
	for i := range ids {
		if version[i] >= 0 {
			ids[k], pts[k] = ids[i], pts[i]
			k++
		}
	}
	if k < len(ids) {
		ids, pts = slices.Clone(ids[:k]), slices.Clone(pts[:k])
	}
	d.ids, d.pts = ids, pts
	return d, nil
}

// fromOSM makes BoundaryData of o for the assembly, without changing o.
// Like the old osmgeojson path, a way or node given twice counts in its
// last copy. Nodes that are sorted by id without duplicates (as
// ReadBoundaryData returns them) are searched in place.
func fromOSM(o *osm.OSM) *BoundaryData {
	d := &BoundaryData{Relations: o.Relations, ways: make(map[osm.WayID]cway, len(o.Ways))}
	for _, w := range o.Ways {
		if w != nil {
			d.ways[w.ID] = compactWay(w.Version, w.Nodes)
		}
	}
	sorted := true
	for i, n := range o.Nodes {
		if n == nil || (i > 0 && o.Nodes[i-1].ID >= n.ID) {
			sorted = false
			break
		}
	}
	if sorted {
		d.nodes = o.Nodes
		return d
	}
	idx := make([]int, 0, len(o.Nodes))
	for i, n := range o.Nodes {
		if n != nil {
			idx = append(idx, i)
		}
	}
	sort.SliceStable(idx, func(a, b int) bool { return o.Nodes[idx[a]].ID < o.Nodes[idx[b]].ID })
	for i, j := range idx {
		n := o.Nodes[j]
		if i+1 < len(idx) && o.Nodes[idx[i+1]].ID == n.ID {
			continue // a later copy follows
		}
		d.ids = append(d.ids, n.ID)
		d.pts = append(d.pts, orb.Point{n.Lon, n.Lat})
	}
	return d
}

// Point returns the position of the node id, if the data holds it.
func (d *BoundaryData) Point(id osm.NodeID) (orb.Point, bool) {
	if d.nodes != nil {
		i, ok := slices.BinarySearchFunc(d.nodes, id, func(n *osm.Node, id osm.NodeID) int {
			switch {
			case n.ID < id:
				return -1
			case n.ID > id:
				return 1
			}
			return 0
		})
		if !ok {
			return orb.Point{}, false
		}
		return d.nodes[i].Point(), true
	}
	i, ok := slices.BinarySearch(d.ids, id)
	if !ok {
		return orb.Point{}, false
	}
	return d.pts[i], true
}

// Ways is the number of ways held.
func (d *BoundaryData) Ways() int { return len(d.ways) }

// Nodes is the number of nodes held.
func (d *BoundaryData) Nodes() int {
	if d.nodes != nil {
		return len(d.nodes)
	}
	return len(d.ids)
}

// HasWay reports whether the data holds the way id.
func (d *BoundaryData) HasWay(id osm.WayID) bool {
	_, ok := d.ways[id]
	return ok
}

// FirstPoint returns the position of the first node of r's member ways (in
// member order, then way order) that the data holds; ok is false when it
// holds none.
func (d *BoundaryData) FirstPoint(r *osm.Relation) (p orb.Point, ok bool) {
	for _, m := range r.Members {
		if m.Type != osm.TypeWay {
			continue
		}
		w, have := d.ways[osm.WayID(m.Ref)]
		if !have {
			continue
		}
		for _, id := range w.ids {
			if p, ok := d.Point(id); ok {
				return p, true
			}
		}
	}
	return orb.Point{}, false
}

// line returns the positions of w's nodes; nodes absent from the data are
// left out, as osmgeojson does. A position given on the way itself wins.
func (d *BoundaryData) line(w cway) orb.LineString {
	ls := make(orb.LineString, 0, len(w.ids))
	for i, id := range w.ids {
		if w.aug != nil && (w.aug[i][0] != 0 || w.aug[i][1] != 0) {
			ls = append(ls, w.aug[i])
		} else if p, ok := d.Point(id); ok {
			ls = append(ls, p)
		}
	}
	return ls
}

// OSM returns the data as raw OSM objects, for code that needs them (the
// fixture writer, ReadBoundaryData): the relations as read, and ways and
// nodes with ids and positions only (and ways with their version), sorted
// by id. The nodes share one
// allocation.
func (d *BoundaryData) OSM() *osm.OSM {
	o := &osm.OSM{Relations: d.Relations}
	wayIDs := make([]osm.WayID, 0, len(d.ways))
	for id := range d.ways {
		wayIDs = append(wayIDs, id)
	}
	slices.Sort(wayIDs)
	o.Ways = make(osm.Ways, len(wayIDs))
	for i, id := range wayIDs {
		w := d.ways[id]
		wn := make(osm.WayNodes, len(w.ids))
		for j, nid := range w.ids {
			wn[j].ID = nid
			if w.aug != nil {
				wn[j].Lon, wn[j].Lat = w.aug[j][0], w.aug[j][1]
			}
		}
		o.Ways[i] = &osm.Way{ID: id, Visible: true, Version: w.version, Nodes: wn}
	}
	if d.nodes != nil {
		o.Nodes = d.nodes
		return o
	}
	slab := make([]osm.Node, len(d.ids))
	o.Nodes = make(osm.Nodes, len(d.ids))
	for i, id := range d.ids {
		slab[i] = osm.Node{ID: id, Visible: true, Lon: d.pts[i][0], Lat: d.pts[i][1]}
		o.Nodes[i] = &slab[i]
	}
	return o
}
