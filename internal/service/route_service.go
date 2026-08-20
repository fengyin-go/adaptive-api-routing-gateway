package service

import (
	"sort"
	"strings"
	"time"

	"apigateway/internal/model"
	"apigateway/pkg/idgen"
)

func (s *Service) CreateRoute(r model.Route) (*model.Route, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.store.GetService(r.ServiceID); err != nil {
		return nil, err
	}
	// 若指定了限流规则，校验其存在
	if r.RateLimitRuleID != "" {
		if _, err := s.store.GetRateLimitRule(r.RateLimitRuleID); err != nil {
			return nil, err
		}
	}
	r.ID = idgen.Hex()
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	if err := s.store.CreateRoute(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Service) GetRoute(id string) (*model.Route, error) {
	return s.store.GetRoute(id)
}

func (s *Service) ListRoutes(filter model.RouteFilter, page, size int) ([]*model.Route, int, error) {
	all := s.store.ListRoutes()
	matched := make([]*model.Route, 0, len(all))
	for _, r := range all {
		if filter.Match(r) {
			matched = append(matched, r)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})
	return paginate(matched, page, size)
}

func (s *Service) UpdateRoute(id string, r model.Route) (*model.Route, error) {
	exist, err := s.store.GetRoute(id)
	if err != nil {
		return nil, err
	}
	exist.Path = r.Path
	exist.Method = r.Method
	exist.ServiceID = r.ServiceID
	exist.StripPrefix = r.StripPrefix
	exist.RequiresAuth = r.RequiresAuth
	exist.RateLimitRuleID = r.RateLimitRuleID
	exist.UpdatedAt = time.Now()
	if err := exist.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.store.GetService(exist.ServiceID); err != nil {
		return nil, err
	}
	if exist.RateLimitRuleID != "" {
		if _, err := s.store.GetRateLimitRule(exist.RateLimitRuleID); err != nil {
			return nil, err
		}
	}
	if err := s.store.UpdateRoute(exist); err != nil {
		return nil, err
	}
	return exist, nil
}

func (s *Service) DeleteRoute(id string) error {
	return s.store.DeleteRoute(id)
}

// MatchRoute 按 method + path 前缀匹配一条路由，最长前缀优先。
func (s *Service) MatchRoute(method, path string) (*model.Route, bool) {
	best := -1
	var matched *model.Route
	for _, r := range s.store.ListRoutes() {
		if r.Method != method {
			continue
		}
		if path == r.Path || strings.HasPrefix(path, r.Path+"/") {
			if len(r.Path) > best {
				best = len(r.Path)
				matched = r
			}
		}
	}
	return matched, matched != nil
}
