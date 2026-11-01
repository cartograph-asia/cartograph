package storage

import (
	"context"
	"io"
	"sync"
)

// Memory is an in-memory Store for tests.
type Memory struct {
	BaseURL string

	mu      sync.Mutex
	objects map[string]Object
}

// Object is a stored object in Memory.
type Object struct {
	ContentType string
	Data        []byte
}

// NewMemory returns an empty in-memory store.
func NewMemory(baseURL string) *Memory {
	return &Memory{BaseURL: baseURL, objects: map[string]Object{}}
}

// Put stores the content of r under key.
func (m *Memory) Put(_ context.Context, key, contentType string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = Object{ContentType: contentType, Data: data}
	return nil
}

// URL returns the public URL of key.
func (m *Memory) URL(key string) string { return PublicURL(m.BaseURL, key) }

// Get returns the object stored under key.
func (m *Memory) Get(key string) (Object, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objects[key]
	return o, ok
}

// Keys returns all stored keys.
func (m *Memory) Keys() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.objects))
	for k := range m.objects {
		keys = append(keys, k)
	}
	return keys
}
