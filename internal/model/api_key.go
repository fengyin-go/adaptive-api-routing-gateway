package model

import (
	"strings"
	"time"
)

const (
	APIKeyActive   = "active"
	APIKeyDisabled = "disabled"
)

// APIKey 接入应用用于鉴权的密钥。
type APIKey struct {
	ID        string    `json:"id"`
	AppID     string    `json:"app_id"`
	Key       string    `json:"key"`
	Secret    string    `json:"secret,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func (k *APIKey) Validate() error {
	k.AppID = strings.TrimSpace(k.AppID)
	k.Key = strings.TrimSpace(k.Key)
	if k.AppID == "" {
		return NewValidationError("app_id", "应用 ID 不能为空")
	}
	if k.Key == "" {
		return NewValidationError("key", "密钥不能为空")
	}
	if k.Status == "" {
		k.Status = APIKeyActive
	}
	if k.Status != APIKeyActive && k.Status != APIKeyDisabled {
		return NewValidationError("status", "密钥状态不合法")
	}
	if !k.ExpiresAt.IsZero() && k.ExpiresAt.Before(time.Now()) {
		return NewValidationError("expires_at", "过期时间不能早于当前时间")
	}
	return nil
}

// IsExpired 判断密钥是否已过期。
func (k *APIKey) IsExpired() bool {
	return !k.ExpiresAt.IsZero() && k.ExpiresAt.Before(time.Now())
}
