package osmfile

import (
	"sort"

	"github.com/paulmach/osm"
)

// ReadBoundaryDataFiles is ReadBoundaryData over several files read as one,
// such as the cuts of two neighbouring extracts (map/merge-cut.sh makes the
// same union as a file): the relations keep accepts in any of them, with
// their member ways and those ways' nodes from whichever file holds them. An
// object found in more than one file is kept once, in its highest version.
// The result is sorted by id within nodes, ways and relations.
func ReadBoundaryDataFiles(names []string, keep func(*osm.Relation) bool) (*osm.OSM, error) {
	rels := map[osm.RelationID]*osm.Relation{}
	for _, name := range names {
		err := scan(name, scanOpts{relations: true, filterRelation: keep}, func(obj osm.Object) {
			r := obj.(*osm.Relation)
			if old := rels[r.ID]; old == nil || r.Version > old.Version {
				rels[r.ID] = r
			}
		})
		if err != nil {
			return nil, err
		}
	}
	wayIDs := map[osm.WayID]bool{}
	for _, r := range rels {
		for _, m := range r.Members {
			if m.Type == osm.TypeWay {
				wayIDs[osm.WayID(m.Ref)] = true
			}
		}
	}
	ways := map[osm.WayID]*osm.Way{}
	for _, name := range names {
		err := scan(name, scanOpts{ways: true, filterWay: func(w *osm.Way) bool { return wayIDs[w.ID] }}, func(obj osm.Object) {
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
	for _, name := range names {
		err := scan(name, scanOpts{nodes: true, filterNode: func(n *osm.Node) bool { return nodeIDs[n.ID] }}, func(obj osm.Object) {
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
