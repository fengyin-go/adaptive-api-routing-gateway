package model

import (
	"testing"
	"time"
)

func TestServiceValidateDefaults(t *testing.T) {
	s := &Service{Name: "svc", BaseURL: "http://x", Timeout: 0, Retries: 0}
	if err := s.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if s.Timeout != 5 {
		t.Fatalf("timeout = %d, want 5", s.Timeout)
	}
	if s.Retries != 0 {
		t.Fatalf("retries = %d, want 0", s.Retries)
	}
}

func TestServiceValidateBoundary(t *testing.T) {
	// 边界值：timeout=300 合法，retries=10 合法
	s := &Service{Name: "s", BaseURL: "http://x", Timeout: 300, Retries: 10}
	if err := s.Validate(); err != nil {
		t.Fatalf("boundary should pass: %v", err)
	}
	// retries=-1 非法
	if err := (&Service{Name: "s", BaseURL: "http://x", Retries: -1}).Validate(); err == nil {
		t.Fatal("negative retries should fail")
	}
}

func TestRouteValidateDefaults(t *testing.T) {
	r := &Route{Path: "/x", Method: "", ServiceID: "s"}
	if err := r.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if r.Method != "GET" {
		t.Fatalf("default method = %s, want GET", r.Method)
	}
}

func TestRouteValidateMethodList(t *testing.T) {
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		r := &Route{Path: "/x", Method: m, ServiceID: "s"}
		if err := r.Validate(); err != nil {
			t.Fatalf("method %s should be valid: %v", m, err)
		}
	}
	if err := (&Route{Path: "/x", Method: "HEAD", ServiceID: "s"}).Validate(); err == nil {
		t.Fatal("HEAD should be unsupported")
	}
}

func TestAPIKeyNoExpiry(t *testing.T) {
	k := &APIKey{AppID: "a", Key: "k", Status: APIKeyActive}
	if err := k.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if k.IsExpired() {
		t.Fatal("no expiry should not be expired")
	}
}

func TestAPIKeyFutureExpiry(t *testing.T) {
	k := &APIKey{AppID: "a", Key: "k", ExpiresAt: time.Now().Add(24 * time.Hour)}
	if err := k.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if k.IsExpired() {
		t.Fatal("future expiry should not be expired")
	}
}

func TestRateLimitRuleRatePerSecond(t *testing.T) {
	cases := []struct {
		limit  int
		window int
		want   float64
	}{
		{60, 60, 1.0},
		{120, 60, 2.0},
		{100, 10, 10.0},
		{10, 0, 10}, // window=0 时按 limit
	}
	for _, c := range cases {
		r := &RateLimitRule{Limit: c.limit, Window: c.window}
		if got := r.RatePerSecond(); got != c.want {
			t.Fatalf("RatePerSecond(%d,%d) = %v, want %v", c.limit, c.window, got, c.want)
		}
	}
}

func TestHealthCheckRecordLatencyTracking(t *testing.T) {
	h := &HealthCheck{ServiceID: "s"}
	h.Record(true, 15)
	if h.LatencyMs != 15 {
		t.Fatalf("latency = %d, want 15", h.LatencyMs)
	}
	h.Record(false, 25)
	if h.LatencyMs != 25 {
		t.Fatalf("latency = %d, want 25", h.LatencyMs)
	}
	if h.LastChecked.IsZero() {
		t.Fatal("last checked should be set")
	}
}

func TestRequestLogFilterAllEmpty(t *testing.T) {
	f := RequestLogFilter{}
	if !f.Match(&RequestLog{StatusCode: 500}) {
		t.Fatal("empty filter should match all")
	}
}

func TestRouteFilterMethodCase(t *testing.T) {
	f := RouteFilter{Method: "GET"}
	if !f.Match(&Route{Method: "GET"}) {
		t.Fatal("should match")
	}
	if f.Match(&Route{Method: "get"}) {
		t.Fatal("filter is case-sensitive, lowercase should not match")
	}
}

func TestServiceFilterKeywordCaseInsensitive(t *testing.T) {
	f := ServiceFilter{Keyword: "USER"}
	if !f.Match(&Service{Name: "user-service", BaseURL: "http://x"}) {
		t.Fatal("keyword should be case-insensitive")
	}
}
