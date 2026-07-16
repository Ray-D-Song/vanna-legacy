package session

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/domain"
)

type Store struct {
	ttl  time.Duration
	mu   sync.RWMutex
	data map[string]*domain.AskState
}

func NewStore(ttl time.Duration) *Store {
	s := &Store{
		ttl:  ttl,
		data: map[string]*domain.AskState{},
	}
	go s.cleanupLoop()
	return s
}

func (s *Store) Create(state domain.AskState) *domain.AskState {
	now := time.Now()
	if state.ID == "" {
		state.ID = uuid.NewString()
	}
	state.CreatedAt = now
	state.UpdatedAt = now
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := state
	s.data[state.ID] = &copy
	return s.data[state.ID]
}

func (s *Store) Get(id string) (*domain.AskState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.data[id]
	if !ok {
		return nil, false
	}
	if s.expired(state) {
		return nil, false
	}
	copy := *state
	return &copy, true
}

func (s *Store) Update(id string, fn func(*domain.AskState)) (*domain.AskState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.data[id]
	if !ok || s.expired(state) {
		return nil, false
	}
	fn(state)
	state.UpdatedAt = time.Now()
	copy := *state
	return &copy, true
}

func (s *Store) ListQuestions() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0)
	for _, state := range s.data {
		if s.expired(state) {
			continue
		}
		if state.Question != "" {
			out = append(out, state.Question)
		}
	}
	return out
}

func (s *Store) expired(state *domain.AskState) bool {
	return s.ttl > 0 && time.Since(state.UpdatedAt) > s.ttl
}

func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		for id, state := range s.data {
			if s.expired(state) {
				delete(s.data, id)
			}
		}
		s.mu.Unlock()
	}
}
