package store

import (
	"apigateway/internal/model"
)

func (s *MemoryStore) CreateHealthCheck(h *model.HealthCheck) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthChecks[h.ID] = h
	return nil
}

func (s *MemoryStore) GetHealthCheck(id string) (*model.HealthCheck, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.healthChecks[id]
	if !ok {
		return nil, ErrNotFound
	}
	return h, nil
}

func (s *MemoryStore) GetHealthCheckByService(serviceID string) (*model.HealthCheck, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, h := range s.healthChecks {
		if h.ServiceID == serviceID {
			return h, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) ListHealthChecks() []*model.HealthCheck {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.HealthCheck, 0, len(s.healthChecks))
	for _, h := range s.healthChecks {
		list = append(list, h)
	}
	return list
}

func (s *MemoryStore) UpdateHealthCheck(h *model.HealthCheck) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.healthChecks[h.ID]; !ok {
		return ErrNotFound
	}
	s.healthChecks[h.ID] = h
	return nil
}
