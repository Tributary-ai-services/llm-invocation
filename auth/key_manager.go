package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tributary-ai/llm-invocation/crypto"
	"github.com/tributary-ai/llm-invocation/types"
)

// KeyManager handles decryption and validation of user-provided API keys
type KeyManager interface {
	// DecryptUserKey decrypts an encrypted API key using the provided password/secret
	DecryptUserKey(encryptedKey []byte, userSecret string) (string, error)
	
	// ValidateUserCredentials validates the format and content of user credentials
	ValidateUserCredentials(ctx context.Context, creds *types.UserCredentials) error
	
	// ExtractAPIKey extracts the plaintext API key for a provider from user credentials
	ExtractAPIKey(ctx context.Context, creds *types.UserCredentials, provider, userSecret string) (string, error)
	
	// CreateEncryptedKey creates an encrypted key for client-side use
	CreateEncryptedKey(apiKey, userSecret string) (*types.EncryptedAPIKey, error)
}

// RequestKeyManager implements KeyManager for request-embedded keys (Phase 1)
type RequestKeyManager struct {
	// Security options
	maxKeyAge       time.Duration
	requireValidIP  bool
	auditLogger     AuditLogger
}

// AuditLogger defines interface for security audit logging
type AuditLogger interface {
	LogKeyAccess(ctx context.Context, event *KeyAccessEvent) error
}

// KeyAccessEvent represents a key access event for audit purposes
type KeyAccessEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	UserID       string    `json:"user_id"`
	Provider     string    `json:"provider"`
	Action       string    `json:"action"` // decrypt, validate, extract
	Success      bool      `json:"success"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	RequestID    string    `json:"request_id,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	// Never log actual keys - only fingerprints!
	KeyFingerprint string `json:"key_fingerprint,omitempty"`
}

// NewRequestKeyManager creates a new RequestKeyManager
func NewRequestKeyManager(options ...KeyManagerOption) *RequestKeyManager {
	km := &RequestKeyManager{
		maxKeyAge:      24 * time.Hour, // Default 24 hour max age
		requireValidIP: false,
		auditLogger:    &noOpAuditLogger{},
	}
	
	for _, opt := range options {
		opt(km)
	}
	
	return km
}

// KeyManagerOption defines configuration options for KeyManager
type KeyManagerOption func(*RequestKeyManager)

// WithMaxKeyAge sets the maximum age for encrypted keys
func WithMaxKeyAge(duration time.Duration) KeyManagerOption {
	return func(km *RequestKeyManager) {
		km.maxKeyAge = duration
	}
}

// WithAuditLogger sets the audit logger
func WithAuditLogger(logger AuditLogger) KeyManagerOption {
	return func(km *RequestKeyManager) {
		km.auditLogger = logger
	}
}

// WithIPValidation enables IP address validation
func WithIPValidation(enabled bool) KeyManagerOption {
	return func(km *RequestKeyManager) {
		km.requireValidIP = enabled
	}
}

// DecryptUserKey decrypts an encrypted API key using PBKDF2 + AES-256-GCM
func (km *RequestKeyManager) DecryptUserKey(encryptedKey []byte, userSecret string) (string, error) {
	// Parse the encrypted data structure
	var encData crypto.EncryptedData
	if err := parseEncryptedKey(encryptedKey, &encData); err != nil {
		return "", fmt.Errorf("invalid encrypted key format: %w", err)
	}
	
	// Decrypt using password-based decryption
	plaintext, err := crypto.DecryptWithPassword(&encData, []byte(userSecret))
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}
	
	// Convert to string and validate it looks like an API key
	apiKey := string(plaintext)
	if err := validateAPIKeyFormat(apiKey); err != nil {
		return "", fmt.Errorf("invalid API key format: %w", err)
	}
	
	// Securely clear plaintext from memory
	crypto.SecureZero(plaintext)
	
	return apiKey, nil
}

// ValidateUserCredentials performs comprehensive validation of user credentials
func (km *RequestKeyManager) ValidateUserCredentials(ctx context.Context, creds *types.UserCredentials) error {
	startTime := time.Now()
	
	// Basic structure validation
	if err := creds.Validate(); err != nil {
		km.logAuditEvent(ctx, &KeyAccessEvent{
			Timestamp:    startTime,
			UserID:       creds.UserID,
			Action:       "validate",
			Success:      false,
			ErrorMessage: err.Error(),
		})
		return err
	}
	
	// Validate each provider configuration
	for _, providerConfig := range creds.Providers {
		if err := km.validateProviderConfig(ctx, creds.UserID, &providerConfig); err != nil {
			km.logAuditEvent(ctx, &KeyAccessEvent{
				Timestamp:      startTime,
				UserID:         creds.UserID,
				Provider:       providerConfig.Provider,
				Action:         "validate",
				Success:        false,
				ErrorMessage:   err.Error(),
				KeyFingerprint: providerConfig.KeyFingerprint,
			})
			return fmt.Errorf("provider %s validation failed: %w", providerConfig.Provider, err)
		}
	}
	
	// Log successful validation
	km.logAuditEvent(ctx, &KeyAccessEvent{
		Timestamp: startTime,
		UserID:    creds.UserID,
		Action:    "validate",
		Success:   true,
	})
	
	return nil
}

// ExtractAPIKey extracts the plaintext API key for a specific provider
func (km *RequestKeyManager) ExtractAPIKey(ctx context.Context, creds *types.UserCredentials, provider, userSecret string) (string, error) {
	startTime := time.Now()
	
	// Find provider configuration
	providerConfig := creds.GetProviderConfig(provider)
	if providerConfig == nil {
		err := fmt.Errorf("no credentials found for provider %s", provider)
		km.logAuditEvent(ctx, &KeyAccessEvent{
			Timestamp:    startTime,
			UserID:       creds.UserID,
			Provider:     provider,
			Action:       "extract",
			Success:      false,
			ErrorMessage: err.Error(),
		})
		return "", err
	}
	
	// For Phase 1, we only handle embedded encrypted keys
	if len(providerConfig.EncryptedKey) == 0 {
		err := fmt.Errorf("no encrypted key found for provider %s", provider)
		km.logAuditEvent(ctx, &KeyAccessEvent{
			Timestamp:      startTime,
			UserID:         creds.UserID,
			Provider:       provider,
			Action:         "extract",
			Success:        false,
			ErrorMessage:   err.Error(),
			KeyFingerprint: providerConfig.KeyFingerprint,
		})
		return "", err
	}
	
	// Check if key has expired
	if providerConfig.IsExpired() {
		err := fmt.Errorf("credentials for provider %s have expired", provider)
		km.logAuditEvent(ctx, &KeyAccessEvent{
			Timestamp:      startTime,
			UserID:         creds.UserID,
			Provider:       provider,
			Action:         "extract",
			Success:        false,
			ErrorMessage:   err.Error(),
			KeyFingerprint: providerConfig.KeyFingerprint,
		})
		return "", err
	}
	
	// Decrypt the API key
	apiKey, err := km.DecryptUserKey(providerConfig.EncryptedKey, userSecret)
	if err != nil {
		km.logAuditEvent(ctx, &KeyAccessEvent{
			Timestamp:      startTime,
			UserID:         creds.UserID,
			Provider:       provider,
			Action:         "extract",
			Success:        false,
			ErrorMessage:   err.Error(),
			KeyFingerprint: providerConfig.KeyFingerprint,
		})
		return "", err
	}
	
	// Validate key fingerprint if provided
	if providerConfig.KeyFingerprint != "" {
		expectedFingerprint := crypto.CreateKeyFingerprint([]byte(apiKey))
		if expectedFingerprint != providerConfig.KeyFingerprint {
			err := fmt.Errorf("key fingerprint mismatch for provider %s", provider)
			km.logAuditEvent(ctx, &KeyAccessEvent{
				Timestamp:      startTime,
				UserID:         creds.UserID,
				Provider:       provider,
				Action:         "extract",
				Success:        false,
				ErrorMessage:   err.Error(),
				KeyFingerprint: providerConfig.KeyFingerprint,
			})
			return "", err
		}
	}
	
	// Log successful extraction
	km.logAuditEvent(ctx, &KeyAccessEvent{
		Timestamp:      startTime,
		UserID:         creds.UserID,
		Provider:       provider,
		Action:         "extract",
		Success:        true,
		KeyFingerprint: providerConfig.KeyFingerprint,
	})
	
	return apiKey, nil
}

// CreateEncryptedKey creates an encrypted key for client-side use
func (km *RequestKeyManager) CreateEncryptedKey(apiKey, userSecret string) (*types.EncryptedAPIKey, error) {
	// Encrypt the API key
	encData, err := crypto.EncryptWithPassword([]byte(apiKey), []byte(userSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}
	
	// Create fingerprint
	fingerprint := crypto.CreateKeyFingerprint([]byte(apiKey))
	
	// Securely clear the plaintext key from memory
	crypto.SecureZero([]byte(apiKey))
	
	return &types.EncryptedAPIKey{
		EncryptedData:  encData,
		KeyFingerprint: fingerprint,
		CreatedAt:      time.Now(),
	}, nil
}

// validateProviderConfig validates a single provider configuration
func (km *RequestKeyManager) validateProviderConfig(ctx context.Context, userID string, config *types.UserProviderConfig) error {
	// Basic validation
	if err := config.Validate(); err != nil {
		return err
	}
	
	// Check if key has expired
	if config.IsExpired() {
		return fmt.Errorf("credentials have expired")
	}
	
	// Check age of encrypted key (prevent replay attacks)
	if !config.CreatedAt.IsZero() && time.Since(config.CreatedAt) > km.maxKeyAge {
		return fmt.Errorf("encrypted key is too old (age: %v, max: %v)", time.Since(config.CreatedAt), km.maxKeyAge)
	}
	
	// Validate strategy-specific requirements
	switch config.GetStrategy() {
	case types.StrategyEmbedded:
		if len(config.EncryptedKey) == 0 {
			return fmt.Errorf("encrypted key is required for embedded strategy")
		}
		// TODO: Add format validation for encrypted key
		
	case types.StrategySession:
		return fmt.Errorf("session strategy not yet implemented")
		
	case types.StrategyJWT:
		return fmt.Errorf("JWT strategy not yet implemented")
		
	case types.StrategyKMS:
		return fmt.Errorf("KMS strategy not yet implemented")
	}
	
	return nil
}

// logAuditEvent logs a key access event
func (km *RequestKeyManager) logAuditEvent(ctx context.Context, event *KeyAccessEvent) {
	// Extract request metadata from context
	if requestMeta := getRequestMetadata(ctx); requestMeta != nil {
		event.IPAddress = requestMeta.ClientIP
		event.UserAgent = requestMeta.UserAgent
		event.RequestID = requestMeta.RequestID
	}
	
	// Log the event (errors are logged but don't fail the main operation)
	if err := km.auditLogger.LogKeyAccess(ctx, event); err != nil {
		// TODO: Add structured logging here
		fmt.Printf("Failed to log audit event: %v\n", err)
	}
}

// Helper functions

func parseEncryptedKey(data []byte, encData *crypto.EncryptedData) error {
	// For Phase 1, we'll use a simple JSON encoding
	// In production, you might want a more efficient binary format
	return json.Unmarshal(data, encData)
}

func validateAPIKeyFormat(apiKey string) error {
	// Basic API key validation
	if len(apiKey) < 10 {
		return fmt.Errorf("API key too short")
	}
	
	// Add provider-specific validation as needed
	// OpenAI keys start with "sk-"
	// Anthropic keys start with "sk-ant-"
	// Google keys have different formats
	
	return nil
}

func getRequestMetadata(ctx context.Context) *types.RequestMetadata {
	// Extract request metadata from context
	// This would be set by middleware in a real application
	if meta, ok := ctx.Value("request_metadata").(*types.RequestMetadata); ok {
		return meta
	}
	return nil
}

// noOpAuditLogger is a no-op implementation for testing
type noOpAuditLogger struct{}

func (l *noOpAuditLogger) LogKeyAccess(ctx context.Context, event *KeyAccessEvent) error {
	return nil
}