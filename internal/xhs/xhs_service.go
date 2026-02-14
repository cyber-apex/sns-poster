package xhs

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"sns-poster/internal/config"
	"sns-poster/internal/utils"

	"github.com/mattn/go-runewidth"
	"github.com/sirupsen/logrus"
)

// Service 小红书服务
type Service struct {
	config     *config.Config
	browser    *utils.Browser
	browserMux sync.Mutex
}

const (
	// 标题最大长度（中文2字符，英文1字符, \n 算2个字符）
	MaxTitleRuneWidth = 38
	// 内容最大长度（中文2字符，英文1字符, \n 算2个字符）
	MaxContentRuneWidth = 1200
)

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

// CheckLoginStatus 检查登录状态，accountID 为空时使用默认单账号
func (s *Service) CheckLoginStatus(ctx context.Context, accountID string) (*LoginStatusResponse, error) {
	page := s.getBrowser().NewPage(accountID)
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

// Login 登录到小红书，accountID 为空时使用默认单账号
func (s *Service) Login(ctx context.Context, accountID string) (*LoginResponse, error) {
	page := s.getBrowser().NewPage(accountID)
	defer page.Close()

	loginAction := NewLogin(page)

	err := loginAction.Login(ctx, accountID)
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

// Logout 登出小红书：删除该账号的 cookie 文件，accountID 为空时使用默认单账号
func (s *Service) Logout(ctx context.Context, accountID string) (*LoginResponse, error) {
	cm := utils.NewCookieManagerForAccount(accountID)
	if err := cm.ClearCookieFile(); err != nil {
		return &LoginResponse{
			Success: false,
			Message: fmt.Sprintf("登出失败: %v", err),
		}, nil
	}
	return &LoginResponse{Success: true, Message: "登出成功"}, nil
}

// Close 关闭服务
func (s *Service) Close() {
	if s.browser != nil {
		s.browser.Close()
		logrus.Info("XHS服务清理完成")
	}
}

// filterSensitiveWordsByRegex 使用正则表达式过滤内容中的敏感词
func (s *Service) filterSensitiveWordsByRegex(content string) string {
	// 过滤内容中的敏感词
	regexPatterns := []string{
		// 正则表达式过滤超链接
		"https?://[^\\s]+",
	}

	re := regexp.MustCompile(strings.Join(regexPatterns, "|"))
	return re.ReplaceAllString(content, "***")
}

// PublishContent 发布内容
func (s *Service) PublishContent(ctx context.Context, req *PublishContent) (*PublishResponse, error) {
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

	// 过滤内容中的敏感词
	req.Content = s.filterSensitiveWordsByRegex(req.Content)

	logrus.Infof("处理图片: %v", req.URL)
	// 处理图片：下载URL图片或使用本地路径
	imagePaths, err := s.processImages(req.Images, req.URL)
	if err != nil {
		return nil, err
	}

	// 设置处理后的图片路径
	req.ImagePaths = imagePaths

	accountID := req.AccountID
	page := s.getBrowser().NewPage(accountID)
	defer page.Close()

	publisher, err := NewPublisher(page, accountID)
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
