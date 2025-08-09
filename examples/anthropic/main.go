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
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Create client configuration with Anthropic
	config := &types.Config{
		Providers: map[string]types.ProviderConfig{
			"anthropic": {
				Enabled: true,
				APIKey:  apiKey,
			},
		},
		Tools: types.ToolsConfig{
			Enabled: true,
			Sandbox: true,
		},
		Defaults: types.DefaultsConfig{
			Provider: "anthropic",
			Model:    "claude-3-5-sonnet-20241022",
		},
	}

	// Create client
	client, err := llm.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Register tools
	calculator := &tools.CalculatorTool{}
	timeTool := &tools.TimeTool{}

	if err := client.RegisterTool(calculator); err != nil {
		log.Fatalf("Failed to register calculator tool: %v", err)
	}

	if err := client.RegisterTool(timeTool); err != nil {
		log.Fatalf("Failed to register time tool: %v", err)
	}

	// Example 1: Basic Claude conversation
	fmt.Println("=== Example 1: Basic Claude Conversation ===")
	basicRequest := &types.InvocationRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Explain the difference between machine learning and deep learning in simple terms.",
			},
		},
		Options: types.ModelOptions{
			Temperature: func() *float32 { t := float32(0.7); return &t }(),
			MaxTokens:   func() *int { t := 200; return &t }(),
		},
	}

	response, err := client.InvokeSync(context.Background(), basicRequest)
	if err != nil {
		log.Fatalf("Failed to invoke: %v", err)
	}

	fmt.Printf("Claude says: %s\n\n", response.GetContent())

	// Example 2: Streaming with Claude
	fmt.Println("=== Example 2: Streaming with Claude ===")
	streamRequest := &types.InvocationRequest{
		Model: "claude-3-5-haiku-20241022",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Write a short poem about artificial intelligence and creativity.",
			},
		},
	}

	stream, err := client.Invoke(context.Background(), streamRequest)
	if err != nil {
		log.Fatalf("Failed to invoke streaming: %v", err)
	}
	defer stream.Close()

	fmt.Print("Claude streams: ")
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

	// Example 3: Function calling with Claude
	fmt.Println("=== Example 3: Function Calling with Claude ===")
	toolRequest := &types.InvocationRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "What's 23.7 multiplied by 8.9? Also, what's the current time in Tokyo timezone?",
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

	fmt.Printf("Claude with tools: %s\n", toolResponse.GetContent())

	// Example 4: System message with Claude
	fmt.Println("=== Example 4: System Message with Claude ===")
	systemRequest := &types.InvocationRequest{
		Model: "claude-3-5-sonnet-20241022",
		Messages: []types.Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant that always responds in haiku format.",
			},
			{
				Role:    "user",
				Content: "Explain why the sky is blue.",
			},
		},
	}

	systemResponse, err := client.InvokeSync(context.Background(), systemRequest)
	if err != nil {
		log.Fatalf("Failed to invoke with system message: %v", err)
	}

	fmt.Printf("Claude with system message:\n%s\n", systemResponse.GetContent())

	// List available models
	fmt.Println("=== Available Models ===")
	models, err := client.ListModels(context.Background())
	if err != nil {
		log.Fatalf("Failed to list models: %v", err)
	}

	for _, model := range models {
		if model.Provider == "anthropic" {
			fmt.Printf("- %s (%s): %s [Context: %d tokens]\n", 
				model.ID, model.Provider, model.Name, model.ContextLimit)
		}
	}
}