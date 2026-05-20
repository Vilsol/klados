package helm

import (
	"context"
	"fmt"

	"github.com/Vilsol/slox"
	corev1 "k8s.io/api/core/v1"
)

// secretLister abstracts the actual cluster list call. In tests we inject a
// canned list; in production Task 3 will wire the cluster.Manager-backed
// implementation.
type secretLister interface {
	ListSecrets(ctx context.Context, contextName, namespace, fieldSelector, labelSelector string) ([]corev1.Secret, string, error)
}

// Backend implements the virtual-backend shape that Task 3 will wire into the
// resource engine: List/Get/Watch over a synthetic GVR.
type Backend struct {
	cache        *ClientCache
	aggregator   *Aggregator
	secretLister secretLister
}

// NewBackend constructs a Helm Backend.
func NewBackend(cache *ClientCache, lister secretLister) *Backend {
	return &Backend{
		cache:        cache,
		aggregator:   NewAggregator(),
		secretLister: lister,
	}
}

// Aggregator exposes the underlying aggregator so callers can apply deltas
// from the wired watch loop (Task 3).
func (b *Backend) Aggregator() *Aggregator {
	return b.aggregator
}

// List returns the latest-revision virtual object for every release visible
// in (contextName, namespace). Pass namespace="" for all-namespaces.
//
// Return shape mirrors the existing resource engine: ([]map[string]any, resourceVersion, error).
func (b *Backend) List(ctx context.Context, contextName, namespace string) ([]map[string]any, string, error) {
	secrets, rv, err := b.secretLister.ListSecrets(ctx, contextName, namespace, "type="+ReleaseSecretType, "")
	if err != nil {
		return nil, "", fmt.Errorf("helm list: %w", err)
	}
	reassembled, err := ReassembleContinuation(secrets)
	if err != nil {
		// One malformed chunk shouldn't kill the whole list; log and continue
		// with the secrets we have.
		slox.Warn(ctx, "helm: continuation reassembly failed; falling back to raw secrets", "err", err)
		reassembled = secrets
	}
	out, err := b.aggregator.CollapseSnapshot(reassembled)
	if err != nil {
		return nil, "", fmt.Errorf("helm collapse: %w", err)
	}
	return out, rv, nil
}
