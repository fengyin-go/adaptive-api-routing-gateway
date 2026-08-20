package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminAuthEmptyTokenConfig(t *testing.T) {
	// adminToken 为空时，任何请求都应被拒绝（除空 token 匹配外）
	h := AdminAuth("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("X-Admin-Token", "anything")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d, want 401", rr.Code)
	}
}

func TestRateLimitBurstOne(t *testing.T) {
	h := RateLimit(100, 1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/x", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("first code = %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/x", nil))
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("second code = %d, want 429", rr.Code)
	}
}

func TestRateLimitResponseBody(t *testing.T) {
	h := RateLimit(0, 1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil)) // 耗尽
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/x", nil))
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("code = %d", rr.Code)
	}
	if rr.Body.Len() == 0 {
		t.Fatal("429 should have a body")
	}
}

func TestAdminAuthWrongHeaderCase(t *testing.T) {
	// 请求头大小写不敏感
	h := AdminAuth("tok")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("x-admin-token", "tok")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rr.Code)
	}
}
