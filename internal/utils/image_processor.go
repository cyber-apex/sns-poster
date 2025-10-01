package utils

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const downloadDir = "/tmp/xhs-poster"

// ImageProcessor 图片处理器
type ImageProcessor struct{}

// NewImageProcessor 创建图片处理器
func NewImageProcessor() *ImageProcessor {
	// 确保目录存在
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		logrus.Fatalf("创建目录失败: %v", err)
	}
	return &ImageProcessor{}
}

// ProcessImages 处理图片列表（下载URL或使用本地路径）
func (p *ImageProcessor) ProcessImages(images []string) ([]string, error) {
	var paths []string

	for _, image := range images {
		path, err := p.processImage(image)
		if err != nil {
			return nil, errors.Wrapf(err, "处理图片失败: %s", image)
		}
		paths = append(paths, path)
	}

	return paths, nil
}

// processImage 处理单个图片
func (p *ImageProcessor) processImage(image string) (string, error) {
	// 判断是URL还是本地路径
	if strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://") {
		return p.downloadImage(image)
	}

	// 本地文件：验证存在
	if _, err := os.Stat(image); err != nil {
		return "", errors.Errorf("本地图片不存在: %s", image)
	}

	return image, nil
}

// downloadImage 下载URL图片到 /tmp/xhs-poster
func (p *ImageProcessor) downloadImage(imageURL string) (string, error) {
	logrus.Infof("下载图片: %s", imageURL)

	// URL编码处理
	encodedURL := p.encodeURL(imageURL)

	// 创建HTTP请求
	req, err := http.NewRequest("GET", encodedURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// 下载
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "下载失败")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 读取数据
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 生成文件名
	hash := md5.Sum([]byte(imageURL))
	contentType := resp.Header.Get("Content-Type")

	ext := p.getExtension(contentType)
	filename := fmt.Sprintf("img_%x.%s", hash, ext)
	filePath := filepath.Join(downloadDir, filename)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}

	logrus.Infof("图片已保存: %s (%d bytes)", filePath, len(data))
	return filePath, nil
}

// encodeURL 编码URL（处理中文和特殊字符）
func (p *ImageProcessor) encodeURL(rawURL string) string {
	parts := strings.Split(rawURL, "?")
	if len(parts) == 1 {
		return rawURL
	}

	// 重新编码查询参数
	params := url.Values{}
	for _, param := range strings.Split(parts[1], "&") {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) == 2 {
			value, _ := url.QueryUnescape(kv[1])
			params.Set(kv[0], value)
		}
	}

	return parts[0] + "?" + params.Encode()
}

// getExtension 根据Content-Type获取文件扩展名
func (p *ImageProcessor) getExtension(contentType string) string {
	switch {
	case strings.Contains(contentType, "png"):
		return "png"
	case strings.Contains(contentType, "gif"):
		return "gif"
	case strings.Contains(contentType, "webp"):
		return "webp"
	case strings.Contains(contentType, "jpeg"), strings.Contains(contentType, "jpg"):
		return "jpg"
	default:
		return "jpg" // 默认JPG
	}
}
