package types

type InvocationRequest struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model"`

	Messages []Message `json:"messages"`

	Options ModelOptions `json:"options,omitempty"`

	Tools      []Tool      `json:"tools,omitempty"`
	ToolChoice interface{} `json:"tool_choice,omitempty"`

	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	Stream *bool `json:"stream,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ModelOptions struct {
	Temperature      *float32               `json:"temperature,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	TopP             *float32               `json:"top_p,omitempty"`
	FrequencyPenalty *float32               `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float32               `json:"presence_penalty,omitempty"`
	Stop             []string               `json:"stop,omitempty"`
	Seed             *int                   `json:"seed,omitempty"`
	ProviderOptions  map[string]interface{} `json:"provider_options,omitempty"`
}

type Message struct {
	Role       string         `json:"role"`
	Content    MessageContent `json:"content"`
	Name       string         `json:"name,omitempty"`
	ToolCalls  []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type MessageContent interface{}

type ResponseFormat struct {
	Type   string      `json:"type"`
	Schema interface{} `json:"schema,omitempty"`
}

type ProviderRequest struct {
	Model    string                 `json:"model"`
	Messages []Message              `json:"messages"`
	Options  map[string]interface{} `json:"options,omitempty"`
	Tools    []Tool                 `json:"tools,omitempty"`
	Stream   bool                   `json:"stream"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req *InvocationRequest) Validate() error {
	if req.Model == "" {
		return NewError(ErrInvalidRequest, "model is required")
	}
	if len(req.Messages) == 0 {
		return NewError(ErrInvalidRequest, "at least one message is required")
	}
	return nil
}

func (req *InvocationRequest) SetDefaults() {
	if req.Stream == nil {
		stream := true
		req.Stream = &stream
	}
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}
}