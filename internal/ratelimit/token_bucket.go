// Package ratelimit 提供令牌桶限流算法。
package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket 令牌桶限流器。
// rate 为每秒补充的令牌数，burst 为桶容量（允许的突发请求数）。
type TokenBucket struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

// NewTokenBucket 创建一个令牌桶。rate 需大于 0，burst 需大于等于 1。
func NewTokenBucket(rate float64, burst int) *TokenBucket {
	if rate <= 0 {
		rate = 1
	}
	if burst < 1 {
		burst = 1
	}
	return &TokenBucket{
		rate:   rate,
		burst:  float64(burst),
		tokens: float64(burst),
		last:   time.Now(),
	}
}

// Allow 尝试取走一个令牌。取到返回 true，否则返回 false。
func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// Tokens 返回当前可用令牌数（主要用于测试与观测）。
func (b *TokenBucket) Tokens() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()
	return b.tokens
}

// refill 根据经过的时间补充令牌，最多补到 burst。
func (b *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	if elapsed <= 0 {
		return
	}
	b.tokens += elapsed * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.last = now
}

// Reset 重置令牌桶到满容量。
func (b *TokenBucket) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens = b.burst
	b.last = time.Now()
}
