package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		baseURL = "https://api.openai.com/v1"
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// Ensure streaming is disabled for sync requests
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewProviderError("openai", types.ErrInvalidRequest, fmt.Sprintf("failed to marshal request: %v", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, types.NewProviderError("openai", types.ErrNetworkError, fmt.Sprintf("failed to create request: %v", err))
	}

	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, types.NewProviderError("openai", types.ErrNetworkError, fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, types.NewProviderError("openai", types.ErrInvalidResponse, fmt.Sprintf("failed to decode response: %v", err))
	}

	return &result, nil
}

func (c *Client) CreateChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (<-chan *types.StreamChunk, error) {
	// Ensure streaming is enabled
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewProviderError("openai", types.ErrInvalidRequest, fmt.Sprintf("failed to marshal request: %v", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, types.NewProviderError("openai", types.ErrNetworkError, fmt.Sprintf("failed to create request: %v", err))
	}

	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, types.NewProviderError("openai", types.ErrNetworkError, fmt.Sprintf("request failed: %v", err))
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, c.handleErrorResponse(resp)
	}

	ch := make(chan *types.StreamChunk)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			if line == "" || line == "data: [DONE]" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResp ChatCompletionStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				ch <- &types.StreamChunk{
					Error: types.NewProviderError("openai", types.ErrInvalidResponse, fmt.Sprintf("failed to decode stream response: %v", err)),
				}
				return
			}

			chunk := MapFromOpenAIStreamResponse(&streamResp)
			select {
			case ch <- chunk:
			case <-ctx.Done():
				return
			}

			// If this is the final chunk, break
			if chunk.Done {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- &types.StreamChunk{
				Error: types.NewProviderError("openai", types.ErrStreamingFailed, fmt.Sprintf("stream scanning failed: %v", err)),
			}
		}
	}()

	return ch, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "llm-invocation/1.0")
}

func (c *Client) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewProviderError("openai", types.ErrNetworkError, fmt.Sprintf("failed to read error response: %v", err))
	}

	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		return types.NewProviderError("openai", types.ErrInvalidResponse, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	code := types.ErrNetworkError
	switch resp.StatusCode {
	case 400:
		code = types.ErrInvalidRequest
	case 401:
		code = types.ErrAuthenticationFailed
	case 429:
		code = types.ErrRateLimitExceeded
	case 500, 502, 503, 504:
		code = types.ErrNetworkError
	}

	return types.NewProviderError("openai", code, errorResp.Error.Message)
}