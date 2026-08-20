package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// startUpstream 启动一个真实的上游 HTTP 服务。
func startUpstream(t *testing.T, hf http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(hf)
	t.Cleanup(srv.Close)
	return srv
}

// setupGateway 配置一个上游服务与一条路由，返回路由 ID。
func setupGateway(t *testing.T, h http.Handler, baseURL, path, method string, strip bool, auth bool) (serviceID, routeID string) {
	t.Helper()
	rr := doAdmin(t, h, "POST", "/api/services", fmt.Sprintf(`{"name":"upstream","base_url":%q}`, baseURL))
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)
	serviceID = svc.Data.ID

	rr = doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(
		`{"path":%q,"method":%q,"service_id":%q,"strip_prefix":%t,"requires_auth":%t}`,
		path, method, serviceID, strip, auth))
	var route struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &route)
	routeID = route.Data.ID
	return serviceID, routeID
}

func TestGatewayForwarding(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s", r.URL.Path)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/users", "GET", false, false)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/123", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200, body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "/users/123") {
		t.Fatalf("upstream should receive original path, got %q", rr.Body.String())
	}
}

func TestGatewayStripPrefix(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s", r.URL.Path)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/users", "GET", true, false)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users/123", nil)
	h.ServeHTTP(rr, req)
	if !strings.Contains(rr.Body.String(), "path=/123") {
		t.Fatalf("strip prefix should yield /123, got %q", rr.Body.String())
	}
}

func TestGatewayRouteNotFound(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/nonexistent", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rr.Code)
	}
}

func TestGatewayAuthRequired(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/secure", "GET", false, true)

	// 无 key -> 401
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/secure/data", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("no key code = %d, want 401", rr.Code)
	}
}

func TestGatewayAuthSuccess(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "authorized")
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/secure", "GET", false, true)

	// 创建 app + key
	rr := doAdmin(t, h, "POST", "/api/apps", `{"name":"app"}`)
	var app struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &app)
	rr = doAdmin(t, h, "POST", "/api/api-keys", `{"app_id":"`+app.Data.ID+`"}`)
	var key struct {
		Data struct {
			Key string `json:"key"`
		} `json:"data"`
	}
	decode(t, rr, &key)

	req := httptest.NewRequest("GET", "/secure/data", nil)
	req.Header.Set("X-API-Key", key.Data.Key)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("auth code = %d, want 200", rr.Code)
	}
}

func TestGatewayRateLimit(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestServer()
	h := s.Routes()

	// 创建限流规则 limit=2
	rr := doAdmin(t, h, "POST", "/api/rate-limit-rules", `{"name":"rl","limit":2,"window":60}`)
	var rule struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &rule)

	// 创建服务与路由（绑定限流规则）
	rr = doAdmin(t, h, "POST", "/api/services", `{"name":"upstream","base_url":"`+upstream.URL+`"}`)
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)
	rr = doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(`{"path":"/limited","method":"GET","service_id":%q,"rate_limit_rule_id":%q}`, svc.Data.ID, rule.Data.ID))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create route code = %d", rr.Code)
	}

	ok, limited := 0, 0
	for i := 0; i < 4; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("GET", "/limited/x", nil))
		if rr.Code == http.StatusOK {
			ok++
		} else if rr.Code == http.StatusTooManyRequests {
			limited++
		}
	}
	if ok != 2 || limited != 2 {
		t.Fatalf("ok=%d limited=%d, want 2/2", ok, limited)
	}
}

func TestGatewayUpstreamUnavailable(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	// 上游地址不可达
	_, _ = setupGateway(t, h, "http://127.0.0.1:1", "/down", "GET", false, false)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/down/x", nil))
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("code = %d, want 502", rr.Code)
	}
}

func TestGatewayLogsRecorded(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/logged", "GET", false, false)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/logged/x", nil))

	rr := doAdmin(t, h, "GET", "/api/logs", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list logs code = %d", rr.Code)
	}
	var list struct {
		Data struct {
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
	}
	decode(t, rr, &list)
	if list.Data.Pagination.Total < 1 {
		t.Fatalf("logs total = %d, want >= 1", list.Data.Pagination.Total)
	}
}
