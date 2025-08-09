package llminvocation

import (
	"context"
	"io"

	"github.com/tributary-ai/llm-invocation/types"
)

type Client interface {
	Invoke(ctx context.Context, req *types.InvocationRequest) (ResponseStream, error)
	InvokeSync(ctx context.Context, req *types.InvocationRequest) (*types.InvocationResponse, error)
	
	// Enhanced methods with user credentials support
	InvokeWithCredentials(ctx context.Context, req *types.EnhancedInvocationRequest, userSecret string) (ResponseStream, error)
	InvokeSyncWithCredentials(ctx context.Context, req *types.EnhancedInvocationRequest, userSecret string) (*types.InvocationResponse, error)
	
	ListModels(ctx context.Context) ([]types.ModelInfo, error)
	RegisterTool(tool types.Tool) error
	RegisterMCPServer(server MCPServer) error
	Close() error
}

type ResponseStream interface {
	Next() (*types.StreamChunk, error)
	Close() error
	Aggregate() (*types.InvocationResponse, error)
}

type MCPServer interface {
	Connect(ctx context.Context) error
	ListTools(ctx context.Context) ([]types.Tool, error)
	ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
	Disconnect() error
}

var _ io.Closer = (*client)(nil)