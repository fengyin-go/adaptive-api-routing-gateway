package service

import (
	"context"
	"net/http"
	"sort"
	"time"

	"apigateway/internal/model"
	"apigateway/pkg/idgen"
)

// ListHealthChecks 列出全部健康检查结果。
func (s *Service) ListHealthChecks(page, size int) ([]*model.HealthCheck, int, error) {
	all := s.store.ListHealthChecks()
	sort.Slice(all, func(i, j int) bool {
		return all[i].LastChecked.After(all[j].LastChecked)
	})
	return paginate(all, page, size)
}

// GetHealthCheck 获取某服务的健康检查结果。
func (s *Service) GetHealthCheck(serviceID string) (*model.HealthCheck, error) {
	return s.store.GetHealthCheckByService(serviceID)
}

// ProbeService 主动探测一次上游服务，更新健康检查记录。
func (s *Service) ProbeService(serviceID string) (*model.HealthCheck, error) {
	svc, err := s.store.GetService(serviceID)
	if err != nil {
		return nil, err
	}
	hc, err := s.store.GetHealthCheckByService(serviceID)
	if err != nil {
		hc = &model.HealthCheck{
			ID:        idgen.Hex(),
			ServiceID: serviceID,
			Status:    model.HealthHealthy,
		}
		_ = s.store.CreateHealthCheck(hc)
	}

	start := time.Now()
	ok, latency := s.ping(svc.BaseURL, svc.Timeout)
	hc.Record(ok, latency)
	if err := s.store.UpdateHealthCheck(hc); err != nil {
		return nil, err
	}
	_ = start
	return hc, nil
}

// ping 向上游发起一次探测请求，返回是否成功与耗时（毫秒）。
func (s *Service) ping(baseURL string, timeoutSec int) (bool, int64) {
	if timeoutSec <= 0 {
		timeoutSec = 5
	}
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	start := time.Now()
	resp, err := client.Get(baseURL)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return false, latency
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 500, latency
}

// StartHealthChecker 启动后台健康检查循环，定期探测全部 active 服务。
// 通过传入的 context 控制退出。
func (s *Service) StartHealthChecker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	s.log.Infof("健康检查循环已启动，间隔 %s", interval)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				s.log.Infof("健康检查循环已停止")
				return
			case <-ticker.C:
				for _, svc := range s.store.ListServices() {
					if svc.Status == model.ServiceActive {
						if _, err := s.ProbeService(svc.ID); err != nil {
							s.log.Warnf("探测服务 %s 失败: %v", svc.Name, err)
						}
					}
				}
			}
		}
	}()
}
