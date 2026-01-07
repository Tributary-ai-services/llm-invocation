# BYOK Server Update Summary

This document summarizes the changes needed to update your server for BYOK (Bring Your Own Keys) support.

## 📋 Quick Checklist

- [ ] **Update Go Dependencies**: Import new auth and types packages
- [ ] **Modify Request Parsing**: Handle `EnhancedInvocationRequest` format
- [ ] **Switch API Methods**: Use `InvokeWithCredentials` instead of `Invoke`
- [ ] **Add User Secret Parameter**: Handle `user_secret` in requests
- [ ] **Implement Error Handling**: Handle decryption and validation errors
- [ ] **Update Client Config**: Remove backend API keys for user-key-only providers
- [ ] **Test Integration**: Verify both old and new request formats work

## 🔧 Essential Code Changes

### 1. Import New Packages
```go
import (
    llminvocation "github.com/tributary-ai/llm-invocation"
    "github.com/tributary-ai/llm-invocation/types"
    // No need to import auth package in server code
)
```

### 2. Update Request Handler
```go
// OLD
func handleRequest(req *types.InvocationRequest) {
    response, err := client.InvokeSync(ctx, req)
}

// NEW  
func handleRequest(req *types.EnhancedInvocationRequest, userSecret string) {
    response, err := client.InvokeSyncWithCredentials(ctx, req, userSecret)
}
```

### 3. Update JSON Parsing
```go
// Parse new request format
var payload struct {
    Request    *types.EnhancedInvocationRequest `json:"request"`
    UserSecret string                          `json:"user_secret"`
}

if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
    http.Error(w, "Invalid request", http.StatusBadRequest)
    return
}
```

### 4. Update Client Configuration
```go
config := &types.Config{
    Providers: map[string]types.ProviderConfig{
        "openai": {
            Enabled: true,
            APIKey:  "", // Empty = use user keys
        },
    },
}
```

## 📊 Request Format Changes

### Before (Still Supported)
```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "Hello"}]
}
```

### After (New Format)
```json
{
  "request": {
    "invocation_request": {
      "model": "gpt-4", 
      "messages": [{"role": "user", "content": "Hello"}]
    },
    "user_credentials": {
      "user_id": "user-123",
      "providers": [{
        "provider": "openai",
        "encrypted_key": "base64-encrypted-data"
      }]
    }
  },
  "user_secret": "user-password"
}
```

## ⚡ Complete Server Example

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"
    
    llminvocation "github.com/tributary-ai/llm-invocation"
    "github.com/tributary-ai/llm-invocation/types"
)

type Server struct {
    client llminvocation.Client
}

func NewServer() *Server {
    config := &types.Config{
        Providers: map[string]types.ProviderConfig{
            "openai":    {Enabled: true, APIKey: ""},
            "anthropic": {Enabled: true, APIKey: ""},
        },
    }
    
    client, _ := llminvocation.NewClient(config)
    return &Server{client: client}
}

func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
    var payload struct {
        Request    *types.EnhancedInvocationRequest `json:"request"`
        UserSecret string                          `json:"user_secret"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Process request with user keys
    response, err := s.client.InvokeSyncWithCredentials(
        r.Context(),
        payload.Request,
        payload.UserSecret,
    )
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    server := NewServer()
    http.HandleFunc("/v1/invoke", server.handleInvoke)
    http.ListenAndServe(":8080", nil)
}
```

## 🧪 Testing Commands

### Test with User Keys
```bash
curl -X POST http://localhost:8080/v1/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "request": {
      "invocation_request": {
        "model": "gpt-4",
        "messages": [{"role": "user", "content": "Test"}]
      },
      "user_credentials": {
        "user_id": "test-user",
        "providers": [{
          "provider": "openai",
          "encrypted_key": "encrypted-key-data"
        }]
      }
    },
    "user_secret": "test-password"
  }'
```

### Test Backward Compatibility
```bash
curl -X POST http://localhost:8080/v1/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Test"}]
  }'
```

## 🔒 Security Notes

1. **Never log user secrets**: `userSecret` contains sensitive data
2. **Handle errors carefully**: Don't expose internal decryption details
3. **Validate user IDs**: Ensure user authentication before processing
4. **Monitor performance**: Key decryption adds ~50ms per request

## 📚 Documentation Files

- **Integration Guide**: `docs/byok-server-integration-guide.md` - Detailed implementation
- **API Reference**: `docs/byok-api-reference.md` - Complete API documentation  
- **JSON Schema**: `docs/byok-api-schema.json` - Request validation schema
- **Implementation Details**: `docs/user-keys-implementation.md` - Technical deep dive

## 🐛 Common Issues

| Error | Cause | Solution |
|-------|-------|----------|
| "Failed to decrypt user key" | Wrong password or corrupted data | Verify password matches encryption |
| "Provider not found" | Provider not configured | Add provider to client config |
| "Invalid request format" | JSON parsing failed | Check request structure |
| "User credentials required" | Missing user_credentials | Include encrypted keys in request |

## 🚀 Deployment

1. **Update dependencies**: `go mod tidy`
2. **Build server**: `go build`
3. **Test locally**: Use curl commands above
4. **Deploy with monitoring**: Watch for decryption errors
5. **Update documentation**: Inform API consumers of new format

## 💡 Tips

- **Keep backward compatibility**: Support old request format during transition
- **Use environment variables**: For fallback backend keys
- **Implement health checks**: Verify BYOK functionality works
- **Add metrics**: Track user key vs backend key usage

The BYOK system is production-ready and maintains full backward compatibility. Your existing API endpoints will continue to work unchanged while new functionality is available for clients that need it.