package xhs

import (
	"context"
	"fmt"

	"sns-poster/internal/config"
	"sns-poster/internal/utils"

	"github.com/mattn/go-runewidth"
	"github.com/sirupsen/logrus"
)

// Service 小红书服务
type Service struct {
	config      *config.Config
	browserPool *utils.BrowserPool
}

const (
	MaxTitleRuneWidth   = 38
	MaxContentRuneWidth = 1600
)

// NewService 创建小红书服务
func NewService(cfg *config.Config) *Service {
	config.InitConfig(cfg)
	return &Service{
		config:      cfg,
		browserPool: utils.NewBrowserPool(cfg),
	}
}

// getBrowser 获取指定账号的浏览器实例
func (s *Service) getBrowser(accountID string) *utils.Browser {
	return s.browserPool.GetBrowser(accountID)
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
func (s *Service) CheckLoginStatus(ctx context.Context, accountID string) (*LoginStatusResponse, error) {
	page := s.getBrowser(accountID).NewPage()
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
func (s *Service) Login(ctx context.Context, accountID string) (*LoginResponse, error) {
	page := s.getBrowser(accountID).NewPage()
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
	if s.browserPool != nil {
		s.browserPool.CloseAll()
		logrus.Info("XHS服务清理完成")
	}
}

// CloseBrowser 关闭指定账号的浏览器
func (s *Service) CloseBrowser(accountID string) {
	s.browserPool.CloseBrowser(accountID)
}

// GetActiveBrowserCount 获取活跃的浏览器数量
func (s *Service) GetActiveBrowserCount() int {
	return s.browserPool.GetActiveBrowserCount()
}

// GetActiveAccounts 获取所有活跃账号列表
func (s *Service) GetActiveAccounts() []string {
	return s.browserPool.GetActiveAccounts()
}

// PublishContent 发布内容
func (s *Service) PublishContent(ctx context.Context, accountID string, req *PublishContent) (*PublishResponse, error) {
	// 自动截取标题长度 - 小红书限制：最大40个字符(中文2字符，英文1字符)
	// 使用 runewidth 计算显示宽度（中文2字符，英文1字符）
	originalWidth := runewidth.StringWidth(req.Title)
	if originalWidth > MaxTitleRuneWidth {
		logrus.Warnf("标题长度超过限制 (%d > %d)，开始截取", originalWidth, MaxTitleRuneWidth)

		// 截取到指定宽度
		req.Title = runewidth.Truncate(req.Title, MaxTitleRuneWidth, "")

		logrus.Infof("截取完成: %d字符 -> %d字符", originalWidth, runewidth.StringWidth(req.Title))
		logrus.Infof("截取后的标题: %s", req.Title)
	}

	// 自动截取内容长度 - 小红书限制：最大2000个字符
	// 使用 runewidth 计算显示宽度（中文2字符，英文1字符）
	originalContentWidth := runewidth.StringWidth(req.Content)
	if originalContentWidth > MaxContentRuneWidth {
		logrus.Warnf("内容长度超过限制 (%d > %d)，开始截取", originalContentWidth, MaxContentRuneWidth)
		req.Content = runewidth.Truncate(req.Content, MaxContentRuneWidth, "")

		logrus.Infof("截取完成: %d字符 -> %d字符", originalContentWidth, runewidth.StringWidth(req.Content))
	}

	logrus.Infof("处理图片 [账号: %s]: %v", accountID, req.URL)
	// 处理图片：下载URL图片或使用本地路径
	imagePaths, err := s.processImages(req.Images, req.URL)
	if err != nil {
		return nil, err
	}

	// 设置处理后的图片路径
	req.ImagePaths = imagePaths

	page := s.getBrowser(accountID).NewPage()
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
