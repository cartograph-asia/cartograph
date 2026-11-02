package db_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/views"
)

// plotRow is a plot's claim columns as stored.
type plotRow struct {
	first      *uuid.UUID
	surveyor   *string
	surveyorID *uuid.UUID
	claimed    *time.Time
	count      int
}

func readPlot(t *testing.T, d *db.DB, cell string) plotRow {
	t.Helper()
	var p plotRow
	err := d.Pool.QueryRow(context.Background(), `
		select first_photo, first_surveyor, first_surveyor_id, claimed_at, photo_count from plots where cell = $1`, cell,
	).Scan(&p.first, &p.surveyor, &p.surveyorID, &p.claimed, &p.count)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (p plotRow) equal(o plotRow) bool {
	eqU := func(a, b *uuid.UUID) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	eqT := func(a, b *time.Time) bool { return (a == nil) == (b == nil) && (a == nil || a.Equal(*b)) }
	eqS := func(a, b *string) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	return eqU(p.first, o.first) && eqS(p.surveyor, o.surveyor) && eqU(p.surveyorID, o.surveyorID) && eqT(p.claimed, o.claimed) && p.count == o.count
}

type regionRow struct {
	claimed int
	last    *time.Time
}

func readRegion(t *testing.T, d *db.DB) regionRow {
	t.Helper()
	var r regionRow
	if err := d.Pool.QueryRow(context.Background(), `select claimed_cells, last_surveyed_at from regions where kind = 'province'`).
		Scan(&r.claimed, &r.last); err != nil {
		t.Fatal(err)
	}
	return r
}

func cellOf(t *testing.T, ll [2]float64) string {
	t.Helper()
	c, err := h3.CellFromLatLng(ll[0], ll[1], h3.PlotRes)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestHideAndUnhide(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	a := w.d.Admin()
	v := w.d.Views().WithNow(func() time.Time { return t0 })
	admin := w.surveyors["adrian"].String()
	beachCell, nongPruCell := cellOf(t, beach), cellOf(t, nongPru)
	beachBefore, nongPruBefore, regionBefore := readPlot(t, w.d, beachCell), readPlot(t, w.d, nongPruCell), readRegion(t, w.d)

	// A photo that does not hold the claim: only the count changes.
	if err := a.HidePhoto(ctx, admin, w.photos[4].String(), "duplicate", t0); err != nil {
		t.Fatal(err)
	}
	if p := readPlot(t, w.d, beachCell); p.count != 2 || *p.first != w.photos[0] {
		t.Fatalf("after hiding a non-claiming photo: %+v", p)
	}
	// The claiming photo with another left: the claim passes on.
	e, err := a.HideEffect(ctx, w.photos[0].String(), t0)
	if err != nil || !e.Claims || e.PassesTo != "Somchai-K" {
		t.Fatalf("effect %+v %v", e, err)
	}
	if err := a.HidePhoto(ctx, admin, w.photos[0].String(), "someone else's photo", t0); err != nil {
		t.Fatal(err)
	}
	p := readPlot(t, w.d, beachCell)
	if p.count != 1 || *p.first != w.photos[2] || *p.surveyor != "Somchai-K" || *p.surveyorID != w.surveyors["Somchai-K"] ||
		!p.claimed.Equal(t0.Add(-3*24*time.Hour)) {
		t.Fatalf("claim did not pass on: %+v", p)
	}
	pg, err := v.Plot(ctx, beachCell)
	if err != nil || pg.FirstSurveyor.Name != "Somchai-K" || len(pg.Photos) != 1 || !pg.Photos[0].FirstSurvey {
		t.Fatalf("plot page %+v %v", pg, err)
	}
	// The only photo: the plot is blank again and coverage drops.
	e, err = a.HideEffect(ctx, w.photos[3].String(), t0)
	if err != nil || !e.Claims || e.PassesTo != "" {
		t.Fatalf("effect of the only photo %+v %v", e, err)
	}
	if err := a.HidePhoto(ctx, admin, w.photos[3].String(), "a face", t0); err != nil {
		t.Fatal(err)
	}
	if p := readPlot(t, w.d, nongPruCell); p.count != 0 || p.first != nil || p.surveyor != nil || p.surveyorID != nil || p.claimed != nil {
		t.Fatalf("blank plot %+v", p)
	}
	if r := readRegion(t, w.d); r.claimed != regionBefore.claimed-1 {
		t.Fatalf("region %+v, before %+v", r, regionBefore)
	}

	// Gone from every public read.
	h, err := v.Home(ctx)
	if err != nil || h.Counters.Plots != 3 || h.Counters.Photos != 3 || len(h.Latest) != 3 {
		t.Fatalf("home %+v %v", h.Counters, err)
	}
	for _, it := range h.Latest {
		if it.ID == w.photos[0].String() || it.ID == w.photos[3].String() || it.ID == w.photos[4].String() {
			t.Fatalf("hidden photo %s on the front page", it.ID)
		}
	}
	if blank, err := v.Plot(ctx, nongPruCell); err != nil || blank.Surveyed || len(blank.Photos) != 0 {
		t.Fatalf("blank plot page %+v %v", blank, err)
	}
	if _, err := v.Photo(ctx, w.photos[3].String()); !errors.Is(err, views.ErrGone) {
		t.Fatalf("hidden photo page: %v", err)
	}
	if _, err := v.Photo(ctx, uuid.NewString()); !errors.Is(err, views.ErrNotFound) {
		t.Fatalf("unknown photo page: %v", err)
	}
	ph, err := v.Photo(ctx, w.photos[2].String())
	if err != nil || ph.Count != 1 || ph.Prev != nil || ph.Next != nil {
		t.Fatalf("sibling photo %+v %v", ph, err)
	}
	if f, err := v.LatestFeed(ctx); err != nil || len(f.Items) != 3 {
		t.Fatalf("feed %d %v", len(f.Items), err)
	}
	if s, err := v.Surveyor(ctx, "adrian", views.SurveyorQuery{Page: 1}); err != nil || s.Stats.Photos != 2 || s.Stats.Plots != 2 || len(s.Photos) != 2 {
		t.Fatalf("surveyor %+v %v", s.Stats, err)
	}
	if list, err := w.d.Plots(ctx, []string{beachCell, nongPruCell}); err != nil || len(list) != 1 || list[0].Cell != beachCell {
		t.Fatalf("api plots %+v %v", list, err)
	}
	if _, err := w.d.Plot(ctx, nongPruCell); !errors.Is(err, db.ErrNotFound) {
		t.Fatalf("api blank plot: %v", err)
	}
	if bp, err := w.d.Plot(ctx, beachCell); err != nil || len(bp.Photos) != 1 || bp.FirstSurveyor != "Somchai-K" {
		t.Fatalf("api plot %+v %v", bp, err)
	}
	p6, _ := h3.Parent(nongPruCell, 6)
	if cov, err := w.d.Coverage(ctx, 6, []string{p6}); err != nil || cov[p6] != 0 {
		t.Fatalf("coverage %v %v", cov, err)
	}

	// Twice, unknown, without a reason.
	if err := a.HidePhoto(ctx, admin, w.photos[3].String(), "again", t0); !errors.Is(err, db.ErrUnchanged) {
		t.Fatalf("hide twice: %v", err)
	}
	if err := a.HidePhoto(ctx, admin, uuid.NewString(), "x", t0); !errors.Is(err, views.ErrNotFound) {
		t.Fatalf("hide unknown: %v", err)
	}
	if err := a.HidePhoto(ctx, admin, w.photos[1].String(), "  ", t0); !errors.Is(err, db.ErrInvalid) {
		t.Fatalf("hide without reason: %v", err)
	}
	if err := a.UnhidePhoto(ctx, admin, w.photos[1].String()); !errors.Is(err, db.ErrUnchanged) {
		t.Fatalf("unhide a visible photo: %v", err)
	}

	// Unhiding is the exact inverse.
	for _, i := range []int{3, 0, 4} {
		if err := a.UnhidePhoto(ctx, admin, w.photos[i].String()); err != nil {
			t.Fatal(err)
		}
	}
	if p := readPlot(t, w.d, beachCell); !p.equal(beachBefore) {
		t.Fatalf("beach after unhide %+v, before %+v", p, beachBefore)
	}
	if p := readPlot(t, w.d, nongPruCell); !p.equal(nongPruBefore) {
		t.Fatalf("nong prue after unhide %+v, before %+v", p, nongPruBefore)
	}
	if r := readRegion(t, w.d); r.claimed != regionBefore.claimed || !r.last.Equal(*regionBefore.last) {
		t.Fatalf("region after unhide %+v, before %+v", r, regionBefore)
	}

	// Every action is in the log, and the log cannot be changed.
	lg, err := a.Log(ctx, 1)
	if err != nil || lg.Pagination.Total != 6 || lg.Actions[0].Action != "photo.unhide" || lg.Actions[0].Admin != "adrian" {
		t.Fatalf("log %+v %v", lg, err)
	}
	if _, err := w.d.Pool.Exec(ctx, `update admin_actions set reason = 'x'`); err == nil {
		t.Fatal("admin_actions can be updated")
	}
	if _, err := w.d.Pool.Exec(ctx, `delete from admin_actions`); err == nil {
		t.Fatal("admin_actions can be deleted")
	}
}

func TestBlankPlotIsClaimedAgain(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	cell := cellOf(t, nongPru)
	if err := w.d.Admin().HidePhoto(ctx, w.surveyors["adrian"].String(), w.photos[3].String(), "test", t0); err != nil {
		t.Fatal(err)
	}
	before := readRegion(t, w.d)
	got, claimed := w.claim(t, "adrian", nongPru[0], nongPru[1], t0.Add(time.Minute))
	if got != cell || !claimed {
		t.Fatalf("claim of a blanked plot: %s %v", got, claimed)
	}
	p := readPlot(t, w.d, cell)
	if p.count != 1 || *p.first != w.photos[len(w.photos)-1] || *p.surveyor != "adrian" || !p.claimed.Equal(t0.Add(time.Minute)) {
		t.Fatalf("plot %+v", p)
	}
	if r := readRegion(t, w.d); r.claimed != before.claimed+1 {
		t.Fatalf("region %+v, before %+v", r, before)
	}
	// Unhiding the older photo takes the claim back.
	if err := w.d.Admin().UnhidePhoto(ctx, w.surveyors["adrian"].String(), w.photos[3].String()); err != nil {
		t.Fatal(err)
	}
	if p := readPlot(t, w.d, cell); p.count != 2 || *p.first != w.photos[3] || *p.surveyor != "Somchai-K" {
		t.Fatalf("after unhide %+v", p)
	}
}

func TestBlockSurveyor(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	a := w.d.Admin()
	admin := w.surveyors["adrian"].String()
	som := w.surveyors["Somchai-K"]
	_, err := w.d.Pool.Exec(ctx, `insert into sessions (token_hash, email, surveyor_id, expires_at) values ($1, 'somchai-k@example.test', $2, $3)`,
		make32(1), som, t0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	// Hidden on its own before the block: stays hidden after the unblock.
	if err := a.HidePhoto(ctx, admin, w.photos[2].String(), "blurred", t0.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := a.BlockSurveyor(ctx, admin, "adrian", "test", false, t0); !errors.Is(err, db.ErrInvalid) {
		t.Fatalf("self-block: %v", err)
	}
	if err := a.BlockSurveyor(ctx, admin, "somchai-k", "spam", true, t0); err != nil {
		t.Fatal(err)
	}
	if err := a.BlockSurveyor(ctx, admin, "Somchai-K", "spam", true, t0); !errors.Is(err, db.ErrUnchanged) {
		t.Fatalf("block twice: %v", err)
	}
	var sessions, visible int
	if err := w.d.Pool.QueryRow(ctx, `select (select count(*) from sessions where surveyor_id = $1),
		(select count(*) from visible_photos where surveyor_id = $1)`, som).Scan(&sessions, &visible); err != nil {
		t.Fatal(err)
	}
	if sessions != 0 || visible != 0 {
		t.Fatalf("after block: %d sessions, %d visible photos", sessions, visible)
	}
	s, err := a.Surveyor(ctx, "Somchai-K", t0)
	if err != nil || s.Surveyor.BlockedAt == nil || s.BlockedReason != "spam" || s.HiddenByBlock != 1 || s.Surveyor.Hidden != 2 {
		t.Fatalf("admin surveyor %+v %v", s, err)
	}
	// Nong Prue had only their photo.
	if p := readPlot(t, w.d, cellOf(t, nongPru)); p.count != 0 {
		t.Fatalf("nong prue %+v", p)
	}
	if err := a.UnblockSurveyor(ctx, admin, "Somchai-K", true); err != nil {
		t.Fatal(err)
	}
	if err := w.d.Pool.QueryRow(ctx, `select count(*) from visible_photos where surveyor_id = $1`, som).Scan(&visible); err != nil || visible != 1 {
		t.Fatalf("after unblock: %d visible %v", visible, err)
	}
	if err := a.UnblockSurveyor(ctx, admin, "Somchai-K", true); !errors.Is(err, db.ErrUnchanged) {
		t.Fatalf("unblock twice: %v", err)
	}
	if err := a.EndSessions(ctx, admin, "nobody-here"); !errors.Is(err, views.ErrNotFound) {
		t.Fatalf("end sessions of nobody: %v", err)
	}
	if _, err := w.d.Pool.Exec(ctx, `update surveyors set bio = 'Buy cheap watches' where id = $1`, som); err != nil {
		t.Fatal(err)
	}
	if err := a.ClearBio(ctx, admin, "Somchai-K", "spam"); err != nil {
		t.Fatal(err)
	}
	if err := a.ClearBio(ctx, admin, "Somchai-K", "spam"); !errors.Is(err, db.ErrUnchanged) {
		t.Fatalf("clear an empty bio: %v", err)
	}
	s, err = a.Surveyor(ctx, "Somchai-K", t0)
	if err != nil || s.Bio != "" || len(s.Actions) == 0 || s.Actions[0].Action != "surveyor.clear_bio" || !strings.Contains(s.Actions[0].Before, "watches") {
		t.Fatalf("after clear bio %+v %v", s.Actions, err)
	}
}

func make32(b byte) []byte {
	out := make([]byte, 32)
	out[0] = b
	return out
}

func TestFeaturedAdmin(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	a := w.d.Admin()
	admin := w.surveyors["adrian"].String()
	v := w.d.Views().WithNow(func() time.Time { return t0 })

	id, err := a.CreateFeatured(ctx, admin, w.photos[2].String(), "Second Road at six.", t0.Add(-time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	next, err := a.CreateFeatured(ctx, admin, w.photos[1].String(), "", t0.Add(24*time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.CreateFeatured(ctx, admin, w.photos[1].String(), "", t0, ptr(t0)); !errors.Is(err, db.ErrInvalid) {
		t.Fatalf("end before start: %v", err)
	}
	if _, err := a.CreateFeatured(ctx, admin, w.photos[1].String(), strings.Repeat("x", views.FeaturedTextMax+1), t0, nil); !errors.Is(err, db.ErrInvalid) {
		t.Fatalf("long text: %v", err)
	}
	f, err := a.Featured(ctx, t0)
	if err != nil || f.Current == nil || f.Current.ID != id || len(f.Scheduled) != 1 || f.Scheduled[0].ID != next {
		t.Fatalf("featured %+v %v", f, err)
	}
	if h, err := v.Home(ctx); err != nil || !h.Featured.Editorial || h.Featured.Photo.ID != w.photos[2].String() {
		t.Fatalf("home featured %+v %v", h.Featured, err)
	}
	if err := a.DeleteFeatured(ctx, admin, id, t0); !errors.Is(err, db.ErrUnchanged) {
		t.Fatalf("delete a started entry: %v", err)
	}
	if err := a.DeleteFeatured(ctx, admin, next, t0); err != nil {
		t.Fatal(err)
	}
	if err := a.EndFeatured(ctx, admin, id, t0); err != nil {
		t.Fatal(err)
	}
	if h, err := v.Home(ctx); err != nil || h.Featured.Editorial {
		t.Fatalf("home after end %+v %v", h.Featured, err)
	}
	if err := a.EndFeatured(ctx, admin, id, t0); !errors.Is(err, db.ErrUnchanged) {
		t.Fatalf("end twice: %v", err)
	}
	if err := a.HidePhoto(ctx, admin, w.photos[1].String(), "test", t0); err != nil {
		t.Fatal(err)
	}
	if _, err := a.CreateFeatured(ctx, admin, w.photos[1].String(), "", t0, nil); !errors.Is(err, db.ErrInvalid) {
		t.Fatalf("feature a hidden photo: %v", err)
	}
	if f, err := a.Featured(ctx, t0); err != nil || f.Current != nil || len(f.Past) != 1 {
		t.Fatalf("featured after %+v %v", f, err)
	}
}

func ptr(t time.Time) *time.Time { return &t }

func TestPlaceOverrideReplace(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	a := w.d.Admin()
	admin := w.surveyors["adrian"].String()
	walkingCell := cellOf(t, walking)
	near := [2]float64{walking[0] + 0.004, walking[1]}
	loc, err := w.d.LocatePlot(ctx, cellOf(t, near))
	if err != nil || !loc.Near {
		t.Fatalf("locate %+v %v", loc, err)
	}
	var stored string
	if err := w.d.Pool.QueryRow(ctx, `select place_name from plots where cell = $1`, walkingCell).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	o := views.PlaceOverride{Country: "th", Name: loc.PlaceEn, Kind: "town", Action: views.OverrideUndecided, Note: "test"}
	if err := a.CreatePlaceOverride(ctx, admin, o); err != nil {
		t.Fatal(err)
	}
	if err := a.CreatePlaceOverride(ctx, admin, o); !errors.Is(err, db.ErrInvalid) {
		t.Fatalf("listed twice: %v", err)
	}
	o.Action = views.OverrideReplace
	if err := a.UpdatePlaceOverride(ctx, admin, o); !errors.Is(err, db.ErrInvalid) {
		t.Fatalf("replace without replacement: %v", err)
	}
	o.Replacement = "Walking Street"
	if err := a.UpdatePlaceOverride(ctx, admin, o); err != nil {
		t.Fatal(err)
	}
	if err := a.UpdatePlaceOverride(ctx, admin, o); !errors.Is(err, db.ErrUnchanged) {
		t.Fatalf("save unchanged: %v", err)
	}
	cell, _ := w.claim(t, "adrian", near[0], near[1], t0)
	var name string
	var th *string
	if err := w.d.Pool.QueryRow(ctx, `select place_name, place_name_local from plots where cell = $1`, cell).Scan(&name, &th); err != nil {
		t.Fatal(err)
	}
	if name != "Walking Street" || th != nil {
		t.Fatalf("new plot named %q %v", name, th)
	}
	var again string
	if err := w.d.Pool.QueryRow(ctx, `select place_name from plots where cell = $1`, walkingCell).Scan(&again); err != nil || again != stored {
		t.Fatalf("existing plot renamed: %q -> %q %v", stored, again, err)
	}
	ps, err := a.Places(ctx)
	// Every opened country has a place_naming row (0075 Thailand, 0152 Laos, …).
	if err != nil || !slices.Contains(ps.Countries, "th") {
		t.Fatalf("places %+v %v", ps, err)
	}
}

func TestAdminReads(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	a := w.d.Admin()
	if err := a.HidePhoto(ctx, w.surveyors["adrian"].String(), w.photos[3].String(), "test", t0); err != nil {
		t.Fatal(err)
	}
	o, err := a.Overview(ctx, t0)
	if err != nil || o.Counters.Surveyors != 3 || o.Counters.Hidden != 1 || o.Counters.Plots != 3 || len(o.Uploads) != 6 || len(o.Actions) != 1 {
		t.Fatalf("overview %+v %v", o, err)
	}
	ps, err := a.Photos(ctx, 1, true, "")
	if err != nil || len(ps.Photos) != 1 || !ps.Photos[0].Hidden() || ps.Photos[0].HiddenReason != "test" {
		t.Fatalf("hidden photos %+v %v", ps, err)
	}
	if ps, err := a.Photos(ctx, 1, false, "SOMCHAI-K"); err != nil || len(ps.Photos) != 2 {
		t.Fatalf("photos by surveyor %+v %v", ps, err)
	}
	d, err := a.Photo(ctx, w.photos[3].String())
	if err != nil || !d.Photo.Hidden() || d.PlotPhotos != 0 || d.FirstSurveyor != "" || d.Email != "somchai-k@example.test" || len(d.Actions) != 1 {
		t.Fatalf("admin photo %+v %v", d, err)
	}
	ss, err := a.Surveyors(ctx, 1, t0)
	if err != nil || len(ss.Surveyors) != 3 || ss.Surveyors[0].Name != "no-photos" {
		t.Fatalf("surveyors %+v %v", ss, err)
	}
}

// TestPublicReadsUseVisiblePhotos fails when a query outside admin.go reads
// the photos table instead of visible_photos.
func TestPublicReadsUseVisiblePhotos(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?i)\b(from|join)\s+photos\b`)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "admin.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			// The one exception: telling a hidden photo (410) from an unknown one (404).
			if re.MatchString(line) && !strings.Contains(line, "hidden_at is not null") {
				t.Errorf("%s:%d reads photos directly; public reads use visible_photos: %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
}

func TestRemovePhotoFiles(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	a := w.d.Admin()
	admin := w.surveyors["adrian"].String()
	id := w.photos[3].String()
	var deleted []string
	del := func(_ context.Context, key string) error { deleted = append(deleted, key); return nil }

	if err := a.RemovePhotoFiles(ctx, admin, id, "copyright", t0, del); !errors.Is(err, db.ErrInvalid) {
		t.Fatalf("visible photo: %v", err)
	}
	if err := a.HidePhoto(ctx, admin, id, "copyright", t0); err != nil {
		t.Fatal(err)
	}
	if err := a.RemovePhotoFiles(ctx, admin, id, "copyright", t0.Add(time.Hour), del); !errors.Is(err, db.ErrInvalid) || len(deleted) != 0 {
		t.Fatalf("too early: %v %v", err, deleted)
	}
	// A failing delete records nothing.
	boom := errors.New("bucket down")
	if err := a.RemovePhotoFiles(ctx, admin, id, "copyright", t0.Add(25*time.Hour), func(context.Context, string) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("failing delete: %v", err)
	}
	if err := a.RemovePhotoFiles(ctx, admin, id, "copyright", t0.Add(25*time.Hour), del); err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 3 || !strings.HasSuffix(deleted[0], "_large.jpg") {
		t.Fatalf("deleted %v", deleted)
	}
	if err := a.RemovePhotoFiles(ctx, admin, id, "again", t0.Add(26*time.Hour), del); !errors.Is(err, db.ErrUnchanged) {
		t.Fatalf("twice: %v", err)
	}
	if err := a.UnhidePhoto(ctx, admin, id); !errors.Is(err, db.ErrInvalid) {
		t.Fatalf("unhide after removal: %v", err)
	}
	d, err := a.Photo(ctx, id)
	if err != nil || d.Photo.FilesRemovedAt == nil || d.Actions[0].Action != "photo.remove_files" {
		t.Fatalf("admin photo %+v %v", d, err)
	}
	// Public reads still treat it as removed.
	if _, err := w.d.Views().Photo(ctx, id); !errors.Is(err, views.ErrGone) {
		t.Fatalf("public page: %v", err)
	}
}
