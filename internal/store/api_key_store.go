package store

import (
	"apigateway/internal/model"
)

func (s *MemoryStore) CreateAPIKey(k *model.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, exist := range s.apiKeys {
		if exist.Key == k.Key {
			return ErrConflict
		}
	}
	s.apiKeys[k.ID] = k
	return nil
}

func (s *MemoryStore) GetAPIKey(id string) (*model.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.apiKeys[id]
	if !ok {
		return nil, ErrNotFound
	}
	return k, nil
}

func (s *MemoryStore) GetAPIKeyByKey(key string) (*model.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, k := range s.apiKeys {
		if k.Key == key {
			return k, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) ListAPIKeys() []*model.APIKey {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.APIKey, 0, len(s.apiKeys))
	for _, k := range s.apiKeys {
		list = append(list, k)
	}
	return list
}

func (s *MemoryStore) UpdateAPIKey(k *model.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apiKeys[k.ID]; !ok {
		return ErrNotFound
	}
	for _, exist := range s.apiKeys {
		if exist.ID != k.ID && exist.Key == k.Key {
			return ErrConflict
		}
	}
	s.apiKeys[k.ID] = k
	return nil
}

func (s *MemoryStore) DeleteAPIKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apiKeys[id]; !ok {
		return ErrNotFound
	}
	delete(s.apiKeys, id)
	return nil
}
