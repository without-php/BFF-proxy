package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/without-php/BFF-proxy/internal/config"
)

var (
	logFile   *os.File
	logMutex  sync.Mutex
	logBuffer []*RequestLog
	bufferSize = 100
)

// RequestLog 请求日志
type RequestLog struct {
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     time.Duration     `json:"duration"`
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	Query        string            `json:"query"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`
	StatusCode   int               `json:"status_code"`
	ResponseBody string            `json:"response_body"`
	Target       string            `json:"target"`
	RuleName     string            `json:"rule_name"`
	Error        string            `json:"error,omitempty"`
}

// InitLogger 初始化日志
func InitLogger() {
	cfg := config.GetConfig()
	if cfg == nil {
		// 使用默认配置
		cfg = &config.Config{
			Log: config.LogConfig{
				File:       "logs/bff-proxy.log",
				Level:      "info",
				MaxSize:    100,
				MaxBackups: 10,
				MaxAge:     30,
			},
		}
	}

	// 创建日志目录
	logDir := filepath.Dir(cfg.Log.File)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logrus.Errorf("创建日志目录失败: %v", err)
		return
	}

	// 打开日志文件
	var err error
	logFile, err = os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logrus.Errorf("打开日志文件失败: %v", err)
		return
	}

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	logrus.SetOutput(logFile)

	// 启动日志刷新协程
	go flushLogs()
}

// LogRequest 记录请求日志
func LogRequest(log *RequestLog) {
	logMutex.Lock()
	defer logMutex.Unlock()

	logBuffer = append(logBuffer, log)

	// 如果缓冲区满了，立即刷新
	if len(logBuffer) >= bufferSize {
		flushLogsLocked()
	}
}

// flushLogs 定期刷新日志
func flushLogs() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		logMutex.Lock()
		if len(logBuffer) > 0 {
			flushLogsLocked()
		}
		logMutex.Unlock()
	}
}

// flushLogsLocked 刷新日志（需要先获取锁）
func flushLogsLocked() {
	if logFile == nil {
		return
	}

	for _, log := range logBuffer {
		logJSON, err := json.Marshal(log)
		if err != nil {
			logrus.Errorf("序列化日志失败: %v", err)
			continue
		}

		_, err = logFile.WriteString(string(logJSON) + "\n")
		if err != nil {
			logrus.Errorf("写入日志失败: %v", err)
		}
	}

	logFile.Sync()
	logBuffer = logBuffer[:0]
}

// GetLogs 获取日志（用于 Web UI）
func GetLogs(limit int) ([]*RequestLog, error) {
	cfg := config.GetConfig()
	if cfg == nil {
		return nil, fmt.Errorf("配置未加载")
	}

	// 先刷新缓冲区
	logMutex.Lock()
	if len(logBuffer) > 0 {
		flushLogsLocked()
	}
	logMutex.Unlock()

	// 读取日志文件
	file, err := os.Open(cfg.Log.File)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer file.Close()

	// 读取文件内容
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 从文件末尾读取（最多读取最后 1MB）
	maxReadSize := int64(1024 * 1024)
	readSize := stat.Size()
	if readSize > maxReadSize {
		readSize = maxReadSize
		file.Seek(-readSize, 2) // 从文件末尾向前读取
	}

	buffer := make([]byte, readSize)
	if _, err := file.Read(buffer); err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %w", err)
	}

	// 解析日志
	content := string(buffer)
	lines := strings.Split(content, "\n")
	logs := make([]*RequestLog, 0)

	// 从后往前读取，最多读取 limit 条（最新的在前，倒序）
	count := 0
	for i := len(lines) - 1; i >= 0 && count < limit; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		var log RequestLog
		if err := json.Unmarshal([]byte(line), &log); err != nil {
			continue
		}

		logs = append(logs, &log)
		count++
	}

	// 不反转顺序，保持倒序（最新的在前）
	return logs, nil
}

// Close 关闭日志
func Close() {
	logMutex.Lock()
	defer logMutex.Unlock()

	if len(logBuffer) > 0 {
		flushLogsLocked()
	}

	if logFile != nil {
		logFile.Close()
	}
}

