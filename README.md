# LLM Invocation Package

A Go package for unified LLM provider interactions with streaming support, tools, and MCP protocol capabilities.

## Features

- **Multi-Provider Support**: OpenAI, Anthropic, Google (extensible)
- **Streaming by Default**: All responses support streaming
- **Tool System**: Function calling with sandboxed execution
- **Type Safety**: Strongly-typed interfaces throughout
- **Configuration Flexibility**: Per-request and global configuration
- **MCP Protocol**: Model Context Protocol support (planned)

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    llm "github.com/tributary-ai/llm-invocation"
    "github.com/tributary-ai/llm-invocation/types"
)

func main() {
    // Create client with multiple providers
    client, err := llm.NewClient(&types.Config{
        Providers: map[string]types.ProviderConfig{
            "openai": {
                Enabled: true,
                APIKey:  os.Getenv("OPENAI_API_KEY"),
            },
            "anthropic": {
                Enabled: true,
                APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
            },
            "google": {
                Enabled: true,
                APIKey:  os.Getenv("GOOGLE_API_KEY"),
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Auto-selects provider based on model
    response, err := client.InvokeSync(context.Background(), &types.InvocationRequest{
        Model: "gemini-1.5-flash", // or "gpt-3.5-turbo" or "claude-3-5-haiku-20241022"
        Messages: []types.Message{{
            Role:    "user",
            Content: "Hello, world!",
        }},
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.GetContent())
}
```

## Streaming

```go
stream, err := client.Invoke(context.Background(), request)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    chunk, err := stream.Next()
    if err != nil {
        break
    }
    
    if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
        fmt.Print(chunk.Choices[0].Delta.Content)
    }
}
```

## Tools

```go
// Implement the Tool interface
type CalculatorTool struct{}

func (c *CalculatorTool) Name() string { return "calculator" }
func (c *CalculatorTool) Description() string { return "Basic math operations" }
func (c *CalculatorTool) Parameters() map[string]interface{} { /* JSON schema */ }
func (c *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    // Tool logic here
}

// Register and use
client.RegisterTool(&CalculatorTool{})
```

## Installation

```bash
go get github.com/tributary-ai/llm-invocation
```

## Examples

See the `examples/` directory for complete working examples:

- `examples/basic/` - Multi-provider usage with sync and async calls
- `examples/streaming/` - Advanced streaming with typewriter effects  
- `examples/anthropic/` - Claude-specific examples with system messages
- `examples/google/` - Gemini-specific examples with function calling
- `examples/tools/` - Custom tool implementation

## Supported Providers

- ✅ **OpenAI** (GPT-3.5, GPT-4, function calling, vision)
- ✅ **Anthropic** (Claude 3.5 Sonnet/Haiku, Claude 3 Opus/Sonnet/Haiku, function calling, vision)
- ✅ **Google** (Gemini 1.5 Pro/Flash, Gemini Pro/Pro Vision, function calling, vision, 2M context)

## Contributing

This package is actively under development. See the design document for implementation details and roadmap.

## License

Apache 2.0 - see LICENSE file for details.
