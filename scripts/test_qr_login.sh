#!/bin/bash

echo "测试小红书二维码登录功能..."

API_BASE="http://localhost:6170"

# Simple JSON parsing functions
get_json_bool() {
    local json="$1"
    local key="$2"
    echo "$json" | grep -o "\"$key\":[^,}]*" | cut -d: -f2 | tr -d ' "' | head -1
}

get_json_string() {
    local json="$1"
    local key="$2"
    echo "$json" | grep -o "\"$key\":\"[^\"]*\"" | cut -d'"' -f4 | head -1
}

pretty_print() {
    local json="$1"
    echo "$json" | sed 's/,/,\n  /g' | sed 's/{/{\n  /' | sed 's/}/\n}/'
}

echo "=== 1. 检查服务器状态 ==="
HEALTH_RESPONSE=$(curl -s $API_BASE/health)
echo "服务器响应:"
pretty_print "$HEALTH_RESPONSE"

echo -e "\n=== 2. 检查当前登录状态 ==="
LOGIN_STATUS=$(curl -s $API_BASE/api/v1/xhs/login/status)
echo "登录状态响应:"
pretty_print "$LOGIN_STATUS"

IS_LOGGED_IN=$(get_json_bool "$LOGIN_STATUS" "is_logged_in")

echo -e "\n=== 3. 登录状态分析 ==="
if [ "$IS_LOGGED_IN" = "true" ]; then
    USERNAME=$(get_json_string "$LOGIN_STATUS" "username")
    echo "✅ 已登录，用户名: $USERNAME"
    echo "可以直接使用发布功能"
else
    echo "❌ 未登录，需要进行登录"
    echo ""
    echo "=== 4. 启动登录流程 ==="
    echo "正在调用登录API..."
    echo "注意：这将在无头模式下显示二维码，请按照提示进行扫码"
    echo ""
    
    # 调用登录API
    LOGIN_RESULT=$(curl -s -X POST $API_BASE/api/v1/xhs/login)
    echo "登录API响应:"
    pretty_print "$LOGIN_RESULT"
    
    # 检查登录结果
    LOGIN_SUCCESS=$(get_json_bool "$LOGIN_RESULT" "success")
    LOGIN_MESSAGE=$(get_json_string "$LOGIN_RESULT" "message")
    
    if [ "$LOGIN_SUCCESS" = "true" ]; then
        echo "✅ 登录API调用成功: $LOGIN_MESSAGE"
    else
        echo "❌ 登录API调用失败: $LOGIN_MESSAGE"
    fi
    
    # 检查登录是否成功
    echo -e "\n=== 5. 验证登录结果 ==="
    FINAL_STATUS=$(curl -s $API_BASE/api/v1/xhs/login/status)
    echo "最终登录状态:"
    pretty_print "$FINAL_STATUS"
    
    FINAL_LOGGED_IN=$(get_json_bool "$FINAL_STATUS" "is_logged_in")
    if [ "$FINAL_LOGGED_IN" = "true" ]; then
        FINAL_USERNAME=$(get_json_string "$FINAL_STATUS" "username")
        echo "🎉 登录成功！用户名: $FINAL_USERNAME"
    else
        echo "⚠️  登录可能未完成，请检查二维码扫描情况"
    fi
fi

echo -e "\n=== 二维码登录说明 ==="
echo "1. 新的登录系统支持无头模式下的二维码登录"
echo "2. 登录时会自动显示二维码信息和扫码说明"
echo "3. 二维码图片会保存到当前目录的 qrcode_login.png 文件"
echo "4. 支持cookie持久化，登录状态会自动保存"
echo "5. 如果已有有效的cookies，会自动恢复登录状态"

echo -e "\n=== 测试完成 ==="
