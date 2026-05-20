package helm

import (
	"context"
	"log/slog"
	"sync"

	"helm.sh/helm/v4/pkg/action"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type cacheKey struct {
	contextName string
	namespace   string
}

// ClientCache memoises *action.Configuration per (contextName, namespace) so
// repeated verb calls don't re-init Helm storage and the kube client.
type ClientCache struct {
	mu    sync.Mutex
	items map[cacheKey]*action.Configuration
}

// NewClientCache constructs an empty cache.
func NewClientCache() *ClientCache {
	return &ClientCache{items: map[cacheKey]*action.Configuration{}}
}

// Get returns a cached *action.Configuration for (contextName, namespace), or
// builds one using the provided RESTClientGetter. Reuses cached entries even
// across different getter arguments — call Evict if you swap kubeconfigs.
func (c *ClientCache) Get(
	_ context.Context,
	contextName, namespace string,
	getter genericclioptions.RESTClientGetter,
	logHandler slog.Handler,
) (*action.Configuration, error) {
	key := cacheKey{contextName: contextName, namespace: namespace}
	c.mu.Lock()
	defer c.mu.Unlock()
	if cfg, ok := c.items[key]; ok {
		return cfg, nil
	}
	var opts []action.ConfigurationOption
	if logHandler != nil {
		opts = append(opts, action.ConfigurationSetLogger(logHandler))
	}
	cfg := action.NewConfiguration(opts...)
	if err := cfg.Init(getter, namespace, "secret"); err != nil {
		return nil, err
	}
	c.items[key] = cfg
	return cfg, nil
}

// Evict drops every cached entry for the given context. Call on cluster
// disconnect or kubeconfig change.
func (c *ClientCache) Evict(contextName string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.items {
		if k.contextName == contextName {
			delete(c.items, k)
		}
	}
}
