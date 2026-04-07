package resource

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Enricher interface {
	Enrich(contextName string, obj *unstructured.Unstructured) error
}

type EnricherRegistry struct {
	enrichers map[string][]Enricher
}

func NewEnricherRegistry() *EnricherRegistry {
	return &EnricherRegistry{
		enrichers: make(map[string][]Enricher),
	}
}

func (r *EnricherRegistry) Register(gvr string, enricher Enricher) {
	r.enrichers[gvr] = append(r.enrichers[gvr], enricher)
}

func (r *EnricherRegistry) GetAll(gvr string) []Enricher {
	return r.enrichers[gvr]
}

// NamedEnricher is an Enricher that can identify its owning plugin.
type NamedEnricher interface {
	Enricher
	GetPluginName() string
}

// UnregisterPlugin removes all enrichers belonging to the named plugin.
func (r *EnricherRegistry) UnregisterPlugin(name string) {
	for gvr, enrichers := range r.enrichers {
		var filtered []Enricher
		for _, e := range enrichers {
			if ne, ok := e.(NamedEnricher); ok && ne.GetPluginName() == name {
				continue
			}
			filtered = append(filtered, e)
		}
		r.enrichers[gvr] = filtered
	}
}
