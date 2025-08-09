package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	llm "github.com/tributary-ai/llm-invocation"
	"github.com/tributary-ai/llm-invocation/types"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	config := &types.Config{
		Providers: map[string]types.ProviderConfig{
			"openai": {
				Enabled: true,
				APIKey:  apiKey,
			},
		},
	}

	client, err := llm.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Streaming example with type-writer effect
	fmt.Println("=== Streaming Example ===")
	request := &types.InvocationRequest{
		Model: "gpt-3.5-turbo",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "Write a short story about a robot learning to paint, in about 200 words.",
			},
		},
		Options: types.ModelOptions{
			Temperature: func() *float32 { t := float32(0.8); return &t }(),
			MaxTokens:   func() *int { t := 300; return &t }(),
		},
	}

	stream, err := client.Invoke(context.Background(), request)
	if err != nil {
		log.Fatalf("Failed to invoke streaming: %v", err)
	}
	defer stream.Close()

	fmt.Print("Response: ")
	startTime := time.Now()
	totalChunks := 0

	for {
		chunk, err := stream.Next()
		if err != nil {
			break
		}

		totalChunks++

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				fmt.Print(content)
				
				// Add a small delay for typewriter effect
				time.Sleep(10 * time.Millisecond)
			}
		}

		// Print usage info at the end
		if chunk.Done && chunk.Usage != nil {
			fmt.Printf("\n\n--- Streaming Stats ---\n")
			fmt.Printf("Duration: %v\n", time.Since(startTime))
			fmt.Printf("Chunks received: %d\n", totalChunks)
			fmt.Printf("Tokens used: %d prompt + %d completion = %d total\n",
				chunk.Usage.PromptTokens,
				chunk.Usage.CompletionTokens,
				chunk.Usage.TotalTokens)
		}
	}

	// Also demonstrate aggregation
	fmt.Println("\n\n=== Aggregation Example ===")
	stream2, err := client.Invoke(context.Background(), &types.InvocationRequest{
		Model: "gpt-3.5-turbo",
		Messages: []types.Message{
			{
				Role:    "user",
				Content: "List 5 benefits of renewable energy.",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to invoke second stream: %v", err)
	}

	// Collect all chunks into a complete response
	response, err := stream2.Aggregate()
	if err != nil {
		log.Fatalf("Failed to aggregate stream: %v", err)
	}

	fmt.Printf("Aggregated response:\n%s\n", response.GetContent())
	fmt.Printf("Total tokens: %d\n", response.Usage.TotalTokens)
}