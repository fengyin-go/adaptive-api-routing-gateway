package model

import (
	"time"
)

const (
	HealthHealthy   = "healthy"
	HealthUnhealthy = "unhealthy"
)

// HealthCheck 上游服务的健康检查结果。
type HealthCheck struct {
	ID           string    `json:"id"`
	ServiceID    string    `json:"service_id"`
	Status       string    `json:"status"`
	LatencyMs    int64     `json:"latency_ms"`
	FailureCount int       `json:"failure_count"`
	LastChecked  time.Time `json:"last_checked"`
}

// Record 根据一次探测结果更新健康状态。
func (h *HealthCheck) Record(ok bool, latencyMs int64) {
	h.LastChecked = time.Now()
	h.LatencyMs = latencyMs
	if ok {
		h.Status = HealthHealthy
		h.FailureCount = 0
	} else {
		h.FailureCount++
		if h.FailureCount >= 3 {
			h.Status = HealthUnhealthy
		}
	}
}
