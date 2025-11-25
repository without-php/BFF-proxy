package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/without-php/BFF-proxy/internal/config"
	"github.com/without-php/BFF-proxy/internal/logger"
)

// ProxyMiddleware 代理中间件
type ProxyMiddleware struct {
	config *config.Config
}

// NewProxyMiddleware 创建代理中间件
func NewProxyMiddleware(cfg *config.Config) *ProxyMiddleware {
	return &ProxyMiddleware{
		config: cfg,
	}
}

// Handle 处理请求
func (p *ProxyMiddleware) Handle(ctx context.Context, c *app.RequestContext) {
	// 跳过管理界面路由
	path := string(c.Path())
	if strings.HasPrefix(path, "/admin") {
		c.Next(ctx)
		return
	}

	startTime := time.Now()
	cfg := config.GetConfig()

	// 读取请求体
	bodyBytes := c.Request.BodyBytes()
	c.Request.SetBody(bodyBytes)

	// 查找匹配的规则
	rule := p.findMatchingRule(c, cfg)

	// 准备请求日志（无论是否找到规则都要记录）
	queryString := string(c.QueryArgs().QueryString())
	reqLog := &logger.RequestLog{
		StartTime: startTime,
		Method:    string(c.Method()),
		Path:      path,
		Query:     queryString,
		Headers:   p.extractHeaders(c),
		Body:      string(bodyBytes),
	}

	if rule == nil {
		// 没有找到匹配的规则，也要记录日志
		reqLog.EndTime = time.Now()
		reqLog.Duration = reqLog.EndTime.Sub(reqLog.StartTime)
		reqLog.StatusCode = http.StatusNotFound
		reqLog.ResponseBody = "404 page not found"
		reqLog.Target = ""
		reqLog.RuleName = ""
		reqLog.Error = "没有找到匹配的代理规则"

		// 记录日志
		logger.LogRequest(reqLog)

		// 返回404，不暴露代理信息
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// 设置规则信息
	reqLog.Target = rule.Target
	reqLog.RuleName = rule.Name

	// 执行代理转发
	statusCode, responseBody, err := p.proxyRequest(ctx, c, rule)

	reqLog.EndTime = time.Now()
	reqLog.Duration = reqLog.EndTime.Sub(reqLog.StartTime)
	reqLog.StatusCode = statusCode
	reqLog.ResponseBody = responseBody
	if err != nil {
		reqLog.Error = err.Error()
	}

	// 记录日志
	logger.LogRequest(reqLog)

	// 返回响应
	if err != nil {
		c.JSON(http.StatusBadGateway, map[string]string{
			"error": fmt.Sprintf("代理请求失败: %v", err),
		})
		return
	}

	c.Data(statusCode, consts.MIMEApplicationJSON, []byte(responseBody))
}

// findMatchingRule 查找匹配的规则
func (p *ProxyMiddleware) findMatchingRule(c *app.RequestContext, cfg *config.Config) *config.ProxyRule {
	path := string(c.Path())
	method := string(c.Method())

	for _, rule := range cfg.Proxy.Rules {
		match := rule.Match

		// 路径匹配
		if match.Path != "" {
			if !strings.HasPrefix(path, match.Path) {
				continue
			}
		}

		// 方法匹配
		if match.Method != "" {
			if strings.ToUpper(match.Method) != strings.ToUpper(method) {
				continue
			}
		}

		// Header 匹配
		if len(match.Headers) > 0 {
			if !p.matchHeaders(c, match.Headers) {
				continue
			}
		}

		// Query 匹配
		if len(match.Query) > 0 {
			if !p.matchQuery(c, match.Query) {
				continue
			}
		}

		// Body 匹配
		if len(match.Body) > 0 {
			if !p.matchBody(c, match.Body) {
				continue
			}
		}

		return &rule
	}

	return nil
}

// matchHeaders 匹配请求头
func (p *ProxyMiddleware) matchHeaders(c *app.RequestContext, conditions map[string]string) bool {
	for key, value := range conditions {
		headerValue := string(c.GetHeader(key))
		if headerValue != value {
			return false
		}
	}
	return true
}

// matchQuery 匹配查询参数
func (p *ProxyMiddleware) matchQuery(c *app.RequestContext, conditions map[string]string) bool {
	for key, value := range conditions {
		queryValue := string(c.Query(key))
		if queryValue != value {
			return false
		}
	}
	return true
}

// matchBody 匹配请求体（仅支持 JSON）
func (p *ProxyMiddleware) matchBody(c *app.RequestContext, conditions map[string]string) bool {
	bodyBytes := c.Request.BodyBytes()
	if len(bodyBytes) == 0 {
		return false
	}

	var bodyMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
		return false
	}

	for key, value := range conditions {
		if bodyValue, ok := bodyMap[key]; !ok {
			return false
		} else {
			if fmt.Sprintf("%v", bodyValue) != value {
				return false
			}
		}
	}

	return true
}

// proxyRequest 执行代理请求
func (p *ProxyMiddleware) proxyRequest(ctx context.Context, c *app.RequestContext, rule *config.ProxyRule) (int, string, error) {
	// 构建目标 URL
	targetURL := rule.Target
	path := string(c.Path())

	// 路径重写
	if rule.RewritePath != "" {
		path = strings.Replace(path, rule.Match.Path, rule.RewritePath, 1)
	}
	targetURL = strings.TrimSuffix(targetURL, "/") + path

	// 添加查询参数
	queryArgs := c.QueryArgs()
	if queryArgs.Len() > 0 {
		targetURL += "?" + string(queryArgs.QueryString())
	}

	// 创建请求
	timeout := time.Duration(rule.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, string(c.Method()), targetURL, bytes.NewReader(c.Request.BodyBytes()))
	if err != nil {
		return 0, "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 复制请求头
	c.Request.Header.VisitAll(func(key, value []byte) {
		req.Header.Add(string(key), string(value))
	})

	// 添加额外请求头
	for key, value := range rule.Headers {
		req.Header.Set(key, value)
	}

	// 检查请求是否是 SSE 请求
	isSSERequest := strings.Contains(strings.ToLower(req.Header.Get("Accept")), "text/event-stream")

	// 发送请求
	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应是否是 SSE
	isSSEResponse := strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream")
	isSSE := isSSERequest || isSSEResponse

	// 如果是 SSE，需要流式传输
	if isSSE {
		// 复制响应头
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		// 设置状态码
		c.Status(resp.StatusCode)

		// 流式传输响应体
		buffer := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				c.Write(buffer[:n])
				c.Flush()
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return resp.StatusCode, "", fmt.Errorf("读取响应流失败: %w", err)
			}
		}
		return resp.StatusCode, "[SSE Stream]", nil
	}

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("读取响应失败: %w", err)
	}

	return resp.StatusCode, string(bodyBytes), nil
}

// extractHeaders 提取请求头
func (p *ProxyMiddleware) extractHeaders(c *app.RequestContext) map[string]string {
	headers := make(map[string]string)
	c.Request.Header.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		if _, exists := headers[keyStr]; !exists {
			headers[keyStr] = string(value)
		}
	})
	return headers
}
