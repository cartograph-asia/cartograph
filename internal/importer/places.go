package importer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/geo"
	"github.com/cartograph-asia/cartograph/internal/geo/osmfile"
)

// PlaceOptions configures Places.
type PlaceOptions struct {
	PBF  string // OSM extract
	BBox string // optional clip "minLng,minLat,maxLng,maxLat"
}

// PlaceStats reports what Places did.
type PlaceStats struct {
	Places          int
	OutsideRegions  int // places in no region (kept, region_id null)
	PlotsBackfilled int
}

// Places replaces the places table with the place nodes (osmfile.PlaceKinds) of the
// extract (inside BBox if set), each with the region containing it, then
// fills place names of plots that have none. Run it after Regions.
func Places(ctx context.Context, d *db.DB, o PlaceOptions) (*PlaceStats, error) {
	if o.PBF == "" {
		return nil, fmt.Errorf("importer: no OSM file given")
	}
	clip, err := geo.ParseBound(o.BBox)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	ps, err := osmfile.ReadPlaces(o.PBF, clip)
	if err != nil {
		return nil, err
	}
	if err := d.ReloadRegions(ctx); err != nil {
		return nil, err
	}
	st := &PlaceStats{Places: len(ps)}
	rows := make([]db.Place, len(ps))
	for i, p := range ps {
		rows[i] = db.Place{OSMType: p.OSMType, OSMID: p.OSMID, Kind: p.Kind, NameEn: p.NameEn, NameLocal: p.NameLocal, Lat: p.Point[1], Lng: p.Point[0]}
		id, ok, err := d.LocateRegion(ctx, p.Point[1], p.Point[0])
		if err != nil {
			return nil, err
		}
		if ok {
			rows[i].RegionID = &id
		} else {
			st.OutsideRegions++
		}
	}
	// Thailand until the import reads the country's settings (WP C).
	if err := d.ReplacePlaces(ctx, "th", rows); err != nil {
		return nil, err
	}
	if st.PlotsBackfilled, _, err = d.BackfillPlots(ctx); err != nil {
		return nil, err
	}
	slog.Info("imported places", "places", st.Places, "outside_regions", st.OutsideRegions,
		"plots_backfilled", st.PlotsBackfilled, "took", time.Since(start).Round(time.Millisecond))
	return st, nil
}
