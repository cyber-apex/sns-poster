package xhs

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// PublishContent 发布内容结构
type PublishContent struct {
	Title      string   `json:"title" binding:"required"`
	Content    string   `json:"content" binding:"required"`
	Images     []string `json:"images" binding:"required,min=1"`
	Tags       []string `json:"tags,omitempty"`
	ImagePaths []string `json:"-"` // 处理后的图片路径
	URL        string   `json:"url,omitempty"`
}

// Publisher 小红书发布器
type Publisher struct {
	page *rod.Page
}

const (
	// 直接进入图片发布模式
	publishURL = `https://creator.xiaohongshu.com/publish/publish?source=official&from=menu&target=image`
)

// debugScreenshot 保存调试截图
func debugScreenshot(page *rod.Page, filename string) error {
	newFilename := fmt.Sprintf("./debug/%s_%d.png", filename, time.Now().Unix())
	screenshot, err := page.Screenshot(true, nil)
	if err != nil {
		return err
	}
	if screenshot != nil {
		err = os.WriteFile(newFilename, screenshot, 0644)
		if err != nil {
			return err
		}
		logrus.Infof("保存调试截图: %s", newFilename)
	}
	return nil
}

// NewPublisher 创建发布器实例
func NewPublisher(page *rod.Page) (*Publisher, error) {
	// 使用独立的context，设置足够长的超时时间
	pp := page.Timeout(300 * time.Second) // 5分钟超时，足够完成发布流程

	logrus.Info("开始导航到小红书发布页面", "url", publishURL)

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
		logrus.Info("检测到登录页面，开始登录流程", "url", currentURL)

		// 在当前浏览器实例中执行登录
		loginHandler := &Login{page: pp}
		loginErr := loginHandler.Login(context.Background())
		if loginErr != nil {
			return nil, fmt.Errorf("发布时登录失败: %w", loginErr)
		}

		logrus.Info("发布时登录成功，重新导航到发布页面")

		// 重新导航到发布页面
		err = pp.Navigate(publishURL)
		if err != nil {
			return nil, fmt.Errorf("登录后重新导航失败: %w", err)
		}

		// 再次等待页面加载
		time.Sleep(3 * time.Second)
	}

	logrus.Info("页面加载完成，开始查找上传内容区域")

	// 等待上传内容区域可见
	uploadElem, err := pp.Element("div.upload-wrapper")
	if err != nil {
		debugScreenshot(pp, "upload_wrapper_not_found.png")
		return nil, fmt.Errorf("找不到上传区域: %w", err)
	}

	err = uploadElem.WaitVisible()
	if err != nil {
		debugScreenshot(pp, "upload_wrapper_not_visible.png")
		return nil, fmt.Errorf("等待上传内容区域可见失败: %w", err)
	}
	logrus.Info("上传区域已可见，发布页面加载成功")

	return &Publisher{
		page: pp,
	}, nil
}

// Publish 发布内容
func (p *Publisher) Publish(ctx context.Context, content PublishContent) error {
	if len(content.ImagePaths) == 0 {
		return errors.New("图片不能为空")
	}

	page := p.page.Context(ctx)

	// 上传图片
	if err := p.uploadImages(page, content.ImagePaths); err != nil {
		return errors.Wrap(err, "小红书上传图片失败")
	}

	// 提交发布
	if err := p.submitPublish(page, content.Title, content.Content, content.Tags); err != nil {
		return errors.Wrap(err, "小红书发布失败")
	}

	return nil
}

func (p *Publisher) uploadImages(page *rod.Page, imagesPaths []string) error {
	logrus.Info("开始上传图片", "count", len(imagesPaths))

	// 验证文件
	for i, path := range imagesPaths {
		stat, err := os.Stat(path)
		if os.IsNotExist(err) {
			return errors.Wrapf(err, "图片文件不存在: %s", path)
		}
		logrus.Info("准备上传", "index", i+1, "path", path, "size_mb", float64(stat.Size())/1024/1024)

		if stat.Size() > 20*1024*1024 {
			return fmt.Errorf("图片过大: %.2fMB > 20MB", float64(stat.Size())/1024/1024)
		}
	}

	// 查找文件输入框，设置超时
	logrus.Info("查找文件上传输入框...")

	uploadInput, err := page.Timeout(10 * time.Second).Element("div.upload-wrapper input.upload-input[type='file']")
	if err != nil {
		// 截图调试
		debugScreenshot(page, "upload_input_not_found.png")
		return fmt.Errorf("未找到文件上传输入框: %w", err)
	}
	logrus.Info("找到文件上传输入框, 开始上传图片")

	// 上传文件
	err = uploadInput.SetFiles(imagesPaths)
	if err != nil {
		debugScreenshot(page, "upload_file_failed.png")
		return fmt.Errorf("上传文件失败: %w", err)
	}

	logrus.Info("文件已上传，等待处理...")
	time.Sleep(3 * time.Second)

	// 简单验证上传完成
	return p.waitForUploadComplete(page, len(imagesPaths))
}

// waitForUploadComplete 等待并验证上传完成
func (p *Publisher) waitForUploadComplete(page *rod.Page, expectedCount int) error {
	maxWaitTime := 60 * time.Second
	checkInterval := 500 * time.Millisecond
	start := time.Now()

	for time.Since(start) < maxWaitTime {
		// 使用具体的pr类名检查已上传的图片
		uploadedImages, err := page.Elements(".img-preview-area .pr")

		if err == nil {
			currentCount := len(uploadedImages)
			logrus.Info("检测到已上传图片", "current_count", currentCount, "expected_count", expectedCount)
			if currentCount >= expectedCount {
				logrus.Info("所有图片上传完成", "count", currentCount)
				return nil
			}
		} else {
			debugScreenshot(page, "upload_indicators_not_found.png")
			logrus.Debug("未找到已上传图片元素")
		}

		time.Sleep(checkInterval)
	}

	return errors.New("上传超时，请检查网络连接和图片大小")
}

func (p *Publisher) submitPublish(page *rod.Page, title, content string, tags []string) error {
	titleElem, err := page.Element("div.d-input input.d-text")
	if err != nil {
		debugScreenshot(page, "title_input_not_found.png")
		return fmt.Errorf("查找标题输入框失败: %w", err)
	}
	err = titleElem.Input(title)
	if err != nil {
		return fmt.Errorf("输入标题失败: %w", err)
	}

	time.Sleep(1 * time.Second)

	contentElem, err := page.Element("div.edit-container div[contenteditable='true']")
	if err != nil {
		debugScreenshot(page, "content_input_not_found.png")
		return fmt.Errorf("查找内容输入框失败: %w", err)
	}

	err = contentElem.Input(content)
	if err != nil {
		return fmt.Errorf("输入内容失败: %w", err)
	}

	p.inputTags(contentElem, tags)

	time.Sleep(1 * time.Second)

	submitButton, err := page.Element("div.submit button.d-button")

	if err != nil {
		debugScreenshot(page, "submit_button_not_found.png")
		return fmt.Errorf("查找提交按钮失败: %w", err)
	}
	err = submitButton.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		debugScreenshot(page, "submit_button_click_failed.png")
		return fmt.Errorf("点击提交按钮失败: %w", err)
	}

	time.Sleep(3 * time.Second)

	return nil
}

func (p *Publisher) inputTags(contentElem *rod.Element, tags []string) {
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

func (p *Publisher) inputTag(contentElem *rod.Element, tag string) {
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
			logrus.Info("成功点击标签联想选项", "tag", tag)
			time.Sleep(200 * time.Millisecond)
		} else {
			logrus.Warn("未找到标签联想选项，直接输入空格", "tag", tag)
			contentElem.MustInput(" ")
		}
	} else {
		logrus.Warn("未找到标签联想下拉框，直接输入空格", "tag", tag)
		contentElem.MustInput(" ")
	}

	time.Sleep(500 * time.Millisecond)
}
