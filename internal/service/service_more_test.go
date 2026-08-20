package service

import (
	"testing"

	"apigateway/internal/model"
)

func TestListAppsPagination(t *testing.T) {
	s := newTestService()
	for i := 0; i < 6; i++ {
		s.CreateApp(model.App{Name: "app" + string(rune('a'+i))})
	}
	items, total, _ := s.ListApps(model.AppFilter{}, 1, 3)
	if total != 6 || len(items) != 3 {
		t.Fatalf("page1 total=%d len=%d", total, len(items))
	}
	items, _, _ = s.ListApps(model.AppFilter{}, 2, 3)
	if len(items) != 3 {
		t.Fatalf("page2 len=%d", len(items))
	}
	items, _, _ = s.ListApps(model.AppFilter{}, 3, 3)
	if len(items) != 0 {
		t.Fatalf("page3 len=%d", len(items))
	}
}

func TestListAppsFiltered(t *testing.T) {
	s := newTestService()
	a, _ := s.CreateApp(model.App{Name: "active-app"})
	s.CreateApp(model.App{Name: "disabled-app", Status: model.AppDisabled})
	_ = a

	items, total, _ := s.ListApps(model.AppFilter{Status: model.AppActive}, 1, 20)
	if total != 1 || items[0].Name != "active-app" {
		t.Fatalf("active total=%d", total)
	}
	items, total, _ = s.ListApps(model.AppFilter{Status: model.AppDisabled}, 1, 20)
	if total != 1 || items[0].Name != "disabled-app" {
		t.Fatalf("disabled total=%d", total)
	}
}

func TestListServicesPaginationEdge(t *testing.T) {
	s := newTestService()
	// 空列表
	items, total, _ := s.ListServices(model.ServiceFilter{}, 1, 20)
	if total != 0 || len(items) != 0 {
		t.Fatalf("empty total=%d len=%d", total, len(items))
	}
}

func TestListRequestLogsPagination(t *testing.T) {
	s := newTestService()
	for i := 0; i < 5; i++ {
		s.RecordRequest(model.RequestLog{RouteID: "r1", Method: "GET", Path: "/x", StatusCode: 200})
	}
	items, total, _ := s.ListRequestLogs(model.RequestLogFilter{}, 1, 2)
	if total != 5 || len(items) != 2 {
		t.Fatalf("page1 total=%d len=%d", total, len(items))
	}
	items, _, _ = s.ListRequestLogs(model.RequestLogFilter{}, 3, 2)
	if len(items) != 1 {
		t.Fatalf("page3 len=%d", len(items))
	}
}

func TestDeleteServiceAlsoHasHealth(t *testing.T) {
	s := newTestService()
	v := mustCreateService(t, s, "api")
	// 删除服务后健康检查仍存在（无级联删除，这是预期行为）
	s.DeleteService(v.ID)
	hc, err := s.GetHealthCheck(v.ID)
	if err != nil || hc == nil {
		t.Fatalf("health check should still exist: %v", err)
	}
}

func TestUpdateAPIKeyStatusTransitions(t *testing.T) {
	s := newTestService()
	app, _ := s.CreateApp(model.App{Name: "app"})
	key, _ := s.CreateAPIKey(app.ID, "s", zero())

	// active -> disabled
	k, err := s.UpdateAPIKeyStatus(key.ID, model.APIKeyDisabled)
	if err != nil || k.Status != model.APIKeyDisabled {
		t.Fatalf("disable: %v", err)
	}
	// disabled -> active
	k, err = s.UpdateAPIKeyStatus(key.ID, model.APIKeyActive)
	if err != nil || k.Status != model.APIKeyActive {
		t.Fatalf("enable: %v", err)
	}
}

func TestCreateRouteDefaultMethod(t *testing.T) {
	s := newTestService()
	svc := mustCreateService(t, s, "api")
	r, err := s.CreateRoute(model.Route{Path: "/x", ServiceID: svc.ID})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if r.Method != "GET" {
		t.Fatalf("default method = %s, want GET", r.Method)
	}
}

func TestMatchRouteNoRoutes(t *testing.T) {
	s := newTestService()
	if _, ok := s.MatchRoute("GET", "/x"); ok {
		t.Fatal("should not match with no routes")
	}
}

func TestGetGatewayStatsEmpty(t *testing.T) {
	s := newTestService()
	stats, err := s.GetGatewayStats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Services != 0 || stats.Routes != 0 || stats.Apps != 0 || stats.APIKeys != 0 {
		t.Fatalf("empty stats = %+v", stats)
	}
	if stats.Requests.Total != 0 {
		t.Fatalf("empty requests = %+v", stats.Requests)
	}
}

func TestRecordRequestSetsIDAndTime(t *testing.T) {
	s := newTestService()
	l := s.RecordRequest(model.RequestLog{RouteID: "r1", Method: "GET", Path: "/x", StatusCode: 200})
	if l.ID == "" {
		t.Fatal("ID should be auto-generated")
	}
	if l.Timestamp.IsZero() {
		t.Fatal("timestamp should be set")
	}
}
