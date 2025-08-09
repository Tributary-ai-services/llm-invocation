package llminvocation

import (
	"context"
	"testing"

	"github.com/tributary-ai/llm-invocation/auth"
	"github.com/tributary-ai/llm-invocation/types"
)

func TestIntegration_UserKeysEndToEnd(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	
	// Create client with minimal config (no backend keys)
	config := &types.Config{
		Providers: map[string]types.ProviderConfig{
			"openai": {
				Enabled: true,
				APIKey:  "", // No backend key
			},
		},
	}
	
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Test user-provided API key flow
	testUserKeyFlow(t, client)
	
	// Test multiple providers
	testMultipleProvidersFlow(t, client)
	
	// Test security features
	testSecurityFeaturesFlow(t, client)
}

func testUserKeyFlow(t *testing.T, client Client) {
	t.Log("Testing user key flow...")
	
	keyUtils := auth.NewClientKeyUtils()
	
	// Simulate user providing API key
	userAPIKey := "sk-test-user-key-12345" // Mock key for testing
	userSecret := "integration-test-password123"
	
	// Client encrypts the key
	userCredentials, err := keyUtils.CreateUserCredentials(
		"integration-user",
		"integration-tenant",
		map[string]string{
			"openai": userAPIKey,
		},
		userSecret,
	)
	if err != nil {
		t.Fatalf("Failed to create user credentials: %v", err)
	}
	
	// Create request
	baseRequest := &types.InvocationRequest{
		Model: "gpt-3.5-turbo",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "This is a test message",
			},
		},
		Stream: &[]bool{false}[0],
	}
	
	enhancedRequest := keyUtils.CreateEnhancedRequest(
		baseRequest,
		userCredentials,
		false, // Don't use backend keys
	)
	
	// Make request - this will fail with mock key, but should reach provider level
	ctx := context.Background()
	_, err = client.InvokeSyncWithCredentials(ctx, enhancedRequest, userSecret)
	
	// We expect this to fail since we're using a mock key, but it should fail at the provider level
	// not at the credential level
	if err != nil {
		t.Logf("Expected failure with mock key: %v", err)
		// Verify it's not a credential-related error
		if containsCredentialError(err.Error()) {
			t.Errorf("Should not fail with credential error: %v", err)
		}
	}
}

func testMultipleProvidersFlow(t *testing.T, client Client) {
	t.Log("Testing multiple providers flow...")
	
	keyUtils := auth.NewClientKeyUtils()
	userSecret := "multi-provider-password456"
	
	// Multiple provider keys
	providerKeys := map[string]string{
		"openai":    "sk-openai-mock-key",
		"anthropic": "sk-ant-anthropic-mock-key",
	}
	
	userCredentials, err := keyUtils.CreateUserCredentials(
		"multi-provider-user",
		"multi-provider-tenant",
		providerKeys,
		userSecret,
	)
	if err != nil {
		t.Fatalf("Failed to create multi-provider credentials: %v", err)
	}
	
	// Test that we can access different providers
	for provider := range providerKeys {
		t.Logf("Testing provider: %s", provider)
		
		if !userCredentials.HasProvider(provider) {
			t.Errorf("Should have credentials for provider %s", provider)
		}
		
		providerConfig := userCredentials.GetProviderConfig(provider)
		if providerConfig == nil {
			t.Errorf("Should have provider config for %s", provider)
			continue
		}
		
		if len(providerConfig.EncryptedKey) == 0 {
			t.Errorf("Provider %s should have encrypted key", provider)
		}
		
		// Verify strategy
		if providerConfig.GetStrategy() != types.StrategyEmbedded {
			t.Errorf("Provider %s should use embedded strategy", provider)
		}
	}
}

func testSecurityFeaturesFlow(t *testing.T, client Client) {
	t.Log("Testing security features...")
	
	keyUtils := auth.NewClientKeyUtils()
	
	// Test password validation
	weakPasswords := []string{"weak", "123", "nodigits"}
	for _, pwd := range weakPasswords {
		if err := keyUtils.ValidateUserSecret(pwd); err == nil {
			t.Errorf("Password '%s' should fail validation", pwd)
		}
	}
	
	// Test strong password
	strongPassword := "strong-secure-password123"
	if err := keyUtils.ValidateUserSecret(strongPassword); err != nil {
		t.Errorf("Strong password should pass validation: %v", err)
	}
	
	// Test key rotation
	originalKey := "sk-original-test-key"
	oldSecret := "old-secret-password123"
	newSecret := "new-secret-password456"
	
	// Encrypt with old secret
	oldEncrypted, err := keyUtils.EncryptAPIKey(originalKey, oldSecret)
	if err != nil {
		t.Fatalf("Failed to encrypt with old secret: %v", err)
	}
	
	// Rotate key
	newEncrypted, err := keyUtils.RotateEncryptedKey(oldEncrypted, oldSecret, newSecret)
	if err != nil {
		t.Fatalf("Failed to rotate key: %v", err)
	}
	
	// Verify old secret doesn't work
	if err := keyUtils.TestKeyDecryption(newEncrypted, oldSecret); err == nil {
		t.Error("Old secret should not work with rotated key")
	}
	
	// Verify new secret works
	if err := keyUtils.TestKeyDecryption(newEncrypted, newSecret); err != nil {
		t.Errorf("New secret should work with rotated key: %v", err)
	}
	
	// Test credentials summary (safe for logging)
	userCredentials, err := keyUtils.CreateUserCredentials(
		"security-user",
		"security-tenant",
		map[string]string{
			"openai": "sk-security-test-key",
		},
		strongPassword,
	)
	if err != nil {
		t.Fatalf("Failed to create security credentials: %v", err)
	}
	
	summary := keyUtils.CreateCredentialsSummary(userCredentials)
	
	// Verify no sensitive data in summary
	if _, hasKey := summary["api_key"]; hasKey {
		t.Error("Summary should not contain api_key")
	}
	if _, hasKey := summary["encrypted_key"]; hasKey {
		t.Error("Summary should not contain encrypted_key")
	}
	if _, hasKey := summary["user_secret"]; hasKey {
		t.Error("Summary should not contain user_secret")
	}
	
	// Should have safe metadata
	if summary["user_id"] != "security-user" {
		t.Errorf("Expected user_id 'security-user', got %v", summary["user_id"])
	}
	if summary["num_providers"] != 1 {
		t.Errorf("Expected num_providers 1, got %v", summary["num_providers"])
	}
}

func containsCredentialError(errMsg string) bool {
	credentialErrorKeywords := []string{
		"credential",
		"user_id",
		"tenant_id", 
		"validation failed",
		"decrypt",
		"fingerprint",
		"expired",
	}
	
	for _, keyword := range credentialErrorKeywords {
		if containsString(errMsg, keyword) {
			return true
		}
	}
	return false
}

func containsString(str, substr string) bool {
	// Simple contains check - in real code you'd use strings.Contains
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIntegration_BackendFallback(t *testing.T) {
	// Skip if not running integration tests  
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	
	// Create client with backend keys
	config := &types.Config{
		Providers: map[string]types.ProviderConfig{
			"openai": {
				Enabled: true,
				APIKey:  "sk-backend-test-key",
			},
		},
	}
	
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	keyUtils := auth.NewClientKeyUtils()
	
	// Create request with user credentials but UseBackendKeys = true
	baseRequest := &types.InvocationRequest{
		Model: "gpt-3.5-turbo", 
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Test backend fallback",
			},
		},
		Stream: &[]bool{false}[0],
	}
	
	// Create user credentials but use backend keys
	userCredentials, err := keyUtils.CreateUserCredentials(
		"fallback-user",
		"fallback-tenant", 
		map[string]string{
			"openai": "sk-user-key",
		},
		"fallback-password123",
	)
	if err != nil {
		t.Fatalf("Failed to create user credentials: %v", err)
	}
	
	enhancedRequest := &types.EnhancedInvocationRequest{
		InvocationRequest: baseRequest,
		UserCredentials:   userCredentials,
		UseBackendKeys:    true, // Use backend keys instead of user keys
	}
	
	ctx := context.Background()
	_, err = client.InvokeSyncWithCredentials(ctx, enhancedRequest, "fallback-password123")
	
	// Should attempt to use backend key (will fail with mock key but not due to user credentials)
	if err != nil {
		t.Logf("Expected failure with mock backend key: %v", err)
		if containsCredentialError(err.Error()) {
			t.Errorf("Should not fail with user credential error when using backend keys: %v", err)
		}
	}
}