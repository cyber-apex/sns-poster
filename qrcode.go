package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"os"
	"strings"

	"github.com/mdp/qrterminal/v3"
	"github.com/sirupsen/logrus"
)

// QRCodeDisplay 二维码显示器
type QRCodeDisplay struct {
	Scale     int // 图像缩放因子 (1=原始大小, 2=缩小一半)
	CharScale int // 字符放大因子 (每个像素用几个字符表示)
}

// NewQRCodeDisplay 创建二维码显示器
func NewQRCodeDisplay() *QRCodeDisplay {
	return &QRCodeDisplay{
		Scale:     1, // 默认原始大小
		CharScale: 2, // 默认每个像素用2个字符，提高可扫描性
	}
}

// SetSize 设置二维码显示大小
func (q *QRCodeDisplay) SetSize(scale, charScale int) {
	if scale > 0 {
		q.Scale = scale
	}
	if charScale > 0 {
		q.CharScale = charScale
	}
}

// DisplayQRCode 在终端显示二维码
func (q *QRCodeDisplay) DisplayQRCode(dataURL string) error {
	// 提取base64数据
	if !strings.HasPrefix(dataURL, "data:image/") {
		return fmt.Errorf("invalid data URL format")
	}

	// 分离MIME类型和base64数据
	parts := strings.Split(dataURL, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid data URL format")
	}

	base64Data := parts[1]
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("failed to decode base64 data: %v", err)
	}

	logrus.Infof("二维码数据大小: %d bytes", len(imageData))

	// 在日志中显示二维码图像信息
	q.printQRCodeImageInLog(dataURL)

	// 显示原始小红书二维码的ASCII版本
	err = q.printQRCodeASCII(imageData)
	if err != nil {
		logrus.Warnf("无法显示原始二维码ASCII版本: %v", err)
	}

	// 同时显示一个备用的提示QR码
	q.displayBackupQRCodeWithQRTerminal()

	return nil
}

// printQRCodeImageInLog 在日志中显示二维码图像信息
func (q *QRCodeDisplay) printQRCodeImageInLog(dataURL string) {
	logrus.Info("========================================")
	logrus.Info("🔍 小红书登录二维码图像")
	logrus.Info("========================================")

	// 记录完整的数据URL，可以直接在浏览器中打开
	logrus.Infof("📷 二维码图像数据URL (复制到浏览器地址栏查看):")
	logrus.Infof("%s", dataURL)

	// 生成一个可点击的HTML链接
	htmlLink := fmt.Sprintf("data:text/html,<html><body style='display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:white;'><img src='%s' style='width:300px;height:300px;border:1px solid #ccc;'/></body></html>", dataURL)
	logrus.Infof("🌐 HTML查看链接 (复制到浏览器地址栏):")
	logrus.Infof("%s", htmlLink)

	logrus.Info("📱 使用方法:")
	logrus.Info("   1. 复制上面的数据URL到浏览器地址栏")
	logrus.Info("   2. 或者复制HTML链接到浏览器查看大图")
	logrus.Info("   3. 使用小红书APP扫描二维码")
	logrus.Info("========================================")
}

// displayQRCodeWithQRTerminal 使用qrterminal库生成高质量二维码
func (q *QRCodeDisplay) displayQRCodeWithQRTerminal(dataURL string) {
	logrus.Info("========================================")
	logrus.Info("🔍 小红书登录二维码 - 高质量显示")
	logrus.Info("========================================")

	// 创建一个字符串缓冲区来捕获qrterminal的输出
	var buf strings.Builder

	// 配置qrterminal - 使用半块模式获得最佳分辨率
	config := qrterminal.Config{
		HalfBlocks: true,         // 使用半块字符获得更好分辨率
		Level:      qrterminal.L, // 使用低错误纠正以减少数据量
		Writer:     &buf,
		BlackChar:  qrterminal.BLACK,
		WhiteChar:  qrterminal.WHITE,
		QuietZone:  1, // 减少边距以节省空间
	}

	// 数据URL通常太长无法生成QR码，我们生成一个简化的提示信息
	displayText := "请查看上方日志中的数据URL链接，复制到浏览器查看二维码"

	// 使用defer和recover来处理可能的panic
	defer func() {
		if r := recover(); r != nil {
			logrus.Warnf("qrterminal生成失败: %v", r)
			logrus.Info("由于数据过长，无法生成QR码")
			logrus.Info("请使用上方的数据URL链接在浏览器中查看二维码")
		}
	}()

	// 生成QR码
	qrterminal.GenerateWithConfig(displayText, config)

	// 将生成的QR码通过日志输出
	if buf.Len() > 0 {
		qrLines := strings.Split(buf.String(), "\n")
		for _, line := range qrLines {
			if strings.TrimSpace(line) != "" {
				logrus.Info(line)
			}
		}
	}

	logrus.Info("========================================")
	logrus.Info("📱 请使用小红书APP扫描上方二维码登录")
	logrus.Info("💾 原始二维码图片已保存到: qrcode_login.png")
	logrus.Info("🌐 或复制上面的数据URL到浏览器查看原始二维码")
	logrus.Info("========================================")
}

// printQRCodeASCII 将二维码图像转换为ASCII艺术并打印
func (q *QRCodeDisplay) printQRCodeASCII(imageData []byte) error {
	// 解码图像
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return fmt.Errorf("failed to decode image: %v", err)
	}

	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// 使用配置的缩放参数
	scale := q.Scale
	charScale := q.CharScale

	// 如果未设置，使用智能默认值
	if scale == 0 {
		scale = 1
		if width > 200 || height > 200 {
			scale = 2 // 大图像时适当缩小
		}
	}
	if charScale == 0 {
		charScale = 2 // 默认放大2倍
	}

	logrus.Info("========================================")
	logrus.Infof("🔍 小红书登录二维码 (%dx%d -> 缩放:%d 字符放大:%d)", width, height, scale, charScale)
	logrus.Info("========================================")

	// 添加顶部边距
	topMargin := strings.Repeat(" ", (width/scale)*charScale+8)
	for i := 0; i < 2; i++ {
		logrus.Info(topMargin)
	}

	// 使用半块字符获得更好的分辨率
	for y := bounds.Min.Y; y < bounds.Max.Y; y += scale * 2 {
		line := "    " // 左边距
		for x := bounds.Min.X; x < bounds.Max.X; x += scale {
			// 获取上半部分像素
			r1, g1, b1, _ := img.At(x, y).RGBA()
			gray1 := (r1 + g1 + b1) / 3
			isBlack1 := gray1 < 32768

			// 获取下半部分像素（如果存在）
			var isBlack2 bool
			if y+scale < bounds.Max.Y {
				r2, g2, b2, _ := img.At(x, y+scale).RGBA()
				gray2 := (r2 + g2 + b2) / 3
				isBlack2 = gray2 < 32768
			}

			// 根据上下两个像素的组合选择半块字符，并按charScale放大
			var char string
			if isBlack1 && isBlack2 {
				char = "█" // 全块
			} else if isBlack1 && !isBlack2 {
				char = "▀" // 上半块
			} else if !isBlack1 && isBlack2 {
				char = "▄" // 下半块
			} else {
				char = " " // 空格
			}

			// 按charScale重复字符以放大显示
			line += strings.Repeat(char, charScale)
		}
		line += "    " // 右边距
		logrus.Info(line)
	}

	// 添加底部边距
	for i := 0; i < 2; i++ {
		logrus.Info(topMargin)
	}

	logrus.Info("========================================")
	return nil
}

// displayBackupQRCodeWithQRTerminal 显示备用提示信息
func (q *QRCodeDisplay) displayBackupQRCodeWithQRTerminal() {
	logrus.Info("📱 主要方式: 扫描上方ASCII格式的小红书二维码")
	logrus.Info("💾 备选方式: 查看保存的 qrcode_login.png 文件")
	logrus.Info("🌐 备选方式: 复制数据URL到浏览器查看")
}

// printQRCodeInstructions 打印二维码说明
func (q *QRCodeDisplay) printQRCodeInstructions(dataURL string) {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("           小红书登录二维码")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("请使用小红书手机App扫描以下二维码登录：")
	fmt.Println()

	// 如果有可用的二维码转ASCII工具，可以在这里显示
	// 目前先显示数据URL供调试
	fmt.Printf("二维码数据URL (可在浏览器中查看): \n%s\n", dataURL)
	fmt.Println()
	fmt.Println("或者访问以下链接查看二维码：")
	fmt.Printf("data:text/html,<img src='%s' style='width:300px;height:300px;'/>\n", dataURL)
	fmt.Println()
	fmt.Println("登录步骤：")
	fmt.Println("1. 打开小红书手机App")
	fmt.Println("2. 点击右下角 '我'")
	fmt.Println("3. 点击右上角扫码图标")
	fmt.Println("4. 扫描上方二维码")
	fmt.Println("5. 在手机上确认登录")
	fmt.Println()
	fmt.Println("等待扫码登录...")
	fmt.Println("========================================")
	fmt.Println()
}

// SaveQRCodeToFile 保存二维码到文件
func (q *QRCodeDisplay) SaveQRCodeToFile(dataURL string, filename string) error {
	// 分离base64数据
	parts := strings.Split(dataURL, ",")
	if len(parts) != 2 {
		return fmt.Errorf("invalid data URL format")
	}

	base64Data := parts[1]
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("failed to decode base64 data: %v", err)
	}

	// 保存到文件
	err = os.WriteFile(filename, imageData, 0644)
	if err != nil {
		return fmt.Errorf("failed to save QR code to file: %v", err)
	}

	logrus.Infof("二维码已保存到: %s", filename)
	return nil
}

// printASCIIQRCode 打印ASCII二维码（简化版本）
func (q *QRCodeDisplay) printASCIIQRCode() {
	// 这里可以实现一个简单的ASCII二维码显示
	// 由于复杂性，目前显示提示信息
	fmt.Println("┌─────────────────────┐")
	fmt.Println("│                     │")
	fmt.Println("│  █▀▀ █▀█  █▀▀ █▀█  │")
	fmt.Println("│  █▄▄ █▀▄  █▄▄ █▄█  │")
	fmt.Println("│                     │")
	fmt.Println("│  请在浏览器中查看    │")
	fmt.Println("│  完整二维码          │")
	fmt.Println("│                     │")
	fmt.Println("└─────────────────────┘")
}
