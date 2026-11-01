package auth

import (
	"context"
	"errors"
	"fmt"
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
)

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
	Viewer   views.Viewer
	LastSeen time.Time
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
	// being one); otherwise the viewer has no name yet.
	CreateSession(ctx context.Context, hash []byte, email string, now, expires time.Time) (*views.Viewer, error)
	// Session returns an unexpired session, or ErrNotFound.
	Session(ctx context.Context, hash []byte, now time.Time) (*Session, error)
	// TouchSession records a visit and moves the expiry.
	TouchSession(ctx context.Context, hash []byte, now, expires time.Time) error
	// DeleteSession ends a session; a missing one is not an error.
	DeleteSession(ctx context.Context, hash []byte) error
	// ChooseName creates the Surveyor for the session's e-mail with name
	// (joined now) and attaches every nameless session of that e-mail. It
	// returns ErrNameTaken when the name is taken regardless of case, and
	// ErrNotFound for an unknown or expired session. If the session already
	// has a Surveyor, that one is returned unchanged.
	ChooseName(ctx context.Context, hash []byte, name string, now time.Time) (*views.Viewer, error)
	// Profile returns a Surveyor's bio and home region slug ("" for none).
	Profile(ctx context.Context, surveyorID string) (bio, homeRegion string, err error)
	// UpdateProfile sets bio and home region (slug, "" for none).
	UpdateProfile(ctx context.Context, surveyorID, bio, homeRegion string) error
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
		var placeholder bool
		err := tx.QueryRow(ctx, "select id::text, name, placeholder from surveyors where lower(email) = $1", email).
			Scan(&v.SurveyorID, &v.Name, &placeholder)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return err
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
	if err != nil {
		return nil, fmt.Errorf("auth: create session: %w", err)
	}
	return v, nil
}

// Session implements Store.
func (s *PGStore) Session(ctx context.Context, hash []byte, now time.Time) (*Session, error) {
	var out Session
	err := s.pool.QueryRow(ctx, `
		select s.email, coalesce(u.id::text, ''), coalesce(u.name, ''), s.last_seen_at
		from sessions s left join surveyors u on u.id = s.surveyor_id
		where s.token_hash = $1 and s.expires_at > $2`, hash, now).
		Scan(&out.Viewer.Email, &out.Viewer.SurveyorID, &out.Viewer.Name, &out.LastSeen)
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
func (s *PGStore) ChooseName(ctx context.Context, hash []byte, name string, now time.Time) (*views.Viewer, error) {
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
			select id::text, name from surveyors
			where id = $1::uuid or lower(email) = $2 limit 1`, sid, v.Email).Scan(&v.SurveyorID, &v.Name)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			v.SurveyorID, v.Name = uuid.NewString(), name
			_, err = tx.Exec(ctx, `
				insert into surveyors (id, name, email, joined_at) values ($1::uuid, $2, $3, $4)`,
				v.SurveyorID, name, v.Email, now)
			var pe *pgconn.PgError
			if errors.As(err, &pe) && pe.Code == "23505" && pe.ConstraintName == "surveyors_name_lower_idx" {
				return ErrNameTaken
			}
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
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrNameTaken) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("auth: choose name: %w", err)
	}
	return v, nil
}

// Profile implements Store.
func (s *PGStore) Profile(ctx context.Context, surveyorID string) (string, string, error) {
	var bio, home string
	err := s.pool.QueryRow(ctx, `
		select s.bio, coalesce(r.slug, '')
		from surveyors s left join regions r on r.id = s.home_region_id
		where s.id = $1::uuid`, surveyorID).Scan(&bio, &home)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("auth: profile: %w", err)
	}
	return bio, home, nil
}

// UpdateProfile implements Store.
func (s *PGStore) UpdateProfile(ctx context.Context, surveyorID, bio, homeRegion string) error {
	tag, err := s.pool.Exec(ctx, `
		update surveyors set bio = $2, home_region_id = (select id from regions where slug = $3)
		where id = $1::uuid`, surveyorID, bio, homeRegion)
	if err != nil {
		return fmt.Errorf("auth: update profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
