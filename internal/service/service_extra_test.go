package service

import (
	"testing"
	"time"

	"apigateway/internal/circuitbreaker"
	"apigateway/internal/model"
)

func TestBreakerForCachesInstance(t *testing.T) {
	s := newTestService()
	b1 := s.BreakerFor("svc1")
	b2 := s.BreakerFor("svc1")
	if b1 != b2 {
		t.Fatal("breaker should be cached per service")
	}
	b3 := s.BreakerFor("svc2")
	if b1 == b3 {
		t.Fatal("different services should have different breakers")
	}
}

func TestAllowServiceAndRecord(t *testing.T) {
	s := newTestService()
	if !s.AllowService("svc1") {
		t.Fatal("initial should allow")
	}
	// 连续失败 5 次触发熔断
	for i := 0; i < 5; i++ {
		s.RecordServiceResult("svc1", false)
	}
	if s.AllowService("svc1") {
		t.Fatal("should trip after 5 failures")
	}
	if s.BreakerFor("svc1").State() != circuitbreaker.StateOpen {
		t.Fatalf("state = %s", s.BreakerFor("svc1").State())
	}
}

func TestRecordServiceSuccessCloses(t *testing.T) {
	s := newTestService()
	for i := 0; i < 5; i++ {
		s.RecordServiceResult("svc1", false)
	}
	if s.BreakerFor("svc1").State() != circuitbreaker.StateOpen {
		t.Fatal("should be open")
	}
	s.RecordServiceResult("svc1", true)
	if s.BreakerFor("svc1").State() != circuitbreaker.StateClosed {
		t.Fatal("should close after success")
	}
}

func TestServiceListPagination(t *testing.T) {
	s := newTestService()
	for i := 0; i < 5; i++ {
		mustCreateService(t, s, "svc"+string(rune('a'+i)))
	}
	items, total, _ := s.ListServices(model.ServiceFilter{}, 1, 2)
	if total != 5 || len(items) != 2 {
		t.Fatalf("page1 total=%d len=%d", total, len(items))
	}
	items, _, _ = s.ListServices(model.ServiceFilter{}, 3, 2)
	if len(items) != 1 {
		t.Fatalf("page3 len=%d", len(items))
	}
	items, total, _ = s.ListServices(model.ServiceFilter{}, 99, 2)
	if total != 5 || len(items) != 0 {
		t.Fatalf("page99 total=%d len=%d", total, len(items))
	}
}

func TestServiceListFilter(t *testing.T) {
	s := newTestService()
	mustCreateService(t, s, "user-service")
	mustCreateService(t, s, "order-service")
	// 停用一个
	v, _ := s.GetServiceByName("order-service")
	s.UpdateService(v.ID, model.Service{Name: "order-service", BaseURL: "http://x", Status: model.ServiceInactive})

	items, total, _ := s.ListServices(model.ServiceFilter{Status: model.ServiceActive}, 1, 20)
	if total != 1 || items[0].Name != "user-service" {
		t.Fatalf("active total=%d", total)
	}
	items, total, _ = s.ListServices(model.ServiceFilter{Keyword: "order"}, 1, 20)
	if total != 1 || items[0].Name != "order-service" {
		t.Fatalf("keyword total=%d", total)
	}
}

func TestUpdateServiceValidation(t *testing.T) {
	s := newTestService()
	v := mustCreateService(t, s, "user")
	// 更新为非法超时
	if _, err := s.UpdateService(v.ID, model.Service{Name: "user", BaseURL: "http://x", Timeout: 999}); err == nil {
		t.Fatal("expect error for too-large timeout")
	}
	// 更新不存在的服务
	if _, err := s.UpdateService("nope", model.Service{Name: "x", BaseURL: "http://x"}); err == nil {
		t.Fatal("expect error for missing service")
	}
}

func TestUpdateRouteValidation(t *testing.T) {
	s := newTestService()
	svc := mustCreateService(t, s, "user")
	r, _ := s.CreateRoute(model.Route{Path: "/x", Method: "GET", ServiceID: svc.ID})
	// 引用不存在的服务
	if _, err := s.UpdateRoute(r.ID, model.Route{Path: "/x", Method: "GET", ServiceID: "nope"}); err == nil {
		t.Fatal("expect error for missing service")
	}
	// 引用不存在的限流规则
	if _, err := s.UpdateRoute(r.ID, model.Route{Path: "/x", Method: "GET", ServiceID: svc.ID, RateLimitRuleID: "nope"}); err == nil {
		t.Fatal("expect error for missing rule")
	}
}

func TestListHealthChecks(t *testing.T) {
	s := newTestService()
	mustCreateService(t, s, "svc1")
	mustCreateService(t, s, "svc2")
	items, total, _ := s.ListHealthChecks(1, 20)
	if total != 2 || len(items) != 2 {
		t.Fatalf("health checks total=%d", total)
	}
}

func TestGetHealthCheckMissing(t *testing.T) {
	s := newTestService()
	if _, err := s.GetHealthCheck("nope"); err == nil {
		t.Fatal("expect error for missing health check")
	}
}

func TestProbeServiceMissing(t *testing.T) {
	s := newTestService()
	if _, err := s.ProbeService("nope"); err == nil {
		t.Fatal("expect error for missing service")
	}
}

func TestCreateAppDuplicate(t *testing.T) {
	s := newTestService()
	s.CreateApp(model.App{Name: "app"})
	if _, err := s.CreateApp(model.App{Name: "app"}); err == nil {
		t.Fatal("expect conflict for duplicate app")
	}
}

func TestUpdateAppValidation(t *testing.T) {
	s := newTestService()
	a, _ := s.CreateApp(model.App{Name: "app"})
	if _, err := s.UpdateApp(a.ID, model.App{Name: "app", Status: "bad"}); err == nil {
		t.Fatal("expect error for bad status")
	}
	if _, err := s.UpdateApp("nope", model.App{Name: "x"}); err == nil {
		t.Fatal("expect error for missing app")
	}
}

func TestUpdateAPIKeyStatusValidation(t *testing.T) {
	s := newTestService()
	app, _ := s.CreateApp(model.App{Name: "app"})
	key, _ := s.CreateAPIKey(app.ID, "s", time.Time{})
	if _, err := s.UpdateAPIKeyStatus(key.ID, "bad"); err == nil {
		t.Fatal("expect error for bad status")
	}
	if _, err := s.UpdateAPIKeyStatus("nope", model.APIKeyActive); err == nil {
		t.Fatal("expect error for missing key")
	}
}

func TestRateLimitRuleListPagination(t *testing.T) {
	s := newTestService()
	for i := 0; i < 4; i++ {
		s.CreateRateLimitRule(model.RateLimitRule{Name: "rl" + string(rune('a'+i)), Limit: 10, Window: 60})
	}
	items, total, _ := s.ListRateLimitRules(1, 3)
	if total != 4 || len(items) != 3 {
		t.Fatalf("total=%d len=%d", total, len(items))
	}
}

func TestRouteListFiltered(t *testing.T) {
	s := newTestService()
	svc := mustCreateService(t, s, "user")
	s.CreateRoute(model.Route{Path: "/a", Method: "GET", ServiceID: svc.ID})
	s.CreateRoute(model.Route{Path: "/b", Method: "POST", ServiceID: svc.ID})

	items, total, _ := s.ListRoutes(model.RouteFilter{Method: "GET"}, 1, 20)
	if total != 1 || items[0].Path != "/a" {
		t.Fatalf("GET routes total=%d", total)
	}
	items, total, _ = s.ListRoutes(model.RouteFilter{ServiceID: svc.ID}, 1, 20)
	if total != 2 {
		t.Fatalf("service routes total=%d", total)
	}
}

func TestMatchRouteExactPath(t *testing.T) {
	s := newTestService()
	svc := mustCreateService(t, s, "user")
	r, _ := s.CreateRoute(model.Route{Path: "/exact", Method: "GET", ServiceID: svc.ID})
	// 精确匹配
	got, ok := s.MatchRoute("GET", "/exact")
	if !ok || got.ID != r.ID {
		t.Fatalf("exact match failed: %v", got)
	}
	// 子路径匹配
	if _, ok := s.MatchRoute("GET", "/exact/sub"); !ok {
		t.Fatal("sub-path should match")
	}
	// 前缀错误不匹配
	if _, ok := s.MatchRoute("GET", "/exactly"); ok {
		t.Fatal("/exactly should not match /exact")
	}
}
