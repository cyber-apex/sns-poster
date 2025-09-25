# SNS Notify

一个支持多社交网络平台的内容发布工具，当前支持小红书(XHS)平台。

## 🌟 特性

- 🔐 **自动登录**: 支持二维码扫码登录
- 📝 **内容发布**: 支持图文内容发布到小红书
- 🖼️ **图片处理**: 自动处理和优化图片
- 🌐 **HTTP API**: 提供RESTful API接口
- 📊 **健康检查**: 内置服务健康监控
- 🔧 **易于扩展**: 模块化架构，易于添加新平台

## 📁 项目结构

```
sns-notify/
├── cmd/sns-notify/          # 主程序入口
├── internal/
│   ├── config/             # 配置管理
│   ├── logger/             # 日志系统
│   ├── server/             # HTTP服务器
│   ├── xhs/               # 小红书平台支持
│   └── utils/             # 通用工具
├── scripts/               # 构建和部署脚本
├── docs/                 # 文档
└── go.mod               # Go模块定义
```

## 🚀 快速开始

### 使用 Makefile (推荐)

```bash
# 查看所有可用命令
make help

# 安装依赖
make deps

# 开发构建并运行
make dev

# 生产构建 (Linux AMD64)
make build-linux

# 运行测试
make test

# 快速启动
make quick-start
```

### 传统方式

#### 安装依赖
```bash
go mod download
```

#### 构建项目
```bash
# 使用构建脚本
./scripts/build.sh

# 或手动构建
go build -o sns-notify ./cmd/sns-notify
```

#### 运行服务器
```bash
# 开发模式
./scripts/dev.sh

# 生产模式
./sns-notify -http-port=:6170 -log-file=/var/logs/sns-notify/sns-notify.log
```

## 📖 API 文档

### 健康检查
```bash
GET /health
```

### 小红书 (XHS) API

#### 检查登录状态
```bash
GET /api/v1/xhs/login/status
```

#### 手动登录
```bash
POST /api/v1/xhs/login
```

#### 发布内容
```bash
POST /api/v1/xhs/publish
Content-Type: application/json

{
  "title": "标题",
  "content": "内容文本",
  "images": ["图片URL或本地路径"],
  "tags": ["标签1", "标签2"]
}
```

## 🔧 配置说明

### 命令行参数

- `-http-port`: HTTP服务器端口，默认 `:6170`
- `-log-file`: 日志文件路径，留空输出到控制台

### 环境要求

- Go 1.24+
- Chrome/Chromium 浏览器（用于自动化）
- 足够的磁盘空间用于图片处理

## 🔐 登录流程

1. 首次访问发布API时会自动触发登录
2. 终端会显示二维码，使用小红书APP扫码登录
3. 登录成功后cookie会自动保存，后续无需重复登录

## 🛠️ 系统服务部署

### 安装为系统服务
```bash
# 安装服务 (需要sudo权限)
make install

# 启动服务
make service-start

# 设置开机自启
make service-enable

# 查看服务状态
make service-status

# 查看服务日志
make service-logs
```

### 服务管理命令
```bash
make service-start    # 启动服务
make service-stop     # 停止服务
make service-restart  # 重启服务
make service-status   # 查看状态
make service-logs     # 查看日志
make service-enable   # 启用开机自启
make service-disable  # 禁用开机自启
make uninstall        # 卸载服务
```

## 🧪 测试和验证

```bash
# 测试API端点
make test-api

# 测试发布功能
make test-post

# 健康检查
make health-check

# 运行完整测试套件
make test

# 运行性能测试
make test-race
```

## 📊 监控和日志

- 健康检查端点: `/health`
- 结构化日志输出
- 支持文件和控制台日志输出
- systemd服务日志集成

## 🐳 Docker 部署

参考 `docs/DOCKER_SETUP.md` 了解Docker部署详情。

## 🏗️ CI/CD

参考 `docs/JENKINS.md` 了解Jenkins CI/CD配置。

## 🤝 扩展新平台

1. 在 `internal/` 下创建新平台目录
2. 实现平台特定的登录和发布逻辑
3. 在 `internal/server/` 中添加相应的API路由
4. 更新构建脚本和文档

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🏗️ 架构图

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   HTTP Client   │────│   HTTP Server    │────│  XHS Service    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                       ┌──────────────────┐    ┌─────────────────┐
                       │   Logger/Config  │    │   Browser       │
                       └──────────────────┘    └─────────────────┘
                                                        │
                                               ┌─────────────────┐
                                               │  小红书网站      │
                                               └─────────────────┘
```
