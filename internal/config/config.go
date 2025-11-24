package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

var (
	globalConfig *Config
	configMutex  sync.RWMutex
)

// Config 应用配置
type Config struct {
	Server   ServerConfig `yaml:"server" json:"server"`
	Proxy    ProxyConfig  `yaml:"proxy" json:"proxy"`
	Log      LogConfig    `yaml:"log" json:"log"`
	AdminAuth AdminAuthConfig `yaml:"admin_auth" json:"admin_auth"`
}

// AdminAuthConfig 管理后台认证配置
type AdminAuthConfig struct {
	CookieKey   string `yaml:"cookie_key" json:"cookie_key"`     // Cookie 键名
	CookieValue string `yaml:"cookie_value" json:"cookie_value"` // Cookie 值
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int `yaml:"port" json:"port"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Rules []ProxyRule `yaml:"rules" json:"rules"`
}

// ProxyRule 代理规则
type ProxyRule struct {
	Name        string            `yaml:"name" json:"name"`
	Match       MatchCondition    `yaml:"match" json:"match"`
	Target      string            `yaml:"target" json:"target"`
	Timeout     int               `yaml:"timeout" json:"timeout"`           // 超时时间（秒）
	Headers     map[string]string `yaml:"headers" json:"headers"`          // 额外添加的请求头
	RewritePath string            `yaml:"rewrite_path" json:"rewrite_path"` // 路径重写
}

// MatchCondition 匹配条件
type MatchCondition struct {
	Path    string            `yaml:"path" json:"path"`         // 路径匹配（支持前缀匹配）
	Method  string            `yaml:"method" json:"method"`     // HTTP 方法
	Headers map[string]string `yaml:"headers" json:"headers"`   // Header 匹配
	Query   map[string]string `yaml:"query" json:"query"`        // Query 参数匹配
	Body    map[string]string `yaml:"body" json:"body"`         // Body 参数匹配（仅支持 JSON）
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level" json:"level"`             // debug, info, warn, error
	File       string `yaml:"file" json:"file"`               // 日志文件路径
	MaxSize    int    `yaml:"max_size" json:"max_size"`       // 最大文件大小（MB）
	MaxBackups int    `yaml:"max_backups" json:"max_backups"` // 保留的备份文件数
	MaxAge     int    `yaml:"max_age" json:"max_age"`         // 保留天数
}

// LoadConfig 加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	// 服务器端口固定，不允许修改（安全考虑）
	cfg.Server.Port = 8080
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	// 日志文件路径固定，不允许修改（安全考虑）
	cfg.Log.File = "logs/bff-proxy.log"
	if cfg.Log.MaxSize == 0 {
		cfg.Log.MaxSize = 100
	}
	if cfg.Log.MaxBackups == 0 {
		cfg.Log.MaxBackups = 10
	}
	if cfg.Log.MaxAge == 0 {
		cfg.Log.MaxAge = 30
	}
	// 管理后台认证配置默认值
	if cfg.AdminAuth.CookieKey == "" {
		cfg.AdminAuth.CookieKey = "bff_admin_token"
	}
	if cfg.AdminAuth.CookieValue == "" {
		cfg.AdminAuth.CookieValue = "change_me_in_production"
	}

	configMutex.Lock()
	globalConfig = &cfg
	configMutex.Unlock()

	return &cfg, nil
}

// GetConfig 获取当前配置
func GetConfig() *Config {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return globalConfig
}

// SaveConfig 保存配置
func SaveConfig(cfg *Config, path string) error {
	// 设置默认值，确保配置完整
	// 服务器端口固定，不允许修改（安全考虑）
	cfg.Server.Port = 8080
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	// 日志文件路径固定，不允许修改（安全考虑）
	cfg.Log.File = "logs/bff-proxy.log"
	if cfg.Log.MaxSize == 0 {
		cfg.Log.MaxSize = 100
	}
	if cfg.Log.MaxBackups == 0 {
		cfg.Log.MaxBackups = 10
	}
	if cfg.Log.MaxAge == 0 {
		cfg.Log.MaxAge = 30
	}
	// 管理后台认证配置默认值
	if cfg.AdminAuth.CookieKey == "" {
		cfg.AdminAuth.CookieKey = "bff_admin_token"
	}
	if cfg.AdminAuth.CookieValue == "" {
		cfg.AdminAuth.CookieValue = "change_me_in_production"
	}

	// 确保每个规则都有完整的结构
	for i := range cfg.Proxy.Rules {
		rule := &cfg.Proxy.Rules[i]
		if rule.Timeout == 0 {
			rule.Timeout = 30
		}
		if rule.Match.Headers == nil {
			rule.Match.Headers = make(map[string]string)
		}
		if rule.Match.Query == nil {
			rule.Match.Query = make(map[string]string)
		}
		if rule.Match.Body == nil {
			rule.Match.Body = make(map[string]string)
		}
		if rule.Headers == nil {
			rule.Headers = make(map[string]string)
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	// 更新内存中的配置
	configMutex.Lock()
	globalConfig = cfg
	configMutex.Unlock()

	return nil
}

// WatchConfig 监听配置文件变化
func WatchConfig(path string, callback func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				// 延迟一下，确保文件写入完成
				time.Sleep(100 * time.Millisecond)
				if _, err := LoadConfig(path); err == nil {
					callback()
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("配置文件监听错误: %v\n", err)
		}
	}
}
