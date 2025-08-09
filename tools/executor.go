package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tributary-ai/llm-invocation/types"
)

type Executor struct {
	registry Registry
	timeout  time.Duration
	sandbox  bool
}

type ExecutionResult struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func NewExecutor(registry Registry, timeout time.Duration, sandbox bool) *Executor {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Executor{
		registry: registry,
		timeout:  timeout,
		sandbox:  sandbox,
	}
}

func (e *Executor) Execute(ctx context.Context, toolCall types.ToolCall) (*ExecutionResult, error) {
	if toolCall.Function == nil {
		return nil, types.NewError(types.ErrInvalidRequest, "tool call must have function information")
	}

	// Parse arguments from JSON string
	var args map[string]interface{}
	if toolCall.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return &ExecutionResult{
				Error: fmt.Sprintf("failed to parse tool arguments: %v", err),
			}, nil
		}
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Execute the tool
	result, err := e.executeWithSandbox(execCtx, toolCall.Function.Name, args)
	if err != nil {
		return &ExecutionResult{
			Error: err.Error(),
		}, nil
	}

	return &ExecutionResult{
		Result: result,
	}, nil
}

func (e *Executor) executeWithSandbox(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	if e.sandbox {
		// In a real implementation, this would run in a sandboxed environment
		// For now, we'll just add some basic protections
		return e.executeSafely(ctx, name, args)
	}

	return e.registry.Execute(ctx, name, args)
}

func (e *Executor) executeSafely(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	// Create a channel to capture the result
	resultChan := make(chan interface{}, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorChan <- types.NewError(types.ErrToolExecutionFailed, fmt.Sprintf("tool %s panicked: %v", name, r))
			}
		}()

		result, err := e.registry.Execute(ctx, name, args)
		if err != nil {
			errorChan <- err
			return
		}

		resultChan <- result
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, types.NewError(types.ErrTimeout, fmt.Sprintf("tool %s execution timed out", name))
	}
}

func (e *Executor) ExecuteMultiple(ctx context.Context, toolCalls []types.ToolCall) ([]*ExecutionResult, error) {
	results := make([]*ExecutionResult, len(toolCalls))

	// Execute tools in parallel if supported
	for i, toolCall := range toolCalls {
		result, err := e.Execute(ctx, toolCall)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}

	return results, nil
}