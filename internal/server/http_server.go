package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sns-poster/internal/xhs"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// HTTPServer HTTP服务器
type HTTPServer struct {
	xhsService  *xhs.Service
	redisClient *redis.Client
	router      *gin.Engine
	server      *http.Server
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(xhsService *xhs.Service, redisClient *redis.Client) *HTTPServer {
	return &HTTPServer{
		xhsService:  xhsService,
		redisClient: redisClient,
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

	// 长任务
	router.GET("/test/long-running-task", s.longRunningTaskHandler)
	router.GET("/test/error-response", s.errorResponseTestHandler)

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
				protected.POST("/logout", s.xhsLogoutHandler)
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
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Account-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// getAccountID 从请求中读取 accountID：优先 Query account_id，其次 Header X-Account-ID
// 如果都没有，返回空字符串（使用默认账号 cookies.json）
func getAccountID(c *gin.Context) string {
	// 优先级 1: Query String (最高优先级)
	if v := c.Query("account_id"); v != "" {
		return v
	}
	// 优先级 2: Header
	if v := c.GetHeader("X-Account-ID"); v != "" {
		return v
	}

	// FIXME: remove this after testing
	// hardcode accountID for testing
	return "65b09837000000000e025e61"
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

	// TODO: delete
	// // send notify to wecom regardless of sucess for failure, make sure it executes before exiting the function
	// go func() {
	// 	defer func() {
	// 		if r := recover(); r != nil {
	// 			logrus.Errorf("发送通知失败: %v", r)
	// 		}
	// 	}()
	// 	payload := map[string]string{
	// 		"content": fmt.Sprintf("XHS发布失败: %s\n %s", message, details),
	// 	}
	// 	jsonData, err := json.Marshal(payload)
	// 	if err != nil {
	// 		logrus.Errorf("JSON编码失败: %v", err)
	// 		return
	// 	}

	// 	resp, err := http.Post("http://localhost:6181/api/v1/notify/wecom", "application/json", bytes.NewReader(jsonData))
	// 	if err != nil {
	// 		logrus.Errorf("发送通知失败: %v", err)
	// 	}
	// 	if resp.StatusCode != http.StatusOK {
	// 		logrus.Errorf("发送通知失败: %v", resp.StatusCode)
	// 	}
	// 	defer resp.Body.Close()
	// }()

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

// xhsAuthMiddleware XHS认证中间件 - 按请求中的 accountID 检查登录状态
// 注意：不在中间件强制登录，让 Publisher 在发布时自动处理登录（同一浏览器会话，cookie 一致）
func (s *HTTPServer) xhsAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Header/Query 读取 accountID（body 还未解析）
		accountID := getAccountID(c)

		logrus.Infof("[Middleware] 检查账号登录状态: %s", accountID)

		status, err := s.xhsService.CheckLoginStatus(c.Request.Context(), accountID)
		if err != nil {
			// 检查失败只记录日志，不阻止请求（Publisher 会自动处理登录）
			logrus.Warnf("[Middleware] 登录状态检查失败: %v，发布器将自动处理", err)
			c.Set("xhs_is_logged_in", false)
			c.Set("xhs_account_id", accountID)
			c.Next()
			return
		}

		if !status.IsLoggedIn {
			logrus.Infof("[Middleware] 账号 %s 未登录，发布器将自动登录", accountID)
		} else {
			logrus.Infof("[Middleware] 账号 %s 已登录", accountID)
		}

		// 保存 middleware 检查的账号信息到 context
		c.Set("xhs_is_logged_in", status.IsLoggedIn)
		c.Set("xhs_account_id", accountID)
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

func (s *HTTPServer) longRunningTaskHandler(c *gin.Context) {
	time.Sleep(20 * time.Second)

	s.respondSuccess(c, map[string]any{
		"status":    "completed",
		"service":   "sns-poster",
		"message":   "长任务完成",
		"timestamp": time.Now().Unix(),
	}, "长任务完成")
}

func (s *HTTPServer) errorResponseTestHandler(c *gin.Context) {
	s.respondError(c, http.StatusInternalServerError, "ERROR_TEST",
		"错误测试", "错误测试详情")
}

// checkXHSLoginStatusHandler 检查XHS登录状态，accountID 通过 Header X-Account-ID 或 Query account_id 传递
func (s *HTTPServer) checkXHSLoginStatusHandler(c *gin.Context) {
	accountID := getAccountID(c)
	status, err := s.xhsService.CheckLoginStatus(c.Request.Context(), accountID)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "XHS_STATUS_CHECK_FAILED",
			"检查XHS登录状态失败", err.Error())
		return
	}

	s.respondSuccess(c, status, "检查XHS登录状态成功")
}

// xhsLoginHandler XHS登录处理，accountID 通过 Header X-Account-ID 或 Query account_id 传递
func (s *HTTPServer) xhsLoginHandler(c *gin.Context) {
	accountID := getAccountID(c)
	logrus.Infof("登录请求，accountID: %s", accountID)

	result, err := s.xhsService.Login(c.Request.Context(), accountID)
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

// xhsLogoutHandler XHS登出处理，accountID 通过 Header X-Account-ID 或 Query account_id 传递
func (s *HTTPServer) xhsLogoutHandler(c *gin.Context) {
	accountID := getAccountID(c)
	result, err := s.xhsService.Logout(c.Request.Context(), accountID)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "XHS_LOGOUT_FAILED",
			"XHS登出失败", err.Error())
		return
	}

	if !result.Success {
		s.respondError(c, http.StatusBadRequest, "XHS_LOGOUT_FAILED",
			result.Message, nil)
		return
	}

	s.respondSuccess(c, nil, "XHS登出成功")
}

// xhsPublishHandler XHS发布内容，accountID 可从 body.account_id 或 Header X-Account-ID 或 Query account_id 传递
func (s *HTTPServer) xhsPublishHandler(c *gin.Context) {
	var req xhs.PublishContent
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	// 优先使用 middleware 已验证的 accountID，确保一致性
	middlewareAccountID, exists := c.Get("xhs_account_id")
	if exists && middlewareAccountID != nil {
		middlewareAccIDStr := middlewareAccountID.(string)
		// 如果 body 有 account_id 且与 middleware 不同，以 body 为准（但需要记录警告）
		if req.AccountID != "" && req.AccountID != middlewareAccIDStr {
			logrus.Warnf("⚠️  账号不一致！Middleware: %s, Body: %s，使用 Body 的值",
				middlewareAccIDStr, req.AccountID)
		} else if req.AccountID == "" {
			// Body 没有 account_id，使用 middleware 的
			req.AccountID = middlewareAccIDStr
		}
	} else if req.AccountID == "" {
		// Middleware 和 Body 都没有，从 Header/Query 读取
		req.AccountID = getAccountID(c)
	}

	logrus.Infof("[Handler] 发布请求 - AccountID: %s, Title: %s", req.AccountID, req.Title)

	// 在redis中检查是否存在该账号的发布记录
	redisKey := fmt.Sprintf("%s:%s:%v", os.Getenv("SNS_POSTER_QUEUE_NAME"), req.AccountID, "success")
	redisValue := req.URL

	// 检查是否在set中存在该账号的发布记录
	exists, err := s.redisClient.SIsMember(c.Request.Context(), redisKey, redisValue).Result()

	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "REDIS_CHECK_FAILED",
			"Redis检查失败", err.Error())
		return
	}
	if exists {
		s.respondError(c, http.StatusBadRequest, "TITLE_ALREADY_PUBLISHED",
			fmt.Sprintf("该标题已存在发布记录: %s", redisValue), nil)
		return
	}

	result, err := s.xhsService.PublishContent(c.Request.Context(), &req)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "XHS_PUBLISH_FAILED",
			"XHS发布失败", err.Error())
		return
	}

	logrus.Infof("[Handler] 发布成功 - AccountID: %s, Title: %s", req.AccountID, req.Title)

	// 将发布记录添加到set中
	s.redisClient.SAdd(c.Request.Context(), redisKey, req.URL)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "REDIS_ADD_FAILED",
			"Redis添加失败", err.Error())
		return
	}

	s.respondSuccess(c, result, "XHS发布成功")
}
