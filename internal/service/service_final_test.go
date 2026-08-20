package service

import (
	"testing"

	"apigateway/internal/model"
)

func TestGetGatewayStatsTopRoutesCapped(t *testing.T) {
	s := newTestService()
	svc := mustCreateService(t, s, "api")
	// 创建 12 条路由，各产生请求，验证 TopRoutes 最多返回 10 条
	var routes []*model.Route
	for i := 0; i < 12; i++ {
		r, _ := s.CreateRoute(model.Route{Path: "/r" + string(rune('a'+i)), Method: "GET", ServiceID: svc.ID})
		routes = append(routes, r)
	}
	for i, r := range routes {
		for j := 0; j < i+1; j++ {
			s.RecordRequest(model.RequestLog{RouteID: r.ID, Method: "GET", Path: "/r", StatusCode: 200})
		}
	}
	stats, err := s.GetGatewayStats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if len(stats.TopRoutes) != 10 {
		t.Fatalf("top routes = %d, want 10 (capped)", len(stats.TopRoutes))
	}
	// 请求量最多的路由排最前
	if stats.TopRoutes[0].Requests != 12 {
		t.Fatalf("top request = %d, want 12", stats.TopRoutes[0].Requests)
	}
}

func TestRecordRequestLatencyAggregation(t *testing.T) {
	s := newTestService()
	svc := mustCreateService(t, s, "api")
	r, _ := s.CreateRoute(model.Route{Path: "/lat", Method: "GET", ServiceID: svc.ID})

	s.RecordRequest(model.RequestLog{RouteID: r.ID, Method: "GET", Path: "/lat", StatusCode: 200, LatencyMs: 10})
	s.RecordRequest(model.RequestLog{RouteID: r.ID, Method: "GET", Path: "/lat", StatusCode: 200, LatencyMs: 20})

	stats, _ := s.GetGatewayStats()
	if len(stats.TopRoutes) != 1 {
		t.Fatalf("top routes = %d", len(stats.TopRoutes))
	}
	if stats.TopRoutes[0].AvgLatency != 15 {
		t.Fatalf("avg latency = %d, want 15", stats.TopRoutes[0].AvgLatency)
	}
}

func TestAuthenticateWhitespaceKey(t *testing.T) {
	s := newTestService()
	// 空 key 字符串
	if _, err := s.Authenticate(""); err == nil {
		t.Fatal("expect error for empty key")
	}
	if _, err := s.Authenticate("   "); err == nil {
		t.Fatal("expect error for whitespace key")
	}
}

func TestCreateAPIKeyGeneratedKeyFormat(t *testing.T) {
	s := newTestService()
	app, _ := s.CreateApp(model.App{Name: "app"})
	k, err := s.CreateAPIKey(app.ID, "secret", zero())
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	if len(k.Key) == 0 {
		t.Fatal("key should be non-empty")
	}
	if k.Secret != "secret" {
		t.Fatalf("secret = %s", k.Secret)
	}
}
