package model

import (
	"time"
)

// RequestLog 网关访问日志，记录每次转发的请求。
type RequestLog struct {
	ID         string    `json:"id"`
	RouteID    string    `json:"route_id"`
	AppID      string    `json:"app_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	LatencyMs  int64     `json:"latency_ms"`
	ClientIP   string    `json:"client_ip"`
	Timestamp  time.Time `json:"timestamp"`
}

// RequestLogFilter 日志筛选条件。
type RequestLogFilter struct {
	RouteID    string
	AppID      string
	StatusCode int
}

func (f RequestLogFilter) Match(l *RequestLog) bool {
	if f.RouteID != "" && l.RouteID != f.RouteID {
		return false
	}
	if f.AppID != "" && l.AppID != f.AppID {
		return false
	}
	if f.StatusCode != 0 && l.StatusCode != f.StatusCode {
		return false
	}
	return true
}
