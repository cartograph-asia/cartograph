package db

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

// PlaceNaming is one country's rules for naming plots (table place_naming).
type PlaceNaming struct {
	Country       string
	UrbanKinds    []string   // inside cities, preferred first
	UrbanRadiusKm float64    // radius for UrbanKinds
	Tiers         [][]string // everywhere, preferred first; places in one tier compete by distance
	RadiusKm      float64    // radius for Tiers, and for the place=city that makes a plot urban
	UrbanRegions  []string   // province slugs whose districts are all urban
}

// defaultNaming applies to countries without a place_naming row.
var defaultNaming = &PlaceNaming{Tiers: [][]string{{"village", "town"}, {"city"}}, RadiusKm: 5, UrbanRadiusKm: 3}

// loadNaming reads every country's rules.
func (d *DB) loadNaming(ctx context.Context) (map[string]*PlaceNaming, error) {
	rows, err := d.Pool.Query(ctx, `
		select country, urban_kinds, urban_radius_km, kinds, radius_km, urban_regions from place_naming`)
	if err != nil {
		return nil, fmt.Errorf("db: place naming: %w", err)
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (*PlaceNaming, error) {
		var n PlaceNaming
		var kinds []string
		var ur, r5 float32
		if err := r.Scan(&n.Country, &n.UrbanKinds, &ur, &kinds, &r5, &n.UrbanRegions); err != nil {
			return nil, err
		}
		n.UrbanRadiusKm, n.RadiusKm = float64(ur), float64(r5)
		for _, k := range kinds {
			n.Tiers = append(n.Tiers, strings.Split(k, "|"))
		}
		return &n, nil
	})
	if err != nil {
		return nil, fmt.Errorf("db: place naming: %w", err)
	}
	out := make(map[string]*PlaceNaming, len(list))
	for _, n := range list {
		out[n.Country] = n
	}
	return out, nil
}

// naming returns the rules for country from the cache loaded with the regions.
func (d *DB) naming(country string) *PlaceNaming {
	d.regions.mu.Lock()
	n := d.regions.naming[country]
	d.regions.mu.Unlock()
	if n == nil {
		return defaultNaming
	}
	return n
}

// placeCandidate is a place near a plot.
type placeCandidate struct {
	id             int64
	kind           string
	nameEn, nameTh string
	km             float64 // distance from the plot centre
}

// placesNear returns the places within radiusKm of (lat, lng), using the
// (lat, lng) index for a box prefilter and an equirectangular distance.
func placesNear(ctx context.Context, q querier, lat, lng, radiusKm float64) ([]placeCandidate, error) {
	const kmPerDeg = 111.195
	dLat := radiusKm / kmPerDeg
	dLng := dLat / math.Cos(lat*math.Pi/180)
	rows, err := q.Query(ctx, `
		select id, kind, name_en, coalesce(name_th, ''), lat, lng from places
		where lat between $1 and $2 and lng between $3 and $4`, lat-dLat, lat+dLat, lng-dLng, lng+dLng)
	if err != nil {
		return nil, fmt.Errorf("db: places near: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (placeCandidate, error) {
		var c placeCandidate
		var plat, plng float64
		err := r.Scan(&c.id, &c.kind, &c.nameEn, &c.nameTh, &plat, &plng)
		c.km = kmPerDeg * math.Hypot(plat-lat, (plng-lng)*math.Cos(lat*math.Pi/180))
		return c, err
	})
	if err != nil {
		return nil, fmt.Errorf("db: places near: %w", err)
	}
	return out, nil
}

// maxRadiusKm is the largest radius the rules look at.
func (n *PlaceNaming) maxRadiusKm() float64 { return max(n.RadiusKm, n.UrbanRadiusKm) }

// choose picks the place that names a plot from the candidates around it, or
// nil. urbanDistrict reports whether the plot's district is all urban; a
// place=city within RadiusKm makes a plot urban too.
func (n *PlaceNaming) choose(cs []placeCandidate, urbanDistrict bool) *placeCandidate {
	nearest := func(kinds []string, radius float64) *placeCandidate {
		var best *placeCandidate
		for i := range cs {
			c := &cs[i]
			if c.km > radius || !slices.Contains(kinds, c.kind) {
				continue
			}
			if best == nil || c.km < best.km || (c.km == best.km && c.id < best.id) {
				best = c
			}
		}
		return best
	}
	if urbanDistrict || nearest([]string{"city"}, n.RadiusKm) != nil {
		for _, k := range n.UrbanKinds {
			if c := nearest([]string{k}, n.UrbanRadiusKm); c != nil {
				return c
			}
		}
	}
	for _, tier := range n.Tiers {
		if c := nearest(tier, n.RadiusKm); c != nil {
			return c
		}
	}
	return nil
}
