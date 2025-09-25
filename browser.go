package main

import (
	"github.com/xpzouying/headless_browser"
)

// NewBrowser 创建浏览器实例
func NewBrowser(config *Config) *headless_browser.Browser {
	options := []headless_browser.Option{
		headless_browser.WithHeadless(config.Headless),
	}

	if config.BinPath != "" {
		options = append(options, headless_browser.WithChromeBinPath(config.BinPath))
	}

	return headless_browser.New(options...)
}
