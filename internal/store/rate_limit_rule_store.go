package store

import (
	"apigateway/internal/model"
)

func (s *MemoryStore) CreateRateLimitRule(r *model.RateLimitRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, exist := range s.rateLimitRules {
		if exist.Name == r.Name {
			return ErrConflict
		}
	}
	s.rateLimitRules[r.ID] = r
	return nil
}

func (s *MemoryStore) GetRateLimitRule(id string) (*model.RateLimitRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rateLimitRules[id]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *MemoryStore) ListRateLimitRules() []*model.RateLimitRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.RateLimitRule, 0, len(s.rateLimitRules))
	for _, r := range s.rateLimitRules {
		list = append(list, r)
	}
	return list
}

func (s *MemoryStore) UpdateRateLimitRule(r *model.RateLimitRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rateLimitRules[r.ID]; !ok {
		return ErrNotFound
	}
	for _, exist := range s.rateLimitRules {
		if exist.ID != r.ID && exist.Name == r.Name {
			return ErrConflict
		}
	}
	s.rateLimitRules[r.ID] = r
	return nil
}

func (s *MemoryStore) DeleteRateLimitRule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rateLimitRules[id]; !ok {
		return ErrNotFound
	}
	delete(s.rateLimitRules, id)
	return nil
}
