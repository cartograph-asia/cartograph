package crawlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/api"
	"github.com/cartograph-asia/cartograph/internal/views"
	"github.com/cartograph-asia/cartograph/internal/views/viewstest"
	"github.com/cartograph-asia/cartograph/internal/web"
)

func env(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestFromEnv(t *testing.T) {
	for in, want := range map[string]Policy{"": Block, "block": Block, " BLOCK ": Block, "allow": Allow, "Allow": Allow} {
		got, err := FromEnv(env("CRAWLERS", in))
		if err != nil || got != want {
			t.Errorf("CRAWLERS=%q: %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"no", "true", "disallow", "block,allow"} {
		if _, err := FromEnv(env("CRAWLERS", in)); err == nil || !strings.Contains(err.Error(), "CRAWLERS must be block or allow") {
			t.Errorf("CRAWLERS=%q: err %v", in, err)
		}
	}
}

// groups parses robots.txt into user agent -> Disallow lines.
func groups(t *testing.T, body string) map[string][]string {
	t.Helper()
	g := map[string][]string{}
	ua := ""
	for _, line := range strings.Split(body, "\n") {
		k, v, ok := strings.Cut(line, ": ")
		switch {
		case !ok || strings.HasPrefix(line, "#"):
		case k == "User-agent":
			if _, dup := g[v]; dup {
				t.Errorf("user agent %s named twice", v)
			}
			ua, g[v] = v, []string{}
		case k == "Disallow":
			g[ua] = append(g[ua], v)
		default:
			t.Errorf("unexpected line %q", line)
		}
	}
	return g
}

func getRobots(t *testing.T, p Policy) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	p.Robots(rec, httptest.NewRequest(http.MethodGet, "/robots.txt", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" || rec.Header().Get("Cache-Control") != "public, max-age=300" {
		t.Fatalf("%s: %d %v", p, rec.Code, rec.Header())
	}
	return rec
}

func TestRobotsBlock(t *testing.T) {
	g := groups(t, getRobots(t, Block).Body.String())
	if len(g) != len(AICrawlers)+1 {
		t.Errorf("%d groups, want %d", len(g), len(AICrawlers)+1)
	}
	for _, ua := range append([]string{"*"}, AICrawlers...) {
		if d := g[ua]; len(d) != 1 || d[0] != "/" {
			t.Errorf("%s: Disallow %q, want [/]", ua, d)
		}
	}
}

func TestRobotsAllow(t *testing.T) {
	g := groups(t, getRobots(t, Allow).Body.String())
	if got, want := strings.Join(g["*"], " "), "/settings /signin /api/"; got != want {
		t.Errorf("*: Disallow %q, want %q", got, want)
	}
	for _, d := range g["*"] {
		if d == "/" || strings.HasPrefix(d, "/admin") {
			t.Errorf("*: Disallow %q", d)
		}
	}
	// The AI crawlers stay out with allow.
	for _, ua := range []string{"GPTBot", "ChatGPT-User", "OAI-SearchBot", "ClaudeBot", "anthropic-ai", "Google-Extended", "CCBot",
		"PerplexityBot", "Bytespider", "Amazonbot", "Applebot-Extended", "meta-externalagent", "AhrefsBot", "SemrushBot", "MJ12bot"} {
		if d := g[ua]; len(d) != 1 || d[0] != "/" {
			t.Errorf("%s: Disallow %q, want [/]", ua, d)
		}
	}
}

// site is the api's router in miniature, as cmd/api builds it: the pages
// from internal/web on a fake source, the JSON API, robots.txt and a route
// that sets its own weaker tag like the admin guard.
func site(t *testing.T, p Policy) http.Handler {
	t.Helper()
	rd, err := web.NewRenderer(web.Config{BaseURL: "https://cartograph.asia", ImgBaseURL: "https://img.cartograph.asia"})
	if err != nil {
		t.Fatal(err)
	}
	pages := web.NewHandlers(viewstest.NewFake(), rd)
	r := chi.NewRouter()
	r.Use(p.Middleware)
	r.Handle("/api/*", api.New(nil, nil, api.Config{}).Handler())
	r.Get("/robots.txt", p.Robots)
	r.Get("/private", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex")
		w.WriteHeader(http.StatusOK)
	})
	pages.Routes(r)
	r.NotFound(pages.NotFound)
	return r
}

func fetch(t *testing.T, h http.Handler, target string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req = req.WithContext(views.WithCSRF(req.Context(), "tok-123"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	b, _ := io.ReadAll(res.Body)
	return res, string(b)
}

func TestRobotsTag(t *testing.T) {
	block, allow := site(t, Block), site(t, Allow)
	_, home := fetch(t, block, "/")
	css := regexp.MustCompile(`/static/style\.[0-9a-f]{8}\.css`).FindString(home)
	if css == "" {
		t.Fatal("no stylesheet link on the front page")
	}
	for _, c := range []struct{ name, path, ctype string }{
		{"page", "/latest", "text/html"},
		{"feed", "/latest.rss", "application/rss+xml"},
		{"api", "/api/config", "application/json"},
		{"api error", "/api/nothing", "application/json"},
		{"stylesheet", css, "text/css"},
		{"404 page", "/nothing", "text/html"},
		{"robots.txt", "/robots.txt", "text/plain"},
	} {
		res, _ := fetch(t, block, c.path)
		if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, c.ctype) {
			t.Errorf("%s %s: content type %q, want %s", c.name, c.path, ct, c.ctype)
		}
		if got := res.Header.Values("X-Robots-Tag"); len(got) != 1 || got[0] != RobotsTag {
			t.Errorf("block, %s %s: X-Robots-Tag %q, want %q", c.name, c.path, got, RobotsTag)
		}
		if res, _ := fetch(t, allow, c.path); res.Header.Get("X-Robots-Tag") != "" {
			t.Errorf("allow, %s %s: X-Robots-Tag %q, want none", c.name, c.path, res.Header.Get("X-Robots-Tag"))
		}
	}

	// A handler's own noindex does not weaken the block, and stays with allow.
	if res, _ := fetch(t, block, "/private"); res.Header.Get("X-Robots-Tag") != RobotsTag {
		t.Errorf("block, own tag: %q", res.Header.Get("X-Robots-Tag"))
	}
	if res, _ := fetch(t, allow, "/private"); res.Header.Get("X-Robots-Tag") != "noindex" {
		t.Errorf("allow, own tag: %q", res.Header.Get("X-Robots-Tag"))
	}
	// Pages that ask not to be indexed keep their meta tag with allow.
	if _, body := fetch(t, allow, "/nothing"); !strings.Contains(body, `<meta name="robots" content="noindex">`) {
		t.Error("allow: 404 page lost its noindex meta tag")
	}
}

// The upload route sets deadlines through http.ResponseController; that
// must still work through the middleware.
func TestResponseControllerThroughMiddleware(t *testing.T) {
	var rcErr error
	srv := httptest.NewServer(Block.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		rcErr = rc.SetReadDeadline(time.Now().Add(time.Minute))
		_, _ = w.Write([]byte("ok"))
		_ = rc.Flush()
	})))
	defer srv.Close()
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if rcErr != nil {
		t.Errorf("SetReadDeadline: %v", rcErr)
	}
	if res.Header.Get("X-Robots-Tag") != RobotsTag {
		t.Errorf("X-Robots-Tag %q", res.Header.Get("X-Robots-Tag"))
	}
}
