package db

import (
	"context"
	"testing"
	"time"

	"github.com/paulmach/orb"

	"github.com/cartograph-asia/cartograph/internal/h3"
)

func square(w, s float64) orb.MultiPolygon {
	return orb.MultiPolygon{{{{w, s}, {w + 1, s}, {w + 1, s + 1}, {w, s + 1}, {w, s}}}}
}

// The language of a region's local names and the kind word come from the
// region's own country, on every page that shows them: Thailand keeps "th"
// and "Province"; Laos has its own language; a country that is not in the
// list (the synthetic "zz") has none; a country may call its first level
// districts or regions.
func TestPagesLangAndKindPerCountry(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	testRegion(t, d) // Chon Buri, th, 100.5-101.5 E, 12.5-13.5 N
	rs := []Region{
		{OSMID: 10, Kind: "province", AdminLevel: 4, Country: "la", Slug: "vientiane-laos", NameEn: "Vientiane", NameLocal: "ຄຳ", Boundary: square(104, 15), LandCells: 10},
		{OSMID: 11, Kind: "province", KindName: "district", AdminLevel: 4, Country: "bn", Slug: "belait-brunei", NameEn: "Belait", Boundary: square(114, 4), LandCells: 10},
		{OSMID: 12, Kind: "province", KindName: "region", AdminLevel: 4, Country: "sg", Slug: "central-singapore", NameEn: "Central", Boundary: square(103, 1), LandCells: 10},
		{OSMID: 13, Kind: "province", KindName: "prefecture", AdminLevel: 4, Country: "zz", Slug: "zz-lang", NameEn: "Zed", NameLocal: "Z", Boundary: square(120, 20), LandCells: 10},
	}
	if _, err := d.UpsertRegions(ctx, rs); err != nil {
		t.Fatal(err)
	}
	if err := d.ReloadRegions(ctx); err != nil {
		t.Fatal(err)
	}
	v := d.Views()

	for _, tc := range []struct{ slug, lang, kind string }{
		{"chon-buri", "th", "Province"},
		{"vientiane-laos", "lo", "Province"},
		{"belait-brunei", "ms", "District"},
		{"central-singapore", "en", "Region"},
		{"zz-lang", "", "Prefecture"},
	} {
		p, err := v.Region(ctx, tc.slug, 1)
		if err != nil {
			t.Fatal(tc.slug, err)
		}
		if p.Region.Lang != tc.lang || p.KindName() != tc.kind {
			t.Errorf("%s: lang %q kind %q, want %q %q", tc.slug, p.Region.Lang, p.KindName(), tc.lang, tc.kind)
		}
	}

	// A list shows several countries' provinces, each with its own language.
	want := map[string]string{"chon-buri": "th", "vientiane-laos": "lo", "belait-brunei": "ms", "central-singapore": "en", "zz-lang": ""}
	list, err := v.Regions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, r := range list.Regions {
		listed[r.Region.Slug] = true
		if r.Region.Lang != want[r.Region.Slug] {
			t.Errorf("regions list: %s has lang %q, want %q", r.Region.Slug, r.Region.Lang, want[r.Region.Slug])
		}
	}
	for slug := range want {
		if !listed[slug] {
			t.Errorf("regions list lacks %s", slug)
		}
	}
	refs, err := v.RegionRefs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range refs {
		if r.Lang != want[r.Slug] {
			t.Errorf("RegionRefs: %s has lang %q, want %q", r.Slug, r.Lang, want[r.Slug])
		}
	}

	// A blank plot, then the same plot surveyed with its photo list.
	for _, tc := range []struct {
		lat, lng float64
		lang     string
	}{{15.5, 104.5, "lo"}, {13.0, 101.0, "th"}} {
		cell, err := h3.CellFromLatLng(tc.lat, tc.lng, h3.PlotRes)
		if err != nil {
			t.Fatal(err)
		}
		blank, err := v.Plot(ctx, cell)
		if err != nil {
			t.Fatal(err)
		}
		if blank.Plot.LocalLang() != tc.lang {
			t.Errorf("blank plot %s: lang %q, want %q", cell, blank.Plot.LocalLang(), tc.lang)
		}
		n := newPhoto(cell, "ada", time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC))
		n.Lat, n.Lng = tc.lat, tc.lng
		if _, err := d.ClaimPhoto(ctx, n); err != nil {
			t.Fatal(err)
		}
		plot, err := v.Plot(ctx, cell)
		if err != nil {
			t.Fatal(err)
		}
		if !plot.Surveyed || plot.Plot.LocalLang() != tc.lang || len(plot.Photos) != 1 || plot.Photos[0].Plot.LocalLang() != tc.lang {
			t.Errorf("plot %s: surveyed %v lang %q photos %+v, want %q", cell, plot.Surveyed, plot.Plot.LocalLang(), plot.Photos, tc.lang)
		}
	}

	open, err := d.OpenCountries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var codes []string
	for _, c := range open {
		codes = append(codes, c.Code)
	}
	if len(codes) != 4 || codes[0] != "bn" || codes[1] != "la" || codes[2] != "sg" || codes[3] != "th" {
		t.Errorf("open countries %v, want bn la sg th (zz is not in the list)", codes)
	}
}
