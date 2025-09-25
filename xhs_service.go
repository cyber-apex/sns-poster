package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/sirupsen/logrus"
)

// XHSService 小红书服务
type XHSService struct {
	config *Config
}

// NewXHSService 创建小红书服务
func NewXHSService(config *Config) *XHSService {
	InitConfig(config)
	return &XHSService{
		config: config,
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
func (s *XHSService) CheckLoginStatus(ctx context.Context) (*LoginStatusResponse, error) {
	browser := NewBrowser(s.config)
	defer browser.Close()

	page := browser.NewPage()
	defer page.Close()

	loginAction := NewXHSLogin(page)

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
func (s *XHSService) Login(ctx context.Context) (*LoginResponse, error) {
	browser := NewBrowser(s.config)
	defer browser.Close()

	page := browser.NewPage()
	defer page.Close()

	loginAction := NewXHSLogin(page)

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

// PublishContent 发布内容
func (s *XHSService) PublishContent(ctx context.Context, req *PublishContent) (*PublishResponse, error) {
	// 验证标题长度 - 小红书限制：最大40个单位长度
	// 中文/日文/韩文占2个单位，英文/数字占1个单位
	if titleWidth := runewidth.StringWidth(req.Title); titleWidth > 40 {
		return nil, fmt.Errorf("标题长度超过限制，当前长度: %d，最大允许: 40", titleWidth)
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
func (s *XHSService) processImages(images []string) ([]string, error) {
	processor := NewImageProcessor()
	return processor.ProcessImages(images)
}

// publishContent 执行内容发布
func (s *XHSService) publishContent(ctx context.Context, content PublishContent) error {
	// 为发布操作创建更长的超时上下文（5分钟）
	publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	browser := NewBrowser(s.config)
	defer browser.Close()

	page := browser.NewPage()
	defer page.Close()

	// 确保在发布前加载保存的cookies
	cookieManager := NewCookieManager()
	err := cookieManager.SetCookies(page)
	if err != nil {
		logrus.Warnf("加载cookies失败: %v", err)
	} else {
		logrus.Info("已加载保存的cookies")
	}

	publisher, err := NewXHSPublisher(page)
	if err != nil {
		return fmt.Errorf("创建发布器失败: %w", err)
	}

	// 执行发布
	return publisher.Publish(publishCtx, content)
}
