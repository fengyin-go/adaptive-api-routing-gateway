package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateServiceEndpoint(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"svc","base_url":"http://x"}`)
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)

	rr = doAdmin(t, h, "PUT", "/api/services/"+svc.Data.ID, `{"name":"svc2","base_url":"http://y","timeout":20,"retries":3}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("update code = %d", rr.Code)
	}
	var updated struct {
		Data struct {
			Name    string `json:"name"`
			Timeout int    `json:"timeout"`
			Retries int    `json:"retries"`
		} `json:"data"`
	}
	decode(t, rr, &updated)
	if updated.Data.Name != "svc2" || updated.Data.Timeout != 20 || updated.Data.Retries != 3 {
		t.Fatalf("updated = %+v", updated.Data)
	}
}

func TestDeleteAppEndpointTwice(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/apps", `{"name":"app"}`)
	var app struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &app)

	doAdmin(t, h, "DELETE", "/api/apps/"+app.Data.ID, "")
	rr = doAdmin(t, h, "DELETE", "/api/apps/"+app.Data.ID, "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("second delete code = %d, want 404", rr.Code)
	}
}

func TestDeleteRouteEndpointTwice(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"svc","base_url":"http://x"}`)
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)
	rr = doAdmin(t, h, "POST", "/api/routes", fmt.Sprintf(`{"path":"/x","method":"GET","service_id":%q}`, svc.Data.ID))
	var route struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &route)

	doAdmin(t, h, "DELETE", "/api/routes/"+route.Data.ID, "")
	rr = doAdmin(t, h, "DELETE", "/api/routes/"+route.Data.ID, "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("second delete code = %d, want 404", rr.Code)
	}
}

func TestDeleteRateLimitRuleTwice(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/rate-limit-rules", `{"name":"rl","limit":10,"window":60}`)
	var rule struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &rule)

	doAdmin(t, h, "DELETE", "/api/rate-limit-rules/"+rule.Data.ID, "")
	rr = doAdmin(t, h, "DELETE", "/api/rate-limit-rules/"+rule.Data.ID, "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("second delete code = %d, want 404", rr.Code)
	}
}

func TestGetLogsByID(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/gl", "GET", false, false)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/gl/x", nil))

	rr := doAdmin(t, h, "GET", "/api/logs", "")
	var list struct {
		Data struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		} `json:"data"`
	}
	decode(t, rr, &list)
	if len(list.Data.Items) < 1 {
		t.Fatal("should have logs")
	}
	rr = doAdmin(t, h, "GET", "/api/logs/"+list.Data.Items[0].ID, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("get log by id code = %d", rr.Code)
	}
}

func TestGetLogMissing(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "GET", "/api/logs/nope", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rr.Code)
	}
}

func TestGatewayRouteNotFoundAfterDelete(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestServer()
	h := s.Routes()
	svcID, routeID := setupGateway(t, h, upstream.URL, "/tmproute", "GET", false, false)
	_ = svcID

	// 删除路由后转发应 404
	doAdmin(t, h, "DELETE", "/api/routes/"+routeID, "")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/tmproute/x", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rr.Code)
	}
}

func TestStatsAfterTraffic(t *testing.T) {
	upstream := startUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newTestServer()
	h := s.Routes()
	_, _ = setupGateway(t, h, upstream.URL, "/st", "GET", false, false)

	for i := 0; i < 3; i++ {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/st/x", nil))
	}

	rr := doAdmin(t, h, "GET", "/api/stats", "")
	var stats struct {
		Data struct {
			Requests struct {
				Total   int `json:"total"`
				Success int `json:"success"`
			} `json:"requests"`
			TopRoutes []struct {
				Requests int `json:"requests"`
			} `json:"top_routes"`
		} `json:"data"`
	}
	decode(t, rr, &stats)
	if stats.Data.Requests.Total != 3 || stats.Data.Requests.Success != 3 {
		t.Fatalf("requests = %+v", stats.Data.Requests)
	}
	if len(stats.Data.TopRoutes) != 1 || stats.Data.TopRoutes[0].Requests != 3 {
		t.Fatalf("top routes = %+v", stats.Data.TopRoutes)
	}
}

func TestExportContainsAllEntities(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"svc","base_url":"http://x"}`)
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)
	doAdmin(t, h, "POST", "/api/apps", `{"name":"app"}`)

	rr = doAdmin(t, h, "GET", "/api/export", "")
	var snap struct {
		Data struct {
			Services []interface{} `json:"services"`
			Apps     []interface{} `json:"apps"`
		} `json:"data"`
	}
	decode(t, rr, &snap)
	if len(snap.Data.Services) != 1 || len(snap.Data.Apps) != 1 {
		t.Fatalf("export = %+v", snap.Data)
	}
}
