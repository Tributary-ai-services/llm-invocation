package streaming

import (
	"sync"
	"time"

	"github.com/tributary-ai/llm-invocation/types"
)

type StreamAggregator struct {
	chunks []types.StreamChunk
	mu     sync.Mutex
}

func NewStreamAggregator() *StreamAggregator {
	return &StreamAggregator{
		chunks: make([]types.StreamChunk, 0),
	}
}

func (a *StreamAggregator) Add(chunk types.StreamChunk) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.chunks = append(a.chunks, chunk)
}

func (a *StreamAggregator) Aggregate() (*types.InvocationResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if len(a.chunks) == 0 {
		return nil, types.NewError(types.ErrInvalidResponse, "no chunks to aggregate")
	}
	
	firstChunk := a.chunks[0]
	response := &types.InvocationResponse{
		ID:       firstChunk.ID,
		Provider: firstChunk.Provider,
		Model:    firstChunk.Model,
		Created:  time.Now(),
		Choices:  make([]types.Choice, 0),
		Usage:    &types.Usage{},
		Metadata: make(map[string]interface{}),
	}
	
	// Track choices by index
	choiceMap := make(map[int]*types.Choice)
	
	for _, chunk := range a.chunks {
		// Update usage from final chunk
		if chunk.Usage != nil {
			response.Usage = chunk.Usage
		}
		
		// Process delta choices
		for _, deltaChoice := range chunk.Choices {
			choice, exists := choiceMap[deltaChoice.Index]
			if !exists {
				choice = &types.Choice{
					Index: deltaChoice.Index,
					Message: types.Message{
						Role:      "",
						Content:   "",
						ToolCalls: make([]types.ToolCall, 0),
					},
					ToolCalls: make([]types.ToolCall, 0),
				}
				choiceMap[deltaChoice.Index] = choice
			}
			
			if deltaChoice.Delta != nil {
				// Accumulate role
				if deltaChoice.Delta.Role != "" {
					choice.Message.Role = deltaChoice.Delta.Role
				}
				
				// Accumulate content
				if deltaChoice.Delta.Content != "" {
					if content, ok := choice.Message.Content.(string); ok {
						choice.Message.Content = content + deltaChoice.Delta.Content
					} else {
						choice.Message.Content = deltaChoice.Delta.Content
					}
				}
				
				// Accumulate tool calls
				choice.Message.ToolCalls = append(choice.Message.ToolCalls, deltaChoice.Delta.ToolCalls...)
			}
			
			// Set finish reason from final chunk
			if deltaChoice.FinishReason != "" {
				choice.FinishReason = deltaChoice.FinishReason
			}
			
			// Add tool calls
			choice.ToolCalls = append(choice.ToolCalls, deltaChoice.ToolCalls...)
		}
	}
	
	// Convert map to slice, maintaining order
	maxIndex := -1
	for index := range choiceMap {
		if index > maxIndex {
			maxIndex = index
		}
	}
	
	response.Choices = make([]types.Choice, maxIndex+1)
	for index, choice := range choiceMap {
		response.Choices[index] = *choice
	}
	
	return response, nil
}

func (a *StreamAggregator) GetContent() string {
	response, err := a.Aggregate()
	if err != nil {
		return ""
	}
	
	if len(response.Choices) > 0 {
		if content, ok := response.Choices[0].Message.Content.(string); ok {
			return content
		}
	}
	
	return ""
}

func (a *StreamAggregator) GetToolCalls() []types.ToolCall {
	response, err := a.Aggregate()
	if err != nil {
		return nil
	}
	
	if len(response.Choices) > 0 {
		return response.Choices[0].Message.ToolCalls
	}
	
	return nil
}