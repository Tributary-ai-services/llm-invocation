package main

import (
	"context"
	"fmt"
	"log"
	"os"

	llm "github.com/tributary-ai/llm-invocation"
	"github.com/tributary-ai/llm-invocation/tools"
	"github.com/tributary-ai/llm-invocation/types"
)

func main() {
	// Get API keys from environment
	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	googleKey := os.Getenv("GOOGLE_API_KEY")
	
	if openaiKey == "" && anthropicKey == "" && googleKey == "" {
		log.Fatal("At least one API key is required: OPENAI_API_KEY, ANTHROPIC_API_KEY, or GOOGLE_API_KEY")
	}

	// Create client configuration with multiple providers
	config := &types.Config{
		Providers: map[string]types.ProviderConfig{},
		Tools: types.ToolsConfig{
			Enabled: true,
			Sandbox: true,
		},
		Defaults: types.DefaultsConfig{
			Model: "gpt-3.5-turbo",
		},
	}

	// Add OpenAI if key is available
	if openaiKey != "" {
		config.Providers["openai"] = types.ProviderConfig{
			Enabled: true,
			APIKey:  openaiKey,
		}
		config.Defaults.Provider = "openai"
	}

	// Add Anthropic if key is available
	if anthropicKey != "" {
		config.Providers["anthropic"] = types.ProviderConfig{
			Enabled: true,
			APIKey:  anthropicKey,
		}
		// If no OpenAI, use Anthropic as default
		if openaiKey == "" {
			config.Defaults.Provider = "anthropic"
			config.Defaults.Model = "claude-3-5-haiku-20241022"
		}
	}

	// Add Google if key is available
	if googleKey != "" {
		config.Providers["google"] = types.ProviderConfig{
			Enabled: true,
			APIKey:  googleKey,
		}
		// If no other providers, use Google as default
		if openaiKey == "" && anthropicKey == "" {
			config.Defaults.Provider = "google"
			config.Defaults.Model = "gemini-1.5-flash"
		}
	}

	// Create client
	client, err := llm.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Register some tools
	calculator := &tools.CalculatorTool{}
	timeTool := &tools.TimeTool{}

	if err := client.RegisterTool(calculator); err != nil {
		log.Fatalf("Failed to register calculator tool: %v", err)
	}

	if err := client.RegisterTool(timeTool); err != nil {
		log.Fatalf("Failed to register time tool: %v", err)
	}

	// Example 1: Basic text generation
	fmt.Println("=== Example 1: Basic Text Generation ===")
	basicRequest := &types.InvocationRequest{
		Model: "gpt-3.5-turbo",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Explain quantum computing in simple terms, in about 100 words.",
			},
		},
		Options: types.ModelOptions{
			Temperature: func() *float32 { t := float32(0.7); return &t }(),
			MaxTokens:   func() *int { t := 150; return &t }(),
		},
	}

	response, err := client.InvokeSync(context.Background(), basicRequest)
	if err != nil {
		log.Fatalf("Failed to invoke: %v", err)
	}

	fmt.Printf("Response: %s\n\n", response.GetContent())

	// Example 2: Streaming response
	fmt.Println("=== Example 2: Streaming Response ===")
	streamRequest := &types.InvocationRequest{
		Model: "gpt-3.5-turbo",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Write a haiku about programming.",
			},
		},
	}

	stream, err := client.Invoke(context.Background(), streamRequest)
	if err != nil {
		log.Fatalf("Failed to invoke streaming: %v", err)
	}
	defer stream.Close()

	fmt.Print("Streaming response: ")
	for {
		chunk, err := stream.Next()
		if err != nil {
			break
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	fmt.Println()

	// Example 3: Function calling (with tools)
	fmt.Println("=== Example 3: Function Calling ===")
	toolRequest := &types.InvocationRequest{
		Model: "gpt-3.5-turbo",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "What's 15.5 multiplied by 7.2? Also, what time is it in New York?",
			},
		},
		Tools: []types.Tool{
			calculator,
			timeTool,
		},
	}

	toolResponse, err := client.InvokeSync(context.Background(), toolRequest)
	if err != nil {
		log.Fatalf("Failed to invoke with tools: %v", err)
	}

	fmt.Printf("Tool response: %s\n", toolResponse.GetContent())

	// Example 4: Cross-provider comparison
	availableProviders := 0
	if openaiKey != "" { availableProviders++ }
	if anthropicKey != "" { availableProviders++ }
	if googleKey != "" { availableProviders++ }
	
	if availableProviders >= 2 {
		fmt.Println("=== Example 4: Cross-Provider Comparison ===")
		
		question := "Explain recursion in programming in one sentence."
		
		// Ask OpenAI
		if openaiKey != "" {
			openaiResp, err := client.InvokeSync(context.Background(), &types.InvocationRequest{
				Provider: "openai",
				Model:    "gpt-3.5-turbo",
				Messages: []types.Message{{Role: "user", Content: question}},
			})
			if err != nil {
				log.Printf("OpenAI error: %v", err)
			} else {
				fmt.Printf("OpenAI: %s\n\n", openaiResp.GetContent())
			}
		}
		
		// Ask Claude
		if anthropicKey != "" {
			claudeResp, err := client.InvokeSync(context.Background(), &types.InvocationRequest{
				Provider: "anthropic",
				Model:    "claude-3-5-haiku-20241022",
				Messages: []types.Message{{Role: "user", Content: question}},
			})
			if err != nil {
				log.Printf("Anthropic error: %v", err)
			} else {
				fmt.Printf("Claude: %s\n\n", claudeResp.GetContent())
			}
		}
		
		// Ask Gemini
		if googleKey != "" {
			geminiResp, err := client.InvokeSync(context.Background(), &types.InvocationRequest{
				Provider: "google",
				Model:    "gemini-1.5-flash",
				Messages: []types.Message{{Role: "user", Content: question}},
			})
			if err != nil {
				log.Printf("Google error: %v", err)
			} else {
				fmt.Printf("Gemini: %s\n", geminiResp.GetContent())
			}
		}
	}

	// List available models
	fmt.Println("=== Available Models ===")
	models, err := client.ListModels(context.Background())
	if err != nil {
		log.Fatalf("Failed to list models: %v", err)
	}

	for _, model := range models {
		fmt.Printf("- %s (%s): %s\n", model.ID, model.Provider, model.Name)
	}
}