package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/tributary-ai/llm-invocation/types"
)

type Registry struct {
	providers map[string]Provider
	mu        sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

func (r *Registry) Register(provider Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	name := provider.Name()
	if name == "" {
		return types.NewError(types.ErrInvalidRequest, "provider name cannot be empty")
	}
	
	if _, exists := r.providers[name]; exists {
		return types.NewError(types.ErrInvalidRequest, fmt.Sprintf("provider %s already registered", name))
	}
	
	r.providers[name] = provider
	return nil
}

func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	provider, exists := r.providers[name]
	if !exists {
		return nil, types.NewError(types.ErrProviderNotFound, fmt.Sprintf("provider %s not found", name))
	}
	
	return provider, nil
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	
	return names
}

func (r *Registry) ListModels(ctx context.Context) ([]types.ModelInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var allModels []types.ModelInfo
	
	for _, provider := range r.providers {
		models := provider.Models()
		allModels = append(allModels, models...)
	}
	
	return allModels, nil
}

func (r *Registry) SelectProvider(req *types.InvocationRequest) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// If provider is explicitly specified, use it
	if req.Provider != "" {
		provider, exists := r.providers[req.Provider]
		if !exists {
			return nil, types.NewError(types.ErrProviderNotFound, fmt.Sprintf("provider %s not found", req.Provider))
		}
		return provider, nil
	}
	
	// Auto-select provider based on model
	for _, provider := range r.providers {
		if r.supportsModel(provider, req.Model) {
			return provider, nil
		}
	}
	
	return nil, types.NewError(types.ErrModelNotSupported, fmt.Sprintf("no provider supports model %s", req.Model))
}

func (r *Registry) supportsModel(provider Provider, modelID string) bool {
	models := provider.Models()
	
	for _, model := range models {
		// Exact match
		if model.ID == modelID {
			return true
		}
		
		// Partial match for model families (e.g., "gpt-4" matches "gpt-4-turbo")
		if strings.HasPrefix(modelID, model.ID) {
			return true
		}
	}
	
	return false
}

func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var errs []string
	
	for name, provider := range r.providers {
		if err := provider.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("failed to close provider %s: %v", name, err))
		}
	}
	
	if len(errs) > 0 {
		return types.NewError("close_failed", strings.Join(errs, "; "))
	}
	
	return nil
}