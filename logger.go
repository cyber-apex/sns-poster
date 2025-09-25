package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// LogConfig 日志配置
type LogConfig struct {
	Level      string `json:"level"`       // 日志级别: debug, info, warn, error
	Format     string `json:"format"`      // 日志格式: text, json
	OutputFile string `json:"output_file"` // 输出文件路径
	MaxSize    int64  `json:"max_size"`    // 最大文件大小(MB)
	Console    bool   `json:"console"`     // 是否同时输出到控制台
}

// DefaultLogConfig 默认日志配置
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      "info",
		Format:     "text",
		OutputFile: "", // 默认为空，输出到控制台
		MaxSize:    100,
		Console:    true,
	}
}

// SetupGlobalLogger 设置全局日志记录器
// 如果 logFile 为空，则输出到控制台；否则输出到指定文件
func SetupGlobalLogger(logFile string) error {
	if logFile == "" {
		// 输出到控制台
		logrus.SetOutput(os.Stdout)
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
		logrus.SetLevel(logrus.InfoLevel)
		logrus.Info("日志输出到控制台")
		return nil
	}

	// 确保日志目录存在，使用适合系统守护进程的权限
	dir := filepath.Dir(logFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// 如果无法创建目录，提供有用的错误信息
		if os.IsPermission(err) {
			return fmt.Errorf("创建日志目录失败 %s: %v (提示: 对于/var/logs路径，可能需要sudo权限运行)", dir, err)
		}
		return fmt.Errorf("创建日志目录失败 %s: %v", dir, err)
	}

	// 打开或创建日志文件，使用适合守护进程的权限
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败 %s: %v (提示: 确保目录存在且有写权限)", logFile, err)
	}

	// 设置全局logrus配置
	logrus.SetOutput(file)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	logrus.SetLevel(logrus.InfoLevel)

	logrus.Infof("日志输出到文件: %s", logFile)
	return nil
}
