# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go package for LLM invocation that provides a unified interface for interacting with multiple LLM providers (OpenAI, Anthropic, Google, etc.). The package is designed to be a generic LLM client for tas-llm-router with support for:

- **Provider Abstraction**: Support for multiple LLM providers through a common interface
- **Streaming by Default**: All LLM invocations support streaming responses  
- **Tools & Functions**: Support for function calling, tools, and MCP (Model Context Protocol)
- **Type Safety**: Strongly-typed interfaces for requests and responses
- **Per-Request Configuration**: Model-specific options on each request

## Project Status

**IMPLEMENTED**: The core functionality is now implemented! This includes:

✅ **Core Infrastructure**:
- Go module with proper project structure
- Type-safe request/response types
- Error handling with provider-specific errors
- Configuration management

✅ **Streaming Support**:
- Full streaming infrastructure with ResponseStream interface
- Stream aggregation for collecting complete responses
- Both sync and async invocation methods

✅ **Provider System**:
- Provider registry for managing multiple LLM providers
- OpenAI provider with full GPT-3.5/GPT-4 support
- Automatic provider selection based on model
- Feature detection (function calling, vision, etc.)

✅ **Tool System**:
- Tool registry and execution framework
- Basic sandboxing and timeout protection
- Example tools (Calculator, Time)
- JSON schema parameter validation

✅ **Examples & Documentation**:
- Working examples in `examples/` directory
- Comprehensive documentation and README
- Unit tests for core components

## Development Commands

```bash
go mod tidy          # Update dependencies
go build ./...       # Build all packages
go test ./...        # Run all tests
go test -race ./...  # Run tests with race detection
go vet ./...         # Run static analysis

# Run examples (requires API keys)
cd examples/basic && go run main.go
cd examples/streaming && go run main.go
```

## Planned Architecture

Based on the design document, the package will follow this structure:

```
github.com/tributary-ai/llm-invocation/
├── client.go           # Main client interface
├── providers/          # Provider implementations (OpenAI, Anthropic, Google)
├── streaming/          # Streaming utilities and aggregation
├── tools/              # Tools, function calling, and MCP support
├── types/              # Common request/response types
└── examples/           # Usage examples
```

### Key Design Principles

- **Streaming First**: Default to streaming responses for all providers
- **Provider Agnostic**: Unified interface abstracts provider differences
- **Tool Integration**: Built-in support for function calling and MCP protocol
- **Configuration Flexibility**: Per-request and per-provider configuration options
- **Type Safety**: Strong typing throughout the API surface

## Important Implementation Notes

When implementing this package:

1. **Provider Registry**: Implement a registry system for managing multiple LLM providers
2. **Stream Aggregation**: Create utilities to collect streaming chunks into complete responses  
3. **Tool Execution**: Implement secure tool execution with proper sandboxing
4. **Error Handling**: Provide consistent error types across all providers
5. **Configuration Management**: Support both global and per-request configuration options

## Security Considerations

- Validate all inputs before sending to LLM providers
- Implement secure API key management and rotation
- Sandbox tool execution to prevent malicious code execution
- Add rate limiting and audit logging capabilities