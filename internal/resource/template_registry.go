package resource

import (
	"github.com/sasha-s/go-deadlock"
)

type TemplateRegistry struct {
	mu      deadlock.RWMutex
	builtin map[string][]Template
	plugins map[string][]Template
}

func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{
		builtin: make(map[string][]Template),
		plugins: make(map[string][]Template),
	}
}

func (r *TemplateRegistry) Register(t Template) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builtin[t.GVR] = append(r.builtin[t.GVR], t)
}

func (r *TemplateRegistry) RegisterPlugin(pluginName string, t Template) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[pluginName] = append(r.plugins[pluginName], t)
}

func (r *TemplateRegistry) UnregisterPlugin(pluginName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.plugins, pluginName)
}

func (r *TemplateRegistry) GetTemplates(gvr string) []Template {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Template
	out = append(out, r.builtin[gvr]...)
	for _, templates := range r.plugins {
		for _, t := range templates {
			if t.GVR == gvr {
				out = append(out, t)
			}
		}
	}
	return out
}

func (r *TemplateRegistry) GetAllGVRs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{})
	for gvr := range r.builtin {
		seen[gvr] = struct{}{}
	}
	for _, templates := range r.plugins {
		for _, t := range templates {
			seen[t.GVR] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for gvr := range seen {
		out = append(out, gvr)
	}
	return out
}
