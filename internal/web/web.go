package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/without-php/BFF-proxy/internal/config"
	"github.com/without-php/BFF-proxy/internal/logger"
)

// RegisterRoutes 注册 Web UI 路由
func RegisterRoutes(h *server.Hertz, cfg *config.Config) {
	admin := h.Group("/admin")
	// 为所有 admin 路由添加认证中间件
	admin.Use(AdminAuthMiddlewareFromConfig())
	{
		// 根路径直接返回 index.html
		admin.GET("/", func(ctx context.Context, c *app.RequestContext) {
			c.File("./web/static/index.html")
		})

		// 静态文件服务（处理其他静态资源）
		admin.StaticFile("/index.html", "./web/static/index.html")

		// API 路由
		api := admin.Group("/api")
		{
			// 获取配置
			api.GET("/config", getConfig)
			// 更新配置
			api.POST("/config", updateConfig)
			// 获取日志
			api.GET("/logs", getLogs)
		}
	}
}

// getConfig 获取配置
func getConfig(ctx context.Context, c *app.RequestContext) {
	cfg := config.GetConfig()
	c.JSON(http.StatusOK, cfg)
}

// updateConfig 更新配置
func updateConfig(ctx context.Context, c *app.RequestContext) {
	var cfg config.Config
	if err := c.BindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, map[string]string{
			"error": "无效的配置格式: " + err.Error(),
		})
		return
	}

	if err := config.SaveConfig(&cfg, "config.yaml"); err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "保存配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"message": "配置已更新",
	})
}

// getLogs 获取日志
func getLogs(ctx context.Context, c *app.RequestContext) {
	limit := c.DefaultQuery("limit", "100")
	var limitInt int
	if _, err := fmt.Sscanf(limit, "%d", &limitInt); err != nil {
		limitInt = 100
	}

	logs, err := logger.GetLogs(limitInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "获取日志失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, logs)
}
