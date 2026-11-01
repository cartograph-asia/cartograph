// Package auth is the minimum account system of the Home brief: e-mail magic
// links, a session cookie, the one-time Surveyor name, the profile form and
// GET /api/me. There are no passwords, no OAuth and no roles.
//
// Routes (mounted by Routes):
//
//	GET  /signin        e-mail form (?next=/local/path)
//	POST /signin        fields email, next, csrf → "check your e-mail"
//	GET  /signin/link   ?token=… from the e-mail: a page with one button
//	POST /signin/link   fields token, csrf → session cookie, redirect
//	GET  /signin/name   one-time Surveyor name form (?next=…)
//	POST /signin/name   fields name, next, csrf
//	POST /signout       field csrf
//	GET  /settings      profile form (bio, home region)
//	POST /settings      fields bio, home_region, csrf
//	GET  /api/me        {"name": …} or 401 {"error": …}
//
// Middleware runs in front of every route (pages and API): it loads the
// session, puts the *views.Viewer and the CSRF token into the request
// context (views.WithViewer, views.WithCSRF) and slides the session expiry.
//
// CSRF: every POST form carries views.CSRFField. The token is an HMAC of the
// visitor's session token (or, before sign-in, of a cg_csrf cookie set only
// by the sign-in pages), so nothing is stored for it. In addition every
// state-changing request must pass net/http's CrossOriginProtection
// (Sec-Fetch-Site / Origin), which is also what guards POST /api/photos.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// Lifetimes and limits (see docs/decisions.md, Home).
const (
	LinkTTL            = 15 * time.Minute    // a magic link works once, within this time
	SessionTTL         = 30 * 24 * time.Hour // a session ends after this long without a visit
	sessionTouchEvery  = time.Hour           // last_seen_at/expires_at are written at most this often
	linksPerAddress    = 5                   // magic links per e-mail address per hour
	signInsPerIP       = 20                  // sign-in form posts per IP (IPv6: /64) per hour
	throttleWindow     = time.Hour
	maxFormBytes       = 16 << 10
	mailTimeout        = 20 * time.Second
	sessionCookieName  = "cg_session"
	formCookieName     = "cg_csrf"
	secureCookiePrefix = "__Host-" // used when BASE_URL is https
)

// Config holds the settings of the account system.
type Config struct {
	// BaseURL is the public origin without a trailing slash, e.g.
	// "https://cartograph.asia". Magic links point here; an https BaseURL
	// makes cookies Secure.
	BaseURL string
	SMTP    SMTPConfig
}

// ConfigFromEnv reads BASE_URL and the SMTP_* / MAIL_FROM variables.
// getenv is os.Getenv in production.
func ConfigFromEnv(getenv func(string) string) (Config, error) {
	cfg := Config{BaseURL: strings.TrimRight(strings.TrimSpace(getenv("BASE_URL")), "/")}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8080"
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || (u.Path != "" && u.Path != "/") {
		return Config{}, fmt.Errorf("auth: BASE_URL must be an http(s) origin such as https://cartograph.asia, got %q", cfg.BaseURL)
	}
	cfg.SMTP = SMTPConfig{
		Host:     strings.TrimSpace(getenv("SMTP_HOST")),
		Port:     587,
		User:     getenv("SMTP_USER"),
		Password: getenv("SMTP_PASSWORD"),
		From:     strings.TrimSpace(getenv("MAIL_FROM")),
	}
	if p := strings.TrimSpace(getenv("SMTP_PORT")); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 || n > 65535 {
			return Config{}, fmt.Errorf("auth: SMTP_PORT must be a port number, got %q", p)
		}
		cfg.SMTP.Port = n
	}
	// Without SMTP_HOST the dev mailer writes every sign-in link to the log,
	// where anyone who reads the log can use it. That is fine on a laptop,
	// never on a public https origin.
	if cfg.SMTP.Host == "" && u.Scheme == "https" {
		return Config{}, fmt.Errorf("auth: SMTP_HOST is required when BASE_URL is https (%s); without it sign-in links would be written to the log instead of sent", cfg.BaseURL)
	}
	if cfg.SMTP.Host != "" {
		if cfg.SMTP.From == "" {
			return Config{}, errors.New("auth: MAIL_FROM is required when SMTP_HOST is set")
		}
		if _, err := mail.ParseAddress(cfg.SMTP.From); err != nil {
			return Config{}, fmt.Errorf("auth: MAIL_FROM is not an address: %w", err)
		}
	}
	return cfg, nil
}

// RegionLister lists the options of the home-region select; views.Source
// (and so *db.DB) implements it.
type RegionLister interface {
	RegionRefs(ctx context.Context) ([]views.RegionRef, error)
}

// Auth serves the account routes and the session middleware.
type Auth struct {
	cfg     Config
	store   Store
	mail    Mailer
	render  views.RenderFunc
	regions RegionLister
	cop     *http.CrossOriginProtection
	ipLimit *limiter
	secure  bool
	now     func() time.Time
}

// New creates the account system. mailer is usually NewMailer(cfg.SMTP).
func New(cfg Config, store Store, mailer Mailer, render views.RenderFunc, regions RegionLister) *Auth {
	return &Auth{
		cfg:     cfg,
		store:   store,
		mail:    mailer,
		render:  render,
		regions: regions,
		cop:     http.NewCrossOriginProtection(),
		ipLimit: newLimiter(signInsPerIP, throttleWindow),
		secure:  strings.HasPrefix(cfg.BaseURL, "https://"),
		now:     time.Now,
	}
}

// Routes mounts the account routes on r. Middleware must run in front of
// them (and of every other route).
func (a *Auth) Routes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(privateHeaders)
		r.Get("/signin", a.signInForm)
		r.Post("/signin", a.signIn)
		r.Get("/signin/link", a.confirmForm)
		r.Post("/signin/link", a.confirm)
		r.Get("/signin/name", a.nameForm)
		r.Post("/signin/name", a.chooseName)
		r.Post("/signout", a.signOut)
		r.Get("/settings", a.settingsForm)
		r.Post("/settings", a.saveSettings)
		r.Get("/api/me", a.me)
	})
}

type sessionKey struct{}

// Middleware loads the visitor's session and stores the viewer and the CSRF
// token in the request context. A database error is logged and the request
// continues anonymously: nothing on Cartograph needs an account to view.
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var viewer *views.Viewer
		csrf := ""
		if raw, ok := a.readCookie(r, sessionCookieName); ok {
			h := hashToken(raw)
			now := a.now()
			s, err := a.store.Session(ctx, h, now)
			switch {
			case err == nil:
				viewer = &s.Viewer
				csrf = csrfToken(raw)
				ctx = context.WithValue(ctx, sessionKey{}, h)
				if now.Sub(s.LastSeen) >= sessionTouchEvery {
					exp := now.Add(SessionTTL)
					if err := a.store.TouchSession(ctx, h, now, exp); err != nil {
						slog.Warn("touch session", "err", err)
					} else {
						a.setCookie(w, sessionCookieName, raw, exp)
					}
				}
			case errors.Is(err, ErrNotFound):
				a.clearCookie(w, sessionCookieName)
			default:
				slog.Warn("load session", "err", err)
			}
		}
		if csrf == "" {
			if raw, ok := a.readCookie(r, formCookieName); ok {
				csrf = csrfToken(raw)
			}
		}
		ctx = views.WithViewer(ctx, viewer)
		ctx = views.WithCSRF(ctx, csrf)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func sessionHashFrom(ctx context.Context) []byte {
	h, _ := ctx.Value(sessionKey{}).([]byte)
	return h
}

// csrfToken derives the form token from a session or form-cookie token.
func csrfToken(raw []byte) string {
	m := hmac.New(sha256.New, raw)
	m.Write([]byte("cartograph csrf v1"))
	return encodeToken(m.Sum(nil))
}

// formOK checks a state-changing form: same origin and a valid CSRF field.
// It renders a 403 page and returns false otherwise.
func (a *Auth) formOK(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseForm(); err != nil {
		a.fail(w, r, http.StatusBadRequest, "The form could not be read. Please go back and try again.")
		return false
	}
	want := views.CSRFFrom(r.Context())
	got := r.PostForm.Get(views.CSRFField)
	if a.cop.Check(r) != nil || want == "" || !hmac.Equal([]byte(want), []byte(got)) {
		a.fail(w, r, http.StatusForbidden, "This form has expired. Please go back, reload the page and try again.")
		return false
	}
	return true
}

func (a *Auth) cookieName(name string) string {
	if a.secure {
		return secureCookiePrefix + name
	}
	return name
}

func (a *Auth) readCookie(r *http.Request, name string) ([]byte, bool) {
	c, err := r.Cookie(a.cookieName(name))
	if err != nil {
		return nil, false
	}
	return decodeToken(c.Value)
}

// setCookie sets an HttpOnly, SameSite=Lax cookie; a zero exp makes it a
// browser-session cookie.
func (a *Auth) setCookie(w http.ResponseWriter, name string, raw []byte, exp time.Time) {
	c := &http.Cookie{
		Name:     a.cookieName(name),
		Value:    encodeToken(raw),
		Path:     "/",
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
	}
	if !exp.IsZero() {
		c.Expires = exp
		c.MaxAge = int(exp.Sub(a.now()).Seconds())
		if c.MaxAge <= 0 {
			c.MaxAge = 1
		}
	}
	http.SetCookie(w, c)
}

func (a *Auth) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name: a.cookieName(name), Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteLaxMode,
	})
}

// ensureFormCookie gives an anonymous visitor the cookie that backs the CSRF
// token of the sign-in forms. It is set only on the sign-in pages, so plain
// visitors get no cookie at all.
func (a *Auth) ensureFormCookie(w http.ResponseWriter, r *http.Request) *http.Request {
	if views.CSRFFrom(r.Context()) != "" {
		return r
	}
	raw := newToken()
	a.setCookie(w, formCookieName, raw, time.Time{})
	return r.WithContext(views.WithCSRF(r.Context(), csrfToken(raw)))
}

// privateHeaders keeps account pages out of caches and keeps tokens in URLs
// out of Referer headers.
func privateHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("write response", "err", err)
	}
}
