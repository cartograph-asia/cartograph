package views

import (
	"context"
	"testing"
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
