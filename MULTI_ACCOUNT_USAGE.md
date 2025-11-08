# Multi-Account Browser Pool - Usage Guide

## Overview

The sns-poster now supports managing multiple account sessions simultaneously using a **Browser Pool** architecture. Each account gets its own isolated browser instance with separate cookie storage.

## Architecture

### Key Components

1. **BrowserPool** (`internal/utils/browser_pool.go`)
   - Manages multiple browser instances
   - Each account ID maps to a dedicated browser
   - Automatic connection management and recovery

2. **Account-Specific Cookies** (`internal/utils/cookies.go`)
   - Cookie files stored per account: `cookies/cookies_<account_id>.json`
   - Default account uses: `cookies.json`

3. **Enhanced XHS Service** (`internal/xhs/xhs_service.go`)
   - All methods now accept `accountID` parameter
   - Concurrent operations on different accounts

## API Usage

### 1. Check Login Status

Check if a specific account is logged in:

```bash
# Default account
curl "http://localhost:8080/api/v1/xhs/login/status"

# Specific account
curl "http://localhost:8080/api/v1/xhs/login/status?account_id=alice"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "is_logged_in": true,
    "username": "alice",
    "account_id": "alice"
  },
  "message": "检查XHS登录状态成功"
}
```

### 2. Login

Login to a specific account:

```bash
# Default account
curl -X POST "http://localhost:8080/api/v1/xhs/login"

# Specific account (alice)
curl -X POST "http://localhost:8080/api/v1/xhs/login?account_id=alice"

# Another account (bob)
curl -X POST "http://localhost:8080/api/v1/xhs/login?account_id=bob"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "success": true,
    "message": "登录成功"
  },
  "message": "XHS登录成功"
}
```

### 3. Publish Content

Publish content using a specific account:

```bash
# Default account
curl -X POST "http://localhost:8080/api/v1/xhs/publish" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My Post",
    "content": "Post content here",
    "images": ["path/to/image1.jpg", "path/to/image2.jpg"]
  }'

# Specific account (alice)
curl -X POST "http://localhost:8080/api/v1/xhs/publish?account_id=alice" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Alice Post",
    "content": "Posted by Alice",
    "images": ["path/to/image1.jpg"]
  }'

# Another account (bob)
curl -X POST "http://localhost:8080/api/v1/xhs/publish?account_id=bob" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Bob Post",
    "content": "Posted by Bob",
    "images": ["path/to/image2.jpg"]
  }'
```

### 4. Browser Pool Management

#### Get Active Browsers

Check how many browsers are currently active:

```bash
curl "http://localhost:8080/api/v1/xhs/browsers"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "active_browsers": 2,
    "active_accounts": ["alice", "bob"]
  },
  "message": "获取浏览器池状态成功"
}
```

#### Close Specific Browser

Close a specific account's browser to free resources:

```bash
curl -X DELETE "http://localhost:8080/api/v1/xhs/browsers/alice"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "account_id": "alice",
    "closed": true
  },
  "message": "账号 alice 的浏览器已关闭"
}
```

## Cookie File Structure

Cookies are stored in the following locations:

```
project-root/
├── cookies.json                    # Default account (backward compatible)
└── cookies/
    ├── cookies_alice.json          # Alice's account cookies
    ├── cookies_bob.json            # Bob's account cookies
    └── cookies_charlie.json        # Charlie's account cookies
```

## Concurrent Usage Example

You can publish to multiple accounts simultaneously:

```bash
# Terminal 1: Publish as Alice
curl -X POST "http://localhost:8080/api/v1/xhs/publish?account_id=alice" \
  -H "Content-Type: application/json" \
  -d '{"title": "Alice Post", "content": "..."}'

# Terminal 2: Publish as Bob (runs concurrently)
curl -X POST "http://localhost:8080/api/v1/xhs/publish?account_id=bob" \
  -H "Content-Type: application/json" \
  -d '{"title": "Bob Post", "content": "..."}'
```

## Best Practices

### 1. Account ID Naming

- Use simple, descriptive names: `alice`, `bob`, `account1`
- Avoid special characters to ensure file system compatibility
- Keep it consistent across your application

### 2. Resource Management

- **Monitor active browsers**: Use `/api/v1/xhs/browsers` to check status
- **Close unused browsers**: Use DELETE endpoint to free resources
- **Automatic cleanup**: Browsers are closed when service shuts down

### 3. Session Management

Each account maintains its own:
- ✅ Login session (separate cookies)
- ✅ Browser state
- ✅ Independent publishing capabilities

### 4. Backward Compatibility

The system remains **fully backward compatible**:
- Omitting `account_id` uses the default account
- Existing code without account IDs continues to work
- Default account uses original `cookies.json` file

## Migration from Single Account

If you're upgrading from single-account usage:

### Before (Single Account):
```bash
curl -X POST "http://localhost:8080/api/v1/xhs/publish" \
  -H "Content-Type: application/json" \
  -d '{"title": "My Post", "content": "..."}'
```

### After (Multi-Account):
```bash
# Still works (uses default account)
curl -X POST "http://localhost:8080/api/v1/xhs/publish" \
  -H "Content-Type: application/json" \
  -d '{"title": "My Post", "content": "..."}'

# NEW: Use specific account
curl -X POST "http://localhost:8080/api/v1/xhs/publish?account_id=alice" \
  -H "Content-Type: application/json" \
  -d '{"title": "Alice Post", "content": "..."}'
```

## Performance Considerations

### Memory Usage
- Each browser instance uses ~100-200MB RAM
- Monitor system resources when running many concurrent sessions
- Close inactive browsers to free memory

### Connection Pooling
- Browsers are created lazily (on first use)
- Connections are automatically verified and reconnected if needed
- Stale connections are detected and rebuilt

### Scalability
```
Recommended limits:
- 1-5 accounts: Optimal performance
- 5-10 accounts: Good performance, monitor resources
- 10+ accounts: Consider horizontal scaling or sequential processing
```

## Troubleshooting

### Issue: Browser connection fails
**Solution:** 
```bash
# Check Docker container status
docker ps | grep xhs-poster-rod

# Restart if needed
docker restart xhs-poster-rod
```

### Issue: Cookie file not found
**Solution:** Login first for each account:
```bash
curl -X POST "http://localhost:8080/api/v1/xhs/login?account_id=alice"
```

### Issue: Too many open browsers
**Solution:** Close unused browsers:
```bash
# Check active browsers
curl "http://localhost:8080/api/v1/xhs/browsers"

# Close specific ones
curl -X DELETE "http://localhost:8080/api/v1/xhs/browsers/alice"
```

## Code Examples

### Go Client Example

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

func publishToMultipleAccounts() {
    accounts := []string{"alice", "bob", "charlie"}
    
    for _, account := range accounts {
        go func(accountID string) {
            url := "http://localhost:8080/api/v1/xhs/publish?account_id=" + accountID
            
            payload := map[string]interface{}{
                "title":   "Post by " + accountID,
                "content": "Content for " + accountID,
                "images":  []string{"image.jpg"},
            }
            
            data, _ := json.Marshal(payload)
            resp, _ := http.Post(url, "application/json", bytes.NewReader(data))
            defer resp.Body.Close()
        }(account)
    }
}
```

### Python Client Example

```python
import requests
import concurrent.futures

def publish_post(account_id, title, content):
    url = f"http://localhost:8080/api/v1/xhs/publish?account_id={account_id}"
    payload = {
        "title": title,
        "content": content,
        "images": ["image.jpg"]
    }
    
    response = requests.post(url, json=payload)
    return response.json()

# Publish to multiple accounts concurrently
accounts = ["alice", "bob", "charlie"]
with concurrent.futures.ThreadPoolExecutor() as executor:
    futures = [
        executor.submit(publish_post, acc, f"Post by {acc}", f"Content for {acc}")
        for acc in accounts
    ]
    
    for future in concurrent.futures.as_completed(futures):
        print(future.result())
```

## Summary

The Browser Pool implementation provides:

- ✅ **Multi-account support**: Manage multiple XHS accounts simultaneously
- ✅ **Isolated sessions**: Each account has its own browser and cookies
- ✅ **Concurrent operations**: Publish to multiple accounts at once
- ✅ **Resource management**: Monitor and control active browsers
- ✅ **Backward compatible**: Existing single-account code still works
- ✅ **Simple API**: Just add `?account_id=<name>` to any endpoint

For questions or issues, please refer to the project documentation or open an issue on GitHub.

