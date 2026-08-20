package ratelimit

import (
	"testing"
	"time"
)

func TestTokenBucketInitialFull(t *testing.T) {
	b := NewTokenBucket(1, 5)
	if got := b.Tokens(); got != 5 {
		t.Fatalf("initial tokens = %v, want 5", got)
	}
}

func TestTokenBucketAllowDrains(t *testing.T) {
	b := NewTokenBucket(0, 3) // rate 会被归一化为 1
	for i := 0; i < 3; i++ {
		if !b.Allow() {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	if b.Allow() {
		t.Fatal("4th request should be denied")
	}
}

func TestTokenBucketRefill(t *testing.T) {
	b := NewTokenBucket(100, 2) // 每秒 100 令牌，burst 2
	// 先耗尽
	b.Allow()
	b.Allow()
	if b.Allow() {
		t.Fatal("bucket should be empty")
	}
	// 等待足够时间补充
	time.Sleep(30 * time.Millisecond)
	if !b.Allow() {
		t.Fatal("bucket should be refilled after wait")
	}
}

func TestTokenBucketBurstCap(t *testing.T) {
	b := NewTokenBucket(1000, 3)
	time.Sleep(50 * time.Millisecond)
	// 长时间不取，令牌也不会超过 burst
	if got := b.Tokens(); got > 3 {
		t.Fatalf("tokens = %v, should cap at burst 3", got)
	}
}

func TestTokenBucketReset(t *testing.T) {
	b := NewTokenBucket(0, 2)
	b.Allow()
	b.Allow()
	if b.Allow() {
		t.Fatal("third allow should fail (bucket empty)")
	}
	b.Reset()
	if got := b.Tokens(); got != 2 {
		t.Fatalf("tokens after reset = %v, want 2", got)
	}
}

func TestTokenBucketDefaultRateAndBurst(t *testing.T) {
	b := NewTokenBucket(0, 0)
	if got := b.Tokens(); got < 1 {
		t.Fatalf("default burst should be >= 1, got %v", got)
	}
	if !b.Allow() {
		t.Fatal("first allow should succeed with default burst")
	}
}

func TestTokenBucketConcurrent(t *testing.T) {
	b := NewTokenBucket(10000, 1000)
	allowed := 0
	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 20; j++ {
				if b.Allow() {
					allowed++
				}
			}
		}()
	}
	for i := 0; i < 50; i++ {
		<-done
	}
	if allowed > 1000 {
		t.Fatalf("allowed = %d, should not exceed burst 1000", allowed)
	}
}
