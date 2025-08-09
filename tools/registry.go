package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/tributary-ai/llm-invocation/types"
)

type Registry interface {
	Register(tool types.Tool) error
	Get(name string) (types.Tool, error)
	List() []types.Tool
	Execute(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
}

type registry struct {
	tools map[string]types.Tool
	mu    sync.RWMutex
}

func NewRegistry() Registry {
	return &registry{
		tools: make(map[string]types.Tool),
	}
}

func (r *registry) Register(tool types.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if name == "" {
		return types.NewError(types.ErrInvalidRequest, "tool name cannot be empty")
	}

	if _, exists := r.tools[name]; exists {
		return types.NewError(types.ErrInvalidRequest, fmt.Sprintf("tool %s already registered", name))
	}

	r.tools[name] = tool
	return nil
}

func (r *registry) Get(name string) (types.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, types.NewError("tool_not_found", fmt.Sprintf("tool %s not found", name))
	}

	return tool, nil
}

func (r *registry) List() []types.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]types.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

func (r *registry) Execute(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	// Validate arguments against tool parameters
	if err := r.validateArgs(tool, args); err != nil {
		return nil, types.NewError(types.ErrInvalidRequest, fmt.Sprintf("invalid arguments for tool %s: %v", name, err))
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		return nil, types.NewError(types.ErrToolExecutionFailed, fmt.Sprintf("tool %s execution failed: %v", name, err))
	}

	return result, nil
}

func (r *registry) validateArgs(tool types.Tool, args map[string]interface{}) error {
	parameters := tool.Parameters()
	if parameters == nil {
		return nil
	}

	// Basic validation - check required parameters
	if _, ok := parameters["properties"].(map[string]interface{}); ok {
		if required, ok := parameters["required"].([]interface{}); ok {
			for _, req := range required {
				if reqStr, ok := req.(string); ok {
					if _, exists := args[reqStr]; !exists {
						return fmt.Errorf("required parameter %s missing", reqStr)
					}
				}
			}
		}
	}

	return nil
}