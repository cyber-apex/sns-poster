# Docker环境设置指南 - 共享目录方案

本文档介绍如何在Docker环境中运行SNS Notify，使用共享目录挂载方式上传图片。

## 概述

SNS Notify 使用 Rod Manager 在Docker容器中运行浏览器。共享目录方案的优势：

- ✅ **传统文件上传**：使用标准的文件路径方式
- ✅ **稳定可靠**：基于成熟的Docker卷挂载技术
- ✅ **易于调试**：可以直接查看临时文件
- ✅ **高性能**：直接文件传输，无编码开销

## 工作原理

### 共享目录上传流程

1. **临时目录创建**：
   - 在宿主机创建 `/tmp/sns-notify` 目录
   - Docker挂载该目录到容器相同路径

2. **图片处理**：
   - 下载网络图片到临时目录
   - 复制本地图片到临时目录
   - 验证图片格式和大小

3. **路径转换**：
   - 确保图片在 `/tmp/sns-notify` 目录下
   - 容器内可以直接访问相同路径

4. **文件上传**：
   - 使用标准的SetFiles方法
   - 浏览器直接读取文件
   - 上传到小红书服务器

### 技术优势

- **标准方案**：使用浏览器原生文件上传
- **高性能**：无需编码转换开销
- **易维护**：传统文件操作，易于理解和调试

## 快速开始

### 1. 启动Rod Manager

使用提供的启动脚本：

```bash
./start-rod-manager.sh
```

脚本会自动：
- 创建 `/tmp/xhs-poster` 目录
- 启动Docker容器并挂载该目录
- 验证容器状态

### 2. 验证Rod Manager状态

```bash
# 检查容器状态
docker ps | grep xhs-poster-rod

# 检查挂载目录
ls -la /tmp/xhs-poster

# 查看管理器界面
curl http://localhost:7317
```

### 3. 启动SNS Notify

```bash
./sns-notify -http-port :6170
```

### 4. 测试上传功能

```bash
curl -X POST http://localhost:6170/publish \
  -H "Content-Type: application/json" \
  -d '{
    "title": "测试标题",
    "content": "测试内容",
    "images": ["https://example.com/image.jpg"]
  }'
```

## 配置说明

### Rod Manager配置

- **端口映射**：`7317:7317`
- **卷挂载**：`/tmp/xhs-poster:/tmp/xhs-poster`
- **镜像**：`ghcr.io/go-rod/rod`

### 临时目录

- **宿主机路径**：`/tmp/xhs-poster`
- **容器内路径**：`/tmp/xhs-poster`
- **权限**：755（可读写执行）

## 文件管理

### 清理临时文件

使用提供的清理脚本：

```bash
# 清理临时图片文件
./cleanup-temp.sh

# 手动清理
rm -rf /tmp/xhs-poster/*
```

### 监控磁盘使用

```bash
# 查看临时目录大小
du -sh /tmp/xhs-poster

# 列出临时文件
ls -la /tmp/xhs-poster
```

## 故障排除

### 1. 文件上传失败

如果遇到"no such file or directory"错误：

```bash
# 检查临时目录
ls -la /tmp/xhs-poster

# 检查容器挂载
docker inspect xhs-poster-rod | grep -A 10 Mounts

# 重启Rod Manager
docker restart xhs-poster-rod
```

### 2. 权限问题

```bash
# 检查目录权限
ls -ld /tmp/xhs-poster

# 修复权限
sudo chmod 755 /tmp/xhs-poster
sudo chown $USER:$USER /tmp/xhs-poster
```

### 3. 容器启动失败

```bash
# 检查Docker状态
docker info

# 重新拉取镜像
docker pull ghcr.io/go-rod/rod

# 清理并重启
docker stop xhs-poster-rod
docker rm xhs-poster-rod
./start-rod-manager.sh
```

## 高级配置

### 自定义临时目录

修改 `start-rod-manager.sh` 中的 `TEMP_DIR` 变量：

```bash
# 使用自定义目录
TEMP_DIR="/your/custom/path"
```

### 监控和日志

```bash
# 查看Rod Manager日志
docker logs -f xhs-poster-rod

# 查看XHS Poster日志
./xhs-poster -http-port :6170 2>&1 | tee xhs-poster.log

# 监控容器资源
docker stats xhs-poster-rod
```

### 性能优化

- **磁盘空间**：定期清理临时文件
- **网络优化**：确保良好的网络连接
- **图片优化**：建议图片大小控制在5MB以内

## 安全考虑

### 文件安全

- 临时文件自动清理
- 文件权限控制
- 路径验证防止目录遍历

### 网络安全

- 仅监听本地端口
- 使用HTTPS下载图片
- 避免敏感信息泄露

## 总结

共享目录方案提供了一个稳定、高性能的图片上传解决方案，特别适合：

- 需要高性能文件传输的场景
- 传统的容器化部署环境
- 生产环境部署

通过本指南，您应该能够成功在Docker环境中运行XHS Poster。