package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cartograph-asia/cartograph/internal/db"
	"github.com/cartograph-asia/cartograph/internal/h3"
	"github.com/cartograph-asia/cartograph/internal/photos"
	"github.com/cartograph-asia/cartograph/internal/storage"
	"github.com/cartograph-asia/cartograph/internal/views"
)

// fakeRepo is an in-memory Repo with the same claim semantics as db.DB.
type fakeRepo struct {
	mu     sync.Mutex
	plots  map[string]*db.PlotDetail
	parent map[string][2]string // cell -> parent5, parent6
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{plots: map[string]*db.PlotDetail{}, parent: map[string][2]string{}}
}

func (f *fakeRepo) ClaimPhoto(_ context.Context, p db.NewPhoto) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	pl, ok := f.plots[p.Cell]
	if !ok {
		pl = &db.PlotDetail{Cell: p.Cell, FirstSurveyor: p.Surveyor, ClaimedAt: p.UploadedAt}
		f.plots[p.Cell] = pl
		f.parent[p.Cell] = [2]string{p.Parent5, p.Parent6}
	}
	pl.PhotoCount++
	pl.Photos = append(pl.Photos, db.PlotPhoto{
		ID: p.ID, Surveyor: p.Surveyor, TakenAt: p.TakenAt, UploadedAt: p.UploadedAt,
		KeyLarge: p.KeyLarge, KeyThumb: p.KeyThumb,
	})
	return !ok, nil
}

func (f *fakeRepo) Plots(_ context.Context, cells []string) ([]db.PlotSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []db.PlotSummary
	for _, c := range cells {
		if p, ok := f.plots[c]; ok {
			out = append(out, db.PlotSummary{Cell: c, PhotoCount: p.PhotoCount, FirstSurveyor: p.FirstSurveyor,
				ClaimedAt: p.ClaimedAt, ThumbKey: p.Photos[0].KeyThumb})
		}
	}
	return out, nil
}

func (f *fakeRepo) Plot(_ context.Context, cell string) (*db.PlotDetail, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.plots[cell]
	if !ok {
		return nil, db.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (f *fakeRepo) Coverage(_ context.Context, res int, parents []string) (map[string]int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	want := map[string]bool{}
	for _, p := range parents {
		want[p] = true
	}
	out := map[string]int64{}
	for _, pp := range f.parent {
		p := pp[res-5]
		if want[p] {
			out[p]++
		}
	}
	return out, nil
}

type testEnv struct {
	t     *testing.T
	srv   *httptest.Server
	s     *Server
	store *storage.Memory
	repo  *fakeRepo
	as    atomic.Pointer[views.Viewer] // the signed-in viewer internal/auth would set; nil = signed out
}

func newEnv(t *testing.T, maxMB int) *testEnv { return newEnvWith(t, Config{MaxUploadMB: maxMB}, nil) }

// newEnvWith starts the API with cfg (SpoolDir is set to a test directory)
// on an http.Server that srvCfg may adjust before it starts. Every test
// checks at its end that no spooled upload is left behind.
func newEnvWith(t *testing.T, cfg Config, srvCfg func(*http.Server)) *testEnv {
	repo := newFakeRepo()
	store := storage.NewMemory("http://img.test/img")
	cfg.SpoolDir = t.TempDir()
	s := New(repo, store, cfg)
	s.now = func() time.Time { return time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC) }
	e := &testEnv{t: t, s: s, store: store, repo: repo}
	e.signIn("Ada")
	h := s.Handler()
	e.srv = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := e.as.Load()
		if name := r.Header.Get("X-Test-Surveyor"); name != "" {
			v = &views.Viewer{SurveyorID: "id-" + name, Name: name, Email: name + "@example.org"}
		}
		h.ServeHTTP(w, r.WithContext(views.WithViewer(r.Context(), v)))
	}))
	if srvCfg != nil {
		srvCfg(e.srv.Config)
	}
	e.srv.Start()
	t.Cleanup(e.srv.Close)
	t.Cleanup(func() {
		if left, _ := os.ReadDir(cfg.SpoolDir); len(left) != 0 {
			t.Errorf("spooled uploads left behind: %v", left)
		}
	})
	return e
}

// signIn makes the following requests come from Surveyor name ("" = a
// session without a name yet).
func (e *testEnv) signIn(name string) {
	e.as.Store(&views.Viewer{SurveyorID: "id-" + name, Name: name, Email: name + "@example.org"})
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("../photos/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func multipartBody(file []byte, fields map[string]string) ([]byte, string) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	if file != nil {
		fw, _ := mw.CreateFormFile("photo", "photo.jpg")
		_, _ = fw.Write(file)
	}
	_ = mw.Close()
	return body.Bytes(), mw.FormDataContentType()
}

func (e *testEnv) upload(file []byte, fields map[string]string) (int, map[string]any) {
	e.t.Helper()
	return e.uploadAs("", file, fields)
}

// uploadAs uploads as Surveyor name ("" = the signed-in viewer).
func (e *testEnv) uploadAs(name string, file []byte, fields map[string]string) (int, map[string]any) {
	e.t.Helper()
	body, ct := multipartBody(file, fields)
	return e.post(name, ct, bytes.NewReader(body))
}

func (e *testEnv) post(name, contentType string, body io.Reader) (int, map[string]any) {
	e.t.Helper()
	req, _ := http.NewRequest(http.MethodPost, e.srv.URL+"/api/photos", body)
	req.Header.Set("Content-Type", contentType)
	if name != "" {
		req.Header.Set("X-Test-Surveyor", name)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	return decode(e.t, resp)
}

func (e *testEnv) get(path string) (int, map[string]any) {
	e.t.Helper()
	resp, err := http.Get(e.srv.URL + path)
	if err != nil {
		e.t.Fatal(err)
	}
	return decode(e.t, resp)
}

func decode(t *testing.T, resp *http.Response) (int, map[string]any) {
	t.Helper()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("status %d: response is not JSON: %q", resp.StatusCode, b)
	}
	return resp.StatusCode, m
}

func TestCreatePhotoClaimsThenAdds(t *testing.T) {
	e := newEnv(t, 25)
	img := fixture(t, "gps.jpg")
	wantCell, _ := h3.CellFromLatLng(12.8517888888889, 100.899880555556, h3.PlotRes)

	status, body := e.upload(img, map[string]string{"surveyor": "Ada"})
	if status != http.StatusCreated {
		t.Fatalf("first upload: %d %v", status, body)
	}
	if body["cell"] != wantCell || body["claimed"] != true {
		t.Fatalf("first upload: %v", body)
	}
	id := body["id"].(string)
	urls := body["urls"].(map[string]any)
	wantLarge := "http://img.test/img/photos/2026/11/" + id + "_large.jpg"
	if urls["large"] != wantLarge || urls["thumb"] != "http://img.test/img/photos/2026/11/"+id+"_thumb.jpg" {
		t.Fatalf("urls: %v", urls)
	}

	keys := e.store.Keys()
	sort.Strings(keys)
	if len(keys) != 3 || keys[1] != "photos/2026/11/"+id+"_medium.jpg" {
		t.Fatalf("stored keys: %v", keys)
	}
	large, _ := e.store.Get("photos/2026/11/" + id + "_large.jpg")
	if large.ContentType != "image/jpeg" {
		t.Errorf("content type %q", large.ContentType)
	}
	if m := photos.ReadMeta(large.Data); m.HasGPS {
		t.Error("stored large variant still has GPS EXIF")
	}

	e.signIn("Grace")
	status, body = e.upload(img, map[string]string{"surveyor": "Ada"}) // the field is ignored
	if status != http.StatusCreated || body["claimed"] != false || body["cell"] != wantCell {
		t.Fatalf("second upload: %d %v", status, body)
	}

	status, body = e.get("/api/plots/" + wantCell)
	if status != http.StatusOK {
		t.Fatalf("get plot: %d %v", status, body)
	}
	if body["photo_count"] != 2.0 || body["first_surveyor"] != "Ada" {
		t.Fatalf("get plot: %v", body)
	}
	if s, _ := body["claimed_at"].(string); s == "" || s == "0001-01-01T00:00:00Z" {
		t.Fatalf("get plot: claimed_at missing: %v", body)
	}
	ph := body["photos"].([]any)
	if len(ph) != 2 || ph[0].(map[string]any)["surveyor"] != "Ada" || ph[0].(map[string]any)["taken_at"] != "2026-09-09T09:38:06Z" {
		t.Fatalf("photos: %v", ph)
	}
	if ph[1].(map[string]any)["surveyor"] != "Grace" {
		t.Fatalf("second photo should be the signed-in Grace's: %v", ph[1])
	}

	status, body = e.get("/api/plots?cells=" + wantCell + ",88283082a3fffff")
	plots, _ := body["plots"].([]any)
	if status != http.StatusOK || len(plots) != 1 || plots[0].(map[string]any)["photo_count"] != 2.0 ||
		plots[0].(map[string]any)["thumb"] != "http://img.test/img/photos/2026/11/"+id+"_thumb.jpg" {
		t.Fatalf("list plots: %d %v", status, body)
	}

	p6, _ := h3.Parent(wantCell, 6)
	status, body = e.get("/api/coverage?res=6&cells=" + p6)
	cells, _ := body["cells"].([]any)
	if status != http.StatusOK || len(cells) != 1 {
		t.Fatalf("coverage: %d %v", status, body)
	}
	if c := cells[0].(map[string]any); c["cell"] != p6 || c["surveyed"] != 1.0 || c["total"] != 49.0 {
		t.Fatalf("coverage: %v", c)
	}
}

func TestCreatePhotoFallbackLatLng(t *testing.T) {
	e := newEnv(t, 25)
	img := fixture(t, "nogps.jpg")

	status, body := e.upload(img, map[string]string{"surveyor": "Ada"})
	if status != http.StatusBadRequest || !strings.Contains(body["error"].(string), "lat and lng") {
		t.Fatalf("no GPS, no lat/lng: %d %v", status, body)
	}
	status, body = e.upload(img, map[string]string{"surveyor": "Ada", "lat": "13.7563", "lng": "100.5018"})
	want, _ := h3.CellFromLatLng(13.7563, 100.5018, h3.PlotRes)
	if status != http.StatusCreated || body["cell"] != want {
		t.Fatalf("with lat/lng: %d %v", status, body)
	}
	// EXIF GPS wins over the form fields.
	status, body = e.upload(fixture(t, "gps.jpg"), map[string]string{"surveyor": "Ada", "lat": "13.7563", "lng": "100.5018"})
	if status != http.StatusCreated || body["cell"] == want {
		t.Fatalf("EXIF should win: %d %v", status, body)
	}
}

func TestCreatePhotoRejects(t *testing.T) {
	e := newEnv(t, 1)
	heic := append([]byte{0, 0, 0, 0x18}, []byte("ftypheic\x00\x00\x00\x00mif1heic")...)
	big := append(fixture(t, "gps.jpg"), make([]byte, 1<<20)...)
	cases := []struct {
		name   string
		file   []byte
		fields map[string]string
		want   string
	}{
		{"no photo", nil, map[string]string{"surveyor": "Ada"}, "photo is required"},
		{"heic", heic, map[string]string{"surveyor": "Ada"}, "HEIC is not supported"},
		{"garbage", []byte("definitely not a photo"), map[string]string{"surveyor": "Ada"}, "unsupported file type"},
		{"too large", big, map[string]string{"surveyor": "Ada"}, "larger than 1 MB"},
		{"bad lat", fixture(t, "nogps.jpg"), map[string]string{"surveyor": "Ada", "lat": "95", "lng": "100"}, "lat and lng"},
		{"lat only", fixture(t, "nogps.jpg"), map[string]string{"surveyor": "Ada", "lat": "13"}, "lat and lng"},
	}
	for _, c := range cases {
		status, body := e.upload(c.file, c.fields)
		msg, _ := body["error"].(string)
		if status != http.StatusBadRequest || !strings.Contains(msg, c.want) {
			t.Errorf("%s: %d %v, want 400 containing %q", c.name, status, body, c.want)
		}
	}
	if keys := e.store.Keys(); len(keys) != 0 {
		t.Errorf("rejected uploads stored objects: %v", keys)
	}

	resp, err := http.Post(e.srv.URL+"/api/photos", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	if status, _ := decode(t, resp); status != http.StatusBadRequest {
		t.Errorf("non-multipart: %d", status)
	}
}

// slowUpload is an upload by Surveyor name whose body arrives until done
// is called; status then receives its response's status.
type slowUpload struct {
	w      *io.PipeWriter
	rest   []byte
	status chan int
}

func (e *testEnv) startSlowUpload(name string, file []byte, fields map[string]string) *slowUpload {
	body, ct := multipartBody(file, fields)
	pr, pw := io.Pipe()
	u := &slowUpload{w: pw, rest: body[len(body)/2:], status: make(chan int, 1)}
	go func() {
		req, _ := http.NewRequest(http.MethodPost, e.srv.URL+"/api/photos", pr)
		req.Header.Set("Content-Type", ct)
		req.Header.Set("X-Test-Surveyor", name)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			u.status <- 0
			return
		}
		resp.Body.Close()
		u.status <- resp.StatusCode
	}()
	go func() { _, _ = pw.Write(body[:len(body)/2]) }()
	return u
}

func (u *slowUpload) done() int {
	_, _ = u.w.Write(u.rest)
	_ = u.w.Close()
	return <-u.status
}

// waitFor polls cond for up to two seconds.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for i := 0; !cond(); i++ {
		if i == 200 {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// While a body is still arriving, the upload holds its file on disk and no
// processing slot; meanwhile the same Surveyor cannot start a second upload,
// and other Surveyors can upload.
func TestCreatePhotoSpoolsWithoutSlot(t *testing.T) {
	e := newEnvWith(t, Config{MaxUploadMB: 25, MaxConcurrentProcessing: 1}, nil)
	img := fixture(t, "gps.jpg")
	slow := e.startSlowUpload("Ada", img, nil)
	waitFor(t, "the spooled upload", func() bool {
		files, _ := os.ReadDir(e.s.cfg.SpoolDir)
		if len(files) != 1 {
			return false
		}
		fi, err := files[0].Info()
		return err == nil && fi.Size() > 0
	})
	if n := len(e.s.sem); n != 0 {
		t.Fatalf("%d processing slots held while the body arrives", n)
	}

	status, body := e.uploadAs("Ada", img, nil)
	if msg, _ := body["error"].(string); status != http.StatusTooManyRequests || !strings.Contains(msg, "still in progress") {
		t.Errorf("second upload by the same Surveyor: %d %v", status, body)
	}
	if status, body := e.uploadAs("Bo", img, nil); status != http.StatusCreated {
		t.Errorf("another Surveyor during a slow upload: %d %v", status, body)
	}

	if status := slow.done(); status != http.StatusCreated {
		t.Fatalf("slow upload: %d", status)
	}
	if status, body := e.uploadAs("Ada", img, nil); status != http.StatusCreated {
		t.Errorf("next upload after the first finished: %d %v", status, body)
	}
	waitFor(t, "the slots to be released", func() bool { return len(e.s.sem) == 0 })
}

// The upload route sets its own deadline: it outlasts a short server
// ReadTimeout and is cut off by a short UploadTimeout. Other routes keep the
// server's timeouts, since nothing else changes them.
func TestCreatePhotoUploadTimeout(t *testing.T) {
	body, ct := multipartBody(fixture(t, "gps.jpg"), nil)
	trickle := func() io.Reader { // about one second for the whole body
		pr, pw := io.Pipe()
		go func() {
			step := len(body)/4 + 1
			for i := 0; i < len(body); i += step {
				if _, err := pw.Write(body[i:min(i+step, len(body))]); err != nil {
					return
				}
				time.Sleep(250 * time.Millisecond)
			}
			_ = pw.Close()
		}()
		return pr
	}

	e := newEnvWith(t, Config{MaxUploadMB: 25, UploadTimeout: 10 * time.Second},
		func(s *http.Server) { s.ReadTimeout = 300 * time.Millisecond })
	if status, resp := e.post("Ada", ct, trickle()); status != http.StatusCreated {
		t.Fatalf("slow upload, short ReadTimeout: %d %v", status, resp)
	}

	e = newEnvWith(t, Config{MaxUploadMB: 25, UploadTimeout: 400 * time.Millisecond},
		func(s *http.Server) { s.ReadTimeout = time.Minute })
	req, _ := http.NewRequest(http.MethodPost, e.srv.URL+"/api/photos", trickle())
	req.Header.Set("Content-Type", ct)
	req.Header.Set("X-Test-Surveyor", "Ada")
	if resp, err := http.DefaultClient.Do(req); err == nil {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			t.Fatalf("slow upload, short UploadTimeout: %d %s", resp.StatusCode, b)
		}
	}
	waitFor(t, "the upload to end", func() bool {
		e.s.mu.Lock()
		defer e.s.mu.Unlock()
		return len(e.s.uploading) == 0
	})
}

func TestCreatePhotoNeedsSurveyor(t *testing.T) {
	e := newEnv(t, 25)
	img := fixture(t, "gps.jpg")
	e.as.Store(nil)
	if status, body := e.upload(img, map[string]string{"surveyor": "Ada"}); status != http.StatusUnauthorized || body["error"] == nil {
		t.Errorf("signed out: %d %v", status, body)
	}
	e.signIn("")
	if status, body := e.upload(img, nil); status != http.StatusUnauthorized {
		t.Errorf("no name yet: %d %v", status, body)
	}

	e.signIn("Ada")
	for _, h := range [][2]string{{"Sec-Fetch-Site", "cross-site"}, {"Sec-Fetch-Site", "same-site"}, {"Origin", "https://evil.example"}} {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		fw, _ := mw.CreateFormFile("photo", "photo.jpg")
		_, _ = fw.Write(img)
		_ = mw.Close()
		req, _ := http.NewRequest(http.MethodPost, e.srv.URL+"/api/photos", &body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set(h[0], h[1])
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if status, _ := decode(t, resp); status != http.StatusForbidden {
			t.Errorf("%s: %s: %d", h[0], h[1], status)
		}
	}
	if keys := e.store.Keys(); len(keys) != 0 {
		t.Errorf("refused uploads stored objects: %v", keys)
	}
}

func TestReadEndpointsValidate(t *testing.T) {
	e := newEnv(t, 25)
	cell, _ := h3.CellFromLatLng(13.7563, 100.5018, h3.PlotRes)
	p5, _ := h3.Parent(cell, 5)
	many := strings.TrimSuffix(strings.Repeat(cell+",", 2001), ",")

	cases := []struct {
		path   string
		status int
	}{
		{"/api/plots/" + cell, http.StatusNotFound},
		{"/api/plots/nonsense", http.StatusBadRequest},
		{"/api/plots/" + p5, http.StatusBadRequest},
		{"/api/plots?cells=" + p5, http.StatusBadRequest},
		{"/api/plots?cells=xyz", http.StatusBadRequest},
		{"/api/plots?cells=" + many, http.StatusBadRequest},
		{"/api/plots?cells=", http.StatusOK},
		{"/api/coverage?res=7&cells=" + p5, http.StatusBadRequest},
		{"/api/coverage?res=6&cells=" + p5, http.StatusBadRequest},
		{"/api/coverage?res=5&cells=" + p5, http.StatusOK},
		{"/api/nope", http.StatusNotFound},
	}
	for _, c := range cases {
		status, body := e.get(c.path)
		if status != c.status {
			t.Errorf("GET %.80s: %d %v, want %d", c.path, status, body, c.status)
		}
		if status >= 400 && body["error"] == nil {
			t.Errorf("GET %.80s: error body missing: %v", c.path, body)
		}
	}

	status, body := e.get("/api/coverage?res=5&cells=" + p5)
	if c, ok := body["cells"].([]any); status != 200 || !ok || len(c) != 0 {
		t.Errorf("blank parents must be omitted from coverage: %v", body)
	}

	resp, err := http.Post(e.srv.URL+"/api/plots", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	if status, body := decode(t, resp); status != http.StatusMethodNotAllowed || body["error"] == nil {
		t.Errorf("POST /api/plots: %d %v", status, body)
	}
}

func TestHealthzAndCORS(t *testing.T) {
	e := newEnv(t, 25)
	resp, err := http.Get(e.srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || string(b) != "ok" {
		t.Errorf("healthz: %d %q", resp.StatusCode, b)
	}
	req, _ := http.NewRequest(http.MethodOptions, e.srv.URL+"/api/photos", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent || resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("preflight: %d %v", resp.StatusCode, resp.Header)
	}
}
