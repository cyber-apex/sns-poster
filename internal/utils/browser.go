package utils

import (
	"sns-notify/internal/config"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/sirupsen/logrus"
)

// Browser 浏览器实例
type Browser struct {
	*rod.Browser
	launcher      *launcher.Launcher
	cookieManager *CookieManager
}

// NewPage 创建新页面并自动加载cookies
func (b *Browser) NewPage() *rod.Page {
	page := b.Browser.MustPage()

	// 自动加载cookies
	if b.cookieManager != nil {
		b.cookieManager.SetCookies(page)
	}

	return page
}

// Close 关闭浏览器连接（不关闭远程浏览器实例）
func (b *Browser) Close() {
	// 对于远程浏览器管理器，我们只需要断开连接，不关闭浏览器实例
	logrus.Info("断开浏览器连接...")

	// 不调用 b.Browser.Close()，因为这会关闭远程浏览器实例
	// 远程浏览器实例由管理器维护，应该保持运行状态

	// 也不需要清理launcher，因为它管理的是远程实例
	logrus.Info("浏览器连接已断开，远程实例保持运行")
}

// NewBrowser 创建浏览器实例（硬编码配置）
func NewBrowser(cfg *config.Config) *Browser {
	// 硬编码使用管理器模式
	l := launcher.MustNewManaged("")
	// Launch with headful mode
	l.Headless(false).XVFB("--server-num=5", "--server-args=-screen 0 1600x900x16")

	// 启动浏览器并连接
	browser := rod.New().Client(l.MustClient()).MustConnect()

	// 创建cookie管理器
	cookieManager := NewCookieManager()

	return &Browser{
		Browser:       browser,
		launcher:      l,
		cookieManager: cookieManager,
	}
}
