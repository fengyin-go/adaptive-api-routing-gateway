package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGatewayCombinedAuthAndRateLimit(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestServer()
	h := s.Routes()

	// 创建限流规则 limit=3
	rr := doAdmin(t, h, "POST", "/api/rate-limit-rules", `{"name":"rl","limit":3,"window":60}`)
	var rule struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &rule)

	// 创建服务
	rr = doAdmin(t, h, "POST", "/api/services", fmt.Sprintf(`{"name":"secured","base_url":%q}`, upstream.URL))
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)

	// 创建需鉴权 + 限流的路由
	rr = doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(`{"path":"/secured","method":"GET","service_id":%q,"requires_auth":true,"rate_limit_rule_id":%q}`, svc.Data.ID, rule.Data.ID))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create route code = %d", rr.Code)
	}

	// 创建 app + key
	rr = doAdmin(t, h, "POST", "/api/apps", `{"name":"app"}`)
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

	// 无鉴权 -> 401
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/secured/x", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unauth code = %d", rr.Code)
	}

	// 带鉴权，前 3 次放行
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/secured/x", nil)
		req.Header.Set("X-API-Key", key.Data.Key)
		rr = httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d code = %d", i, rr.Code)
		}
	}
	// 第 4 次限流 -> 429
	req := httptest.NewRequest("GET", "/secured/x", nil)
	req.Header.Set("X-API-Key", key.Data.Key)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("limited code = %d, want 429", rr.Code)
	}
}

func TestGatewayLongestPrefixPriority(t *testing.T) {
	upstream1 := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "base")
	})
	upstream2 := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "vip")
	})
	s := newTestServer()
	h := s.Routes()

	// 两条路由指向不同上游，验证最长前缀优先
	rr := doAdmin(t, h, "POST", "/api/services", fmt.Sprintf(`{"name":"base","base_url":%q}`, upstream1.URL))
	var s1 struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &s1)
	rr = doAdmin(t, h, "POST", "/api/services", fmt.Sprintf(`{"name":"vip","base_url":%q}`, upstream2.URL))
	var s2 struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &s2)

	doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(`{"path":"/users","method":"GET","service_id":%q}`, s1.Data.ID))
	doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(`{"path":"/users/vip","method":"GET","service_id":%q}`, s2.Data.ID))

	// 命中 vip 路由
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/users/vip/1", nil))
	if rr.Body.String() != "vip" {
		t.Fatalf("should hit vip, got %q", rr.Body.String())
	}
	// 命中 base 路由
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/users/1", nil))
	if rr.Body.String() != "base" {
		t.Fatalf("should hit base, got %q", rr.Body.String())
	}
}

func TestDeleteServiceEndpointTwice(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"x","base_url":"http://x"}`)
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)

	doAdmin(t, h, "DELETE", "/api/services/"+svc.Data.ID, "")
	rr = doAdmin(t, h, "DELETE", "/api/services/"+svc.Data.ID, "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("second delete code = %d, want 404", rr.Code)
	}
}

func TestBadRequestEmptyBody(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	req := httptest.NewRequest("POST", "/api/services", nil)
	req.Header.Set("X-Admin-Token", testAdminToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rr.Code)
	}
}

func TestBadRequestWrongType(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"x","base_url":"http://x","timeout":"abc"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rr.Code)
	}
}

func TestGatewayNonJSONUpstreamError(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/err", "GET", false, false)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/err/x", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d, want 500", rr.Code)
	}
}

func TestClientIPRemoteAddr(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	// 无 X-Forwarded-For 时使用 RemoteAddr
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	_, _ = setupGateway(t, h, upstream.URL, "/ip2", "GET", false, false)

	req := httptest.NewRequest("GET", "/ip2/x", nil)
	req.RemoteAddr = "192.168.1.10:54321"
	h.ServeHTTP(httptest.NewRecorder(), req)

	rr := doAdmin(t, h, "GET", "/api/logs", "")
	var list struct {
		Data struct {
			Items []struct {
				ClientIP string `json:"client_ip"`
			} `json:"items"`
		} `json:"data"`
	}
	decode(t, rr, &list)
	if len(list.Data.Items) < 1 || list.Data.Items[0].ClientIP != "192.168.1.10" {
		t.Fatalf("client ip = %+v", list.Data.Items)
	}
}

func TestListRoutesAndRulesEmpty(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "GET", "/api/routes", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("routes code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "GET", "/api/rate-limit-rules", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("rules code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "GET", "/api/logs", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("logs code = %d", rr.Code)
	}
}

func TestGatewayStripPrefixPreservesQuery(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "path=%s query=%s", r.URL.Path, r.URL.RawQuery)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/proxy", "GET", true, false)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/proxy/users?a=1", nil))
	body := rr.Body.String()
	if !strings.Contains(body, "path=/users") || !strings.Contains(body, "query=a=1") {
		t.Fatalf("strip prefix with query: %s", body)
	}
}
