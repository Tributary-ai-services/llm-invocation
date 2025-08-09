package tools

import (
	"context"
	"testing"
)

type mockTool struct {
	name        string
	description string
	params      map[string]interface{}
	result      interface{}
	err         error
}

func (m *mockTool) Name() string { return m.name }
func (m *mockTool) Description() string { return m.description }
func (m *mockTool) Parameters() map[string]interface{} { return m.params }
func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return m.result, m.err
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	tool := &mockTool{name: "test-tool"}

	err := registry.Register(tool)
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Test duplicate registration
	err = registry.Register(tool)
	if err == nil {
		t.Error("Register() should fail for duplicate tools")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	tool := &mockTool{name: "test-tool"}
	registry.Register(tool)

	retrieved, err := registry.Get("test-tool")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.Name() != tool.Name() {
		t.Error("Get() returned wrong tool")
	}

	// Test non-existent tool
	_, err = registry.Get("non-existent")
	if err == nil {
		t.Error("Get() should fail for non-existent tools")
	}
}

func TestRegistry_Execute(t *testing.T) {
	registry := NewRegistry()
	expectedResult := "test result"
	tool := &mockTool{
		name:   "test-tool",
		result: expectedResult,
	}
	registry.Register(tool)

	result, err := registry.Execute(context.Background(), "test-tool", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if result != expectedResult {
		t.Errorf("Execute() returned %v, expected %v", result, expectedResult)
	}
}