package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sns-poster/internal/xhs"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HTTPServer HTTP服务器
type HTTPServer struct {
	xhsService *xhs.Service
	router     *gin.Engine
	server     *http.Server
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(xhsService *xhs.Service) *HTTPServer {
	return &HTTPServer{
		xhsService: xhsService,
	}
}

// Start 启动服务器（带信号处理）
func (s *HTTPServer) Start(port string) error {
	s.router = s.setupRoutes()

	s.server = &http.Server{
		Addr:    port,
		Handler: s.router,
	}

	// 启动服务器的 goroutine
	go func() {
		logrus.Infof("启动 HTTP 服务器: %s", port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("服务器启动失败: %v", err)
			os.Exit(1)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Infof("正在关闭服务器...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		logrus.Errorf("服务器关闭失败: %v", err)
		return err
	}

	logrus.Infof("服务器已关闭")
	return nil
}

// StartWithoutSignalHandling 启动服务器（不处理信号）
func (s *HTTPServer) StartWithoutSignalHandling(port string) error {
	s.router = s.setupRoutes()

	s.server = &http.Server{
		Addr:    port,
		Handler: s.router,
	}

	logrus.Infof("启动 HTTP 服务器: %s", port)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown 优雅关闭服务器
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// setupRoutes 设置路由
func (s *HTTPServer) setupRoutes() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	// 设置gin使用logrus的输出
	gin.DefaultWriter = logrus.StandardLogger().Out
	gin.DefaultErrorWriter = logrus.StandardLogger().Out

	router := gin.New()

	// 使用自定义的logrus中间件
	router.Use(s.ginLogrusMiddleware())
	router.Use(gin.Recovery())
	router.Use(s.corsMiddleware())

	// 健康检查
	router.GET("/health", s.healthHandler)

	// API 路由组
	api := router.Group("/api/v1")
	{
		// XHS (小红书) 相关路由
		xhs := api.Group("/xhs")
		{
			// 公开路由 - 不需要认证
			xhs.GET("/login/status", s.checkXHSLoginStatusHandler)
			xhs.POST("/login", s.xhsLoginHandler)

			// 受保护的路由 - 自动触发登录
			protected := xhs.Group("/")
			protected.Use(s.xhsAuthMiddleware())
			{
				protected.POST("/publish", s.xhsPublishHandler)
			}
		}
	}

	return router
}

// ginLogrusMiddleware 使用logrus的gin日志中间件
func (s *HTTPServer) ginLogrusMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 记录HTTP请求到logrus
		logrus.WithFields(logrus.Fields{
			"status":     param.StatusCode,
			"method":     param.Method,
			"path":       param.Path,
			"ip":         param.ClientIP,
			"user_agent": param.Request.UserAgent(),
			"latency":    param.Latency,
		}).Info("HTTP请求")

		// 返回空字符串，因为我们已经通过logrus记录了
		return ""
	})
}

// corsMiddleware CORS中间件
func (s *HTTPServer) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details any    `json:"details,omitempty"`
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

// respondError 返回错误响应
func (s *HTTPServer) respondError(c *gin.Context, statusCode int, code, message string, details any) {
	response := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}

	// 记录详细错误信息
	logrus.WithFields(logrus.Fields{
		"method":      c.Request.Method,
		"path":        c.Request.URL.Path,
		"status_code": statusCode,
		"error_code":  code,
		"message":     message,
		"details":     details,
	}).Errorf("API请求失败: %s", message)

	// send notify to wecom regardless of sucess for failure, make sure it executes before exiting the function
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("发送通知失败: %v", r)
			}
		}()
		payload := map[string]string{
			"content": fmt.Sprintf("XHS发布失败: %s\n %s", message, details),
		}
		jsonData, err := json.Marshal(payload)
		if err != nil {
			logrus.Errorf("JSON编码失败: %v", err)
			return
		}

		resp, err := http.Post("http://localhost:6181/api/v1/notify/wecom", "application/json", bytes.NewReader(jsonData))
		if err != nil {
			logrus.Errorf("发送通知失败: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			logrus.Errorf("发送通知失败: %v", resp.StatusCode)
		}
		defer resp.Body.Close()
	}()

	c.JSON(statusCode, response)
}

// respondSuccess 返回成功响应
func (s *HTTPServer) respondSuccess(c *gin.Context, data any, message string) {
	response := SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	}

	logrus.Infof("%s %s %d", c.Request.Method, c.Request.URL.Path, http.StatusOK)
	c.JSON(http.StatusOK, response)
}

// xhsAuthMiddleware XHS认证中间件 - 自动触发登录
func (s *HTTPServer) xhsAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查XHS登录状态
		status, err := s.xhsService.CheckLoginStatus(c.Request.Context())
		if err != nil {
			s.respondError(c, http.StatusInternalServerError, "XHS_AUTH_CHECK_FAILED",
				"无法验证XHS登录状态", err.Error())
			c.Abort()
			return
		}

		if !status.IsLoggedIn {
			logrus.Info("XHS用户未登录，发布器将在需要时处理登录流程")
			// 不在中间件中强制登录，让发布器根据实际情况处理
			// 这样可以确保登录和发布在同一个浏览器会话中进行
		}

		// 将用户信息存储在上下文中
		c.Set("xhs_username", status.Username)
		c.Set("xhs_is_logged_in", status.IsLoggedIn)
		c.Next()
	}
}

// healthHandler 健康检查
func (s *HTTPServer) healthHandler(c *gin.Context) {
	s.respondSuccess(c, map[string]any{
		"status":    "healthy",
		"service":   "sns-poster",
		"timestamp": time.Now().Unix(),
	}, "服务正常")
}

// checkXHSLoginStatusHandler 检查XHS登录状态
func (s *HTTPServer) checkXHSLoginStatusHandler(c *gin.Context) {
	status, err := s.xhsService.CheckLoginStatus(c.Request.Context())
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "XHS_STATUS_CHECK_FAILED",
			"检查XHS登录状态失败", err.Error())
		return
	}

	s.respondSuccess(c, status, "检查XHS登录状态成功")
}

// xhsLoginHandler XHS登录处理
func (s *HTTPServer) xhsLoginHandler(c *gin.Context) {
	result, err := s.xhsService.Login(c.Request.Context())
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "XHS_LOGIN_FAILED",
			"XHS登录失败", err.Error())
		return
	}

	if !result.Success {
		s.respondError(c, http.StatusBadRequest, "XHS_LOGIN_FAILED",
			result.Message, nil)
		return
	}

	s.respondSuccess(c, result, "XHS登录成功")
}

// xhsPublishHandler XHS发布内容
func (s *HTTPServer) xhsPublishHandler(c *gin.Context) {
	var req xhs.PublishContent
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	// 从上下文获取XHS用户信息
	username, _ := c.Get("xhs_username")
	logrus.Infof("XHS用户 %v 请求发布内容: %s", username, req.Title)

	// 执行XHS发布
	result, err := s.xhsService.PublishContent(c.Request.Context(), &req)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "XHS_PUBLISH_FAILED",
			"XHS发布失败", err.Error())
		return
	}

	logrus.Infof("XHS用户 %v 发布内容成功: %s", username, req.Title)
	s.respondSuccess(c, result, "XHS发布成功")
}
