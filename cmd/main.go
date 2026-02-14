package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sns-poster/internal/config"
	"sns-poster/internal/logger"
	"sns-poster/internal/server"
	"sns-poster/internal/xhs"

	"github.com/sirupsen/logrus"
)

func main() {
	// 首先定义和解析所有命令行参数
	var (
		httpPort string
		logFile  string
	)
	flag.StringVar(&httpPort, "http-port", ":6170", "HTTP服务器端口")
	flag.StringVar(&logFile, "log-file", "", "日志文件路径 (留空则输出到控制台)")

	// 立即解析标志，避免与rod的标志冲突
	flag.Parse()

	// 设置全局日志记录器
	if err := logger.SetupGlobalLogger(logFile); err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}

	// 初始化配置（accountID 由各 HTTP 请求 / 消息携带，不在此指定）
	cfg := &config.Config{}

	// 延迟初始化小红书服务，避免rod在flag.Parse()之前注册标志
	xhsService := initializeServices(cfg)

	// 创建HTTP服务器
	httpServer := server.NewHTTPServer(xhsService)

	// 设置信号处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// 启动HTTP服务器
	go func() {
		logrus.Infof("启动HTTP服务器在端口 %s", httpPort)
		if err := httpServer.StartWithoutSignalHandling(httpPort); err != nil {
			logrus.Errorf("HTTP服务器启动失败: %v", err)
		}
	}()

	// 服务器启动后的信息提示
	go func() {
		time.Sleep(2 * time.Second) // 等待服务器完全启动
		logServerStartupInfo()
	}()

	// 等待中断信号
	<-quit
	logrus.Info("收到关闭信号，开始优雅关闭...")

	// 开始优雅关闭
	gracefulShutdown(httpServer, xhsService)
}

// initializeServices 初始化所有服务（在flag.Parse()之后调用）
func initializeServices(cfg *config.Config) *xhs.Service {
	// 初始化小红书服务
	xhsService := xhs.NewService(cfg)
	return xhsService
}

// gracefulShutdown 优雅关闭HTTP服务器
func gracefulShutdown(httpServer *server.HTTPServer, xhsService *xhs.Service) {
	logrus.Info("开始优雅关闭服务器...")

	// 设置较短的关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭HTTP服务器，停止接收新请求
	logrus.Info("正在关闭HTTP服务器...")
	if err := httpServer.Shutdown(ctx); err != nil {
		logrus.Errorf("HTTP服务器关闭失败: %v", err)
	} else {
		logrus.Info("HTTP服务器已成功关闭")
	}

	// XHS服务使用远程浏览器实例，无需关闭浏览器，只需清理连接
	logrus.Info("清理XHS服务连接...")
	xhsService.Close()
	// 注意：不关闭远程浏览器实例，只清理本地连接

	logrus.Info("应用程序已退出")
}

// logServerStartupInfo 显示服务器启动信息
func logServerStartupInfo() {
	logrus.Info("========================================")
	logrus.Info("🚀 SNS Poster HTTP服务已启动")
	logrus.Info("========================================")
	logrus.Info("📡 HTTP API: http://localhost:6170")
}
