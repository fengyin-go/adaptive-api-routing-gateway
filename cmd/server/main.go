package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"apigateway/internal/app"
	"apigateway/internal/config"
	"apigateway/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.NewLevel(logger.ParseLevel(os.Getenv("LOG_LEVEL")))

	application, err := app.New(cfg, log)
	if err != nil {
		log.Errorf("应用初始化失败: %v", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	application.StartHealthChecker(ctx)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           application.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Infof("API 网关已启动，监听 %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("服务启动失败: %v", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infof("正在关闭服务...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Errorf("服务关闭失败: %v", err)
	}
	log.Infof("服务已关闭")
}
