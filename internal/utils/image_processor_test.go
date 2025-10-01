package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImageProcessor_DownloadImage(t *testing.T) {
	processor := NewImageProcessor()

	tests := []struct {
		name        string
		imageURL    string
		expectError bool
	}{
		{
			name:        "普通英文URL",
			imageURL:    "https://placehold.co/600x400/EEE/31343C?font=poppins&text=Poppins",
			expectError: false,
		},
		{
			name:        "包含中文的URL",
			imageURL:    "https://placehold.co/600x400/FF6B6B/FFFFFF?text=测试图片",
			expectError: false,
		},
		{
			name:        "包含特殊字符的URL",
			imageURL:    "https://placehold.co/600x400/4ECDC4/FFFFFF?font=opensans&text=OpenSans%20Test",
			expectError: false,
		},
		{
			name:        "PNG格式",
			imageURL:    "https://placehold.co/400x300.png",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := processor.downloadImage(tt.imageURL)

			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误，但成功了")
				}
				return
			}

			if err != nil {
				t.Errorf("下载失败: %v", err)
				return
			}

			// 验证文件存在
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("文件不存在: %s", path)
				return
			}

			// 验证文件在正确的目录
			if !strings.HasPrefix(path, downloadDir) {
				t.Errorf("文件路径错误: %s, 应该在 %s", path, downloadDir)
			}

			// 验证文件大小
			stat, _ := os.Stat(path)
			if stat.Size() == 0 {
				t.Errorf("文件大小为0: %s", path)
			}

			t.Logf("✓ 下载成功: %s (%.2f KB)", path, float64(stat.Size())/1024)
		})
	}
}

func TestImageProcessor_ProcessImages(t *testing.T) {
	processor := NewImageProcessor()

	// 测试混合输入（URL + 本地文件）
	tests := []struct {
		name        string
		images      []string
		expectError bool
	}{
		{
			name: "单个URL",
			images: []string{
				"https://placehold.co/200x200/000000/FFFFFF?text=Test1",
			},
			expectError: false,
		},
		{
			name: "多个URL",
			images: []string{
				"https://placehold.co/200x200/FF0000/FFFFFF?text=Red",
				"https://placehold.co/200x200/00FF00/FFFFFF?text=Green",
			},
			expectError: false,
		},
		{
			name: "不存在的本地文件",
			images: []string{
				"/tmp/non_existent_file.jpg",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, err := processor.ProcessImages(tt.images)

			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误，但成功了")
				}
				return
			}

			if err != nil {
				t.Errorf("处理失败: %v", err)
				return
			}

			if len(paths) != len(tt.images) {
				t.Errorf("路径数量不匹配: 期望 %d, 实际 %d", len(tt.images), len(paths))
			}

			for i, path := range paths {
				t.Logf("  [%d] %s", i+1, path)
			}
		})
	}
}

func TestImageProcessor_GetExtension(t *testing.T) {
	processor := NewImageProcessor()

	tests := []struct {
		contentType string
		expected    string
	}{
		{"image/png", "png"},
		{"image/jpeg", "jpg"},
		{"image/jpg", "jpg"},
		{"image/gif", "gif"},
		{"image/webp", "webp"},
		{"image/unknown", "jpg"}, // 默认
		{"", "jpg"},              // 空值默认
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := processor.getExtension(tt.contentType)
			if result != tt.expected {
				t.Errorf("期望 %s, 实际 %s", tt.expected, result)
			}
		})
	}
}

func TestImageProcessor_EncodeURL(t *testing.T) {
	processor := NewImageProcessor()

	tests := []struct {
		name     string
		input    string
		contains []string // 检查结果是否包含这些字符串
	}{
		{
			name:  "普通URL",
			input: "https://example.com/image.jpg",
			contains: []string{
				"https://example.com/image.jpg",
			},
		},
		{
			name:  "包含中文",
			input: "https://example.com/image.jpg?text=测试",
			contains: []string{
				"https://example.com/image.jpg?",
				"text=",
			},
		},
		{
			name:  "包含空格",
			input: "https://example.com/image.jpg?text=Hello World",
			contains: []string{
				"https://example.com/image.jpg?",
				"text=",
			},
		},
		{
			name:  "多个参数",
			input: "https://example.com/image.jpg?font=poppins&text=Test&size=large",
			contains: []string{
				"font=poppins",
				"text=Test",
				"size=large",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.encodeURL(tt.input)

			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("结果应包含 '%s', 实际: %s", s, result)
				}
			}

			t.Logf("输入: %s", tt.input)
			t.Logf("输出: %s", result)
		})
	}
}

// 清理测试文件
func TestCleanup(t *testing.T) {
	t.Cleanup(func() {
		// 可选：清理测试生成的文件
		// 注意：这会删除 /tmp/xhs-poster 中的所有测试文件
		t.Log("测试完成，文件保留在 /tmp/xhs-poster 供检查")
	})
}

// Benchmark: 测试下载性能
func BenchmarkImageProcessor_DownloadImage(b *testing.B) {
	processor := NewImageProcessor()
	imageURL := "https://placehold.co/100x100/000000/FFFFFF?text=Benchmark"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.downloadImage(imageURL)
		if err != nil {
			b.Fatalf("下载失败: %v", err)
		}
	}
}

// 辅助函数：创建测试用的临时图片文件
func createTestImage(t *testing.T) string {
	t.Helper()

	tmpFile := filepath.Join(os.TempDir(), "test_image.jpg")
	content := []byte("fake image content")

	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	t.Cleanup(func() {
		os.Remove(tmpFile)
	})

	return tmpFile
}

func TestImageProcessor_ProcessLocalFile(t *testing.T) {
	processor := NewImageProcessor()

	// 创建临时测试文件
	testFile := createTestImage(t)

	paths, err := processor.ProcessImages([]string{testFile})
	if err != nil {
		t.Fatalf("处理本地文件失败: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("期望1个路径，实际 %d", len(paths))
	}

	// 本地文件应该直接返回原路径
	if paths[0] != testFile {
		t.Errorf("期望路径 %s, 实际 %s", testFile, paths[0])
	}

	t.Logf("✓ 本地文件处理成功: %s", paths[0])
}
