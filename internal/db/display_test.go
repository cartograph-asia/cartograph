package db_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cartograph-asia/cartograph/internal/db"
)

func TestDisplayNamesInReads(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	v := w.d.Views().WithNow(func() time.Time { return t0 })
	if _, err := w.d.Pool.Exec(ctx, `update surveyors set display_name = 'Somchai Kittisak' where name = 'Somchai-K'`); err != nil {
		t.Fatal(err)
	}

	h, err := v.Home(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range h.Latest {
		if p.Surveyor.Name == "Somchai-K" && p.Surveyor.Display != "Somchai Kittisak" {
			t.Fatalf("latest item %+v", p.Surveyor)
		}
		if p.Surveyor.Name == "adrian" && p.Surveyor.Display != "" {
			t.Fatalf("adrian got a display name: %+v", p.Surveyor)
		}
	}
	if top := h.Regions[0].TopSurveyors; top[1].Surveyor.Display != "Somchai Kittisak" || top[1].Surveyor.Name != "Somchai-K" {
		t.Fatalf("top surveyors %+v", top)
	}
	if ns := h.NewSurveyors[0]; ns.Surveyor.Display != "Somchai Kittisak" {
		t.Fatalf("new surveyors %+v", ns)
	}
	s, err := v.Surveyor(ctx, "somchai-k", 1)
	if err != nil || s.Profile.Name != "Somchai-K" || s.Profile.Display != "Somchai Kittisak" || s.Title != "Somchai Kittisak" || s.Canonical != "/surveyor/Somchai-K" {
		t.Fatalf("profile %+v %v", s.Page, err)
	}
	sl, err := v.Surveyors(ctx, 1)
	if err != nil || sl.Surveyors[1].Surveyor.Display != "Somchai Kittisak" {
		t.Fatalf("surveyors %+v %v", sl.Surveyors, err)
	}
	pl, err := v.Plot(ctx, cellOf(t, nongPru))
	if err != nil || pl.FirstSurveyor.Label() != "Somchai Kittisak" || pl.FirstSurveyor.Name != "Somchai-K" {
		t.Fatalf("plot %+v %v", pl.FirstSurveyor, err)
	}
	f, err := v.SurveyorFeed(ctx, "Somchai-K")
	if err != nil || f.Title != "cartograph.asia: Somchai Kittisak" || f.Items[0].Surveyor.Display != "Somchai Kittisak" {
		t.Fatalf("feed %q %v", f.Title, err)
	}
	// The JSON API: handle for links, label for text.
	list, err := w.d.Plots(ctx, []string{cellOf(t, nongPru)})
	if err != nil || list[0].FirstSurveyor != "Somchai-K" || list[0].FirstLabel != "Somchai Kittisak" {
		t.Fatalf("api plots %+v %v", list, err)
	}
	bp, err := w.d.Plot(ctx, cellOf(t, beach))
	if err != nil || bp.FirstLabel != "adrian" || bp.Photos[1].Surveyor != "Somchai-K" || bp.Photos[1].Label != "Somchai Kittisak" {
		t.Fatalf("api plot %+v %v", bp, err)
	}

	// Admins can clear it; the old one is in the log.
	a := w.d.Admin()
	admin := w.surveyors["adrian"].String()
	if err := a.ClearDisplayName(ctx, admin, "Somchai-K", "impersonation"); err != nil {
		t.Fatal(err)
	}
	if err := a.ClearDisplayName(ctx, admin, "Somchai-K", "again"); !errors.Is(err, db.ErrUnchanged) {
		t.Fatalf("clear twice: %v", err)
	}
	as, err := a.Surveyor(ctx, "Somchai-K", t0)
	if err != nil || as.Surveyor.Display != "" || as.Actions[0].Action != "surveyor.clear_display_name" || as.Actions[0].Before != `{"display_name": "Somchai Kittisak"}` {
		t.Fatalf("after clear %+v %v", as.Actions, err)
	}
}
