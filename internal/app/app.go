// Package app 负责依赖装配。
package app

import (
	"context"
	"net/http"
	"time"

	"apigateway/internal/config"
	"apigateway/internal/handler"
	"apigateway/internal/service"
	"apigateway/internal/store"
	"apigateway/pkg/logger"
)

type App struct {
	server *handler.Server
	svc    *service.Service
	log    *logger.Logger
	cfg    *config.Config
}

func New(cfg *config.Config, log *logger.Logger) (*App, error) {
	st := store.NewMemoryStore()
	svc := service.New(st, log, cfg)
	server := handler.NewServer(svc, log, cfg)
	log.Infof("应用装配完成，配置：%s", cfg.String())
	return &App{server: server, svc: svc, log: log, cfg: cfg}, nil
}

func (a *App) Routes() http.Handler { return a.server.Routes() }

// StartHealthChecker 启动后台健康检查循环。
func (a *App) StartHealthChecker(ctx context.Context) {
	interval := time.Duration(a.cfg.HealthCheckInterval) * time.Second
	a.svc.StartHealthChecker(ctx, interval)
}
