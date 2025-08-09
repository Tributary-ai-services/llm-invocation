package anthropic

import (
	"encoding/json"
	"time"

	"github.com/tributary-ai/llm-invocation/types"
)

func MapToAnthropicRequest(req *types.ProviderRequest) *MessageRequest {
	anthropicReq := &MessageRequest{
		Model:     req.Model,
		MaxTokens: 4096, // Default, will be overridden if specified
		Messages:  make([]Message, 0),
		Stream:    req.Stream,
	}

	// Extract system message if present
	var systemMessage string
	var nonSystemMessages []types.Message

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if content, ok := msg.Content.(string); ok {
				systemMessage = content
			}
		} else {
			nonSystemMessages = append(nonSystemMessages, msg)
		}
	}

	if systemMessage != "" {
		anthropicReq.System = systemMessage
	}

	// Map messages (excluding system messages)
	for _, msg := range nonSystemMessages {
		anthropicMsg := Message{
			Role:    msg.Role,
			Content: make([]Content, 0),
		}

		// Handle different content types
		switch content := msg.Content.(type) {
		case string:
			if content != "" {
				anthropicMsg.Content = append(anthropicMsg.Content, Content{
					Type: "text",
					Text: content,
				})
			}
		case []interface{}:
			// Handle multimodal content
			for _, item := range content {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if contentType, ok := itemMap["type"].(string); ok {
						switch contentType {
						case "text":
							if text, ok := itemMap["text"].(string); ok {
								anthropicMsg.Content = append(anthropicMsg.Content, Content{
									Type: "text",
									Text: text,
								})
							}
						}
					}
				}
			}
		}

		// Handle tool calls
		for _, toolCall := range msg.ToolCalls {
			if toolCall.Function != nil {
				var input interface{}
				if toolCall.Function.Arguments != "" {
					json.Unmarshal([]byte(toolCall.Function.Arguments), &input)
				}

				anthropicMsg.Content = append(anthropicMsg.Content, Content{
					Type:  "tool_use",
					ID:    toolCall.ID,
					Name:  toolCall.Function.Name,
					Input: input,
				})
			}
		}

		// Handle tool results (for assistant messages with tool call results)
		if msg.ToolCallID != "" {
			if content, ok := msg.Content.(string); ok {
				anthropicMsg.Content = append(anthropicMsg.Content, Content{
					Type:      "tool_result",
					ToolUseID: msg.ToolCallID,
					Content:   content,
				})
			}
		}

		// Only add messages that have content
		if len(anthropicMsg.Content) > 0 {
			anthropicReq.Messages = append(anthropicReq.Messages, anthropicMsg)
		}
	}

	// Map tools
	if len(req.Tools) > 0 {
		anthropicReq.Tools = make([]Tool, len(req.Tools))
		for i, tool := range req.Tools {
			anthropicReq.Tools[i] = Tool{
				Name:        tool.Name(),
				Description: tool.Description(),
				InputSchema: tool.Parameters(),
			}
		}
	}

	// Map options from the options map
	if req.Options != nil {
		if temp, ok := req.Options["temperature"].(float32); ok {
			anthropicReq.Temperature = &temp
		}
		if maxTokens, ok := req.Options["max_tokens"].(int); ok {
			anthropicReq.MaxTokens = maxTokens
		}
		if topP, ok := req.Options["top_p"].(float32); ok {
			anthropicReq.TopP = &topP
		}
		if topK, ok := req.Options["top_k"].(int); ok {
			anthropicReq.TopK = &topK
		}
		if stop, ok := req.Options["stop"].([]string); ok {
			anthropicReq.StopSequences = stop
		}
	}

	return anthropicReq
}

func MapFromAnthropicResponse(resp *MessageResponse) *types.InvocationResponse {
	response := &types.InvocationResponse{
		ID:       resp.ID,
		Provider: "anthropic",
		Model:    resp.Model,
		Created:  time.Now(),
		Choices:  make([]types.Choice, 1),
		Usage: &types.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		Metadata: make(map[string]interface{}),
	}

	// Map the single choice (Anthropic always returns one choice)
	choice := types.Choice{
		Index: 0,
		Message: types.Message{
			Role:      resp.Role,
			Content:   "",
			ToolCalls: make([]types.ToolCall, 0),
		},
		FinishReason: mapFinishReason(resp.StopReason),
	}

	// Combine all text content
	var textContent string
	for _, content := range resp.Content {
		switch content.Type {
		case "text":
			textContent += content.Text
		case "tool_use":
			// Map tool use to tool calls
			var args string
			if content.Input != nil {
				if argsBytes, err := json.Marshal(content.Input); err == nil {
					args = string(argsBytes)
				}
			}

			choice.Message.ToolCalls = append(choice.Message.ToolCalls, types.ToolCall{
				ID:   content.ID,
				Type: "function",
				Function: &types.FunctionCall{
					Name:      content.Name,
					Arguments: args,
				},
			})
		}
	}

	choice.Message.Content = textContent
	response.Choices[0] = choice

	return response
}

func mapFinishReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	case "stop_sequence":
		return "stop"
	default:
		return reason
	}
}