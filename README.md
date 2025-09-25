# XHS Poster

A web server that provides HTTP REST API for posting content to Xiaohongshu (小红书).

## Features

- **Login Status Check**: Check if you're logged into Xiaohongshu
- **Content Publishing**: Post images and text content to Xiaohongshu
- **HTTP REST API**: Simple and intuitive REST interface
- **Image Processing**: Support for both URL images and local file paths
- **Tag Support**: Add hashtags to your posts

## Quick Start

### 1. Build the Project

```bash
go mod tidy
go build -o xhs-poster .
```

### 2. Run the Server

```bash
# Run with default settings (HTTP on :6170)
./xhs-poster

# Run with custom port  
./xhs-poster -http-port=:8080

# Run with visible browser (for debugging)
./xhs-poster -headless=false
```

**🚀 智能自动登录**: 服务采用按需登录策略：
1. 启动HTTP服务器
2. 当访问需要认证的API时，自动检查登录状态
3. 如果未登录，立即触发登录流程并显示二维码
4. 扫码完成后，请求继续正常处理

### 3. Login System

The application features an intelligent login system that works in both headless and non-headless modes:

#### Headless Mode Login (Recommended)
- Automatically displays QR codes in the terminal
- Saves QR code image to `qrcode_login.png` 
- Shows detailed scanning instructions
- Supports cookie persistence for automatic re-login

#### Manual Browser Login
- Run with `-headless=false` for visual browser login
- Traditional browser-based QR code scanning

#### Login Process
1. The system first checks for saved cookies
2. If not logged in, triggers QR code display
3. Provides scanning instructions and saves QR image
4. Waits for user to scan with Xiaohongshu mobile app
5. Automatically saves login session for future use

## Testing

We provide comprehensive test scripts to verify the posting functionality:

### Quick Test
```bash
# Simple test script - posts a single test message
./quick_test_post.sh
```

### Comprehensive Test Suite
```bash
# Full test suite with multiple scenarios
./test_poster.sh

# Test only HTTP endpoints
./test_poster.sh --http-only

# Test only gRPC endpoints  
./test_poster.sh --grpc-only

# Skip optional tests
./test_poster.sh --no-local --no-errors --no-performance

# Show help
./test_poster.sh --help
```

### Test Features
- ✅ **HTTP & gRPC API Testing**: Tests both REST and gRPC endpoints
- ✅ **Login Status Verification**: Ensures user is authenticated before testing
- ✅ **Multiple Image Sources**: Tests both URL images and local files
- ✅ **Error Handling**: Validates error responses for invalid requests
- ✅ **Performance Metrics**: Measures API response times
- ✅ **Unicode & Emoji Support**: Tests Chinese characters and emojis
- ✅ **Tag System**: Tests hashtag functionality

### Prerequisites for Testing
- Server running and user logged in
- `grpcurl` for gRPC tests (optional): `go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest`
- `ImageMagick` for local image tests (optional): `sudo apt install imagemagick`

## API Usage

### Authentication System

The API uses intelligent auto-login with session-based authentication. Protected endpoints automatically trigger login when needed.

**Public Endpoints** (no authentication required):
- `GET /health` - Health check
- `GET /api/v1/login/status` - Check login status
- `POST /api/v1/login` - Manual login (optional)

**Auto-Login Endpoints** (automatically trigger login if needed):
- `POST /api/v1/publish` - Publish content (auto-login on first access)

### HTTP REST API

#### Check Login Status
```bash
curl -X GET http://localhost:8080/api/v1/login/status
```

#### Login to Xiaohongshu
```bash
# Triggers QR code login process
curl -X POST http://localhost:8080/api/v1/login

# This will:
# 1. Display QR code instructions in the server console
# 2. Save QR code image to qrcode_login.png
# 3. Wait for mobile app scanning (up to 5 minutes)
# 4. Return success/failure status
```

#### Publish Content (Protected)
```bash
# This will fail if not logged in (returns 401)
curl -X POST http://localhost:8080/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{
    "title": "我的测试标题",
    "content": "这是测试内容 #测试标签",
    "images": [
      "https://example.com/image.jpg"
    ],
    "tags": ["测试标签", "API"]
  }'
```

#### Authentication Flow Example
```bash
# 1. Check if logged in
curl http://localhost:8080/api/v1/login/status

# 2. If not logged in, login first
curl -X POST http://localhost:8080/api/v1/login

# 3. Now you can access protected endpoints
curl -X POST http://localhost:8080/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{"title": "Test", "content": "Content", "images": ["url"], "tags": []}'
```

### gRPC API

See the example client in `examples/grpc_client.go`:

```bash
# Run the example gRPC client
go run examples/grpc_client.go
```

## API Specification

### HTTP Endpoints

- `GET /health` - Health check
- `GET /api/v1/login/status` - Check login status
- `POST /api/v1/publish` - Publish content

### gRPC Methods

- `CheckLoginStatus()` - Check login status
- `PublishContent()` - Publish content

### Request/Response Formats

#### Publish Content Request
```json
{
  "title": "Post title (max 40 characters)",
  "content": "Post content",
  "images": ["image_url_or_path"],
  "tags": ["tag1", "tag2"]
}
```

#### Publish Content Response
```json
{
  "success": true,
  "data": {
    "title": "Post title",
    "content": "Post content", 
    "images": 1,
    "status": "发布完成"
  },
  "message": "发布成功"
}
```

## Configuration

### Command Line Options

- `-headless`: Run browser in headless mode (default: true)
- `-bin`: Custom browser binary path
- `-http-port`: HTTP server port (default: :8080)
- `-grpc-port`: gRPC server port (default: :9090)

### Image Support

The service supports two types of image inputs:

1. **HTTP/HTTPS URLs**: Images will be downloaded automatically
   ```json
   ["https://example.com/image1.jpg", "https://example.com/image2.png"]
   ```

2. **Local file paths**: Direct file paths (recommended for better performance)
   ```json
   ["/path/to/image1.jpg", "/path/to/image2.png"]
   ```

## Notes

- **Title Length**: Xiaohongshu limits titles to 40 character units (Chinese characters count as 2 units)
- **Login Persistence**: Login cookies are automatically saved and reused
- **Single Session**: Only one browser session per account is allowed
- **Rate Limiting**: Be mindful of Xiaohongshu's posting limits

## Architecture

```
┌─────────────────┐    ┌─────────────────┐
│   HTTP Client   │    │   gRPC Client   │
└─────────┬───────┘    └─────────┬───────┘
          │                      │
          ▼                      ▼
┌─────────────────────────────────────────┐
│            XHS Poster Server            │
│  ┌─────────────┐    ┌─────────────────┐ │
│  │ HTTP Server │    │   gRPC Server   │ │
│  └─────────────┘    └─────────────────┘ │
│           │                  │           │
│           └──────────┬───────┘           │
│                      ▼                   │
│              ┌─────────────┐             │
│              │ XHS Service │             │
│              └─────────────┘             │
│                      │                   │
│                      ▼                   │
│      ┌─────────────────────────────────┐ │
│      │     Browser Automation          │ │
│      │  (Login, Publish, etc.)         │ │
│      └─────────────────────────────────┘ │
└─────────────────────────────────────────┘
                       │
                       ▼
              ┌─────────────────┐
              │   Xiaohongshu   │
              │   (小红书)       │
              └─────────────────┘
```

## Development

### Adding New Features

1. Define new methods in `proto/xhs.proto`
2. Regenerate gRPC code: `protoc --go_out=. --go-grpc_out=. proto/xhs.proto`
3. Implement methods in both HTTP and gRPC servers
4. Update documentation

### Testing

```bash
# Test HTTP API
curl -X GET http://localhost:8080/health

# Test gRPC API
go run examples/grpc_client.go
```
