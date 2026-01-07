# BYOK Server Integration Guide

Quick guide for updating your server to support user-controlled API keys (BYOK).

## What Changed

The client now supports two new methods for requests with user-provided encrypted API keys:
- `InvokeWithCredentials()` - Streaming requests
- `InvokeSyncWithCredentials()` - Synchronous requests

## Request Format

### Old Format (Still Supported)
```json
{
  "model": "gpt-4",
  "messages": [{"role": "user", "content": "Hello"}]
}
```

### New Format (With User Keys)
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
        "encrypted_key": "base64-encrypted-data",
        "key_fingerprint": "sha256-hash"
      }]
    },
    "use_backend_keys": false
  },
  "user_secret": "user-password-for-decryption"
}
```

## Server Changes Required

### 1. Update Request Handler

```go
// Before
func handleRequest(req *types.InvocationRequest) (*types.InvocationResponse, error) {
    return client.InvokeSync(ctx, req)
}

// After
func handleRequest(enhancedReq *types.EnhancedInvocationRequest, userSecret string) (*types.InvocationResponse, error) {
    return client.InvokeSyncWithCredentials(ctx, enhancedReq, userSecret)
}
```

### 2. Update HTTP Endpoint

```go
func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
    // Parse enhanced request format
    var payload struct {
        Request    *types.EnhancedInvocationRequest `json:"request"`
        UserSecret string                          `json:"user_secret"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        // Try parsing old format for backward compatibility
        var oldReq types.InvocationRequest
        if err := json.NewDecoder(r.Body).Decode(&oldReq); err == nil {
            // Convert to enhanced format
            payload.Request = &types.EnhancedInvocationRequest{
                InvocationRequest: &oldReq,
                UseBackendKeys:   true,
            }
            payload.UserSecret = ""
        } else {
            http.Error(w, "Invalid request", http.StatusBadRequest)
            return
        }
    }
    
    // Process with new method
    response, err := s.client.InvokeSyncWithCredentials(
        r.Context(),
        payload.Request,
        payload.UserSecret,
    )
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(response)
}
```

### 3. Update Streaming Handler

```go
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    
    // Parse request (same as above)
    var payload struct {
        Request    *types.EnhancedInvocationRequest `json:"request"`
        UserSecret string                          `json:"user_secret"`
    }
    json.NewDecoder(r.Body).Decode(&payload)
    
    // Get stream with new method
    stream, err := s.client.InvokeWithCredentials(
        r.Context(),
        payload.Request, 
        payload.UserSecret,
    )
    if err != nil {
        fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
        w.(http.Flusher).Flush()
        return
    }
    defer stream.Close()
    
    // Stream chunks (unchanged)
    for {
        chunk, err := stream.Next()
        if err != nil {
            break
        }
        
        data, _ := json.Marshal(chunk)
        fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
        w.(http.Flusher).Flush()
    }
}
```

## Client Configuration

Update your client initialization to support user keys:

```go
// Configure providers without backend keys
config := &types.Config{
    Providers: map[string]types.ProviderConfig{
        "openai": {
            Enabled: true,
            APIKey:  "", // Empty - will use user keys
        },
        "anthropic": {
            Enabled: true,
            APIKey:  "", // Empty - will use user keys
        },
    },
}

client, err := llminvocation.NewClient(config)
```

## Security Best Practices

### 1. Never Log Passwords
```go
// DON'T
log.Printf("Request with password: %s", userSecret)

// DO
log.Printf("Request from user: %s", req.UserCredentials.UserID)
```

### 2. Add Audit Logging
```go
keyManager := auth.NewRequestKeyManager(
    auth.WithAuditLogger(auditLogger),
)
```

### 3. Validate Requests
```go
// Check user credentials if provided
if req.UserCredentials != nil {
    if req.UserCredentials.UserID == "" {
        return errors.New("user_id required with credentials")
    }
}
```

### 4. Handle Errors Appropriately
```go
if err != nil {
    // Don't expose internal errors
    if strings.Contains(err.Error(), "decrypt") {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
    } else {
        http.Error(w, "Request failed", http.StatusInternalServerError)
    }
    
    // Log full error internally
    log.Printf("Request failed: %v", err)
}
```

## Testing

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
          "encrypted_key": "<base64-encrypted-key>",
          "key_fingerprint": "<sha256-hash>"
        }]
      }
    },
    "user_secret": "test-password"
  }'
```

### Test Backward Compatibility
```bash
# Old format should still work
curl -X POST http://localhost:8080/v1/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Test"}]
  }'
```

## Common Issues

### Issue: "Failed to decrypt user key"
- Check that `user_secret` matches the password used for encryption
- Verify the encrypted key data is properly base64 encoded
- Ensure the key hasn't expired (check `expires_at` field)

### Issue: "Provider not found"
- Make sure the provider is enabled in client config
- Check that provider name matches exactly (case-sensitive)

### Issue: Performance degradation
- Key decryption takes ~50ms due to PBKDF2
- Consider implementing request-level caching (not cross-request)
- Monitor for excessive decrypt operations

## Migration Checklist

- [ ] Update request parsing to handle `EnhancedInvocationRequest`
- [ ] Add `user_secret` parameter to request handling
- [ ] Switch to `InvokeWithCredentials` methods
- [ ] Implement backward compatibility for old requests
- [ ] Add error handling for decryption failures
- [ ] Update logging to exclude sensitive data
- [ ] Test with both user keys and backend keys
- [ ] Update API documentation
- [ ] Monitor performance impact

## Need Help?

- Full API Reference: `docs/byok-api-reference.md`
- Implementation Details: `docs/user-keys-implementation.md`
- Examples: `examples/user_keys_basic/` and `examples/user_keys_advanced/`