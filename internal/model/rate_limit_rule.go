package model

import (
	"strings"
	"time"
)

// RateLimitRule 限流规则：在 Window 秒内最多允许 Limit 次请求。
type RateLimitRule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Limit     int       `json:"limit"`  // 窗口内允许的最大请求数
	Window    int       `json:"window"` // 时间窗口，单位：秒
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *RateLimitRule) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return NewValidationError("name", "规则名称不能为空")
	}
	if r.Limit <= 0 {
		return NewValidationError("limit", "限额必须大于 0")
	}
	if r.Limit > 1_000_000 {
		return NewValidationError("limit", "限额不能超过 1000000")
	}
	if r.Window <= 0 {
		r.Window = 60
	}
	if r.Window > 86400 {
		return NewValidationError("window", "时间窗口不能超过 86400 秒")
	}
	return nil
}

// RatePerSecond 计算每秒允许的令牌数。
func (r *RateLimitRule) RatePerSecond() float64 {
	if r.Window <= 0 {
		return float64(r.Limit)
	}
	return float64(r.Limit) / float64(r.Window)
}
