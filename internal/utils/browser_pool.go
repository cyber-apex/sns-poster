package utils

import (
	"sync"

	"sns-poster/internal/config"

	"github.com/sirupsen/logrus"
)

// BrowserPool 浏览器池，管理多个浏览器实例（每个账号一个）
type BrowserPool struct {
	browsers map[string]*Browser // accountID -> Browser
	mu       sync.RWMutex
	config   *config.Config
}

// NewBrowserPool 创建浏览器池
func NewBrowserPool(cfg *config.Config) *BrowserPool {
	return &BrowserPool{
		browsers: make(map[string]*Browser),
		config:   cfg,
	}
}

// GetBrowser 获取指定账号的浏览器实例（如果不存在则创建）
func (p *BrowserPool) GetBrowser(accountID string) *Browser {
	// 快速读取检查
	p.mu.RLock()
	browser, exists := p.browsers[accountID]
	p.mu.RUnlock()

	if exists && p.isBrowserConnected(browser) {
		logrus.Debugf("使用已存在的浏览器实例: %s", accountID)
		return browser
	}

	// 需要创建新浏览器或重新连接
	p.mu.Lock()
	defer p.mu.Unlock()

	// 双重检查（防止并发创建）
	if browser, exists := p.browsers[accountID]; exists && p.isBrowserConnected(browser) {
		return browser
	}

	// 如果存在但已断开，先清理
	if browser != nil {
		logrus.Warnf("账号 %s 的浏览器已断开，正在重新创建...", accountID)
		browser.Close()
	}

	// 创建新浏览器实例
	logrus.Infof("为账号 %s 创建新的浏览器实例", accountID)
	browser = NewBrowserWithAccount(p.config, accountID)
	p.browsers[accountID] = browser

	return browser
}

// isBrowserConnected 检查浏览器连接是否有效
func (p *BrowserPool) isBrowserConnected(browser *Browser) bool {
	if browser == nil || browser.Browser == nil {
		return false
	}

	// 尝试获取浏览器信息，如果失败说明连接已断开
	defer func() {
		if r := recover(); r != nil {
			logrus.Debugf("浏览器连接检查失败: %v", r)
		}
	}()

	// 尝试调用一个轻量级的操作来检测连接
	_ = browser.Browser.GetContext()
	return true
}

// CloseBrowser 关闭指定账号的浏览器
func (p *BrowserPool) CloseBrowser(accountID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if browser, exists := p.browsers[accountID]; exists {
		logrus.Infof("关闭账号 %s 的浏览器", accountID)
		browser.Close()
		delete(p.browsers, accountID)
	}
}

// CloseAll 关闭所有浏览器实例
func (p *BrowserPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	logrus.Info("关闭浏览器池中的所有浏览器...")
	for accountID, browser := range p.browsers {
		logrus.Infof("关闭账号 %s 的浏览器", accountID)
		browser.Close()
	}
	p.browsers = make(map[string]*Browser)
	logrus.Info("所有浏览器已关闭")
}

// GetActiveBrowserCount 获取活跃的浏览器数量
func (p *BrowserPool) GetActiveBrowserCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.browsers)
}

// GetActiveAccounts 获取所有活跃账号列表
func (p *BrowserPool) GetActiveAccounts() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	accounts := make([]string, 0, len(p.browsers))
	for accountID := range p.browsers {
		accounts = append(accounts, accountID)
	}
	return accounts
}
