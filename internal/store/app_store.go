package store

import (
	"apigateway/internal/model"
)

func (s *MemoryStore) CreateApp(a *model.App) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, exist := range s.apps {
		if exist.Name == a.Name {
			return ErrConflict
		}
	}
	s.apps[a.ID] = a
	return nil
}

func (s *MemoryStore) GetApp(id string) (*model.App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.apps[id]
	if !ok {
		return nil, ErrNotFound
	}
	return a, nil
}

func (s *MemoryStore) GetAppByName(name string) (*model.App, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.apps {
		if a.Name == name {
			return a, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) ListApps() []*model.App {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.App, 0, len(s.apps))
	for _, a := range s.apps {
		list = append(list, a)
	}
	return list
}

func (s *MemoryStore) UpdateApp(a *model.App) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[a.ID]; !ok {
		return ErrNotFound
	}
	for _, exist := range s.apps {
		if exist.ID != a.ID && exist.Name == a.Name {
			return ErrConflict
		}
	}
	s.apps[a.ID] = a
	return nil
}

func (s *MemoryStore) DeleteApp(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apps[id]; !ok {
		return ErrNotFound
	}
	delete(s.apps, id)
	return nil
}
