package auth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tributary-ai/llm-invocation/crypto"
	"github.com/tributary-ai/llm-invocation/types"
)

// ClientKeyUtils provides utilities for client-side key management
type ClientKeyUtils struct{}

// NewClientKeyUtils creates a new ClientKeyUtils instance
func NewClientKeyUtils() *ClientKeyUtils {
	return &ClientKeyUtils{}
}

// EncryptAPIKey encrypts an API key using a user secret
func (c *ClientKeyUtils) EncryptAPIKey(apiKey, userSecret string) ([]byte, error) {
	// Encrypt the API key
	encData, err := crypto.EncryptWithPassword([]byte(apiKey), []byte(userSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt API key: %w", err)
	}
	
	// Serialize to JSON for storage/transmission
	encBytes, err := json.Marshal(encData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize encrypted key: %w", err)
	}
	
	// Securely clear the plaintext API key from memory
	crypto.SecureZero([]byte(apiKey))
	
	return encBytes, nil
}

// CreateUserCredentials creates user credentials with encrypted API keys
func (c *ClientKeyUtils) CreateUserCredentials(userID, tenantID string, providerKeys map[string]string, userSecret string) (*types.UserCredentials, error) {
	creds := &types.UserCredentials{
		UserID:    userID,
		TenantID:  tenantID,
		Providers: make([]types.UserProviderConfig, 0, len(providerKeys)),
		RequestMeta: &types.RequestMetadata{
			EncryptionVersion: "1.0",
			Timestamp:         time.Now(),
		},
	}
	
	// Encrypt each provider's API key
	for provider, apiKey := range providerKeys {
		encryptedKey, err := c.EncryptAPIKey(apiKey, userSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt key for provider %s: %w", provider, err)
		}
		
		// Create fingerprint for verification
		fingerprint := crypto.CreateKeyFingerprint([]byte(apiKey))
		
		providerConfig := types.UserProviderConfig{
			Provider:       provider,
			EncryptedKey:   encryptedKey,
			KeyFingerprint: fingerprint,
			CreatedAt:      time.Now(),
		}
		
		creds.Providers = append(creds.Providers, providerConfig)
		
		// Securely clear the plaintext API key
		crypto.SecureZero([]byte(apiKey))
	}
	
	return creds, nil
}

// CreateEnhancedRequest creates an enhanced invocation request with user credentials
func (c *ClientKeyUtils) CreateEnhancedRequest(
	baseRequest *types.InvocationRequest,
	userCredentials *types.UserCredentials,
	useBackendKeys bool,
) *types.EnhancedInvocationRequest {
	return &types.EnhancedInvocationRequest{
		InvocationRequest: baseRequest,
		UserCredentials:   userCredentials,
		UseBackendKeys:    useBackendKeys,
		SecurityOptions: &types.SecurityOptions{
			RequireEncryption: true,
			MaxKeyAge:        24 * time.Hour,
		},
	}
}

// ValidateUserSecret checks if a user secret meets security requirements
func (c *ClientKeyUtils) ValidateUserSecret(secret string) error {
	if len(secret) < 12 {
		return fmt.Errorf("user secret must be at least 12 characters long")
	}
	
	// Check for basic complexity (at least one letter and one number)
	hasLetter := false
	hasNumber := false
	
	for _, char := range secret {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			hasLetter = true
		}
		if char >= '0' && char <= '9' {
			hasNumber = true
		}
	}
	
	if !hasLetter {
		return fmt.Errorf("user secret must contain at least one letter")
	}
	if !hasNumber {
		return fmt.Errorf("user secret must contain at least one number")
	}
	
	return nil
}

// GenerateSecureSecret generates a cryptographically secure user secret
func (c *ClientKeyUtils) GenerateSecureSecret(length int) (string, error) {
	if length < 16 {
		length = 16 // Minimum secure length
	}
	
	// Character sets for ensuring required types
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const numbers = "0123456789"
	const symbols = "!@#$%^&*"
	const charset = letters + numbers + symbols
	
	for {
		bytes, err := crypto.GenerateRandomBytes(length)
		if err != nil {
			return "", fmt.Errorf("failed to generate random bytes: %w", err)
		}
		
		secret := make([]byte, length)
		for i, b := range bytes {
			secret[i] = charset[b%byte(len(charset))]
		}
		
		secretStr := string(secret)
		
		// Verify it meets requirements (has letters and numbers)
		if err := c.ValidateUserSecret(secretStr); err == nil {
			return secretStr, nil
		}
		// If validation fails, try again (rare case)
	}
}

// TestKeyDecryption tests if a key can be decrypted with the given secret
func (c *ClientKeyUtils) TestKeyDecryption(encryptedKey []byte, userSecret string) error {
	// Parse encrypted data
	var encData crypto.EncryptedData
	if err := json.Unmarshal(encryptedKey, &encData); err != nil {
		return fmt.Errorf("invalid encrypted key format: %w", err)
	}
	
	// Attempt decryption
	plaintext, err := crypto.DecryptWithPassword(&encData, []byte(userSecret))
	if err != nil {
		return fmt.Errorf("decryption failed: %w", err)
	}
	
	// Securely clear decrypted data
	crypto.SecureZero(plaintext)
	
	return nil
}

// RotateEncryptedKey re-encrypts a key with a new user secret
func (c *ClientKeyUtils) RotateEncryptedKey(oldEncryptedKey []byte, oldSecret, newSecret string) ([]byte, error) {
	// Decrypt with old secret
	var encData crypto.EncryptedData
	if err := json.Unmarshal(oldEncryptedKey, &encData); err != nil {
		return nil, fmt.Errorf("invalid encrypted key format: %w", err)
	}
	
	plaintext, err := crypto.DecryptWithPassword(&encData, []byte(oldSecret))
	if err != nil {
		return nil, fmt.Errorf("decryption with old secret failed: %w", err)
	}
	
	// Re-encrypt with new secret
	newEncryptedKey, err := c.EncryptAPIKey(string(plaintext), newSecret)
	
	// Securely clear plaintext
	crypto.SecureZero(plaintext)
	
	if err != nil {
		return nil, fmt.Errorf("re-encryption failed: %w", err)
	}
	
	return newEncryptedKey, nil
}

// GetKeyFingerprint extracts the key fingerprint without decrypting
func (c *ClientKeyUtils) GetKeyFingerprint(providerConfig *types.UserProviderConfig) string {
	return providerConfig.KeyFingerprint
}

// IsKeyExpired checks if a key has expired
func (c *ClientKeyUtils) IsKeyExpired(providerConfig *types.UserProviderConfig, maxAge time.Duration) bool {
	if providerConfig.CreatedAt.IsZero() {
		return false // No creation time, assume not expired
	}
	
	return time.Since(providerConfig.CreatedAt) > maxAge
}

// CreateCredentialsSummary creates a safe summary of credentials for logging
func (c *ClientKeyUtils) CreateCredentialsSummary(creds *types.UserCredentials) map[string]interface{} {
	summary := map[string]interface{}{
		"user_id":       creds.UserID,
		"tenant_id":     creds.TenantID,
		"num_providers": len(creds.Providers),
		"providers":     make([]map[string]interface{}, len(creds.Providers)),
	}
	
	for i, provider := range creds.Providers {
		providerSummary := map[string]interface{}{
			"name":            provider.Provider,
			"strategy":        provider.GetStrategy(),
			"has_encrypted":   len(provider.EncryptedKey) > 0,
			"key_fingerprint": provider.KeyFingerprint,
			"created_at":      provider.CreatedAt,
			"is_expired":      provider.IsExpired(),
		}
		
		summary["providers"].([]map[string]interface{})[i] = providerSummary
	}
	
	return summary
}