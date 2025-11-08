#!/bin/bash

# Multi-Account Browser Pool Test Script
# This script demonstrates the multi-account functionality

BASE_URL="http://localhost:8080"

echo "================================================"
echo "Multi-Account Browser Pool Test"
echo "================================================"
echo ""

# Test 1: Check default account login status
echo "1. Checking default account login status..."
curl -s "${BASE_URL}/api/v1/xhs/login/status" | jq '.'
echo ""

# Test 2: Check Alice's account login status
echo "2. Checking Alice's account login status..."
curl -s "${BASE_URL}/api/v1/xhs/login/status?account_id=alice" | jq '.'
echo ""

# Test 3: Check Bob's account login status
echo "3. Checking Bob's account login status..."
curl -s "${BASE_URL}/api/v1/xhs/login/status?account_id=bob" | jq '.'
echo ""

# Test 4: Check browser pool status (should show active browsers)
echo "4. Checking browser pool status..."
curl -s "${BASE_URL}/api/v1/xhs/browsers" | jq '.'
echo ""

# Test 5: Publish to default account (example - commented out to avoid actual posting)
echo "5. Example: Publishing to default account (dry-run)..."
echo "curl -X POST \"${BASE_URL}/api/v1/xhs/publish\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"title\": \"Default Post\", \"content\": \"Posted by default account\", \"images\": []}'"
echo ""

# Test 6: Publish to Alice's account (example - commented out to avoid actual posting)
echo "6. Example: Publishing to Alice's account (dry-run)..."
echo "curl -X POST \"${BASE_URL}/api/v1/xhs/publish?account_id=alice\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"title\": \"Alice Post\", \"content\": \"Posted by Alice\", \"images\": []}'"
echo ""

# Test 7: Publish to Bob's account (example - commented out to avoid actual posting)
echo "7. Example: Publishing to Bob's account (dry-run)..."
echo "curl -X POST \"${BASE_URL}/api/v1/xhs/publish?account_id=bob\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"title\": \"Bob Post\", \"content\": \"Posted by Bob\", \"images\": []}'"
echo ""

# Test 8: Close Alice's browser
echo "8. Example: Closing Alice's browser (dry-run)..."
echo "curl -X DELETE \"${BASE_URL}/api/v1/xhs/browsers/alice\""
echo ""

# Test 9: Check browser pool status again
echo "9. Checking browser pool status after operations..."
curl -s "${BASE_URL}/api/v1/xhs/browsers" | jq '.'
echo ""

echo "================================================"
echo "Test completed!"
echo "================================================"
echo ""
echo "Cookie files are stored in:"
echo "  - cookies.json (default account)"
echo "  - cookies/cookies_alice.json (Alice's account)"
echo "  - cookies/cookies_bob.json (Bob's account)"
echo ""

