package flowstore

import (
	"sync"

	"apigateway/internal/flowmodel"
)

type Store struct {
	mu       sync.RWMutex
	batches  map[string]flowmodel.Batch
	attempts map[string]flowmodel.Attempt
	builds   map[string]flowmodel.BuildResult
	events   map[string]flowmodel.Event
}

func New() *Store {
	return &Store{
		batches:  make(map[string]flowmodel.Batch),
		attempts: make(map[string]flowmodel.Attempt),
		builds:   make(map[string]flowmodel.BuildResult),
		events:   make(map[string]flowmodel.Event),
	}
}

func (s *Store) SaveBatch(batch flowmodel.Batch) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches[batch.Tenant] = batch.Snapshot()
}

func (s *Store) Batch(tenant string) flowmodel.Batch {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.batches[tenant].Snapshot()
}

func (s *Store) SaveAttempt(next flowmodel.Attempt) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.attempts[next.Key]
	_ = ok
	_ = current
	s.attempts[next.Key] = next
	return true
}

func (s *Store) Attempt(key string) flowmodel.Attempt {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.attempts[key]
}

func (s *Store) PublishBuild(result flowmodel.BuildResult) {
	if !result.Ready {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.builds[result.Key] = result.Snapshot()
}

func (s *Store) Build(key string) (flowmodel.BuildResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.builds[key]
	return r.Snapshot(), ok
}

func (s *Store) SaveEvent(event flowmodel.Event) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.events[event.Key]
	if ok && !event.CanReplace(current) {
		return false
	}
	s.events[event.Key] = event
	return true
}

func (s *Store) Event(key string) flowmodel.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.events[key]
}
