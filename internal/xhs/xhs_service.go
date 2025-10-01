package xhs

import (
	"context"
	"fmt"
	"sync"

	"sns-poster/internal/config"
	"sns-poster/internal/utils"

	"github.com/sirupsen/logrus"
)

// Service 小红书服务
type Service struct {
	config     *config.Config
	browser    *utils.Browser
	browserMux sync.Mutex
}

// NewService 创建小红书服务
func NewService(cfg *config.Config) *Service {
	config.InitConfig(cfg)
	return &Service{
		config: cfg,
		// 不在这里创建浏览器，延迟到首次使用
	}
}

// getBrowser 获取或创建浏览器实例（懒加载 + 自动重连）
func (s *Service) getBrowser() *utils.Browser {
	s.browserMux.Lock()
	defer s.browserMux.Unlock()

	// 首次创建或重新连接
	if s.browser == nil {
		logrus.Info("创建新的浏览器连接...")
		s.browser = utils.NewBrowser(s.config)
		return s.browser
	}

	// 检查连接是否有效
	if !s.isBrowserConnected() {
		logrus.Warn("浏览器连接已断开，正在重新连接...")
		s.browser.Close() // 清理旧连接
		s.browser = utils.NewBrowser(s.config)
	}

	return s.browser
}

// isBrowserConnected 检查浏览器连接是否有效
func (s *Service) isBrowserConnected() bool {
	if s.browser == nil || s.browser.Browser == nil {
		return false
	}

	// 尝试获取浏览器信息，如果失败说明连接已断开
	defer func() {
		if r := recover(); r != nil {
			logrus.Debugf("浏览器连接检查失败: %v", r)
		}
	}()

	// 尝试调用一个轻量级的操作来检测连接
	_ = s.browser.Browser.GetContext()
	return true
}

// LoginStatusResponse 登录状态响应
type LoginStatusResponse struct {
	IsLoggedIn bool   `json:"is_logged_in"`
	Username   string `json:"username,omitempty"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// PublishResponse 发布响应
type PublishResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Images  int    `json:"images"`
	Status  string `json:"status"`
}

// CheckLoginStatus 检查登录状态
func (s *Service) CheckLoginStatus(ctx context.Context) (*LoginStatusResponse, error) {
	page := s.getBrowser().NewPage()
	defer page.Close()

	loginAction := NewLogin(page)

	isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
	if err != nil {
		return nil, err
	}

	response := &LoginStatusResponse{
		IsLoggedIn: isLoggedIn,
		Username:   s.config.Username,
	}

	return response, nil
}

// Login 登录到小红书
func (s *Service) Login(ctx context.Context) (*LoginResponse, error) {
	page := s.getBrowser().NewPage()
	defer page.Close()

	loginAction := NewLogin(page)

	err := loginAction.Login(ctx)
	if err != nil {
		return &LoginResponse{
			Success: false,
			Message: fmt.Sprintf("登录失败: %v", err),
		}, nil
	}

	response := &LoginResponse{
		Success: true,
		Message: "登录成功",
	}

	return response, nil
}

// Close 关闭服务
func (s *Service) Close() {
	if s.browser != nil {
		s.browser.Close()
		logrus.Info("XHS服务清理完成")
	}
}

// PublishContent 发布内容
func (s *Service) PublishContent(ctx context.Context, req *PublishContent) (*PublishResponse, error) {
	// 自动截取标题长度 - 小红书限制：最大20个字符
	// 中文、英文、数字都按1个字符计算
	titleRunes := []rune(req.Title)
	originalLength := len(titleRunes)
	if originalLength > 20 {
		logrus.Warnf("标题长度超过限制 (%d > 20)，开始截取", originalLength)

		// 截取前20个字符
		req.Title = string(titleRunes[:20])

		logrus.Infof("截取完成: %d字符 -> %d字符", originalLength, 20)
		logrus.Infof("截取后的标题: %s", req.Title)
	}
	logrus.Infof("处理图片: %v", req.URL)
	// 处理图片：下载URL图片或使用本地路径
	imagePaths, err := s.processImages(req.Images, req.URL)
	if err != nil {
		return nil, err
	}

	// 设置处理后的图片路径
	req.ImagePaths = imagePaths

	page := s.getBrowser().NewPage()
	defer page.Close()

	publisher, err := NewPublisher(page)
	if err != nil {
		return nil, fmt.Errorf("创建发布器失败: %w", err)
	}

	// 执行发布
	return nil, publisher.Publish(ctx, *req)
}

// processImages 处理图片列表，支持URL下载和本地路径
func (s *Service) processImages(images []string, url string) ([]string, error) {
	processor := utils.NewImageProcessor(url)
	return processor.ProcessImages(images)
}
