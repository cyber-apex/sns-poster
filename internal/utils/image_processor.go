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

// ImageProcessor 图片处理器
type ImageProcessor struct {
	downloadDir string
}

// NewImageProcessor 创建图片处理器
func NewImageProcessor() *ImageProcessor {
	// 使用 /tmp/xhs-poster 目录，这个目录会被挂载到Docker容器
	downloadDir := "/tmp/xhs-poster"

	// 确保目录存在
	err := os.MkdirAll(downloadDir, 0755)
	if err != nil {
		logrus.Warnf("创建临时目录失败: %v", err)
		// 如果创建失败，回退到当前目录
		workDir, _ := os.Getwd()
		downloadDir = filepath.Join(workDir, "images")
		os.MkdirAll(downloadDir, 0755)
	}

	logrus.Infof("图片处理目录: %s", downloadDir)

	return &ImageProcessor{
		downloadDir: downloadDir,
	}
}

// ProcessImages 处理图片列表，支持URL下载和本地路径
func (p *ImageProcessor) ProcessImages(images []string) ([]string, error) {
	var imagePaths []string

	logrus.Info("使用Docker共享目录方案，处理图片到临时目录")

	for _, image := range images {
		path, err := p.processImage(image)
		if err != nil {
			return nil, errors.Wrapf(err, "处理图片失败: %s", image)
		}

		// 转换为Docker容器内的路径
		containerPath := p.convertToContainerPath(path)
		imagePaths = append(imagePaths, containerPath)
		logrus.Infof("图片处理完成: %s -> %s (容器内路径: %s)", image, path, containerPath)
	}

	return imagePaths, nil
}

// convertToContainerPath 将宿主机路径转换为容器内路径
func (p *ImageProcessor) convertToContainerPath(hostPath string) string {
	// 如果是 /tmp/xhs-poster 路径，转换为容器内的 /tmp/xhs-poster
	// 由于挂载时两边路径相同，直接返回原路径
	if strings.HasPrefix(hostPath, "/tmp/xhs-poster") {
		return hostPath
	}

	// 如果是其他路径，尝试复制到临时目录
	if !strings.HasPrefix(hostPath, p.downloadDir) {
		return p.copyToTempDir(hostPath)
	}

	return hostPath
}

// copyToTempDir 复制文件到临时目录
func (p *ImageProcessor) copyToTempDir(srcPath string) string {
	// 生成目标文件名
	fileName := filepath.Base(srcPath)
	dstPath := filepath.Join(p.downloadDir, fileName)

	// 如果文件已存在，添加时间戳避免冲突
	if _, err := os.Stat(dstPath); err == nil {
		ext := filepath.Ext(fileName)
		name := strings.TrimSuffix(fileName, ext)
		dstPath = filepath.Join(p.downloadDir, fmt.Sprintf("%s_%d%s", name, time.Now().Unix(), ext))
	}

	// 复制文件
	if err := p.copyFile(srcPath, dstPath); err != nil {
		logrus.Warnf("复制文件到临时目录失败: %v", err)
		return srcPath // 返回原路径
	}

	return dstPath
}

// copyFile 复制文件
func (p *ImageProcessor) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// processImage 处理单个图片
func (p *ImageProcessor) processImage(image string) (string, error) {
	// 判断是URL还是本地路径
	if p.isURL(image) {
		return p.downloadImage(image)
	}

	// 验证本地路径是否存在
	if _, err := os.Stat(image); os.IsNotExist(err) {
		return "", errors.Errorf("本地图片文件不存在: %s", image)
	}

	// 如果是绝对路径或当前目录外的路径，复制到工作目录
	// 这样确保远程浏览器可以访问文件
	if filepath.IsAbs(image) || strings.Contains(image, "..") || !strings.HasPrefix(image, "./") {
		return p.copyImageToWorkDir(image)
	}

	return image, nil
}

// copyImageToWorkDir 复制图片到工作目录
func (p *ImageProcessor) copyImageToWorkDir(imagePath string) (string, error) {
	// 读取原文件
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", errors.Wrap(err, "读取图片文件失败")
	}

	// 生成新文件名
	originalName := filepath.Base(imagePath)
	hash := fmt.Sprintf("%x", md5.Sum(data))
	ext := filepath.Ext(originalName)
	newName := fmt.Sprintf("copy_%s_%s%s", hash[:8], strings.TrimSuffix(originalName, ext), ext)

	newPath := filepath.Join(p.downloadDir, newName)

	// 写入到工作目录
	err = os.WriteFile(newPath, data, 0644)
	if err != nil {
		return "", errors.Wrap(err, "复制图片到工作目录失败")
	}

	return newPath, nil
}

// isURL 判断是否为URL
func (p *ImageProcessor) isURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}

// downloadImage 下载图片到本地
func (p *ImageProcessor) downloadImage(imageURL string) (string, error) {
	// 分离URL和查询参数
	urlParts := strings.Split(imageURL, "?")
	baseURL := urlParts[0]

	var encodedURL string
	if len(urlParts) > 1 {
		// 解析查询参数
		params := url.Values{}
		queryParts := strings.Split(urlParts[1], "&")

		for _, part := range queryParts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				// URL解码原值（如果已编码），然后重新编码
				key := kv[0]
				value, _ := url.QueryUnescape(kv[1])
				params.Set(key, value)
			}
		}

		encodedURL = baseURL + "?" + params.Encode()
	} else {
		encodedURL = baseURL
	}

	logrus.Infof("编码后URL: %s", encodedURL)

	// 生成文件名
	hash := md5.Sum([]byte(imageURL))
	filename := fmt.Sprintf("img_%x", hash)

	// 创建HTTP请求
	req, err := http.NewRequest("GET", encodedURL, nil)
	if err != nil {
		return "", errors.Wrap(err, "创建下载请求失败")
	}

	// 设置User-Agent避免被拦截
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// 下载文件
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "下载图片失败")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("下载图片失败，状态码: %d，URL: %s", resp.StatusCode, encodedURL)
	}

	// 读取文件内容
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "读取图片数据失败")
	}

	logrus.Infof("下载成功，大小: %d bytes", len(data))

	// 简单根据 Content-Type 确定扩展名，默认 jpg
	contentType := resp.Header.Get("Content-Type")
	ext := "jpg"
	if strings.Contains(contentType, "png") {
		ext = "png"
	} else if strings.Contains(contentType, "gif") {
		ext = "gif"
	} else if strings.Contains(contentType, "webp") {
		ext = "webp"
	}

	// 保存文件
	filename = filename + "." + ext
	filePath := filepath.Join(p.downloadDir, filename)

	// 写入文件
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return "", errors.Wrap(err, "保存图片失败")
	}

	return filePath, nil
}
