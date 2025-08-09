package google

type GenerateContentRequest struct {
	Contents         []Content         `json:"contents"`
	Tools            []Tool            `json:"tools,omitempty"`
	ToolConfig       *ToolConfig       `json:"toolConfig,omitempty"`
	SafetySettings   []SafetySetting   `json:"safetySettings,omitempty"`
	SystemInstruction *Content         `json:"systemInstruction,omitempty"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
}

type Content struct {
	Parts []Part `json:"parts"`
	Role  string `json:"role,omitempty"`
}

type Part interface{}

type TextPart struct {
	Text string `json:"text"`
}

type InlineDataPart struct {
	InlineData InlineData `json:"inlineData"`
}

type FunctionCallPart struct {
	FunctionCall FunctionCall `json:"functionCall"`
}

type FunctionResponsePart struct {
	FunctionResponse FunctionResponse `json:"functionResponse"`
}

type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type FunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

type FunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type FunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type ToolConfig struct {
	FunctionCallingConfig FunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

type FunctionCallingConfig struct {
	Mode                 string   `json:"mode,omitempty"`
	AllowedFunctionNames []string `json:"allowedFunctionNames,omitempty"`
}

type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type GenerationConfig struct {
	StopSequences   []string `json:"stopSequences,omitempty"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
	ResponseSchema   interface{} `json:"responseSchema,omitempty"`
	CandidateCount  *int     `json:"candidateCount,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	Temperature     *float32 `json:"temperature,omitempty"`
	TopP            *float32 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
}

type GenerateContentResponse struct {
	Candidates     []Candidate    `json:"candidates,omitempty"`
	PromptFeedback PromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  UsageMetadata  `json:"usageMetadata,omitempty"`
}

type Candidate struct {
	Content       Content        `json:"content,omitempty"`
	FinishReason  string         `json:"finishReason,omitempty"`
	Index         int            `json:"index,omitempty"`
	SafetyRatings []SafetyRating `json:"safetyRatings,omitempty"`
}

type PromptFeedback struct {
	BlockReason    string         `json:"blockReason,omitempty"`
	SafetyRatings  []SafetyRating `json:"safetyRatings,omitempty"`
}

type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
}

// Streaming types
type StreamingResponse struct {
	Candidates     []Candidate    `json:"candidates,omitempty"`
	PromptFeedback PromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  UsageMetadata  `json:"usageMetadata,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Safety categories
const (
	SafetyCategoryHarassment       = "HARM_CATEGORY_HARASSMENT"
	SafetyCategoryHateSpeech      = "HARM_CATEGORY_HATE_SPEECH"
	SafetyCategorySexuallyExplicit = "HARM_CATEGORY_SEXUALLY_EXPLICIT"
	SafetyCategoryDangerousContent = "HARM_CATEGORY_DANGEROUS_CONTENT"
)

// Safety thresholds
const (
	SafetyThresholdBlockNone         = "BLOCK_NONE"
	SafetyThresholdBlockOnlyHigh     = "BLOCK_ONLY_HIGH"
	SafetyThresholdBlockMediumAndHigh = "BLOCK_MEDIUM_AND_ABOVE"
	SafetyThresholdBlockLowAndAbove  = "BLOCK_LOW_AND_ABOVE"
)