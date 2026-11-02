package osmfile

import (
	"sort"

	"github.com/paulmach/osm"
)

// ReadBoundaryDataWith is ReadBoundaryData for a country whose relations
// have member ways in a neighbour's extract (docs/countries/
// incomplete-relations.md): the relations keep accepts are read from name,
// the country's own cut, only; their member ways and those ways' nodes are
// read from name and from every file in more, the neighbours' cuts. The
// neighbours' own relations are never read, so keep may be as broad as a
// district level. An object found in more than one file is kept once, in
// its highest version; the result is sorted by id within nodes, ways and
// relations. With more empty it reads what ReadBoundaryData reads.
func ReadBoundaryDataWith(name string, more []string, keep func(*osm.Relation) bool) (*osm.OSM, error) {
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
	files := append([]string{name}, more...)
	wayIDs := map[osm.WayID]bool{}
	for _, r := range rels {
		for _, m := range r.Members {
			if m.Type == osm.TypeWay {
				wayIDs[osm.WayID(m.Ref)] = true
			}
		}
	}
	ways := map[osm.WayID]*osm.Way{}
	for _, f := range files {
		err := scan(f, scanOpts{ways: true, filterWay: func(w *osm.Way) bool { return wayIDs[w.ID] }}, func(obj osm.Object) {
			w := obj.(*osm.Way)
			if old := ways[w.ID]; old == nil || w.Version > old.Version {
				ways[w.ID] = w
			}
		})
		if err != nil {
			return nil, err
		}
	}
	nodeIDs := map[osm.NodeID]bool{}
	for _, w := range ways {
		for _, n := range w.Nodes {
			nodeIDs[n.ID] = true
		}
	}
	nodes := map[osm.NodeID]*osm.Node{}
	for _, f := range files {
		err := scan(f, scanOpts{nodes: true, filterNode: func(n *osm.Node) bool { return nodeIDs[n.ID] }}, func(obj osm.Object) {
			n := obj.(*osm.Node)
			if old := nodes[n.ID]; old == nil || n.Version > old.Version {
				nodes[n.ID] = n
			}
		})
		if err != nil {
			return nil, err
		}
	}
	o := &osm.OSM{}
	for _, n := range nodes {
		o.Nodes = append(o.Nodes, n)
	}
	for _, w := range ways {
		o.Ways = append(o.Ways, w)
	}
	for _, r := range rels {
		o.Relations = append(o.Relations, r)
	}
	sort.Slice(o.Nodes, func(i, j int) bool { return o.Nodes[i].ID < o.Nodes[j].ID })
	sort.Slice(o.Ways, func(i, j int) bool { return o.Ways[i].ID < o.Ways[j].ID })
	sort.Slice(o.Relations, func(i, j int) bool { return o.Relations[i].ID < o.Relations[j].ID })
	return o, nil
}
