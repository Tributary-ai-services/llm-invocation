package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/tributary-ai/llm-invocation/types"
)

func TestClientKeyUtils_EncryptAPIKey(t *testing.T) {
	keyUtils := NewClientKeyUtils()
	
	testCases := []struct {
		name       string
		apiKey     string
		userSecret string
	}{
		{
			name:       "openai key",
			apiKey:     "sk-1234567890abcdef",
			userSecret: "user-password-123",
		},
		{
			name:       "anthropic key", 
			apiKey:     "sk-ant-api03-example-key",
			userSecret: "strong-user-secret",
		},
		{
			name:       "long key",
			apiKey:     "very-long-api-key-that-exceeds-normal-length-for-testing-edge-cases",
			userSecret: "complex-password-with-special-chars!@#$%",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encryptedKey, err := keyUtils.EncryptAPIKey(tc.apiKey, tc.userSecret)
			if err != nil {
				t.Fatalf("Failed to encrypt API key: %v", err)
			}
			
			if len(encryptedKey) == 0 {
				t.Error("Encrypted key should not be empty")
			}
			
			// Test decryption
			err = keyUtils.TestKeyDecryption(encryptedKey, tc.userSecret)
			if err != nil {
				t.Errorf("Failed to decrypt with correct secret: %v", err)
			}
			
			// Test with wrong secret
			err = keyUtils.TestKeyDecryption(encryptedKey, "wrong-secret")
			if err == nil {
				t.Error("Decryption should fail with wrong secret")
			}
		})
	}
}

func TestClientKeyUtils_CreateUserCredentials(t *testing.T) {
	keyUtils := NewClientKeyUtils()
	
	userID := "test-user-123"
	tenantID := "test-tenant-456"
	userSecret := "test-password-789"
	
	providerKeys := map[string]string{
		"openai":    "sk-openai-key",
		"anthropic": "sk-ant-anthropic-key",
		"google":    "google-api-key",
	}
	
	creds, err := keyUtils.CreateUserCredentials(userID, tenantID, providerKeys, userSecret)
	if err != nil {
		t.Fatalf("Failed to create user credentials: %v", err)
	}
	
	// Validate structure
	if creds.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, creds.UserID)
	}
	if creds.TenantID != tenantID {
		t.Errorf("Expected TenantID %s, got %s", tenantID, creds.TenantID)
	}
	if len(creds.Providers) != len(providerKeys) {
		t.Errorf("Expected %d providers, got %d", len(providerKeys), len(creds.Providers))
	}
	
	// Validate each provider
	for _, provider := range creds.Providers {
		if len(provider.EncryptedKey) == 0 {
			t.Errorf("Provider %s should have encrypted key", provider.Provider)
		}
		if provider.KeyFingerprint == "" {
			t.Errorf("Provider %s should have key fingerprint", provider.Provider)
		}
		if provider.CreatedAt.IsZero() {
			t.Errorf("Provider %s should have creation time", provider.Provider)
		}
		
		// Test decryption
		err := keyUtils.TestKeyDecryption(provider.EncryptedKey, userSecret)
		if err != nil {
			t.Errorf("Failed to decrypt key for provider %s: %v", provider.Provider, err)
		}
	}
	
	// Test credential validation
	if err := creds.Validate(); err != nil {
		t.Errorf("Created credentials should be valid: %v", err)
	}
}

func TestClientKeyUtils_ValidateUserSecret(t *testing.T) {
	keyUtils := NewClientKeyUtils()
	
	testCases := []struct {
		name      string
		secret    string
		shouldErr bool
	}{
		{
			name:      "too short",
			secret:    "short",
			shouldErr: true,
		},
		{
			name:      "no numbers",
			secret:    "onlylettershere",
			shouldErr: true,
		},
		{
			name:      "no letters",
			secret:    "123456789012",
			shouldErr: true,
		},
		{
			name:      "valid simple",
			secret:    "password12345",
			shouldErr: false,
		},
		{
			name:      "valid complex",
			secret:    "Strong-Password-With-Numbers123!",
			shouldErr: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := keyUtils.ValidateUserSecret(tc.secret)
			if tc.shouldErr && err == nil {
				t.Error("Expected validation to fail")
			}
			if !tc.shouldErr && err != nil {
				t.Errorf("Expected validation to pass, got: %v", err)
			}
		})
	}
}

func TestClientKeyUtils_GenerateSecureSecret(t *testing.T) {
	keyUtils := NewClientKeyUtils()
	
	lengths := []int{12, 16, 24, 32}
	
	for _, length := range lengths {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			secret, err := keyUtils.GenerateSecureSecret(length)
			if err != nil {
				t.Fatalf("Failed to generate secure secret: %v", err)
			}
			
			expectedLength := length
			if length < 16 {
				expectedLength = 16 // Minimum length
			}
			
			if len(secret) != expectedLength {
				t.Errorf("Expected length %d, got %d", expectedLength, len(secret))
			}
			
			// Should pass validation
			err = keyUtils.ValidateUserSecret(secret)
			if err != nil {
				t.Errorf("Generated secret should pass validation: %v", err)
			}
		})
	}
	
	// Generate multiple secrets - should be different
	secret1, _ := keyUtils.GenerateSecureSecret(20)
	secret2, _ := keyUtils.GenerateSecureSecret(20)
	
	if secret1 == secret2 {
		t.Error("Generated secrets should be different")
	}
}

func TestClientKeyUtils_RotateEncryptedKey(t *testing.T) {
	keyUtils := NewClientKeyUtils()
	
	originalAPIKey := "sk-original-api-key"
	oldSecret := "old-password-123"
	newSecret := "new-password-456"
	
	// Encrypt with old secret
	oldEncryptedKey, err := keyUtils.EncryptAPIKey(originalAPIKey, oldSecret)
	if err != nil {
		t.Fatalf("Failed to encrypt with old secret: %v", err)
	}
	
	// Rotate key
	newEncryptedKey, err := keyUtils.RotateEncryptedKey(oldEncryptedKey, oldSecret, newSecret)
	if err != nil {
		t.Fatalf("Failed to rotate key: %v", err)
	}
	
	// Old secret should not work with new key
	err = keyUtils.TestKeyDecryption(newEncryptedKey, oldSecret)
	if err == nil {
		t.Error("Old secret should not work with rotated key")
	}
	
	// New secret should work with new key
	err = keyUtils.TestKeyDecryption(newEncryptedKey, newSecret)
	if err != nil {
		t.Errorf("New secret should work with rotated key: %v", err)
	}
}

func TestClientKeyUtils_IsKeyExpired(t *testing.T) {
	keyUtils := NewClientKeyUtils()
	
	now := time.Now()
	
	testCases := []struct {
		name        string
		createdAt   time.Time
		maxAge      time.Duration
		shouldExpire bool
	}{
		{
			name:        "not expired",
			createdAt:   now.Add(-30 * time.Minute),
			maxAge:      1 * time.Hour,
			shouldExpire: false,
		},
		{
			name:        "just expired",
			createdAt:   now.Add(-2 * time.Hour),
			maxAge:      1 * time.Hour,
			shouldExpire: true,
		},
		{
			name:        "zero time",
			createdAt:   time.Time{},
			maxAge:      1 * time.Hour,
			shouldExpire: false, // No creation time = not expired
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			providerConfig := &types.UserProviderConfig{
				Provider:  "test",
				CreatedAt: tc.createdAt,
			}
			
			isExpired := keyUtils.IsKeyExpired(providerConfig, tc.maxAge)
			if isExpired != tc.shouldExpire {
				t.Errorf("Expected expired=%v, got %v", tc.shouldExpire, isExpired)
			}
		})
	}
}

func TestClientKeyUtils_CreateCredentialsSummary(t *testing.T) {
	keyUtils := NewClientKeyUtils()
	
	// Create test credentials
	userID := "summary-user"
	tenantID := "summary-tenant"
	userSecret := "summary-password123"
	
	providerKeys := map[string]string{
		"openai":    "sk-openai-test",
		"anthropic": "sk-ant-test",
	}
	
	creds, err := keyUtils.CreateUserCredentials(userID, tenantID, providerKeys, userSecret)
	if err != nil {
		t.Fatalf("Failed to create test credentials: %v", err)
	}
	
	// Create summary
	summary := keyUtils.CreateCredentialsSummary(creds)
	
	// Validate summary structure
	if summary["user_id"] != userID {
		t.Errorf("Expected user_id %s, got %v", userID, summary["user_id"])
	}
	if summary["tenant_id"] != tenantID {
		t.Errorf("Expected tenant_id %s, got %v", tenantID, summary["tenant_id"])
	}
	if summary["num_providers"] != len(providerKeys) {
		t.Errorf("Expected num_providers %d, got %v", len(providerKeys), summary["num_providers"])
	}
	
	providers, ok := summary["providers"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected providers to be slice of maps")
	}
	
	if len(providers) != len(providerKeys) {
		t.Errorf("Expected %d provider summaries, got %d", len(providerKeys), len(providers))
	}
	
	// Validate that no sensitive data is included
	for _, provider := range providers {
		if _, hasKey := provider["encrypted_key"]; hasKey {
			t.Error("Summary should not contain encrypted_key")
		}
		if _, hasKey := provider["api_key"]; hasKey {
			t.Error("Summary should not contain api_key")
		}
		
		// Should have safe fields
		requiredFields := []string{"name", "strategy", "has_encrypted", "key_fingerprint"}
		for _, field := range requiredFields {
			if _, exists := provider[field]; !exists {
				t.Errorf("Summary should contain field: %s", field)
			}
		}
	}
}