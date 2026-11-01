package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// Plain-text messages shown on the account pages.
const (
	msgBadEmail    = "Please enter a valid e-mail address."
	msgBadLink     = "This sign-in link is not valid. Enter your e-mail address for a new one."
	msgUsedLink    = "This sign-in link has expired or was already used. Enter your e-mail address for a new one."
	msgBadName     = "A Surveyor name has 3 to 30 characters: letters A–Z, digits and hyphens."
	msgNameTaken   = "That name is already taken. Please choose another one."
	msgBioTooLong  = "Your bio can be at most 500 characters."
	msgBadRegion   = "Please choose a region from the list."
	msgTooMany     = "There were too many sign-in requests from your network. Please try again in an hour."
	msgMailFailed  = "We could not send the e-mail just now. Please try again in a few minutes."
	msgServerError = "Something went wrong on our side. Please try again."
)

// page returns the common page base for an account page.
func (a *Auth) page(r *http.Request, title string) views.Page {
	return views.Page{
		Viewer:    views.ViewerFrom(r.Context()),
		CSRF:      views.CSRFFrom(r.Context()),
		Now:       a.now(),
		Title:     title,
		Canonical: r.URL.Path,
		NoIndex:   true,
	}
}

func (a *Auth) fail(w http.ResponseWriter, r *http.Request, status int, msg string) {
	title := "Error"
	switch status {
	case http.StatusForbidden:
		title = "Form expired"
	case http.StatusTooManyRequests:
		title = "Too many requests"
	}
	a.render(w, r, status, views.PageError, &views.Error{Page: a.page(r, title), Status: status, Message: msg})
}

func (a *Auth) internalError(w http.ResponseWriter, r *http.Request, what string, err error) {
	slog.Error(what, "err", err)
	a.fail(w, r, http.StatusInternalServerError, msgServerError)
}

func redirect(w http.ResponseWriter, r *http.Request, to string) {
	http.Redirect(w, r, to, http.StatusSeeOther)
}

func withNext(path, next string) string {
	if next == "" || next == "/" {
		return path
	}
	return path + "?next=" + url.QueryEscape(next)
}

// signInForm handles GET /signin.
func (a *Auth) signInForm(w http.ResponseWriter, r *http.Request) {
	next := safeNext(r.URL.Query().Get("next"))
	v := views.ViewerFrom(r.Context())
	if v.HasName() {
		redirect(w, r, next)
		return
	}
	if v != nil {
		redirect(w, r, withNext("/signin/name", next))
		return
	}
	r = a.ensureFormCookie(w, r)
	a.render(w, r, http.StatusOK, views.PageSignIn, &views.SignIn{Page: a.page(r, "Sign in"), Next: next})
}

// signIn handles POST /signin. Whether or not the address belongs to a
// Surveyor, the answer is the same "check your e-mail" page.
func (a *Auth) signIn(w http.ResponseWriter, r *http.Request) {
	if !a.formOK(w, r) {
		return
	}
	ctx := r.Context()
	now := a.now()
	next := safeNext(r.PostForm.Get("next"))
	raw := r.PostForm.Get("email")
	if !a.ipLimit.allow(clientIP(r), now) {
		slog.Info("sign-in throttled", "reason", "ip")
		a.fail(w, r, http.StatusTooManyRequests, msgTooMany)
		return
	}
	email, ok := normalizeEmail(raw)
	if !ok {
		a.render(w, r, http.StatusBadRequest, views.PageSignIn, &views.SignIn{
			Page: a.page(r, "Sign in"), Email: strings.TrimSpace(raw), Next: next, Error: msgBadEmail,
		})
		return
	}
	checkEmail := func() {
		a.render(w, r, http.StatusOK, views.PageCheckEmail, &views.CheckEmail{Page: a.page(r, "Check your e-mail"), Email: email})
	}

	n, err := a.store.CountLoginTokens(ctx, email, now.Add(-throttleWindow))
	if err != nil {
		a.internalError(w, r, "count login tokens", err)
		return
	}
	if n >= linksPerAddress {
		// The earlier links still work; answer as usual without sending.
		slog.Info("sign-in throttled", "reason", "address")
		checkEmail()
		return
	}

	token := newToken()
	err = a.store.CreateLoginToken(ctx, LoginToken{
		Hash: hashToken(token), Email: email, Next: next, CreatedAt: now, ExpiresAt: now.Add(LinkTTL),
	})
	if err != nil {
		a.internalError(w, r, "create login token", err)
		return
	}
	link := a.cfg.BaseURL + "/signin/link?token=" + encodeToken(token)
	mctx, cancel := context.WithTimeout(ctx, mailTimeout)
	defer cancel()
	if err := a.mail.SendSignInLink(mctx, email, link); err != nil {
		slog.Error("send sign-in link", "err", err)
		a.fail(w, r, http.StatusServiceUnavailable, msgMailFailed)
		return
	}
	checkEmail()
}

// confirmForm handles GET /signin/link: it never uses the token, because
// mail scanners fetch links before the recipient does. The page has one
// button that POSTs the token.
func (a *Auth) confirmForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	r = a.ensureFormCookie(w, r)
	if _, ok := decodeToken(token); !ok {
		a.render(w, r, http.StatusBadRequest, views.PageSignIn, &views.SignIn{Page: a.page(r, "Sign in"), Next: "/", Error: msgBadLink})
		return
	}
	a.render(w, r, http.StatusOK, views.PageConfirmSignIn, &views.ConfirmSignIn{Page: a.page(r, "Sign in"), Token: token})
}

// confirm handles POST /signin/link: it uses the token and starts a session.
func (a *Auth) confirm(w http.ResponseWriter, r *http.Request) {
	if !a.formOK(w, r) {
		return
	}
	ctx := r.Context()
	now := a.now()
	raw, ok := decodeToken(r.PostForm.Get("token"))
	if !ok {
		a.render(w, r, http.StatusBadRequest, views.PageSignIn, &views.SignIn{Page: a.page(r, "Sign in"), Next: "/", Error: msgBadLink})
		return
	}
	email, next, err := a.store.ConsumeLoginToken(ctx, hashToken(raw), now)
	if errors.Is(err, ErrNotFound) {
		a.render(w, r, http.StatusBadRequest, views.PageSignIn, &views.SignIn{Page: a.page(r, "Sign in"), Next: "/", Error: msgUsedLink})
		return
	}
	if err != nil {
		a.internalError(w, r, "consume login token", err)
		return
	}
	next = safeNext(next)

	session := newToken()
	exp := now.Add(SessionTTL)
	v, err := a.store.CreateSession(ctx, hashToken(session), email, now, exp)
	if err != nil {
		a.internalError(w, r, "create session", err)
		return
	}
	if old := sessionHashFrom(ctx); old != nil {
		if err := a.store.DeleteSession(ctx, old); err != nil {
			slog.Warn("delete previous session", "err", err)
		}
	}
	a.setCookie(w, sessionCookieName, session, exp)
	a.clearCookie(w, formCookieName)
	slog.Info("signed in", "named", v.HasName())
	if !v.HasName() {
		redirect(w, r, withNext("/signin/name", next))
		return
	}
	redirect(w, r, next)
}

// nameForm handles GET /signin/name.
func (a *Auth) nameForm(w http.ResponseWriter, r *http.Request) {
	next := safeNext(r.URL.Query().Get("next"))
	v := views.ViewerFrom(r.Context())
	if v == nil {
		redirect(w, r, withNext("/signin", next))
		return
	}
	if v.HasName() {
		redirect(w, r, next)
		return
	}
	a.render(w, r, http.StatusOK, views.PageChooseName, &views.ChooseName{Page: a.page(r, "Choose your Surveyor name"), Next: next})
}

// chooseName handles POST /signin/name. The name is chosen once and never
// changes.
func (a *Auth) chooseName(w http.ResponseWriter, r *http.Request) {
	if !a.formOK(w, r) {
		return
	}
	next := safeNext(r.PostForm.Get("next"))
	v := views.ViewerFrom(r.Context())
	if v == nil {
		redirect(w, r, withNext("/signin", next))
		return
	}
	if v.HasName() {
		redirect(w, r, next)
		return
	}
	name := strings.TrimSpace(r.PostForm.Get("name"))
	again := func(status int, msg string) {
		a.render(w, r, status, views.PageChooseName, &views.ChooseName{
			Page: a.page(r, "Choose your Surveyor name"), Name: name, Next: next, Error: msg,
		})
	}
	if !validName(name) {
		again(http.StatusBadRequest, msgBadName)
		return
	}
	nv, err := a.store.ChooseName(r.Context(), sessionHashFrom(r.Context()), name, a.now())
	switch {
	case errors.Is(err, ErrNameTaken):
		again(http.StatusConflict, msgNameTaken)
		return
	case errors.Is(err, ErrNotFound):
		redirect(w, r, withNext("/signin", next))
		return
	case err != nil:
		a.internalError(w, r, "choose name", err)
		return
	}
	slog.Info("surveyor joined", "name", nv.Name)
	redirect(w, r, next)
}

// signOut handles POST /signout.
func (a *Auth) signOut(w http.ResponseWriter, r *http.Request) {
	if h := sessionHashFrom(r.Context()); h != nil {
		if !a.formOK(w, r) {
			return
		}
		if err := a.store.DeleteSession(r.Context(), h); err != nil {
			a.internalError(w, r, "sign out", err)
			return
		}
	}
	a.clearCookie(w, sessionCookieName)
	redirect(w, r, "/")
}

// settingsForm handles GET /settings, the profile form.
func (a *Auth) settingsForm(w http.ResponseWriter, r *http.Request) {
	v, ok := a.requireName(w, r, "/settings")
	if !ok {
		return
	}
	bio, home, err := a.store.Profile(r.Context(), v.SurveyorID)
	if err != nil {
		a.internalError(w, r, "load profile", err)
		return
	}
	regions, err := a.regions.RegionRefs(r.Context())
	if err != nil {
		a.internalError(w, r, "load regions", err)
		return
	}
	a.render(w, r, http.StatusOK, views.PageEditProfile, &views.EditProfile{
		Page: a.page(r, "Your profile"), Name: v.Name, Bio: bio, HomeRegion: home, Regions: regions,
		Saved: r.URL.Query().Get("saved") == "1",
	})
}

// saveSettings handles POST /settings.
func (a *Auth) saveSettings(w http.ResponseWriter, r *http.Request) {
	v, ok := a.requireName(w, r, "/settings")
	if !ok {
		return
	}
	if !a.formOK(w, r) {
		return
	}
	ctx := r.Context()
	regions, err := a.regions.RegionRefs(ctx)
	if err != nil {
		a.internalError(w, r, "load regions", err)
		return
	}
	rawBio := r.PostForm.Get("bio")
	home := strings.TrimSpace(r.PostForm.Get("home_region"))
	again := func(msg string) {
		a.render(w, r, http.StatusBadRequest, views.PageEditProfile, &views.EditProfile{
			Page: a.page(r, "Your profile"), Name: v.Name, Bio: rawBio, HomeRegion: home, Regions: regions, Error: msg,
		})
	}
	bio, ok := cleanBio(rawBio)
	if !ok {
		again(msgBioTooLong)
		return
	}
	if home != "" && !hasRegion(regions, home) {
		again(msgBadRegion)
		return
	}
	if err := a.store.UpdateProfile(ctx, v.SurveyorID, bio, home); err != nil {
		a.internalError(w, r, "update profile", err)
		return
	}
	redirect(w, r, "/settings?saved=1")
}

func hasRegion(regions []views.RegionRef, slug string) bool {
	for _, r := range regions {
		if r.Slug == slug {
			return true
		}
	}
	return false
}

// requireName redirects visitors without a Surveyor name to sign-in or to
// the name form, returning to path afterwards.
func (a *Auth) requireName(w http.ResponseWriter, r *http.Request, path string) (*views.Viewer, bool) {
	v := views.ViewerFrom(r.Context())
	switch {
	case v == nil:
		redirect(w, r, withNext("/signin", path))
		return nil, false
	case !v.HasName():
		redirect(w, r, withNext("/signin/name", path))
		return nil, false
	}
	return v, true
}

// me handles GET /api/me for the map page.
func (a *Auth) me(w http.ResponseWriter, r *http.Request) {
	v := views.ViewerFrom(r.Context())
	switch {
	case v == nil:
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "not signed in"})
	case !v.HasName():
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "choose a Surveyor name first"})
	default:
		writeJSON(w, http.StatusOK, map[string]string{"name": v.Name})
	}
}
