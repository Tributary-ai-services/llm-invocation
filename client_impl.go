package llminvocation

import (
	"context"
	"fmt"

	"github.com/tributary-ai/llm-invocation/auth"
	"github.com/tributary-ai/llm-invocation/providers"
	"github.com/tributary-ai/llm-invocation/providers/anthropic"
	"github.com/tributary-ai/llm-invocation/providers/google"
	"github.com/tributary-ai/llm-invocation/providers/openai"
	"github.com/tributary-ai/llm-invocation/streaming"
	"github.com/tributary-ai/llm-invocation/tools"
	"github.com/tributary-ai/llm-invocation/types"
)

type client struct {
	registry     *providers.Registry
	toolRegistry tools.Registry
	mcpClient    *MCPClient
	keyManager   auth.KeyManager
	config       *types.Config
}

func NewClient(config *types.Config) (Client, error) {
	if config == nil {
		config = types.DefaultConfig()
	}

	c := &client{
		registry:     providers.NewRegistry(),
		toolRegistry: tools.NewRegistry(),
		keyManager:   auth.NewRequestKeyManager(),
		config:       config,
	}

	if err := c.initializeProviders(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *client) initializeProviders() error {
	for name, providerConfig := range c.config.Providers {
		if !providerConfig.Enabled {
			continue
		}

		var provider providers.Provider

		switch name {
		case "openai":
			provider = openai.NewProvider(providerConfig.APIKey, providerConfig.BaseURL)
		case "anthropic":
			provider = anthropic.NewProvider(providerConfig.APIKey, providerConfig.BaseURL)
		case "google":
			provider = google.NewProvider(providerConfig.APIKey, providerConfig.BaseURL)
		default:
			return types.NewError(types.ErrProviderNotFound, fmt.Sprintf("unknown provider: %s", name))
		}

		if err := c.registry.Register(provider); err != nil {
			return fmt.Errorf("failed to register provider %s: %w", name, err)
		}
	}

	return nil
}

func (c *client) Invoke(ctx context.Context, req *types.InvocationRequest) (ResponseStream, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	req.SetDefaults()

	provider, err := c.registry.SelectProvider(req)
	if err != nil {
		return nil, err
	}

	providerReq := c.convertRequest(req, provider)

	streamChan, err := provider.Invoke(ctx, providerReq)
	if err != nil {
		return nil, err
	}

	return streaming.NewResponseStream(streamChan), nil
}

func (c *client) InvokeSync(ctx context.Context, req *types.InvocationRequest) (*types.InvocationResponse, error) {
	stream, err := c.Invoke(ctx, req)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	return stream.Aggregate()
}

func (c *client) InvokeWithCredentials(ctx context.Context, req *types.EnhancedInvocationRequest, userSecret string) (ResponseStream, error) {
	// Validate and process user credentials
	if req.UserCredentials != nil {
		if err := c.keyManager.ValidateUserCredentials(ctx, req.UserCredentials); err != nil {
			return nil, fmt.Errorf("user credential validation failed: %w", err)
		}
	}
	
	// Use enhanced request processing
	return c.invokeEnhanced(ctx, req, userSecret)
}

func (c *client) InvokeSyncWithCredentials(ctx context.Context, req *types.EnhancedInvocationRequest, userSecret string) (*types.InvocationResponse, error) {
	stream, err := c.InvokeWithCredentials(ctx, req, userSecret)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	return stream.Aggregate()
}

func (c *client) invokeEnhanced(ctx context.Context, req *types.EnhancedInvocationRequest, userSecret string) (ResponseStream, error) {
	if err := req.InvocationRequest.Validate(); err != nil {
		return nil, err
	}

	req.InvocationRequest.SetDefaults()

	// Determine which provider to use
	provider, err := c.registry.SelectProvider(req.InvocationRequest)
	if err != nil {
		return nil, err
	}

	// If user credentials are provided, create a temporary provider with user's API key
	if req.UserCredentials != nil && !req.UseBackendKeys {
		apiKey, err := c.keyManager.ExtractAPIKey(ctx, req.UserCredentials, provider.Name(), userSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to extract API key for provider %s: %w", provider.Name(), err)
		}
		
		// Create temporary provider with user's API key
		userProvider, err := c.createProviderWithUserKey(provider.Name(), apiKey, "")
		if err != nil {
			return nil, fmt.Errorf("failed to create provider with user key: %w", err)
		}
		
		provider = userProvider
	}

	// Create provider request
	providerReq := c.convertRequest(req.InvocationRequest, provider)

	streamChan, err := provider.Invoke(ctx, providerReq)
	if err != nil {
		return nil, err
	}

	return streaming.NewResponseStream(streamChan), nil
}

func (c *client) convertEnhancedRequest(ctx context.Context, req *types.EnhancedInvocationRequest, userSecret string, provider providers.Provider) (*types.ProviderRequest, error) {
	// Convert base request
	providerReq := c.convertRequest(req.InvocationRequest, provider)
	
	// If user credentials are provided, we'll modify the provider request
	// to use user keys. The actual invocation will use a temporary provider
	// with the user's API key - this is handled in invokeEnhanced
	
	return providerReq, nil
}

func (c *client) createProviderWithUserKey(providerName, apiKey, baseURL string) (providers.Provider, error) {
	switch providerName {
	case "openai":
		return openai.NewProvider(apiKey, baseURL), nil
	case "anthropic":
		return anthropic.NewProvider(apiKey, baseURL), nil
	case "google":
		return google.NewProvider(apiKey, baseURL), nil
	default:
		return nil, types.NewError(types.ErrProviderNotFound, fmt.Sprintf("unknown provider: %s", providerName))
	}
}

func (c *client) ListModels(ctx context.Context) ([]types.ModelInfo, error) {
	return c.registry.ListModels(ctx)
}

func (c *client) RegisterTool(tool types.Tool) error {
	return c.toolRegistry.Register(tool)
}

func (c *client) RegisterMCPServer(server MCPServer) error {
	if c.mcpClient == nil {
		c.mcpClient = NewMCPClient()
	}
	return c.mcpClient.RegisterServer(server)
}

func (c *client) Close() error {
	return c.registry.Close()
}

func (c *client) convertRequest(req *types.InvocationRequest, provider providers.Provider) *types.ProviderRequest {
	providerReq := &types.ProviderRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   req.Stream != nil && *req.Stream,
		Metadata: req.Metadata,
		Options:  make(map[string]interface{}),
	}

	// Convert ModelOptions to map
	if req.Options.Temperature != nil {
		providerReq.Options["temperature"] = *req.Options.Temperature
	}
	if req.Options.MaxTokens != nil {
		providerReq.Options["max_tokens"] = *req.Options.MaxTokens
	}
	if req.Options.TopP != nil {
		providerReq.Options["top_p"] = *req.Options.TopP
	}
	if req.Options.FrequencyPenalty != nil {
		providerReq.Options["frequency_penalty"] = *req.Options.FrequencyPenalty
	}
	if req.Options.PresencePenalty != nil {
		providerReq.Options["presence_penalty"] = *req.Options.PresencePenalty
	}
	if len(req.Options.Stop) > 0 {
		providerReq.Options["stop"] = req.Options.Stop
	}
	if req.Options.Seed != nil {
		providerReq.Options["seed"] = *req.Options.Seed
	}

	// Add provider-specific options
	if req.Options.ProviderOptions != nil {
		for k, v := range req.Options.ProviderOptions {
			providerReq.Options[k] = v
		}
	}

	// Add tools
	if len(req.Tools) > 0 {
		providerReq.Tools = req.Tools
	}

	return providerReq
}

// MCPClient is a placeholder for MCP functionality
type MCPClient struct {
	servers []MCPServer
}

func NewMCPClient() *MCPClient {
	return &MCPClient{
		servers: make([]MCPServer, 0),
	}
}

func (m *MCPClient) RegisterServer(server MCPServer) error {
	m.servers = append(m.servers, server)
	return nil
}