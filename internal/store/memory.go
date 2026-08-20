package store

import (
	"sync"

	"apigateway/internal/model"
)

// MemoryStore 基于内存 map 的 Store 实现，线程安全。
type MemoryStore struct {
	mu             sync.RWMutex
	services       map[string]*model.Service
	routes         map[string]*model.Route
	apps           map[string]*model.App
	apiKeys        map[string]*model.APIKey
	rateLimitRules map[string]*model.RateLimitRule
	requestLogs    map[string]*model.RequestLog
	healthChecks   map[string]*model.HealthCheck
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		services:       make(map[string]*model.Service),
		routes:         make(map[string]*model.Route),
		apps:           make(map[string]*model.App),
		apiKeys:        make(map[string]*model.APIKey),
		rateLimitRules: make(map[string]*model.RateLimitRule),
		requestLogs:    make(map[string]*model.RequestLog),
		healthChecks:   make(map[string]*model.HealthCheck),
	}
}

var _ Store = (*MemoryStore)(nil)
