package openai

import (
	"time"

	"github.com/tributary-ai/llm-invocation/types"
)

func MapToOpenAIRequest(req *types.ProviderRequest) *ChatCompletionRequest {
	openaiReq := &ChatCompletionRequest{
		Model:    req.Model,
		Messages: make([]ChatMessage, len(req.Messages)),
		Stream:   req.Stream,
	}

	// Map messages
	for i, msg := range req.Messages {
		openaiReq.Messages[i] = ChatMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCallID: msg.ToolCallID,
		}

		// Map tool calls
		if len(msg.ToolCalls) > 0 {
			openaiReq.Messages[i].ToolCalls = make([]ToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				openaiReq.Messages[i].ToolCalls[j] = ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: FuncCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	// Map tools
	if len(req.Tools) > 0 {
		openaiReq.Tools = make([]Tool, len(req.Tools))
		for i, tool := range req.Tools {
			openaiReq.Tools[i] = Tool{
				Type: "function",
				Function: Function{
					Name:        tool.Name(),
					Description: tool.Description(),
					Parameters:  tool.Parameters(),
				},
			}
		}
	}

	// Map options from the options map
	if req.Options != nil {
		if temp, ok := req.Options["temperature"].(float32); ok {
			openaiReq.Temperature = &temp
		}
		if maxTokens, ok := req.Options["max_tokens"].(int); ok {
			openaiReq.MaxTokens = &maxTokens
		}
		if topP, ok := req.Options["top_p"].(float32); ok {
			openaiReq.TopP = &topP
		}
		if freqPenalty, ok := req.Options["frequency_penalty"].(float32); ok {
			openaiReq.FrequencyPenalty = &freqPenalty
		}
		if presPenalty, ok := req.Options["presence_penalty"].(float32); ok {
			openaiReq.PresencePenalty = &presPenalty
		}
		if stop, ok := req.Options["stop"].([]string); ok {
			openaiReq.Stop = stop
		}
		if seed, ok := req.Options["seed"].(int); ok {
			openaiReq.Seed = &seed
		}
	}

	return openaiReq
}

func MapFromOpenAIResponse(resp *ChatCompletionResponse) *types.InvocationResponse {
	return &types.InvocationResponse{
		ID:       resp.ID,
		Provider: "openai",
		Model:    resp.Model,
		Created:  time.Unix(resp.Created, 0),
		Choices:  mapChoices(resp.Choices),
		Usage: &types.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Metadata: make(map[string]interface{}),
	}
}

func MapFromOpenAIStreamResponse(resp *ChatCompletionStreamResponse) *types.StreamChunk {
	chunk := &types.StreamChunk{
		ID:       resp.ID,
		Provider: "openai",
		Model:    resp.Model,
		Choices:  make([]types.DeltaChoice, len(resp.Choices)),
	}

	for i, choice := range resp.Choices {
		chunk.Choices[i] = types.DeltaChoice{
			Index: choice.Index,
			Delta: &types.Delta{
				Role:    choice.Delta.Role,
				Content: choice.Delta.Content,
			},
			FinishReason: choice.FinishReason,
		}

		// Map tool calls
		if len(choice.Delta.ToolCalls) > 0 {
			chunk.Choices[i].Delta.ToolCalls = make([]types.ToolCall, len(choice.Delta.ToolCalls))
			for j, tc := range choice.Delta.ToolCalls {
				chunk.Choices[i].Delta.ToolCalls[j] = types.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: &types.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}

	// Map usage if present
	if resp.Usage != nil {
		chunk.Usage = &types.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	// Check if this is the final chunk
	for _, choice := range resp.Choices {
		if choice.FinishReason != "" {
			chunk.Done = true
			break
		}
	}

	return chunk
}

func mapChoices(choices []Choice) []types.Choice {
	mapped := make([]types.Choice, len(choices))
	
	for i, choice := range choices {
		mapped[i] = types.Choice{
			Index: choice.Index,
			Message: types.Message{
				Role:    choice.Message.Role,
				Content: choice.Message.Content,
				Name:    choice.Message.Name,
			},
			FinishReason: choice.FinishReason,
		}

		// Map tool calls
		if len(choice.Message.ToolCalls) > 0 {
			mapped[i].Message.ToolCalls = make([]types.ToolCall, len(choice.Message.ToolCalls))
			for j, tc := range choice.Message.ToolCalls {
				mapped[i].Message.ToolCalls[j] = types.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: &types.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
	}
	
	return mapped
}