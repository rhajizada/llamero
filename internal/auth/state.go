package auth

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// StateStore tracks OAuth state parameters to prevent CSRF.
type StateStore struct {
	ttl   time.Duration
	mu    sync.Mutex
	state map[string]time.Time
}

// NewStateStore builds an in-memory store.
func NewStateStore(ttl time.Duration) *StateStore {
	return &StateStore{
		ttl:   ttl,
		state: make(map[string]time.Time),
	}
}

// Issue creates a new opaque state token.
func (s *StateStore) Issue() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()

	token := uuid.NewString()
	s.state[token] = time.Now().Add(s.ttl)
	return token
}

// Consume validates and removes the supplied state token.
func (s *StateStore) Consume(token string) bool {
	if token == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked()

	expiry, ok := s.state[token]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		delete(s.state, token)
		return false
	}
	delete(s.state, token)
	return true
}

func (s *StateStore) cleanupLocked() {
	now := time.Now()
	for token, expiry := range s.state {
		if now.After(expiry) {
			delete(s.state, token)
		}
	}
}
