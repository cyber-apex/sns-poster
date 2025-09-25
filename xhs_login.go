package main

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// XHSLogin 小红书登录处理
type XHSLogin struct {
	page *rod.Page
}

// NewXHSLogin 创建登录处理实例
func NewXHSLogin(page *rod.Page) *XHSLogin {
	return &XHSLogin{page: page}
}

// CheckLoginStatus 检查登录状态
func (l *XHSLogin) CheckLoginStatus(ctx context.Context) (bool, error) {
	pp := l.page.Context(ctx)

	// Cookie已经在Browser.NewPage()中自动加载

	pp.MustNavigate("https://www.xiaohongshu.com/explore").MustWaitLoad()

	time.Sleep(1 * time.Second)

	exists, _, err := pp.Has(`.main-container .user .link-wrapper .channel`)
	if err != nil {
		return false, errors.Wrap(err, "check login status failed")
	}

	if !exists {
		return false, nil
	}

	return true, nil
}

// Login 登录到小红书
func (l *XHSLogin) Login(ctx context.Context) error {
	pp := l.page.Context(ctx)

	// Cookie已经在Browser.NewPage()中自动加载

	// 导航到小红书首页，这会触发二维码弹窗
	pp.MustNavigate("https://www.xiaohongshu.com/explore").MustWaitLoad()

	// 等待一小段时间让页面完全加载
	time.Sleep(2 * time.Second)

	// 检查是否已经登录
	if exists, _, _ := pp.Has(".main-container .user .link-wrapper .channel"); exists {
		// 已经登录，保存cookies
		cookieManager := NewCookieManager()
		if err := cookieManager.SaveCookies(pp); err != nil {
			logrus.Warnf("保存cookies失败: %v", err)
		}
		return nil
	}

	// 需要登录，寻找登录按钮
	logrus.Info("检测到未登录，开始登录流程...")

	// 尝试点击登录按钮触发二维码
	if err := l.triggerLoginQRCode(pp); err != nil {
		return err
	}

	// 等待并显示二维码
	if err := l.waitAndDisplayQRCode(pp, ctx); err != nil {
		return err
	}

	// 等待登录成功
	if err := l.waitForLoginSuccess(pp, ctx); err != nil {
		return err
	}

	// 保存cookies
	cookieManager := NewCookieManager()
	if err := cookieManager.SaveCookies(pp); err != nil {
		logrus.Warnf("保存cookies失败: %v", err)
	}

	logrus.Info("登录成功！")
	return nil
}

// triggerLoginQRCode 触发二维码显示
func (l *XHSLogin) triggerLoginQRCode(page *rod.Page) error {
	// 首先尝试直接导航到登录页面（更可靠的方法）
	logrus.Info("直接导航到登录页面...")
	page.MustNavigate("https://www.xiaohongshu.com/login").MustWaitLoad()
	time.Sleep(3 * time.Second)

	// 检查是否已经在登录页面上，如果有二维码直接返回
	if qrExists, _, _ := page.Has(".qrcode-img"); qrExists {
		logrus.Info("已在登录页面，发现二维码")
		return nil
	}

	// 查找并点击登录按钮（备用方法）
	loginSelectors := []string{
		".login-btn",
		".sign-btn",
		"[data-testid='login-button']",
		"button[class*='login']",
		"a[href*='login']",
		".header-login",
		// 添加更多可能的登录按钮选择器
		"button[type='button']",
		".btn-login",
		"[role='button']",
	}

	for _, selector := range loginSelectors {
		if elem, err := page.Element(selector); err == nil && elem != nil {
			logrus.Infof("找到登录按钮: %s", selector)
			if err := elem.Click(proto.InputMouseButtonLeft, 1); err != nil {
				logrus.Warnf("点击登录按钮失败: %v", err)
				continue
			}

			// 等待页面变化
			logrus.Info("等待点击后的页面变化...")
			time.Sleep(3 * time.Second)

			// 检查是否出现了二维码
			if qrExists, _, _ := page.Has(".qrcode-img"); qrExists {
				logrus.Info("点击后发现二维码")
				return nil
			}

			// 检查是否有模态框或弹窗
			modalSelectors := []string{
				".modal", ".popup", ".dialog", ".overlay",
				"[role='dialog']", "[role='modal']",
			}

			for _, modalSelector := range modalSelectors {
				if modalExists, _, _ := page.Has(modalSelector); modalExists {
					logrus.Infof("发现模态框: %s", modalSelector)
					time.Sleep(2 * time.Second)
					break
				}
			}

			return nil
		}
	}

	logrus.Warn("未找到登录按钮，但继续尝试查找二维码")
	return nil
}

// waitAndDisplayQRCode 等待并显示二维码
func (l *XHSLogin) waitAndDisplayQRCode(page *rod.Page, ctx context.Context) error {
	qrDisplay := NewQRCodeDisplay()
	房产                      // 设置较小的二维码尺寸，节省终端空间
	qrDisplay.SetSize(4, 1) // 缩小到1/4，每个像素用1个字符

	// 等待二维码出现
	logrus.Info("等待二维码加载...")

	// 更多的二维码选择器，包括小红书常用的
	qrSelectors := []string{
		// 小红书特定的选择器（优先级最高）
		".qrcode-img",
		"img.qrcode-img",
		// 其他常用选择器
		"img[src*='qr']",
		"img[src*='QR']",
		"img[alt*='二维码']",
		"img[alt*='QR']",
		"img[alt*='qrcode']",
		".qrcode img",
		".qr-code img",
		".login-qr img",
		".scan-qr img",
		"canvas",
		"img[src^='data:image']",
		"img[src*='base64']",
		// 其他可能的选择器
		".qr-img",
		".qr-container img",
		".login-scan img",
		"[class*='qr'] img",
		"[class*='QR'] img",
		"[class*='qrcode'] img",
	}

	var qrElement *rod.Element
	var foundSelector string

	for i := 0; i < 30; i++ { // 减少到30秒，避免长时间卡死
		if i == 0 {
			logrus.Info("开始搜索二维码...")
		} else if i%5 == 0 {
			logrus.Infof("仍在等待二维码出现... (%d/30秒)", i)
		}

		for _, selector := range qrSelectors {
			if elem, err := page.Element(selector); err == nil && elem != nil {
				// 检查元素是否可见
				if visible, _ := elem.Visible(); visible {
					logrus.Infof("找到二维码元素: %s", selector)
					qrElement = elem
					foundSelector = selector
					break
				}
			}
		}

		if qrElement != nil {
			break
		}

		// 每10秒输出一次页面信息用于调试
		if i > 0 && i%10 == 0 {
			// 尝试查看页面上有哪些图片元素
			if imgs, err := page.Elements("img"); err == nil {
				logrus.Infof("页面上共找到 %d 个图片元素", len(imgs))

				// 输出前几个图片的类名供调试
				for j, img := range imgs {
					if j >= 3 { // 只看前3个
						break
					}
					if class, err := img.Attribute("class"); err == nil && class != nil {
						logrus.Debugf("图片 %d 的class: %s", j+1, *class)
					}
				}
			}

			// 检查当前页面URL
			currentURL := page.MustInfo().URL
			logrus.Infof("当前页面URL: %s", currentURL)
		}

		time.Sleep(1 * time.Second)
	}

	if qrElement == nil {
		// 最后一次尝试：截取整个页面并保存，供调试用
		screenshot, err := page.Screenshot(true, &proto.PageCaptureScreenshot{})
		if err == nil {
			os.WriteFile("debug_page.png", screenshot, 0644)
			logrus.Info("已保存页面截图到 debug_page.png 供调试")
		}
		return errors.New("未找到二维码元素，请检查登录页面")
	}

	logrus.Infof("成功找到二维码，使用选择器: %s", foundSelector)

	// 调试：记录找到的元素信息
	tagName, _ := qrElement.Eval("el => el.tagName")
	className, _ := qrElement.Eval("el => el.className")
	logrus.Infof("找到的QR元素: 标签=%v, 类名=%v", tagName, className)

	// 获取二维码图片
	src, err := qrElement.Attribute("src")
	if err != nil || src == nil {
		logrus.Info("无法获取二维码src属性，尝试截图方式...")

		// 检查元素是否可见和有尺寸
		box, err := qrElement.Shape()
		if err == nil && len(box.Quads) > 0 {
			logrus.Infof("QR元素位置信息: %+v", box.Quads[0])
		}

		// 尝试截图方式获取二维码
		screenshot, err := qrElement.Screenshot(proto.PageCaptureScreenshotFormat("png"), 90)
		if err != nil {
			return errors.Wrap(err, "failed to capture QR code screenshot")
		}

		// 保存截图到文件
		filename := "qrcode_login.png"
		if err := os.WriteFile(filename, screenshot, 0644); err != nil {
			return errors.Wrap(err, "failed to save QR code screenshot")
		}

		logrus.Infof("二维码截图已保存到: %s", filename)

		// 验证截图大小
		if len(screenshot) < 1000 {
			logrus.Warnf("截图文件过小 (%d bytes)，可能不是有效的二维码", len(screenshot))
		}

		// 将截图转换为data URL格式
		base64Data := base64.StdEncoding.EncodeToString(screenshot)
		dataURL := "data:image/png;base64," + base64Data

		logrus.Infof("二维码截图转换为data URL，大小: %d bytes", len(base64Data))

		// 显示二维码
		if err := qrDisplay.DisplayQRCode(dataURL); err != nil {
			logrus.Warnf("显示二维码失败: %v", err)
			// 回退到基本说明
			qrDisplay.printQRCodeInstructions(dataURL)
		}
	} else {
		logrus.Infof("获取到二维码src: %s", (*src)[:min(100, len(*src))])
		// 显示二维码
		if err := qrDisplay.DisplayQRCode(*src); err != nil {
			logrus.Warnf("显示二维码失败: %v", err)
		}

		// 如果是data URL，也保存到文件
		if strings.HasPrefix(*src, "data:image/") {
			if err := qrDisplay.SaveQRCodeToFile(*src, "qrcode_login.png"); err != nil {
				logrus.Warnf("保存二维码失败: %v", err)
			}
		}
	}

	return nil
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// waitForLoginSuccess 等待登录成功
func (l *XHSLogin) waitForLoginSuccess(page *rod.Page, ctx context.Context) error {
	logrus.Info("等待用户扫码登录...")

	// 等待登录成功的元素出现，最多等待5分钟
	timeout := 300 * time.Second
	checkInterval := 2 * time.Second

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// 检查是否登录成功
		if exists, _, _ := page.Has(".main-container .user .link-wrapper .channel"); exists {
			return nil
		}

		// 检查是否有其他登录成功的标识
		successSelectors := []string{
			".user-info",
			".profile-info",
			"[data-testid='user-avatar']",
			".avatar",
		}

		for _, selector := range successSelectors {
			if exists, _, _ := page.Has(selector); exists {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(checkInterval):
			// 继续等待
		}
	}

	return errors.New("登录超时，请重试")
}
