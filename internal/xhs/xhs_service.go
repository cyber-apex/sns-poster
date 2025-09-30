package xhs

import (
	"context"
	"fmt"
	"time"

	"sns-poster/internal/config"
	"sns-poster/internal/utils"

	"github.com/mattn/go-runewidth"
	"github.com/sirupsen/logrus"
)

// Service 小红书服务
type Service struct {
	config  *config.Config
	browser *utils.Browser
}

// NewService 创建小红书服务
func NewService(cfg *config.Config) *Service {
	config.InitConfig(cfg)
	return &Service{
		config:  cfg,
		browser: utils.NewBrowser(cfg), // 创建持久的浏览器实例
	}
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
	page := s.browser.NewPage()
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
	page := s.browser.NewPage()
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

// Close 关闭服务（不关闭远程浏览器实例）
func (s *Service) Close() {
	// 对于远程浏览器管理器，我们只需要断开连接，不关闭浏览器实例
	// 远程浏览器实例由管理器维护，应该保持运行状态
	if s.browser != nil {
		logrus.Info("断开浏览器连接...")
		s.browser.Close() // 这只是断开连接，不会关闭远程实例
		logrus.Info("XHS服务清理完成，远程浏览器实例保持运行")
	}
}

// PublishContent 发布内容
func (s *Service) PublishContent(ctx context.Context, req *PublishContent) (*PublishResponse, error) {
	// 自动截取标题长度 - 小红书限制：最大40个显示单位
	// CJK字符（中文/日文/韩文）占2个单位，英文/数字/符号占1个单位
	originalWidth := runewidth.StringWidth(req.Title)
	if originalWidth > 40 {
		logrus.Warnf("标题长度超过限制 (%d > 40)，开始智能截取", originalWidth)

		// 使用runewidth进行精确截取，确保不超过40个显示单位
		// 这会正确处理CJK字符的双宽度特性
		truncated := runewidth.Truncate(req.Title, 40, "")
		finalWidth := runewidth.StringWidth(truncated)

		originalRunes := len([]rune(req.Title))
		req.Title = truncated
		truncatedRunes := len([]rune(req.Title))

		logrus.Infof("截取完成: %d字符 -> %d字符 (%d显示单位 -> %d显示单位)",
			originalRunes, truncatedRunes, originalWidth, finalWidth)
		logrus.Infof("截取后的标题: %s", req.Title)
	}

	// 处理图片：下载URL图片或使用本地路径
	imagePaths, err := s.processImages(req.Images)
	if err != nil {
		return nil, err
	}

	// 设置处理后的图片路径
	req.ImagePaths = imagePaths

	// 执行发布
	if err := s.publishContent(ctx, *req); err != nil {
		return nil, err
	}

	response := &PublishResponse{
		Title:   req.Title,
		Content: req.Content,
		Images:  len(imagePaths),
		Status:  "发布完成",
	}

	return response, nil
}

// processImages 处理图片列表，支持URL下载和本地路径
func (s *Service) processImages(images []string) ([]string, error) {
	processor := utils.NewImageProcessor()
	return processor.ProcessImages(images)
}

// publishContent 执行内容发布
func (s *Service) publishContent(ctx context.Context, content PublishContent) error {
	// 为发布操作创建更长的超时上下文（5分钟）
	publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	page := s.browser.NewPage()
	defer page.Close()

	publisher, err := NewPublisher(page)
	if err != nil {
		return fmt.Errorf("创建发布器失败: %w", err)
	}

	// 执行发布
	return publisher.Publish(publishCtx, content)
}
