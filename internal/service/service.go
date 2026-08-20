// Package service 实现业务逻辑层。
package service

import (
	"sync"
	"time"

	"apigateway/internal/circuitbreaker"
	"apigateway/internal/config"
	"apigateway/internal/model"
	"apigateway/internal/ratelimit"
	"apigateway/internal/store"
	"apigateway/pkg/logger"
)

// Service 业务服务，聚合 store、限流器与熔断器缓存。
type Service struct {
	store    store.Store
	log      *logger.Logger
	cfg      *config.Config

	mu       sync.Mutex
	buckets  map[string]*ratelimit.TokenBucket            // 路由 ID -> 令牌桶
	breakers map[string]*circuitbreaker.CircuitBreaker    // 服务 ID -> 熔断器
}

// New 构造业务服务。
func New(st store.Store, log *logger.Logger, cfg *config.Config) *Service {
	return &Service{
		store:    st,
		log:      log,
		cfg:      cfg,
		buckets:  make(map[string]*ratelimit.TokenBucket),
		breakers: make(map[string]*circuitbreaker.CircuitBreaker),
	}
}

// paginate 通用分页截取。
func paginate[T any](list []T, page, size int) ([]T, int, error) {
	total := len(list)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	start := (page - 1) * size
	if start >= total {
		return []T{}, total, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return list[start:end], total, nil
}

// BucketFor 获取（或创建）某路由对应的令牌桶。
func (s *Service) BucketFor(rule *model.RateLimitRule) *ratelimit.TokenBucket {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.buckets[rule.ID]
	if !ok {
		b = ratelimit.NewTokenBucket(rule.RatePerSecond(), rule.Limit)
		s.buckets[rule.ID] = b
	}
	return b
}

// ResetBuckets 清空限流桶缓存（规则变更后调用）。
func (s *Service) ResetBuckets() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buckets = make(map[string]*ratelimit.TokenBucket)
}

// BreakerFor 获取（或创建）某上游服务的熔断器。
func (s *Service) BreakerFor(serviceID string) *circuitbreaker.CircuitBreaker {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.breakers[serviceID]
	if !ok {
		b = circuitbreaker.New(5, 30*time.Second)
		s.breakers[serviceID] = b
	}
	return b
}

// AllowService 判断某上游服务是否允许转发（熔断器未打开）。
func (s *Service) AllowService(serviceID string) bool {
	return s.BreakerFor(serviceID).Allow()
}

// RecordServiceResult 记录一次上游转发结果，成功关闭熔断器、失败累计失败次数。
func (s *Service) RecordServiceResult(serviceID string, ok bool) {
	cb := s.BreakerFor(serviceID)
	if ok {
		cb.RecordSuccess()
	} else {
		cb.RecordFailure()
	}
}
