package openai

import (
	"context"

	"github.com/tributary-ai/llm-invocation/providers"
	"github.com/tributary-ai/llm-invocation/types"
)

type Provider struct {
	*providers.BaseProvider
	client *Client
}

func NewProvider(apiKey, baseURL string) *Provider {
	models := []types.ModelInfo{
		{ID: "gpt-4", Provider: "openai", Name: "GPT-4", Features: []string{"function_calling", "vision"}},
		{ID: "gpt-4-turbo", Provider: "openai", Name: "GPT-4 Turbo", Features: []string{"function_calling", "vision", "json_mode"}},
		{ID: "gpt-4-turbo-preview", Provider: "openai", Name: "GPT-4 Turbo Preview", Features: []string{"function_calling", "vision", "json_mode"}},
		{ID: "gpt-3.5-turbo", Provider: "openai", Name: "GPT-3.5 Turbo", Features: []string{"function_calling"}},
		{ID: "gpt-3.5-turbo-16k", Provider: "openai", Name: "GPT-3.5 Turbo 16K", Features: []string{"function_calling"}},
	}

	features := []providers.Feature{
		providers.FeatureFunctionCalling,
		providers.FeatureVision,
		providers.FeatureJSONMode,
		providers.FeatureStreaming,
	}

	return &Provider{
		BaseProvider: providers.NewBaseProvider("openai", models, features),
		client:       NewClient(apiKey, baseURL),
	}
}

func (p *Provider) Invoke(ctx context.Context, req *types.ProviderRequest) (<-chan *types.StreamChunk, error) {
	if err := p.ValidateRequest(req); err != nil {
		return nil, err
	}

	openaiReq := MapToOpenAIRequest(req)

	if req.Stream {
		return p.client.CreateChatCompletionStream(ctx, openaiReq)
	}

	// For non-streaming, we still return a channel for consistency
	ch := make(chan *types.StreamChunk, 1)

	go func() {
		defer close(ch)

		resp, err := p.client.CreateChatCompletion(ctx, openaiReq)
		if err != nil {
			ch <- &types.StreamChunk{
				Error: err.(*types.Error),
			}
			return
		}

		// Convert sync response to single stream chunk
		invocationResp := MapFromOpenAIResponse(resp)
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

	// OpenAI-specific validations
	if req.Model == "" {
		return types.NewProviderError("openai", types.ErrInvalidRequest, "model is required")
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
		return types.NewProviderError("openai", types.ErrModelNotSupported, "model "+req.Model+" is not supported")
	}

	return nil
}

func (p *Provider) Close() error {
	// OpenAI client doesn't need explicit cleanup
	return nil
}