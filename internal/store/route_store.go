package store

import (
	"apigateway/internal/model"
)

func (s *MemoryStore) CreateRoute(r *model.Route) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, exist := range s.routes {
		if exist.Path == r.Path && exist.Method == r.Method {
			return ErrConflict
		}
	}
	s.routes[r.ID] = r
	return nil
}

func (s *MemoryStore) GetRoute(id string) (*model.Route, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.routes[id]
	if !ok {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *MemoryStore) ListRoutes() []*model.Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.Route, 0, len(s.routes))
	for _, r := range s.routes {
		list = append(list, r)
	}
	return list
}

func (s *MemoryStore) UpdateRoute(r *model.Route) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.routes[r.ID]; !ok {
		return ErrNotFound
	}
	for _, exist := range s.routes {
		if exist.ID != r.ID && exist.Path == r.Path && exist.Method == r.Method {
			return ErrConflict
		}
	}
	s.routes[r.ID] = r
	return nil
}

func (s *MemoryStore) DeleteRoute(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.routes[id]; !ok {
		return ErrNotFound
	}
	delete(s.routes, id)
	return nil
}
