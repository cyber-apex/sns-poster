package xhs

import (
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/assert"
)

// 使用 runewidth 计算显示宽度（中文2字符，英文1字符）
func TestRuneWidth(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		expectedWidth int
		description   string
	}{
		{
			name:          "Empty string",
			text:          "",
			expectedWidth: 0,
			description:   "Empty string should have width 0",
		},
		{
			name:          "Pure English",
			text:          "Hello World",
			expectedWidth: 11,
			description:   "English characters count as 1 width each",
		},
		{
			name:          "Pure Chinese",
			text:          "你好世界",
			expectedWidth: 8,
			description:   "Chinese characters count as 2 width each (4 chars * 2)",
		},
		{
			name:          "Mixed English and Chinese",
			text:          "Hello你好World世界",
			expectedWidth: 18,
			description:   "Mixed text: 'Hello' (5) + '你好' (4) + 'World' (5) + '世界' (4) = 18",
		},
		{
			name:          "Numbers",
			text:          "12345",
			expectedWidth: 5,
			description:   "Numbers count as 1 width each",
		},
		{
			name:          "Special characters",
			text:          "!@#$%^&*()",
			expectedWidth: 10,
			description:   "Special ASCII characters count as 1 width each",
		},
		{
			name:          "Chinese with punctuation",
			text:          "你好，世界！",
			expectedWidth: 12,
			description:   "Chinese with full-width punctuation: 你好(4) + ，(2) + 世界(4) + ！(2)",
		},
		{
			name:          "Mixed with numbers",
			text:          "测试123Test",
			expectedWidth: 11,
			description:   "Mixed: 测试(4) + 123(3) + Test(4) = 11",
		},
		{
			name:          "Japanese Hiragana",
			text:          "こんにちは",
			expectedWidth: 10,
			description:   "Japanese Hiragana characters count as 2 width each",
		},
		{
			name:          "Japanese Katakana",
			text:          "カタカナ",
			expectedWidth: 8,
			description:   "Japanese Katakana characters count as 2 width each",
		},
		{
			name:          "Korean characters",
			text:          "안녕하세요",
			expectedWidth: 10,
			description:   "Korean characters count as 2 width each",
		},
		{
			name:          "Title at max width",
			text:          "这是一个测试标题Test",
			expectedWidth: 20,
			description:   "这是一个测试标题(8 chars = 16) + Test(4) = 20",
		},
		{
			name:          "Emoji",
			text:          "Hello 😀 World",
			expectedWidth: 14,
			description:   "Emoji typically counts as 2 width",
		},
		{
			name:          "Tab and newline",
			text:          "Hello\tWorld\n",
			expectedWidth: 10,
			description:   "Tab and newline are control chars with no display width",
		},
		{
			name:          "Newline",
			text:          "■发售日期：\n",
			expectedWidth: 11,
			description:   "Newline counts as 2 width: ■发售日期：(15) + \n(2) = 17",
		},
		{
			name:          "Title near max limit",
			text:          "■发售日期：\n实体店销售：预计自2025年11月08日（周六）起陆续发售\n线上销售：预计自2025年11月10日（周一）17:00起开始销售\n■制造商建议零售价：每次790日元（含10%消费税）\n■销售店铺：罗森便利店、书店、模型玩具店、一番赏官方商店、一番赏ONLINE等\n■双倍机会活动期间：发售日起至2026年2月底\n\nA奖 桓骑 MASTERLISE：\n■全1种\n■尺寸：约27cm\n来自动画《王者天下》，\"桓骑\"首次登场一番赏！桓骑首次以MASTERLISE系列立体化!! \"一切都会顺利\"\n\nB奖 腾 MASTERLISE：\n■全1种\n■尺寸：约27cm\n来自动画《王者天下》，\"腾\"首次登场一番赏！腾首次以MASTERLISE系列立体化!! \"法尔法尔法尔\"\n\nC奖 王翦 MASTERLISE：\n■全1种\n■尺寸：约27cm\n来自动画《王者天下》，\"王翦\"首次登场一番赏！王翦首次以MASTERLISE系列立体化!! \"我只对『必胜之战』感兴趣\"\n\nD奖 大盘：\n■全2种（不可选）\n■尺寸：约19cm\n推出以动画《王者天下》中军旗为主题的大盘与全新绘制插画的腾设计两种大盘！采用充满《王者天下》风格的设计！\n\nE奖 ACLLECT -春秋战国大战王者天下 The Animation vol.1-：\n■全16种（不可选）\n■尺寸：约8.5cm\n新系列\"ACLLECT\"首次推出《王者天下》动画主题阵容!! 结合亚克力立牌与收藏卡元素，是收集乐趣十足的收藏品！采用全新绘制插图，让人想要集齐所有款式!!\n\nF奖 带吸盘板：\n■全6种（可选）\n■尺寸：约15cm\n设计有动画《王者天下》登场角色的带吸盘板！包含令人会心一笑的设计和实用文字等双面规格，可根据用途选择使用！\n\nG奖 军旗橡胶挂件：\n■全8种（不可选）\n■尺寸：约7cm\n推出以动画《王者天下》中军旗为主题设计的橡胶挂件！既可挂在包上，也可与手办一起装饰的实用设计!!\n\nH奖 帆布风格板：\n■全6种（可选）\n■尺寸：B6\n推出设计有动画《王者天下》登场角色的帆布风格板！全部采用全新绘制插图，打造出令人想要装饰展示的设计！\n\n最终奖 桓骑 MASTERLISE 最终版：\n■尺寸：约27cm\n来自动画《王者天下》，\"桓骑\"首次登场一番赏！桓骑首次以MASTERLISE系列立体化!! 最终版中呈现手持佩剑的桓骑经典姿态。抽中最后一个即可获得！※请在各店铺确认剩余抽奖数量。",
			expectedWidth: 1675,
			description:   "1675 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualWidth := runewidth.StringWidth(tt.text)
			assert.Equal(t, tt.expectedWidth, actualWidth,
				"Width mismatch for '%s': %s", tt.text, tt.description)
		})
	}
}

func TestRuneWidthTruncate(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		maxWidth       int
		suffix         string
		expectedResult string
		expectedWidth  int
		description    string
	}{
		{
			name:           "No truncation needed - English",
			text:           "Hello",
			maxWidth:       10,
			suffix:         "",
			expectedResult: "Hello",
			expectedWidth:  5,
			description:    "Text shorter than max, no truncation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runewidth.Truncate(tt.text, tt.maxWidth, tt.suffix)
			assert.Equal(t, tt.expectedResult, result,
				"Truncation mismatch for '%s': %s", tt.text, tt.description)

			actualWidth := runewidth.StringWidth(result)
			assert.LessOrEqual(t, actualWidth, tt.maxWidth,
				"Truncated text width (%d) should not exceed max (%d): %s",
				actualWidth, tt.maxWidth, tt.description)

			// Only check exact width if specified (sometimes truncation can't hit exact width)
			if tt.expectedWidth > 0 {
				assert.Equal(t, tt.expectedWidth, actualWidth,
					"Expected width %d but got %d for result '%s': %s",
					tt.expectedWidth, actualWidth, result, tt.description)
			}
		})
	}
}
