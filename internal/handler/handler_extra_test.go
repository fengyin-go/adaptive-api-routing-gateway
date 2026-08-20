package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServiceDuplicateConflict(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	doAdmin(t, h, "POST", "/api/services", `{"name":"user","base_url":"http://x"}`)
	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"user","base_url":"http://y"}`)
	if rr.Code != http.StatusConflict {
		t.Fatalf("code = %d, want 409", rr.Code)
	}
}

func TestRouteDuplicateConflict(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"user","base_url":"http://x"}`)
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)

	doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(`{"path":"/a","method":"GET","service_id":%q}`, svc.Data.ID))
	rr = doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(`{"path":"/a","method":"GET","service_id":%q}`, svc.Data.ID))
	if rr.Code != http.StatusConflict {
		t.Fatalf("code = %d, want 409", rr.Code)
	}
}

func TestAppDuplicateConflict(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	doAdmin(t, h, "POST", "/api/apps", `{"name":"app"}`)
	rr := doAdmin(t, h, "POST", "/api/apps", `{"name":"app"}`)
	if rr.Code != http.StatusConflict {
		t.Fatalf("code = %d, want 409", rr.Code)
	}
}

func TestGatewayMethodMismatch(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/users", "GET", false, false)

	// POST 不匹配 GET 路由
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("POST", "/users/1", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rr.Code)
	}
}

func TestGatewayPostForwarding(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "method=%s", r.Method)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/submit", "POST", false, false)

	req := httptest.NewRequest("POST", "/submit", strings.NewReader(`{"a":1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "method=POST") {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGatewayStripPrefixRoot(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s", r.URL.Path)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/root", "GET", true, false)

	// 精确匹配 /root，strip 后应为 /
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/root", nil))
	if !strings.Contains(rr.Body.String(), "path=/") {
		t.Fatalf("strip root should yield /, got %q", rr.Body.String())
	}
}

func TestGatewayCircuitBreakerTrips(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	// 上游不可达
	_, _ = setupGateway(t, h, "http://127.0.0.1:1", "/trip", "GET", false, false)

	// 连续 5 次失败触发熔断
	for i := 0; i < 5; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("GET", "/trip/x", nil))
		if rr.Code != http.StatusBadGateway {
			t.Fatalf("request %d code = %d, want 502", i, rr.Code)
		}
	}
	// 第 6 次被熔断器拦截 -> 503
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/trip/x", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503 (circuit open)", rr.Code)
	}
}

func TestGatewayAuthDisabledKey(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/secure", "GET", false, true)

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
			ID  string `json:"id"`
			Key string `json:"key"`
		} `json:"data"`
	}
	decode(t, rr, &key)
	// 禁用密钥
	doAdmin(t, h, "PATCH", "/api/api-keys/"+key.Data.ID+"/status", `{"status":"disabled"}`)

	req := httptest.NewRequest("GET", "/secure/x", nil)
	req.Header.Set("X-API-Key", key.Data.Key)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rr.Code)
	}
}

func TestListPaginationQuery(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	for i := 0; i < 5; i++ {
		doAdmin(t, h, "POST", "/api/services", fmt.Sprintf(`{"name":"svc%d","base_url":"http://x"}`, i))
	}
	rr := doAdmin(t, h, "GET", "/api/services?page=1&size=2", "")
	var list struct {
		Data struct {
			Items      []interface{} `json:"items"`
			Pagination struct {
				Page  int `json:"page"`
				Size  int `json:"size"`
				Total int `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
	}
	decode(t, rr, &list)
	if list.Data.Pagination.Total != 5 || len(list.Data.Items) != 2 || list.Data.Pagination.Page != 1 {
		t.Fatalf("pagination = %+v", list.Data.Pagination)
	}
}

func TestMalformedJSON(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	req := httptest.NewRequest("POST", "/api/services", strings.NewReader("{bad"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", testAdminToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rr.Code)
	}
}

func TestImportRateLimitRulesEndpoint(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/import/rate-limit-rules", `{"rules":[{"name":"r1","limit":10,"window":60}]}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("import code = %d", rr.Code)
	}
	var resp struct {
		Data struct {
			Created int `json:"created"`
		} `json:"data"`
	}
	decode(t, rr, &resp)
	if resp.Data.Created != 1 {
		t.Fatalf("created = %d, want 1", resp.Data.Created)
	}
}

func TestUpdateAndDeleteAPIKey(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
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
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &key)

	rr = doAdmin(t, h, "GET", "/api/api-keys/"+key.Data.ID, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("get key code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "DELETE", "/api/api-keys/"+key.Data.ID, "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete key code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "GET", "/api/api-keys/"+key.Data.ID, "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("get deleted key code = %d, want 404", rr.Code)
	}
}
