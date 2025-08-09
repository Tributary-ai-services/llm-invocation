package anthropic

type MessageRequest struct {
	Model         string    `json:"model"`
	MaxTokens     int       `json:"max_tokens"`
	Messages      []Message `json:"messages"`
	System        string    `json:"system,omitempty"`
	Temperature   *float32  `json:"temperature,omitempty"`
	TopP          *float32  `json:"top_p,omitempty"`
	TopK          *int      `json:"top_k,omitempty"`
	Stream        bool      `json:"stream,omitempty"`
	StopSequences []string  `json:"stop_sequences,omitempty"`
	Tools         []Tool    `json:"tools,omitempty"`
	ToolChoice    *ToolChoice `json:"tool_choice,omitempty"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	
	// For tool use
	ID        string      `json:"id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Input     interface{} `json:"input,omitempty"`
	
	// For tool results
	ToolUseID string      `json:"tool_use_id,omitempty"`
	Content   string      `json:"content,omitempty"`
	IsError   bool        `json:"is_error,omitempty"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type ToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type MessageResponse struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Role         string  `json:"role"`
	Content      []Content `json:"content"`
	Model        string  `json:"model"`
	StopReason   string  `json:"stop_reason,omitempty"`
	StopSequence string  `json:"stop_sequence,omitempty"`
	Usage        Usage   `json:"usage"`
}

type StreamEvent struct {
	Type string `json:"type"`
	
	// message_start
	Message *MessageResponse `json:"message,omitempty"`
	
	// content_block_start
	Index       *int     `json:"index,omitempty"`
	ContentBlock *Content `json:"content_block,omitempty"`
	
	// content_block_delta
	Delta *ContentDelta `json:"delta,omitempty"`
	
	// content_block_stop, message_delta, message_stop
	Usage *Usage `json:"usage,omitempty"`
}

type ContentDelta struct {
	Type         string      `json:"type"`
	Text         string      `json:"text,omitempty"`
	PartialJSON  string      `json:"partial_json,omitempty"`
	Input        interface{} `json:"input,omitempty"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ErrorResponse struct {
	Type  string      `json:"type"`
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}