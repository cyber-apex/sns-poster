package utils

import (
	"context"
	"time"

	"sns-poster/internal/config"

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

// Close 关闭浏览器连接
func (b *Browser) Close() {
	logrus.Info("断开浏览器连接...")

	// 关闭浏览器实例（对于远程管理器，这只会关闭连接，不会关闭远程浏览器进程）
	if b.Browser != nil {
		b.Browser.MustClose()
	}

	logrus.Info("浏览器连接已断开")
}

// NewBrowser 创建浏览器实例（硬编码配置）
func NewBrowser(cfg *config.Config) *Browser {
	logrus.Info("初始化浏览器管理器...")

	// 硬编码使用管理器模式
	l := launcher.MustNewManaged("")
	// Launch with headful mode
	l.Headless(false).XVFB("--server-num=5", "--server-args=-screen 0 1600x900x16")

	logrus.Info("连接到远程浏览器...")

	// 创建带超时的上下文用于连接
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()

	// 使用通道来处理连接结果
	type result struct {
		browser *rod.Browser
		err     error
	}
	resultChan := make(chan result, 1)

	// 在goroutine中尝试连接
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- result{nil, nil}
			}
		}()

		// 注意：不在浏览器实例上设置超时context，避免后续操作被取消
		browser := rod.New().Client(l.MustClient()).MustConnect()
		resultChan <- result{browser, nil}
	}()

	// 等待连接结果或超时
	select {
	case res := <-resultChan:
		if res.browser != nil {
			logrus.Info("浏览器连接成功")

			// 创建cookie管理器
			cookieManager := NewCookieManager()

			return &Browser{
				Browser:       res.browser,
				launcher:      l,
				cookieManager: cookieManager,
			}
		}
		logrus.Fatal("浏览器连接失败")
		return nil

	case <-connectCtx.Done():
		logrus.Fatal("浏览器连接超时(10秒)，请确保Rod管理器已启动: docker ps | grep rod")
		return nil
	}
}
