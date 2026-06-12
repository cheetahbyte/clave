package update

import (
	"context"
)

type ProviderRegistry struct {
	providers map[ProviderKey]UpdateProvider
}

func NewProviderRegistry(providers ...UpdateProvider) *ProviderRegistry {
	r := &ProviderRegistry{
		providers: make(map[ProviderKey]UpdateProvider, len(providers)),
	}
	for _, p := range providers {
		r.Register(p)
	}
	return r
}

func (r *ProviderRegistry) Register(p UpdateProvider) {
	r.providers[p.Key()] = p
}

func (r *ProviderRegistry) Get(key ProviderKey) (UpdateProvider, bool) {
	p, ok := r.providers[key]
	return p, ok
}

func (r *ProviderRegistry) List(ctx context.Context) []ProviderInfo {
	result := make([]ProviderInfo, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, ProviderInfo{
			Key:          string(p.Key()),
			Name:         p.Name(),
			Capabilities: p.Capabilities(ctx),
		})
	}
	return result
}
