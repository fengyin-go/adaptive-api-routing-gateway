package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFullGatewayLifecycle 完整闭环：配置服务/路由/应用/密钥/限流 -> 转发 ->
// 鉴权 -> 统计 -> 导出 -> 健康检查。
func TestFullGatewayLifecycle(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello %s", r.URL.Path)
	})
	s := newTestServer()
	h := s.Routes()

	// 1. 创建上游服务
	rr := doAdmin(t, h, "POST", "/api/services", fmt.Sprintf(`{"name":"api","base_url":%q}`, upstream.URL))
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)

	// 2. 创建限流规则
	rr = doAdmin(t, h, "POST", "/api/rate-limit-rules", `{"name":"rl","limit":100,"window":60}`)
	var rule struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &rule)

	// 3. 创建路由（需鉴权 + 限流）
	rr = doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(`{"path":"/api-data","method":"GET","service_id":%q,"requires_auth":true,"rate_limit_rule_id":%q}`, svc.Data.ID, rule.Data.ID))
	var route struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &route)

	// 4. 创建应用与密钥
	rr = doAdmin(t, h, "POST", "/api/apps", `{"name":"client"}`)
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

	// 5. 鉴权转发成功
	req := httptest.NewRequest("GET", "/api-data/users", nil)
	req.Header.Set("X-API-Key", key.Data.Key)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || rr.Body.String() != "hello /api-data/users" {
		t.Fatalf("forward code=%d body=%s", rr.Code, rr.Body.String())
	}

	// 6. 无鉴权失败
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/api-data/users", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unauth code = %d, want 401", rr.Code)
	}

	// 7. 统计
	rr = doAdmin(t, h, "GET", "/api/stats", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("stats code = %d", rr.Code)
	}
	var stats struct {
		Data struct {
			Services int `json:"services"`
			Routes   int `json:"routes"`
			Apps     int `json:"apps"`
			APIKeys  int `json:"api_keys"`
			Requests struct {
				Total int `json:"total"`
			} `json:"requests"`
		} `json:"data"`
	}
	decode(t, rr, &stats)
	if stats.Data.Services != 1 || stats.Data.Routes != 1 || stats.Data.Apps != 1 || stats.Data.APIKeys != 1 {
		t.Fatalf("stats counts = %+v", stats.Data)
	}
	if stats.Data.Requests.Total < 2 {
		t.Fatalf("requests total = %d, want >= 2", stats.Data.Requests.Total)
	}

	// 8. 导出
	rr = doAdmin(t, h, "GET", "/api/export", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("export code = %d", rr.Code)
	}

	// 9. 健康检查（探测成功）
	rr = doAdmin(t, h, "POST", "/api/health/"+svc.Data.ID+"/probe", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("probe code = %d", rr.Code)
	}
}

func TestGatewayForwardingWithQuery(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "query=%s", r.URL.RawQuery)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/q", "GET", false, false)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/q/data?a=1&b=2", nil))
	if rr.Code != http.StatusOK || rr.Body.String() != "query=a=1&b=2" {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGatewayLogsFilterByStatusCode(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/nf", "GET", false, false)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/nf/x", nil))

	rr := doAdmin(t, h, "GET", "/api/logs?status_code=404", "")
	var list struct {
		Data struct {
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
	}
	decode(t, rr, &list)
	if list.Data.Pagination.Total < 1 {
		t.Fatalf("404 logs total = %d, want >= 1", list.Data.Pagination.Total)
	}
}

func TestUpdateRouteEndpoint(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"svc","base_url":"http://x"}`)
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)
	rr = doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(`{"path":"/a","method":"GET","service_id":%q}`, svc.Data.ID))
	var route struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &route)

	rr = doAdmin(t, h, "PUT", "/api/routes/"+route.Data.ID, fmt.Sprintf(`{"path":"/b","method":"POST","service_id":%q,"strip_prefix":true}`, svc.Data.ID))
	if rr.Code != http.StatusOK {
		t.Fatalf("update route code = %d", rr.Code)
	}
	var updated struct {
		Data struct {
			Path   string `json:"path"`
			Method string `json:"method"`
		} `json:"data"`
	}
	decode(t, rr, &updated)
	if updated.Data.Path != "/b" || updated.Data.Method != "POST" {
		t.Fatalf("updated = %+v", updated.Data)
	}
}
