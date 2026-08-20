package service

import (
	"sort"
	"time"

	"apigateway/internal/model"
	"apigateway/pkg/idgen"
)

func (s *Service) CreateRateLimitRule(r model.RateLimitRule) (*model.RateLimitRule, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	r.ID = idgen.Hex()
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now
	if err := s.store.CreateRateLimitRule(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Service) GetRateLimitRule(id string) (*model.RateLimitRule, error) {
	return s.store.GetRateLimitRule(id)
}

func (s *Service) ListRateLimitRules(page, size int) ([]*model.RateLimitRule, int, error) {
	all := s.store.ListRateLimitRules()
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	return paginate(all, page, size)
}

func (s *Service) UpdateRateLimitRule(id string, r model.RateLimitRule) (*model.RateLimitRule, error) {
	exist, err := s.store.GetRateLimitRule(id)
	if err != nil {
		return nil, err
	}
	exist.Name = r.Name
	exist.Limit = r.Limit
	exist.Window = r.Window
	exist.UpdatedAt = time.Now()
	if err := exist.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.UpdateRateLimitRule(exist); err != nil {
		return nil, err
	}
	// 规则变更后重置限流桶
	s.ResetBuckets()
	return exist, nil
}

func (s *Service) DeleteRateLimitRule(id string) error {
	return s.store.DeleteRateLimitRule(id)
}

// AllowRequest 判断某路由是否放行（基于其绑定的限流规则）。
// 无规则时默认放行。
func (s *Service) AllowRequest(route *model.Route) bool {
	if route == nil || route.RateLimitRuleID == "" {
		return true
	}
	rule, err := s.store.GetRateLimitRule(route.RateLimitRuleID)
	if err != nil {
		return true
	}
	return s.BucketFor(rule).Allow()
}
