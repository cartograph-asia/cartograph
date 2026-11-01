// Package crawlers keeps search engines and harvesters off the site while
// it is in alpha (CRAWLERS=block) and lets them in for the beta
// (CRAWLERS=allow) without a code change. It serves /robots.txt, the one
// source of the site's robots rules, and with block marks every response
// of the api with X-Robots-Tag. Caddy adds the same header to what it
// serves itself (deploy/Caddyfile).
package crawlers

import (
	"fmt"
	"net/http"
	"strings"
)

// Policy is the CRAWLERS setting.
type Policy string

const (
	Block Policy = "block" // default: robots.txt disallows everything, every response carries RobotsTag
	Allow Policy = "allow" // robots.txt keeps out the private areas and AI crawlers only
)

// RobotsTag is the X-Robots-Tag header every response carries with Block.
const RobotsTag = "noindex, nofollow, noarchive"

// AICrawlers are named in robots.txt with Disallow: / in both modes: the
// photos are CC BY-SA, but bulk harvesting for AI training was never agreed.
// The SEO crawlers at the end are kept out for the same reason.
var AICrawlers = []string{
	"GPTBot", "ChatGPT-User", "OAI-SearchBot",
	"ClaudeBot", "anthropic-ai",
	"Google-Extended", "CCBot", "PerplexityBot", "Bytespider",
	"Amazonbot", "Applebot-Extended", "meta-externalagent",
	"AhrefsBot", "SemrushBot", "MJ12bot",
}

// Private are the paths robots.txt keeps every crawler out of with Allow.
// /admin is not among them: it answers 404 to everyone but admins and
// sends noindex itself, and naming it would only advertise it
// (docs/ADMIN_BRIEF.md).
var Private = []string{"/settings", "/signin", "/api/"}

// FromEnv reads CRAWLERS: block (also when empty) or allow, case-insensitive.
func FromEnv(getenv func(string) string) (Policy, error) {
	switch v := strings.ToLower(strings.TrimSpace(getenv("CRAWLERS"))); v {
	case "", string(Block):
		return Block, nil
	case string(Allow):
		return Allow, nil
	default:
		return "", fmt.Errorf("CRAWLERS must be block or allow, got %q", v)
	}
}

// RobotsTXT is the body of /robots.txt.
func (p Policy) RobotsTXT() string {
	var b strings.Builder
	if p == Allow {
		b.WriteString("# cartograph.asia\n\nUser-agent: *\n")
		for _, path := range Private {
			b.WriteString("Disallow: " + path + "\n")
		}
	} else {
		b.WriteString("# cartograph.asia is in alpha: nothing here is to be indexed or harvested yet.\n\nUser-agent: *\nDisallow: /\n")
	}
	for _, ua := range AICrawlers {
		b.WriteString("\nUser-agent: " + ua + "\nDisallow: /\n")
	}
	return b.String()
}

// Robots serves GET /robots.txt.
func (p Policy) Robots(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/plain; charset=utf-8")
	h.Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(p.RobotsTXT()))
}

// Middleware sets X-Robots-Tag: RobotsTag on every response with Block and
// does nothing with Allow. The header is set again just before the status
// line goes out, so a handler that sets a weaker tag of its own (the admin
// guard's "noindex") cannot loosen it.
func (p Policy) Middleware(next http.Handler) http.Handler {
	if p != Block {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", RobotsTag)
		next.ServeHTTP(&taggedWriter{ResponseWriter: w}, r)
	})
}

// taggedWriter re-asserts X-Robots-Tag before the header is written. Unwrap
// keeps http.ResponseController working (the upload route sets its own
// deadlines through it).
type taggedWriter struct {
	http.ResponseWriter
	sent bool
}

func (w *taggedWriter) tag() {
	if !w.sent {
		w.sent = true
		w.ResponseWriter.Header().Set("X-Robots-Tag", RobotsTag)
	}
}

func (w *taggedWriter) WriteHeader(code int) {
	if code >= 200 {
		w.tag()
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *taggedWriter) Write(b []byte) (int, error) {
	w.tag()
	return w.ResponseWriter.Write(b)
}

func (w *taggedWriter) Flush() {
	w.tag()
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *taggedWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
