package osmfile

import (
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
// relations. With more empty it reads what ReadBoundaryData reads. It is
// ReadBoundaries(name, more, keep) as raw OSM data.
func ReadBoundaryDataWith(name string, more []string, keep func(*osm.Relation) bool) (*osm.OSM, error) {
	d, err := ReadBoundaries(name, more, keep)
	if err != nil {
		return nil, err
	}
	return d.OSM(), nil
}
