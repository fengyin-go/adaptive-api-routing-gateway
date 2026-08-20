package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGatewayPutAndDeleteForwarding(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", r.Method)
	})
	s := newTestServer()
	h := s.Routes()

	// 创建一个服务，然后注册 PUT 和 DELETE 两条路由
	rr := doAdmin(t, h, "POST", "/api/services", fmt.Sprintf(`{"name":"res-svc","base_url":%q}`, upstream.URL))
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)
	doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(`{"path":"/res","method":"PUT","service_id":%q}`, svc.Data.ID))
	doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(`{"path":"/res","method":"DELETE","service_id":%q}`, svc.Data.ID))

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("PUT", "/res/1", strings.NewReader("data")))
	if rr.Code != http.StatusOK || rr.Body.String() != "PUT" {
		t.Fatalf("PUT code=%d body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("DELETE", "/res/1", nil))
	if rr.Code != http.StatusOK || rr.Body.String() != "DELETE" {
		t.Fatalf("DELETE code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGatewayHeaderForwarding(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "auth=%s", r.Header.Get("Authorization"))
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/hdr", "GET", false, false)

	req := httptest.NewRequest("GET", "/hdr/x", nil)
	req.Header.Set("Authorization", "Bearer token123")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !strings.Contains(rr.Body.String(), "Bearer token123") {
		t.Fatalf("header not forwarded: %s", rr.Body.String())
	}
}

func TestGatewayClientIPLogging(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/ip", "GET", false, false)

	req := httptest.NewRequest("GET", "/ip/x", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.5")
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
	if len(list.Data.Items) < 1 || list.Data.Items[0].ClientIP != "203.0.113.5" {
		t.Fatalf("client ip = %+v", list.Data.Items)
	}
}

func TestGatewayLatencyRecorded(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/latency", "GET", false, false)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/latency/x", nil))

	rr := doAdmin(t, h, "GET", "/api/logs", "")
	var list struct {
		Data struct {
			Items []struct {
				LatencyMs int64 `json:"latency_ms"`
			} `json:"items"`
		} `json:"data"`
	}
	decode(t, rr, &list)
	if len(list.Data.Items) < 1 || list.Data.Items[0].LatencyMs < 0 {
		t.Fatalf("latency = %+v", list.Data.Items)
	}
}

func TestUpdateRateLimitRuleEndpoint(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/rate-limit-rules", `{"name":"rl","limit":10,"window":60}`)
	var rule struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &rule)

	rr = doAdmin(t, h, "PUT", "/api/rate-limit-rules/"+rule.Data.ID, `{"name":"rl","limit":20,"window":30}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("update code = %d", rr.Code)
	}
	var updated struct {
		Data struct {
			Limit  int `json:"limit"`
			Window int `json:"window"`
		} `json:"data"`
	}
	decode(t, rr, &updated)
	if updated.Data.Limit != 20 || updated.Data.Window != 30 {
		t.Fatalf("updated = %+v", updated.Data)
	}
}

func TestUpdateAppEndpoint(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/apps", `{"name":"app"}`)
	var app struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &app)

	rr = doAdmin(t, h, "PUT", "/api/apps/"+app.Data.ID, `{"name":"app2","status":"disabled"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("update code = %d", rr.Code)
	}
	var updated struct {
		Data struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"data"`
	}
	decode(t, rr, &updated)
	if updated.Data.Name != "app2" || updated.Data.Status != "disabled" {
		t.Fatalf("updated = %+v", updated.Data)
	}
}

func TestHealthProbeUnreachable(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"bad","base_url":"http://127.0.0.1:1","timeout":1}`)
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)

	rr = doAdmin(t, h, "POST", "/api/health/"+svc.Data.ID+"/probe", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("probe code = %d", rr.Code)
	}
	var hc struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	decode(t, rr, &hc)
	if hc.Data.Status == "" {
		t.Fatal("health status should be set")
	}
}

func TestHealthGetMissing(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "GET", "/api/health/nope", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rr.Code)
	}
}
