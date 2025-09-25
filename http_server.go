package main

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// QRCodeData 二维码数据
type QRCodeData struct {
	DataURL   string    `json:"data_url"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"` // "active", "expired", "used"
	mutex     sync.RWMutex
}

// HTTPServer HTTP服务器
type HTTPServer struct {
	xhsService *XHSService
	router     *gin.Engine
	server     *http.Server
	qrCode     *QRCodeData // 当前二维码数据
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(xhsService *XHSService) *HTTPServer {
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

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(s.corsMiddleware())

	// 健康检查
	router.GET("/health", s.healthHandler)

	// QR 码显示页面 - 用户友好的界面
	router.GET("/qr", s.qrDisplayPageHandler)
	router.GET("/api/qr/current", s.getCurrentQRHandler)
	router.GET("/api/qr/image", s.getQRImageHandler)

	// API 路由组
	api := router.Group("/api/v1")
	{
		// 公开路由 - 不需要认证
		api.GET("/login/status", s.checkLoginStatusHandler)
		api.POST("/login", s.loginHandler) // 保留手动登录选项

		// 受保护的路由 - 自动触发登录
		protected := api.Group("/")
		protected.Use(s.authMiddleware())
		{
			protected.POST("/publish", s.publishHandler)
		}
	}

	return router
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

	logrus.Errorf("%s %s %d", c.Request.Method, c.Request.URL.Path, statusCode)
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

// authMiddleware 认证中间件 - 自动触发登录
func (s *HTTPServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查登录状态
		status, err := s.xhsService.CheckLoginStatus(c.Request.Context())
		if err != nil {
			s.respondError(c, http.StatusInternalServerError, "AUTH_CHECK_FAILED",
				"无法验证登录状态", err.Error())
			c.Abort()
			return
		}

		if !status.IsLoggedIn {
			logrus.Info("用户未登录，发布器将在需要时处理登录流程")
			// 不在中间件中强制登录，让发布器根据实际情况处理
			// 这样可以确保登录和发布在同一个浏览器会话中进行
		}

		// 将用户信息存储在上下文中
		c.Set("username", status.Username)
		c.Set("is_logged_in", status.IsLoggedIn)
		c.Next()
	}
}

// healthHandler 健康检查
func (s *HTTPServer) healthHandler(c *gin.Context) {
	s.respondSuccess(c, map[string]any{
		"status":    "healthy",
		"service":   "xhs-poster",
		"timestamp": time.Now().Unix(),
	}, "服务正常")
}

// checkLoginStatusHandler 检查登录状态
func (s *HTTPServer) checkLoginStatusHandler(c *gin.Context) {
	status, err := s.xhsService.CheckLoginStatus(c.Request.Context())
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "STATUS_CHECK_FAILED",
			"检查登录状态失败", err.Error())
		return
	}

	s.respondSuccess(c, status, "检查登录状态成功")
}

// loginHandler 登录处理
func (s *HTTPServer) loginHandler(c *gin.Context) {
	result, err := s.xhsService.Login(c.Request.Context())
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "LOGIN_FAILED",
			"登录失败", err.Error())
		return
	}

	if !result.Success {
		s.respondError(c, http.StatusBadRequest, "LOGIN_FAILED",
			result.Message, nil)
		return
	}

	s.respondSuccess(c, result, "登录成功")
}

// publishHandler 发布内容
func (s *HTTPServer) publishHandler(c *gin.Context) {
	var req PublishContent
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	// 从上下文获取用户信息
	username, _ := c.Get("username")
	logrus.Infof("用户 %v 请求发布内容: %s", username, req.Title)

	// 执行发布
	result, err := s.xhsService.PublishContent(c.Request.Context(), &req)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, "PUBLISH_FAILED",
			"发布失败", err.Error())
		return
	}

	logrus.Infof("用户 %v 发布内容成功: %s", username, req.Title)
	s.respondSuccess(c, result, "发布成功")
}

// SetQRCode 设置当前二维码数据
func (s *HTTPServer) SetQRCode(dataURL string) {
	if s.qrCode == nil {
		s.qrCode = &QRCodeData{}
	}

	s.qrCode.mutex.Lock()
	defer s.qrCode.mutex.Unlock()

	s.qrCode.DataURL = dataURL
	s.qrCode.Timestamp = time.Now()
	s.qrCode.Status = "active"

	// 自动打开浏览器显示二维码
	go s.openQRInBrowser()
}

// openQRInBrowser 在浏览器中打开二维码页面
func (s *HTTPServer) openQRInBrowser() {
	// 等待一下确保服务器已启动
	time.Sleep(500 * time.Millisecond)

	url := "http://localhost:6170/qr"
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		logrus.Warnf("不支持的操作系统: %s，无法自动打开浏览器", runtime.GOOS)
		logrus.Infof("请手动访问: %s", url)
		return
	}

	if err := cmd.Start(); err != nil {
		logrus.Warnf("无法自动打开浏览器: %v", err)
		logrus.Infof("请手动访问: %s", url)
	} else {
		logrus.Infof("已在浏览器中打开二维码页面: %s", url)
	}
}

// qrDisplayPageHandler 二维码显示页面
func (s *HTTPServer) qrDisplayPageHandler(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>小红书登录 - 扫码登录</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #333;
        }
        
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            padding: 40px;
            text-align: center;
            max-width: 500px;
            width: 90%;
        }
        
        .header {
            margin-bottom: 30px;
        }
        
        .logo {
            font-size: 24px;
            font-weight: bold;
            color: #ff2442;
            margin-bottom: 10px;
        }
        
        .subtitle {
            color: #666;
            font-size: 16px;
        }
        
        .qr-container {
            margin: 30px 0;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 15px;
            border: 2px dashed #ddd;
        }
        
        .qr-image {
            max-width: 280px;
            max-height: 280px;
            border-radius: 10px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.1);
        }
        
        .loading {
            font-size: 18px;
            color: #666;
            margin: 40px 0;
        }
        
        .spinner {
            border: 3px solid #f3f3f3;
            border-top: 3px solid #ff2442;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 20px auto;
        }
        
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        
        .instructions {
            background: #f0f7ff;
            border-left: 4px solid #007bff;
            padding: 20px;
            margin: 20px 0;
            border-radius: 8px;
            text-align: left;
        }
        
        .instructions h3 {
            color: #007bff;
            margin-bottom: 15px;
            font-size: 18px;
        }
        
        .instructions ol {
            margin-left: 20px;
        }
        
        .instructions li {
            margin: 8px 0;
            line-height: 1.5;
        }
        
        .status {
            margin-top: 20px;
            padding: 15px;
            border-radius: 8px;
            font-weight: bold;
        }
        
        .status.waiting {
            background: #fff3cd;
            color: #856404;
            border: 1px solid #ffeaa7;
        }
        
        .status.success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        
        .footer {
            margin-top: 30px;
            color: #888;
            font-size: 14px;
        }
        
        .refresh-btn {
            background: #ff2442;
            color: white;
            border: none;
            padding: 12px 24px;
            border-radius: 25px;
            cursor: pointer;
            font-size: 16px;
            margin: 15px;
            transition: background 0.3s;
        }
        
        .refresh-btn:hover {
            background: #e01e3c;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">📱 小红书</div>
            <div class="subtitle">扫码登录</div>
        </div>
        
        <div class="qr-container">
            <div id="qr-content">
                <div class="loading">
                    <div class="spinner"></div>
                    正在加载二维码...
                </div>
            </div>
        </div>
        
        <div class="instructions">
            <h3>📋 扫码步骤</h3>
            <ol>
                <li>打开小红书手机App</li>
                <li>点击右下角 <strong>「我」</strong></li>
                <li>点击右上角 <strong>扫码图标</strong></li>
                <li>扫描上方二维码</li>
                <li>在手机上确认登录</li>
            </ol>
        </div>
        
        <div id="status" class="status waiting">
            ⏳ 等待扫码登录...
        </div>
        
        <button class="refresh-btn" onclick="refreshQR()">🔄 刷新二维码</button>
        
        <div class="footer">
            XHS Poster - 自动化发布工具
        </div>
    </div>

    <script>
        let statusCheckInterval;
        
        // 加载二维码
        function loadQR() {
            fetch('/api/qr/current')
                .then(response => response.json())
                .then(data => {
                    if (data.success && data.data && data.data.data_url) {
                        const qrContent = document.getElementById('qr-content');
                        qrContent.innerHTML = ` + "`" + `<img src="${data.data.data_url}" alt="登录二维码" class="qr-image">` + "`" + `;
                        
                        // 开始检查登录状态
                        startStatusCheck();
                    } else {
                        document.getElementById('qr-content').innerHTML = 
                            '<div class="loading">❌ 暂无可用的二维码<br><small>请尝试刷新或检查服务状态</small></div>';
                    }
                })
                .catch(error => {
                    console.error('加载二维码失败:', error);
                    document.getElementById('qr-content').innerHTML = 
                        '<div class="loading">❌ 加载失败<br><small>请检查网络连接</small></div>';
                });
        }
        
        // 刷新二维码
        function refreshQR() {
            document.getElementById('qr-content').innerHTML = 
                '<div class="loading"><div class="spinner"></div>正在刷新...</div>';
            document.getElementById('status').className = 'status waiting';
            document.getElementById('status').innerHTML = '⏳ 等待扫码登录...';
            
            // 停止状态检查
            if (statusCheckInterval) {
                clearInterval(statusCheckInterval);
            }
            
            // 重新加载
            setTimeout(loadQR, 1000);
        }
        
        // 检查登录状态
        function checkLoginStatus() {
            fetch('/api/v1/login/status')
                .then(response => response.json())
                .then(data => {
                    if (data.success && data.data && data.data.is_logged_in) {
                        document.getElementById('status').className = 'status success';
                        document.getElementById('status').innerHTML = '✅ 登录成功！';
                        
                        // 停止状态检查
                        if (statusCheckInterval) {
                            clearInterval(statusCheckInterval);
                        }
                        
                        // 3秒后可以关闭页面
                        setTimeout(() => {
                            document.getElementById('status').innerHTML = '✅ 登录成功！您可以关闭此页面了';
                        }, 3000);
                    }
                })
                .catch(error => {
                    console.error('检查登录状态失败:', error);
                });
        }
        
        // 开始状态检查
        function startStatusCheck() {
            // 每2秒检查一次登录状态
            statusCheckInterval = setInterval(checkLoginStatus, 2000);
        }
        
        // 页面加载时自动加载二维码
        window.onload = loadQR;
        
        // 页面关闭时清理定时器
        window.onbeforeunload = function() {
            if (statusCheckInterval) {
                clearInterval(statusCheckInterval);
            }
        };
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// getCurrentQRHandler 获取当前二维码数据
func (s *HTTPServer) getCurrentQRHandler(c *gin.Context) {
	if s.qrCode == nil {
		s.respondError(c, http.StatusNotFound, "QR_NOT_FOUND", "当前没有可用的二维码", nil)
		return
	}

	s.qrCode.mutex.RLock()
	defer s.qrCode.mutex.RUnlock()

	// 检查二维码是否过期（10分钟）
	if time.Since(s.qrCode.Timestamp) > 10*time.Minute {
		s.respondError(c, http.StatusGone, "QR_EXPIRED", "二维码已过期", nil)
		return
	}

	s.respondSuccess(c, s.qrCode, "获取二维码成功")
}

// getQRImageHandler 直接返回二维码图片
func (s *HTTPServer) getQRImageHandler(c *gin.Context) {
	if s.qrCode == nil {
		c.Status(http.StatusNotFound)
		return
	}

	s.qrCode.mutex.RLock()
	defer s.qrCode.mutex.RUnlock()

	if s.qrCode.DataURL == "" {
		c.Status(http.StatusNotFound)
		return
	}

	// 解析data URL
	parts := strings.Split(s.qrCode.DataURL, ",")
	if len(parts) != 2 {
		c.Status(http.StatusInternalServerError)
		return
	}

	// 设置正确的Content-Type
	if strings.Contains(parts[0], "image/png") {
		c.Header("Content-Type", "image/png")
	} else if strings.Contains(parts[0], "image/jpeg") {
		c.Header("Content-Type", "image/jpeg")
	} else {
		c.Header("Content-Type", "image/png") // 默认
	}

	// 直接返回data URL，浏览器会自动处理
	c.String(http.StatusOK, s.qrCode.DataURL)
}
