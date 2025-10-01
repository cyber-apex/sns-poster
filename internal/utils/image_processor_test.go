package utils

import (
	"os"
	"testing"
)

func TestDownloadCloudFrontImage(t *testing.T) {
	// 测试下载 Bandai Hobby CloudFront 图片
	imageURL := "https://d3bk8pkqsprcvh.cloudfront.net/hobby/jp/product/2025/09/zpBQsAENiJLiq8Pu/S7KlcbpaB2yDBRPO.jpeg"

	processor := NewImageProcessor("https://bandai-hobby.net/item/01_5968/")

	t.Logf("测试URL: %s", imageURL)

	filePath, err := processor.downloadImage(imageURL)
	if err != nil {
		t.Fatalf("下载失败: %v", err)
	}

	// 验证文件存在和大小
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("文件不存在: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Fatalf("文件大小为0")
	}

	t.Logf("✓ 下载成功: %s (%d bytes)", filePath, fileInfo.Size())

	// 清理
	os.Remove(filePath)
}
