// Package handler 实现 HTTP 处理器层与网关转发。
package handler

import (
	"errors"
	"net/http"
	"runtime/debug"
	"time"

	"apigateway/internal/config"
	"apigateway/internal/middleware"
	"apigateway/internal/model"
	"apigateway/internal/service"
	"apigateway/internal/store"
	"apigateway/pkg/httpx"
	"apigateway/pkg/logger"
)

// Server HTTP 服务器，装配管理与网关转发路由。
type Server struct {
	svc *service.Service
	log *logger.Logger
	cfg *config.Config
}

func NewServer(svc *service.Service, log *logger.Logger, cfg *config.Config) *Server {
	return &Server{svc: svc, log: log, cfg: cfg}
}

// Routes 组装全部路由：管理 API 挂 /api/ 前缀（带管理员鉴权），其余走网关转发。
func (s *Server) Routes() http.Handler {
	admin := http.NewServeMux()
	s.registerServiceRoutes(admin)
	s.registerRouteRoutes(admin)
	s.registerAppRoutes(admin)
	s.registerAPIKeyRoutes(admin)
	s.registerRateLimitRuleRoutes(admin)
	s.registerRequestLogRoutes(admin)
	s.registerHealthRoutes(admin)
	s.registerStatsRoutes(admin)
	s.registerExportRoutes(admin)

	mux := http.NewServeMux()
	mux.Handle("/api/", middleware.AdminAuth(s.cfg.AdminToken)(admin))
	mux.HandleFunc("/", s.gatewayHandler)

	return s.loggingMiddleware(s.recoveryMiddleware(mux))
}

func (s *Server) maxPageSize() int {
	if s.cfg != nil && s.cfg.MaxPageSize > 0 {
		return s.cfg.MaxPageSize
	}
	return 100
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.log.Infof("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Errorf("panic: %v\n%s", rec, debug.Stack())
				httpx.InternalError(w, "服务器内部错误")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case model.IsValidationError(err):
		httpx.BadRequest(w, err.Error())
	case errors.Is(err, store.ErrNotFound):
		httpx.NotFound(w, err.Error())
	case errors.Is(err, store.ErrConflict):
		httpx.Conflict(w, err.Error())
	default:
		httpx.InternalError(w, err.Error())
	}
}
