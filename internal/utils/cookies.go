package utils

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
)

// CookieManager Cookie管理器，支持按账号隔离
type CookieManager struct {
	filePath  string
	accountID string
}

// NewCookieManager 创建Cookie管理器（默认账号，向后兼容）
func NewCookieManager() *CookieManager {
	return NewCookieManagerForAccount("")
}

// NewCookieManagerForAccount 创建指定账号的Cookie管理器，多账号时每个账号独立文件
func NewCookieManagerForAccount(accountID string) *CookieManager {
	return &CookieManager{
		filePath:  getCookiesFilePath(accountID),
		accountID: accountID,
	}
}

// AccountID 返回当前管理器对应的账号ID，空表示默认账号
func (c *CookieManager) AccountID() string {
	return c.accountID
}

// LoadCookies 加载Cookies
func (c *CookieManager) LoadCookies() ([]*proto.NetworkCookie, error) {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // 文件不存在是正常的
		}
		return nil, errors.Wrap(err, "failed to read cookies file")
	}

	var cookies []*proto.NetworkCookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal cookies")
	}

	return cookies, nil
}

// SaveCookies 保存Cookies
func (c *CookieManager) SaveCookies(page *rod.Page) error {
	cookies, err := page.Browser().GetCookies()
	if err != nil {
		return errors.Wrap(err, "failed to get cookies from browser")
	}

	data, err := json.Marshal(cookies)
	if err != nil {
		return errors.Wrap(err, "failed to marshal cookies")
	}

	// 确保目录存在
	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, "failed to create cookies directory")
	}

	return os.WriteFile(c.filePath, data, 0644)
}

// ClearCookies 清理浏览器中的 Cookies（当前页面所在浏览器）
func (c *CookieManager) ClearCookies(page *rod.Page) error {
	err := page.Browser().SetCookies(nil)
	if err != nil {
		return errors.Wrap(err, "failed to clear cookies")
	}
	return nil
}

// ClearCookieFile 删除该账号的 cookie 存储文件，登出后下次将无 cookie 可用
func (c *CookieManager) ClearCookieFile() error {
	if err := os.Remove(c.filePath); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to remove cookies file")
	}
	return nil
}

// SetCookies 设置Cookies到浏览器
func (c *CookieManager) SetCookies(page *rod.Page) error {
	cookies, err := c.LoadCookies()
	if err != nil {
		return err
	}

	if len(cookies) == 0 {
		return nil // 没有cookies需要设置
	}

	// 转换为SetCookies需要的格式
	var cookieParams []*proto.NetworkCookieParam
	for _, cookie := range cookies {
		cookieParam := &proto.NetworkCookieParam{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Secure:   cookie.Secure,
			HTTPOnly: cookie.HTTPOnly,
			SameSite: cookie.SameSite,
		}
		if cookie.Expires > 0 {
			cookieParam.Expires = cookie.Expires
		}
		cookieParams = append(cookieParams, cookieParam)
	}

	return page.Browser().SetCookies(cookieParams)
}

// getCookiesFilePath 获取cookies文件路径
// - accountID 为空或未指定时：使用 ./cookies.json（单账号默认路径）
// - accountID 非空时：使用 ./cookies/<accountID>.json（多账号隔离）
func getCookiesFilePath(accountID string) string {
	baseDir := "."
	
	// accountID 为空：使用默认单账号路径 cookies.json
	if accountID == "" {
		// 向后兼容：优先使用旧的 /tmp/cookies.json（如果存在）
		tmpPath := filepath.Join(os.TempDir(), "cookies.json")
		if _, err := os.Stat(tmpPath); err == nil {
			return tmpPath
		}
		// 默认使用当前目录的 cookies.json
		return filepath.Join(baseDir, "cookies.json")
	}
	
	// accountID 非空：使用 cookies/<accountID>.json 实现多账号隔离
	return filepath.Join(baseDir, "cookies", accountID+".json")
}
