package types

import "context"

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

type ToolCall struct {
	ID       string                 `json:"id,omitempty"`
	Type     string                 `json:"type"`
	Function *FunctionCall          `json:"function,omitempty"`
	Args     map[string]interface{} `json:"args,omitempty"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolRegistry interface {
	Register(tool Tool) error
	Get(name string) (Tool, error)
	List() []Tool
	Execute(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
}

type ToolDefinition struct {
	Type     string                 `json:"type"`
	Function *FunctionDefinition    `json:"function,omitempty"`
}

type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

func NewFunctionTool(name, description string, parameters map[string]interface{}) *ToolDefinition {
	return &ToolDefinition{
		Type: "function",
		Function: &FunctionDefinition{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}