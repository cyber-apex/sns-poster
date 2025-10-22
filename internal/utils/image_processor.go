package utils

import (
	"crypto/md5"
	"encoding/json"
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
type ImageProcessor struct {
	// 爬虫的URL
	url string
}

// NewImageProcessor 创建图片处理器
func NewImageProcessor(url string) *ImageProcessor {
	// 确保目录存在，使用更宽松的权限
	if err := os.MkdirAll(downloadDir, 0777); err != nil {
		logrus.Fatalf("创建目录失败: %v", err)
	}

	// 尝试设置目录权限为所有用户可写
	if err := os.Chmod(downloadDir, 0777); err != nil {
		logrus.Warnf("设置目录权限失败: %v", err)
	}

	return &ImageProcessor{
		url: url,
	}
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
func (p *ImageProcessor) downloadImage(url string) (string, error) {
	imageURL := url

	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Referer":    p.url,
	}

	// 处理 Bandai Hobby CloudFront 图片
	if strings.Contains(imageURL, "/hobby/jp") {
		logrus.Infof("处理 Bandai Hobby CloudFront 图片: %s", imageURL)
		headers["Referer"] = "https://bandai-hobby.net/"
		signedURL, err := p.signBandaiHobbyImage(imageURL)
		if err != nil {
			logrus.Warnf("获取签名URL失败，尝试直接下载: %v", err)
		} else {
			imageURL = signedURL
		}
	}

	logrus.Infof("下载图片: %s", imageURL)

	// 创建HTTP请求
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		logrus.Warnf("创建请求失败: %v", err)
		return "", err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

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

	if err := os.WriteFile(filePath, data, 0666); err != nil {
		// 如果写入失败，尝试创建目录并重试
		if err := os.MkdirAll(filepath.Dir(filePath), 0777); err != nil {
			return "", errors.Wrap(err, "创建目录失败")
		}
		if err := os.WriteFile(filePath, data, 0666); err != nil {
			return "", errors.Wrap(err, "写入文件失败")
		}
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

// signBandaiHobbyImage 为 Bandai Hobby CloudFront 图片生成签名URL
func (p *ImageProcessor) signBandaiHobbyImage(imageURL string) (string, error) {
	// extract path from imageURL
	u, err := url.Parse(imageURL)
	if err != nil {
		return "", errors.Wrap(err, "解析URL失败")
	}
	path := u.Path

	// 调用签名服务
	signURL := fmt.Sprintf("https://assets-signedurl.bandai-hobby.net/get-signed-url?path=%s", path)

	logrus.Infof("请求给Image URL签名: %s", signURL)

	// request application/json
	req, err := http.NewRequest("GET", signURL, nil)
	if err != nil {
		return "", errors.Wrap(err, "创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "请求签名服务失败")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("签名服务返回错误，状态码: %d", resp.StatusCode)
	}

	// 解析JSON响应
	var result struct {
		SignedURL string `json:"signedUrl"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", errors.Wrap(err, "解析签名响应失败")
	}

	if result.SignedURL == "" {
		return "", errors.New("签名URL为空")
	}

	logrus.Infof("获取签名URL成功 %s", result.SignedURL)
	return result.SignedURL, nil
}
