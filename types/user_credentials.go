package types

import (
	"encoding/json"
	"time"

	"github.com/tributary-ai/llm-invocation/crypto"
)

// UserCredentials contains user-provided API keys and authentication information
type UserCredentials struct {
	UserID      string                `json:"user_id"`
	TenantID    string                `json:"tenant_id,omitempty"`
	Providers   []UserProviderConfig  `json:"providers"`
	RequestMeta *RequestMetadata      `json:"request_meta,omitempty"`
}

// UserProviderConfig represents a user's configuration for a specific provider
type UserProviderConfig struct {
	Provider string `json:"provider"`
	
	// Encrypted API key (Phase 1: Request Embedding)
	EncryptedKey   []byte `json:"encrypted_key,omitempty"`
	KeyFingerprint string `json:"key_fingerprint,omitempty"`
	
	// Future: Session-based (Phase 2)
	SessionToken string `json:"session_token,omitempty"`
	
	// Future: JWT-based (Phase 3) 
	KeyToken string `json:"key_token,omitempty"`
	
	// Future: KMS-based (Phase 4)
	KeyID     string     `json:"key_id,omitempty"`
	KMSConfig *KMSConfig `json:"kms_config,omitempty"`
	
	// Metadata
	CreatedAt time.Time `json:"created_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// RequestMetadata contains metadata about the encrypted request
type RequestMetadata struct {
	EncryptionVersion string    `json:"encryption_version"`
	Timestamp         time.Time `json:"timestamp"`
	ClientIP          string    `json:"client_ip,omitempty"`
	UserAgent         string    `json:"user_agent,omitempty"`
	RequestID         string    `json:"request_id,omitempty"`
}

// KMSConfig contains configuration for KMS-based key storage (Phase 4)
type KMSConfig struct {
	Provider  string `json:"provider"`   // aws, vault, azure, gcp
	Region    string `json:"region,omitempty"`
	KeyARN    string `json:"key_arn,omitempty"`
	VaultPath string `json:"vault_path,omitempty"`
}

// EnhancedInvocationRequest extends the base request with user credentials
type EnhancedInvocationRequest struct {
	*InvocationRequest
	
	// User-provided credentials
	UserCredentials *UserCredentials `json:"user_credentials,omitempty"`
	
	// Fallback to backend keys if no user credentials provided
	UseBackendKeys bool `json:"use_backend_keys,omitempty"`
	
	// Request-level security options
	SecurityOptions *SecurityOptions `json:"security_options,omitempty"`
}

// SecurityOptions provides additional security controls
type SecurityOptions struct {
	// Require encryption for user keys
	RequireEncryption bool `json:"require_encryption,omitempty"`
	
	// Maximum age of encrypted keys (to prevent replay attacks)
	MaxKeyAge time.Duration `json:"max_key_age,omitempty"`
	
	// IP address restrictions
	AllowedIPs []string `json:"allowed_ips,omitempty"`
	
	// Rate limiting per user
	RateLimit *RateLimit `json:"rate_limit,omitempty"`
}

// RateLimit defines rate limiting rules for user requests
type RateLimit struct {
	RequestsPerMinute int `json:"requests_per_minute,omitempty"`
	TokensPerMinute   int `json:"tokens_per_minute,omitempty"`
	BurstLimit        int `json:"burst_limit,omitempty"`
}

// ProviderStrategy defines how keys are provided for a specific request
type ProviderStrategy string

const (
	StrategyBackend      ProviderStrategy = "backend"       // Use backend-stored keys
	StrategyEmbedded     ProviderStrategy = "embedded"      // Encrypted keys in request
	StrategySession      ProviderStrategy = "session"       // Session-based keys (Phase 2)
	StrategyJWT          ProviderStrategy = "jwt"           // JWT tokens (Phase 3)
	StrategyKMS          ProviderStrategy = "kms"           // KMS-stored keys (Phase 4)
)

// EncryptedAPIKey represents an encrypted API key with metadata
type EncryptedAPIKey struct {
	*crypto.EncryptedData
	
	Provider       string    `json:"provider"`
	KeyFingerprint string    `json:"key_fingerprint"`
	CreatedAt      time.Time `json:"created_at"`
	UserID         string    `json:"user_id"`
}

// Validate checks if the user credentials are valid
func (uc *UserCredentials) Validate() error {
	if uc.UserID == "" {
		return NewError(ErrInvalidRequest, "user_id is required")
	}
	
	if len(uc.Providers) == 0 {
		return NewError(ErrInvalidRequest, "at least one provider configuration is required")
	}
	
	// Validate each provider config
	for _, provider := range uc.Providers {
		if err := provider.Validate(); err != nil {
			return err
		}
	}
	
	return nil
}

// Validate checks if a provider configuration is valid
func (upc *UserProviderConfig) Validate() error {
	if upc.Provider == "" {
		return NewError(ErrInvalidRequest, "provider name is required")
	}
	
	// Check that at least one credential method is provided
	hasCredential := len(upc.EncryptedKey) > 0 ||
		upc.SessionToken != "" ||
		upc.KeyToken != "" ||
		upc.KeyID != ""
	
	if !hasCredential {
		return NewError(ErrInvalidRequest, "provider must have at least one credential method")
	}
	
	// If KMS is used, validate KMS config
	if upc.KeyID != "" && upc.KMSConfig == nil {
		return NewError(ErrInvalidRequest, "KMS config required when key_id is provided")
	}
	
	return nil
}

// GetStrategy returns the strategy being used for this provider config
func (upc *UserProviderConfig) GetStrategy() ProviderStrategy {
	switch {
	case len(upc.EncryptedKey) > 0:
		return StrategyEmbedded
	case upc.SessionToken != "":
		return StrategySession
	case upc.KeyToken != "":
		return StrategyJWT
	case upc.KeyID != "":
		return StrategyKMS
	default:
		return StrategyBackend
	}
}

// IsExpired checks if the provider config has expired
func (upc *UserProviderConfig) IsExpired() bool {
	if upc.ExpiresAt.IsZero() {
		return false // No expiry set
	}
	return time.Now().After(upc.ExpiresAt)
}

// GetProviderConfig finds a provider configuration by name
func (uc *UserCredentials) GetProviderConfig(provider string) *UserProviderConfig {
	for _, pc := range uc.Providers {
		if pc.Provider == provider {
			return &pc
		}
	}
	return nil
}

// HasProvider checks if credentials are provided for a specific provider
func (uc *UserCredentials) HasProvider(provider string) bool {
	return uc.GetProviderConfig(provider) != nil
}

// ToJSON serializes user credentials to JSON (for logging without sensitive data)
func (uc *UserCredentials) ToJSON() ([]byte, error) {
	// Create a safe copy without sensitive data
	safe := struct {
		UserID      string `json:"user_id"`
		TenantID    string `json:"tenant_id,omitempty"`
		NumProviders int   `json:"num_providers"`
		Providers   []struct {
			Provider       string           `json:"provider"`
			Strategy       ProviderStrategy `json:"strategy"`
			KeyFingerprint string           `json:"key_fingerprint,omitempty"`
			HasEncryptedKey bool            `json:"has_encrypted_key"`
			HasSessionToken bool            `json:"has_session_token"`
			HasKeyToken     bool            `json:"has_key_token"`
			HasKeyID        bool            `json:"has_key_id"`
			IsExpired       bool            `json:"is_expired"`
		} `json:"providers"`
	}{
		UserID:       uc.UserID,
		TenantID:     uc.TenantID,
		NumProviders: len(uc.Providers),
	}
	
	for _, pc := range uc.Providers {
		safe.Providers = append(safe.Providers, struct {
			Provider       string           `json:"provider"`
			Strategy       ProviderStrategy `json:"strategy"`
			KeyFingerprint string           `json:"key_fingerprint,omitempty"`
			HasEncryptedKey bool            `json:"has_encrypted_key"`
			HasSessionToken bool            `json:"has_session_token"`
			HasKeyToken     bool            `json:"has_key_token"`
			HasKeyID        bool            `json:"has_key_id"`
			IsExpired       bool            `json:"is_expired"`
		}{
			Provider:       pc.Provider,
			Strategy:       pc.GetStrategy(),
			KeyFingerprint: pc.KeyFingerprint,
			HasEncryptedKey: len(pc.EncryptedKey) > 0,
			HasSessionToken: pc.SessionToken != "",
			HasKeyToken:     pc.KeyToken != "",
			HasKeyID:        pc.KeyID != "",
			IsExpired:       pc.IsExpired(),
		})
	}
	
	return json.Marshal(safe)
}