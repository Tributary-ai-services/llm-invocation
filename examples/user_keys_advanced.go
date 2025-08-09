package main

import (
	"context"
	"fmt"
	"log"
	"time"

	llminvocation "github.com/tributary-ai/llm-invocation"
	"github.com/tributary-ai/llm-invocation/auth"
	"github.com/tributary-ai/llm-invocation/types"
)

// Example: Advanced user key management features
func main() {
	fmt.Println("LLM Invocation - Advanced User Keys Example")
	
	config := &types.Config{
		Providers: map[string]types.ProviderConfig{
			"openai": {
				Enabled: true,
				APIKey:  "", // User will provide key
			},
		},
	}
	
	client, err := llminvocation.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Example 1: Key validation and testing
	if err := exampleKeyValidation(); err != nil {
		log.Printf("Key validation example failed: %v", err)
	}
	
	// Example 2: Key rotation
	if err := exampleKeyRotation(); err != nil {
		log.Printf("Key rotation example failed: %v", err)
	}
	
	// Example 3: Streaming with user keys
	if err := exampleStreaming(client); err != nil {
		log.Printf("Streaming example failed: %v", err)
	}
	
	// Example 4: Security features
	if err := exampleSecurityFeatures(); err != nil {
		log.Printf("Security example failed: %v", err)
	}
}

func exampleKeyValidation() error {
	fmt.Println("\n=== Key Validation Example ===")
	
	keyUtils := auth.NewClientKeyUtils()
	
	// Test password validation
	passwords := []string{
		"weak",
		"better123",
		"strong-password-with-numbers456",
	}
	
	for _, password := range passwords {
		err := keyUtils.ValidateUserSecret(password)
		if err != nil {
			fmt.Printf("❌ Password '%s' failed validation: %v\n", password, err)
		} else {
			fmt.Printf("✅ Password '%s' passed validation\n", password)
		}
	}
	
	// Generate secure password
	securePassword, err := keyUtils.GenerateSecureSecret(20)
	if err != nil {
		return fmt.Errorf("failed to generate secure password: %w", err)
	}
	fmt.Printf("Generated secure password: %s (validation: ", securePassword)
	
	if err := keyUtils.ValidateUserSecret(securePassword); err != nil {
		fmt.Printf("failed - %v)\n", err)
	} else {
		fmt.Printf("passed)\n")
	}
	
	// Test key encryption/decryption
	testAPIKey := "sk-test-api-key-for-validation"
	userSecret := "validation-password123"
	
	encryptedKey, err := keyUtils.EncryptAPIKey(testAPIKey, userSecret)
	if err != nil {
		return fmt.Errorf("failed to encrypt key: %w", err)
	}
	
	// Test decryption with correct password
	if err := keyUtils.TestKeyDecryption(encryptedKey, userSecret); err != nil {
		fmt.Printf("❌ Decryption test with correct password failed: %v\n", err)
	} else {
		fmt.Printf("✅ Decryption test with correct password passed\n")
	}
	
	// Test decryption with wrong password
	if err := keyUtils.TestKeyDecryption(encryptedKey, "wrong-password"); err != nil {
		fmt.Printf("✅ Decryption test with wrong password correctly failed: %v\n", err)
	} else {
		fmt.Printf("❌ Decryption test with wrong password should have failed!\n")
	}
	
	return nil
}

func exampleKeyRotation() error {
	fmt.Println("\n=== Key Rotation Example ===")
	
	keyUtils := auth.NewClientKeyUtils()
	
	// Original credentials
	originalAPIKey := "sk-original-api-key"
	originalSecret := "original-password123"
	
	// Create encrypted key
	originalEncryptedKey, err := keyUtils.EncryptAPIKey(originalAPIKey, originalSecret)
	if err != nil {
		return fmt.Errorf("failed to encrypt original key: %w", err)
	}
	
	fmt.Printf("Original encrypted key created (size: %d bytes)\n", len(originalEncryptedKey))
	
	// Rotate to new secret
	newSecret := "new-rotated-password456"
	
	rotatedKey, err := keyUtils.RotateEncryptedKey(originalEncryptedKey, originalSecret, newSecret)
	if err != nil {
		return fmt.Errorf("failed to rotate key: %w", err)
	}
	
	fmt.Printf("Key rotated successfully (size: %d bytes)\n", len(rotatedKey))
	
	// Verify old secret no longer works
	if err := keyUtils.TestKeyDecryption(rotatedKey, originalSecret); err != nil {
		fmt.Printf("✅ Old secret correctly no longer works: %v\n", err)
	} else {
		fmt.Printf("❌ Old secret should not work with rotated key!\n")
	}
	
	// Verify new secret works
	if err := keyUtils.TestKeyDecryption(rotatedKey, newSecret); err != nil {
		fmt.Printf("❌ New secret should work: %v\n", err)
	} else {
		fmt.Printf("✅ New secret works with rotated key\n")
	}
	
	return nil
}

func exampleStreaming(client llminvocation.Client) error {
	fmt.Println("\n=== Streaming with User Keys Example ===")
	
	keyUtils := auth.NewClientKeyUtils()
	userAPIKey := "sk-user-streaming-key"
	userSecret := "streaming-password789"
	
	// Create user credentials
	providerKeys := map[string]string{
		"openai": userAPIKey,
	}
	
	userCredentials, err := keyUtils.CreateUserCredentials(
		"streaming-user", 
		"streaming-tenant",
		providerKeys,
		userSecret,
	)
	if err != nil {
		return fmt.Errorf("failed to create streaming credentials: %w", err)
	}
	
	// Create streaming request
	baseRequest := &types.InvocationRequest{
		Model: "gpt-3.5-turbo",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Count from 1 to 5, putting each number on a new line.",
			},
		},
		Stream: &[]bool{true}[0], // Enable streaming
	}
	
	enhancedRequest := keyUtils.CreateEnhancedRequest(
		baseRequest,
		userCredentials,
		false,
	)
	
	// Start streaming
	ctx := context.Background()
	stream, err := client.InvokeWithCredentials(ctx, enhancedRequest, userSecret)
	if err != nil {
		return fmt.Errorf("failed to start streaming: %w", err)
	}
	defer stream.Close()
	
	fmt.Printf("Streaming response:\n")
	for {
		chunk, err := stream.Next()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("streaming error: %w", err)
		}
		
		if chunk.Content != "" {
			fmt.Printf("%s", chunk.Content)
		}
	}
	fmt.Printf("\nStreaming complete.\n")
	
	return nil
}

func exampleSecurityFeatures() error {
	fmt.Println("\n=== Security Features Example ===")
	
	keyUtils := auth.NewClientKeyUtils()
	
	// Create credentials with security options
	userCredentials, err := keyUtils.CreateUserCredentials(
		"security-user",
		"security-tenant",
		map[string]string{
			"openai": "sk-secure-api-key",
		},
		"secure-password123",
	)
	if err != nil {
		return fmt.Errorf("failed to create secure credentials: %w", err)
	}
	
	// Create request with enhanced security options
	baseRequest := &types.InvocationRequest{
		Model: "gpt-3.5-turbo",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Hello, secure world!",
			},
		},
	}
	
	// Add security options
	enhancedRequest := &types.EnhancedInvocationRequest{
		InvocationRequest: baseRequest,
		UserCredentials:   userCredentials,
		UseBackendKeys:    false,
		SecurityOptions: &types.SecurityOptions{
			RequireEncryption: true,
			MaxKeyAge:        1 * time.Hour, // Keys expire in 1 hour
			AllowedIPs:       []string{"127.0.0.1", "10.0.0.0/8"},
			RateLimit: &types.RateLimit{
				RequestsPerMinute: 100,
				TokensPerMinute:   10000,
				BurstLimit:        10,
			},
		},
	}
	
	fmt.Printf("Created secure request with:\n")
	fmt.Printf("- Encryption required: %v\n", enhancedRequest.SecurityOptions.RequireEncryption)
	fmt.Printf("- Max key age: %v\n", enhancedRequest.SecurityOptions.MaxKeyAge)
	fmt.Printf("- Allowed IPs: %v\n", enhancedRequest.SecurityOptions.AllowedIPs)
	fmt.Printf("- Rate limits: %+v\n", enhancedRequest.SecurityOptions.RateLimit)
	
	// Test key expiry checking
	for _, provider := range userCredentials.Providers {
		isExpired := keyUtils.IsKeyExpired(&provider, 1*time.Hour)
		fingerprint := keyUtils.GetKeyFingerprint(&provider)
		
		fmt.Printf("Provider %s: expired=%v, fingerprint=%s\n", 
			provider.Provider, isExpired, fingerprint)
	}
	
	// Create safe summary for logging
	summary := keyUtils.CreateCredentialsSummary(userCredentials)
	fmt.Printf("Safe credentials summary for logging:\n%+v\n", summary)
	
	return nil
}