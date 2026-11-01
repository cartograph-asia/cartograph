package auth

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// inviteEnv is an env with SIGNUP=invite, admin@example.org as the admin,
// and /admin behind the guard.
func inviteEnv(t *testing.T) *env {
	t.Helper()
	e := newEnv(t, "http://cartograph.test")
	e.a.cfg.Signup = SignupInvite
	e.a.cfg.Admins = map[string]bool{"admin@example.org": true}
	e.router.With(e.a.AdminGuard).Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "admin:"+views.ViewerFrom(r.Context()).Name)
	})
	return e
}

// signUp signs in email and chooses name; it returns the status of the
// name form.
func (e *env) signUp(email, name string) int {
	e.t.Helper()
	tok := e.requestLink(email, "/")
	want(e.t, e.redeem(tok), http.StatusSeeOther, "/signin/name")
	want(e.t, e.get("/signin/name"), http.StatusOK, "")
	return e.post("/signin/name", url.Values{"name": {name}, "csrf": {e.csrf()}}).StatusCode
}

func (e *env) adminViewer() *views.Viewer {
	e.t.Helper()
	for _, s := range e.store.surveyors {
		if s.email == "admin@example.org" {
			return &views.Viewer{SurveyorID: s.id, Name: s.name, Email: s.email, Admin: true}
		}
	}
	e.t.Fatal("no admin")
	return nil
}

func TestInviteOnlySignIn(t *testing.T) {
	e := inviteEnv(t)
	e.store.addSurveyor("ada", "ada@example.org", false)

	// An existing Surveyor and the admin address get links.
	e.requestLink("ada@example.org", "/")
	if e.signUp("admin@example.org", "boss") != http.StatusSeeOther {
		t.Fatalf("admin sign-up: %+v", e.last())
	}
	want(t, e.get("/settings"), http.StatusOK, "")
	e.post("/signout", url.Values{"csrf": {e.csrf()}})

	// A stranger gets the same page, the notice instead of a link, once a day.
	for range 3 {
		want(t, e.get("/signin"), http.StatusOK, "")
		if si := e.last().data.(*views.SignIn); !si.InviteOnly {
			t.Fatal("sign-in page does not say invitation-only")
		}
		resp := e.post("/signin", url.Values{"email": {"Stranger@Example.org"}, "csrf": {e.csrf()}})
		if resp.StatusCode != http.StatusOK || e.last().page != views.PageCheckEmail {
			t.Fatalf("stranger: %d %s", resp.StatusCode, e.last().page)
		}
	}
	if len(e.mail.links["stranger@example.org"]) != 0 || e.mail.notices["stranger@example.org"] != 1 {
		t.Fatalf("stranger got links %v, notices %d", e.mail.links["stranger@example.org"], e.mail.notices["stranger@example.org"])
	}
	e.advance(25 * time.Hour)
	e.post("/signin", url.Values{"email": {"stranger@example.org"}, "csrf": {e.csrf()}})
	if e.mail.notices["stranger@example.org"] != 2 {
		t.Fatalf("notice after a day: %d", e.mail.notices["stranger@example.org"])
	}

	// A link sent before the switch does not make an account.
	e.a.cfg.Signup = SignupOpen
	tok := e.requestLink("early@example.org", "/")
	e.a.cfg.Signup = SignupInvite
	want(t, e.redeem(tok), http.StatusSeeOther, "/signin/name")
	want(t, e.get("/signin/name"), http.StatusOK, "")
	if resp := e.post("/signin/name", url.Values{"name": {"early"}, "csrf": {e.csrf()}}); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("name without invitation: %d", resp.StatusCode)
	}
	if cn := e.last().data.(*views.ChooseName); cn.Error != msgNotInvited {
		t.Fatalf("name form error %q", cn.Error)
	}
	if e.store.byEmail("early@example.org") != nil {
		t.Fatal("account created without invitation")
	}
}

func TestInvitations(t *testing.T) {
	e := inviteEnv(t)
	e.signUp("admin@example.org", "boss")
	admin := e.adminViewer()
	ctx := t.Context()

	if err := e.a.Invite(ctx, &views.Viewer{Name: "x"}, "new@example.org", ""); err == nil {
		t.Fatal("a non-admin could invite")
	}
	if err := e.a.Invite(ctx, admin, "not an address", ""); !errors.Is(err, ErrBadEmail) {
		t.Fatalf("bad address: %v", err)
	}
	var ae *AccountError
	if err := e.a.Invite(ctx, admin, "Admin@Example.org", ""); !errors.As(err, &ae) || ae.Name != "boss" {
		t.Fatalf("invite an account: %v", err)
	}
	if err := e.a.Invite(ctx, admin, " New@Example.org ", "Come and survey Jomtien."); err != nil {
		t.Fatal(err)
	}
	if got := e.mail.invites["new@example.org"]; len(got) != 1 || got[0] != "boss|Come and survey Jomtien." {
		t.Fatalf("invitation mail %v", got)
	}
	// The invitation's link lives a week, not 15 minutes.
	link, _ := url.Parse(e.mail.last(t, "new@example.org"))
	e.advance(6 * 24 * time.Hour)
	if err := e.a.ResendInvite(ctx, admin, "new@example.org"); err != nil {
		t.Fatal(err)
	}
	if got := e.mail.invites["new@example.org"]; len(got) != 2 || got[1] != "boss|" {
		t.Fatalf("resend mail %v", got)
	}
	want(t, e.post("/signout", url.Values{"csrf": {e.csrf()}}), http.StatusSeeOther, "/")
	want(t, e.redeem(link.Query().Get("token")), http.StatusSeeOther, "/signin/name")
	want(t, e.get("/signin/name"), http.StatusOK, "")
	want(t, e.post("/signin/name", url.Values{"name": {"newbie"}, "csrf": {e.csrf()}}), http.StatusSeeOther, "/")
	inv := e.store.invites["new@example.org"]
	if inv.AcceptedAt == nil || inv.AcceptedAs != "newbie" {
		t.Fatalf("invitation not accepted: %+v", inv)
	}
	if err := e.a.ResendInvite(ctx, admin, "new@example.org"); err == nil {
		t.Fatal("resend for an accepted invitation")
	}

	// Revoking stops the links that were sent.
	if err := e.a.Invite(ctx, admin, "late@example.org", ""); err != nil {
		t.Fatal(err)
	}
	late, _ := url.Parse(e.mail.last(t, "late@example.org"))
	if err := e.a.RevokeInvite(ctx, admin, "late@example.org"); err != nil {
		t.Fatal(err)
	}
	if err := e.a.RevokeInvite(ctx, admin, "late@example.org"); !errors.Is(err, ErrNotPending) {
		t.Fatalf("revoke twice: %v", err)
	}
	e.post("/signout", url.Values{"csrf": {e.csrf()}})
	want(t, e.redeem(late.Query().Get("token")), http.StatusBadRequest, "")
	if len(e.store.actions) != 4 {
		t.Fatalf("audit rows %v", e.store.actions)
	}
	// The throttle covers invitation links too.
	for range linksPerAddress {
		_ = e.a.Invite(ctx, admin, "busy@example.org", "")
	}
	if err := e.a.Invite(ctx, admin, "busy@example.org", ""); !errors.Is(err, ErrTooManyLinks) {
		t.Fatalf("throttle: %v", err)
	}
}

func TestBlockedAccount(t *testing.T) {
	e := inviteEnv(t)
	s := e.store.addSurveyor("ploy88", "ploy@example.org", false)
	tok := e.requestLink("ploy@example.org", "/")
	want(t, e.redeem(tok), http.StatusSeeOther, "/")
	s.blocked = true
	// The open session ends at once.
	if status, _ := e.me(); status != http.StatusUnauthorized {
		t.Fatalf("blocked session still works: %d", status)
	}
	tok = e.requestLink("ploy@example.org", "/")
	if resp := e.redeem(tok); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("blocked sign-in: %d", resp.StatusCode)
	}
	if er := e.last().data.(*views.Error); er.Message != "This account is blocked. Write to hello@cartograph.asia." {
		t.Fatalf("blocked page %q", er.Message)
	}
}

func TestAdminGuard(t *testing.T) {
	e := inviteEnv(t)
	check := func(status int, page string) {
		t.Helper()
		resp := e.get("/admin")
		if resp.StatusCode != status || (page != "" && e.last().page != page) {
			t.Fatalf("/admin: %d %s, want %d %s", resp.StatusCode, e.last().page, status, page)
		}
		for k, v := range map[string]string{"Cache-Control": "no-store", "Referrer-Policy": "no-referrer", "X-Robots-Tag": "noindex"} {
			if got := resp.Header.Get(k); got != v {
				t.Errorf("%s: %q", k, got)
			}
		}
	}
	check(http.StatusNotFound, views.PageError) // anonymous
	e.store.addSurveyor("ada", "ada@example.org", false)
	want(t, e.redeem(e.requestLink("ada@example.org", "/")), http.StatusSeeOther, "/")
	check(http.StatusNotFound, views.PageError) // not an admin
	e.post("/signout", url.Values{"csrf": {e.csrf()}})

	tok := e.requestLink("admin@example.org", "/")
	want(t, e.redeem(tok), http.StatusSeeOther, "/signin/name")
	check(http.StatusNotFound, views.PageError) // admin address without a name
	want(t, e.get("/signin/name"), http.StatusOK, "")
	e.post("/signin/name", url.Values{"name": {"boss"}, "csrf": {e.csrf()}})
	check(http.StatusOK, "")

	e.advance(25 * time.Hour)
	check(http.StatusForbidden, views.PageSignInAgain)
	again := e.last().data.(*views.SignInAgain)
	if again.SignIn != "/signin?next=%2Fadmin" {
		t.Fatalf("sign in again → %q", again.SignIn)
	}
	want(t, e.post("/signout", url.Values{"csrf": {e.csrf()}, "next": {again.SignIn}}), http.StatusSeeOther, again.SignIn)
}

func TestAdminConfig(t *testing.T) {
	env := func(kv ...string) func(string) string {
		m := map[string]string{"MAIL_DEV_LOG": "1"}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return func(k string) string { return m[k] }
	}
	cfg, err := ConfigFromEnv(env())
	if err != nil || len(cfg.Admins) != 0 || cfg.Signup != SignupInvite || !cfg.InviteOnly() || cfg.AdminMaxAge != 24*time.Hour || cfg.InviteTTL != 168*time.Hour {
		t.Fatalf("defaults %+v %v", cfg, err)
	}
	cfg, err = ConfigFromEnv(env("ADMIN_EMAILS", " Adrian@Example.org, ,b@example.org ", "SIGNUP", "OPEN", "ADMIN_SESSION_MAX_AGE", "2h",
		"MAIL_FROM", "cartograph.asia <hello@cartograph.asia>"))
	if err != nil || !cfg.IsAdmin("adrian@example.org") || !cfg.IsAdmin("b@example.org") || len(cfg.Admins) != 2 ||
		cfg.InviteOnly() || cfg.AdminMaxAge != 2*time.Hour || cfg.Contact != "hello@cartograph.asia" {
		t.Fatalf("set %+v %v", cfg, err)
	}
	for _, kv := range [][]string{{"ADMIN_EMAILS", "adrian"}, {"SIGNUP", "closed"}, {"ADMIN_SESSION_MAX_AGE", "1 day"}, {"INVITE_TTL", "-1h"}} {
		if _, err := ConfigFromEnv(env(kv...)); err == nil {
			t.Errorf("%v accepted", kv)
		}
	}
}
