package service

import (
	"testing"
	"time"

	"apigateway/internal/model"
)

func TestAllowRequestNoRule(t *testing.T) {
	s := newTestService()
	// 无规则的路由默认放行
	if !s.AllowRequest(&model.Route{RateLimitRuleID: ""}) {
		t.Fatal("route without rule should be allowed")
	}
	// 规则不存在也放行
	if !s.AllowRequest(&model.Route{RateLimitRuleID: "nope"}) {
		t.Fatal("route with missing rule should be allowed")
	}
}

func TestAllowRequestWithRule(t *testing.T) {
	s := newTestService()
	svc := mustCreateService(t, s, "user")
	rule, _ := s.CreateRateLimitRule(model.RateLimitRule{Name: "rl", Limit: 2, Window: 60})
	route, _ := s.CreateRoute(model.Route{Path: "/x", Method: "GET", ServiceID: svc.ID, RateLimitRuleID: rule.ID})

	if !s.AllowRequest(route) || !s.AllowRequest(route) {
		t.Fatal("first two requests should be allowed")
	}
	if s.AllowRequest(route) {
		t.Fatal("third request should be limited")
	}
}

func TestRateLimitRuleUpdateResetsBuckets(t *testing.T) {
	s := newTestService()
	svc := mustCreateService(t, s, "user")
	rule, _ := s.CreateRateLimitRule(model.RateLimitRule{Name: "rl", Limit: 1, Window: 60})
	route, _ := s.CreateRoute(model.Route{Path: "/x", Method: "GET", ServiceID: svc.ID, RateLimitRuleID: rule.ID})

	s.AllowRequest(route) // 耗尽
	if s.AllowRequest(route) {
		t.Fatal("should be limited")
	}
	// 更新规则触发桶重置
	s.UpdateRateLimitRule(rule.ID, model.RateLimitRule{Name: "rl", Limit: 10, Window: 60})
	if !s.AllowRequest(route) {
		t.Fatal("should be allowed after rule update resets bucket")
	}
}

func TestRecordAndListRequestLogs(t *testing.T) {
	s := newTestService()
	s.RecordRequest(model.RequestLog{RouteID: "r1", Method: "GET", Path: "/x", StatusCode: 200, LatencyMs: 10})
	s.RecordRequest(model.RequestLog{RouteID: "r1", Method: "POST", Path: "/x", StatusCode: 500, LatencyMs: 20})
	s.RecordRequest(model.RequestLog{RouteID: "r2", Method: "GET", Path: "/y", StatusCode: 429, LatencyMs: 5})

	items, total, _ := s.ListRequestLogs(model.RequestLogFilter{}, 1, 20)
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	items, total, _ = s.ListRequestLogs(model.RequestLogFilter{RouteID: "r1"}, 1, 20)
	if total != 2 {
		t.Fatalf("r1 total = %d, want 2", total)
	}
	items, total, _ = s.ListRequestLogs(model.RequestLogFilter{StatusCode: 500}, 1, 20)
	if total != 1 || items[0].StatusCode != 500 {
		t.Fatalf("500 total = %d", total)
	}
}

func TestGetGatewayStats(t *testing.T) {
	s := newTestService()
	svc := mustCreateService(t, s, "user")
	app, _ := s.CreateApp(model.App{Name: "app1"})
	s.CreateAPIKey(app.ID, "s", zero())
	route, _ := s.CreateRoute(model.Route{Path: "/users", Method: "GET", ServiceID: svc.ID, RequiresAuth: true})

	// 制造日志：2 成功、1 客户端错误、1 限流、1 服务端错误
	s.RecordRequest(model.RequestLog{RouteID: route.ID, AppID: app.ID, Method: "GET", Path: "/users", StatusCode: 200})
	s.RecordRequest(model.RequestLog{RouteID: route.ID, AppID: app.ID, Method: "GET", Path: "/users", StatusCode: 200})
	s.RecordRequest(model.RequestLog{RouteID: route.ID, AppID: app.ID, Method: "GET", Path: "/users", StatusCode: 404})
	s.RecordRequest(model.RequestLog{RouteID: route.ID, AppID: app.ID, Method: "GET", Path: "/users", StatusCode: 429})
	s.RecordRequest(model.RequestLog{RouteID: route.ID, AppID: app.ID, Method: "GET", Path: "/users", StatusCode: 500})

	stats, err := s.GetGatewayStats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Services != 1 || stats.Routes != 1 || stats.Apps != 1 || stats.APIKeys != 1 {
		t.Fatalf("counts = %+v", stats)
	}
	if stats.Requests.Total != 5 || stats.Requests.Success != 2 || stats.Requests.ClientErr != 1 || stats.Requests.Limited != 1 || stats.Requests.ServerErr != 1 {
		t.Fatalf("request counts = %+v", stats.Requests)
	}
	if len(stats.TopRoutes) != 1 || stats.TopRoutes[0].Requests != 5 {
		t.Fatalf("top routes = %+v", stats.TopRoutes)
	}
}

func TestExportConfig(t *testing.T) {
	s := newTestService()
	mustCreateService(t, s, "user")
	svc2 := mustCreateService(t, s, "order")
	s.CreateRoute(model.Route{Path: "/orders", Method: "GET", ServiceID: svc2.ID})
	app, _ := s.CreateApp(model.App{Name: "app1"})
	s.CreateAPIKey(app.ID, "s", zero())
	s.CreateRateLimitRule(model.RateLimitRule{Name: "rl", Limit: 10, Window: 60})

	snap := s.ExportConfig()
	if len(snap.Services) != 2 || len(snap.Routes) != 1 || len(snap.Apps) != 1 || len(snap.APIKeys) != 1 || len(snap.RateLimitRules) != 1 {
		t.Fatalf("snapshot = %+v", snap)
	}
}

func TestImportServices(t *testing.T) {
	s := newTestService()
	created, errs := s.ImportServices([]model.Service{
		{Name: "svc1", BaseURL: "http://a"},
		{Name: "svc2", BaseURL: "http://b"},
		{Name: "svc1", BaseURL: "http://c"}, // 重复
		{Name: "", BaseURL: "http://d"},     // 非法
	})
	if created != 2 || len(errs) != 2 {
		t.Fatalf("created=%d errs=%v", created, errs)
	}
}

func TestImportRateLimitRules(t *testing.T) {
	s := newTestService()
	created, errs := s.ImportRateLimitRules([]model.RateLimitRule{
		{Name: "r1", Limit: 10, Window: 60},
		{Name: "r2", Limit: 20, Window: 60},
		{Name: "r1", Limit: 30, Window: 60}, // 重复
	})
	if created != 2 || len(errs) != 1 {
		t.Fatalf("created=%d errs=%v", created, errs)
	}
}

func TestProbeServiceUnreachable(t *testing.T) {
	s := newTestService()
	// 探测一个不可达地址
	v, _ := s.CreateService(model.Service{Name: "bad", BaseURL: "http://127.0.0.1:1", Timeout: 1})
	hc, err := s.ProbeService(v.ID)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if hc.Status != model.HealthUnhealthy && hc.FailureCount < 1 {
		t.Fatalf("unreachable service should have failures: %+v", hc)
	}
}

func zero() (t time.Time) { return }
