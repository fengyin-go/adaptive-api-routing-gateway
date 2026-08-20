package store

import (
	"apigateway/internal/model"
)

func (s *MemoryStore) CreateService(v *model.Service) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, exist := range s.services {
		if exist.Name == v.Name {
			return ErrConflict
		}
	}
	s.services[v.ID] = v
	return nil
}

func (s *MemoryStore) GetService(id string) (*model.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.services[id]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}

func (s *MemoryStore) GetServiceByName(name string) (*model.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.services {
		if v.Name == name {
			return v, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) ListServices() []*model.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.Service, 0, len(s.services))
	for _, v := range s.services {
		list = append(list, v)
	}
	return list
}

func (s *MemoryStore) UpdateService(v *model.Service) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[v.ID]; !ok {
		return ErrNotFound
	}
	for _, exist := range s.services {
		if exist.ID != v.ID && exist.Name == v.Name {
			return ErrConflict
		}
	}
	s.services[v.ID] = v
	return nil
}

func (s *MemoryStore) DeleteService(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[id]; !ok {
		return ErrNotFound
	}
	delete(s.services, id)
	return nil
}
