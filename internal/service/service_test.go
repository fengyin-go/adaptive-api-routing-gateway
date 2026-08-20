package service

import (
	"testing"
	"time"

	"apigateway/internal/config"
	"apigateway/internal/model"
	"apigateway/internal/store"
	"apigateway/pkg/logger"
)

func newTestService() *Service {
	cfg := &config.Config{MaxPageSize: 100, AdminToken: "test-token"}
	log := logger.NewLevel(logger.LevelError)
	return New(store.NewMemoryStore(), log, cfg)
}

func mustCreateService(t *testing.T, s *Service, name string) *model.Service {
	t.Helper()
	v, err := s.CreateService(model.Service{Name: name, BaseURL: "http://localhost:9000", Timeout: 3})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	return v
}

func TestCreateServiceAndHealthInit(t *testing.T) {
	s := newTestService()
	v, err := s.CreateService(model.Service{Name: "user", BaseURL: "http://x"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// 创建服务时同步初始化健康检查
	hc, err := s.GetHealthCheck(v.ID)
	if err != nil || hc.Status != model.HealthHealthy {
		t.Fatalf("health check init: %v", err)
	}
}

func TestCreateServiceDuplicate(t *testing.T) {
	s := newTestService()
	mustCreateService(t, s, "user")
	if _, err := s.CreateService(model.Service{Name: "user", BaseURL: "http://y"}); err == nil {
		t.Fatal("expect conflict for duplicate name")
	}
}

func TestUpdateAndDeleteService(t *testing.T) {
	s := newTestService()
	v := mustCreateService(t, s, "user")
	updated, err := s.UpdateService(v.ID, model.Service{Name: "user2", BaseURL: "http://y", Timeout: 10, Retries: 2})
	if err != nil || updated.Name != "user2" || updated.Timeout != 10 {
		t.Fatalf("update: %v", err)
	}
	if err := s.DeleteService(v.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetService(v.ID); err == nil {
		t.Fatal("expect not found after delete")
	}
}

func TestCreateRouteWithDeps(t *testing.T) {
	s := newTestService()
	svc := mustCreateService(t, s, "user")
	rule, _ := s.CreateRateLimitRule(model.RateLimitRule{Name: "rl", Limit: 10, Window: 60})

	r, err := s.CreateRoute(model.Route{Path: "/users", Method: "GET", ServiceID: svc.ID, RequiresAuth: true, RateLimitRuleID: rule.ID})
	if err != nil {
		t.Fatalf("create route: %v", err)
	}
	// 引用不存在的服务
	if _, err := s.CreateRoute(model.Route{Path: "/x", Method: "GET", ServiceID: "nope"}); err == nil {
		t.Fatal("expect error for missing service")
	}
	// 引用不存在的限流规则
	if _, err := s.CreateRoute(model.Route{Path: "/y", Method: "GET", ServiceID: svc.ID, RateLimitRuleID: "nope"}); err == nil {
		t.Fatal("expect error for missing rule")
	}
	_ = r
}

func TestMatchRouteLongestPrefix(t *testing.T) {
	s := newTestService()
	svc := mustCreateService(t, s, "user")
	// 注册 /users 和 /users/vip 两条路由
	base, _ := s.CreateRoute(model.Route{Path: "/users", Method: "GET", ServiceID: svc.ID})
	vip, _ := s.CreateRoute(model.Route{Path: "/users/vip", Method: "GET", ServiceID: svc.ID})

	r, ok := s.MatchRoute("GET", "/users/vip/123")
	if !ok || r.ID != vip.ID {
		t.Fatalf("should match vip route, got %v", r)
	}
	r, ok = s.MatchRoute("GET", "/users/123")
	if !ok || r.ID != base.ID {
		t.Fatalf("should match base route, got %v", r)
	}
	if _, ok := s.MatchRoute("POST", "/users/123"); ok {
		t.Fatal("should not match wrong method")
	}
	if _, ok := s.MatchRoute("GET", "/orders"); ok {
		t.Fatal("should not match unrelated path")
	}
}

func TestAuthenticate(t *testing.T) {
	s := newTestService()
	app, _ := s.CreateApp(model.App{Name: "myapp"})
	key, _ := s.CreateAPIKey(app.ID, "secret", time.Time{})

	// 正常鉴权
	got, err := s.Authenticate(key.Key)
	if err != nil || got.ID != app.ID {
		t.Fatalf("authenticate: %v", err)
	}
	// 密钥不存在
	if _, err := s.Authenticate("ak_nope"); err == nil {
		t.Fatal("expect error for unknown key")
	}
	// 密钥禁用
	s.UpdateAPIKeyStatus(key.ID, model.APIKeyDisabled)
	if _, err := s.Authenticate(key.Key); err == nil {
		t.Fatal("expect error for disabled key")
	}
}

func TestAuthenticateExpiredKey(t *testing.T) {
	st := store.NewMemoryStore()
	s := New(st, logger.NewLevel(logger.LevelError), &config.Config{MaxPageSize: 100})
	app, _ := s.CreateApp(model.App{Name: "expapp"})
	// 直接通过 store 构造一个已过期的密钥（绕过创建时的过期校验）
	k := &model.APIKey{ID: "k", AppID: app.ID, Key: "ak_expired", Status: model.APIKeyActive, ExpiresAt: time.Now().Add(-time.Hour)}
	if err := st.CreateAPIKey(k); err != nil {
		t.Fatalf("create key: %v", err)
	}
	if _, err := s.Authenticate("ak_expired"); err == nil {
		t.Fatal("expect error for expired key")
	}
}

func TestAuthenticateDisabledApp(t *testing.T) {
	s := newTestService()
	app, _ := s.CreateApp(model.App{Name: "disapp"})
	key, _ := s.CreateAPIKey(app.ID, "s", time.Time{})
	s.UpdateApp(app.ID, model.App{Name: "disapp", Status: model.AppDisabled})
	if _, err := s.Authenticate(key.Key); err == nil {
		t.Fatal("expect error for disabled app")
	}
}

func TestCreateAPIKeyWithBadApp(t *testing.T) {
	s := newTestService()
	if _, err := s.CreateAPIKey("nope", "s", time.Time{}); err == nil {
		t.Fatal("expect error for missing app")
	}
}

func TestListAPIKeysFiltered(t *testing.T) {
	s := newTestService()
	app1, _ := s.CreateApp(model.App{Name: "app1"})
	app2, _ := s.CreateApp(model.App{Name: "app2"})
	s.CreateAPIKey(app1.ID, "s1", time.Time{})
	s.CreateAPIKey(app1.ID, "s2", time.Time{})
	s.CreateAPIKey(app2.ID, "s3", time.Time{})

	items, total, _ := s.ListAPIKeys(app1.ID, 1, 20)
	if total != 2 || len(items) != 2 {
		t.Fatalf("app1 keys total=%d", total)
	}
	items, total, _ = s.ListAPIKeys("", 1, 20)
	if total != 3 {
		t.Fatalf("all keys total=%d, want 3", total)
	}
}
