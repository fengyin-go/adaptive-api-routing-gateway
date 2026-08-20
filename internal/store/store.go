// Package store 定义数据访问接口与内存实现。
package store

import (
	"errors"

	"apigateway/internal/model"
)

var (
	ErrNotFound = errors.New("记录不存在")
	ErrConflict = errors.New("记录已存在或状态冲突")
)

// Store 聚合全部实体的数据访问方法，便于测试时替换实现。
type Store interface {
	CreateService(s *model.Service) error
	GetService(id string) (*model.Service, error)
	GetServiceByName(name string) (*model.Service, error)
	ListServices() []*model.Service
	UpdateService(s *model.Service) error
	DeleteService(id string) error

	CreateRoute(r *model.Route) error
	GetRoute(id string) (*model.Route, error)
	ListRoutes() []*model.Route
	UpdateRoute(r *model.Route) error
	DeleteRoute(id string) error

	CreateApp(a *model.App) error
	GetApp(id string) (*model.App, error)
	GetAppByName(name string) (*model.App, error)
	ListApps() []*model.App
	UpdateApp(a *model.App) error
	DeleteApp(id string) error

	CreateAPIKey(k *model.APIKey) error
	GetAPIKey(id string) (*model.APIKey, error)
	GetAPIKeyByKey(key string) (*model.APIKey, error)
	ListAPIKeys() []*model.APIKey
	UpdateAPIKey(k *model.APIKey) error
	DeleteAPIKey(id string) error

	CreateRateLimitRule(r *model.RateLimitRule) error
	GetRateLimitRule(id string) (*model.RateLimitRule, error)
	ListRateLimitRules() []*model.RateLimitRule
	UpdateRateLimitRule(r *model.RateLimitRule) error
	DeleteRateLimitRule(id string) error

	CreateRequestLog(l *model.RequestLog) error
	GetRequestLog(id string) (*model.RequestLog, error)
	ListRequestLogs() []*model.RequestLog

	CreateHealthCheck(h *model.HealthCheck) error
	GetHealthCheck(id string) (*model.HealthCheck, error)
	GetHealthCheckByService(serviceID string) (*model.HealthCheck, error)
	ListHealthChecks() []*model.HealthCheck
	UpdateHealthCheck(h *model.HealthCheck) error
}
