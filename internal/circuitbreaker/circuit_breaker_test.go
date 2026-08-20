package circuitbreaker

import (
	"testing"
	"time"
)

func TestCircuitBreakerStartsClosed(t *testing.T) {
	cb := New(3, time.Second)
	if cb.State() != StateClosed {
		t.Fatalf("state = %s, want closed", cb.State())
	}
	if !cb.Allow() {
		t.Fatal("closed should allow")
	}
}

func TestCircuitBreakerOpensAfterThreshold(t *testing.T) {
	cb := New(3, time.Hour)
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if cb.State() != StateOpen {
		t.Fatalf("state = %s, want open", cb.State())
	}
	if cb.Allow() {
		t.Fatal("open should reject")
	}
}

func TestCircuitBreakerResetTimeout(t *testing.T) {
	cb := New(2, 20*time.Millisecond)
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatalf("state = %s, want open", cb.State())
	}
	// 等待冷却期结束
	time.Sleep(30 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("after reset timeout should allow (half-open)")
	}
	if cb.State() != StateHalfOpen {
		t.Fatalf("state = %s, want half-open", cb.State())
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	cb := New(2, time.Hour)
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatalf("state = %s, want open", cb.State())
	}
	// 半开状态下成功一次即恢复关闭
	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.mu.Unlock()
	cb.RecordSuccess()
	if cb.State() != StateClosed {
		t.Fatalf("state = %s, want closed", cb.State())
	}
	if cb.FailureCount() != 0 {
		t.Fatalf("failure count = %d, want 0", cb.FailureCount())
	}
}

func TestCircuitBreakerHalfOpenFailureReopens(t *testing.T) {
	cb := New(2, time.Hour)
	cb.RecordFailure()
	cb.RecordFailure()
	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.mu.Unlock()
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatalf("state = %s, want open after half-open failure", cb.State())
	}
}

func TestCircuitBreakerSuccessResetsCount(t *testing.T) {
	cb := New(3, time.Hour)
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordSuccess()
	if cb.FailureCount() != 0 {
		t.Fatalf("failure count = %d, want 0", cb.FailureCount())
	}
	// 再失败一次不应打开
	cb.RecordFailure()
	if cb.State() != StateClosed {
		t.Fatalf("state = %s, want closed", cb.State())
	}
}

func TestCircuitBreakerDefaults(t *testing.T) {
	cb := New(0, 0)
	if cb.State() != StateClosed {
		t.Fatal("should start closed")
	}
	// 默认阈值 5
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	if cb.State() != StateOpen {
		t.Fatal("should open after default 5 failures")
	}
}
