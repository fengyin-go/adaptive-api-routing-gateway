package model

import (
	"errors"
	"testing"
)

func TestServiceValidateTableDriven(t *testing.T) {
	cases := []struct {
		name    string
		svc     Service
		wantErr bool
	}{
		{"valid", Service{Name: "s", BaseURL: "http://x"}, false},
		{"valid with opts", Service{Name: "s", BaseURL: "http://x", Timeout: 10, Retries: 3}, false},
		{"empty name", Service{Name: "", BaseURL: "http://x"}, true},
		{"empty baseurl", Service{Name: "s", BaseURL: ""}, true},
		{"timeout over limit", Service{Name: "s", BaseURL: "http://x", Timeout: 301}, true},
		{"negative retries", Service{Name: "s", BaseURL: "http://x", Retries: -1}, true},
		{"retries over limit", Service{Name: "s", BaseURL: "http://x", Retries: 11}, true},
		{"bad status", Service{Name: "s", BaseURL: "http://x", Status: "weird"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.svc.Validate()
			if (err != nil) != c.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, c.wantErr)
			}
		})
	}
}

func TestRouteValidateTableDriven(t *testing.T) {
	cases := []struct {
		name    string
		route   Route
		wantErr bool
	}{
		{"valid", Route{Path: "/x", Method: "GET", ServiceID: "s"}, false},
		{"empty path", Route{Path: "", Method: "GET", ServiceID: "s"}, true},
		{"path no slash", Route{Path: "x", Method: "GET", ServiceID: "s"}, true},
		{"bad method", Route{Path: "/x", Method: "OPTIONS", ServiceID: "s"}, true},
		{"empty service", Route{Path: "/x", Method: "GET", ServiceID: ""}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.route.Validate()
			if (err != nil) != c.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, c.wantErr)
			}
		})
	}
}

func TestRateLimitRuleValidateTableDriven(t *testing.T) {
	cases := []struct {
		name    string
		rule    RateLimitRule
		wantErr bool
	}{
		{"valid", RateLimitRule{Name: "r", Limit: 10, Window: 60}, false},
		{"empty name", RateLimitRule{Name: "", Limit: 10}, true},
		{"zero limit", RateLimitRule{Name: "r", Limit: 0}, true},
		{"negative limit", RateLimitRule{Name: "r", Limit: -5}, true},
		{"limit too large", RateLimitRule{Name: "r", Limit: 2000000}, true},
		{"window too large", RateLimitRule{Name: "r", Limit: 10, Window: 100000}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.rule.Validate()
			if (err != nil) != c.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, c.wantErr)
			}
		})
	}
}

func TestAPIKeyValidateTableDriven(t *testing.T) {
	cases := []struct {
		name    string
		key     APIKey
		wantErr bool
	}{
		{"valid", APIKey{AppID: "a", Key: "k"}, false},
		{"empty app", APIKey{AppID: "", Key: "k"}, true},
		{"empty key", APIKey{AppID: "a", Key: ""}, true},
		{"bad status", APIKey{AppID: "a", Key: "k", Status: "bad"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.key.Validate()
			if (err != nil) != c.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, c.wantErr)
			}
		})
	}
}

func TestAppValidateTableDriven(t *testing.T) {
	cases := []struct {
		name    string
		app     App
		wantErr bool
	}{
		{"valid", App{Name: "a"}, false},
		{"empty name", App{Name: ""}, true},
		{"bad status", App{Name: "a", Status: "bad"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.app.Validate()
			if (err != nil) != c.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, c.wantErr)
			}
		})
	}
}

func TestValidationErrorFormatting(t *testing.T) {
	e := &ValidationError{Field: "name", Message: "不能为空"}
	if e.Error() != "name: 不能为空" {
		t.Fatalf("Error() = %q", e.Error())
	}
	e2 := &ValidationError{Message: "通用错误"}
	if e2.Error() != "通用错误" {
		t.Fatalf("Error() = %q", e2.Error())
	}
	if !IsValidationError(e) {
		t.Fatal("IsValidationError should be true")
	}
	if IsValidationError(errors.New("other")) {
		t.Fatal("IsValidationError should be false for generic error")
	}
}
