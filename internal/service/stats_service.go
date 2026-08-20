package service

import (
	"sort"

	"apigateway/internal/model"
)

// RouteTraffic 单条路由的流量统计。
type RouteTraffic struct {
	RouteID   string `json:"route_id"`
	Path      string `json:"path"`
	Method    string `json:"method"`
	Requests  int    `json:"requests"`
	Errors    int    `json:"errors"`
	AvgLatency int64 `json:"avg_latency_ms"`
}

// GatewayStats 网关全局统计。
type GatewayStats struct {
	Services       int            `json:"services"`
	ActiveServices int            `json:"active_services"`
	Unhealthy      int            `json:"unhealthy_services"`
	Routes         int            `json:"routes"`
	Apps           int            `json:"apps"`
	APIKeys        int            `json:"api_keys"`
	Requests       RequestCounts  `json:"requests"`
	TopRoutes      []RouteTraffic `json:"top_routes"`
}

// RequestCounts 请求状态统计。
type RequestCounts struct {
	Total    int `json:"total"`
	Success  int `json:"success"`  // 2xx
	ClientErr int `json:"client_err"` // 4xx
	ServerErr int `json:"server_err"` // 5xx
	Limited  int `json:"limited"`  // 429
}

// GetGatewayStats 汇总网关运行统计。
func (s *Service) GetGatewayStats() (*GatewayStats, error) {
	stats := &GatewayStats{}
	stats.Services = len(s.store.ListServices())
	stats.Routes = len(s.store.ListRoutes())
	stats.Apps = len(s.store.ListApps())
	stats.APIKeys = len(s.store.ListAPIKeys())

	// 服务与健康状态
	routeByID := make(map[string]*model.Route, stats.Routes)
	for _, r := range s.store.ListRoutes() {
		routeByID[r.ID] = r
	}
	for _, svc := range s.store.ListServices() {
		if svc.Status == model.ServiceActive {
			stats.ActiveServices++
		}
	}
	for _, hc := range s.store.ListHealthChecks() {
		if hc.Status == model.HealthUnhealthy {
			stats.Unhealthy++
		}
	}

	// 请求统计
	traffic := make(map[string]*RouteTraffic)
	for _, l := range s.store.ListRequestLogs() {
		stats.Requests.Total++
		switch {
		case l.StatusCode == 429:
			stats.Requests.Limited++
		case l.StatusCode >= 500:
			stats.Requests.ServerErr++
		case l.StatusCode >= 400:
			stats.Requests.ClientErr++
		default:
			stats.Requests.Success++
		}
		tr, ok := traffic[l.RouteID]
		if !ok {
			tr = &RouteTraffic{RouteID: l.RouteID}
			if r, ok := routeByID[l.RouteID]; ok {
				tr.Path = r.Path
				tr.Method = r.Method
			}
			traffic[l.RouteID] = tr
		}
		tr.Requests++
		if l.StatusCode >= 400 {
			tr.Errors++
		}
		tr.AvgLatency = (tr.AvgLatency*int64(tr.Requests-1) + l.LatencyMs) / int64(tr.Requests)
	}

	stats.TopRoutes = make([]RouteTraffic, 0, len(traffic))
	for _, tr := range traffic {
		stats.TopRoutes = append(stats.TopRoutes, *tr)
	}
	sort.Slice(stats.TopRoutes, func(i, j int) bool {
		return stats.TopRoutes[i].Requests > stats.TopRoutes[j].Requests
	})
	if len(stats.TopRoutes) > 10 {
		stats.TopRoutes = stats.TopRoutes[:10]
	}
	return stats, nil
}
