#!/bin/bash

# XHS Poster Test Script
# Tests both HTTP and gRPC publishing functionality

set -e

# Configuration
HTTP_BASE="http://localhost:6170"
GRPC_ADDR="localhost:6171"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

check_json_field() {
    local json="$1"
    local field="$2"
    echo "$json" | grep -o "\"$field\":[^,}]*" | cut -d':' -f2 | tr -d '"' | tr -d ' '
}

get_json_value() {
    local json="$1"
    local field="$2"
    echo "$json" | grep -o "\"$field\":\"[^\"]*\"" | cut -d'"' -f4
}

# Test server connectivity
test_server_health() {
    log_info "Testing server connectivity..."
    
    # Test HTTP health endpoint
    HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$HTTP_BASE/health" || echo "000")
    if [ "$HTTP_STATUS" = "200" ]; then
        log_success "HTTP server is responding on port 6170"
    else
        log_error "HTTP server not responding (HTTP $HTTP_STATUS)"
        exit 1
    fi
}

# Check login status (informational only)
check_login_status() {
    log_info "Checking current login status..."
    
    RESPONSE=$(curl -s "$HTTP_BASE/api/v1/login/status")
    IS_LOGGED_IN=$(check_json_field "$RESPONSE" "is_logged_in")
    
    if [ "$IS_LOGGED_IN" = "true" ]; then
        USERNAME=$(get_json_value "$RESPONSE" "username")
        log_success "User is already logged in: $USERNAME"
        return 0
    else
        log_info "User is not logged in - auto-login will be triggered on first protected API call"
        return 1
    fi
}

# Test HTTP publishing
test_http_publish() {
    log_info "Testing HTTP publish endpoint..."
    
    # Create test post data
    local test_data='{
        "title": "API测试发布 - HTTP",
        "content": "这是通过HTTP API测试发布的内容。\n\n包含以下特性：\n• 多行内容\n• 特殊字符测试\n• Emoji 🚀\n\n发布时间：'$(date)'",
        "images": [
            "https://placehold.co/400x400/FF6B6B/FFFFFF.png?text=Test+Image+1",
            "https://placehold.co/400x400/4ECDC4/FFFFFF.png?text=Test+Image+2"
        ],
        "tags": ["API测试", "自动化", "小红书"]
    }'
    
    log_info "Sending publish request via HTTP..."
    log_info "Note: This may trigger auto-login if user is not authenticated"
    RESPONSE=$(curl -s -X POST "$HTTP_BASE/api/v1/publish" \
        -H "Content-Type: application/json" \
        -d "$test_data")
    
    # Check response
    SUCCESS=$(check_json_field "$RESPONSE" "success")
    if [ "$SUCCESS" = "true" ]; then
        log_success "HTTP publish test completed successfully"
        echo "Response: $RESPONSE"
    else
        ERROR=$(get_json_value "$RESPONSE" "error")
        CODE=$(get_json_value "$RESPONSE" "code")
        if echo "$RESPONSE" | grep -q "AUTO_LOGIN_FAILED\|LOGIN_REQUIRED"; then
            log_warning "Auto-login was triggered but requires user action"
            log_info "Please check server logs for QR code and scan with Xiaohongshu app"
        else
            log_error "HTTP publish failed: $CODE - $ERROR"
        fi
        echo "Full response: $RESPONSE"
    fi
}

# Test gRPC publishing (if grpcurl is available)
test_grpc_publish() {
    if ! command -v grpcurl &> /dev/null; then
        log_warning "grpcurl not installed, skipping gRPC test"
        log_info "To install: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
        return
    fi
    
    log_info "Testing gRPC publish endpoint..."
    
    # Create test post data for gRPC
    local grpc_data='{
        "title": "API测试发布 - gRPC",
        "content": "这是通过gRPC API测试发布的内容。\n\n包含以下特性：\n• gRPC协议\n• 二进制传输\n• 高性能 ⚡\n\n发布时间：'$(date)'",
        "images": [
            "https://placehold.co/400x400/845EC2/FFFFFF.png?text=gRPC+Test+1",
            "https://placehold.co/400x400/F093FB/FFFFFF.png?text=gRPC+Test+2"
        ],
        "tags": ["gRPC测试", "自动化", "API"]
    }'
    
    log_info "Sending publish request via gRPC..."
    RESPONSE=$(grpcurl -plaintext -d "$grpc_data" "$GRPC_ADDR" XHSService/PublishContent 2>&1)
    
    if echo "$RESPONSE" | grep -q "status"; then
        log_success "gRPC publish test completed"
        echo "Response: $RESPONSE"
    else
        log_error "gRPC publish failed"
        echo "Error: $RESPONSE"
    fi
}

# Test with local images (if available)
test_local_images() {
    log_info "Testing with local image files..."
    
    # Create a simple test image if it doesn't exist
    if ! command -v convert &> /dev/null; then
        log_warning "ImageMagick not installed, creating simple test file instead"
        echo "Test image content" > test_image.txt
        log_info "Created test_image.txt as placeholder"
        return
    fi
    
    # Create test images using ImageMagick
    convert -size 400x400 xc:lightblue -gravity center -pointsize 24 \
        -annotate +0+0 "Local Test Image 1\n$(date +%H:%M:%S)" test_image_1.png
    convert -size 400x400 xc:lightgreen -gravity center -pointsize 24 \
        -annotate +0+0 "Local Test Image 2\n$(date +%H:%M:%S)" test_image_2.png
    
    # Test with local images
    local local_test_data='{
        "title": "本地图片测试",
        "content": "这是使用本地图片文件的测试发布。\n\n测试内容：\n• 本地文件上传\n• 图片处理\n• 文件路径处理\n\n测试时间：'$(date)'",
        "images": [
            "./test_image_1.png",
            "./test_image_2.png"
        ],
        "tags": ["本地测试", "图片上传", "文件测试"]
    }'
    
    log_info "Publishing with local images..."
    RESPONSE=$(curl -s -X POST "$HTTP_BASE/api/v1/publish" \
        -H "Content-Type: application/json" \
        -d "$local_test_data")
    
    SUCCESS=$(check_json_field "$RESPONSE" "success")
    if [ "$SUCCESS" = "true" ]; then
        log_success "Local images test completed successfully"
    else
        log_error "Local images test failed"
        echo "Response: $RESPONSE"
    fi
    
    # Clean up test images
    rm -f test_image_1.png test_image_2.png test_image.txt 2>/dev/null || true
}

# Test error cases
test_error_cases() {
    log_info "Testing error handling..."
    
    # Test missing required fields
    log_info "Testing missing title..."
    RESPONSE=$(curl -s -X POST "$HTTP_BASE/api/v1/publish" \
        -H "Content-Type: application/json" \
        -d '{"content": "Test", "images": ["test.jpg"]}')
    
    if echo "$RESPONSE" | grep -q "error"; then
        log_success "Missing title error handled correctly"
    else
        log_warning "Missing title should return error"
    fi
    
    # Test empty images array
    log_info "Testing empty images array..."
    RESPONSE=$(curl -s -X POST "$HTTP_BASE/api/v1/publish" \
        -H "Content-Type: application/json" \
        -d '{"title": "Test", "content": "Test", "images": []}')
    
    if echo "$RESPONSE" | grep -q "error"; then
        log_success "Empty images array error handled correctly"
    else
        log_warning "Empty images array should return error"
    fi
    
    # Test very long title
    log_info "Testing title length limit..."
    LONG_TITLE=$(python3 -c "print('很长的标题' * 20)" 2>/dev/null || echo "Very long title that exceeds the 40 character limit for Xiaohongshu posts which should be rejected")
    RESPONSE=$(curl -s -X POST "$HTTP_BASE/api/v1/publish" \
        -H "Content-Type: application/json" \
        -d "{\"title\": \"$LONG_TITLE\", \"content\": \"Test\", \"images\": [\"test.jpg\"]}")
    
    if echo "$RESPONSE" | grep -q "error\|长度"; then
        log_success "Title length limit handled correctly"
    else
        log_warning "Long title should return error"
    fi
}

# Performance test
test_performance() {
    log_info "Running performance test..."
    
    local test_data='{
        "title": "性能测试发布",
        "content": "这是性能测试发布的内容，用于测试API响应时间。",
        "images": ["https://placehold.co/400x400/FFA07A/FFFFFF.png?text=Performance+Test"],
        "tags": ["性能测试"]
    }'
    
    log_info "Measuring API response time..."
    START_TIME=$(date +%s.%N)
    RESPONSE=$(curl -s -X POST "$HTTP_BASE/api/v1/publish" \
        -H "Content-Type: application/json" \
        -d "$test_data")
    END_TIME=$(date +%s.%N)
    
    DURATION=$(echo "$END_TIME - $START_TIME" | bc 2>/dev/null || echo "N/A")
    
    SUCCESS=$(check_json_field "$RESPONSE" "success")
    if [ "$SUCCESS" = "true" ]; then
        log_success "Performance test completed in ${DURATION}s"
    else
        log_error "Performance test failed"
    fi
}

# Main test execution
main() {
    echo "=========================================="
    echo "🚀 XHS Poster API Test Suite"
    echo "=========================================="
    echo ""
    
    # Basic connectivity test
    test_server_health
    echo ""
    
    # Check login status (informational)
    check_login_status
    echo ""
    
    log_info "Note: If not logged in, auto-login will be triggered during protected API calls"
    log_info "Watch server logs for QR code if login is required"
    echo ""
    
    # Run publishing tests
    echo "=========================================="
    echo "📝 Publishing Tests"
    echo "=========================================="
    
    test_http_publish
    echo ""
    
    test_grpc_publish
    echo ""
    
    test_local_images
    echo ""
    
    # Error handling tests
    echo "=========================================="
    echo "🛠️  Error Handling Tests"
    echo "=========================================="
    
    test_error_cases
    echo ""
    
    # Performance tests
    echo "=========================================="
    echo "⚡ Performance Tests"
    echo "=========================================="
    
    test_performance
    echo ""
    
    # Summary
    echo "=========================================="
    echo "📊 Test Summary"
    echo "=========================================="
    log_success "All tests completed!"
    log_info "Check the Xiaohongshu creator center to verify published content"
    log_info "API Documentation: http://localhost:6170/swagger (if available)"
    echo ""
}

# Show usage if help requested
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "XHS Poster Test Script"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --help, -h          Show this help message"
    echo "  --http-only         Test only HTTP endpoints"
    echo "  --grpc-only         Test only gRPC endpoints"
    echo "  --no-local          Skip local image tests"
    echo "  --no-errors         Skip error handling tests"
    echo "  --no-performance    Skip performance tests"
    echo ""
    echo "Prerequisites:"
    echo "  - Server running on localhost:6170 (HTTP) and localhost:6171 (gRPC)"
    echo "  - User logged in to Xiaohongshu"
    echo "  - grpcurl installed for gRPC tests (optional)"
    echo "  - ImageMagick for local image tests (optional)"
    echo ""
    exit 0
fi

# Parse command line arguments
HTTP_ONLY=false
GRPC_ONLY=false
NO_LOCAL=false
NO_ERRORS=false
NO_PERFORMANCE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --http-only)
            HTTP_ONLY=true
            shift
            ;;
        --grpc-only)
            GRPC_ONLY=true
            shift
            ;;
        --no-local)
            NO_LOCAL=true
            shift
            ;;
        --no-errors)
            NO_ERRORS=true
            shift
            ;;
        --no-performance)
            NO_PERFORMANCE=true
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Execute main function with options
if [ "$HTTP_ONLY" = true ]; then
    test_server_health
    check_login_status && test_http_publish
elif [ "$GRPC_ONLY" = true ]; then
    test_server_health
    check_login_status && test_grpc_publish
else
    main
fi
