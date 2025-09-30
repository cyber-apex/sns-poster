# SNS Poster 架构文档

## 🏗️ 项目架构

SNS Poster 采用模块化架构，支持多平台扩展，当前主要支持小红书平台。

## 📁 目录结构

```
sns-poster/
├── cmd/                      # 应用程序入口点
│   └── sns-poster/          # 主程序
│       └── main.go         # 程序入口，处理命令行参数和启动服务
├── internal/               # 内部模块（不对外暴露）
│   ├── config/            # 配置管理
│   │   └── config.go      # 全局配置结构和管理
│   ├── logger/            # 日志系统
│   │   └── logger.go      # 日志配置和初始化
│   ├── server/            # HTTP服务器
│   │   └── http_server.go # Gin HTTP服务器和路由配置
│   ├── xhs/              # 小红书平台支持
│   │   ├── xhs_service.go # XHS服务主要逻辑
│   │   ├── xhs_login.go   # 登录处理
│   │   └── xhs_publish.go # 内容发布
│   └── utils/             # 通用工具
│       ├── browser.go     # 浏览器自动化
│       ├── cookies.go     # Cookie管理
│       ├── qrcode.go      # 二维码处理
│       └── image_processor.go # 图片处理
├── scripts/               # 构建和部署脚本
│   ├── build.sh          # 构建脚本
│   ├── dev.sh            # 开发运行脚本
│   ├── quick_test_post.sh # 快速测试脚本
│   └── test_qr_login.sh  # 登录测试脚本
├── docs/                 # 文档
│   ├── ARCHITECTURE.md   # 架构文档（本文件）
│   ├── DOCKER_SETUP.md   # Docker部署指南
│   ├── JENKINS.md        # CI/CD配置
│   └── LOGIN_GUIDE.md    # 登录使用指南
├── go.mod               # Go模块定义
├── go.sum               # Go模块校验和
└── README.md            # 项目说明
```

## 🔄 数据流

### 1. HTTP请求处理流程

```
客户端请求 → HTTP Server → Platform Service → Browser → 目标平台
     ↓             ↓              ↓           ↓         ↓
   响应 ←─────── 响应 ←──────── 响应 ←──── 响应 ←─── 处理结果
```

### 2. 登录流程

```
1. 客户端请求登录状态检查
2. XHS Service 检查本地Cookie
3. 如果未登录，启动浏览器自动化
4. 显示二维码等待扫码
5. 检测登录成功后保存Cookie
6. 返回登录状态
```

### 3. 发布流程

```
1. 客户端发送发布请求
2. 检查登录状态（如未登录自动触发登录）
3. 处理图片（下载、压缩、格式转换）
4. 启动浏览器导航到发布页面
5. 自动填写内容和上传图片
6. 提交发布
7. 返回发布结果
```

## 🧩 核心模块

### Config (配置管理)
- 全局配置结构
- 配置初始化和获取
- 支持用户名等基本配置

### Logger (日志系统)
- 统一的日志格式和输出
- 支持文件和控制台输出
- 集成Gin框架日志

### Server (HTTP服务器)
- 基于Gin框架的HTTP服务
- RESTful API路由
- 中间件支持（CORS、日志、认证）
- 健康检查端点

### XHS (小红书平台)
- **Service**: 主要业务逻辑协调
- **Login**: 登录处理和状态管理
- **Publisher**: 内容发布功能

### Utils (通用工具)
- **Browser**: 浏览器自动化封装
- **Cookies**: Cookie持久化管理
- **QRCode**: 二维码显示和保存
- **ImageProcessor**: 图片处理和优化

## 🔌 扩展新平台

要添加新的社交平台支持，需要：

### 1. 创建平台模块
```
internal/
└── newplatform/
    ├── service.go      # 平台服务主逻辑
    ├── login.go        # 登录处理
    └── publish.go      # 发布功能
```

### 2. 添加HTTP路由
在 `internal/server/http_server.go` 中添加新平台的API路由：
```go
newplatform := api.Group("/newplatform")
{
    newplatform.GET("/login/status", s.checkNewPlatformLoginStatusHandler)
    newplatform.POST("/login", s.newPlatformLoginHandler)
    // ... 其他路由
}
```

### 3. 实现处理器
添加对应的HTTP处理器函数。

### 4. 更新构建脚本
确保新模块被正确编译和测试。

## 🔐 安全考虑

### 认证和授权
- 每个平台独立的认证机制
- Cookie安全存储
- 敏感信息不记录到日志

### 数据保护
- 图片临时文件自动清理
- Cookie文件权限控制
- 输入数据验证

## 📊 监控和观测

### 日志
- 结构化日志输出
- 请求响应时间记录
- 错误堆栈跟踪

### 健康检查
- `/health` 端点提供服务状态
- 浏览器连接状态检查
- 平台服务可用性检查

## 🚀 部署架构

### 单机部署
```
┌─────────────────┐
│   SNS Poster    │
│   (Binary)      │
├─────────────────┤
│   Chrome        │
│   (Headless)    │
├─────────────────┤
│   File System   │
│   (Logs/Images) │
└─────────────────┘
```

### 容器化部署
```
┌─────────────────┐
│   Docker        │
│  ┌─────────────┐│
│  │ SNS Poster  ││
│  │ + Chrome    ││
│  └─────────────┘│
├─────────────────┤
│   Volume        │
│ (Logs/Config)   │
└─────────────────┘
```

## 🔧 配置管理

### 环境变量
- `LOG_LEVEL`: 日志级别
- `HTTP_PORT`: HTTP服务端口
- `CHROME_PATH`: Chrome浏览器路径

### 配置文件
当前主要通过命令行参数配置，未来可扩展支持配置文件。

## 📈 性能优化

### 浏览器管理
- 复用浏览器实例
- 及时关闭页面释放内存
- 合理的超时设置

### 图片处理
- 异步图片下载
- 智能压缩和格式转换
- 临时文件清理

### 并发控制
- 单浏览器实例避免资源冲突
- 合理的请求队列管理
