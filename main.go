package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/without-php/BFF-proxy/internal/config"
	"github.com/without-php/BFF-proxy/internal/logger"
	"github.com/without-php/BFF-proxy/internal/proxy"
	"github.com/without-php/BFF-proxy/internal/web"
)

func main() {
	// 初始化日志
	logger.InitLogger()

	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		hlog.Fatalf("加载配置失败: %v", err)
	}

	// 启动配置热加载
	go config.WatchConfig("config.yaml", func() {
		hlog.Info("配置已重新加载")
	})

	// 创建 Hertz 服务器
	h := server.Default(
		server.WithHostPorts(fmt.Sprintf(":%d", cfg.Server.Port)),
	)

	// 注册代理中间件
	proxyMiddleware := proxy.NewProxyMiddleware(cfg)
	h.Use(proxyMiddleware.Handle)

	// 注册 Web UI 路由
	web.RegisterRoutes(h, cfg)

	hlog.Infof("BFF Proxy 服务启动在端口 %d", cfg.Server.Port)
	hlog.Info("Web UI 访问地址: http://localhost:" + fmt.Sprintf("%d", cfg.Server.Port) + "/admin")

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		hlog.Info("正在关闭服务器...")
		if err := h.Shutdown(context.Background()); err != nil {
			hlog.Errorf("服务器关闭失败: %v", err)
		}
		logger.Close()
		os.Exit(0)
	}()

	// 启动服务器
	h.Spin()
}
