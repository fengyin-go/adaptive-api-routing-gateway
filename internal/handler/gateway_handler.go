package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"apigateway/internal/model"
	"apigateway/pkg/httpx"
)

// gatewayHandler 网关转发入口：路由匹配 -> 鉴权 -> 限流 -> 反向代理 -> 记录日志。
func (s *Server) gatewayHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// 1. 路由匹配
	route, ok := s.svc.MatchRoute(r.Method, r.URL.Path)
	if !ok {
		httpx.NotFound(w, "未找到匹配的路由")
		return
	}

	// 2. 鉴权
	var appID string
	if route.RequiresAuth {
		app, err := s.svc.Authenticate(r.Header.Get("X-API-Key"))
		if err != nil {
			s.recordGatewayLog(route, "", r, http.StatusUnauthorized, start)
			httpx.Unauthorized(w, err.Error())
			return
		}
		appID = app.ID
	}

	// 3. 限流
	if !s.svc.AllowRequest(route) {
		s.recordGatewayLog(route, appID, r, http.StatusTooManyRequests, start)
		httpx.Error(w, http.StatusTooManyRequests, 429, "请求过于频繁，请稍后重试")
		return
	}

	// 4. 解析上游
	svc, err := s.svc.GetService(route.ServiceID)
	if err != nil {
		s.recordGatewayLog(route, appID, r, http.StatusBadGateway, start)
		httpx.InternalError(w, "上游服务不存在")
		return
	}

	// 5. 熔断检查
	if !s.svc.AllowService(svc.ID) {
		s.recordGatewayLog(route, appID, r, http.StatusServiceUnavailable, start)
		httpx.Error(w, http.StatusServiceUnavailable, 503, "上游服务熔断中，请稍后重试")
		return
	}

	// 6. 反向代理转发
	status := s.proxyRequest(w, r, svc, route)
	s.recordGatewayLog(route, appID, r, status, start)
}

// proxyRequest 将请求反向代理到上游服务，返回响应状态码。
func (s *Server) proxyRequest(w http.ResponseWriter, r *http.Request, svc *model.Service, route *model.Route) int {
	target, err := url.Parse(svc.BaseURL)
	if err != nil {
		httpx.InternalError(w, "上游地址非法")
		return http.StatusBadGateway
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		origPath := req.URL.Path
		if route.StripPrefix && strings.HasPrefix(origPath, route.Path) {
			trimmed := strings.TrimPrefix(origPath, route.Path)
			if trimmed == "" {
				trimmed = "/"
			}
			req.URL.Path = trimmed
		}
	}

	status := http.StatusOK
	proxy.ModifyResponse = func(resp *http.Response) error {
		status = resp.StatusCode
		return nil
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		status = http.StatusBadGateway
		httpx.Error(rw, http.StatusBadGateway, 502, "上游服务不可用")
	}

	proxy.ServeHTTP(w, r)
	// 记录熔断结果：5xx 视为失败
	s.svc.RecordServiceResult(svc.ID, status < 500)
	return status
}

// recordGatewayLog 记录一次网关访问日志。
func (s *Server) recordGatewayLog(route *model.Route, appID string, r *http.Request, status int, start time.Time) {
	s.svc.RecordRequest(model.RequestLog{
		RouteID:    route.ID,
		AppID:      appID,
		Method:     r.Method,
		Path:       r.URL.Path,
		StatusCode: status,
		LatencyMs:  time.Since(start).Milliseconds(),
		ClientIP:   clientIP(r),
		Timestamp:  start,
	})
}

// clientIP 提取客户端 IP（优先 X-Forwarded-For，其次 RemoteAddr）。
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		return host[:idx]
	}
	return host
}
