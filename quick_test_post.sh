#!/bin/bash

# Quick XHS Post Test
# Simple script to test posting functionality

# Configuration
API_BASE="http://localhost:6170"

echo "🚀 Quick XHS Post Test"
echo "========================"

# Check if server is running
echo "1. Checking server status..."
if ! curl -s "$API_BASE/health" > /dev/null; then
    echo "❌ Server not responding. Please start the server first."
    exit 1
fi
echo "✅ Server is running"

# Test post with auto-login
echo "2. Sending test post (auto-login enabled)..."
echo "   ℹ️  If not logged in, QR code will appear in server logs"
TEST_DATA='{
    "title": "快速测试发布 🧪",
    "content": "这是一个快速测试发布！\n\n测试时间：'$(date)'\n测试内容包括：\n• 中文字符\n• Emoji 🎉\n• 换行符\n• 特殊字符 @#$%",
    "images": [
        "https://placehold.co/400x400/42A5F5/FFFFFF.png?text=Quick+Test"
    ],
    "tags": ["快速测试", "API", "自动化"]
}'

echo "   📤 Making POST request to /api/v1/publish..."
echo "   ⏳ This may take time if login is required (check server logs for QR code)"

RESPONSE=$(curl -s -X POST "$API_BASE/api/v1/publish" \
    -H "Content-Type: application/json" \
    -d "$TEST_DATA")

echo ""
echo "3. Analyzing response..."
if echo "$RESPONSE" | grep -q '"success":true'; then
    echo "✅ Post published successfully!"
    echo "📋 Response: $RESPONSE"
elif echo "$RESPONSE" | grep -q 'AUTO_LOGIN_FAILED\|LOGIN_REQUIRED'; then
    echo "🔐 Auto-login was triggered but needs user action"
    echo "📱 Please check server logs for QR code and scan with Xiaohongshu app"
    echo "🔍 Response: $RESPONSE"
elif echo "$RESPONSE" | grep -q 'AUTH_CHECK_FAILED'; then
    echo "❌ Authentication system error"
    echo "🔍 Response: $RESPONSE"
else
    echo "❌ Post failed to publish"
    echo "🔍 Response: $RESPONSE"
fi

echo ""
echo "🎯 Test completed! Check your Xiaohongshu creator dashboard to verify the post."
