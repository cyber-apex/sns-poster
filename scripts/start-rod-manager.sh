#!/bin/bash

# Rod Manager 启动脚本 - 共享目录模式
# 
# 功能：
# - 启动Rod管理器Docker容器
# - 挂载 /tmp/xhs-poster 目录用于文件共享
# - 支持传统文件上传方式

set -e

# 配置
CONTAINER_NAME="xhs-poster-rod"
HOST_PORT="7317"
CONTAINER_PORT="7317"
TEMP_DIR="/tmp/xhs-poster"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Docker是否运行
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker未运行或无访问权限"
        log_info "请确保:"
        log_info "1. Docker已安装并运行"
        log_info "2. 当前用户有Docker权限"
        exit 1
    fi
}

# 停止现有容器
stop_existing_container() {
    if docker ps -q -f name=$CONTAINER_NAME | grep -q .; then
        log_info "停止现有的Rod管理器容器..."
        docker stop $CONTAINER_NAME >/dev/null 2>&1
    fi
    
    if docker ps -aq -f name=$CONTAINER_NAME | grep -q .; then
        log_info "删除现有的Rod管理器容器..."
        docker rm $CONTAINER_NAME >/dev/null 2>&1
    fi
}

# 启动Rod管理器
start_rod_manager() {
    log_info "启动Rod管理器容器..."
    log_info "端口映射: $HOST_PORT -> $CONTAINER_PORT"
    log_info "挂载目录: $TEMP_DIR -> $TEMP_DIR"
    
    # 确保临时目录存在
    if [ ! -d "$TEMP_DIR" ]; then
        log_info "创建临时目录: $TEMP_DIR"
        mkdir -p "$TEMP_DIR"
        chmod 755 "$TEMP_DIR"
    fi
    
    docker run -d \
        --name $CONTAINER_NAME \
        -p $HOST_PORT:$CONTAINER_PORT \
        -v "$TEMP_DIR:$TEMP_DIR" \
        --restart unless-stopped \
        ghcr.io/go-rod/rod
    
    # 等待容器启动
    log_info "等待Rod管理器启动..."
    sleep 3
    
    # 检查容器状态
    if docker ps -q -f name=$CONTAINER_NAME | grep -q .; then
        log_success "Rod管理器启动成功!"
        log_info "管理器地址: http://localhost:$HOST_PORT"
        log_info "共享目录: $TEMP_DIR"
        
        # 显示容器信息
        echo
        log_info "容器状态:"
        docker ps -f name=$CONTAINER_NAME --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        
        # 显示使用说明
        echo
        log_info "使用说明:"
        log_info "1. 图片文件存储在 $TEMP_DIR 目录"
        log_info "2. 容器内可直接访问相同路径的文件"
        log_info "3. 停止管理器: docker stop $CONTAINER_NAME"
        log_info "4. 查看日志: docker logs $CONTAINER_NAME"
        log_info "5. 清理临时文件: rm -rf $TEMP_DIR/*"
        
    else
        log_error "Rod管理器启动失败"
        log_info "查看错误日志:"
        docker logs $CONTAINER_NAME 2>&1 || true
        exit 1
    fi
}

# 主函数
main() {
    echo "=================================="
    echo "🚀 XHS Poster - Rod Manager 启动器"
    echo "=================================="
    echo
    
    check_docker
    stop_existing_container
    start_rod_manager
    
    echo
    log_success "Setup完成! 现在可以启动xhs-poster应用了"
    echo
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
