package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/h2non/filetype"
	"github.com/pkg/errors"
)

// ImageProcessor 图片处理器
type ImageProcessor struct {
	downloadDir string
}

// NewImageProcessor 创建图片处理器
func NewImageProcessor() *ImageProcessor {
	downloadDir := os.TempDir()
	return &ImageProcessor{
		downloadDir: downloadDir,
	}
}

// ProcessImages 处理图片列表，支持URL下载和本地路径
func (p *ImageProcessor) ProcessImages(images []string) ([]string, error) {
	var imagePaths []string

	for _, image := range images {
		path, err := p.processImage(image)
		if err != nil {
			return nil, errors.Wrapf(err, "处理图片失败: %s", image)
		}
		imagePaths = append(imagePaths, path)
	}

	return imagePaths, nil
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

	return image, nil
}

// isURL 判断是否为URL
func (p *ImageProcessor) isURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}

// downloadImage 下载图片到本地
func (p *ImageProcessor) downloadImage(url string) (string, error) {
	// 生成文件名
	hash := md5.Sum([]byte(url))
	filename := fmt.Sprintf("img_%x", hash)

	// 下载文件
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "下载图片失败")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("下载图片失败，状态码: %d", resp.StatusCode)
	}

	// 读取文件内容
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "读取图片数据失败")
	}

	// 检测文件类型
	kind, err := filetype.Match(data)
	if err != nil {
		return "", errors.Wrap(err, "检测文件类型失败")
	}

	if !filetype.IsImage(data) {
		return "", errors.New("文件不是图片格式")
	}

	// 添加正确的扩展名
	filename = filename + "." + kind.Extension
	filePath := filepath.Join(p.downloadDir, filename)

	// 写入文件
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return "", errors.Wrap(err, "保存图片失败")
	}

	return filePath, nil
}
