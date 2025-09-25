#!/bin/bash

echo "Testing XHS Poster Servers..."

# Test HTTP server health
echo "1. Testing HTTP server health..."
HTTP_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health)
if [ "$HTTP_RESPONSE" = "200" ]; then
    echo "✅ HTTP server is running on port 8080"
else
    echo "❌ HTTP server is not responding (got HTTP $HTTP_RESPONSE)"
fi

# Test HTTP login status
echo "2. Testing HTTP login status endpoint..."
curl -s -X GET http://localhost:8080/api/v1/login/status | jq '.'

# Test gRPC server (requires grpcurl to be installed)
echo "3. Testing gRPC server availability..."
if command -v grpcurl &> /dev/null; then
    echo "Testing gRPC server with grpcurl..."
    grpcurl -plaintext localhost:9090 list
else
    echo "ℹ️  grpcurl not installed. Install with: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
    echo "To test gRPC manually, run: go run examples/grpc_client.go"
fi

echo "4. Server status summary:"
echo "   HTTP API: http://localhost:8080"
echo "   gRPC API: localhost:9090"
echo "   Health Check: http://localhost:8080/health"
