package main

// Config 应用配置
type Config struct {
	Headless bool   // 是否使用无头浏览器
	BinPath  string // 浏览器二进制文件路径
	Username string // 登录用户名（可选，用于显示）
}

// 全局配置变量
var globalConfig *Config

// InitConfig 初始化配置
func InitConfig(config *Config) {
	globalConfig = config
}

// GetConfig 获取配置
func GetConfig() *Config {
	if globalConfig == nil {
		return &Config{
			Headless: true,
			BinPath:  "",
			Username: "",
		}
	}
	return globalConfig
}
