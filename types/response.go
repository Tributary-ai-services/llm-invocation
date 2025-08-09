package types

import "time"

type InvocationResponse struct {
	ID       string                 `json:"id"`
	Provider string                 `json:"provider"`
	Model    string                 `json:"model"`
	Created  time.Time              `json:"created"`
	Choices  []Choice               `json:"choices"`
	Usage    *Usage                 `json:"usage,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type StreamChunk struct {
	ID       string                 `json:"id"`
	Provider string                 `json:"provider"`
	Model    string                 `json:"model"`
	Delta    *Delta                 `json:"delta,omitempty"`
	Choices  []DeltaChoice          `json:"choices,omitempty"`
	Usage    *Usage                 `json:"usage,omitempty"`
	Error    *Error                 `json:"error,omitempty"`
	Done     bool                   `json:"done,omitempty"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      Message     `json:"message"`
	FinishReason string      `json:"finish_reason,omitempty"`
	ToolCalls    []ToolCall  `json:"tool_calls,omitempty"`
}

type DeltaChoice struct {
	Index        int         `json:"index"`
	Delta        *Delta      `json:"delta,omitempty"`
	FinishReason string      `json:"finish_reason,omitempty"`
	ToolCalls    []ToolCall  `json:"tool_calls,omitempty"`
}

type Delta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ModelInfo struct {
	ID           string   `json:"id"`
	Provider     string   `json:"provider"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Features     []string `json:"features,omitempty"`
	ContextLimit int      `json:"context_limit,omitempty"`
}

func (r *InvocationResponse) GetFirstChoice() *Choice {
	if len(r.Choices) > 0 {
		return &r.Choices[0]
	}
	return nil
}

func (r *InvocationResponse) GetContent() string {
	choice := r.GetFirstChoice()
	if choice != nil {
		if content, ok := choice.Message.Content.(string); ok {
			return content
		}
	}
	return ""
}