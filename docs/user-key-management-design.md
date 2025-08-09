# User-Controlled API Key Management Design

## Overview

This document outlines the design for enabling users to bring their own API keys (BYOK) for LLM providers, moving away from backend-stored keys to user-controlled, secure key management.

## Current State

- API keys are stored in backend configuration
- Single set of keys shared across all users  
- Keys managed by service operators only
- No user-level key isolation

## Goals

1. **User Control**: Users can provide their own API keys
2. **Security**: Keys are never stored in plaintext or logs
3. **Flexibility**: Multiple storage and access patterns
4. **Compliance**: Meet enterprise security requirements
5. **Backward Compatibility**: Existing backend keys still work
6. **Audit Trail**: Track key usage and access

## Security Requirements

### Threat Model

**Threats to Mitigate**:
- Key exposure in logs, databases, or memory dumps
- Man-in-the-middle attacks during key transmission
- Unauthorized access to user keys
- Key leakage through error messages or debugging
- Replay attacks using intercepted keys

**Security Boundaries**:
- Keys should never be stored in plaintext
- Keys should be encrypted in transit and at rest
- Access should be auditable and time-limited
- Keys should be isolated per user/tenant

### Compliance Requirements

- **SOC 2**: Secure key handling and access controls
- **GDPR**: User control over their data (keys)
- **HIPAA**: Encryption and audit requirements (if applicable)
- **Enterprise**: Role-based access, audit trails

## Design Options

### Option 1: KMS-Based Storage

#### Architecture
```
User → API Gateway → KMS → Encrypted Key → LLM Provider
```

#### Implementation
```go
type KMSKeyManager interface {
    StoreKey(userID, provider, keyID string, encryptedKey []byte) error
    GetKey(userID, provider, keyID string) ([]byte, error)
    RotateKey(userID, provider, keyID string) error
    RevokeKey(userID, provider, keyID string) error
    AuditKeyAccess(userID, provider, keyID string) ([]AuditEntry, error)
}

type AWSKMSManager struct {
    kmsClient *kms.Client
    keyARN    string
    audit     AuditLogger
}

type HashiCorpVaultManager struct {
    vaultClient *vault.Client
    mountPath   string
    audit       AuditLogger
}

type AzureKeyVaultManager struct {
    kvClient *keyvault.BaseClient
    vaultURL string
    audit    AuditLogger
}
```

**Pros**:
- Enterprise-grade security
- Native cloud integration
- Automatic encryption/decryption
- Built-in audit logging
- Key rotation support
- Hardware Security Module (HSM) backing

**Cons**:
- Additional infrastructure dependency
- Cloud vendor lock-in potential
- Higher complexity
- Cost implications
- Latency for key retrieval

#### Supported KMS Providers

1. **AWS KMS**
   - Use AWS KMS for encryption
   - Store in DynamoDB or RDS with encryption
   - IAM-based access control

2. **HashiCorp Vault**
   - Dynamic secrets support
   - Built-in audit logging
   - Policy-based access
   - Multi-cloud support

3. **Azure Key Vault**
   - Managed HSM support
   - RBAC integration
   - Audit logging

4. **Google Cloud KMS**
   - Cloud HSM integration
   - IAM integration
   - Audit logs

### Option 2: Secure Request Embedding

#### Architecture
```
User → Encrypted Request → API Gateway → Decrypt → LLM Provider
```

#### Implementation
```go
type SecureRequestManager interface {
    EncryptRequest(req *InvocationRequest, userPublicKey []byte) ([]byte, error)
    DecryptRequest(encryptedReq []byte, serverPrivateKey []byte) (*InvocationRequest, error)
    ValidateRequest(req *InvocationRequest) error
}

type E2EEncryptionManager struct {
    serverPrivateKey []byte
    keyDerivation    KeyDerivationFunc
    cipher          CipherSuite
}

// Request with embedded encrypted credentials
type SecureInvocationRequest struct {
    *InvocationRequest
    
    // Encrypted credentials (never logged)
    EncryptedCredentials []byte `json:"encrypted_credentials"`
    
    // Key derivation info
    KeyDerivation KeyDerivationInfo `json:"key_derivation"`
    
    // Request signature for integrity
    Signature []byte `json:"signature"`
    
    // Timestamp for replay protection  
    Timestamp int64 `json:"timestamp"`
    Nonce     []byte `json:"nonce"`
}
```

**Pros**:
- No backend key storage
- End-to-end encryption
- Simple infrastructure
- No external dependencies
- Perfect forward secrecy possible

**Cons**:
- Keys in every request (bandwidth)
- Client-side key management burden
- More complex client implementation
- Potential for key exposure in client

### Option 3: JWT-Based Key Exchange

#### Architecture
```
User → JWT Token → API Gateway → Validate JWT → Extract Key → LLM Provider
```

#### Implementation
```go
type JWTKeyManager interface {
    CreateKeyToken(userID, provider string, apiKey []byte, expiry time.Duration) (string, error)
    ValidateKeyToken(token string) (*KeyClaims, error)
    RevokeKeyToken(tokenID string) error
}

type KeyClaims struct {
    UserID    string            `json:"user_id"`
    Provider  string            `json:"provider"` 
    KeyHash   string            `json:"key_hash"` // Hash of actual key
    ExpiresAt int64             `json:"exp"`
    IssuedAt  int64             `json:"iat"`
    TokenID   string            `json:"jti"`
    
    // Encrypted key payload
    EncryptedKey []byte `json:"enc_key"`
}

// Request with JWT token
type JWTInvocationRequest struct {
    *InvocationRequest
    
    // JWT containing encrypted API key
    KeyToken string `json:"key_token"`
}
```

**Pros**:
- Standard JWT security
- Built-in expiry and revocation
- Stateless validation
- Good tooling ecosystem
- Audit trail via token ID

**Cons**:
- Token management complexity
- Requires secure key distribution
- JWT size limitations
- Clock synchronization needs

### Option 4: Session-Based Key Management

#### Architecture
```
User → Auth + Store Key → Session Token → API Gateway → Session Lookup → LLM Provider
```

#### Implementation
```go
type SessionKeyManager interface {
    CreateSession(userID string, keys map[string][]byte) (*Session, error)
    GetSessionKeys(sessionID string) (map[string][]byte, error)
    ExtendSession(sessionID string, duration time.Duration) error
    RevokeSession(sessionID string) error
}

type Session struct {
    ID          string                 `json:"id"`
    UserID      string                 `json:"user_id"`  
    Keys        map[string][]byte      `json:"keys"`        // provider -> encrypted key
    CreatedAt   time.Time              `json:"created_at"`
    ExpiresAt   time.Time              `json:"expires_at"`
    LastUsedAt  time.Time              `json:"last_used_at"`
    Metadata    map[string]interface{} `json:"metadata"`
}

// Request with session reference
type SessionInvocationRequest struct {
    *InvocationRequest
    
    // Session containing user's keys
    SessionToken string `json:"session_token"`
}
```

**Pros**:
- Familiar session model
- Efficient key reuse
- Good performance
- Simple client integration

**Cons**:
- Stateful server requirement
- Session storage and cleanup
- Scalability considerations

## Hybrid Approach (Recommended)

Combine multiple options for maximum flexibility:

```go
type UserKeyManager interface {
    // KMS-based long-term storage
    StoreUserKey(userID, provider, keyID string, key []byte) error
    
    // Session-based temporary access
    CreateKeySession(userID string, providers []string) (*KeySession, error)
    
    // Direct encrypted embedding  
    EncryptKeyForRequest(userID, provider string, key []byte) ([]byte, error)
    
    // JWT-based token exchange
    CreateKeyToken(userID, provider string, expiry time.Duration) (string, error)
}

type KeyProvisioningStrategy string

const (
    KMSStorage      KeyProvisioningStrategy = "kms"
    RequestEmbedded KeyProvisioningStrategy = "embedded" 
    JWTToken        KeyProvisioningStrategy = "jwt"
    SessionBased    KeyProvisioningStrategy = "session"
)

type UserProviderConfig struct {
    Provider string                  `json:"provider"`
    Strategy KeyProvisioningStrategy `json:"strategy"`
    
    // For KMS strategy
    KeyID     string `json:"key_id,omitempty"`
    KMSConfig *KMSConfig `json:"kms_config,omitempty"`
    
    // For embedded strategy  
    EncryptedKey []byte `json:"encrypted_key,omitempty"`
    
    // For JWT strategy
    KeyToken string `json:"key_token,omitempty"`
    
    // For session strategy
    SessionToken string `json:"session_token,omitempty"`
}
```

## API Design

### Enhanced Request Structure

```go
type EnhancedInvocationRequest struct {
    *InvocationRequest
    
    // User credential strategy
    UserCredentials *UserCredentials `json:"user_credentials,omitempty"`
    
    // Fallback to backend keys if no user credentials
    UseBackendKeys bool `json:"use_backend_keys,omitempty"`
}

type UserCredentials struct {
    UserID   string `json:"user_id"`
    TenantID string `json:"tenant_id,omitempty"`
    
    // Multiple strategies supported
    Providers []UserProviderConfig `json:"providers"`
    
    // Request-level encryption
    RequestEncryption *RequestEncryption `json:"request_encryption,omitempty"`
}

type RequestEncryption struct {
    Algorithm string `json:"algorithm"` // AES-GCM, ChaCha20-Poly1305
    KeyID     string `json:"key_id"`
    Nonce     []byte `json:"nonce"`
}
```

### Key Management APIs

```go
// Key management service
type KeyManagementService interface {
    // Store user keys
    StoreKey(ctx context.Context, req *StoreKeyRequest) (*StoreKeyResponse, error)
    
    // List user keys
    ListKeys(ctx context.Context, req *ListKeysRequest) (*ListKeysResponse, error)
    
    // Update/rotate key
    UpdateKey(ctx context.Context, req *UpdateKeyRequest) (*UpdateKeyResponse, error)
    
    // Delete key
    DeleteKey(ctx context.Context, req *DeleteKeyRequest) (*DeleteKeyResponse, error)
    
    // Test key validity
    TestKey(ctx context.Context, req *TestKeyRequest) (*TestKeyResponse, error)
    
    // Get usage statistics
    GetKeyUsage(ctx context.Context, req *KeyUsageRequest) (*KeyUsageResponse, error)
}

type StoreKeyRequest struct {
    UserID      string                  `json:"user_id"`
    Provider    string                  `json:"provider"`
    KeyID       string                  `json:"key_id,omitempty"`
    
    // Key material (encrypted in transit)
    EncryptedKey []byte `json:"encrypted_key"`
    
    // Storage strategy
    Strategy KeyProvisioningStrategy `json:"strategy"`
    
    // Optional metadata
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

## Security Implementation Details

### Encryption Standards

```go
type EncryptionConfig struct {
    // Symmetric encryption for key storage
    StorageAlgorithm string `json:"storage_algorithm"` // AES-256-GCM
    
    // Asymmetric encryption for key exchange
    KeyExchangeAlgorithm string `json:"key_exchange_algorithm"` // ECDH-P256
    
    // Key derivation
    KDFAlgorithm string `json:"kdf_algorithm"` // PBKDF2, Argon2id
    
    // Message authentication
    MACAlgorithm string `json:"mac_algorithm"` // HMAC-SHA256
    
    // Digital signatures
    SignatureAlgorithm string `json:"signature_algorithm"` // ECDSA-P256
}
```

### Audit and Monitoring

```go
type KeyAuditEvent struct {
    Timestamp   time.Time              `json:"timestamp"`
    UserID      string                 `json:"user_id"`
    Provider    string                 `json:"provider"`
    Action      string                 `json:"action"` // store, retrieve, rotate, delete
    KeyID       string                 `json:"key_id"`
    Success     bool                   `json:"success"`
    IPAddress   string                 `json:"ip_address"`
    UserAgent   string                 `json:"user_agent"`
    Metadata    map[string]interface{} `json:"metadata"`
    
    // Never log actual keys!
    KeyHash     string `json:"key_hash"` // SHA-256 hash for correlation
}
```

## Implementation Phases

### Phase 1: Foundation (Week 1)
- Basic user credential types
- KMS integration (AWS KMS)
- Simple encrypted embedding
- Backward compatibility

### Phase 2: Enhanced Security (Week 2)
- JWT token support
- Session-based management
- Multi-KMS support (Vault, Azure)
- Audit logging

### Phase 3: Advanced Features (Week 3)
- Key rotation
- Usage analytics
- Rate limiting per user key
- Key validation and testing

### Phase 4: Enterprise Features (Week 4)
- RBAC integration
- Tenant isolation
- Compliance reporting
- Key escrow options

## Trade-off Analysis

| Approach | Security | Performance | Complexity | Cost |
|----------|----------|-------------|------------|------|
| KMS Storage | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| Request Embedded | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| JWT Tokens | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| Session Based | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐ |
| Hybrid | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |

## Recommendation

**Implement the Hybrid Approach** with the following priorities:

1. **Start with Request Embedded** - Simple, secure, no infrastructure
2. **Add Session Support** - Better performance for frequent use
3. **Integrate KMS** - Enterprise-grade security for long-term storage
4. **Add JWT Tokens** - Standards-based approach for ecosystem integration

This provides maximum flexibility while maintaining strong security posture and allowing gradual adoption based on user needs.