package auth

import (
	"context"
	"testing"
	"time"

	"github.com/tributary-ai/llm-invocation/types"
)

func TestRequestKeyManager_DecryptUserKey(t *testing.T) {
	km := NewRequestKeyManager()
	
	// Create test key and encrypt it
	testKey := "sk-test-api-key-12345"
	userSecret := "test-password-123"
	
	keyUtils := NewClientKeyUtils()
	encryptedKey, err := keyUtils.EncryptAPIKey(testKey, userSecret)
	if err != nil {
		t.Fatalf("Failed to encrypt test key: %v", err)
	}
	
	// Test decryption with correct password
	decryptedKey, err := km.DecryptUserKey(encryptedKey, userSecret)
	if err != nil {
		t.Fatalf("Failed to decrypt key: %v", err)
	}
	
	if decryptedKey != testKey {
		t.Errorf("Decrypted key doesn't match original. Expected: %s, Got: %s", testKey, decryptedKey)
	}
	
	// Test decryption with wrong password
	_, err = km.DecryptUserKey(encryptedKey, "wrong-password")
	if err == nil {
		t.Error("Decryption should fail with wrong password")
	}
}

func TestRequestKeyManager_ValidateUserCredentials(t *testing.T) {
	km := NewRequestKeyManager()
	ctx := context.Background()
	
	keyUtils := NewClientKeyUtils()
	
	// Test valid credentials
	validCreds, err := keyUtils.CreateUserCredentials(
		"test-user",
		"test-tenant",
		map[string]string{
			"openai": "sk-valid-key",
		},
		"valid-password123",
	)
	if err != nil {
		t.Fatalf("Failed to create valid credentials: %v", err)
	}
	
	err = km.ValidateUserCredentials(ctx, validCreds)
	if err != nil {
		t.Errorf("Valid credentials should pass validation: %v", err)
	}
	
	// Test invalid credentials - missing user ID
	invalidCreds := &types.UserCredentials{
		UserID:    "", // Missing
		TenantID:  "test-tenant",
		Providers: []types.UserProviderConfig{},
	}
	
	err = km.ValidateUserCredentials(ctx, invalidCreds)
	if err == nil {
		t.Error("Invalid credentials should fail validation")
	}
}

func TestRequestKeyManager_ExtractAPIKey(t *testing.T) {
	km := NewRequestKeyManager()
	ctx := context.Background()
	
	keyUtils := NewClientKeyUtils()
	userSecret := "extract-password123"
	testAPIKey := "sk-extract-test-key"
	
	// Create credentials
	creds, err := keyUtils.CreateUserCredentials(
		"extract-user",
		"extract-tenant",
		map[string]string{
			"openai": testAPIKey,
		},
		userSecret,
	)
	if err != nil {
		t.Fatalf("Failed to create credentials: %v", err)
	}
	
	// Extract API key
	extractedKey, err := km.ExtractAPIKey(ctx, creds, "openai", userSecret)
	if err != nil {
		t.Fatalf("Failed to extract API key: %v", err)
	}
	
	if extractedKey != testAPIKey {
		t.Errorf("Extracted key doesn't match original. Expected: %s, Got: %s", testAPIKey, extractedKey)
	}
	
	// Test extraction for non-existent provider
	_, err = km.ExtractAPIKey(ctx, creds, "nonexistent", userSecret)
	if err == nil {
		t.Error("Extraction should fail for non-existent provider")
	}
	
	// Test extraction with wrong secret
	_, err = km.ExtractAPIKey(ctx, creds, "openai", "wrong-secret")
	if err == nil {
		t.Error("Extraction should fail with wrong secret")
	}
}

func TestRequestKeyManager_CreateEncryptedKey(t *testing.T) {
	km := NewRequestKeyManager()
	
	testKey := "sk-create-test-key"
	userSecret := "create-password123"
	
	encryptedKey, err := km.CreateEncryptedKey(testKey, userSecret)
	if err != nil {
		t.Fatalf("Failed to create encrypted key: %v", err)
	}
	
	// Validate structure
	if encryptedKey.EncryptedData == nil {
		t.Error("EncryptedData should not be nil")
	}
	if encryptedKey.KeyFingerprint == "" {
		t.Error("KeyFingerprint should not be empty")
	}
	if encryptedKey.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	
	// Test decryption
	keyUtils2 := NewClientKeyUtils()
	encBytes, err := keyUtils2.EncryptAPIKey(testKey, userSecret)
	if err != nil {
		t.Fatalf("Failed to re-encrypt for test: %v", err)
	}
	
	decryptedKey, err := km.DecryptUserKey(encBytes, userSecret)
	if err != nil {
		t.Fatalf("Failed to decrypt created key: %v", err)
	}
	
	if decryptedKey != testKey {
		t.Errorf("Decrypted key doesn't match original. Expected: %s, Got: %s", testKey, decryptedKey)
	}
}

func TestRequestKeyManager_WithOptions(t *testing.T) {
	// Test with custom max age
	customMaxAge := 30 * time.Minute
	km := NewRequestKeyManager(
		WithMaxKeyAge(customMaxAge),
		WithIPValidation(true),
	)
	
	if km.maxKeyAge != customMaxAge {
		t.Errorf("Expected max age %v, got %v", customMaxAge, km.maxKeyAge)
	}
	
	if !km.requireValidIP {
		t.Error("Expected IP validation to be enabled")
	}
}

func TestRequestKeyManager_ExpiredKeys(t *testing.T) {
	// Create key manager
	km := NewRequestKeyManager()
	ctx := context.Background()
	
	keyUtils := NewClientKeyUtils()
	
	// Create credentials
	creds, err := keyUtils.CreateUserCredentials(
		"expired-user",
		"expired-tenant",
		map[string]string{
			"openai": "sk-expired-key",
		},
		"expired-password123",
	)
	if err != nil {
		t.Fatalf("Failed to create credentials: %v", err)
	}
	
	// Manually set expiry time to past
	for i := range creds.Providers {
		creds.Providers[i].ExpiresAt = time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
	}
	
	// Validation should fail due to expired key
	err = km.ValidateUserCredentials(ctx, creds)
	if err == nil {
		t.Error("Validation should fail for expired credentials")
	}
	
	// Extraction should also fail
	_, err = km.ExtractAPIKey(ctx, creds, "openai", "expired-password123")
	if err == nil {
		t.Error("Extraction should fail for expired credentials")
	}
}

// Mock audit logger for testing
type mockAuditLogger struct {
	events []*KeyAccessEvent
}

func (m *mockAuditLogger) LogKeyAccess(ctx context.Context, event *KeyAccessEvent) error {
	m.events = append(m.events, event)
	return nil
}

func TestRequestKeyManager_AuditLogging(t *testing.T) {
	mockLogger := &mockAuditLogger{}
	km := NewRequestKeyManager(WithAuditLogger(mockLogger))
	
	ctx := context.Background()
	keyUtils := NewClientKeyUtils()
	userSecret := "audit-password123"
	
	// Create credentials
	creds, err := keyUtils.CreateUserCredentials(
		"audit-user",
		"audit-tenant",
		map[string]string{
			"openai": "sk-audit-key",
		},
		userSecret,
	)
	if err != nil {
		t.Fatalf("Failed to create credentials: %v", err)
	}
	
	// Perform operations that should generate audit events
	km.ValidateUserCredentials(ctx, creds)
	km.ExtractAPIKey(ctx, creds, "openai", userSecret)
	
	// Check audit events were logged
	if len(mockLogger.events) == 0 {
		t.Error("Expected audit events to be logged")
	}
	
	// Verify event details
	for _, event := range mockLogger.events {
		if event.UserID != "audit-user" {
			t.Errorf("Expected UserID 'audit-user', got %s", event.UserID)
		}
		if event.Action == "" {
			t.Error("Event action should not be empty")
		}
		if event.Timestamp.IsZero() {
			t.Error("Event timestamp should be set")
		}
	}
}