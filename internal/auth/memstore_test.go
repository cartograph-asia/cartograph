package auth

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cartograph-asia/cartograph/internal/views"
)

// memStore is an in-memory Store with PGStore's semantics, for handler tests.
type memStore struct {
	mu        sync.Mutex
	tokens    map[string]*memToken
	sessions  map[string]*memSession
	surveyors map[string]*memSurveyor // by id
	invites   map[string]*views.Invitation
	refusals  map[string]time.Time
	actions   []string // "action target" of admin_actions rows
	failNext  error    // returned once by Session, to test DB errors
	touched   int
}

type memToken struct {
	LoginToken
	used bool
}

type memSession struct {
	email      string
	surveyorID string
	created    time.Time
	expires    time.Time
	lastSeen   time.Time
}

type memSurveyor struct {
	id, name, email string
	display         string
	placeholder     bool
	blocked         bool
	bio, home       string
	joined          time.Time
}

func newMemStore() *memStore {
	return &memStore{tokens: map[string]*memToken{}, sessions: map[string]*memSession{}, surveyors: map[string]*memSurveyor{},
		invites: map[string]*views.Invitation{}, refusals: map[string]time.Time{}}
}

func (m *memStore) addSurveyor(name, email string, placeholder bool) *memSurveyor {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := &memSurveyor{id: uuid.NewString(), name: name, email: email, placeholder: placeholder}
	m.surveyors[s.id] = s
	return s
}

func (m *memStore) byEmail(email string) *memSurveyor {
	for _, s := range m.surveyors {
		if s.email != "" && strings.EqualFold(s.email, email) {
			return s
		}
	}
	return nil
}

func (m *memStore) CreateLoginToken(_ context.Context, t LoginToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[string(t.Hash)] = &memToken{LoginToken: t}
	return nil
}

func (m *memStore) CountLoginTokens(_ context.Context, email string, since time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, t := range m.tokens {
		if t.Email == email && t.CreatedAt.After(since) {
			n++
		}
	}
	return n, nil
}

func (m *memStore) ConsumeLoginToken(_ context.Context, hash []byte, now time.Time) (string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.tokens[string(hash)]
	if t == nil || t.used || !t.ExpiresAt.After(now) {
		return "", "", ErrNotFound
	}
	t.used = true
	return t.Email, t.Next, nil
}

func (m *memStore) CreateSession(_ context.Context, hash []byte, email string, now, expires time.Time) (*views.Viewer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v := &views.Viewer{Email: email}
	if s := m.byEmail(email); s != nil {
		if s.blocked {
			return nil, ErrBlocked
		}
		s.placeholder = false
		v.SurveyorID, v.Name = s.id, s.name
	}
	m.sessions[string(hash)] = &memSession{email: email, surveyorID: v.SurveyorID, created: now, expires: expires, lastSeen: now}
	return v, nil
}

func (m *memStore) Session(_ context.Context, hash []byte, now time.Time) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.failNext; err != nil {
		m.failNext = nil
		return nil, err
	}
	s := m.sessions[string(hash)]
	if s == nil || !s.expires.After(now) {
		return nil, ErrNotFound
	}
	out := &Session{Viewer: views.Viewer{Email: s.email, SurveyorID: s.surveyorID}, CreatedAt: s.created, LastSeen: s.lastSeen}
	if u := m.surveyors[s.surveyorID]; u != nil {
		if u.blocked {
			return nil, ErrNotFound
		}
		out.Viewer.Name, out.Viewer.Display = u.name, u.display
	}
	return out, nil
}

func (m *memStore) TouchSession(_ context.Context, hash []byte, now, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s := m.sessions[string(hash)]; s != nil {
		s.lastSeen, s.expires = now, expires
		m.touched++
	}
	return nil
}

func (m *memStore) DeleteSession(_ context.Context, hash []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, string(hash))
	return nil
}

func (m *memStore) ChooseName(_ context.Context, hash []byte, name, display string, now time.Time, inviteOnly bool) (*views.Viewer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[string(hash)]
	if s == nil || !s.expires.After(now) {
		return nil, ErrNotFound
	}
	u := m.surveyors[s.surveyorID]
	if u == nil {
		u = m.byEmail(s.email)
	}
	if u == nil {
		for _, o := range m.surveyors {
			if strings.EqualFold(o.name, name) {
				return nil, ErrNameTaken
			}
		}
		inv := m.invites[s.email]
		if inviteOnly && (inv == nil || !inv.Pending()) {
			return nil, ErrNotInvited
		}
		if display != "" && !strings.EqualFold(display, name) && m.handleTaken(display, "") {
			return nil, ErrDisplayTaken
		}
		u = &memSurveyor{id: uuid.NewString(), name: name, display: display, email: s.email, joined: now}
		m.surveyors[u.id] = u
		if inv != nil && inv.Pending() {
			inv.AcceptedAt, inv.AcceptedAs = &now, name
		}
	}
	for _, o := range m.sessions {
		if o.email == s.email && o.surveyorID == "" {
			o.surveyorID = u.id
		}
	}
	return &views.Viewer{SurveyorID: u.id, Name: u.name, Display: u.display, Email: s.email}, nil
}

func (m *memStore) handleTaken(display, self string) bool {
	for _, o := range m.surveyors {
		if o.id != self && strings.EqualFold(o.name, display) {
			return true
		}
	}
	return false
}

func (m *memStore) Profile(_ context.Context, id string) (Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.surveyors[id]
	if u == nil {
		return Profile{}, ErrNotFound
	}
	return Profile{Display: u.display, Bio: u.bio, HomeRegion: u.home}, nil
}

func (m *memStore) UpdateProfile(_ context.Context, id string, p Profile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.surveyors[id]
	if u == nil {
		return ErrNotFound
	}
	if p.Display != "" && m.handleTaken(p.Display, id) {
		return ErrDisplayTaken
	}
	u.display, u.bio, u.home = p.Display, p.Bio, p.HomeRegion
	return nil
}

func (m *memStore) MaySignIn(_ context.Context, email string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv := m.invites[email]
	return m.byEmail(email) != nil || (inv != nil && inv.Pending()), nil
}

func (m *memStore) RecordRefusal(_ context.Context, h []byte, now time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if last, ok := m.refusals[string(h)]; ok && now.Sub(last) < refusalEvery {
		return false, nil
	}
	m.refusals[string(h)] = now
	return true, nil
}

func (m *memStore) CreateInvite(_ context.Context, inv Invite) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u := m.byEmail(inv.Email); u != nil {
		return &AccountError{Name: u.name}
	}
	cur := m.invites[inv.Email]
	action := "invite.create"
	if inv.Resend {
		if cur == nil || !cur.Pending() {
			return ErrNotPending
		}
		action = "invite.resend"
	} else {
		m.invites[inv.Email] = &views.Invitation{Email: inv.Email, InvitedBy: inv.AdminID, Note: inv.Note, InvitedAt: inv.CreatedAt}
	}
	m.tokens[string(inv.Token.Hash)] = &memToken{LoginToken: inv.Token}
	m.actions = append(m.actions, action+" "+inv.Email)
	return nil
}

func (m *memStore) RevokeInvite(_ context.Context, _, email string, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv := m.invites[email]
	if inv == nil || !inv.Pending() {
		return ErrNotPending
	}
	inv.RevokedAt = &now
	for k, t := range m.tokens {
		if t.Email == email && !t.used {
			delete(m.tokens, k)
		}
	}
	for k, s := range m.sessions {
		if s.email == email && s.surveyorID == "" {
			delete(m.sessions, k)
		}
	}
	m.actions = append(m.actions, "invite.revoke "+email)
	return nil
}

func (m *memStore) Invitations(context.Context) ([]views.Invitation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []views.Invitation
	for _, i := range m.invites {
		out = append(out, *i)
	}
	return out, nil
}
