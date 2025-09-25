package main

import (
	"os"
	"path/filepath"
	"time"

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

// CreateCustomLogger 创建自定义日志记录器实例
// 如果 logFile 为空，则输出到控制台；否则输出到指定文件
func CreateCustomLogger(logFile string) (*logrus.Logger, error) {
	logger := logrus.New()

	// 设置默认格式
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 设置默认级别
	logger.SetLevel(logrus.InfoLevel)

	if logFile == "" {
		// 如果没有指定文件，输出到控制台
		logger.SetOutput(os.Stdout)
		logger.Info("日志输出到控制台")
	} else {
		// 确保日志目录存在
		dir := filepath.Dir(logFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		// 打开或创建日志文件
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}

		logger.SetOutput(file)
		logger.Infof("日志输出到文件: %s", logFile)
	}

	return logger, nil
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

	// 确保日志目录存在
	dir := filepath.Dir(logFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 打开或创建日志文件
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// 设置全局logrus配置
	logrus.SetOutput(file)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logrus.SetLevel(logrus.InfoLevel)

	logrus.Infof("日志输出到文件: %s", logFile)
	return nil
}
