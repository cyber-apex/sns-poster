package main

import (
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

// Close 关闭浏览器
func (b *Browser) Close() {
	// 关闭浏览器连接
	if b.Browser != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logrus.Warnf("关闭浏览器连接时发生panic: %v", r)
				}
			}()
			b.Browser.Close()
		}()
	}

	// 清理启动器
	if b.launcher != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					logrus.Warnf("清理启动器时发生panic: %v", r)
				}
			}()
			b.launcher.Cleanup()
		}()
	}
}

// NewBrowser 创建浏览器实例（硬编码配置）
func NewBrowser(config *Config) *Browser {
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
