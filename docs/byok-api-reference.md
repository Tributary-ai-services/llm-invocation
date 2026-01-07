# BYOK API Reference

This document provides a complete API reference for integrating the user-controlled API keys (BYOK) feature into your server.

## Table of Contents

1. [Overview](#overview)
2. [Request/Response Types](#requestresponse-types)
3. [Client API](#client-api)
4. [Server Integration](#server-integration)
5. [Security Considerations](#security-considerations)
6. [Example Server Implementation](#example-server-implementation)

## Overview

The BYOK system allows users to provide their own encrypted API keys with requests instead of using backend-stored keys. The server needs to:

1. Accept enhanced requests containing encrypted user credentials
2. Decrypt user API keys using the provided password
3. Create temporary provider instances with user keys
4. Process requests normally with the user's provider instance

## Request/Response Types

### EnhancedInvocationRequest

The enhanced request extends the base `InvocationRequest` with user credentials:

```go
type EnhancedInvocationRequest struct {
    *InvocationRequest
    
    // User-provided encrypted credentials
    UserCredentials *UserCredentials `json:"user_credentials,omitempty"`
    
    // Fallback to backend keys if true (default: false)
    UseBackendKeys bool `json:"use_backend_keys,omitempty"`
    
    // Optional security controls
    SecurityOptions *SecurityOptions `json:"security_options,omitempty"`
}
```

### UserCredentials

Contains encrypted API keys for one or more providers:

```go
type UserCredentials struct {
    UserID      string                `json:"user_id"`
    TenantID    string                `json:"tenant_id,omitempty"`
    Providers   []UserProviderConfig  `json:"providers"`
    RequestMeta *RequestMetadata      `json:"request_meta,omitempty"`
}
```

### UserProviderConfig

Configuration for a single provider's encrypted key:

```go
type UserProviderConfig struct {
    Provider       string    `json:"provider"`
    EncryptedKey   []byte    `json:"encrypted_key,omitempty"`
    KeyFingerprint string    `json:"key_fingerprint,omitempty"`
    CreatedAt      time.Time `json:"created_at,omitempty"`
    ExpiresAt      time.Time `json:"expires_at,omitempty"`
}
```

### SecurityOptions

Optional security controls for the request:

```go
type SecurityOptions struct {
    RequireEncryption bool          `json:"require_encryption,omitempty"`
    MaxKeyAge        time.Duration `json:"max_key_age,omitempty"`
    AllowedIPs       []string      `json:"allowed_ips,omitempty"`
    RateLimit        *RateLimit    `json:"rate_limit,omitempty"`
}
```

## Client API

### Creating Encrypted Credentials (Client-Side)

```go
import (
    "github.com/tributary-ai/llm-invocation/auth"
    "github.com/tributary-ai/llm-invocation/types"
)

// Initialize client utilities
keyUtils := auth.NewClientKeyUtils()

// Encrypt API keys with user password
userCredentials, err := keyUtils.CreateUserCredentials(
    "user-123",        // User ID
    "tenant-456",      // Tenant ID (optional)
    map[string]string{
        "openai": "sk-proj-xxxxx",
        "anthropic": "sk-ant-xxxxx",
    },
    "user-password123", // User's password
)

// Create enhanced request
enhancedRequest := &types.EnhancedInvocationRequest{
    InvocationRequest: &types.InvocationRequest{
        Model: "gpt-4",
        Messages: []types.Message{
            {Role: "user", Content: "Hello!"},
        },
    },
    UserCredentials: userCredentials,
    UseBackendKeys: false, // Use user keys, not backend keys
}
```

### Making Requests with User Keys

```go
// Option 1: Synchronous request
response, err := client.InvokeSyncWithCredentials(
    ctx, 
    enhancedRequest, 
    "user-password123", // Same password used for encryption
)

// Option 2: Streaming request
stream, err := client.InvokeWithCredentials(
    ctx,
    enhancedRequest,
    "user-password123",
)
defer stream.Close()

// Process stream chunks
for {
    chunk, err := stream.Next()
    if err != nil {
        break
    }
    // Process chunk...
}
```

## Server Integration

### 1. Initialize Key Manager

```go
import (
    "github.com/tributary-ai/llm-invocation/auth"
)

// Create key manager with options
keyManager := auth.NewRequestKeyManager(
    auth.WithMaxKeyAge(24 * time.Hour),
    auth.WithAuditLogger(auditLogger),
    auth.WithIPValidation(true),
)
```

### 2. Process Enhanced Requests

```go
func handleEnhancedRequest(
    ctx context.Context,
    req *types.EnhancedInvocationRequest,
    userSecret string,
) (*types.InvocationResponse, error) {
    
    // 1. Validate user credentials if provided
    if req.UserCredentials != nil {
        if err := keyManager.ValidateUserCredentials(ctx, req.UserCredentials); err != nil {
            return nil, fmt.Errorf("invalid credentials: %w", err)
        }
    }
    
    // 2. Determine which API key to use
    var apiKey string
    var err error
    
    if req.UserCredentials != nil && !req.UseBackendKeys {
        // Extract user's API key
        provider := req.InvocationRequest.Provider
        if provider == "" {
            provider = determineProviderFromModel(req.InvocationRequest.Model)
        }
        
        apiKey, err = keyManager.ExtractAPIKey(
            ctx,
            req.UserCredentials,
            provider,
            userSecret,
        )
        if err != nil {
            return nil, fmt.Errorf("failed to decrypt user key: %w", err)
        }
    } else {
        // Use backend API key
        apiKey = getBackendAPIKey(provider)
    }
    
    // 3. Create provider with appropriate key
    provider := createProvider(providerName, apiKey)
    
    // 4. Process request normally
    return provider.Invoke(ctx, req.InvocationRequest)
}
```

### 3. HTTP Handler Example

```go
func handleLLMRequest(w http.ResponseWriter, r *http.Request) {
    // Parse request body
    var payload struct {
        Request    *types.EnhancedInvocationRequest `json:"request"`
        UserSecret string                          `json:"user_secret"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Add request metadata to context
    ctx := context.WithValue(r.Context(), "request_metadata", &types.RequestMetadata{
        ClientIP:  r.RemoteAddr,
        UserAgent: r.UserAgent(),
        RequestID: r.Header.Get("X-Request-ID"),
        Timestamp: time.Now(),
    })
    
    // Process request
    response, err := handleEnhancedRequest(ctx, payload.Request, payload.UserSecret)
    if err != nil {
        // Handle error appropriately
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

### 4. Streaming Handler Example

```go
func handleStreamingRequest(w http.ResponseWriter, r *http.Request) {
    // Set up SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }
    
    // Parse request
    var payload struct {
        Request    *types.EnhancedInvocationRequest `json:"request"`
        UserSecret string                          `json:"user_secret"`
    }
    json.NewDecoder(r.Body).Decode(&payload)
    
    // Enable streaming
    payload.Request.InvocationRequest.Stream = &[]bool{true}[0]
    
    // Get stream
    stream, err := client.InvokeWithCredentials(
        r.Context(),
        payload.Request,
        payload.UserSecret,
    )
    if err != nil {
        fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
        flusher.Flush()
        return
    }
    defer stream.Close()
    
    // Stream chunks to client
    for {
        chunk, err := stream.Next()
        if err != nil {
            if err.Error() != "EOF" {
                fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
            }
            break
        }
        
        chunkJSON, _ := json.Marshal(chunk)
        fmt.Fprintf(w, "event: message\ndata: %s\n\n", chunkJSON)
        flusher.Flush()
    }
    
    fmt.Fprintf(w, "event: done\ndata: {}\n\n")
    flusher.Flush()
}
```

## Security Considerations

### 1. Password Handling

**NEVER store or log user passwords!**

```go
// BAD - Don't do this
log.Printf("Processing request with password: %s", userSecret) // NEVER!

// GOOD - Log safe information only
log.Printf("Processing request for user: %s", req.UserCredentials.UserID)
```

### 2. Audit Logging

Implement an audit logger to track key operations:

```go
type AuditLogger struct {
    logger *log.Logger
}

func (a *AuditLogger) LogKeyAccess(ctx context.Context, event *auth.KeyAccessEvent) error {
    // Log security events without sensitive data
    a.logger.Printf(
        "KeyAccess: user=%s provider=%s action=%s success=%v ip=%s",
        event.UserID,
        event.Provider,
        event.Action,
        event.Success,
        event.IPAddress,
    )
    return nil
}

// Use with key manager
keyManager := auth.NewRequestKeyManager(
    auth.WithAuditLogger(&AuditLogger{logger: securityLogger}),
)
```

### 3. Rate Limiting

Implement per-user rate limiting:

```go
func checkRateLimit(userID string, limits *types.RateLimit) error {
    // Check requests per minute
    if getUserRequestCount(userID) > limits.RequestsPerMinute {
        return fmt.Errorf("rate limit exceeded")
    }
    
    // Check tokens per minute
    if getUserTokenCount(userID) > limits.TokensPerMinute {
        return fmt.Errorf("token limit exceeded")
    }
    
    return nil
}
```

### 4. IP Validation

Validate client IP if configured:

```go
func validateClientIP(clientIP string, allowedIPs []string) bool {
    if len(allowedIPs) == 0 {
        return true // No restrictions
    }
    
    for _, allowed := range allowedIPs {
        if matchesIPPattern(clientIP, allowed) {
            return true
        }
    }
    
    return false
}
```

## Example Server Implementation

### Complete HTTP Server

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
    
    llminvocation "github.com/tributary-ai/llm-invocation"
    "github.com/tributary-ai/llm-invocation/auth"
    "github.com/tributary-ai/llm-invocation/types"
)

type Server struct {
    client     llminvocation.Client
    keyManager auth.KeyManager
}

func NewServer() (*Server, error) {
    // Initialize LLM client
    config := &types.Config{
        Providers: map[string]types.ProviderConfig{
            "openai": {
                Enabled: true,
                APIKey:  "", // Will use user keys
            },
            "anthropic": {
                Enabled: true,
                APIKey:  "", // Will use user keys
            },
        },
    }
    
    client, err := llminvocation.NewClient(config)
    if err != nil {
        return nil, err
    }
    
    // Initialize key manager
    keyManager := auth.NewRequestKeyManager(
        auth.WithMaxKeyAge(24 * time.Hour),
    )
    
    return &Server{
        client:     client,
        keyManager: keyManager,
    }, nil
}

// API endpoint for LLM requests
func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Parse request
    var req struct {
        Request    *types.EnhancedInvocationRequest `json:"request"`
        UserSecret string                          `json:"user_secret"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request format", http.StatusBadRequest)
        return
    }
    
    // Validate request
    if req.Request == nil || req.Request.InvocationRequest == nil {
        http.Error(w, "Missing request data", http.StatusBadRequest)
        return
    }
    
    // Add context metadata
    ctx := context.WithValue(r.Context(), "request_metadata", &types.RequestMetadata{
        ClientIP:  r.RemoteAddr,
        UserAgent: r.UserAgent(),
        RequestID: r.Header.Get("X-Request-ID"),
        Timestamp: time.Now(),
    })
    
    // Process request
    response, err := s.client.InvokeSyncWithCredentials(
        ctx,
        req.Request,
        req.UserSecret,
    )
    
    if err != nil {
        // Return error response
        errorResp := map[string]interface{}{
            "error": map[string]string{
                "message": err.Error(),
                "type":    "invocation_error",
            },
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(errorResp)
        return
    }
    
    // Return success response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
        "service": "llm-invocation-server",
    })
}

func main() {
    server, err := NewServer()
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }
    defer server.client.Close()
    
    // Set up routes
    http.HandleFunc("/v1/invoke", server.handleInvoke)
    http.HandleFunc("/health", server.handleHealth)
    
    // Start server
    addr := ":8080"
    log.Printf("Server starting on %s", addr)
    if err := http.ListenAndServe(addr, nil); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

### Request Examples

#### With User Keys

```bash
curl -X POST http://localhost:8080/v1/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "request": {
      "invocation_request": {
        "model": "gpt-4",
        "messages": [
          {"role": "user", "content": "Hello!"}
        ]
      },
      "user_credentials": {
        "user_id": "user-123",
        "providers": [{
          "provider": "openai",
          "encrypted_key": "base64-encrypted-key-data",
          "key_fingerprint": "sha256-fingerprint"
        }]
      },
      "use_backend_keys": false
    },
    "user_secret": "user-password123"
  }'
```

#### With Backend Keys (Fallback)

```bash
curl -X POST http://localhost:8080/v1/invoke \
  -H "Content-Type: application/json" \
  -d '{
    "request": {
      "invocation_request": {
        "model": "gpt-4",
        "messages": [
          {"role": "user", "content": "Hello!"}
        ]
      },
      "use_backend_keys": true
    },
    "user_secret": ""
  }'
```

## Error Handling

Common errors and how to handle them:

```go
switch err := err.(type) {
case *types.Error:
    switch err.Code {
    case types.ErrInvalidRequest:
        // Return 400 Bad Request
    case types.ErrProviderNotFound:
        // Return 404 Not Found
    case types.ErrRateLimitExceeded:
        // Return 429 Too Many Requests
    default:
        // Return 500 Internal Server Error
    }
default:
    // Handle other errors
}
```

## Migration Guide

### Updating Existing Endpoints

1. **Accept Enhanced Requests**: Update request parsing to handle `EnhancedInvocationRequest`
2. **Add Password Parameter**: Include `user_secret` in request payload
3. **Update Client Calls**: Use `InvokeWithCredentials` instead of `Invoke`
4. **Add Error Handling**: Handle decryption and validation errors
5. **Implement Audit Logging**: Track key usage for security

### Backward Compatibility

The system maintains full backward compatibility:

```go
// Old requests still work
if req.UserCredentials == nil {
    // Process as before with backend keys
    return client.Invoke(ctx, req.InvocationRequest)
}

// New requests with user keys
return client.InvokeWithCredentials(ctx, req, userSecret)
```

## Performance Considerations

- **Key Decryption**: ~50ms per operation (PBKDF2 with 100k iterations)
- **Memory Usage**: Minimal - keys are decrypted on demand
- **Caching**: Consider caching decrypted keys per request (not across requests)
- **Connection Pooling**: Reuse provider instances when possible

## Testing

### Unit Tests

```go
func TestUserKeyDecryption(t *testing.T) {
    keyManager := auth.NewRequestKeyManager()
    
    // Create test credentials
    keyUtils := auth.NewClientKeyUtils()
    creds, _ := keyUtils.CreateUserCredentials(
        "test-user",
        "test-tenant",
        map[string]string{"openai": "sk-test"},
        "test-password",
    )
    
    // Test extraction
    apiKey, err := keyManager.ExtractAPIKey(
        context.Background(),
        creds,
        "openai",
        "test-password",
    )
    
    assert.NoError(t, err)
    assert.Equal(t, "sk-test", apiKey)
}
```

### Integration Tests

Test the full flow from client to server with encrypted keys.

## Support

For questions or issues:
- Review the implementation documentation in `docs/`
- Check the example code in `examples/user_keys_basic/` and `examples/user_keys_advanced/`
- Refer to the design document for security analysis

This completes the API reference for server integration. The BYOK system is designed to be secure, performant, and easy to integrate into existing servers.