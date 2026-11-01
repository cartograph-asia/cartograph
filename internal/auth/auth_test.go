package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/views"
)

type rendered struct {
	status int
	page   string
	data   any
}

type fakeMailer struct {
	mu    sync.Mutex
	links map[string][]string
	err   error
}

func (f *fakeMailer) SendSignInLink(_ context.Context, to, link string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.links[to] = append(f.links[to], link)
	return nil
}

func (f *fakeMailer) last(t *testing.T, to string) string {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	l := f.links[to]
	if len(l) == 0 {
		t.Fatalf("no link sent to %s", to)
	}
	return l[len(l)-1]
}

type fakeRegions []views.RegionRef

func (f fakeRegions) RegionRefs(context.Context) ([]views.RegionRef, error) { return f, nil }

type env struct {
	t      *testing.T
	a      *Auth
	store  *memStore
	mail   *fakeMailer
	srv    *httptest.Server
	client *http.Client

	mu      sync.Mutex
	now     time.Time
	renders []rendered
}

func newEnv(t *testing.T, baseURL string) *env {
	t.Helper()
	e := &env{t: t, store: newMemStore(), mail: &fakeMailer{links: map[string][]string{}},
		now: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)}
	render := func(w http.ResponseWriter, r *http.Request, status int, page string, data any) {
		e.mu.Lock()
		e.renders = append(e.renders, rendered{status, page, data})
		e.mu.Unlock()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, page)
	}
	e.a = New(Config{BaseURL: baseURL}, e.store, e.mail, render,
		fakeRegions{{Slug: "chon-buri", NameEn: "Chon Buri"}, {Slug: "bangkok", NameEn: "Bangkok"}})
	e.a.now = func() time.Time {
		e.mu.Lock()
		defer e.mu.Unlock()
		return e.now
	}
	r := chi.NewRouter()
	r.Use(e.a.Middleware)
	e.a.Routes(r)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		v := views.ViewerFrom(r.Context())
		name := "anon"
		if v != nil {
			name = v.Name
		}
		_, _ = io.WriteString(w, "home:"+name)
	})
	e.srv = httptest.NewServer(r)
	t.Cleanup(e.srv.Close)
	jar, _ := cookiejar.New(nil)
	e.client = &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return e
}

func (e *env) advance(d time.Duration) {
	e.mu.Lock()
	e.now = e.now.Add(d)
	e.mu.Unlock()
}

func (e *env) last() rendered {
	e.t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.renders) == 0 {
		e.t.Fatal("nothing rendered")
	}
	return e.renders[len(e.renders)-1]
}

// csrf returns the CSRF token of the last rendered page.
func (e *env) csrf() string {
	e.t.Helper()
	b, ok := e.last().data.(views.Based)
	if !ok {
		e.t.Fatalf("last render %T has no Page", e.last().data)
	}
	return b.Base().CSRF
}

func (e *env) do(req *http.Request) *http.Response {
	e.t.Helper()
	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp
}

func (e *env) get(path string) *http.Response {
	e.t.Helper()
	req, _ := http.NewRequest(http.MethodGet, e.srv.URL+path, nil)
	return e.do(req)
}

func (e *env) post(path string, form url.Values, header ...string) *http.Response {
	e.t.Helper()
	req, _ := http.NewRequest(http.MethodPost, e.srv.URL+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	return e.do(req)
}

func (e *env) me() (int, map[string]string) {
	e.t.Helper()
	resp, err := e.client.Get(e.srv.URL + "/api/me")
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	var m map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		e.t.Fatal(err)
	}
	return resp.StatusCode, m
}

func want(t *testing.T, resp *http.Response, status int, location string) {
	t.Helper()
	if resp.StatusCode != status || resp.Header.Get("Location") != location {
		t.Fatalf("got %d %q, want %d %q", resp.StatusCode, resp.Header.Get("Location"), status, location)
	}
}

// requestLink fills the sign-in form and returns the token from the e-mail.
func (e *env) requestLink(email, next string) string {
	e.t.Helper()
	want(e.t, e.get("/signin?next="+url.QueryEscape(next)), http.StatusOK, "")
	resp := e.post("/signin", url.Values{"email": {email}, "next": {next}, "csrf": {e.csrf()}})
	if resp.StatusCode != http.StatusOK || e.last().page != views.PageCheckEmail {
		e.t.Fatalf("sign-in form: %d %s", resp.StatusCode, e.last().page)
	}
	link, err := url.Parse(e.mail.last(e.t, strings.ToLower(strings.TrimSpace(email))))
	if err != nil {
		e.t.Fatal(err)
	}
	return link.Query().Get("token")
}

// redeem opens the link and presses the button.
func (e *env) redeem(token string) *http.Response {
	e.t.Helper()
	want(e.t, e.get("/signin/link?token="+token), http.StatusOK, "")
	if e.last().page != views.PageConfirmSignIn {
		e.t.Fatalf("link page: %s", e.last().page)
	}
	return e.post("/signin/link", url.Values{"token": {token}, "csrf": {e.csrf()}})
}

func TestNewSurveyorFlow(t *testing.T) {
	e := newEnv(t, "http://cartograph.test")

	if status, m := e.me(); status != http.StatusUnauthorized || m["error"] == "" {
		t.Fatalf("me signed out: %d %v", status, m)
	}

	resp := e.get("/signin?next=/map")
	want(t, resp, http.StatusOK, "")
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control %q", resp.Header.Get("Cache-Control"))
	}
	si := e.last().data.(*views.SignIn)
	if si.Next != "/map" || si.CSRF == "" || !si.NoIndex {
		t.Fatalf("sign-in page: %+v", si)
	}

	resp = e.post("/signin", url.Values{"email": {"  New@Example.org "}, "next": {"/map"}, "csrf": {e.csrf()}})
	ce, _ := e.last().data.(*views.CheckEmail)
	if resp.StatusCode != http.StatusOK || ce == nil || ce.Email != "new@example.org" {
		t.Fatalf("check e-mail: %d %+v", resp.StatusCode, e.last())
	}
	link := e.mail.last(t, "new@example.org")
	if !strings.HasPrefix(link, "http://cartograph.test/signin/link?token=") {
		t.Fatalf("link %q", link)
	}
	u, _ := url.Parse(link)
	token := u.Query().Get("token")

	// Opening the link (as a mail scanner would) does not use the token.
	want(t, e.get("/signin/link?token="+token), http.StatusOK, "")
	want(t, e.get("/signin/link?token="+token), http.StatusOK, "")
	resp = e.redeem(token)
	want(t, resp, http.StatusSeeOther, "/signin/name?next=%2Fmap")
	var sc *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "cg_session" {
			sc = c
		}
	}
	if sc == nil || !sc.HttpOnly || sc.SameSite != http.SameSiteLaxMode || sc.Secure || sc.MaxAge != int(SessionTTL.Seconds()) {
		t.Fatalf("session cookie: %+v", sc)
	}

	// Single use.
	want(t, e.redeem(token), http.StatusBadRequest, "")
	if si := e.last().data.(*views.SignIn); si.Error != msgUsedLink {
		t.Fatalf("reuse: %+v", si)
	}

	if status, m := e.me(); status != http.StatusUnauthorized || !strings.Contains(m["error"], "name") {
		t.Fatalf("me without name: %d %v", status, m)
	}
	// Without a name, /signin and /settings lead to the name form.
	want(t, e.get("/signin?next=/map"), http.StatusSeeOther, "/signin/name?next=%2Fmap")
	want(t, e.get("/settings"), http.StatusSeeOther, "/signin/name?next=%2Fsettings")

	want(t, e.get("/signin/name?next=/map"), http.StatusOK, "")
	csrf := e.csrf()
	for _, bad := range []string{"ab", "has space", "under_score", "ä-umlaut", strings.Repeat("a", 31)} {
		want(t, e.post("/signin/name", url.Values{"name": {bad}, "next": {"/map"}, "csrf": {csrf}}), http.StatusBadRequest, "")
	}
	e.store.addSurveyor("Taken", "", true)
	want(t, e.post("/signin/name", url.Values{"name": {"taken"}, "next": {"/map"}, "csrf": {csrf}}), http.StatusConflict, "")
	if cn := e.last().data.(*views.ChooseName); cn.Error != msgNameTaken || cn.Name != "taken" || cn.Next != "/map" {
		t.Fatalf("taken: %+v", cn)
	}
	want(t, e.post("/signin/name", url.Values{"name": {"Nok-2026"}, "next": {"/map"}, "csrf": {csrf}}), http.StatusSeeOther, "/map")

	if status, m := e.me(); status != http.StatusOK || m["name"] != "Nok-2026" {
		t.Fatalf("me: %d %v", status, m)
	}
	// The name is chosen once.
	want(t, e.post("/signin/name", url.Values{"name": {"Other"}, "csrf": {csrf}}), http.StatusSeeOther, "/")
	want(t, e.get("/signin/name"), http.StatusSeeOther, "/")
	want(t, e.get("/signin?next=/map"), http.StatusSeeOther, "/map")
	if status, m := e.me(); status != http.StatusOK || m["name"] != "Nok-2026" {
		t.Fatalf("me after second name: %d %v", status, m)
	}

	// Sign out.
	want(t, e.post("/signout", url.Values{}), http.StatusForbidden, "")
	want(t, e.post("/signout", url.Values{"csrf": {csrf}}), http.StatusSeeOther, "/")
	if status, _ := e.me(); status != http.StatusUnauthorized {
		t.Fatalf("me after sign-out: %d", status)
	}
	if len(e.store.sessions) != 0 {
		t.Fatalf("sessions left: %d", len(e.store.sessions))
	}
}

func TestReturningSurveyorAndPlaceholder(t *testing.T) {
	e := newEnv(t, "http://cartograph.test")
	ph := e.store.addSurveyor("beach-walker", "adrian@example.org", true)
	token := e.requestLink("Adrian@Example.org", "/plot/88658bd11dfffff?x=1")
	want(t, e.redeem(token), http.StatusSeeOther, "/plot/88658bd11dfffff?x=1")
	if ph.placeholder {
		t.Fatal("placeholder flag not cleared on first sign-in")
	}
	if status, m := e.me(); status != http.StatusOK || m["name"] != "beach-walker" {
		t.Fatalf("me: %d %v", status, m)
	}
}

func TestSameAnswerForKnownAndUnknownAddresses(t *testing.T) {
	e := newEnv(t, "http://cartograph.test")
	e.store.addSurveyor("Known", "known@example.org", false)
	for _, addr := range []string{"known@example.org", "unknown@example.org"} {
		want(t, e.get("/signin"), http.StatusOK, "")
		resp := e.post("/signin", url.Values{"email": {addr}, "csrf": {e.csrf()}})
		if resp.StatusCode != http.StatusOK || e.last().page != views.PageCheckEmail {
			t.Fatalf("%s: %d %s", addr, resp.StatusCode, e.last().page)
		}
	}
}

func TestSignInRejects(t *testing.T) {
	e := newEnv(t, "http://cartograph.test")
	want(t, e.get("/signin"), http.StatusOK, "")
	csrf := e.csrf()

	// Bad addresses re-render the form with the input.
	for _, bad := range []string{"", "nobody", "a@b", "Ada <ada@example.org>", "a b@example.org", "x@example.org\r\nBcc: y@example.org"} {
		want(t, e.post("/signin", url.Values{"email": {bad}, "csrf": {csrf}}), http.StatusBadRequest, "")
		if si := e.last().data.(*views.SignIn); si.Error != msgBadEmail {
			t.Fatalf("%q: %+v", bad, si)
		}
	}
	// CSRF: missing, wrong, or a cross-site request.
	ok := url.Values{"email": {"ada@example.org"}}
	want(t, e.post("/signin", ok), http.StatusForbidden, "")
	ok.Set("csrf", "x"+csrf[1:])
	want(t, e.post("/signin", ok), http.StatusForbidden, "")
	ok.Set("csrf", csrf)
	want(t, e.post("/signin", ok, "Sec-Fetch-Site", "cross-site"), http.StatusForbidden, "")
	want(t, e.post("/signin", ok, "Origin", "https://evil.example"), http.StatusForbidden, "")
	if e.mail.links["ada@example.org"] != nil {
		t.Fatal("link sent despite failed CSRF check")
	}
	want(t, e.post("/signin", ok, "Sec-Fetch-Site", "same-origin"), http.StatusOK, "")

	// Malformed and expired links.
	want(t, e.get("/signin/link?token=abc"), http.StatusBadRequest, "")
	token := e.requestLink("late@example.org", "/")
	e.advance(LinkTTL + time.Second)
	want(t, e.redeem(token), http.StatusBadRequest, "")
	if si := e.last().data.(*views.SignIn); si.Error != msgUsedLink {
		t.Fatalf("expired: %+v", si)
	}

	// Mail failure is reported, not hidden.
	e.mail.err = errors.New("smtp down")
	want(t, e.get("/signin"), http.StatusOK, "")
	want(t, e.post("/signin", url.Values{"email": {"b@example.org"}, "csrf": {e.csrf()}}), http.StatusServiceUnavailable, "")
}

func TestNextIsLocal(t *testing.T) {
	cases := map[string]string{
		"":                             "/",
		"/map":                         "/map",
		"/map?cell=88658bd11dfffff":    "/map?cell=88658bd11dfffff",
		"/surveyor/Ada#photos":         "/surveyor/Ada#photos",
		"//evil.example":               "/",
		"/\\evil.example":              "/",
		"\\\\evil.example":             "/",
		"https://evil.example/":        "/",
		"javascript:alert(1)":          "/",
		"map":                          "/",
		"/\t/evil.example":             "/",
		"/%0a/x":                       "/%0a/x",
		"/" + strings.Repeat("a", 600): "/",
	}
	for in, out := range cases {
		if got := safeNext(in); got != out {
			t.Errorf("safeNext(%q) = %q, want %q", in, got, out)
		}
	}

	// A next that is not local never reaches the token or the redirect.
	e := newEnv(t, "http://cartograph.test")
	token := e.requestLink("ada@example.org", "https://evil.example/")
	want(t, e.redeem(token), http.StatusSeeOther, "/signin/name")
}

func TestThrottle(t *testing.T) {
	e := newEnv(t, "http://cartograph.test")
	want(t, e.get("/signin"), http.StatusOK, "")
	csrf := e.csrf()
	for i := 0; i < linksPerAddress+2; i++ {
		resp := e.post("/signin", url.Values{"email": {"ada@example.org"}, "csrf": {csrf}})
		if resp.StatusCode != http.StatusOK || e.last().page != views.PageCheckEmail {
			t.Fatalf("request %d: %d %s", i, resp.StatusCode, e.last().page)
		}
	}
	if n := len(e.mail.links["ada@example.org"]); n != linksPerAddress {
		t.Fatalf("sent %d links, want %d", n, linksPerAddress)
	}
	e.advance(throttleWindow + time.Second)
	e.post("/signin", url.Values{"email": {"ada@example.org"}, "csrf": {csrf}})
	if n := len(e.mail.links["ada@example.org"]); n != linksPerAddress+1 {
		t.Fatalf("after the window: %d links", n)
	}

	// Per IP: the httptest client is loopback, so X-Forwarded-For counts.
	for i := 0; i < signInsPerIP; i++ {
		resp := e.post("/signin", url.Values{"email": {"x@example.org"}, "csrf": {csrf}}, "X-Forwarded-For", "203.0.113.7")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("ip request %d: %d", i, resp.StatusCode)
		}
	}
	want(t, e.post("/signin", url.Values{"email": {"y@example.org"}, "csrf": {csrf}}, "X-Forwarded-For", "203.0.113.7"), http.StatusTooManyRequests, "")
	want(t, e.post("/signin", url.Values{"email": {"y@example.org"}, "csrf": {csrf}}, "X-Forwarded-For", "203.0.113.8"), http.StatusOK, "")
}

func TestSessionSlidesAndExpires(t *testing.T) {
	e := newEnv(t, "http://cartograph.test")
	e.store.addSurveyor("Ada", "ada@example.org", false)
	want(t, e.redeem(e.requestLink("ada@example.org", "/")), http.StatusSeeOther, "/")

	e.advance(10 * time.Minute)
	e.get("/")
	if e.store.touched != 0 {
		t.Fatal("touched within the hour")
	}
	e.advance(2 * time.Hour)
	resp := e.get("/")
	if e.store.touched != 1 {
		t.Fatal("not touched after an hour")
	}
	if c := resp.Cookies(); len(c) != 1 || c[0].Name != "cg_session" || c[0].MaxAge != int(SessionTTL.Seconds()) {
		t.Fatalf("refreshed cookie: %+v", c)
	}
	e.advance(SessionTTL - time.Minute) // past the first expiry, within the slid one
	if status, _ := e.me(); status != http.StatusOK {
		t.Fatalf("slid session: %d", status)
	}
	e.advance(SessionTTL + time.Minute)
	resp = e.get("/")
	if c := resp.Cookies(); len(c) != 1 || c[0].MaxAge != -1 {
		t.Fatalf("expired session cookie not cleared: %+v", c)
	}
	if status, _ := e.me(); status != http.StatusUnauthorized {
		t.Fatalf("expired session: %d", status)
	}
}

func TestDatabaseErrorIsAnonymous(t *testing.T) {
	e := newEnv(t, "http://cartograph.test")
	e.store.addSurveyor("Ada", "ada@example.org", false)
	want(t, e.redeem(e.requestLink("ada@example.org", "/")), http.StatusSeeOther, "/")
	e.store.failNext = errors.New("db down")
	if status, _ := e.me(); status != http.StatusUnauthorized {
		t.Fatalf("db error: %d", status)
	}
	if status, _ := e.me(); status != http.StatusOK {
		t.Fatalf("session lost after a db error: %d", status)
	}
}

func TestSettings(t *testing.T) {
	e := newEnv(t, "http://cartograph.test")
	want(t, e.get("/settings"), http.StatusSeeOther, "/signin?next=%2Fsettings")
	ada := e.store.addSurveyor("Ada", "ada@example.org", false)
	want(t, e.redeem(e.requestLink("ada@example.org", "/settings")), http.StatusSeeOther, "/settings")

	want(t, e.get("/settings"), http.StatusOK, "")
	ep := e.last().data.(*views.EditProfile)
	if ep.Name != "Ada" || len(ep.Regions) != 2 || ep.Saved || ep.CSRF == "" {
		t.Fatalf("form: %+v", ep)
	}
	csrf := ep.CSRF

	want(t, e.post("/settings", url.Values{"bio": {"x"}}), http.StatusForbidden, "")
	want(t, e.post("/settings", url.Values{"bio": {strings.Repeat("ก", 501)}, "csrf": {csrf}}), http.StatusBadRequest, "")
	if ep := e.last().data.(*views.EditProfile); ep.Error != msgBioTooLong {
		t.Fatalf("long bio: %+v", ep)
	}
	want(t, e.post("/settings", url.Values{"bio": {"hi"}, "home_region": {"atlantis"}, "csrf": {csrf}}), http.StatusBadRequest, "")

	bio := "  Walking the coast\r\nfrom Pattaya to Sattahip.\x00 " + strings.Repeat("ก", 400) + "  "
	want(t, e.post("/settings", url.Values{"bio": {bio}, "home_region": {"chon-buri"}, "csrf": {csrf}}), http.StatusSeeOther, "/settings?saved=1")
	wantBio := "Walking the coast\nfrom Pattaya to Sattahip. " + strings.Repeat("ก", 400)
	if ada.bio != wantBio || ada.home != "chon-buri" {
		t.Fatalf("saved %q %q", ada.bio, ada.home)
	}
	want(t, e.get("/settings?saved=1"), http.StatusOK, "")
	if ep := e.last().data.(*views.EditProfile); !ep.Saved || ep.Bio != wantBio || ep.HomeRegion != "chon-buri" {
		t.Fatalf("after save: %+v", ep)
	}
	want(t, e.post("/settings", url.Values{"bio": {""}, "home_region": {""}, "csrf": {csrf}}), http.StatusSeeOther, "/settings?saved=1")
	if ada.bio != "" || ada.home != "" {
		t.Fatalf("cleared %q %q", ada.bio, ada.home)
	}
}

func TestSecureCookies(t *testing.T) {
	e := newEnv(t, "https://cartograph.asia")
	resp := e.get("/signin")
	cs := resp.Cookies()
	if len(cs) != 1 || cs[0].Name != "__Host-cg_csrf" || !cs[0].Secure || !cs[0].HttpOnly || cs[0].Path != "/" {
		t.Fatalf("form cookie: %+v", cs)
	}
}

func TestConfigFromEnv(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	cfg, err := ConfigFromEnv(env(nil))
	if err != nil || cfg.BaseURL != "http://localhost:8080" || cfg.SMTP.Host != "" || cfg.SMTP.Port != 587 {
		t.Fatalf("defaults: %+v %v", cfg, err)
	}
	cfg, err = ConfigFromEnv(env(map[string]string{
		"BASE_URL": "https://cartograph.asia/", "SMTP_HOST": "smtp.example.org", "SMTP_PORT": "2525",
		"SMTP_USER": "u", "SMTP_PASSWORD": "p", "MAIL_FROM": "Cartograph <hello@cartograph.asia>",
	}))
	if err != nil || cfg.BaseURL != "https://cartograph.asia" || cfg.SMTP.Port != 2525 || cfg.SMTP.From == "" {
		t.Fatalf("full: %+v %v", cfg, err)
	}
	for _, bad := range []map[string]string{
		{"BASE_URL": "cartograph.asia"},
		{"BASE_URL": "https://cartograph.asia/app"},
		{"SMTP_PORT": "x"},
		{"SMTP_HOST": "smtp.example.org"},
		{"SMTP_HOST": "smtp.example.org", "MAIL_FROM": "not an address"},
	} {
		if _, err := ConfigFromEnv(env(bad)); err == nil {
			t.Errorf("%v: no error", bad)
		}
	}
}

func TestBuildMessage(t *testing.T) {
	msg, err := buildMessage("Cartograph <hello@cartograph.asia>", "ada@example.org", signInSubject,
		signInBody("https://cartograph.asia/signin/link?token=abc"), time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	s := string(msg)
	for _, w := range []string{"From: \"Cartograph\" <hello@cartograph.asia>\r\n", "To: ada@example.org\r\n",
		"Subject: Your Cartograph sign-in link\r\n", "@cartograph.asia>\r\n", "\r\n\r\nHello,\r\n", "token=3Dabc"} {
		if !strings.Contains(s, w) {
			t.Errorf("message lacks %q:\n%s", w, s)
		}
	}
	if _, err := buildMessage("hello@cartograph.asia", "a@example.org\r\nBcc: b@example.org", "s", "b", time.Now()); err == nil {
		t.Error("header injection accepted")
	}
}

func TestClientIP(t *testing.T) {
	cases := []struct{ remote, xff, want string }{
		{"203.0.113.9:1234", "198.51.100.1", "203.0.113.9"}, // public peer: XFF ignored
		{"127.0.0.1:1234", "198.51.100.1", "198.51.100.1"},
		{"172.18.0.5:1234", "10.0.0.1, 198.51.100.2", "198.51.100.2"},
		{"172.18.0.5:1234", "", "172.18.0.5"},
		{"[2001:db8:1:2:3:4:5:6]:1", "", "2001:db8:1:2::/64"},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodPost, "/signin", nil)
		r.RemoteAddr = c.remote
		if c.xff != "" {
			r.Header.Set("X-Forwarded-For", c.xff)
		}
		if got := clientIP(r); got != c.want {
			t.Errorf("%s %q: %s, want %s", c.remote, c.xff, got, c.want)
		}
	}
}
