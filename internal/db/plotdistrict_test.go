package db

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/cartograph-asia/cartograph/internal/h3"
)

// A plot's district is one of its own province's. Where the simplified
// district of one country overlaps the province of another (Sarawak's
// Marudi over a corner of Brunei's Belait, which has no districts), the
// plot gets no district instead of the neighbour's, in the backfill and on
// the plot page.
func TestPlotDistrictOfOwnProvince(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	sq := func(x0, y0, x1, y1 float64) orb.MultiPolygon {
		return orb.MultiPolygon{{{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}, {x0, y0}}}}
	}
	ids, err := d.UpsertRegions(ctx, []Region{
		{OSMID: 11, Kind: "province", AdminLevel: 4, Country: "zz1", Slug: "one-zz1", NameEn: "One", Boundary: sq(100, 10, 100.1, 10.1), LandCells: 1},
		{OSMID: 21, Kind: "province", AdminLevel: 4, Country: "zz2", Slug: "two-zz2", NameEn: "Two", Boundary: sq(100.1, 10, 100.2, 10.1), LandCells: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	one, two := ids[0], ids[1]
	// zz1's district reaches 0.03° into zz2's province, stored first so the
	// district index finds it first; zz2's district covers only the east.
	ids, err = d.UpsertRegions(ctx, []Region{
		{OSMID: 12, Kind: "district", AdminLevel: 6, ParentID: &one, Country: "zz1", Slug: "one-zz1-marudi", NameEn: "Marudi", Boundary: sq(100, 10, 100.13, 10.1)},
		{OSMID: 22, Kind: "district", AdminLevel: 6, ParentID: &two, Country: "zz2", Slug: "two-zz2-east", NameEn: "East", Boundary: sq(100.15, 10, 100.2, 10.1)},
	})
	if err != nil {
		t.Fatal(err)
	}
	marudi, east := ids[0], ids[1]
	if err := d.ReloadRegions(ctx); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name     string
		lng      float64
		region   int
		district int // 0: none
	}{
		{"in zz1 and its district", 100.05, one, marudi},
		{"in zz2 under zz1's district", 100.115, two, 0},
		{"in zz2 and its district", 100.175, two, east},
	}
	cells := make([]string, len(cases))
	for i, c := range cases {
		cell, err := h3.CellFromLatLng(10.05, c.lng, h3.PlotRes)
		if err != nil {
			t.Fatal(err)
		}
		cells[i] = cell
		// The page of the blank plot.
		loc, err := d.LocatePlot(ctx, cell)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		got := 0
		if loc.DistrictID != nil {
			got = *loc.DistrictID
		}
		if loc.RegionID != c.region || got != c.district || (got == 0 && loc.District != "") {
			t.Errorf("%s: located in region %d, district %d %q; want %d, %d", c.name, loc.RegionID, got, loc.District, c.region, c.district)
		}
		// A plot claimed before the import.
		if _, err := d.Pool.Exec(ctx, `insert into plots (cell, first_photo, first_surveyor, claimed_at, photo_count) values ($1, $2, 'proto', $3, 1)`,
			cell, uuid.New(), time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	for _, country := range []string{"zz2", "zz1"} {
		if _, _, err := d.BackfillPlotsOf(ctx, country); err != nil {
			t.Fatal(err)
		}
	}
	for i, c := range cases {
		var region, district *int
		if err := d.Pool.QueryRow(ctx, `select region_id, district_id from plots where cell = $1`, cells[i]).Scan(&region, &district); err != nil {
			t.Fatal(err)
		}
		got := 0
		if district != nil {
			got = *district
		}
		if region == nil || *region != c.region || got != c.district {
			t.Errorf("%s: backfilled region %v, district %d; want %d, %d", c.name, region, got, c.region, c.district)
		}
	}
}
