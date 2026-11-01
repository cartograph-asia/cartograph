package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// Errors of the invitation actions, for the admin area's messages.
var (
	ErrBadEmail     = errors.New("auth: not an e-mail address")
	ErrTooManyLinks = errors.New("auth: too many links for this address")
	ErrNoteTooLong  = fmt.Errorf("auth: the note is longer than %d characters", views.InviteNoteMax)
	// ErrInviteNotSent: the invitation is saved but its mail failed.
	ErrInviteNotSent = errors.New("auth: invitation saved, mail not sent")
	errNotAdminInput = errors.New("auth: inviter is not a named admin")
)

// AdminGuard protects the admin area: anonymous, nameless and non-admin
// viewers get the ordinary 404 page, so the area does not announce itself;
// an admin whose magic link is older than AdminMaxAge is asked to sign in
// again. Every response is private, unindexed and sends no referrer.
func (a *Auth) AdminGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Robots-Tag", "noindex")
		v := views.ViewerFrom(r.Context())
		if !v.IsAdmin() {
			a.render(w, r, http.StatusNotFound, views.PageError, &views.Error{
				Page: a.page(r, "Not found"), Status: http.StatusNotFound, Message: "There is nothing at this address.",
			})
			return
		}
		if a.now().Sub(v.SignedInAt) > a.cfg.AdminMaxAge {
			next := "/signin?next=" + url.QueryEscape(safeNext(r.URL.RequestURI()))
			p := &views.SignInAgain{Page: a.page(r, "Sign in again"), SignIn: next}
			a.render(w, r, http.StatusForbidden, views.PageSignInAgain, p)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// InviteOnly reports whether sign-up needs an invitation (SIGNUP=invite).
func (a *Auth) InviteOnly() bool { return a.cfg.InviteOnly() }

// Invite invites email on behalf of admin and sends the invitation mail
// with a magic link that lives InviteTTL. Inviting a pending address again
// replaces its note and sends a new link. An address with an account is an
// *AccountError.
func (a *Auth) Invite(ctx context.Context, admin *views.Viewer, email, note string) error {
	return a.invite(ctx, admin, email, note, false)
}

// ResendInvite sends a new link for a pending invitation; ErrNotPending if
// there is none.
func (a *Auth) ResendInvite(ctx context.Context, admin *views.Viewer, email string) error {
	return a.invite(ctx, admin, email, "", true)
}

func (a *Auth) invite(ctx context.Context, admin *views.Viewer, raw, note string, resend bool) error {
	if !admin.IsAdmin() {
		return errNotAdminInput
	}
	email, ok := normalizeEmail(raw)
	if !ok {
		return ErrBadEmail
	}
	note = strings.TrimSpace(strings.ReplaceAll(note, "\r", ""))
	if len([]rune(note)) > views.InviteNoteMax {
		return ErrNoteTooLong
	}
	now := a.now()
	n, err := a.store.CountLoginTokens(ctx, email, now.Add(-throttleWindow))
	if err != nil {
		return err
	}
	if n >= linksPerAddress {
		return ErrTooManyLinks
	}
	token := newToken()
	err = a.store.CreateInvite(ctx, Invite{
		Email: email, AdminID: admin.SurveyorID, Note: note, Resend: resend, CreatedAt: now,
		Token: LoginToken{Hash: hashToken(token), Email: email, Next: "/", CreatedAt: now, ExpiresAt: now.Add(a.cfg.InviteTTL)},
	})
	if err != nil {
		return err
	}
	if resend {
		// The note of the first invitation is not repeated on a resend.
		note = ""
	}
	link := a.cfg.BaseURL + "/signin/link?token=" + encodeToken(token)
	mctx, cancel := context.WithTimeout(ctx, mailTimeout)
	defer cancel()
	if err := a.mail.SendInvitation(mctx, email, admin.Name, note, link, a.cfg.InviteTTL); err != nil {
		return fmt.Errorf("%w: %v", ErrInviteNotSent, err)
	}
	return nil
}

// RevokeInvite revokes a pending invitation; its unused links stop working.
func (a *Auth) RevokeInvite(ctx context.Context, admin *views.Viewer, raw string) error {
	if !admin.IsAdmin() {
		return errNotAdminInput
	}
	email, ok := normalizeEmail(raw)
	if !ok {
		return ErrBadEmail
	}
	return a.store.RevokeInvite(ctx, admin.SurveyorID, email, a.now())
}

// Invitations lists every invitation, newest first.
func (a *Auth) Invitations(ctx context.Context) ([]views.Invitation, error) {
	return a.store.Invitations(ctx)
}
