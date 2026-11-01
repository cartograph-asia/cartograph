package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// Errors returned by Store implementations.
var (
	ErrNotFound  = errors.New("auth: not found")
	ErrNameTaken = errors.New("auth: name taken")
	// ErrBlocked: the address belongs to a blocked Surveyor.
	ErrBlocked = errors.New("auth: account blocked")
	// ErrNotInvited: sign-up needs an invitation and the address has none.
	ErrNotInvited = errors.New("auth: not invited")
	// ErrNotPending: the invitation does not exist or was accepted or revoked.
	ErrNotPending = errors.New("auth: no pending invitation")
	// ErrDisplayTaken: the display name is another Surveyor's Surveyor name.
	ErrDisplayTaken = errors.New("auth: display name is another Surveyor's name")
)

// AccountError reports that an invited address is a Surveyor already.
type AccountError struct{ Name string }

func (e *AccountError) Error() string { return "auth: address belongs to " + e.Name }

// LoginToken is a magic link waiting to be used.
type LoginToken struct {
	Hash      []byte // SHA-256 of the token in the link
	Email     string // lower case
	Next      string // local path to continue to
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Session is a signed-in browser.
type Session struct {
	Viewer    views.Viewer
	CreatedAt time.Time // when the magic link was redeemed
	LastSeen  time.Time
}

// Profile is what a Surveyor edits in /settings.
type Profile struct {
	Display    string
	Bio        string
	HomeRegion string // region slug
}

// Invite is an invitation to write, with the magic link that goes with it.
type Invite struct {
	Email     string // normalised
	AdminID   string // surveyors.id of the inviting admin
	Note      string
	Token     LoginToken
	Resend    bool // a new link for a pending invitation; Note is kept
	CreatedAt time.Time
}

// Store is the persistence behind the account system; PGStore implements
// it on the tables of migrations 0005 and 0006.
type Store interface {
	// CreateLoginToken stores a magic link (and drops long-expired ones).
	CreateLoginToken(ctx context.Context, t LoginToken) error
	// CountLoginTokens counts the links created for email since since.
	CountLoginTokens(ctx context.Context, email string, since time.Time) (int, error)
	// ConsumeLoginToken marks an unused, unexpired link as used and returns
	// its e-mail and next path, or ErrNotFound.
	ConsumeLoginToken(ctx context.Context, hash []byte, now time.Time) (email, next string, err error)
	// CreateSession starts a session for email. If a Surveyor has that
	// e-mail the session belongs to them (and a placeholder account stops
	// being one); otherwise the viewer has no name yet. ErrBlocked if that
	// Surveyor is blocked.
	CreateSession(ctx context.Context, hash []byte, email string, now, expires time.Time) (*views.Viewer, error)
	// Session returns an unexpired session of an unblocked account, or ErrNotFound.
	Session(ctx context.Context, hash []byte, now time.Time) (*Session, error)
	// TouchSession records a visit and moves the expiry.
	TouchSession(ctx context.Context, hash []byte, now, expires time.Time) error
	// DeleteSession ends a session; a missing one is not an error.
	DeleteSession(ctx context.Context, hash []byte) error
	// ChooseName creates the Surveyor for the session's e-mail with name
	// (joined now), attaches every nameless session of that e-mail and
	// accepts a pending invitation of the address. With inviteOnly a new
	// account needs that invitation, else ErrNotInvited. It returns
	// ErrNameTaken when the name is taken regardless of case, and
	// ErrNotFound for an unknown or expired session. If the session already
	// has a Surveyor, that one is returned unchanged.
	// display is the optional display name, already cleaned; ErrDisplayTaken
	// if it is another Surveyor's name, ignoring case.
	ChooseName(ctx context.Context, hash []byte, name, display string, now time.Time, inviteOnly bool) (*views.Viewer, error)
	// MaySignIn reports whether email belongs to a Surveyor or has a pending
	// invitation.
	MaySignIn(ctx context.Context, email string) (bool, error)
	// RecordRefusal reports whether the invitation-only mail may go to the
	// address with this hash now (none sent within refusalEvery), and if so
	// records it as sent.
	RecordRefusal(ctx context.Context, emailHash []byte, now time.Time) (bool, error)
	// CreateInvite stores an invitation (or, with Resend, only a new link
	// for a pending one), its login token and the admin_actions row, in one
	// transaction. An address with an account is an *AccountError; a Resend
	// or a second invitation of a pending address follows Resend's rule: a
	// pending address without Resend is invited again (new link, new note).
	CreateInvite(ctx context.Context, inv Invite) error
	// RevokeInvite revokes a pending invitation, deletes the address's
	// unused links and nameless sessions, and writes the audit row.
	// ErrNotPending if there is none.
	RevokeInvite(ctx context.Context, adminID, email string, now time.Time) error
	// Invitations returns every invitation, newest first.
	Invitations(ctx context.Context) ([]views.Invitation, error)
	// Profile returns a Surveyor's display name, bio and home region slug
	// ("" for none).
	Profile(ctx context.Context, surveyorID string) (Profile, error)
	// UpdateProfile sets display name, bio and home region (slug); "" means
	// none. ErrDisplayTaken if the display name is another Surveyor's name.
	UpdateProfile(ctx context.Context, surveyorID string, p Profile) error
}

// PGStore is the Postgres Store.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore returns a Store on pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

// CreateLoginToken implements Store.
func (s *PGStore) CreateLoginToken(ctx context.Context, t LoginToken) error {
	if _, err := s.pool.Exec(ctx, "delete from login_tokens where expires_at < $1", t.CreatedAt.Add(-24*time.Hour)); err != nil {
		return fmt.Errorf("auth: prune login tokens: %w", err)
	}
	var next *string
	if t.Next != "" {
		next = &t.Next
	}
	_, err := s.pool.Exec(ctx, `
		insert into login_tokens (token_hash, email, next_path, created_at, expires_at)
		values ($1, $2, $3, $4, $5)`, t.Hash, t.Email, next, t.CreatedAt, t.ExpiresAt)
	if err != nil {
		return fmt.Errorf("auth: create login token: %w", err)
	}
	return nil
}

// CountLoginTokens implements Store.
func (s *PGStore) CountLoginTokens(ctx context.Context, email string, since time.Time) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, "select count(*) from login_tokens where email = $1 and created_at > $2", email, since).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("auth: count login tokens: %w", err)
	}
	return n, nil
}

// ConsumeLoginToken implements Store.
func (s *PGStore) ConsumeLoginToken(ctx context.Context, hash []byte, now time.Time) (string, string, error) {
	var email, next string
	err := s.pool.QueryRow(ctx, `
		update login_tokens set used_at = $2
		where token_hash = $1 and used_at is null and expires_at > $2
		returning email, coalesce(next_path, '')`, hash, now).Scan(&email, &next)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("auth: consume login token: %w", err)
	}
	return email, next, nil
}

// CreateSession implements Store.
func (s *PGStore) CreateSession(ctx context.Context, hash []byte, email string, now, expires time.Time) (*views.Viewer, error) {
	v := &views.Viewer{Email: email}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "delete from sessions where expires_at < $1", now); err != nil {
			return err
		}
		var placeholder, blocked bool
		err := tx.QueryRow(ctx, "select id::text, name, coalesce(display_name, ''), placeholder, blocked_at is not null from surveyors where lower(email) = $1", email).
			Scan(&v.SurveyorID, &v.Name, &v.Display, &placeholder, &blocked)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return err
		case blocked:
			return ErrBlocked
		case placeholder:
			// Adrian set this placeholder's e-mail in SQL; its first sign-in claims it.
			if _, err := tx.Exec(ctx, "update surveyors set placeholder = false where id = $1::uuid", v.SurveyorID); err != nil {
				return err
			}
		}
		var sid *string
		if v.SurveyorID != "" {
			sid = &v.SurveyorID
		}
		_, err = tx.Exec(ctx, `
			insert into sessions (token_hash, email, surveyor_id, created_at, expires_at, last_seen_at)
			values ($1, $2, $3::uuid, $4, $5, $4)`, hash, email, sid, now, expires)
		return err
	})
	if errors.Is(err, ErrBlocked) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("auth: create session: %w", err)
	}
	return v, nil
}

// Session implements Store.
func (s *PGStore) Session(ctx context.Context, hash []byte, now time.Time) (*Session, error) {
	var out Session
	err := s.pool.QueryRow(ctx, `
		select s.email, coalesce(u.id::text, ''), coalesce(u.name, ''), coalesce(u.display_name, ''), s.created_at, s.last_seen_at
		from sessions s left join surveyors u on u.id = s.surveyor_id
		where s.token_hash = $1 and s.expires_at > $2 and u.blocked_at is null`, hash, now).
		Scan(&out.Viewer.Email, &out.Viewer.SurveyorID, &out.Viewer.Name, &out.Viewer.Display, &out.CreatedAt, &out.LastSeen)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("auth: session: %w", err)
	}
	return &out, nil
}

// TouchSession implements Store.
func (s *PGStore) TouchSession(ctx context.Context, hash []byte, now, expires time.Time) error {
	if _, err := s.pool.Exec(ctx, "update sessions set last_seen_at = $2, expires_at = $3 where token_hash = $1", hash, now, expires); err != nil {
		return fmt.Errorf("auth: touch session: %w", err)
	}
	return nil
}

// DeleteSession implements Store.
func (s *PGStore) DeleteSession(ctx context.Context, hash []byte) error {
	if _, err := s.pool.Exec(ctx, "delete from sessions where token_hash = $1", hash); err != nil {
		return fmt.Errorf("auth: delete session: %w", err)
	}
	return nil
}

// ChooseName implements Store.
func (s *PGStore) ChooseName(ctx context.Context, hash []byte, name, display string, now time.Time, inviteOnly bool) (*views.Viewer, error) {
	v := &views.Viewer{}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var sid *string
		err := tx.QueryRow(ctx, `
			select email, surveyor_id::text from sessions
			where token_hash = $1 and expires_at > $2 for update`, hash, now).Scan(&v.Email, &sid)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		// Already named (this session, or another session of the same
		// e-mail chose a name in the meantime): keep that name.
		err = tx.QueryRow(ctx, `
			select id::text, name, coalesce(display_name, '') from surveyors
			where id = $1::uuid or lower(email) = $2 limit 1`, sid, v.Email).Scan(&v.SurveyorID, &v.Name, &v.Display)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			tag, err := tx.Exec(ctx, `
				select 1 from invitations
				where email = $1 and accepted_at is null and revoked_at is null for update`, v.Email)
			if err != nil {
				return err
			}
			if inviteOnly && tag.RowsAffected() == 0 {
				return ErrNotInvited
			}
			// Nobody's display name may be this Surveyor name either.
			var clash bool
			if err := tx.QueryRow(ctx, `select exists (select 1 from surveyors where lower(display_name) = lower($1))`, name).Scan(&clash); err != nil {
				return err
			}
			if clash {
				return ErrNameTaken
			}
			if display != "" && !strings.EqualFold(display, name) {
				taken, err := displayTaken(ctx, tx, display, "")
				if err != nil {
					return err
				}
				if taken {
					return ErrDisplayTaken
				}
			}
			v.SurveyorID, v.Name, v.Display = uuid.NewString(), name, display
			_, err = tx.Exec(ctx, `
				insert into surveyors (id, name, display_name, email, joined_at) values ($1::uuid, $2, nullif($3, ''), $4, $5)`,
				v.SurveyorID, name, display, v.Email, now)
			var pe *pgconn.PgError
			if errors.As(err, &pe) && pe.Code == "23505" && pe.ConstraintName == "surveyors_name_lower_idx" {
				return ErrNameTaken
			}
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `
				update invitations set accepted_at = $2, accepted_by = $3::uuid
				where email = $1 and accepted_at is null and revoked_at is null`, v.Email, now, v.SurveyorID)
			if err != nil {
				return err
			}
		case err != nil:
			return err
		}
		_, err = tx.Exec(ctx, `
			update sessions set surveyor_id = $1::uuid
			where email = $2 and surveyor_id is null`, v.SurveyorID, v.Email)
		return err
	})
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrNameTaken) || errors.Is(err, ErrNotInvited) || errors.Is(err, ErrDisplayTaken) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("auth: choose name: %w", err)
	}
	return v, nil
}

// displayTaken reports whether display is the Surveyor name of anyone
// but the Surveyor with id self ("" for nobody), ignoring case.
func displayTaken(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, display, self string) (bool, error) {
	var taken bool
	err := q.QueryRow(ctx, `select exists (select 1 from surveyors where lower(name) = lower($1) and ($2 = '' or id::text <> $2))`,
		display, self).Scan(&taken)
	return taken, err
}

// Profile implements Store.
func (s *PGStore) Profile(ctx context.Context, surveyorID string) (Profile, error) {
	var p Profile
	err := s.pool.QueryRow(ctx, `
		select coalesce(s.display_name, ''), s.bio, coalesce(r.slug, '')
		from surveyors s left join regions r on r.id = s.home_region_id
		where s.id = $1::uuid`, surveyorID).Scan(&p.Display, &p.Bio, &p.HomeRegion)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, fmt.Errorf("auth: profile: %w", err)
	}
	return p, nil
}

// UpdateProfile implements Store.
func (s *PGStore) UpdateProfile(ctx context.Context, surveyorID string, p Profile) error {
	if p.Display != "" {
		taken, err := displayTaken(ctx, s.pool, p.Display, surveyorID)
		if err != nil {
			return fmt.Errorf("auth: update profile: %w", err)
		}
		if taken {
			return ErrDisplayTaken
		}
	}
	tag, err := s.pool.Exec(ctx, `
		update surveyors set display_name = nullif($2, ''), bio = $3, home_region_id = (select id from regions where slug = $4)
		where id = $1::uuid`, surveyorID, p.Display, p.Bio, p.HomeRegion)
	if err != nil {
		return fmt.Errorf("auth: update profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MaySignIn implements Store.
func (s *PGStore) MaySignIn(ctx context.Context, email string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `select
		exists (select 1 from surveyors where lower(email) = $1) or
		exists (select 1 from invitations where email = $1 and accepted_at is null and revoked_at is null)`, email).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("auth: may sign in: %w", err)
	}
	return ok, nil
}

// RecordRefusal implements Store.
func (s *PGStore) RecordRefusal(ctx context.Context, emailHash []byte, now time.Time) (bool, error) {
	var send bool
	err := s.pool.QueryRow(ctx, `
		insert into signup_refusals (email_hash, sent_at) values ($1, $2)
		on conflict (email_hash) do update set sent_at = excluded.sent_at
		where signup_refusals.sent_at <= $3
		returning true`, emailHash, now, now.Add(-refusalEvery)).Scan(&send)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("auth: record refusal: %w", err)
	}
	return true, nil
}

// audit writes one admin_actions row in tx (the same table internal/db's
// admin actions write to).
func audit(ctx context.Context, tx pgx.Tx, adminID, action, email, reason string, after any) error {
	var b []byte
	if after != nil {
		var err error
		if b, err = json.Marshal(after); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `
		insert into admin_actions (admin_id, action, target, reason, after) values ($1::uuid, $2, $3, $4, $5)`,
		adminID, action, "invite:"+email, reason, b)
	if err != nil {
		return fmt.Errorf("audit %s: %w", action, err)
	}
	slog.Info("admin action", "action", action, "target", "invite")
	return nil
}

// CreateInvite implements Store.
func (s *PGStore) CreateInvite(ctx context.Context, inv Invite) error {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var name string
		err := tx.QueryRow(ctx, `select name from surveyors where lower(email) = $1`, inv.Email).Scan(&name)
		switch {
		case err == nil:
			return &AccountError{Name: name}
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}
		action := "invite.create"
		if inv.Resend {
			action = "invite.resend"
			tag, err := tx.Exec(ctx, `select 1 from invitations where email = $1 and accepted_at is null and revoked_at is null for update`, inv.Email)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return ErrNotPending
			}
		} else {
			// A new invitation, a second one for a pending address (new note,
			// new date) or one that reopens a revoked row.
			_, err := tx.Exec(ctx, `
				insert into invitations (email, invited_by, note, invited_at) values ($1, $2::uuid, $3, $4)
				on conflict (email) do update set invited_by = excluded.invited_by, note = excluded.note,
				  invited_at = excluded.invited_at, revoked_at = null
				where invitations.accepted_at is null`, inv.Email, inv.AdminID, inv.Note, inv.CreatedAt)
			if err != nil {
				return err
			}
		}
		t := inv.Token
		_, err = tx.Exec(ctx, `
			insert into login_tokens (token_hash, email, next_path, created_at, expires_at)
			values ($1, $2, $3, $4, $5)`, t.Hash, t.Email, nullable(t.Next), t.CreatedAt, t.ExpiresAt)
		if err != nil {
			return err
		}
		after := map[string]any{"link_expires_at": t.ExpiresAt}
		if !inv.Resend {
			after["note"] = inv.Note
		}
		return audit(ctx, tx, inv.AdminID, action, inv.Email, "", after)
	})
	var ae *AccountError
	if errors.As(err, &ae) || errors.Is(err, ErrNotPending) {
		return err
	}
	if err != nil {
		return fmt.Errorf("auth: create invite: %w", err)
	}
	return nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// RevokeInvite implements Store.
func (s *PGStore) RevokeInvite(ctx context.Context, adminID, email string, now time.Time) error {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			update invitations set revoked_at = $2
			where email = $1 and accepted_at is null and revoked_at is null`, email, now)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotPending
		}
		links, err := tx.Exec(ctx, `delete from login_tokens where email = $1 and used_at is null`, email)
		if err != nil {
			return err
		}
		sessions, err := tx.Exec(ctx, `delete from sessions where email = $1 and surveyor_id is null`, email)
		if err != nil {
			return err
		}
		return audit(ctx, tx, adminID, "invite.revoke", email, "",
			map[string]any{"links_deleted": links.RowsAffected(), "sessions_ended": sessions.RowsAffected()})
	})
	if errors.Is(err, ErrNotPending) {
		return err
	}
	if err != nil {
		return fmt.Errorf("auth: revoke invite: %w", err)
	}
	return nil
}

// Invitations implements Store.
func (s *PGStore) Invitations(ctx context.Context) ([]views.Invitation, error) {
	rows, err := s.pool.Query(ctx, `
		select i.email, coalesce(a.name, ''), i.note, i.invited_at, i.accepted_at, coalesce(u.name, ''), i.revoked_at
		from invitations i left join surveyors a on a.id = i.invited_by left join surveyors u on u.id = i.accepted_by
		order by i.invited_at desc, i.email`)
	if err != nil {
		return nil, fmt.Errorf("auth: invitations: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (views.Invitation, error) {
		var i views.Invitation
		err := r.Scan(&i.Email, &i.InvitedBy, &i.Note, &i.InvitedAt, &i.AcceptedAt, &i.AcceptedAs, &i.RevokedAt)
		i.InvitedAt = i.InvitedAt.UTC()
		return i, err
	})
	if err != nil {
		return nil, fmt.Errorf("auth: invitations: %w", err)
	}
	return out, nil
}
