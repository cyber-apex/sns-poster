package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
)

// PublishContent 发布内容结构
type PublishContent struct {
	Title      string   `json:"title" binding:"required"`
	Content    string   `json:"content" binding:"required"`
	Images     []string `json:"images" binding:"required,min=1"`
	Tags       []string `json:"tags,omitempty"`
	ImagePaths []string `json:"-"` // 处理后的图片路径
}

// XHSPublisher 小红书发布器
type XHSPublisher struct {
	page *rod.Page
}

const (
	publishURL = `https://creator.xiaohongshu.com/publish/publish?source=official`
)

// NewXHSPublisher 创建发布器实例
func NewXHSPublisher(page *rod.Page) (*XHSPublisher, error) {
	// 设置更长的超时时间
	pp := page.Timeout(120 * time.Second)

	slog.Info("开始导航到小红书发布页面", "url", publishURL)

	// 导航到发布页面
	err := pp.Navigate(publishURL)
	if err != nil {
		return nil, fmt.Errorf("导航到发布页面失败: %w", err)
	}

	// 等待页面完全加载
	time.Sleep(3 * time.Second)

	// 检查是否重定向到登录页面
	currentURL := pp.MustInfo().URL
	if strings.Contains(currentURL, "login") {
		slog.Info("检测到登录页面，开始登录流程", "url", currentURL)

		// 在当前浏览器实例中执行登录
		loginHandler := &XHSLogin{page: pp}
		loginErr := loginHandler.Login(context.Background())
		if loginErr != nil {
			return nil, fmt.Errorf("发布时登录失败: %w", loginErr)
		}

		slog.Info("发布时登录成功，重新导航到发布页面")

		// 重新导航到发布页面
		err = pp.Navigate(publishURL)
		if err != nil {
			return nil, fmt.Errorf("登录后重新导航失败: %w", err)
		}

		// 再次等待页面加载
		time.Sleep(3 * time.Second)
	}

	slog.Info("页面加载完成，开始查找上传内容区域")

	// 等待上传内容区域可见，增加更多的选择器尝试
	var uploadElem *rod.Element
	selectors := []string{
		`div.upload-content`,
		`.upload-content`,
		`[class*="upload-content"]`,
		`div[class*="upload"]`,
	}

	for _, selector := range selectors {
		uploadElem, err = pp.Element(selector)
		if err == nil {
			slog.Info("找到上传区域", "selector", selector)
			break
		}
		slog.Debug("选择器未找到元素", "selector", selector, "error", err)
	}

	if uploadElem == nil {
		// 截图用于调试
		screenshot, _ := pp.Screenshot(true, nil)
		if screenshot != nil {
			os.WriteFile("publish_page_debug.png", screenshot, 0644)
			slog.Info("保存调试截图: publish_page_debug.png")
		}
		return nil, fmt.Errorf("找不到上传内容区域，请检查页面是否正确加载")
	}

	err = uploadElem.WaitVisible()
	if err != nil {
		return nil, fmt.Errorf("等待上传内容区域可见失败: %w", err)
	}
	slog.Info("wait for upload-content visible success")

	// 等待一段时间确保页面完全加载
	time.Sleep(2 * time.Second)

	createElems, err := pp.Elements("div.creator-tab")
	if err != nil {
		return nil, fmt.Errorf("查找创作标签失败: %w", err)
	}

	// 过滤掉隐藏的元素
	var visibleElems []*rod.Element
	for _, elem := range createElems {
		if isElementVisible(elem) {
			visibleElems = append(visibleElems, elem)
		}
	}

	if len(visibleElems) == 0 {
		return nil, errors.New("没有找到上传图文元素")
	}

	for _, elem := range visibleElems {
		text, err := elem.Text()
		if err != nil {
			slog.Error("获取元素文本失败", "error", err)
			continue
		}

		if text == "上传图文" {
			if err := elem.Click(proto.InputMouseButtonLeft, 1); err != nil {
				slog.Error("点击元素失败", "error", err)
				continue
			}
			break
		}
	}

	time.Sleep(1 * time.Second)

	return &XHSPublisher{
		page: pp,
	}, nil
}

// Publish 发布内容
func (p *XHSPublisher) Publish(ctx context.Context, content PublishContent) error {
	if len(content.ImagePaths) == 0 {
		return errors.New("图片不能为空")
	}

	page := p.page.Context(ctx)

	if err := p.uploadImages(page, content.ImagePaths); err != nil {
		return errors.Wrap(err, "小红书上传图片失败")
	}

	if err := p.submitPublish(page, content.Title, content.Content, content.Tags); err != nil {
		return errors.Wrap(err, "小红书发布失败")
	}

	return nil
}

func (p *XHSPublisher) uploadImages(page *rod.Page, imagesPaths []string) error {
	pp := page.Timeout(30 * time.Second)

	// 验证文件路径有效性
	for _, path := range imagesPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.Wrapf(err, "图片文件不存在: %s", path)
		}
	}

	// 等待上传输入框出现
	uploadInput, err := pp.Element(".upload-input")
	if err != nil {
		return fmt.Errorf("查找上传输入框失败: %w", err)
	}

	// 上传多个文件
	err = uploadInput.SetFiles(imagesPaths)
	if err != nil {
		return fmt.Errorf("设置上传文件失败: %w", err)
	}

	// 等待并验证上传完成
	return p.waitForUploadComplete(pp, len(imagesPaths))
}

// waitForUploadComplete 等待并验证上传完成
func (p *XHSPublisher) waitForUploadComplete(page *rod.Page, expectedCount int) error {
	maxWaitTime := 60 * time.Second
	checkInterval := 500 * time.Millisecond
	start := time.Now()

	slog.Info("开始等待图片上传完成", "expected_count", expectedCount)

	for time.Since(start) < maxWaitTime {
		// 使用具体的pr类名检查已上传的图片
		uploadedImages, err := page.Elements(".img-preview-area .pr")

		if err == nil {
			currentCount := len(uploadedImages)
			slog.Info("检测到已上传图片", "current_count", currentCount, "expected_count", expectedCount)
			if currentCount >= expectedCount {
				slog.Info("所有图片上传完成", "count", currentCount)
				return nil
			}
		} else {
			slog.Debug("未找到已上传图片元素")
		}

		time.Sleep(checkInterval)
	}

	return errors.New("上传超时，请检查网络连接和图片大小")
}

func (p *XHSPublisher) submitPublish(page *rod.Page, title, content string, tags []string) error {
	titleElem, err := page.Element("div.d-input input")
	if err != nil {
		return fmt.Errorf("查找标题输入框失败: %w", err)
	}
	err = titleElem.Input(title)
	if err != nil {
		return fmt.Errorf("输入标题失败: %w", err)
	}

	time.Sleep(1 * time.Second)

	if contentElem, ok := p.getContentElement(page); ok {
		err = contentElem.Input(content)
		if err != nil {
			return fmt.Errorf("输入内容失败: %w", err)
		}
		p.inputTags(contentElem, tags)
	} else {
		return errors.New("没有找到内容输入框")
	}

	time.Sleep(1 * time.Second)

	submitButton, err := page.Element("div.submit div.d-button-content")
	if err != nil {
		return fmt.Errorf("查找提交按钮失败: %w", err)
	}
	err = submitButton.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return fmt.Errorf("点击提交按钮失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	return nil
}

// 查找内容输入框 - 使用Race方法处理两种样式
func (p *XHSPublisher) getContentElement(page *rod.Page) (*rod.Element, bool) {
	var foundElement *rod.Element
	var found bool

	page.Race().
		Element("div.ql-editor").MustHandle(func(e *rod.Element) {
		foundElement = e
		found = true
	}).
		ElementFunc(func(page *rod.Page) (*rod.Element, error) {
			return p.findTextboxByPlaceholder(page)
		}).MustHandle(func(e *rod.Element) {
		foundElement = e
		found = true
	}).
		MustDo()

	if found {
		return foundElement, true
	}

	slog.Warn("no content element found by any method")
	return nil, false
}

func (p *XHSPublisher) inputTags(contentElem *rod.Element, tags []string) {
	if len(tags) == 0 {
		return
	}

	time.Sleep(1 * time.Second)

	for i := 0; i < 20; i++ {
		contentElem.MustKeyActions().
			Type(input.ArrowDown).
			MustDo()
		time.Sleep(10 * time.Millisecond)
	}

	contentElem.MustKeyActions().
		Press(input.Enter).
		Press(input.Enter).
		MustDo()

	time.Sleep(1 * time.Second)

	for _, tag := range tags {
		tag = strings.TrimLeft(tag, "#")
		p.inputTag(contentElem, tag)
	}
}

func (p *XHSPublisher) inputTag(contentElem *rod.Element, tag string) {
	contentElem.MustInput("#")
	time.Sleep(200 * time.Millisecond)

	for _, char := range tag {
		contentElem.MustInput(string(char))
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)

	page := contentElem.Page()
	topicContainer, err := page.Element("#creator-editor-topic-container")
	if err == nil && topicContainer != nil {
		firstItem, err := topicContainer.Element(".item")
		if err == nil && firstItem != nil {
			firstItem.MustClick()
			slog.Info("成功点击标签联想选项", "tag", tag)
			time.Sleep(200 * time.Millisecond)
		} else {
			slog.Warn("未找到标签联想选项，直接输入空格", "tag", tag)
			contentElem.MustInput(" ")
		}
	} else {
		slog.Warn("未找到标签联想下拉框，直接输入空格", "tag", tag)
		contentElem.MustInput(" ")
	}

	time.Sleep(500 * time.Millisecond)
}

func (p *XHSPublisher) findTextboxByPlaceholder(page *rod.Page) (*rod.Element, error) {
	elements := page.MustElements("p")
	if elements == nil {
		return nil, errors.New("no p elements found")
	}

	// 查找包含指定placeholder的元素
	placeholderElem := p.findPlaceholderElement(elements, "输入正文描述")
	if placeholderElem == nil {
		return nil, errors.New("no placeholder element found")
	}

	// 向上查找textbox父元素
	textboxElem := p.findTextboxParent(placeholderElem)
	if textboxElem == nil {
		return nil, errors.New("no textbox parent found")
	}

	return textboxElem, nil
}

func (p *XHSPublisher) findPlaceholderElement(elements []*rod.Element, searchText string) *rod.Element {
	for _, elem := range elements {
		placeholder, err := elem.Attribute("data-placeholder")
		if err != nil || placeholder == nil {
			continue
		}

		if strings.Contains(*placeholder, searchText) {
			return elem
		}
	}
	return nil
}

func (p *XHSPublisher) findTextboxParent(elem *rod.Element) *rod.Element {
	currentElem := elem
	for i := 0; i < 5; i++ {
		parent, err := currentElem.Parent()
		if err != nil {
			break
		}

		role, err := parent.Attribute("role")
		if err != nil || role == nil {
			currentElem = parent
			continue
		}

		if *role == "textbox" {
			return parent
		}

		currentElem = parent
	}
	return nil
}

// isElementVisible 检查元素是否可见
func isElementVisible(elem *rod.Element) bool {
	// 检查是否有隐藏样式
	style, err := elem.Attribute("style")
	if err == nil && style != nil {
		styleStr := *style

		if strings.Contains(styleStr, "left: -9999px") ||
			strings.Contains(styleStr, "top: -9999px") ||
			strings.Contains(styleStr, "position: absolute; left: -9999px") ||
			strings.Contains(styleStr, "display: none") ||
			strings.Contains(styleStr, "visibility: hidden") {
			return false
		}
	}

	visible, err := elem.Visible()
	if err != nil {
		slog.Warn("无法获取元素可见性", "error", err)
		return true
	}

	return visible
}
