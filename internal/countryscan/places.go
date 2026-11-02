package countryscan

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
	"github.com/cartograph-asia/cartograph/internal/h3"
)

// placeNode is a place=* node inside the country.
type placeNode struct {
	id       int64
	kind     string
	en       string // English-capable name, if any
	named    bool
	lat, lng float64
	latin    bool
	pop      int
}

// cityCount is how many of the largest cities are looked at.
const cityCount = 5

// urbanCandidates are the place values a city's districts are named after,
// in the order the Thai row prefers them.
var urbanCandidates = []string{"quarter", "neighbourhood", "suburb"}

// minUrbanNodes is how many English-named nodes of a kind the largest cities
// must hold together for the kind to be proposed.
const minUrbanNodes = 5

func cellID(lat, lng float64, res int) (uint64, error) { return h3.CellID(lat, lng, res) }

// places counts the place nodes in the country and what lies around its
// largest cities.
func (s *scanner) places() error {
	var nodes []placeNode
	err := osmfile.Nodes(s.o.PBF, func(n *osm.Node) bool { return n.Tags.Find("place") != "" }, func(n *osm.Node) {
		id, err := cellID(n.Lat, n.Lon, s.o.Res)
		if err != nil || !cellIn(s.inCountry, id) {
			return
		}
		t := tagSet{}
		for _, tg := range n.Tags {
			if tg.Key == "name" || tg.Key == "int_name" || strings.HasPrefix(tg.Key, "name:") {
				t[tg.Key] = tg.Value
			}
		}
		en, _ := englishName(t, s.lang)
		nodes = append(nodes, placeNode{
			id: int64(n.ID), kind: n.Tags.Find("place"), en: en, named: t["name"] != "" || en != "",
			lat: n.Lat, lng: n.Lon, latin: en != "", pop: population(n.Tags.Find("population")),
		})
	})
	if err != nil {
		return err
	}
	p := &s.r.Places
	p.Nodes = len(nodes)
	p.Kinds = kindStats(nodes, 0, 0, 0, false)
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].id < nodes[j].id })

	var cities []placeNode
	for _, n := range nodes {
		if n.kind == "city" && n.pop > 0 && n.en != "" {
			cities = append(cities, n)
		}
	}
	sort.SliceStable(cities, func(i, j int) bool { return cities[i].pop > cities[j].pop })
	// One entry per name: a city mapped twice counts once, with the larger population.
	seen := map[string]bool{}
	uniq := cities[:0]
	for _, c := range cities {
		if !seen[c.en] {
			seen[c.en] = true
			uniq = append(uniq, c)
		}
	}
	cities = uniq
	if len(cities) > cityCount {
		cities = cities[:cityCount]
	}
	totals := map[string]int{}
	for _, c := range cities {
		cs := CityStat{Name: c.en, Population: c.pop, Kinds: kindStats(nodes, c.lat, c.lng, CityRadiusKm, true)}
		// the city node itself is not "around" the city
		for i := range cs.Kinds {
			if cs.Kinds[i].Kind == "city" && cs.Kinds[i].Total > 0 {
				cs.Kinds[i].Total--
				cs.Kinds[i].Named--
				cs.Kinds[i].Latin--
			}
		}
		p.Cities = append(p.Cities, cs)
		for _, k := range cs.Kinds {
			totals[k.Kind] += k.Latin
		}
	}
	nm := &s.r.Naming
	nm.UrbanRadiusKm, nm.Kinds, nm.RadiusKm = 3, "{village|town,city}", 5
	for _, k := range urbanCandidates {
		if totals[k] >= minUrbanNodes {
			nm.UrbanKinds = append(nm.UrbanKinds, k)
		}
	}
	switch {
	case len(cities) == 0:
		nm.Basis = "no place=city node with a population tag and an English-capable name; urban_kinds left empty"
	default:
		nm.Basis = "English-named nodes within " + strconv.Itoa(CityRadiusKm) + " km of the " + strconv.Itoa(len(cities)) +
			" largest cities (by population tag): kinds with at least " + strconv.Itoa(minUrbanNodes) + " are proposed; the radii and tiers are Thailand's, not measured here"
	}
	return nil
}

// population parses an OSM population value ("8 305 218", "1,200,000").
func population(v string) int {
	v = strings.NewReplacer(" ", "", ",", "", ".", "", "_", "").Replace(strings.TrimSpace(v))
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// kindStats counts nodes per place value, all of them or, with near set,
// those within radiusKm of (lat, lng).
func kindStats(nodes []placeNode, lat, lng, radiusKm float64, near bool) []KindStat {
	const kmPerDeg = 111.195
	cosLat := math.Cos(lat * math.Pi / 180)
	m := map[string]*KindStat{}
	for _, n := range nodes {
		if near && kmPerDeg*math.Hypot(n.lat-lat, (n.lng-lng)*cosLat) > radiusKm {
			continue
		}
		k := m[n.kind]
		if k == nil {
			k = &KindStat{Kind: n.kind}
			m[n.kind] = k
		}
		k.Total++
		if n.named {
			k.Named++
		}
		if n.latin {
			k.Latin++
		}
	}
	out := make([]KindStat, 0, len(m))
	for _, k := range m {
		out = append(out, *k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}
