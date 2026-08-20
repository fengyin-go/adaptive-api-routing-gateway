// Package circuitbreaker 提供熔断器，用于保护上游服务。
package circuitbreaker

import (
	"sync"
	"time"
)

// State 熔断器状态。
type State int

const (
	// StateClosed 关闭状态，正常放行请求。
	StateClosed State = iota
	// StateOpen 打开状态，快速失败，拒绝请求。
	StateOpen
	// StateHalfOpen 半开状态，放行少量探测请求。
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker 基于连续失败次数与冷却时间的熔断器。
type CircuitBreaker struct {
	mu                  sync.Mutex
	state               State
	failureThreshold    int
	resetTimeout        time.Duration
	consecutiveFailures int
	lastFailure         time.Time
}

// New 创建熔断器。failureThreshold 为连续失败次数阈值，resetTimeout 为打开后的冷却时间。
func New(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 5
	}
	if resetTimeout <= 0 {
		resetTimeout = 30 * time.Second
	}
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
	}
}

// Allow 判断当前是否放行请求。
// 关闭状态直接放行；打开状态在冷却期结束后转入半开并放行；半开状态仅放行单个探测请求。
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailure) >= cb.resetTimeout {
			cb.state = StateHalfOpen
			return true
		}
		return false
	case StateHalfOpen:
		return false
	default:
		return true
	}
}

// RecordSuccess 记录一次成功，关闭熔断器并清零失败计数。
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.consecutiveFailures = 0
	cb.state = StateClosed
}

// RecordFailure 记录一次失败，达到阈值则打开熔断器。
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.consecutiveFailures++
	cb.lastFailure = time.Now()
	if cb.state == StateHalfOpen || cb.consecutiveFailures >= cb.failureThreshold {
		cb.state = StateOpen
	}
}

// State 返回当前状态。
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// FailureCount 返回当前连续失败次数。
func (cb *CircuitBreaker) FailureCount() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.consecutiveFailures
}
