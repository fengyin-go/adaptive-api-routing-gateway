package store

import (
	"fmt"
	"sync"
	"testing"

	"apigateway/internal/model"
)

func TestConcurrentServiceAccess(t *testing.T) {
	s := NewMemoryStore()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			svc := &model.Service{ID: fmt.Sprintf("s%d", n), Name: fmt.Sprintf("svc%d", n), BaseURL: "http://x"}
			if err := s.CreateService(svc); err != nil {
				t.Errorf("create %d: %v", n, err)
				return
			}
			if _, err := s.GetService(svc.ID); err != nil {
				t.Errorf("get %d: %v", n, err)
			}
		}(i)
	}
	wg.Wait()
	if len(s.ListServices()) != 20 {
		t.Fatalf("services = %d, want 20", len(s.ListServices()))
	}
}

func TestConcurrentRequestLogWrite(t *testing.T) {
	s := NewMemoryStore()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			l := &model.RequestLog{ID: fmt.Sprintf("l%d", n), RouteID: "r1", Method: "GET", Path: "/x", StatusCode: 200}
			if err := s.CreateRequestLog(l); err != nil {
				t.Errorf("create log %d: %v", n, err)
			}
		}(i)
	}
	wg.Wait()
	if len(s.ListRequestLogs()) != 50 {
		t.Fatalf("logs = %d, want 50", len(s.ListRequestLogs()))
	}
}

func TestConcurrentHealthCheckUpdate(t *testing.T) {
	s := NewMemoryStore()
	h := &model.HealthCheck{ID: "h1", ServiceID: "s1", Status: model.HealthHealthy}
	if err := s.CreateHealthCheck(h); err != nil {
		t.Fatalf("create: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			got, err := s.GetHealthCheckByService("s1")
			if err != nil {
				t.Errorf("get: %v", err)
				return
			}
			got.FailureCount = n
			_ = s.UpdateHealthCheck(got)
		}(i)
	}
	wg.Wait()
	// 并发更新不应 panic，最终状态有效
	if _, err := s.GetHealthCheckByService("s1"); err != nil {
		t.Fatalf("final get: %v", err)
	}
}
