package store

import (
	"errors"
	"testing"
	"time"

	"apigateway/internal/model"
)

func TestServiceStoreCRUD(t *testing.T) {
	s := NewMemoryStore()
	v := &model.Service{ID: "s1", Name: "user", BaseURL: "http://x", Status: model.ServiceActive}
	if err := s.CreateService(v); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.CreateService(&model.Service{ID: "s2", Name: "user", BaseURL: "http://y"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expect conflict, got %v", err)
	}
	got, err := s.GetService("s1")
	if err != nil || got.Name != "user" {
		t.Fatalf("get: %v", err)
	}
	gotByName, err := s.GetServiceByName("user")
	if err != nil || gotByName.ID != "s1" {
		t.Fatalf("get by name: %v", err)
	}
	if len(s.ListServices()) != 1 {
		t.Fatalf("list len = %d", len(s.ListServices()))
	}
	got.Name = "user2"
	if err := s.UpdateService(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := s.DeleteService("s1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetService("s1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
}

func TestRouteStoreCRUD(t *testing.T) {
	s := NewMemoryStore()
	r := &model.Route{ID: "r1", Path: "/users", Method: "GET", ServiceID: "s1"}
	if err := s.CreateRoute(r); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.CreateRoute(&model.Route{ID: "r2", Path: "/users", Method: "GET", ServiceID: "s2"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expect conflict, got %v", err)
	}
	// 不同 method 不冲突
	if err := s.CreateRoute(&model.Route{ID: "r3", Path: "/users", Method: "POST", ServiceID: "s2"}); err != nil {
		t.Fatalf("create POST: %v", err)
	}
	if _, err := s.GetRoute("r1"); err != nil {
		t.Fatalf("get: %v", err)
	}
	r.Method = "PUT"
	if err := s.UpdateRoute(r); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := s.DeleteRoute("r1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetRoute("r1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
}

func TestAppStoreCRUD(t *testing.T) {
	s := NewMemoryStore()
	a := &model.App{ID: "a1", Name: "app1", Status: model.AppActive}
	if err := s.CreateApp(a); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.CreateApp(&model.App{ID: "a2", Name: "app1"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expect conflict, got %v", err)
	}
	got, err := s.GetAppByName("app1")
	if err != nil || got.ID != "a1" {
		t.Fatalf("get by name: %v", err)
	}
	a.Status = model.AppDisabled
	if err := s.UpdateApp(a); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := s.DeleteApp("a1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetApp("a1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
}

func TestAPIKeyStoreCRUD(t *testing.T) {
	s := NewMemoryStore()
	k := &model.APIKey{ID: "k1", AppID: "a1", Key: "ak_abc", Status: model.APIKeyActive}
	if err := s.CreateAPIKey(k); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.CreateAPIKey(&model.APIKey{ID: "k2", AppID: "a1", Key: "ak_abc"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expect conflict, got %v", err)
	}
	got, err := s.GetAPIKeyByKey("ak_abc")
	if err != nil || got.ID != "k1" {
		t.Fatalf("get by key: %v", err)
	}
	if _, err := s.GetAPIKeyByKey("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	k.Status = model.APIKeyDisabled
	if err := s.UpdateAPIKey(k); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := s.DeleteAPIKey("k1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetAPIKey("k1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
}

func TestRateLimitRuleStoreCRUD(t *testing.T) {
	s := NewMemoryStore()
	r := &model.RateLimitRule{ID: "rl1", Name: "r1", Limit: 100, Window: 60}
	if err := s.CreateRateLimitRule(r); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.CreateRateLimitRule(&model.RateLimitRule{ID: "rl2", Name: "r1", Limit: 10}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expect conflict, got %v", err)
	}
	if _, err := s.GetRateLimitRule("rl1"); err != nil {
		t.Fatalf("get: %v", err)
	}
	r.Limit = 200
	if err := s.UpdateRateLimitRule(r); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := s.DeleteRateLimitRule("rl1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetRateLimitRule("rl1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
}

func TestRequestLogStore(t *testing.T) {
	s := NewMemoryStore()
	l := &model.RequestLog{ID: "l1", RouteID: "r1", Method: "GET", Path: "/x", StatusCode: 200, Timestamp: time.Now()}
	if err := s.CreateRequestLog(l); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.GetRequestLog("l1"); err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(s.ListRequestLogs()) != 1 {
		t.Fatalf("list len = %d", len(s.ListRequestLogs()))
	}
	if _, err := s.GetRequestLog("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
}

func TestHealthCheckStore(t *testing.T) {
	s := NewMemoryStore()
	h := &model.HealthCheck{ID: "h1", ServiceID: "s1", Status: model.HealthHealthy, LastChecked: time.Now()}
	if err := s.CreateHealthCheck(h); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := s.GetHealthCheckByService("s1")
	if err != nil || got.ID != "h1" {
		t.Fatalf("get by service: %v", err)
	}
	if _, err := s.GetHealthCheckByService("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	h.Status = model.HealthUnhealthy
	if err := s.UpdateHealthCheck(h); err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(s.ListHealthChecks()) != 1 {
		t.Fatalf("list len = %d", len(s.ListHealthChecks()))
	}
}

func TestStoreNotFoundOnMissing(t *testing.T) {
	s := NewMemoryStore()
	if _, err := s.GetService("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	if err := s.DeleteRoute("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	if err := s.UpdateApp(&model.App{ID: "nope"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
	if err := s.UpdateHealthCheck(&model.HealthCheck{ID: "nope"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expect not found, got %v", err)
	}
}
