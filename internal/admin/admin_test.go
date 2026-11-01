package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/storage"
	"github.com/cartograph-asia/cartograph/internal/views"
)

// fakeStore answers every read with an empty page and records writes.
type fakeStore struct {
	mu       sync.Mutex
	writes   []string
	err      error      // returned by the next write
	hiddenAt *time.Time // when set, Photo returns a photo hidden then
}

func (f *fakeStore) write(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes = append(f.writes, name)
	err := f.err
	f.err = nil
	return err
}

func photo() views.AdminPhoto {
	return views.AdminPhoto{PhotoItem: views.PhotoItem{ID: "3f2a0000-0000-4000-8000-000000000001", Surveyor: views.SurveyorRef{Name: "ploy88"}}}
}

func (f *fakeStore) Overview(context.Context, time.Time) (*views.AdminOverview, error) {
	return &views.AdminOverview{}, nil
}
func (f *fakeStore) Photos(_ context.Context, _ int, hidden bool, surveyor string) (*views.AdminPhotos, error) {
	return &views.AdminPhotos{HiddenOnly: hidden, Surveyor: surveyor}, nil
}
func (f *fakeStore) Photo(_ context.Context, id string) (*views.AdminPhotoDetail, error) {
	p := photo()
	p.HiddenAt = f.hiddenAt
	if p.HiddenAt != nil {
		p.HiddenReason = "test"
	}
	return &views.AdminPhotoDetail{Photo: p}, nil
}
func (f *fakeStore) HideEffect(context.Context, string, time.Time) (*views.HideEffect, error) {
	return &views.HideEffect{Claims: true, PassesTo: "mali"}, nil
}
func (f *fakeStore) HidePhoto(context.Context, string, string, string, time.Time) error {
	return f.write("hide")
}
func (f *fakeStore) UnhidePhoto(context.Context, string, string) error { return f.write("unhide") }
func (f *fakeStore) RemovePhotoFiles(_ context.Context, _, _, _ string, _ time.Time, del func(context.Context, string) error) error {
	if err := del(context.Background(), "photos/x_large.jpg"); err != nil {
		return err
	}
	return f.write("remove-files")
}
func (f *fakeStore) Featured(context.Context, time.Time) (*views.AdminFeaturedList, error) {
	return &views.AdminFeaturedList{}, nil
}
func (f *fakeStore) CreateFeatured(context.Context, string, string, string, time.Time, *time.Time) (int, error) {
	return 1, f.write("featured.create")
}
func (f *fakeStore) EndFeatured(context.Context, string, int, time.Time) error {
	return f.write("featured.end")
}
func (f *fakeStore) DeleteFeatured(context.Context, string, int, time.Time) error {
	return f.write("featured.delete")
}
func (f *fakeStore) Surveyors(context.Context, int, time.Time) (*views.AdminSurveyors, error) {
	return &views.AdminSurveyors{}, nil
}
func (f *fakeStore) Surveyor(_ context.Context, name string, _ time.Time) (*views.AdminSurveyor, error) {
	return &views.AdminSurveyor{Surveyor: views.AdminSurveyorItem{Name: name, Display: "Ploy", Photos: 3}, Bio: "spam"}, nil
}
func (f *fakeStore) BlockSurveyor(context.Context, string, string, string, bool, time.Time) error {
	return f.write("block")
}
func (f *fakeStore) UnblockSurveyor(context.Context, string, string, bool) error {
	return f.write("unblock")
}
func (f *fakeStore) EndSessions(context.Context, string, string) error {
	return f.write("end-sessions")
}
func (f *fakeStore) ClearBio(context.Context, string, string, string) error {
	return f.write("clear-bio")
}
func (f *fakeStore) ClearDisplayName(context.Context, string, string, string) error {
	return f.write("clear-display-name")
}
func (f *fakeStore) Places(context.Context) (*views.AdminPlaces, error) {
	return &views.AdminPlaces{}, nil
}
func (f *fakeStore) CreatePlaceOverride(context.Context, string, views.PlaceOverride) error {
	return f.write("place.create")
}
func (f *fakeStore) UpdatePlaceOverride(context.Context, string, views.PlaceOverride) error {
	return f.write("place.update")
}
func (f *fakeStore) Log(context.Context, int) (*views.AdminLog, error) { return &views.AdminLog{}, nil }

// fakeAccounts guards with the viewer's Admin flag and accepts a form when
// it carries csrf=ok, like auth's checks without the cookies.
type fakeAccounts struct {
	store   *fakeStore
	guarded int
}

func (a *fakeAccounts) AdminGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.guarded++
		if !views.ViewerFrom(r.Context()).IsAdmin() {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *fakeAccounts) CheckForm(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil || r.PostForm.Get(views.CSRFField) != "ok" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}
func (a *fakeAccounts) InviteOnly() bool { return true }
func (a *fakeAccounts) Invite(context.Context, *views.Viewer, string, string) error {
	return a.store.write("invite")
}
func (a *fakeAccounts) ResendInvite(context.Context, *views.Viewer, string) error {
	return a.store.write("resend")
}
func (a *fakeAccounts) RevokeInvite(context.Context, *views.Viewer, string) error {
	return a.store.write("revoke")
}
func (a *fakeAccounts) Invitations(context.Context) ([]views.Invitation, error) { return nil, nil }

type rendered struct {
	status int
	page   string
	data   any
}

type testEnv struct {
	store  *fakeStore
	acc    *fakeAccounts
	router chi.Router
	pages  []rendered
	viewer *views.Viewer
}

func newTestEnv() *testEnv {
	e := &testEnv{store: &fakeStore{}}
	e.acc = &fakeAccounts{store: e.store}
	render := func(w http.ResponseWriter, r *http.Request, status int, page string, data any) {
		e.pages = append(e.pages, rendered{status, page, data})
		w.WriteHeader(status)
	}
	h := New(e.store, e.acc, storage.NewMemory("http://img.test"), render)
	h.now = func() time.Time { return time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC) }
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(views.WithViewer(req.Context(), e.viewer)))
		})
	})
	h.Routes(r)
	e.router = r
	return e
}

func (e *testEnv) do(method, path string, form url.Values) *httptest.ResponseRecorder {
	var body *strings.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// routes lists every registered route with its parameters filled in.
func routes(t *testing.T, r chi.Router) [][2]string {
	t.Helper()
	var out [][2]string
	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		p := route
		for k, v := range map[string]string{"{id}": photo().ID, "{fid}": "1", "{name}": "ploy88", "{country}": "th", "{name}/": "ploy88/"} {
			p = strings.ReplaceAll(p, k, v)
		}
		out = append(out, [2]string{method, strings.TrimSuffix(p, "/")})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 25 {
		t.Fatalf("only %d routes", len(out))
	}
	return out
}

func TestEveryRouteIsGuarded(t *testing.T) {
	e := newTestEnv()
	for _, v := range []*views.Viewer{nil, {Name: "mali", SurveyorID: "x"}, {Email: "admin@example.org", Admin: false}} {
		e.viewer = v
		for _, rt := range routes(t, e.router) {
			before := e.acc.guarded
			rec := e.do(rt[0], rt[1], url.Values{views.CSRFField: {"ok"}, "reason": {"r"}, "photo": {photo().ID}, "email": {"a@example.org"}})
			if rec.Code != http.StatusNotFound || e.acc.guarded != before+1 {
				t.Errorf("%v %s %s: %d, guarded %v", v, rt[0], rt[1], rec.Code, e.acc.guarded > before)
			}
		}
	}
	if len(e.store.writes) != 0 {
		t.Fatalf("writes without an admin: %v", e.store.writes)
	}
}

func TestGetsChangeNothingAndPostsNeedTheForm(t *testing.T) {
	e := newTestEnv()
	e.viewer = &views.Viewer{Name: "boss", SurveyorID: "a1", Admin: true}
	for _, rt := range routes(t, e.router) {
		switch rt[0] {
		case http.MethodGet:
			// The files of a visible photo cannot be removed: no page for it.
			if rec := e.do(rt[0], rt[1], nil); rec.Code != http.StatusOK && rec.Code != http.StatusSeeOther &&
				!(rec.Code == http.StatusNotFound && strings.HasSuffix(rt[1], "/remove-files")) {
				t.Errorf("GET %s: %d", rt[1], rec.Code)
			}
		case http.MethodPost:
			if rec := e.do(rt[0], rt[1], url.Values{"reason": {"r"}}); rec.Code != http.StatusForbidden {
				t.Errorf("POST %s without CSRF: %d", rt[1], rec.Code)
			}
		}
	}
	if len(e.store.writes) != 0 {
		t.Fatalf("writes from GETs or forms without CSRF: %v", e.store.writes)
	}
	// With the form, each POST writes once and redirects back with ?done=.
	posts := 0
	for _, rt := range routes(t, e.router) {
		if rt[0] != http.MethodPost {
			continue
		}
		posts++
		rec := e.do(rt[0], rt[1], url.Values{views.CSRFField: {"ok"}, "reason": {"r"}, "photo": {photo().ID}, "email": {"a@example.org"}})
		if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "done=") {
			t.Errorf("POST %s: %d %q", rt[1], rec.Code, rec.Header().Get("Location"))
		}
	}
	if len(e.store.writes) != posts {
		t.Fatalf("%d writes for %d posts: %v", len(e.store.writes), posts, e.store.writes)
	}
}

func TestConfirmPages(t *testing.T) {
	e := newTestEnv()
	e.viewer = &views.Viewer{Name: "boss", SurveyorID: "a1", Admin: true}
	e.do(http.MethodGet, "/admin/photo/"+photo().ID+"/hide", nil)
	c := e.pages[len(e.pages)-1].data.(*views.AdminConfirm)
	if !c.NeedReason || !strings.Contains(strings.Join(c.Consequences, " "), "passes to mali") || c.Action != "/admin/photo/"+photo().ID+"/hide" {
		t.Fatalf("hide confirm %+v", c)
	}
	e.do(http.MethodGet, "/admin/surveyor/ploy88/block", nil)
	c = e.pages[len(e.pages)-1].data.(*views.AdminConfirm)
	if c.Option != "hide_photos" || !strings.Contains(c.OptionLabel, "3 visible photos") {
		t.Fatalf("block confirm %+v", c)
	}
	// An invalid reason renders the confirm page again with the message.
	e.store.err = db.ErrInvalid
	rec := e.do(http.MethodPost, "/admin/photo/"+photo().ID+"/hide", url.Values{views.CSRFField: {"ok"}, "reason": {""}})
	if rec.Code != http.StatusBadRequest || e.pages[len(e.pages)-1].page != views.PageAdminConfirm {
		t.Fatalf("invalid reason: %d %s", rec.Code, e.pages[len(e.pages)-1].page)
	}
	// Nothing to do: back with "unchanged".
	e.store.err = db.ErrUnchanged
	rec = e.do(http.MethodPost, "/admin/surveyor/ploy88/unblock", url.Values{views.CSRFField: {"ok"}})
	if rec.Header().Get("Location") != "/admin/surveyor/ploy88?done=unchanged" {
		t.Fatalf("unchanged: %q", rec.Header().Get("Location"))
	}
	// ?done= only shows fixed lines.
	e.do(http.MethodGet, "/admin/log?done=<script>", nil)
	if l := e.pages[len(e.pages)-1].data.(*views.AdminLog); l.Done != "" {
		t.Fatalf("done line %q", l.Done)
	}
	e.do(http.MethodGet, "/admin/log?done=hidden", nil)
	if l := e.pages[len(e.pages)-1].data.(*views.AdminLog); l.Done != "The photo is hidden." {
		t.Fatalf("done line %q", l.Done)
	}
}

func TestFeaturedForm(t *testing.T) {
	e := newTestEnv()
	e.viewer = &views.Viewer{Name: "boss", SurveyorID: "a1", Admin: true}
	for _, f := range []url.Values{
		{"photo": {"no id here"}},
		{"photo": {photo().ID}, "starts": {"tomorrow"}},
		{"photo": {photo().ID}, "ends": {"2026-13-01T00:00"}},
	} {
		f.Set(views.CSRFField, "ok")
		if rec := e.do(http.MethodPost, "/admin/featured", f); rec.Code != http.StatusBadRequest {
			t.Errorf("%v: %d", f, rec.Code)
		}
	}
	rec := e.do(http.MethodPost, "/admin/featured", url.Values{views.CSRFField: {"ok"},
		"photo": {"https://cartograph.asia/photo/" + photo().ID}, "starts": {"2026-11-02T08:00"}})
	if rec.Code != http.StatusSeeOther || len(e.store.writes) != 1 {
		t.Fatalf("pasted URL: %d %v", rec.Code, e.store.writes)
	}
}

func TestPageQueryKeepsFilters(t *testing.T) {
	e := newTestEnv()
	e.viewer = &views.Viewer{Name: "boss", SurveyorID: "a1", Admin: true}
	e.do(http.MethodGet, "/admin/photos?hidden=1&surveyor=ploy88&page=2", nil)
	p := e.pages[len(e.pages)-1].data.(*views.AdminPhotos)
	if p.Canonical != "/admin/photos?hidden=1&surveyor=ploy88" {
		t.Fatalf("canonical %q", p.Canonical)
	}
}

func TestRemoveFiles(t *testing.T) {
	e := newTestEnv()
	e.viewer = &views.Viewer{Name: "boss", SurveyorID: "a1", Admin: true}
	path := "/admin/photo/" + photo().ID + "/remove-files"
	// Hidden an hour ago: too early, no confirm page.
	hour := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	e.store.hiddenAt = &hour
	if rec := e.do(http.MethodGet, path, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("too early: %d", rec.Code)
	}
	// Hidden two days ago: the confirm page, then the store deletes through the files.
	days := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
	e.store.hiddenAt = &days
	if rec := e.do(http.MethodGet, path, nil); rec.Code != http.StatusOK {
		t.Fatalf("confirm: %d", rec.Code)
	}
	c := e.pages[len(e.pages)-1].data.(*views.AdminConfirm)
	if !c.NeedReason || !strings.Contains(strings.Join(c.Consequences, " "), "cannot be undone") {
		t.Fatalf("confirm page %+v", c)
	}
	rec := e.do(http.MethodPost, path, url.Values{views.CSRFField: {"ok"}, "reason": {"copyright"}})
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/photo/"+photo().ID+"?done=files-removed" {
		t.Fatalf("remove: %d %q", rec.Code, rec.Header().Get("Location"))
	}
}
