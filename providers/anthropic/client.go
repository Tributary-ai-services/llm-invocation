package anthropic

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
		baseURL = "https://api.anthropic.com"
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) CreateMessage(ctx context.Context, req *MessageRequest) (*MessageResponse, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewProviderError("anthropic", types.ErrInvalidRequest, fmt.Sprintf("failed to marshal request: %v", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, types.NewProviderError("anthropic", types.ErrNetworkError, fmt.Sprintf("failed to create request: %v", err))
	}

	c.setHeaders(httpReq)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, types.NewProviderError("anthropic", types.ErrNetworkError, fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var result MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, types.NewProviderError("anthropic", types.ErrInvalidResponse, fmt.Sprintf("failed to decode response: %v", err))
	}

	return &result, nil
}

func (c *Client) CreateMessageStream(ctx context.Context, req *MessageRequest) (<-chan *types.StreamChunk, error) {
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, types.NewProviderError("anthropic", types.ErrInvalidRequest, fmt.Sprintf("failed to marshal request: %v", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, types.NewProviderError("anthropic", types.ErrNetworkError, fmt.Sprintf("failed to create request: %v", err))
	}

	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, types.NewProviderError("anthropic", types.ErrNetworkError, fmt.Sprintf("request failed: %v", err))
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
		var currentMessage *MessageResponse
		var currentContent strings.Builder

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var event StreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				ch <- &types.StreamChunk{
					Error: types.NewProviderError("anthropic", types.ErrInvalidResponse, fmt.Sprintf("failed to decode stream event: %v", err)),
				}
				return
			}

			chunk := c.processStreamEvent(&event, &currentMessage, &currentContent)
			if chunk != nil {
				select {
				case ch <- chunk:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- &types.StreamChunk{
				Error: types.NewProviderError("anthropic", types.ErrStreamingFailed, fmt.Sprintf("stream scanning failed: %v", err)),
			}
		}
	}()

	return ch, nil
}

func (c *Client) processStreamEvent(event *StreamEvent, currentMessage **MessageResponse, currentContent *strings.Builder) *types.StreamChunk {
	switch event.Type {
	case "message_start":
		*currentMessage = event.Message
		return &types.StreamChunk{
			ID:       event.Message.ID,
			Provider: "anthropic",
			Model:    event.Message.Model,
		}

	case "content_block_delta":
		if event.Delta != nil && event.Delta.Text != "" {
			currentContent.WriteString(event.Delta.Text)
			
			return &types.StreamChunk{
				ID:       (*currentMessage).ID,
				Provider: "anthropic",
				Model:    (*currentMessage).Model,
				Choices: []types.DeltaChoice{
					{
						Index: 0,
						Delta: &types.Delta{
							Content: event.Delta.Text,
						},
					},
				},
			}
		}

	case "message_stop":
		finalChunk := &types.StreamChunk{
			ID:       (*currentMessage).ID,
			Provider: "anthropic",
			Model:    (*currentMessage).Model,
			Done:     true,
			Choices: []types.DeltaChoice{
				{
					Index:        0,
					FinishReason: "stop",
				},
			},
		}

		if event.Usage != nil {
			finalChunk.Usage = &types.Usage{
				PromptTokens:     event.Usage.InputTokens,
				CompletionTokens: event.Usage.OutputTokens,
				TotalTokens:      event.Usage.InputTokens + event.Usage.OutputTokens,
			}
		}

		return finalChunk
	}

	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("User-Agent", "llm-invocation/1.0")
}

func (c *Client) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewProviderError("anthropic", types.ErrNetworkError, fmt.Sprintf("failed to read error response: %v", err))
	}

	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		return types.NewProviderError("anthropic", types.ErrInvalidResponse, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
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

	return types.NewProviderError("anthropic", code, errorResp.Error.Message)
}