package views

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPagination(t *testing.T) {
	cases := []struct {
		p              Pagination
		pages          int
		hasPrev, hasNx bool
	}{
		{Pagination{Page: 1, PerPage: 48, Total: 0}, 1, false, false},
		{Pagination{Page: 1, PerPage: 48, Total: 48}, 1, false, false},
		{Pagination{Page: 1, PerPage: 48, Total: 49}, 2, false, true},
		{Pagination{Page: 2, PerPage: 48, Total: 49}, 2, true, false},
		{Pagination{Page: 2, PerPage: 10, Total: 35}, 4, true, true},
	}
	for _, c := range cases {
		if c.p.Pages() != c.pages || c.p.HasPrev() != c.hasPrev || c.p.HasNext() != c.hasNx {
			t.Errorf("%+v: pages=%d prev=%v next=%v", c.p, c.p.Pages(), c.p.HasPrev(), c.p.HasNext())
		}
	}
}

func TestCoveragePercent(t *testing.T) {
	if p := (Coverage{Claimed: 1, Land: 0}).Percent(); p != 0 {
		t.Fatal(p)
	}
	if p := (Coverage{Claimed: 30, Land: 120}).Percent(); p != 25 {
		t.Fatal(p)
	}
}

func TestViewerContext(t *testing.T) {
	ctx := context.Background()
	if ViewerFrom(ctx) != nil {
		t.Fatal("expected nil viewer")
	}
	v := &Viewer{SurveyorID: "x", Name: "adrian", Email: "a@example.org"}
	if got := ViewerFrom(WithViewer(ctx, v)); got != v || !got.HasName() {
		t.Fatalf("got %+v", got)
	}
	var none *Viewer
	if none.HasName() || (&Viewer{Email: "a@example.org"}).HasName() {
		t.Fatal("HasName without a name")
	}
}

func TestBase(t *testing.T) {
	var d any = &Home{}
	b, ok := d.(Based)
	if !ok {
		t.Fatal("Home does not implement Based")
	}
	b.Base().Title = "x"
	if d.(*Home).Title != "x" {
		t.Fatal("Base does not point into the page")
	}
}

func TestImageKeysBest(t *testing.T) {
	if (ImageKeys{Large: "l"}).Best() != "l" || (ImageKeys{Medium: "m", Large: "l"}).Best() != "m" {
		t.Fatal("Best")
	}
}

func TestCSRFContext(t *testing.T) {
	ctx := context.Background()
	if CSRFFrom(ctx) != "" {
		t.Fatal("expected empty token")
	}
	if got := CSRFFrom(WithCSRF(ctx, "tok")); got != "tok" {
		t.Fatalf("got %q", got)
	}
}

func TestPlotRefName(t *testing.T) {
	cb := RegionRef{Slug: "chon-buri", NameEn: "Chon Buri"}
	for _, c := range []struct {
		p          PlotRef
		name, path string
	}{
		{PlotRef{Cell: "88658bd11dfffff", Place: "Pattaya", Region: cb, Number: 17}, "Pattaya · Chon Buri 17", "/plot/chon-buri/17"},
		{PlotRef{Cell: "88658bd11dfffff", Place: "Chon Buri", Region: cb, Number: 17}, "Chon Buri 17", "/plot/chon-buri/17"},
		{PlotRef{Cell: "88658bd11dfffff", Place: "Pattaya", Region: cb}, "Pattaya · Chon Buri", "/plot/88658bd11dfffff"},
		{PlotRef{Cell: "88658bd11dfffff", Place: "Pattaya"}, "Pattaya", "/plot/88658bd11dfffff"},
		{PlotRef{Cell: "88658bd11dfffff"}, "A plot", "/plot/88658bd11dfffff"},
		{PlotRef{Cell: "88658bd11dfffff", Place: "Hua Mak", District: "Bang Kapi", Region: RegionRef{Slug: "bangkok", NameEn: "Bangkok"}, Number: 1000}, "Hua Mak, Bang Kapi · Bangkok 1000", "/plot/bangkok/1000"},
		{PlotRef{Cell: "88658bd11dfffff", Place: "Bang Kapi", District: "Bang Kapi", Region: RegionRef{Slug: "bangkok", NameEn: "Bangkok"}, Number: 7}, "Bang Kapi · Bangkok 7", "/plot/bangkok/7"},
		{PlotRef{Cell: "88658bd11dfffff", Place: "Chon Buri", District: "Bang Lamung", Region: cb, Number: 17}, "Bang Lamung · Chon Buri 17", "/plot/chon-buri/17"},
	} {
		if got := c.p.Name(); got != c.name {
			t.Errorf("Name(%+v) = %q, want %q", c.p, got, c.name)
		}
		if got := c.p.Path(); got != c.path {
			t.Errorf("Path(%+v) = %q, want %q", c.p, got, c.path)
		}
	}
}

func TestPlotRefShortName(t *testing.T) {
	cb := RegionRef{Slug: "chon-buri", NameEn: "Chon Buri"}
	bkk := RegionRef{Slug: "bangkok", NameEn: "Bangkok"}
	for _, c := range []struct {
		p           PlotRef
		short       string
		place, prov string
	}{
		{PlotRef{Place: "Pattaya", District: "Bang Lamung", Region: cb, Number: 2412}, "Pattaya · Chon Buri 2412", "Pattaya", "Chon Buri 2412"},
		{PlotRef{Place: "Hua Mak", District: "Bang Kapi", Region: bkk, Number: 1000}, "Hua Mak · Bangkok 1000", "Hua Mak", "Bangkok 1000"},
		{PlotRef{Place: "Bangkok", District: "Bang Kapi", Region: bkk, Number: 1000}, "Bang Kapi · Bangkok 1000", "Bang Kapi", "Bangkok 1000"},
		{PlotRef{Place: "Chon Buri", Region: cb, Number: 17}, "Chon Buri 17", "", "Chon Buri 17"},
		{PlotRef{Place: "Pattaya"}, "Pattaya", "Pattaya", ""},
		{PlotRef{}, "A plot", "", ""},
	} {
		if got := c.p.ShortName(); got != c.short {
			t.Errorf("ShortName(%+v) = %q, want %q", c.p, got, c.short)
		}
		if n := c.p.ShortParts(); n.Place != c.place || n.Province != c.prov {
			t.Errorf("ShortParts(%+v) = %+v", c.p, n)
		}
	}
}

func TestRegionKindName(t *testing.T) {
	for kind, want := range map[string]string{"province": "Province", "country": "Country", "": "Region"} {
		if got := (Region{Kind: kind}).KindName(); got != want {
			t.Errorf("KindName(%q) = %q, want %q", kind, got, want)
		}
	}
}

func TestCameraLine(t *testing.T) {
	for _, c := range []struct {
		c    Camera
		want string
	}{
		{Camera{Label: "Apple iPhone 17 Pro", FocalMM: 6.8, Focal35MM: 48, FNumber: 1.8, ExposureS: 1.0 / 16129, ISO: 64},
			"Apple iPhone 17 Pro · 6.8 mm (48 mm equiv.) · f/1.8 · 1/16129 s · ISO 64"},
		{Camera{Label: "Canon EOS R5", FocalMM: 35, FNumber: 2, ExposureS: 1.0 / 250, ISO: 400},
			"Canon EOS R5 · 35 mm · f/2 · 1/250 s · ISO 400"},
		{Camera{Label: "SONY ILCE-7M4", ExposureS: 2, FNumber: 8}, "SONY ILCE-7M4 · f/8 · 2 s"},
		{Camera{Label: "Nikon D850", ExposureS: 0.4}, "Nikon D850 · 0.4 s"},
		{Camera{Label: "Pixel 9", Focal35MM: 24}, "Pixel 9 · 24 mm equiv."},
		{Camera{Label: "Unknown camera", Lens: "ignored on the line"}, "Unknown camera"},
		{Camera{Label: "Odd file", ExposureS: 1e-300, ISO: 100}, "Odd file · ISO 100"},
	} {
		if got := c.c.Line(); got != c.want {
			t.Errorf("Line() = %q, want %q", got, c.want)
		}
	}
}

func TestSortCountries(t *testing.T) {
	old, recent := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	cs := []CountrySummary{
		{Country: CountryRef{NameEn: "Brunei"}},
		{Country: CountryRef{NameEn: "Vietnam"}, Open: true, LastSurveyedAt: &old},
		{Country: CountryRef{NameEn: "Cambodia"}, Open: true},
		{Country: CountryRef{NameEn: "Thailand"}, Open: true, LastSurveyedAt: &recent},
		{Country: CountryRef{NameEn: "Laos"}, Open: true},
	}
	SortCountries(cs)
	var got []string
	for _, c := range cs {
		got = append(got, c.Country.NameEn)
	}
	if want := "Thailand Vietnam Cambodia Laos Brunei"; strings.Join(got, " ") != want {
		t.Errorf("order %v, want %s", got, want)
	}
}
