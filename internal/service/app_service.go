package service

import (
	"sort"
	"time"

	"apigateway/internal/model"
	"apigateway/pkg/idgen"
)

func (s *Service) CreateApp(a model.App) (*model.App, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	a.ID = idgen.Hex()
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now
	if err := s.store.CreateApp(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) GetApp(id string) (*model.App, error) {
	return s.store.GetApp(id)
}

func (s *Service) ListApps(filter model.AppFilter, page, size int) ([]*model.App, int, error) {
	all := s.store.ListApps()
	matched := make([]*model.App, 0, len(all))
	for _, a := range all {
		if filter.Match(a) {
			matched = append(matched, a)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})
	return paginate(matched, page, size)
}

func (s *Service) UpdateApp(id string, a model.App) (*model.App, error) {
	exist, err := s.store.GetApp(id)
	if err != nil {
		return nil, err
	}
	exist.Name = a.Name
	if a.Status != "" {
		exist.Status = a.Status
	}
	exist.UpdatedAt = time.Now()
	if err := exist.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.UpdateApp(exist); err != nil {
		return nil, err
	}
	return exist, nil
}

func (s *Service) DeleteApp(id string) error {
	return s.store.DeleteApp(id)
}

// CreateAPIKey 为应用生成一个 API 密钥。
func (s *Service) CreateAPIKey(appID string, secret string, expiresAt time.Time) (*model.APIKey, error) {
	if _, err := s.store.GetApp(appID); err != nil {
		return nil, err
	}
	k := &model.APIKey{
		AppID:     appID,
		Key:       "ak_" + idgen.HexN(12),
		Secret:    secret,
		Status:    model.APIKeyActive,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
	k.ID = idgen.Hex()
	if err := k.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.CreateAPIKey(k); err != nil {
		return nil, err
	}
	return k, nil
}

func (s *Service) GetAPIKey(id string) (*model.APIKey, error) {
	return s.store.GetAPIKey(id)
}

func (s *Service) ListAPIKeys(appID string, page, size int) ([]*model.APIKey, int, error) {
	all := s.store.ListAPIKeys()
	matched := make([]*model.APIKey, 0, len(all))
	for _, k := range all {
		if appID == "" || k.AppID == appID {
			matched = append(matched, k)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})
	return paginate(matched, page, size)
}

// UpdateAPIKeyStatus 启用/禁用密钥。
func (s *Service) UpdateAPIKeyStatus(id, status string) (*model.APIKey, error) {
	if status != model.APIKeyActive && status != model.APIKeyDisabled {
		return nil, model.NewValidationError("status", "密钥状态不合法")
	}
	k, err := s.store.GetAPIKey(id)
	if err != nil {
		return nil, err
	}
	k.Status = status
	if err := s.store.UpdateAPIKey(k); err != nil {
		return nil, err
	}
	return k, nil
}

func (s *Service) DeleteAPIKey(id string) error {
	return s.store.DeleteAPIKey(id)
}

// Authenticate 校验 API Key，返回对应的应用。密钥需 active 且未过期。
func (s *Service) Authenticate(key string) (*model.App, error) {
	k, err := s.store.GetAPIKeyByKey(key)
	if err != nil {
		return nil, err
	}
	if k.Status != model.APIKeyActive {
		return nil, model.NewValidationError("key", "密钥已禁用")
	}
	if k.IsExpired() {
		return nil, model.NewValidationError("key", "密钥已过期")
	}
	app, err := s.store.GetApp(k.AppID)
	if err != nil {
		return nil, err
	}
	if app.Status != model.AppActive {
		return nil, model.NewValidationError("app", "应用已禁用")
	}
	return app, nil
}
