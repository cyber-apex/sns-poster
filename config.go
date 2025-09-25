package main

// Config 应用配置
type Config struct {
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
			Username: "",
		}
	}
	return globalConfig
}
