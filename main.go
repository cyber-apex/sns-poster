package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	if err := SetupGlobalLogger(logFile); err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}

	// 初始化配置
	config := &Config{}

	// 延迟初始化小红书服务，避免rod在flag.Parse()之前注册标志
	xhsService := initializeServices(config)

	// 创建HTTP服务器
	httpServer := NewHTTPServer(xhsService)

	// 设置信号处理
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// 启动HTTP服务器
	go func() {
		log.Printf("启动HTTP服务器在端口 %s", httpPort)
		if err := httpServer.Start(httpPort); err != nil {
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
func initializeServices(config *Config) *XHSService {
	// 初始化小红书服务
	xhsService := NewXHSService(config)
	return xhsService
}

// gracefulShutdown 优雅关闭HTTP服务器
func gracefulShutdown(httpServer *HTTPServer, xhsService *XHSService) {
	logrus.Info("开始优雅关闭服务器...")

	// 设置关闭超时
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 先关闭HTTP服务器，停止接收新请求
	logrus.Info("正在关闭HTTP服务器...")
	if err := httpServer.Shutdown(ctx); err != nil {
		logrus.Errorf("HTTP服务器关闭失败: %v", err)
	} else {
		logrus.Info("HTTP服务器已成功关闭")
	}

	// 再关闭XHS服务和浏览器
	logrus.Info("正在关闭XHS服务...")
	xhsService.Close()

	logrus.Info("应用程序已退出")
}

// logServerStartupInfo 显示服务器启动信息
func logServerStartupInfo() {
	logrus.Info("========================================")
	logrus.Info("🚀 XHS Poster HTTP服务已启动")
	logrus.Info("========================================")
	logrus.Info("📡 HTTP API: http://localhost:6170")
	logrus.Info("🏥 健康检查: http://localhost:6170/health")
	logrus.Info("")
	logrus.Info("📝 API端点:")
	logrus.Info("  • GET  /api/v1/login/status - 检查登录状态")
	logrus.Info("  • POST /api/v1/login - 手动登录")
	logrus.Info("  • POST /api/v1/publish - 发布内容 (需要登录)")
	logrus.Info("")
	logrus.Info("🔐 自动登录:")
	logrus.Info("  访问 /api/v1/publish 将自动触发登录流程")
	logrus.Info("  首次访问时会在终端显示二维码供扫码登录")
	logrus.Info("")
	logrus.Info("🧪 测试脚本:")
	logrus.Info("  ./quick_test_post.sh - 快速测试")
	logrus.Info("========================================")
}
