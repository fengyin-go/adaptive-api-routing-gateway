package model

import (
	"strings"
	"time"
)

// Route 路由规则，将匹配到的请求转发到指定上游服务。
type Route struct {
	ID             string    `json:"id"`
	Path           string    `json:"path"`
	Method         string    `json:"method"`
	ServiceID      string    `json:"service_id"`
	StripPrefix    bool      `json:"strip_prefix"`
	RequiresAuth   bool      `json:"requires_auth"`
	RateLimitRuleID string   `json:"rate_limit_rule_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (r *Route) Validate() error {
	r.Path = strings.TrimSpace(r.Path)
	r.Method = strings.ToUpper(strings.TrimSpace(r.Method))
	r.ServiceID = strings.TrimSpace(r.ServiceID)
	if r.Path == "" || !strings.HasPrefix(r.Path, "/") {
		return NewValidationError("path", "路由路径必须以 / 开头")
	}
	if r.Method == "" {
		r.Method = "GET"
	}
	if r.Method != "GET" && r.Method != "POST" && r.Method != "PUT" && r.Method != "PATCH" && r.Method != "DELETE" {
		return NewValidationError("method", "不支持的 HTTP 方法")
	}
	if r.ServiceID == "" {
		return NewValidationError("service_id", "上游服务 ID 不能为空")
	}
	return nil
}

// RouteFilter 路由列表筛选条件。
type RouteFilter struct {
	ServiceID string
	Method    string
}

func (f RouteFilter) Match(r *Route) bool {
	if f.ServiceID != "" && r.ServiceID != f.ServiceID {
		return false
	}
	if f.Method != "" && r.Method != f.Method {
		return false
	}
	return true
}
