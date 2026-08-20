package model

import (
	"strings"
	"time"
)

const (
	ServiceActive   = "active"
	ServiceInactive = "inactive"
)

// Service 上游后端服务，网关将请求转发到其 BaseURL。
type Service struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	BaseURL   string    `json:"base_url"`
	Timeout   int       `json:"timeout"`  // 转发超时，单位：秒
	Retries   int       `json:"retries"`  // 失败重试次数
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Service) Validate() error {
	s.Name = strings.TrimSpace(s.Name)
	s.BaseURL = strings.TrimSpace(s.BaseURL)
	if s.Name == "" {
		return NewValidationError("name", "服务名称不能为空")
	}
	if s.BaseURL == "" {
		return NewValidationError("base_url", "上游地址不能为空")
	}
	if s.Timeout <= 0 {
		s.Timeout = 5
	}
	if s.Timeout > 300 {
		return NewValidationError("timeout", "超时时间不能超过 300 秒")
	}
	if s.Retries < 0 || s.Retries > 10 {
		return NewValidationError("retries", "重试次数必须在 0-10 之间")
	}
	if s.Status == "" {
		s.Status = ServiceActive
	}
	if s.Status != ServiceActive && s.Status != ServiceInactive {
		return NewValidationError("status", "服务状态不合法")
	}
	return nil
}

// ServiceFilter 服务列表筛选条件。
type ServiceFilter struct {
	Status  string
	Keyword string
}

func (f ServiceFilter) Match(s *Service) bool {
	if f.Status != "" && s.Status != f.Status {
		return false
	}
	if k := strings.ToLower(strings.TrimSpace(f.Keyword)); k != "" {
		if !strings.Contains(strings.ToLower(s.Name), k) &&
			!strings.Contains(strings.ToLower(s.BaseURL), k) {
			return false
		}
	}
	return true
}
