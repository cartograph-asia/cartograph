package viewstest

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cartograph-asia/cartograph/internal/views"
)

func TestFixturesDeterministic(t *testing.T) {
	if !reflect.DeepEqual(Home(), Home()) || !reflect.DeepEqual(Region(), Region()) {
		t.Fatal("fixtures differ between calls")
	}
}

func TestHome(t *testing.T) {
	h := Home()
	if h.Counters.Photos != len(shots) || h.Counters.Plots != 13 || h.Counters.Surveyors != 5 || h.Counters.Regions != 2 {
		t.Fatalf("counters %+v", h.Counters)
	}
	if len(h.Latest) != views.HomeLatest || !h.Latest[0].UploadedAt.After(h.Latest[1].UploadedAt) {
		t.Fatal("latest not newest first")
	}
	if h.Featured == nil || !h.Featured.Editorial || h.Featured.Photo.ID != FeaturedPhotoID {
		t.Fatalf("featured %+v", h.Featured)
	}
	if len(h.Regions) != 2 || h.Regions[0].Region != ChonBuri || len(h.Regions[0].TopSurveyors) != views.TopSurveyors {
		t.Fatalf("regions %+v", h.Regions)
	}
	if len(h.NewSurveyors) != 5 || h.NewSurveyors[0].Surveyor.Name != "ploy88" {
		t.Fatalf("new surveyors %+v", h.NewSurveyors)
	}
	for _, ph := range h.Latest {
		if ph.Plot.Place == "" || ph.Plot.Region.Slug == "" || ph.Keys.Thumb == "" || strings.Contains(ph.Keys.Thumb, "://") {
			t.Fatalf("incomplete item %+v", ph)
		}
	}
}

func TestPagesAndErrors(t *testing.T) {
	f := &Fake{PerPage: 10, d: build()}
	ctx := context.Background()
	l, err := f.Latest(ctx, 3)
	if err != nil || len(l.Photos) != 10 || l.Pagination.HasNext() || l.Canonical != "/latest?page=3" {
		t.Fatalf("latest p3: %+v %v", l, err)
	}
	if _, err := f.Latest(ctx, 4); !errors.Is(err, views.ErrNotFound) {
		t.Fatalf("latest p4: %v", err)
	}
	s, err := f.Surveyor(ctx, "ADRIAN", 0)
	if err != nil || s.Profile.Name != "adrian" || s.Stats.Plots == 0 || s.Bounds == nil {
		t.Fatalf("surveyor: %+v %v", s, err)
	}
	for _, err := range []error{
		second(f.Surveyor(ctx, "nobody", 1)), second(f.Plot(ctx, "not-a-cell")),
		second(f.Photo(ctx, "nope")), second(f.Region(ctx, "atlantis", 1)),
	} {
		if !errors.Is(err, views.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	}
}

func second[T any](_ T, err error) error { return err }

func TestPlotAndPhoto(t *testing.T) {
	p := Plot()
	if !p.Surveyed || p.FirstSurveyor.Name != "beach-walker" || len(p.Neighbours) != 6 || !p.Photos[0].FirstSurvey {
		t.Fatalf("plot %+v", p)
	}
	if b := BlankPlot(); b.Surveyed || !b.NoIndex || b.FirstSurveyor != nil {
		t.Fatalf("blank plot %+v", b)
	}
	ph := Photo()
	if ph.Prev == nil || ph.Next == nil || ph.Position != 3 || ph.Count != p.PhotoCount || ph.OG.ImageKey != ph.Photo.Keys.Medium {
		t.Fatalf("photo %+v", ph)
	}
	if pp := PrototypePhoto(); pp.Prev != nil || pp.OG.ImageKey != pp.Photo.Keys.Large || !pp.Photo.FirstSurvey {
		t.Fatalf("prototype photo %+v", pp)
	}
}
