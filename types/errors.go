package types

import "fmt"

type Error struct {
	Code     string                 `json:"code"`
	Message  string                 `json:"message"`
	Provider string                 `json:"provider,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

func (e *Error) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Provider, e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewError(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

func NewProviderError(provider, code, message string) *Error {
	return &Error{
		Provider: provider,
		Code:     code,
		Message:  message,
		Details:  make(map[string]interface{}),
	}
}

const (
	ErrProviderNotFound     = "provider_not_found"
	ErrModelNotSupported    = "model_not_supported"
	ErrRateLimitExceeded    = "rate_limit_exceeded"
	ErrInvalidRequest       = "invalid_request"
	ErrToolExecutionFailed  = "tool_execution_failed"
	ErrStreamingFailed      = "streaming_failed"
	ErrAuthenticationFailed = "authentication_failed"
	ErrNetworkError        = "network_error"
	ErrInvalidResponse     = "invalid_response"
	ErrTimeout             = "timeout"
)