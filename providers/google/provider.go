package google

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
		{ID: "gemini-1.5-pro", Provider: "google", Name: "Gemini 1.5 Pro", Features: []string{"function_calling", "vision", "structured_output"}, ContextLimit: 2000000},
		{ID: "gemini-1.5-pro-latest", Provider: "google", Name: "Gemini 1.5 Pro (Latest)", Features: []string{"function_calling", "vision", "structured_output"}, ContextLimit: 2000000},
		{ID: "gemini-1.5-flash", Provider: "google", Name: "Gemini 1.5 Flash", Features: []string{"function_calling", "vision"}, ContextLimit: 1000000},
		{ID: "gemini-1.5-flash-latest", Provider: "google", Name: "Gemini 1.5 Flash (Latest)", Features: []string{"function_calling", "vision"}, ContextLimit: 1000000},
		{ID: "gemini-pro", Provider: "google", Name: "Gemini Pro", Features: []string{"function_calling"}, ContextLimit: 32000},
		{ID: "gemini-pro-vision", Provider: "google", Name: "Gemini Pro Vision", Features: []string{"function_calling", "vision"}, ContextLimit: 16000},
	}

	features := []providers.Feature{
		providers.FeatureFunctionCalling,
		providers.FeatureVision,
		providers.FeatureStructuredOutput,
		providers.FeatureStreaming,
	}

	return &Provider{
		BaseProvider: providers.NewBaseProvider("google", models, features),
		client:       NewClient(apiKey, baseURL),
	}
}

func (p *Provider) Invoke(ctx context.Context, req *types.ProviderRequest) (<-chan *types.StreamChunk, error) {
	if err := p.ValidateRequest(req); err != nil {
		return nil, err
	}

	googleReq := MapToGoogleRequest(req)

	if req.Stream {
		return p.client.GenerateContentStream(ctx, req.Model, googleReq)
	}

	// For non-streaming, we still return a channel for consistency
	ch := make(chan *types.StreamChunk, 1)

	go func() {
		defer close(ch)

		resp, err := p.client.GenerateContent(ctx, req.Model, googleReq)
		if err != nil {
			ch <- &types.StreamChunk{
				Error: err.(*types.Error),
			}
			return
		}

		// Convert sync response to single stream chunk
		invocationResp := MapFromGoogleResponse(resp, req.Model)
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

	// Google-specific validations
	if req.Model == "" {
		return types.NewProviderError("google", types.ErrInvalidRequest, "model is required")
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
		return types.NewProviderError("google", types.ErrModelNotSupported, "model "+req.Model+" is not supported")
	}

	// Validate messages format for Google
	if err := p.validateMessages(req.Messages); err != nil {
		return types.NewProviderError("google", types.ErrInvalidRequest, err.Error())
	}

	return nil
}

func (p *Provider) validateMessages(messages []types.Message) error {
	if len(messages) == 0 {
		return fmt.Errorf("at least one message is required")
	}

	// Google requires alternating user/model messages (after system)
	var lastRole string
	systemCount := 0

	for i, msg := range messages {
		if msg.Role == "system" {
			systemCount++
			if systemCount > 1 {
				return fmt.Errorf("only one system message is allowed")
			}
			if i != 0 {
				return fmt.Errorf("system message must be first")
			}
			continue
		}

		if msg.Role != "user" && msg.Role != "assistant" {
			return fmt.Errorf("invalid role: %s (must be user or assistant)", msg.Role)
		}

		// Map assistant to model for internal validation
		currentRole := msg.Role
		if currentRole == "assistant" {
			currentRole = "model"
		}

		if lastRole != "" && lastRole == currentRole {
			return fmt.Errorf("messages must alternate between user and model roles")
		}

		lastRole = currentRole
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
	// Google client doesn't need explicit cleanup
	return nil
}