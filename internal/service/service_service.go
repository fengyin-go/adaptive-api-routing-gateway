package service

import (
	"sort"
	"time"

	"apigateway/internal/model"
	"apigateway/pkg/idgen"
)

func (s *Service) CreateService(v model.Service) (*model.Service, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	v.ID = idgen.Hex()
	now := time.Now()
	v.CreatedAt = now
	v.UpdatedAt = now
	if err := s.store.CreateService(&v); err != nil {
		return nil, err
	}
	// 初始化健康检查记录
	s.store.CreateHealthCheck(&model.HealthCheck{
		ID:          idgen.Hex(),
		ServiceID:   v.ID,
		Status:      model.HealthHealthy,
		LastChecked: now,
	})
	return &v, nil
}

func (s *Service) GetService(id string) (*model.Service, error) {
	return s.store.GetService(id)
}

// GetServiceByName 按名称查找服务。
func (s *Service) GetServiceByName(name string) (*model.Service, error) {
	return s.store.GetServiceByName(name)
}

func (s *Service) ListServices(filter model.ServiceFilter, page, size int) ([]*model.Service, int, error) {
	all := s.store.ListServices()
	matched := make([]*model.Service, 0, len(all))
	for _, v := range all {
		if filter.Match(v) {
			matched = append(matched, v)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})
	return paginate(matched, page, size)
}

func (s *Service) UpdateService(id string, v model.Service) (*model.Service, error) {
	exist, err := s.store.GetService(id)
	if err != nil {
		return nil, err
	}
	exist.Name = v.Name
	exist.BaseURL = v.BaseURL
	exist.Timeout = v.Timeout
	exist.Retries = v.Retries
	if v.Status != "" {
		exist.Status = v.Status
	}
	exist.UpdatedAt = time.Now()
	if err := exist.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.UpdateService(exist); err != nil {
		return nil, err
	}
	return exist, nil
}

func (s *Service) DeleteService(id string) error {
	return s.store.DeleteService(id)
}
