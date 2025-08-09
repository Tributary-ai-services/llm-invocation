# User-Controlled API Key Management - Implementation Roadmap

## Overview

This roadmap progresses from the simplest implementation (request embedding) to the most complex (full KMS integration), allowing for incremental development and testing while building toward enterprise-grade security.

## Phase 1: Request Embedding (Week 1) 🟢 SIMPLE

### Complexity: ⭐⭐
### Dependencies: None
### Security: ⭐⭐⭐⭐

**Goal**: Enable users to embed encrypted API keys directly in requests without backend storage.

### Implementation Tasks

#### Day 1-2: Core Types and Encryption
```go
// New types to add to types/user_credentials.go
type UserCredentials struct {
    UserID      string `json:"user_id"`
    TenantID    string `json:"tenant_id,omitempty"`
    Providers   []UserProviderConfig `json:"providers"`
}

type UserProviderConfig struct {
    Provider      string `json:"provider"`
    EncryptedKey  []byte `json:"encrypted_key"`
    KeyFingerprint string `json:"key_fingerprint"`
}

// Enhanced request type
type EnhancedInvocationRequest struct {
    *InvocationRequest
    UserCredentials *UserCredentials `json:"user_credentials,omitempty"`
    UseBackendKeys  bool             `json:"use_backend_keys,omitempty"`
}
```

**Implementation Files**:
- `crypto/encryption.go` - AES-256-GCM encryption utilities
- `types/user_credentials.go` - User credential types
- `auth/key_manager.go` - Request-level key decryption

#### Day 3-4: Client Integration
```go
// Client-side key encryption (examples/user-keys/)
func encryptAPIKey(apiKey, userSecret string) ([]byte, error) {
    // PBKDF2 key derivation from user secret
    // AES-256-GCM encryption
    // Return encrypted key bytes
}

// Server-side key decryption
func (c *client) decryptUserKey(encryptedKey []byte, userSecret string) (string, error) {
    // Decrypt using same derivation
    // Validate key format
    // Return plaintext API key
}
```

#### Day 5-6: Integration and Testing
- Update provider clients to accept decrypted keys
- Add backward compatibility with backend keys
- Write integration tests
- Create example applications

### Deliverables
- ✅ Request embedding with AES-256-GCM encryption
- ✅ Client utilities for key encryption
- ✅ Backward compatibility with existing backend keys
- ✅ Example application with BYOK
- ✅ Unit and integration tests

### Success Metrics
- Users can provide encrypted keys in requests
- No plaintext keys stored or logged
- Performance impact < 5ms per request
- 100% backward compatibility

---

## Phase 2: Session-Based Management (Week 2) 🟡 MODERATE

### Complexity: ⭐⭐⭐
### Dependencies: Redis/In-memory store
### Security: ⭐⭐⭐

**Goal**: Add session-based key caching for better performance with frequent requests.

### Implementation Tasks

#### Day 1-2: Session Storage
```go
// Session management types
type KeySession struct {
    ID          string                `json:"id"`
    UserID      string                `json:"user_id"`
    ProviderKeys map[string][]byte    `json:"provider_keys"`
    CreatedAt   time.Time             `json:"created_at"`
    ExpiresAt   time.Time             `json:"expires_at"`
    LastUsedAt  time.Time             `json:"last_used_at"`
}

type SessionManager interface {
    CreateSession(userID string, encryptedKeys map[string][]byte) (*KeySession, error)
    GetSession(sessionID string) (*KeySession, error)
    ExtendSession(sessionID string, duration time.Duration) error
    RevokeSession(sessionID string) error
    CleanupExpiredSessions() error
}
```

**Implementation Files**:
- `session/manager.go` - Session management interface
- `session/redis_store.go` - Redis-based session storage
- `session/memory_store.go` - In-memory storage for testing

#### Day 3-4: Session APIs
```go
// Session creation endpoint
POST /api/v1/sessions/keys
{
    "user_id": "user123",
    "encrypted_keys": {
        "openai": "encrypted_key_bytes",
        "anthropic": "encrypted_key_bytes"
    },
    "expires_in": 3600
}

// Response
{
    "session_token": "sess_abc123",
    "expires_at": "2024-01-01T12:00:00Z"
}

// Enhanced request with session
type SessionInvocationRequest struct {
    *InvocationRequest
    SessionToken string `json:"session_token"`
}
```

#### Day 5-6: Performance Optimization
- Implement session connection pooling
- Add session pre-warming
- Background session cleanup
- Performance benchmarking

### Deliverables
- ✅ Redis-based session storage
- ✅ Session creation/management APIs
- ✅ Session-aware request processing
- ✅ Automatic session cleanup
- ✅ Performance benchmarks showing >10x improvement

### Success Metrics
- Session lookup < 1ms
- Memory usage < 1KB per session
- Automatic cleanup of expired sessions
- Support for 10,000+ concurrent sessions

---

## Phase 3: JWT Token System (Week 3) 🟠 MODERATE-COMPLEX

### Complexity: ⭐⭐⭐⭐
### Dependencies: JWT library, key signing
### Security: ⭐⭐⭐⭐

**Goal**: Add JWT-based key tokens for standards-compliant, time-limited access.

### Implementation Tasks

#### Day 1-2: JWT Infrastructure
```go
// JWT key token structure
type KeyTokenClaims struct {
    jwt.RegisteredClaims
    UserID       string            `json:"user_id"`
    Provider     string            `json:"provider"`
    KeyHash      string            `json:"key_hash"`
    EncryptedKey []byte            `json:"encrypted_key"`
    Permissions  []string          `json:"permissions,omitempty"`
}

type JWTKeyManager interface {
    CreateKeyToken(userID, provider string, apiKey []byte, expiry time.Duration) (string, error)
    ValidateKeyToken(token string) (*KeyTokenClaims, error)
    RevokeKeyToken(tokenID string) error
    ListActiveTokens(userID string) ([]*KeyTokenClaims, error)
}
```

**Implementation Files**:
- `auth/jwt_manager.go` - JWT token creation and validation
- `auth/key_rotation.go` - Signing key rotation
- `auth/revocation.go` - Token revocation list

#### Day 3-4: Token Management APIs
```go
// Token creation
POST /api/v1/tokens/keys
{
    "provider": "openai",
    "api_key": "encrypted_api_key",
    "expires_in": 7200,
    "permissions": ["invoke", "list_models"]
}

// Response
{
    "key_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2024-01-01T14:00:00Z",
    "token_id": "tok_abc123"
}

// JWT-based request
type JWTInvocationRequest struct {
    *InvocationRequest
    KeyToken string `json:"key_token"`
}
```

#### Day 5-6: Advanced Features
- Token revocation lists
- Key rotation for JWT signing
- Permission-based access control
- Audit logging for token usage

### Deliverables
- ✅ JWT-based key token system
- ✅ Token creation and validation APIs
- ✅ Token revocation mechanism
- ✅ Permission-based access control
- ✅ Comprehensive audit logging

### Success Metrics
- Token validation < 2ms
- Support for fine-grained permissions
- Secure token revocation
- Full audit trail of token usage

---

## Phase 4: KMS Integration (Week 4) 🔴 COMPLEX

### Complexity: ⭐⭐⭐⭐⭐
### Dependencies: AWS KMS/HashiCorp Vault/Azure Key Vault
### Security: ⭐⭐⭐⭐⭐

**Goal**: Enterprise-grade key management with HSM backing and advanced features.

### Implementation Tasks

#### Day 1-2: KMS Abstractions
```go
// Universal KMS interface
type KMSProvider interface {
    Encrypt(ctx context.Context, keyID string, plaintext []byte) (*EncryptResponse, error)
    Decrypt(ctx context.Context, ciphertext []byte) (*DecryptResponse, error)
    GenerateDataKey(ctx context.Context, keyID string) (*DataKeyResponse, error)
    RotateKey(ctx context.Context, keyID string) error
    AuditKeyAccess(ctx context.Context, keyID string) ([]*AuditEntry, error)
}

// Provider implementations
type AWSKMSProvider struct {
    client *kms.Client
    keyARN string
}

type VaultKMSProvider struct {
    client     *vault.Client
    mountPath  string
    transitKey string
}

type AzureKVProvider struct {
    client   *keyvault.BaseClient
    vaultURL string
    keyName  string
}
```

**Implementation Files**:
- `kms/interface.go` - Universal KMS interface
- `kms/aws/` - AWS KMS implementation
- `kms/vault/` - HashiCorp Vault implementation  
- `kms/azure/` - Azure Key Vault implementation

#### Day 3-4: KMS-Based Storage
```go
// KMS key storage
type KMSKeyStore struct {
    kmsProvider KMSProvider
    database    Database
    audit       AuditLogger
}

func (k *KMSKeyStore) StoreUserKey(userID, provider, keyID string, apiKey []byte) error {
    // Encrypt API key using KMS
    encryptResp, err := k.kmsProvider.Encrypt(ctx, keyID, apiKey)
    if err != nil {
        return err
    }
    
    // Store encrypted key in database
    return k.database.Store(&EncryptedKey{
        UserID:           userID,
        Provider:         provider,
        EncryptedKey:     encryptResp.Ciphertext,
        KeyFingerprint:   sha256Hash(apiKey),
        CreatedAt:        time.Now(),
    })
}
```

#### Day 5-6: Enterprise Features
```go
// Key rotation
type KeyRotationManager struct {
    kmsStore     *KMSKeyStore
    scheduler    *cron.Cron
    notifications NotificationService
}

// Usage analytics
type KeyUsageAnalytics struct {
    UserID        string    `json:"user_id"`
    Provider      string    `json:"provider"`
    RequestCount  int64     `json:"request_count"`
    TokensUsed    int64     `json:"tokens_used"`
    LastUsed      time.Time `json:"last_used"`
    CostEstimate  float64   `json:"cost_estimate"`
}

// Compliance reporting
type ComplianceReport struct {
    UserID          string               `json:"user_id"`
    ReportPeriod    string              `json:"report_period"`
    KeyAccesses     []*KeyAccessEvent   `json:"key_accesses"`
    SecurityEvents  []*SecurityEvent    `json:"security_events"`
    ComplianceScore float64             `json:"compliance_score"`
}
```

### Deliverables
- ✅ Multi-cloud KMS integration (AWS, Azure, Vault)
- ✅ HSM-backed key storage and encryption
- ✅ Automated key rotation
- ✅ Advanced usage analytics
- ✅ Compliance reporting
- ✅ Enterprise audit trails

### Success Metrics
- HSM-level security for all user keys
- Automatic key rotation
- Comprehensive audit trails
- Sub-second key retrieval from KMS
- SOC 2 / GDPR compliance ready

---

## Phase 5: Advanced Enterprise Features (Week 5+) 🔵 ENTERPRISE

### Complexity: ⭐⭐⭐⭐⭐
### Dependencies: Full enterprise stack
### Security: ⭐⭐⭐⭐⭐

**Goal**: Advanced enterprise features for large-scale deployments.

### Key Features

#### Multi-Tenant Isolation
```go
type TenantKeyManager struct {
    tenantID        string
    isolationLevel  IsolationLevel
    kmsProvider     KMSProvider
    accessControls  *TenantAccessControls
}

type IsolationLevel string
const (
    SharedInfra    IsolationLevel = "shared"
    DedicatedKMS   IsolationLevel = "dedicated_kms"  
    DedicatedHSM   IsolationLevel = "dedicated_hsm"
    AirGapped      IsolationLevel = "air_gapped"
)
```

#### Role-Based Access Control (RBAC)
```go
type KeyPermission struct {
    Resource   string   `json:"resource"`   // provider:model
    Actions    []string `json:"actions"`    // invoke, list, manage
    Conditions []string `json:"conditions"` // time_of_day, ip_range
}

type UserRole struct {
    RoleID      string          `json:"role_id"`
    RoleName    string          `json:"role_name"`  
    Permissions []KeyPermission `json:"permissions"`
    Constraints *RoleConstraints `json:"constraints"`
}
```

#### Key Escrow and Recovery
```go
type KeyEscrowService struct {
    escrowProvider EscrowProvider
    recoveryPolicy *RecoveryPolicy
    auditLogger    AuditLogger
}

type RecoveryPolicy struct {
    RequiredApprovers int      `json:"required_approvers"`
    ApproverRoles    []string `json:"approver_roles"`
    RecoveryWindow   string   `json:"recovery_window"`
    NotificationList []string `json:"notification_list"`
}
```

### Deliverables
- ✅ Multi-tenant key isolation
- ✅ Advanced RBAC with conditions
- ✅ Key escrow and recovery
- ✅ Compliance automation
- ✅ Advanced threat detection
- ✅ Performance optimization

---

## Implementation Timeline Summary

| Phase | Duration | Complexity | Features | Dependencies |
|-------|----------|------------|----------|--------------|
| **Phase 1** | 1 week | ⭐⭐ | Request embedding, basic encryption | None |
| **Phase 2** | 1 week | ⭐⭐⭐ | Session management, caching | Redis/Memory store |
| **Phase 3** | 1 week | ⭐⭐⭐⭐ | JWT tokens, revocation | JWT libraries |
| **Phase 4** | 1 week | ⭐⭐⭐⭐⭐ | KMS integration, HSM | Cloud KMS services |
| **Phase 5** | 2+ weeks | ⭐⭐⭐⭐⭐ | Enterprise features | Full enterprise stack |

## Risk Mitigation

### Phase 1 Risks
- **Risk**: Performance impact of per-request decryption
- **Mitigation**: Benchmark and optimize crypto operations

### Phase 2 Risks  
- **Risk**: Session storage scaling
- **Mitigation**: Redis clustering, session partitioning

### Phase 3 Risks
- **Risk**: JWT size limitations
- **Mitigation**: Key references instead of embedding full keys

### Phase 4 Risks
- **Risk**: KMS service availability
- **Mitigation**: Multi-region deployment, fallback mechanisms

### Phase 5 Risks
- **Risk**: Complexity overload
- **Mitigation**: Gradual rollout, extensive testing

## Success Criteria

Each phase must meet these criteria before proceeding:
- ✅ **Security**: Pass security audit
- ✅ **Performance**: Meet latency requirements
- ✅ **Compatibility**: Maintain backward compatibility
- ✅ **Documentation**: Complete API documentation
- ✅ **Testing**: >95% test coverage
- ✅ **Monitoring**: Full observability instrumentation

**Total Timeline**: 5-7 weeks for complete enterprise-grade BYOK implementation.