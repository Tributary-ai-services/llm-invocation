package main

import (
	"context"
	"fmt"
	"log"

	llminvocation "github.com/tributary-ai/llm-invocation"
	"github.com/tributary-ai/llm-invocation/auth"
	"github.com/tributary-ai/llm-invocation/types"
)

// Example: Basic usage of user-provided API keys with request embedding
func main() {
	fmt.Println("LLM Invocation - User Keys Example")
	
	// Initialize client with minimal config (no backend API keys)
	config := &types.Config{
		Providers: map[string]types.ProviderConfig{
			"openai": {
				Enabled: true,
				APIKey:  "", // No backend key - will use user's key
			},
		},
	}
	
	client, err := llminvocation.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	
	// Example 1: Encrypt and use user API key
	if err := exampleBasicUserKey(client); err != nil {
		log.Printf("Basic example failed: %v", err)
	}
	
	// Example 2: Multiple providers with different keys
	if err := exampleMultipleProviders(client); err != nil {
		log.Printf("Multiple providers example failed: %v", err)
	}
}

func exampleBasicUserKey(client llminvocation.Client) error {
	fmt.Println("\n=== Basic User Key Example ===")
	
	// User's API key (in practice, this would come from user input)
	userAPIKey := "sk-example-openai-key-here"
	userSecret := "my-secure-password123"
	
	// Client-side utilities for key management
	keyUtils := auth.NewClientKeyUtils()
	
	// Create user credentials with encrypted API key
	providerKeys := map[string]string{
		"openai": userAPIKey,
	}
	
	userCredentials, err := keyUtils.CreateUserCredentials(
		"user-123",
		"tenant-456", 
		providerKeys,
		userSecret,
	)
	if err != nil {
		return fmt.Errorf("failed to create user credentials: %w", err)
	}
	
	// Create the invocation request
	baseRequest := &types.InvocationRequest{
		Model: "gpt-3.5-turbo",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "What is the capital of France?",
			},
		},
		Stream: &[]bool{false}[0],
	}
	
	// Create enhanced request with user credentials
	enhancedRequest := keyUtils.CreateEnhancedRequest(
		baseRequest,
		userCredentials,
		false, // Don't use backend keys
	)
	
	// Make the request using user's API key
	ctx := context.Background()
	response, err := client.InvokeSyncWithCredentials(ctx, enhancedRequest, userSecret)
	if err != nil {
		return fmt.Errorf("invocation failed: %w", err)
	}
	
	fmt.Printf("Response: %s\n", response.Content)
	fmt.Printf("Provider: %s\n", response.Provider)
	fmt.Printf("Model: %s\n", response.Model)
	
	return nil
}

func exampleMultipleProviders(client llminvocation.Client) error {
	fmt.Println("\n=== Multiple Providers Example ===")
	
	// User has keys for multiple providers
	userSecrets := map[string]string{
		"openai":    "sk-example-openai-key",
		"anthropic": "sk-ant-example-anthropic-key",
	}
	userPassword := "strong-user-password456"
	
	keyUtils := auth.NewClientKeyUtils()
	
	// Create credentials for multiple providers
	userCredentials, err := keyUtils.CreateUserCredentials(
		"user-456",
		"tenant-789",
		userSecrets,
		userPassword,
	)
	if err != nil {
		return fmt.Errorf("failed to create multi-provider credentials: %w", err)
	}
	
	// Show credential summary (safe for logging)
	summary := keyUtils.CreateCredentialsSummary(userCredentials)
	fmt.Printf("User credentials summary: %+v\n", summary)
	
	// Test different providers
	providers := []string{"openai", "anthropic"}
	
	for _, providerName := range providers {
		if !userCredentials.HasProvider(providerName) {
			fmt.Printf("Skipping %s - no credentials provided\n", providerName)
			continue
		}
		
		fmt.Printf("\nTesting %s provider...\n", providerName)
		
		// Create request for this provider
		baseRequest := &types.InvocationRequest{
			Provider: providerName,
			Model:    getModelForProvider(providerName),
			Messages: []types.Message{
				{
					Role:    "user", 
					Content: fmt.Sprintf("Say hello from %s!", providerName),
				},
			},
			Stream: &[]bool{false}[0],
		}
		
		enhancedRequest := keyUtils.CreateEnhancedRequest(
			baseRequest,
			userCredentials,
			false,
		)
		
		// Make request
		ctx := context.Background()
		response, err := client.InvokeSyncWithCredentials(ctx, enhancedRequest, userPassword)
		if err != nil {
			fmt.Printf("  Failed: %v\n", err)
			continue
		}
		
		fmt.Printf("  Success: %s\n", response.Content[:min(100, len(response.Content))])
	}
	
	return nil
}

func getModelForProvider(provider string) string {
	switch provider {
	case "openai":
		return "gpt-3.5-turbo"
	case "anthropic":
		return "claude-3-haiku-20240307"
	case "google":
		return "gemini-pro"
	default:
		return "default-model"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}