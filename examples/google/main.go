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
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY environment variable is required")
	}

	// Create client configuration with Google
	config := &types.Config{
		Providers: map[string]types.ProviderConfig{
			"google": {
				Enabled: true,
				APIKey:  apiKey,
			},
		},
		Tools: types.ToolsConfig{
			Enabled: true,
			Sandbox: true,
		},
		Defaults: types.DefaultsConfig{
			Provider: "google",
			Model:    "gemini-1.5-flash",
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

	// Example 1: Basic Gemini conversation
	fmt.Println("=== Example 1: Basic Gemini Conversation ===")
	basicRequest := &types.InvocationRequest{
		Model: "gemini-1.5-flash",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Explain the concept of quantum entanglement in simple terms for a high school student.",
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

	fmt.Printf("Gemini says: %s\n\n", response.GetContent())

	// Example 2: Streaming with Gemini
	fmt.Println("=== Example 2: Streaming with Gemini ===")
	streamRequest := &types.InvocationRequest{
		Model: "gemini-1.5-flash",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Write a creative short story about a time-traveling librarian in exactly 3 paragraphs.",
			},
		},
	}

	stream, err := client.Invoke(context.Background(), streamRequest)
	if err != nil {
		log.Fatalf("Failed to invoke streaming: %v", err)
	}
	defer stream.Close()

	fmt.Print("Gemini streams: ")
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

	// Example 3: Function calling with Gemini
	fmt.Println("=== Example 3: Function Calling with Gemini ===")
	toolRequest := &types.InvocationRequest{
		Model: "gemini-1.5-pro",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Calculate the area of a circle with radius 7.5 meters, then tell me what time it is in UTC.",
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

	fmt.Printf("Gemini with tools: %s\n", toolResponse.GetContent())

	// Example 4: System instruction with Gemini
	fmt.Println("=== Example 4: System Instruction with Gemini ===")
	systemRequest := &types.InvocationRequest{
		Model: "gemini-1.5-flash",
		Messages: []types.Message{
			{
				Role:    "system",
				Content: "You are a helpful coding assistant. Always provide code examples in Python and explain each step clearly.",
			},
			{
				Role:    "user",
				Content: "How do I read a CSV file?",
			},
		},
	}

	systemResponse, err := client.InvokeSync(context.Background(), systemRequest)
	if err != nil {
		log.Fatalf("Failed to invoke with system instruction: %v", err)
	}

	fmt.Printf("Gemini with system instruction:\n%s\n", systemResponse.GetContent())

	// Example 5: Gemini Pro for complex reasoning
	fmt.Println("=== Example 5: Gemini Pro for Complex Reasoning ===")
	reasoningRequest := &types.InvocationRequest{
		Model: "gemini-1.5-pro",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Solve this step by step: If a train travels at 80 km/h for 2.5 hours, then slows to 60 km/h for another 1.5 hours, what's the total distance traveled and average speed?",
			},
		},
	}

	reasoningResponse, err := client.InvokeSync(context.Background(), reasoningRequest)
	if err != nil {
		log.Fatalf("Failed to invoke reasoning request: %v", err)
	}

	fmt.Printf("Gemini Pro reasoning:\n%s\n", reasoningResponse.GetContent())

	// List available Google models
	fmt.Println("=== Available Google Models ===")
	models, err := client.ListModels(context.Background())
	if err != nil {
		log.Fatalf("Failed to list models: %v", err)
	}

	for _, model := range models {
		if model.Provider == "google" {
			fmt.Printf("- %s (%s): %s [Context: %d tokens]\n", 
				model.ID, model.Provider, model.Name, model.ContextLimit)
		}
	}
}