package providers

import (
	"context"

	"github.com/tributary-ai/llm-invocation/types"
)

type Provider interface {
	Name() string
	Models() []types.ModelInfo
	Invoke(ctx context.Context, req *types.ProviderRequest) (<-chan *types.StreamChunk, error)
	SupportsFeature(feature Feature) bool
	ValidateRequest(req *types.ProviderRequest) error
	Close() error
}

type Feature string

const (
	FeatureFunctionCalling  Feature = "function_calling"
	FeatureParallelTools    Feature = "parallel_tools"
	FeatureVision          Feature = "vision"
	FeatureJSONMode        Feature = "json_mode"
	FeatureStructuredOutput Feature = "structured_output"
	FeatureMCP             Feature = "mcp"
	FeatureStreaming       Feature = "streaming"
)

type BaseProvider struct {
	name     string
	models   []types.ModelInfo
	features map[Feature]bool
}

func NewBaseProvider(name string, models []types.ModelInfo, features []Feature) *BaseProvider {
	featureMap := make(map[Feature]bool)
	for _, feature := range features {
		featureMap[feature] = true
	}
	
	return &BaseProvider{
		name:     name,
		models:   models,
		features: featureMap,
	}
}

func (p *BaseProvider) Name() string {
	return p.name
}

func (p *BaseProvider) Models() []types.ModelInfo {
	return p.models
}

func (p *BaseProvider) SupportsFeature(feature Feature) bool {
	return p.features[feature]
}

func (p *BaseProvider) ValidateRequest(req *types.ProviderRequest) error {
	if req.Model == "" {
		return types.NewProviderError(p.name, types.ErrInvalidRequest, "model is required")
	}
	if len(req.Messages) == 0 {
		return types.NewProviderError(p.name, types.ErrInvalidRequest, "at least one message is required")
	}
	return nil
}