#!/bin/bash

echo "Testing XHS Poster Authentication Flow..."

API_BASE="http://localhost:8080/api/v1"

echo "=== 1. Check if servers are running ==="
curl -s $API_BASE/../health | jq '.'

echo -e "\n=== 2. Check login status ==="
LOGIN_STATUS=$(curl -s $API_BASE/login/status)
echo $LOGIN_STATUS | jq '.'

IS_LOGGED_IN=$(echo $LOGIN_STATUS | jq -r '.data.is_logged_in')

echo -e "\n=== 3. Test protected route without login ==="
if [ "$IS_LOGGED_IN" = "false" ]; then
    echo "User is not logged in. Testing protected route (should fail)..."
    curl -s -X POST $API_BASE/publish \
        -H "Content-Type: application/json" \
        -d '{
            "title": "Test Post",
            "content": "This should fail",
            "images": ["https://example.com/test.jpg"],
            "tags": ["test"]
        }' | jq '.'
else
    echo "User is already logged in. Skipping auth test."
fi

echo -e "\n=== 4. Login instruction ==="
if [ "$IS_LOGGED_IN" = "false" ]; then
    echo "To login, call:"
    echo "curl -X POST $API_BASE/login"
    echo ""
    echo "Note: This will open a browser window for QR code scanning"
    echo "After login, you can test the publish endpoint again"
else
    echo "✅ User is already logged in and can use protected endpoints"
    echo -e "\n=== 5. Test protected route with login ==="
    curl -s -X POST $API_BASE/publish \
        -H "Content-Type: application/json" \
        -d '{
            "title": "Authenticated Test",
            "content": "This should work #authenticated",
            "images": ["https://picsum.photos/800/600"],
            "tags": ["authenticated", "test"]
        }' | jq '.'
fi

echo -e "\n=== Authentication Flow Summary ==="
echo "1. Public endpoints: /health, /login/status, /login"
echo "2. Protected endpoints: /publish (requires authentication)"
echo "3. Authentication is checked via middleware before accessing protected routes"
echo "4. If not logged in, requests return 401 with login instructions"
