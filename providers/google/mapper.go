package google

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tributary-ai/llm-invocation/types"
)

func MapToGoogleRequest(req *types.ProviderRequest) *GenerateContentRequest {
	googleReq := &GenerateContentRequest{
		Contents:       make([]Content, 0),
		GenerationConfig: &GenerationConfig{},
	}

	// Extract system instruction if present
	var systemInstruction *Content
	var nonSystemMessages []types.Message

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if content, ok := msg.Content.(string); ok {
				systemInstruction = &Content{
					Parts: []Part{
						map[string]interface{}{"text": content},
					},
				}
			}
		} else {
			nonSystemMessages = append(nonSystemMessages, msg)
		}
	}

	if systemInstruction != nil {
		googleReq.SystemInstruction = systemInstruction
	}

	// Map messages (excluding system messages)
	for _, msg := range nonSystemMessages {
		googleContent := Content{
			Role:  mapRole(msg.Role),
			Parts: make([]Part, 0),
		}

		// Handle different content types
		switch content := msg.Content.(type) {
		case string:
			if content != "" {
				googleContent.Parts = append(googleContent.Parts, 
					map[string]interface{}{"text": content})
			}
		case []interface{}:
			// Handle multimodal content
			for _, item := range content {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if contentType, ok := itemMap["type"].(string); ok {
						switch contentType {
						case "text":
							if text, ok := itemMap["text"].(string); ok {
								googleContent.Parts = append(googleContent.Parts, 
									map[string]interface{}{"text": text})
							}
						case "image_url":
							// Handle image content for vision models
							if imageUrl, ok := itemMap["image_url"].(map[string]interface{}); ok {
								if url, ok := imageUrl["url"].(string); ok {
									// Note: This would need proper image data extraction
									googleContent.Parts = append(googleContent.Parts,
										map[string]interface{}{
											"inlineData": map[string]interface{}{
												"mimeType": "image/jpeg",
												"data":     url, // This would need base64 conversion
											},
										})
								}
							}
						}
					}
				}
			}
		}

		// Handle tool calls
		for _, toolCall := range msg.ToolCalls {
			if toolCall.Function != nil {
				var args map[string]interface{}
				if toolCall.Function.Arguments != "" {
					json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				}

				googleContent.Parts = append(googleContent.Parts,
					map[string]interface{}{
						"functionCall": map[string]interface{}{
							"name": toolCall.Function.Name,
							"args": args,
						},
					})
			}
		}

		// Handle tool results (function responses)
		if msg.ToolCallID != "" {
			if content, ok := msg.Content.(string); ok {
				var response map[string]interface{}
				// Try to parse as JSON, otherwise use as plain text
				if err := json.Unmarshal([]byte(content), &response); err != nil {
					response = map[string]interface{}{"result": content}
				}

				googleContent.Parts = append(googleContent.Parts,
					map[string]interface{}{
						"functionResponse": map[string]interface{}{
							"name":     msg.Name, // Tool name
							"response": response,
						},
					})
			}
		}

		// Only add messages that have parts
		if len(googleContent.Parts) > 0 {
			googleReq.Contents = append(googleReq.Contents, googleContent)
		}
	}

	// Map tools
	if len(req.Tools) > 0 {
		tool := Tool{
			FunctionDeclarations: make([]FunctionDeclaration, len(req.Tools)),
		}
		
		for i, reqTool := range req.Tools {
			tool.FunctionDeclarations[i] = FunctionDeclaration{
				Name:        reqTool.Name(),
				Description: reqTool.Description(),
				Parameters:  reqTool.Parameters(),
			}
		}
		
		googleReq.Tools = []Tool{tool}
		
		// Configure function calling
		googleReq.ToolConfig = &ToolConfig{
			FunctionCallingConfig: FunctionCallingConfig{
				Mode: "AUTO", // Can be AUTO, ANY, or NONE
			},
		}
	}

	// Map generation config options
	if req.Options != nil {
		if temp, ok := req.Options["temperature"].(float32); ok {
			googleReq.GenerationConfig.Temperature = &temp
		}
		if maxTokens, ok := req.Options["max_tokens"].(int); ok {
			googleReq.GenerationConfig.MaxOutputTokens = &maxTokens
		}
		if topP, ok := req.Options["top_p"].(float32); ok {
			googleReq.GenerationConfig.TopP = &topP
		}
		if topK, ok := req.Options["top_k"].(int); ok {
			googleReq.GenerationConfig.TopK = &topK
		}
		if stop, ok := req.Options["stop"].([]string); ok {
			googleReq.GenerationConfig.StopSequences = stop
		}
	}

	// Add default safety settings (permissive for general use)
	googleReq.SafetySettings = []SafetySetting{
		{Category: SafetyCategoryHarassment, Threshold: SafetyThresholdBlockOnlyHigh},
		{Category: SafetyCategoryHateSpeech, Threshold: SafetyThresholdBlockOnlyHigh},
		{Category: SafetyCategorySexuallyExplicit, Threshold: SafetyThresholdBlockOnlyHigh},
		{Category: SafetyCategoryDangerousContent, Threshold: SafetyThresholdBlockOnlyHigh},
	}

	return googleReq
}

func MapFromGoogleResponse(resp *GenerateContentResponse, model string) *types.InvocationResponse {
	response := &types.InvocationResponse{
		ID:       fmt.Sprintf("google-%d", time.Now().Unix()),
		Provider: "google",
		Model:    model,
		Created:  time.Now(),
		Choices:  make([]types.Choice, len(resp.Candidates)),
		Metadata: make(map[string]interface{}),
	}

	if resp.UsageMetadata.TotalTokenCount > 0 {
		response.Usage = &types.Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	// Map candidates to choices
	for i, candidate := range resp.Candidates {
		choice := types.Choice{
			Index: i,
			Message: types.Message{
				Role:      "assistant",
				Content:   "",
				ToolCalls: make([]types.ToolCall, 0),
			},
			FinishReason: mapFinishReason(candidate.FinishReason),
		}

		// Extract content from parts
		var textContent string
		for _, part := range candidate.Content.Parts {
			switch p := part.(type) {
			case map[string]interface{}:
				if text, ok := p["text"].(string); ok {
					textContent += text
				}
				if functionCall, ok := p["functionCall"].(map[string]interface{}); ok {
					if name, nameOk := functionCall["name"].(string); nameOk {
						argsData, _ := json.Marshal(functionCall["args"])
						toolCall := types.ToolCall{
							ID:   fmt.Sprintf("call_%d_%d", time.Now().UnixNano(), len(choice.Message.ToolCalls)),
							Type: "function",
							Function: &types.FunctionCall{
								Name:      name,
								Arguments: string(argsData),
							},
						}
						choice.Message.ToolCalls = append(choice.Message.ToolCalls, toolCall)
					}
				}
			}
		}

		choice.Message.Content = textContent
		response.Choices[i] = choice
	}

	return response
}

func mapRole(role string) string {
	switch role {
	case "user":
		return "user"
	case "assistant":
		return "model"
	case "system":
		return "user" // Google doesn't have system role, handled via systemInstruction
	default:
		return "user"
	}
}

func mapFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	case "OTHER":
		return "stop"
	default:
		return reason
	}
}