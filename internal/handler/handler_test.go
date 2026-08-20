package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apigateway/internal/config"
	"apigateway/internal/service"
	"apigateway/internal/store"
	"apigateway/pkg/logger"
)

const testAdminToken = "test-token"

func newTestServer() *Server {
	cfg := &config.Config{MaxPageSize: 100, AdminToken: testAdminToken}
	log := logger.NewLevel(logger.LevelError)
	svc := service.New(store.NewMemoryStore(), log, cfg)
	return NewServer(svc, log, cfg)
}

// doAdmin 发起带管理员鉴权的管理 API 请求。
func doAdmin(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Admin-Token", testAdminToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func decode(t *testing.T, rr *httptest.ResponseRecorder, dst interface{}) {
	t.Helper()
	if err := json.Unmarshal(rr.Body.Bytes(), dst); err != nil {
		t.Fatalf("decode: %v, body=%s", err, rr.Body.String())
	}
}

func TestAdminAuthRequired(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	req := httptest.NewRequest("GET", "/api/services", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rr.Code)
	}
}

func TestServiceAPI(t *testing.T) {
	s := newTestServer()
	h := s.Routes()

	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"user","base_url":"http://localhost:9000"}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create code = %d", rr.Code)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &created)

	rr = doAdmin(t, h, "GET", "/api/services/"+created.Data.ID, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("get code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "GET", "/api/services?keyword=user", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "PUT", "/api/services/"+created.Data.ID, `{"name":"user2","base_url":"http://localhost:9001"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("update code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "DELETE", "/api/services/"+created.Data.ID, "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "GET", "/api/services/"+created.Data.ID, "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("get deleted code = %d, want 404", rr.Code)
	}
}

func TestRouteAPI(t *testing.T) {
	s := newTestServer()
	h := s.Routes()

	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"user","base_url":"http://localhost:9000"}`)
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)

	rr = doAdmin(t, h, "POST", "/api/routes", `{"path":"/users","method":"GET","service_id":"`+svc.Data.ID+`","requires_auth":true}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create route code = %d", rr.Code)
	}
	var route struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &route)

	rr = doAdmin(t, h, "GET", "/api/routes?method=GET", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list routes code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "DELETE", "/api/routes/"+route.Data.ID, "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete route code = %d", rr.Code)
	}
}

func TestAppAndAPIKeyAPI(t *testing.T) {
	s := newTestServer()
	h := s.Routes()

	rr := doAdmin(t, h, "POST", "/api/apps", `{"name":"myapp"}`)
	var app struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &app)

	rr = doAdmin(t, h, "POST", "/api/api-keys", `{"app_id":"`+app.Data.ID+`"}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create key code = %d", rr.Code)
	}
	var key struct {
		Data struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		} `json:"data"`
	}
	decode(t, rr, &key)
	if key.Data.Key == "" {
		t.Fatal("key should be auto-generated")
	}

	rr = doAdmin(t, h, "GET", "/api/api-keys?app_id="+app.Data.ID, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list keys code = %d", rr.Code)
	}
	// 禁用密钥
	rr = doAdmin(t, h, "PATCH", "/api/api-keys/"+key.Data.ID+"/status", `{"status":"disabled"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("disable key code = %d", rr.Code)
	}
}

func TestRateLimitRuleAPI(t *testing.T) {
	s := newTestServer()
	h := s.Routes()

	rr := doAdmin(t, h, "POST", "/api/rate-limit-rules", `{"name":"rl","limit":100,"window":60}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create code = %d", rr.Code)
	}
	var rule struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &rule)

	rr = doAdmin(t, h, "GET", "/api/rate-limit-rules", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "DELETE", "/api/rate-limit-rules/"+rule.Data.ID, "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete code = %d", rr.Code)
	}
}

func TestStatsAndExportAPI(t *testing.T) {
	s := newTestServer()
	h := s.Routes()

	rr := doAdmin(t, h, "GET", "/api/stats", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("stats code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "GET", "/api/export", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("export code = %d", rr.Code)
	}
}

func TestImportAPI(t *testing.T) {
	s := newTestServer()
	h := s.Routes()

	rr := doAdmin(t, h, "POST", "/api/import/services", `{"services":[{"name":"s1","base_url":"http://a"},{"name":"s2","base_url":"http://b"}]}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("import code = %d", rr.Code)
	}
	var resp struct {
		Data struct {
			Created int `json:"created"`
		} `json:"data"`
	}
	decode(t, rr, &resp)
	if resp.Data.Created != 2 {
		t.Fatalf("created = %d, want 2", resp.Data.Created)
	}
}

func TestHealthAPI(t *testing.T) {
	s := newTestServer()
	h := s.Routes()

	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"bad","base_url":"http://127.0.0.1:1","timeout":1}`)
	var svc struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decode(t, rr, &svc)

	rr = doAdmin(t, h, "GET", "/api/health", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list health code = %d", rr.Code)
	}
	rr = doAdmin(t, h, "POST", "/api/health/"+svc.Data.ID+"/probe", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("probe code = %d", rr.Code)
	}
}

func TestNotFoundHandling(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "GET", "/api/services/nope", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rr.Code)
	}
}

func TestBadRequestHandling(t *testing.T) {
	s := newTestServer()
	h := s.Routes()
	rr := doAdmin(t, h, "POST", "/api/services", `{"name":"","base_url":"http://x"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rr.Code)
	}
}
