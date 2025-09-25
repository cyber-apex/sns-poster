#!/bin/bash

# 清理临时图片文件脚本

TEMP_DIR="/tmp/xhs-poster"

# 颜色输出
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

main() {
    log_info "清理XHS Poster临时文件..."
    
    if [ ! -d "$TEMP_DIR" ]; then
        log_warning "临时目录不存在: $TEMP_DIR"
        exit 0
    fi
    
    # 显示目录大小
    if command -v du &> /dev/null; then
        SIZE=$(du -sh "$TEMP_DIR" 2>/dev/null | cut -f1)
        log_info "临时目录大小: $SIZE"
    fi
    
    # 显示文件数量
    FILE_COUNT=$(find "$TEMP_DIR" -type f 2>/dev/null | wc -l)
    log_info "临时文件数量: $FILE_COUNT"
    
    if [ "$FILE_COUNT" -eq 0 ]; then
        log_info "没有临时文件需要清理"
        exit 0
    fi
    
    # 列出前10个文件
    log_info "临时文件列表（最多显示10个）:"
    find "$TEMP_DIR" -type f -exec basename {} \; 2>/dev/null | head -10 | sed 's/^/  - /'
    
    if [ "$FILE_COUNT" -gt 10 ]; then
        log_info "  ... 还有 $((FILE_COUNT - 10)) 个文件"
    fi
    
    # 询问是否删除
    echo
    read -p "是否删除所有临时文件? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "正在删除临时文件..."
        rm -rf "$TEMP_DIR"/*
        log_success "临时文件已清理完成"
    else
        log_info "取消清理操作"
    fi
}

# 如果作为脚本直接运行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
