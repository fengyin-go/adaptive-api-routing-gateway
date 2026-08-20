package service

import (
	"sort"
	"time"

	"apigateway/internal/model"
	"apigateway/pkg/idgen"
)

// RecordRequest 记录一次网关访问日志。
func (s *Service) RecordRequest(l model.RequestLog) *model.RequestLog {
	l.ID = idgen.Hex()
	if l.Timestamp.IsZero() {
		l.Timestamp = time.Now()
	}
	_ = s.store.CreateRequestLog(&l)
	return &l
}

func (s *Service) GetRequestLog(id string) (*model.RequestLog, error) {
	return s.store.GetRequestLog(id)
}

func (s *Service) ListRequestLogs(filter model.RequestLogFilter, page, size int) ([]*model.RequestLog, int, error) {
	all := s.store.ListRequestLogs()
	matched := make([]*model.RequestLog, 0, len(all))
	for _, l := range all {
		if filter.Match(l) {
			matched = append(matched, l)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Timestamp.After(matched[j].Timestamp)
	})
	return paginate(matched, page, size)
}
