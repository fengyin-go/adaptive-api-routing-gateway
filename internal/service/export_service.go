package service

import (
	"apigateway/internal/model"
)

// ConfigSnapshot 全量配置快照，用于导入导出。
type ConfigSnapshot struct {
	Services        []*model.Service        `json:"services"`
	Routes          []*model.Route          `json:"routes"`
	Apps            []*model.App            `json:"apps"`
	APIKeys         []*model.APIKey         `json:"api_keys"`
	RateLimitRules  []*model.RateLimitRule  `json:"rate_limit_rules"`
}

// ExportConfig 导出全量配置。
func (s *Service) ExportConfig() *ConfigSnapshot {
	return &ConfigSnapshot{
		Services:       s.store.ListServices(),
		Routes:         s.store.ListRoutes(),
		Apps:           s.store.ListApps(),
		APIKeys:        s.store.ListAPIKeys(),
		RateLimitRules: s.store.ListRateLimitRules(),
	}
}

// ImportServices 批量导入上游服务，返回成功数量与错误列表。
func (s *Service) ImportServices(list []model.Service) (int, []string) {
	created := 0
	var errs []string
	for _, v := range list {
		if _, err := s.CreateService(v); err != nil {
			errs = append(errs, v.Name+": "+err.Error())
			continue
		}
		created++
	}
	return created, errs
}

// ImportRateLimitRules 批量导入限流规则。
func (s *Service) ImportRateLimitRules(list []model.RateLimitRule) (int, []string) {
	created := 0
	var errs []string
	for _, r := range list {
		if _, err := s.CreateRateLimitRule(r); err != nil {
			errs = append(errs, r.Name+": "+err.Error())
			continue
		}
		created++
	}
	return created, errs
}
