package model

import (
	"testing"
	"time"
)

func TestServiceValidate(t *testing.T) {
	if err := (&Service{Name: "", BaseURL: "http://x"}).Validate(); err == nil {
		t.Fatal("expect error for empty name")
	}
	if err := (&Service{Name: "s", BaseURL: ""}).Validate(); err == nil {
		t.Fatal("expect error for empty base url")
	}
	if err := (&Service{Name: "s", BaseURL: "http://x", Timeout: 999}).Validate(); err == nil {
		t.Fatal("expect error for too-large timeout")
	}
	if err := (&Service{Name: "s", BaseURL: "http://x", Retries: 11}).Validate(); err == nil {
		t.Fatal("expect error for too-many retries")
	}
	s := &Service{Name: " s ", BaseURL: " http://x "}
	if err := s.Validate(); err != nil {
		t.Fatalf("valid service: %v", err)
	}
	if s.Name != "s" || s.BaseURL != "http://x" {
		t.Fatalf("not trimmed: %q %q", s.Name, s.BaseURL)
	}
	if s.Timeout != 5 {
		t.Fatalf("default timeout = %d, want 5", s.Timeout)
	}
	if s.Status != ServiceActive {
		t.Fatalf("default status = %s", s.Status)
	}
}

func TestRouteValidate(t *testing.T) {
	if err := (&Route{Path: "noprefix", Method: "GET", ServiceID: "s"}).Validate(); err == nil {
		t.Fatal("expect error for path without slash")
	}
	if err := (&Route{Path: "/x", Method: "OPTIONS", ServiceID: "s"}).Validate(); err == nil {
		t.Fatal("expect error for unsupported method")
	}
	if err := (&Route{Path: "/x", Method: "get", ServiceID: ""}).Validate(); err == nil {
		t.Fatal("expect error for empty service id")
	}
	r := &Route{Path: "/x", Method: "post", ServiceID: "s"}
	if err := r.Validate(); err != nil {
		t.Fatalf("valid route: %v", err)
	}
	if r.Method != "POST" {
		t.Fatalf("method not uppercased: %s", r.Method)
	}
}

func TestAppValidate(t *testing.T) {
	if err := (&App{Name: ""}).Validate(); err == nil {
		t.Fatal("expect error for empty name")
	}
	if err := (&App{Name: "a", Status: "bad"}).Validate(); err == nil {
		t.Fatal("expect error for bad status")
	}
	a := &App{Name: " a "}
	if err := a.Validate(); err != nil {
		t.Fatalf("valid app: %v", err)
	}
	if a.Status != AppActive {
		t.Fatalf("default status = %s", a.Status)
	}
}

func TestAPIKeyValidateAndExpiry(t *testing.T) {
	if err := (&APIKey{AppID: "", Key: "k"}).Validate(); err == nil {
		t.Fatal("expect error for empty app id")
	}
	if err := (&APIKey{AppID: "a", Key: ""}).Validate(); err == nil {
		t.Fatal("expect error for empty key")
	}
	// 过期时间早于当前
	if err := (&APIKey{AppID: "a", Key: "k", ExpiresAt: time.Now().Add(-time.Hour)}).Validate(); err == nil {
		t.Fatal("expect error for past expiry")
	}
	k := &APIKey{AppID: "a", Key: "k", Status: APIKeyActive, ExpiresAt: time.Now().Add(time.Hour)}
	if err := k.Validate(); err != nil {
		t.Fatalf("valid key: %v", err)
	}
	if k.IsExpired() {
		t.Fatal("future key should not be expired")
	}
	expired := &APIKey{AppID: "a", Key: "k", ExpiresAt: time.Now().Add(-time.Minute)}
	if !expired.IsExpired() {
		t.Fatal("past key should be expired")
	}
	noExpiry := &APIKey{AppID: "a", Key: "k"}
	if noExpiry.IsExpired() {
		t.Fatal("key without expiry should never expire")
	}
}

func TestRateLimitRuleValidate(t *testing.T) {
	if err := (&RateLimitRule{Name: "", Limit: 10}).Validate(); err == nil {
		t.Fatal("expect error for empty name")
	}
	if err := (&RateLimitRule{Name: "r", Limit: 0}).Validate(); err == nil {
		t.Fatal("expect error for zero limit")
	}
	if err := (&RateLimitRule{Name: "r", Limit: 10, Window: 100000}).Validate(); err == nil {
		t.Fatal("expect error for too-large window")
	}
	r := &RateLimitRule{Name: "r", Limit: 10}
	if err := r.Validate(); err != nil {
		t.Fatalf("valid rule: %v", err)
	}
	if r.Window != 60 {
		t.Fatalf("default window = %d, want 60", r.Window)
	}
	if r.RatePerSecond() != 10.0/60.0 {
		t.Fatalf("rate per second = %v", r.RatePerSecond())
	}
}

func TestHealthCheckRecord(t *testing.T) {
	h := &HealthCheck{ServiceID: "s", Status: HealthHealthy}
	h.Record(true, 10)
	if h.Status != HealthHealthy || h.FailureCount != 0 || h.LatencyMs != 10 {
		t.Fatalf("healthy record = %+v", h)
	}
	// 连续失败 3 次才 unhealthy
	h.Record(false, 20)
	h.Record(false, 30)
	if h.Status != HealthHealthy || h.FailureCount != 2 {
		t.Fatalf("after 2 failures = %+v", h)
	}
	h.Record(false, 40)
	if h.Status != HealthUnhealthy || h.FailureCount != 3 {
		t.Fatalf("after 3 failures = %+v", h)
	}
	// 恢复后重置计数
	h.Record(true, 5)
	if h.Status != HealthHealthy || h.FailureCount != 0 {
		t.Fatalf("after recovery = %+v", h)
	}
}

func TestFilters(t *testing.T) {
	// ServiceFilter
	sf := ServiceFilter{Status: ServiceActive, Keyword: "user"}
	if !sf.Match(&Service{Name: "user-service", BaseURL: "http://x", Status: ServiceActive}) {
		t.Fatal("service should match")
	}
	if sf.Match(&Service{Name: "order-service", BaseURL: "http://x", Status: ServiceActive}) {
		t.Fatal("service should not match keyword")
	}
	// RouteFilter
	rf := RouteFilter{ServiceID: "s1", Method: "GET"}
	if !rf.Match(&Route{ServiceID: "s1", Method: "GET"}) {
		t.Fatal("route should match")
	}
	if rf.Match(&Route{ServiceID: "s2", Method: "GET"}) {
		t.Fatal("route should not match")
	}
	// RequestLogFilter
	lf := RequestLogFilter{StatusCode: 200}
	if !lf.Match(&RequestLog{StatusCode: 200}) {
		t.Fatal("log should match")
	}
	if lf.Match(&RequestLog{StatusCode: 404}) {
		t.Fatal("log should not match")
	}
	// AppFilter
	af := AppFilter{Status: AppActive}
	if !af.Match(&App{Status: AppActive}) {
		t.Fatal("app should match")
	}
	if af.Match(&App{Status: AppDisabled}) {
		t.Fatal("app should not match")
	}
}
