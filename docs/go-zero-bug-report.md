# Bug Report: JWT Middleware reads entire multipart/form-data causing performance issues with file uploads

## Describe the bug
When using the built-in JWT authentication middleware (`jwt: Auth`) with file upload endpoints, the middleware attempts to read the entire `multipart/form-data` request body (including large file contents) to search for the JWT token. This causes severe performance issues: the entire file content is printed to console, validation takes 5-30 seconds, and memory usage spikes to the file size.

## To Reproduce
Steps to reproduce the behavior:

1. The code is

   ```go
   // core.api
   syntax = "v1"
   
   @server (
       prefix: /api/file
       jwt: Auth  // Using built-in JWT middleware
   )
   service core-api {
       @handler UploadFileHandler
       post /upload (UploadFileRequest) returns (UploadFileResponse)
   }
   
   type UploadFileRequest {
       Hash string `json:"hash,optional"`
       Name string `json:"name,optional"`
       Ext  string `json:"ext,optional"`
       Size int64  `json:"size,optional"`
       Path string `json:"path,optional"`
   }
   
   type UploadFileResponse {
       Identity string `json:"identity"`
   }
   ```
   
   ```go
   // Handler code
   func UploadFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
       return func(w http.ResponseWriter, r *http.Request) {
           file, fileHeader, err := r.FormFile("file")
           if err != nil {
               httpx.ErrorCtx(r.Context(), w, err)
               return
           }
           defer file.Close()
           
           // Process file upload...
       }
   }
   ```
   
   ```bash
   # Upload a large file (e.g., 100MB video) with JWT token in header
   curl -X POST http://localhost:8888/api/file/upload \
     -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
     -F "file=@large_video.mp4"
   ```

2. The error/issue is

   ```
   # Console output (example):
   [Thousands of lines of binary file content printed to console]
   ����JFIF��������....[binary data continues]....
   
   # Response time: 15-30 seconds for a 100MB file
   # Memory usage: Spikes to 100MB+ (file size)
   # Expected: < 1 second response time, minimal memory usage
   ```

## Expected behavior

- JWT middleware should **only check for token in HTTP headers** (`Authorization`, `X-Token`) and query parameters (`?token=xxx`)
- JWT middleware should **NOT parse form data** when Content-Type is `multipart/form-data`
- File upload should complete quickly (within seconds, only limited by network speed)
- Console should remain clean without printing file contents
- Memory usage should be minimal (streaming file upload, not loading entire file to memory)

## Screenshots

**Before (with built-in JWT middleware):**
```
Console output:
����JFIF��������C��....[thousands of lines]....
[Binary file content printed to console during JWT validation]

Response time: 15-30 seconds ❌
Memory usage: 100MB+ (file size) ❌
```

**After (with custom middleware that skips form parsing):**
```
Console output:
[Clean, no binary data]

Response time: < 1 second ✅
Memory usage: < 10MB ✅
```

## Environments

- **OS**: macOS Sonoma 14.x / Linux (Ubuntu 22.04)
- **go-zero version**: v1.6.6 (verified issue exists)
- **goctl version**: 1.9.2
- **Go version**: go1.20+

## More description

### Root Cause Analysis

The JWT middleware in go-zero searches for tokens in multiple locations, including **form parameters**:

```go
// Pseudocode of current implementation (rest/handler/authhandler.go)
func (h *AuthorizeHandler) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Check Authorization header ✅
        token := r.Header.Get("Authorization")
        
        // 2. Check query parameter ✅
        if token == "" {
            token = r.URL.Query().Get("token")
        }
        
        // 3. Check form parameter ❌ THIS IS THE PROBLEM
        if token == "" {
            token = r.FormValue("token")  // <-- Reads entire multipart/form-data!
        }
        
        // Validate token...
    }
}
```

**The Problem**: `r.FormValue("token")` internally calls `r.ParseMultipartForm()`, which parses the **entire** `multipart/form-data` request body, including all file uploads (potentially hundreds of MBs), just to check if there's a `token` field in the form.

### Impact

This bug makes it **impractical to use go-zero's JWT middleware with file upload endpoints** in production:
- ❌ Large file uploads (videos, archives, datasets) become extremely slow
- ❌ High memory usage causes OOM in containerized environments
- ❌ Console pollution makes debugging impossible
- ❌ Poor user experience (30+ second upload times)
- ❌ Cannot handle concurrent file uploads

### Suggested Solutions

**Option 1: Skip Form Parsing for multipart/form-data (Recommended)**

```go
func (h *AuthorizeHandler) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        
        if token == "" {
            token = r.URL.Query().Get("token")
        }
        
        // Only check form if NOT multipart/form-data
        contentType := r.Header.Get("Content-Type")
        if token == "" && !strings.Contains(contentType, "multipart/form-data") {
            token = r.FormValue("token")
        }
        
        // Continue validation...
    }
}
```

**Option 2: Add Configuration Option**

```yaml
# config.yaml
Auth:
  AccessSecret: xxx
  AccessExpire: 36000
  SkipFormParsing: true  # New option to skip form parsing
```

**Option 3: Provide Lightweight JWT Middleware for File Uploads**

Provide a separate middleware (e.g., `jwt-lite: Auth`) that only checks headers and query parameters.

### Current Workaround

Users must implement a custom JWT middleware:

```go
// Custom middleware that skips form parsing
type FileAuthMiddleware struct {
    accessSecret string
    accessExpire int64
}

func (m *FileAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        token = strings.TrimPrefix(token, "Bearer ")
        
        if token == "" {
            token = r.Header.Get("X-Token")
        }
        
        if token == "" {
            token = r.URL.Query().Get("token")
        }
        
        // Skip r.FormValue() entirely!
        
        claims, err := ParseToken(token, m.accessSecret, m.accessExpire)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }
        
        // Store user info in context
        ctx := context.WithValue(r.Context(), "user_id", claims.Id)
        r = r.WithContext(ctx)
        
        next(w, r)
    }
}
```

Then use custom middleware instead of built-in JWT:
```go
@server (
    prefix: /api/file
    middleware: FileAuthMiddleware  // Custom middleware
)
```

### Performance Comparison

| Metric | Built-in JWT | Custom Middleware | Improvement |
|--------|--------------|-------------------|-------------|
| Auth time | 5-30 seconds | < 100ms | **99%+** |
| Memory usage | File size | < 1MB | **Significant** |
| Console output | Binary data | Clean | ✅ |
| Concurrent uploads | ❌ Breaks | ✅ Works | ✅ |

### Related Issues

Similar problems exist in other frameworks:
- Express.js: Middleware order matters for multipart parsing
- Django: Custom authentication recommended for file uploads
- Spring Boot: MultipartResolver configuration needed

### Test Case

```go
func TestJWTMiddlewareWithFileUpload(t *testing.T) {
    // Setup test server with JWT middleware
    // Upload a 100MB file with valid JWT token in header
    // Assert:
    // 1. Request completes in < 5 seconds
    // 2. Memory usage < 10MB (not loading file to memory)
    // 3. No binary data printed to console
}
```

---

**I'm willing to submit a PR to fix this issue** if the maintainers agree on the approach. This significantly impacts go-zero's usability for file-heavy applications.

### References
- Go-Zero Documentation: https://go-zero.dev/
- Current JWT Implementation: `rest/handler/authhandler.go`
- HTTP Multipart Parsing: https://pkg.go.dev/net/http#Request.FormValue
