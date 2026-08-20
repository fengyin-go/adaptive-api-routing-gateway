package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"apigateway/internal/model"
)

func TestPingSuccess(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	s := newTestService()
	ok, latency := s.ping(upstream.URL, 2)
	if !ok {
		t.Fatal("ping should succeed")
	}
	if latency < 0 {
		t.Fatalf("latency = %d", latency)
	}
}

func TestPingFailure(t *testing.T) {
	s := newTestService()
	ok, _ := s.ping("http://127.0.0.1:1", 1)
	if ok {
		t.Fatal("ping should fail for unreachable host")
	}
}

func TestPingServerError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()

	s := newTestService()
	// 500 视为失败（>= 500）
	ok, _ := s.ping(upstream.URL, 2)
	if ok {
		t.Fatal("500 should be treated as failure")
	}
}

func TestPingTimeoutDefault(t *testing.T) {
	s := newTestService()
	// timeout=0 时应使用默认 5 秒，不会 panic
	ok, _ := s.ping("http://127.0.0.1:1", 0)
	if ok {
		t.Fatal("unreachable should fail")
	}
}

func TestStartHealthCheckerRuns(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	s := newTestService()
	v, _ := s.CreateService(model.Service{Name: "hc", BaseURL: upstream.URL, Timeout: 2})

	ctx, cancel := context.WithCancel(context.Background())
	s.StartHealthChecker(ctx, 20*time.Millisecond)

	// 等待至少一次探测
	time.Sleep(60 * time.Millisecond)
	cancel()

	hc, err := s.GetHealthCheck(v.ID)
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	if hc.Status != model.HealthHealthy {
		t.Fatalf("status = %s, want healthy", hc.Status)
	}
	if hc.LastChecked.IsZero() {
		t.Fatal("last checked should be updated")
	}
}

func TestHealthCheckerSkipsInactive(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	s := newTestService()
	v, _ := s.CreateService(model.Service{Name: "inactive", BaseURL: upstream.URL, Status: model.ServiceInactive})

	// 记录初始 LastChecked
	before, _ := s.GetHealthCheck(v.ID)

	ctx, cancel := context.WithCancel(context.Background())
	s.StartHealthChecker(ctx, 20*time.Millisecond)
	time.Sleep(60 * time.Millisecond)
	cancel()

	after, _ := s.GetHealthCheck(v.ID)
	if !after.LastChecked.Equal(before.LastChecked) {
		t.Fatal("inactive service should not be probed")
	}
}
