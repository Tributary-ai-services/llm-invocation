package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tributary-ai/llm-invocation/types"
)

type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) GenerateContent(ctx context.Context, model string, req *GenerateContentRequest) (*GenerateContentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewProviderError("google", types.ErrInvalidRequest, fmt.Sprintf("failed to marshal request: %v", err))
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.baseURL, model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, types.NewProviderError("google", types.ErrNetworkError, fmt.Sprintf("failed to create request: %v", err))
	}

	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, types.NewProviderError("google", types.ErrNetworkError, fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var result GenerateContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, types.NewProviderError("google", types.ErrInvalidResponse, fmt.Sprintf("failed to decode response: %v", err))
	}

	return &result, nil
}

func (c *Client) GenerateContentStream(ctx context.Context, model string, req *GenerateContentRequest) (<-chan *types.StreamChunk, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewProviderError("google", types.ErrInvalidRequest, fmt.Sprintf("failed to marshal request: %v", err))
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent", c.baseURL, model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, types.NewProviderError("google", types.ErrNetworkError, fmt.Sprintf("failed to create request: %v", err))
	}

	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, types.NewProviderError("google", types.ErrNetworkError, fmt.Sprintf("request failed: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, c.handleErrorResponse(resp)
	}

	ch := make(chan *types.StreamChunk)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		decoder := json.NewDecoder(resp.Body)
		responseID := fmt.Sprintf("google-%d", time.Now().Unix())

		for {
			var streamResp StreamingResponse
			if err := decoder.Decode(&streamResp); err != nil {
				if err == io.EOF {
					break
				}
				ch <- &types.StreamChunk{
					Error: types.NewProviderError("google", types.ErrInvalidResponse, fmt.Sprintf("failed to decode stream response: %v", err)),
				}
				return
			}

			chunk := c.processStreamResponse(&streamResp, responseID, model)
			if chunk != nil {
				select {
				case ch <- chunk:
				case <-ctx.Done():
					return
				}

				// Check if this is the final chunk
				if chunk.Done {
					break
				}
			}
		}
	}()

	return ch, nil
}

func (c *Client) processStreamResponse(resp *StreamingResponse, responseID, model string) *types.StreamChunk {
	chunk := &types.StreamChunk{
		ID:       responseID,
		Provider: "google",
		Model:    model,
		Choices:  make([]types.DeltaChoice, 0),
	}

	for i, candidate := range resp.Candidates {
		deltaChoice := types.DeltaChoice{
			Index: i,
			Delta: &types.Delta{},
		}

		// Extract text content from parts
		var textContent string
		var toolCalls []types.ToolCall

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
							ID:   fmt.Sprintf("call_%d_%d", time.Now().UnixNano(), len(toolCalls)),
							Type: "function",
							Function: &types.FunctionCall{
								Name:      name,
								Arguments: string(argsData),
							},
						}
						toolCalls = append(toolCalls, toolCall)
					}
				}
			}
		}

		if textContent != "" {
			deltaChoice.Delta.Content = textContent
		}

		if len(toolCalls) > 0 {
			deltaChoice.Delta.ToolCalls = toolCalls
		}

		// Map finish reason
		if candidate.FinishReason != "" {
			deltaChoice.FinishReason = c.mapFinishReason(candidate.FinishReason)
			chunk.Done = true
		}

		chunk.Choices = append(chunk.Choices, deltaChoice)
	}

	// Add usage metadata if present
	if resp.UsageMetadata.TotalTokenCount > 0 {
		chunk.Usage = &types.Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return chunk
}

func (c *Client) mapFinishReason(reason string) string {
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

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "llm-invocation/1.0")
	
	// Add API key as query parameter (Google's preferred method)
	q := req.URL.Query()
	q.Add("key", c.apiKey)
	req.URL.RawQuery = q.Encode()
}

func (c *Client) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewProviderError("google", types.ErrNetworkError, fmt.Sprintf("failed to read error response: %v", err))
	}

	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		return types.NewProviderError("google", types.ErrInvalidResponse, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	code := types.ErrNetworkError
	switch resp.StatusCode {
	case 400:
		code = types.ErrInvalidRequest
	case 401, 403:
		code = types.ErrAuthenticationFailed
	case 429:
		code = types.ErrRateLimitExceeded
	case 500, 502, 503, 504:
		code = types.ErrNetworkError
	}

	return types.NewProviderError("google", code, errorResp.Error.Message)
}