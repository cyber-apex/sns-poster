package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
)

// CookieManager Cookie管理器
type CookieManager struct {
	accountID string
	filePath  string
}

// NewCookieManager 创建Cookie管理器
// accountID: 账号标识，为空则使用默认cookies.json
func NewCookieManager(accountID string) *CookieManager {
	return &CookieManager{
		accountID: accountID,
		filePath:  getCookiesFilePath(accountID),
	}
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
func getCookiesFilePath(accountID string) string {
	// 如果没有指定accountID，使用默认路径（向后兼容）
	if accountID == "" {
		// 检查旧路径是否存在（向后兼容）
		tmpPath := filepath.Join(os.TempDir(), "cookies.json")
		if _, err := os.Stat(tmpPath); err == nil {
			return tmpPath
		}
		// 使用当前目录下的cookies.json
		return "cookies.json"
	}

	// 使用账号特定的cookie文件
	cookiesDir := "cookies"
	return filepath.Join(cookiesDir, fmt.Sprintf("cookies_%s.json", accountID))
}
