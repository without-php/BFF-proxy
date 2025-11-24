package web

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/without-php/BFF-proxy/internal/config"
)

// AdminAuthMiddleware 管理后台认证中间件
func AdminAuthMiddleware(cfg *config.Config) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		// 获取配置中的 cookie key 和 value
		cookieKey := cfg.AdminAuth.CookieKey
		cookieValue := cfg.AdminAuth.CookieValue

		// 如果未配置，使用默认值
		if cookieKey == "" {
			cookieKey = "bff_admin_token"
		}
		if cookieValue == "" {
			cookieValue = "change_me_in_production"
		}

		// 获取请求中的 cookie
		cookieValueFromReq := string(c.Cookie(cookieKey))
		if cookieValueFromReq == "" {
			// Cookie 不存在，返回 404
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// 检查 cookie 值是否匹配
		if cookieValueFromReq != cookieValue {
			// Cookie 值不匹配，返回 404
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// 认证通过，继续处理请求
		c.Next(ctx)
	}
}

// AdminAuthMiddlewareFromConfig 从全局配置获取认证中间件
func AdminAuthMiddlewareFromConfig() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		cfg := config.GetConfig()
		if cfg == nil {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// 获取配置中的 cookie key 和 value
		cookieKey := cfg.AdminAuth.CookieKey
		cookieValue := cfg.AdminAuth.CookieValue

		// 如果未配置，使用默认值
		if cookieKey == "" {
			cookieKey = "bff_admin_token"
		}
		if cookieValue == "" {
			cookieValue = "change_me_in_production"
		}

		// 获取请求中的 cookie
		cookieValueFromReq := string(c.Cookie(cookieKey))
		if cookieValueFromReq == "" {
			// Cookie 不存在，返回 404
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// 检查 cookie 值是否匹配
		if cookieValueFromReq != cookieValue {
			// Cookie 值不匹配，返回 404
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// 认证通过，继续处理请求
		c.Next(ctx)
	}
}
