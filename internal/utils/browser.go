package utils

import (
	"context"
	"os/exec"
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

// restartRodContainer 重启 Rod Docker 容器并等待其就绪
func restartRodContainer() error {
	logrus.Info("尝试重启 xhs-poster-rod Docker 容器...")

	// 重启 rod 容器
	cmd := exec.Command("docker", "restart", "xhs-poster-rod")
	if err := cmd.Run(); err != nil {
		logrus.Errorf("重启 xhs-poster-rod 容器失败: %v", err)
		return err
	}

	logrus.Info("xhs-poster-rod Docker 容器重启成功，等待容器就绪...")

	// 等待容器启动 (通常需要几秒钟)
	time.Sleep(5 * time.Second)

	logrus.Info("容器应该已就绪")
	return nil
}

// NewPage 创建新页面并自动加载cookies
func (b *Browser) NewPage() *rod.Page {
	// 检查浏览器连接是否有效
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("创建页面失败: %v", r)

			// 尝试重启 Rod 容器
			if err := restartRodContainer(); err != nil {
				logrus.Errorf("重启 Rod 容器失败: %v", err)
			}

			logrus.Panic("浏览器连接已断开，已尝试重启 Rod 容器")
		}
	}()

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
	// l.Headless(false).XVFB("--server-num=5", "--server-args=-screen 0 1600x900x16")

	// 无头模式
	// l.Headless(true)

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

	// 创建cookie管理器（只创建一次，所有分支共享）
	cookieManager := NewCookieManager()

	// 等待连接结果或超时
	select {
	case res := <-resultChan:
		if res.browser != nil {
			logrus.Info("浏览器连接成功")
			return &Browser{
				Browser:       res.browser,
				launcher:      l,
				cookieManager: cookieManager,
			}
		}

		// 初始化阶段连接失败，尝试重启容器并重试一次
		logrus.Warn("初始连接失败，尝试重启 Rod 容器并重试...")
		if err := restartRodContainer(); err != nil {
			logrus.Fatal("重启 Rod 容器失败，无法继续初始化")
		}

		// 重试连接
		logrus.Info("重新尝试连接到远程浏览器...")
		browser := rod.New().Client(l.MustClient())
		if err := browser.Connect(); err != nil {
			logrus.Fatalf("重启后仍无法连接到浏览器: %v", err)
		}

		logrus.Info("重启后浏览器连接成功")
		return &Browser{
			Browser:       browser,
			launcher:      l,
			cookieManager: cookieManager,
		}

	case <-connectCtx.Done():
		// 初始化阶段超时，尝试重启容器
		logrus.Warn("连接超时，尝试重启 Rod 容器...")
		if err := restartRodContainer(); err != nil {
			logrus.Fatal("重启 Rod 容器失败，无法继续初始化")
		}

		// 重试连接（不设置超时，因为刚重启）
		logrus.Info("重新尝试连接到远程浏览器...")
		browser := rod.New().Client(l.MustClient())
		if err := browser.Connect(); err != nil {
			logrus.Fatalf("重启后仍无法连接到浏览器: %v", err)
		}

		logrus.Info("重启后浏览器连接成功")
		return &Browser{
			Browser:       browser,
			launcher:      l,
			cookieManager: cookieManager,
		}
	}
}
