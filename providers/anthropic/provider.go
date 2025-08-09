package anthropic

import (
	"context"
	"fmt"

	"github.com/tributary-ai/llm-invocation/providers"
	"github.com/tributary-ai/llm-invocation/types"
)

type Provider struct {
	*providers.BaseProvider
	client *Client
}

func NewProvider(apiKey, baseURL string) *Provider {
	models := []types.ModelInfo{
		{ID: "claude-3-5-sonnet-20241022", Provider: "anthropic", Name: "Claude 3.5 Sonnet", Features: []string{"function_calling", "vision", "structured_output"}, ContextLimit: 200000},
		{ID: "claude-3-5-haiku-20241022", Provider: "anthropic", Name: "Claude 3.5 Haiku", Features: []string{"function_calling", "vision"}, ContextLimit: 200000},
		{ID: "claude-3-opus-20240229", Provider: "anthropic", Name: "Claude 3 Opus", Features: []string{"function_calling", "vision"}, ContextLimit: 200000},
		{ID: "claude-3-sonnet-20240229", Provider: "anthropic", Name: "Claude 3 Sonnet", Features: []string{"function_calling", "vision"}, ContextLimit: 200000},
		{ID: "claude-3-haiku-20240307", Provider: "anthropic", Name: "Claude 3 Haiku", Features: []string{"function_calling"}, ContextLimit: 200000},
	}

	features := []providers.Feature{
		providers.FeatureFunctionCalling,
		providers.FeatureVision,
		providers.FeatureStructuredOutput,
		providers.FeatureStreaming,
	}

	return &Provider{
		BaseProvider: providers.NewBaseProvider("anthropic", models, features),
		client:       NewClient(apiKey, baseURL),
	}
}

func (p *Provider) Invoke(ctx context.Context, req *types.ProviderRequest) (<-chan *types.StreamChunk, error) {
	if err := p.ValidateRequest(req); err != nil {
		return nil, err
	}

	anthropicReq := MapToAnthropicRequest(req)

	if req.Stream {
		return p.client.CreateMessageStream(ctx, anthropicReq)
	}

	// For non-streaming, we still return a channel for consistency
	ch := make(chan *types.StreamChunk, 1)

	go func() {
		defer close(ch)

		resp, err := p.client.CreateMessage(ctx, anthropicReq)
		if err != nil {
			ch <- &types.StreamChunk{
				Error: err.(*types.Error),
			}
			return
		}

		// Convert sync response to single stream chunk
		invocationResp := MapFromAnthropicResponse(resp)
		chunk := &types.StreamChunk{
			ID:       invocationResp.ID,
			Provider: invocationResp.Provider,
			Model:    invocationResp.Model,
			Usage:    invocationResp.Usage,
			Done:     true,
		}

		// Convert choices to delta format
		chunk.Choices = make([]types.DeltaChoice, len(invocationResp.Choices))
		for i, choice := range invocationResp.Choices {
			content := ""
			if c, ok := choice.Message.Content.(string); ok {
				content = c
			}

			chunk.Choices[i] = types.DeltaChoice{
				Index: choice.Index,
				Delta: &types.Delta{
					Role:      choice.Message.Role,
					Content:   content,
					ToolCalls: choice.Message.ToolCalls,
				},
				FinishReason: choice.FinishReason,
			}
		}

		ch <- chunk
	}()

	return ch, nil
}

func (p *Provider) ValidateRequest(req *types.ProviderRequest) error {
	if err := p.BaseProvider.ValidateRequest(req); err != nil {
		return err
	}

	// Anthropic-specific validations
	if req.Model == "" {
		return types.NewProviderError("anthropic", types.ErrInvalidRequest, "model is required")
	}

	// Check if model is supported
	supported := false
	for _, model := range p.Models() {
		if model.ID == req.Model {
			supported = true
			break
		}
	}

	if !supported {
		return types.NewProviderError("anthropic", types.ErrModelNotSupported, "model "+req.Model+" is not supported")
	}

	// Validate messages format for Anthropic
	if err := p.validateMessages(req.Messages); err != nil {
		return types.NewProviderError("anthropic", types.ErrInvalidRequest, err.Error())
	}

	return nil
}

func (p *Provider) validateMessages(messages []types.Message) error {
	if len(messages) == 0 {
		return fmt.Errorf("at least one message is required")
	}

	// Anthropic requires alternating user/assistant messages (after system)
	var lastRole string
	for i, msg := range messages {
		if msg.Role == "system" {
			if i != 0 {
				return fmt.Errorf("system message must be first")
			}
			continue
		}

		if msg.Role != "user" && msg.Role != "assistant" {
			return fmt.Errorf("invalid role: %s (must be user or assistant)", msg.Role)
		}

		if lastRole != "" && lastRole == msg.Role {
			return fmt.Errorf("messages must alternate between user and assistant roles")
		}

		lastRole = msg.Role
	}

	// First non-system message must be from user
	for _, msg := range messages {
		if msg.Role != "system" {
			if msg.Role != "user" {
				return fmt.Errorf("first message (after system) must be from user")
			}
			break
		}
	}

	return nil
}

func (p *Provider) Close() error {
	// Anthropic client doesn't need explicit cleanup
	return nil
}