package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminAuthAllowed(t *testing.T) {
	h := AdminAuth("secret-token")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/api/services", nil)
	req.Header.Set("X-Admin-Token", "secret-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rr.Code)
	}
}

func TestAdminAuthDenied(t *testing.T) {
	h := AdminAuth("secret-token")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	cases := []struct {
		name  string
		token string
	}{
		{"missing", ""},
		{"wrong", "bad-token"},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", "/api/services", nil)
		if c.token != "" {
			req.Header.Set("X-Admin-Token", c.token)
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s: code = %d, want 401", c.name, rr.Code)
		}
	}
}

func TestRateLimitAllowsWithinBurst(t *testing.T) {
	h := RateLimit(100, 3)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("GET", "/x", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d code = %d, want 200", i, rr.Code)
		}
	}
}

func TestRateLimitBlocksExcess(t *testing.T) {
	h := RateLimit(0, 2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	ok := 0
	limited := 0
	for i := 0; i < 5; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("GET", "/x", nil))
		if rr.Code == http.StatusOK {
			ok++
		} else if rr.Code == http.StatusTooManyRequests {
			limited++
		}
	}
	if ok != 2 || limited != 3 {
		t.Fatalf("ok=%d limited=%d, want 2/3", ok, limited)
	}
}
