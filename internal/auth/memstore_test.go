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
	failNext  error                   // returned once by Session, to test DB errors
	touched   int
}

type memToken struct {
	LoginToken
	used bool
}

type memSession struct {
	email      string
	surveyorID string
	expires    time.Time
	lastSeen   time.Time
}

type memSurveyor struct {
	id, name, email string
	placeholder     bool
	bio, home       string
	joined          time.Time
}

func newMemStore() *memStore {
	return &memStore{tokens: map[string]*memToken{}, sessions: map[string]*memSession{}, surveyors: map[string]*memSurveyor{}}
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
		s.placeholder = false
		v.SurveyorID, v.Name = s.id, s.name
	}
	m.sessions[string(hash)] = &memSession{email: email, surveyorID: v.SurveyorID, expires: expires, lastSeen: now}
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
	out := &Session{Viewer: views.Viewer{Email: s.email, SurveyorID: s.surveyorID}, LastSeen: s.lastSeen}
	if u := m.surveyors[s.surveyorID]; u != nil {
		out.Viewer.Name = u.name
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

func (m *memStore) ChooseName(_ context.Context, hash []byte, name string, now time.Time) (*views.Viewer, error) {
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
		u = &memSurveyor{id: uuid.NewString(), name: name, email: s.email, joined: now}
		m.surveyors[u.id] = u
	}
	for _, o := range m.sessions {
		if o.email == s.email && o.surveyorID == "" {
			o.surveyorID = u.id
		}
	}
	return &views.Viewer{SurveyorID: u.id, Name: u.name, Email: s.email}, nil
}

func (m *memStore) Profile(_ context.Context, id string) (string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.surveyors[id]
	if u == nil {
		return "", "", ErrNotFound
	}
	return u.bio, u.home, nil
}

func (m *memStore) UpdateProfile(_ context.Context, id, bio, home string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.surveyors[id]
	if u == nil {
		return ErrNotFound
	}
	u.bio, u.home = bio, home
	return nil
}
