package store

import (
	"errors"
	"testing"

	"apigateway/internal/model"
)

func TestServiceStoreEdgeCases(t *testing.T) {
	s := NewMemoryStore()
	// 删除不存在
	if err := s.DeleteService("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	// 更新不存在
	if err := s.UpdateService(&model.Service{ID: "nope"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	// 按名查不存在
	if _, err := s.GetServiceByName("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
}

func TestRouteStoreEdgeCases(t *testing.T) {
	s := NewMemoryStore()
	if err := s.DeleteRoute("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	if err := s.UpdateRoute(&model.Route{ID: "nope"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	// 更新导致冲突
	s.CreateRoute(&model.Route{ID: "r1", Path: "/a", Method: "GET", ServiceID: "s1"})
	s.CreateRoute(&model.Route{ID: "r2", Path: "/a", Method: "POST", ServiceID: "s1"})
	if err := s.UpdateRoute(&model.Route{ID: "r2", Path: "/a", Method: "GET", ServiceID: "s1"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expect conflict, got %v", err)
	}
}

func TestAppStoreEdgeCases(t *testing.T) {
	s := NewMemoryStore()
	if err := s.DeleteApp("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	if _, err := s.GetAppByName("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
}

func TestAPIKeyStoreEdgeCases(t *testing.T) {
	s := NewMemoryStore()
	if err := s.DeleteAPIKey("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	if err := s.UpdateAPIKey(&model.APIKey{ID: "nope"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	// 更新导致 key 冲突
	s.CreateAPIKey(&model.APIKey{ID: "k1", AppID: "a", Key: "ak1"})
	s.CreateAPIKey(&model.APIKey{ID: "k2", AppID: "a", Key: "ak2"})
	if err := s.UpdateAPIKey(&model.APIKey{ID: "k2", AppID: "a", Key: "ak1"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expect conflict, got %v", err)
	}
}

func TestRateLimitRuleEdgeCases(t *testing.T) {
	s := NewMemoryStore()
	if err := s.DeleteRateLimitRule("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	if err := s.UpdateRateLimitRule(&model.RateLimitRule{ID: "nope"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
}

func TestHealthCheckEdgeCases(t *testing.T) {
	s := NewMemoryStore()
	if err := s.UpdateHealthCheck(&model.HealthCheck{ID: "nope"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	if _, err := s.GetHealthCheck("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
}

func TestRequestLogEdgeCases(t *testing.T) {
	s := NewMemoryStore()
	if _, err := s.GetRequestLog("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	// 空列表
	if len(s.ListRequestLogs()) != 0 {
		t.Fatal("empty list expected")
	}
}

func TestListReturnsNewSlice(t *testing.T) {
	s := NewMemoryStore()
	s.CreateService(&model.Service{ID: "s1", Name: "a", BaseURL: "http://x"})
	s.CreateService(&model.Service{ID: "s2", Name: "b", BaseURL: "http://x"})
	list := s.ListServices()
	// 对返回切片做截断不应影响内部 map 的记录数
	_ = list[:1]
	if len(s.ListServices()) != 2 {
		t.Fatal("internal store should still have 2 services")
	}
}
