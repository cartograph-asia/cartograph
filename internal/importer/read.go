package importer

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/osm"

	"github.com/cartograph-asia/cartograph/internal/countries"
	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// province is one province relation of a run.
type province struct {
	osmID    int64
	level    int
	names    osmfile.Named
	kindName string           // regions.kind_name, "" for NULL
	shape    orb.MultiPolygon // simplified
	slug     string
	stored   *db.StoredRegion // the stored row with this osm_id, any country

	districts []*district // the run's districts with their centre in it

	// land cells, see regionPlan.landCells
	indexID    int      // id in the combined index (stored id, or negative for a new province)
	land       int      // land cells
	held       int      // land cells another country holds (border cells skipped)
	inDistrict int      // land cells inside one of its districts
	cells      []uint64 // the land cells along the Hilbert curve, until numbered
	newNumbers int      // cells without a number (dry run: counted; run: added)
	id         int      // region id once stored
}

// district is one district relation of a run.
type district struct {
	osmID    int64
	level    int
	names    osmfile.Named
	kindName string
	shape    orb.MultiPolygon
	province *province
	slug     string
	stored   *db.StoredRegion
}

// dropped counts the district relations a run leaves out.
type dropped struct {
	NoEnglish, Outside, NotAssembled, OtherCountry, OutsideParent int
}

// regionSet is what an extract gives for one country, before the database
// is consulted.
type regionSet struct {
	provinces     []*province // by OSM id
	districts     []*district // by OSM id
	notAssembled  []int64     // province-level relations in scope without geometry (check 6)
	scopeUnknown  []int64     // province-level relations without geometry whose scope cannot be told (Parent)
	partial       map[int64]osmfile.PartialInfo
	outsideParent int // province-level relations left out by Level.Parent
	vanished      []int64
	dropped       dropped
	relations     int // relations read
}

// readRegions reads the provinces and districts of c from the OSM file
// path: the relations that match the settings' levels (osmfile.MatchLevel,
// Level.Parent), assembled by osmfile.Assemble, simplified with tol and
// clipped to clip, with their names by the settings. Districts are kept
// when their centre lies in one of the provinces read and they have an
// English name. Slugs are set later (resolveSlugs), with the stored rows.
//
// more are the neighbours' cuts (countries.Import.Neighbours): the
// relations come from path alone, their member ways and nodes from path and
// more (osmfile.ReadBoundaries), so a relation the own extract cuts
// at a border can be assembled. Without neighbours path is read as before.
func readRegions(path string, more []string, c country, clip orb.Bound, tol float64) (*regionSet, error) {
	imp := c.Imp
	pl, dl := imp.Province, imp.District
	keep := func(r *osm.Relation) bool {
		id := int64(r.ID)
		return osmfile.MatchLevel(r, pl) || osmfile.MatchLevel(r, dl) ||
			(pl.Parent != 0 && id == pl.Parent) || (dl.Parent != 0 && id == dl.Parent)
	}
	start := time.Now()
	raw, err := osmfile.ReadBoundaries(path, more, keep)
	if err != nil {
		return nil, err
	}
	slog.Info("read boundaries", "file", path, "neighbours", more, "relations", len(raw.Relations), "ways", raw.Ways(), "nodes", raw.Nodes(),
		"took", time.Since(start).Round(time.Millisecond))
	partial := make(map[int64]bool, len(imp.Partial))
	for _, id := range imp.Partial {
		partial[id] = true
	}
	a, err := raw.Assemble(partial)
	if err != nil {
		return nil, err
	}
	set := &regionSet{partial: a.Partial, relations: len(raw.Relations)}
	built := make(map[int64]osmfile.Boundary, len(a.Built))
	for _, b := range a.Built {
		built[b.OSMID] = b
	}
	parentOf := func(l countries.Level) (orb.MultiPolygon, error) {
		if l.Parent == 0 {
			return nil, nil
		}
		b, ok := built[l.Parent]
		if !ok {
			return nil, fmt.Errorf("importer: the parent relation r%d is not in %s or does not assemble", l.Parent, path)
		}
		return b.Geometry, nil
	}
	pParent, err := parentOf(pl)
	if err != nil {
		return nil, err
	}
	dParent, err := parentOf(dl)
	if err != nil {
		return nil, err
	}

	// Relations without geometry: a province-level one in scope fails check
	// 6; districts are only counted.
	rels := make(map[int64]*osm.Relation, len(raw.Relations))
	for _, r := range raw.Relations {
		rels[int64(r.ID)] = r
	}
	for _, id := range slices.Concat(a.NotWhole, a.NotClosed) {
		r := rels[id]
		switch {
		case osmfile.MatchLevel(r, pl):
			if pParent == nil || slices.Contains(pl.Include, id) {
				set.notAssembled = append(set.notAssembled, id)
				continue
			}
			switch in, known := memberInside(r, raw, pParent); {
			case !known:
				set.scopeUnknown = append(set.scopeUnknown, id)
			case in:
				set.notAssembled = append(set.notAssembled, id)
			default:
				set.outsideParent++
			}
		case osmfile.MatchLevel(r, dl):
			set.dropped.NotAssembled++
		}
	}
	slices.Sort(set.notAssembled)
	raw = nil

	type candidate struct {
		b     osmfile.Boundary
		shape orb.MultiPolygon
	}
	var dcands []candidate
	for _, b := range a.Built {
		r := &osm.Relation{ID: osm.RelationID(b.OSMID), Tags: b.Tags}
		isP, isD := osmfile.MatchLevel(r, pl), osmfile.MatchLevel(r, dl)
		if !isP && !isD {
			continue // a parent relation only
		}
		simple := geo.Simplify(b.Geometry, tol)
		if len(simple) == 0 {
			slog.Warn("boundary vanished after simplification", "relation", b.OSMID)
			set.vanished = append(set.vanished, b.OSMID)
			continue
		}
		if isP {
			if pParent != nil && !slices.Contains(pl.Include, b.OSMID) && !planar.MultiPolygonContains(pParent, centre(simple)) {
				set.outsideParent++
				continue
			}
			if !geo.NoBound(clip) && !simple.Bound().Intersects(clip) {
				continue
			}
			n := osmfile.LevelNames(b.Tags, imp, pl)
			set.provinces = append(set.provinces, &province{osmID: b.OSMID, level: adminLevel(b.Tags, pl), names: n,
				kindName: osmfile.KindName(n.RawEn, n.RawLocal, pl, "province"), shape: simple})
			continue
		}
		if dParent != nil && !slices.Contains(dl.Include, b.OSMID) && !planar.MultiPolygonContains(dParent, centre(simple)) {
			set.dropped.OutsideParent++
			continue
		}
		if !geo.NoBound(clip) && !simple.Bound().Intersects(clip) {
			continue
		}
		dcands = append(dcands, candidate{b, simple})
	}

	// Districts belong to the province that holds their centre; an index of
	// this run's provinces only, as before the country import.
	shapes := make([]geo.RegionShape, len(set.provinces))
	for i, p := range set.provinces {
		shapes[i] = geo.RegionShape{ID: i, Geometry: p.shape}
	}
	idx := geo.NewRegionIndex(shapes)
	for _, x := range dcands {
		pi, ok := idx.Contains(centre(x.shape))
		n := osmfile.LevelNames(x.b.Tags, imp, dl)
		switch {
		case !ok:
			set.dropped.Outside++
			continue
		case n.En == "":
			set.dropped.NoEnglish++
			continue
		}
		p := set.provinces[pi]
		d := &district{osmID: x.b.OSMID, level: adminLevel(x.b.Tags, dl), names: n,
			kindName: osmfile.KindName(n.RawEn, n.RawLocal, dl, "district"), shape: x.shape, province: p}
		set.districts = append(set.districts, d)
		p.districts = append(p.districts, d)
	}
	return set, nil
}

// centre is the point a district or a province is placed by: the centroid
// of its geometry, or its first vertex when the centroid lies outside.
func centre(mp orb.MultiPolygon) orb.Point {
	c, _ := planar.CentroidArea(mp)
	if !planar.MultiPolygonContains(mp, c) {
		c = mp[0][0][0]
	}
	return c
}

// memberInside reports whether the first node found of r's member ways lies
// in parent; known is false when none of its nodes is in the data.
func memberInside(r *osm.Relation, raw *osmfile.BoundaryData, parent orb.MultiPolygon) (in, known bool) {
	p, ok := raw.FirstPoint(r)
	if !ok {
		return false, false
	}
	return planar.MultiPolygonContains(parent, p), true
}

// adminLevel is the relation's admin_level, or the level's first one when
// the tag is not a number (an Include relation without it).
func adminLevel(t osm.Tags, l countries.Level) int {
	if n, err := strconv.Atoi(t.Find("admin_level")); err == nil {
		return n
	}
	if len(l.AdminLevels) > 0 {
		return l.AdminLevels[0]
	}
	return 0
}

// resolveSlugs gives every province and district its slug: a stored row of
// the same country keeps its slug, a new one gets one by the settings
// (osmfile.ResolveSlugs: Slug of the English name, the country suffix on
// provinces when SlugSuffix, a district's from its province's slug,
// duplicates by decision 7). Districts stored for another country are left
// out (that country's import placed them first).
func resolveSlugs(set *regionSet, c country, stored map[int64]*db.StoredRegion) {
	own := func(s *db.StoredRegion) string {
		if s != nil && s.Country == c.Code {
			return s.Slug
		}
		return ""
	}
	sibs := make([]osmfile.Sibling, len(set.provinces))
	for i, p := range set.provinces {
		p.stored = stored[p.osmID]
		sibs[i] = osmfile.Sibling{OSMID: p.osmID, Names: p.names, Stored: own(p.stored)}
	}
	osmfile.ResolveSlugs(sibs, func(base string) string {
		if c.Imp.SlugSuffix {
			return base + "-" + c.Slug
		}
		return base
	})
	for i, p := range set.provinces {
		p.names, p.slug = sibs[i].Names, sibs[i].Slug
	}

	kept := set.districts[:0]
	for _, d := range set.districts {
		d.stored = stored[d.osmID]
		if d.stored != nil && d.stored.Country != c.Code {
			set.dropped.OtherCountry++
			continue
		}
		kept = append(kept, d)
	}
	set.districts = kept
	for _, p := range set.provinces {
		ds := p.districts[:0]
		for _, d := range p.districts {
			if d.stored == nil || d.stored.Country == c.Code {
				ds = append(ds, d)
			}
		}
		p.districts = ds
	}
	if c.Imp.LegacyDuplicateSlug {
		legacyDistrictSlugs(set.districts)
		return
	}
	for _, p := range set.provinces {
		ds := p.districts
		sibs := make([]osmfile.Sibling, len(ds))
		for i, d := range ds {
			sibs[i] = osmfile.Sibling{OSMID: d.osmID, Names: d.names, Stored: own(d.stored)}
		}
		osmfile.ResolveSlugs(sibs, func(base string) string { return p.slug + "-" + base })
		for i, d := range ds {
			d.names, d.slug = sibs[i].Names, sibs[i].Slug
		}
	}
}

// legacyDistrictSlugs gives the run's districts their slugs by the rule
// Thailand's districts were first imported with
// (countries.Import.LegacyDuplicateSlug): a stored slug is kept; the others,
// in OSM id order, get their province's slug, "-" and Slug of the stripped
// English name, and when that slug is taken (by a stored row of the run or
// an earlier district) "-<osm id>" appended. Names stay stripped.
func legacyDistrictSlugs(ds []*district) {
	taken := map[string]bool{}
	for _, d := range ds {
		if d.stored != nil {
			d.slug = d.stored.Slug
			taken[d.slug] = true
		}
	}
	order := slices.Clone(ds)
	slices.SortFunc(order, func(a, b *district) int { return cmp.Compare(a.osmID, b.osmID) })
	for _, d := range order {
		if d.stored != nil {
			continue
		}
		slug := d.province.slug + "-" + osmfile.Slug(d.names.En)
		if taken[slug] {
			slug = fmt.Sprintf("%s-%d", slug, d.osmID)
		}
		taken[slug] = true
		d.slug = slug
	}
}
