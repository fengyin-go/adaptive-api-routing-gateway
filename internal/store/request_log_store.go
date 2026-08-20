package store

import (
	"apigateway/internal/model"
)

func (s *MemoryStore) CreateRequestLog(l *model.RequestLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestLogs[l.ID] = l
	return nil
}

func (s *MemoryStore) GetRequestLog(id string) (*model.RequestLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.requestLogs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return l, nil
}

func (s *MemoryStore) ListRequestLogs() []*model.RequestLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.RequestLog, 0, len(s.requestLogs))
	for _, l := range s.requestLogs {
		list = append(list, l)
	}
	return list
}
