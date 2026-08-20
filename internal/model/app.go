package model

import (
	"strings"
	"time"
)

const (
	AppActive   = "active"
	AppDisabled = "disabled"
)

// App 接入网关的应用，通过 APIKey 关联。
type App struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (a *App) Validate() error {
	a.Name = strings.TrimSpace(a.Name)
	if a.Name == "" {
		return NewValidationError("name", "应用名称不能为空")
	}
	if a.Status == "" {
		a.Status = AppActive
	}
	if a.Status != AppActive && a.Status != AppDisabled {
		return NewValidationError("status", "应用状态不合法")
	}
	return nil
}

// AppFilter 应用列表筛选条件。
type AppFilter struct {
	Status string
}

func (f AppFilter) Match(a *App) bool {
	if f.Status != "" && a.Status != f.Status {
		return false
	}
	return true
}
