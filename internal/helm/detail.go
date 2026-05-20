package helm

import (
	"context"
	"fmt"

	release "helm.sh/helm/v4/pkg/release/v1"
	"sigs.k8s.io/yaml"

	chartutil "helm.sh/helm/v4/pkg/chart/common/util"
)

// Hook is a flattened, JSON-friendly view of a single Helm release hook for
// the UI's "Hooks" detail tab.
type Hook struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Path        string   `json:"path"`
	Events      []string `json:"events,omitempty"`
	Weight      int      `json:"weight"`
	Phase       string   `json:"phase,omitempty"`
	StartedAt   string   `json:"startedAt,omitempty"`
	CompletedAt string   `json:"completedAt,omitempty"`
	Manifest    string   `json:"manifest,omitempty"`
}

// Get returns the latest-revision virtual unstructured object for the named
// release in (contextName, namespace). Implements resource.VirtualBackend.Get.
func (b *Backend) Get(ctx context.Context, contextName, namespace, name string) (map[string]any, error) {
	items, _, err := b.List(ctx, contextName, namespace)
	if err != nil {
		return nil, err
	}
	for _, obj := range items {
		meta, _ := obj["metadata"].(map[string]any)
		n, _ := meta["name"].(string)
		if n == name {
			return obj, nil
		}
	}
	return nil, fmt.Errorf("%w: %s/%s", ErrReleaseNotFound, namespace, name)
}

// GetValues returns the YAML-encoded values for a specific revision. If
// computed is true, chart defaults are merged with user-supplied config
// before serialisation. Secrets are masked. Revision <=0 selects the latest.
func (b *Backend) GetValues(ctx context.Context, contextName, namespace, releaseName string, computed bool, revision int) (string, error) {
	rel, err := b.releaseAtRevision(ctx, contextName, namespace, releaseName, revision)
	if err != nil {
		return "", err
	}
	var values map[string]any
	if computed {
		values = mergedValues(rel)
	} else {
		// Deep-copy via the helper used by diff; treat nil as empty map for stable YAML.
		copied := deepCopyMap(rel.Config)
		if cp, ok := copied.(map[string]any); ok {
			values = cp
		} else {
			values = map[string]any{}
		}
	}
	MaskValues(values)
	out, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("helm getvalues: %w", err)
	}
	return string(out), nil
}

// GetManifest returns the rendered manifest for a specific revision. Revision
// <=0 selects the latest.
func (b *Backend) GetManifest(ctx context.Context, contextName, namespace, releaseName string, revision int) (string, error) {
	rel, err := b.releaseAtRevision(ctx, contextName, namespace, releaseName, revision)
	if err != nil {
		return "", err
	}
	return rel.Manifest, nil
}

// GetNotes returns the rendered NOTES.txt for a specific revision. Revision
// <=0 selects the latest.
func (b *Backend) GetNotes(ctx context.Context, contextName, namespace, releaseName string, revision int) (string, error) {
	rel, err := b.releaseAtRevision(ctx, contextName, namespace, releaseName, revision)
	if err != nil {
		return "", err
	}
	if rel.Info == nil {
		return "", nil
	}
	return rel.Info.Notes, nil
}

// GetHooks returns the flattened hook list for a specific revision. Revision
// <=0 selects the latest.
func (b *Backend) GetHooks(ctx context.Context, contextName, namespace, releaseName string, revision int) ([]Hook, error) {
	rel, err := b.releaseAtRevision(ctx, contextName, namespace, releaseName, revision)
	if err != nil {
		return nil, err
	}
	out := make([]Hook, 0, len(rel.Hooks))
	for _, h := range rel.Hooks {
		if h == nil {
			continue
		}
		events := make([]string, 0, len(h.Events))
		for _, e := range h.Events {
			events = append(events, string(e))
		}
		hook := Hook{
			Name:     h.Name,
			Kind:     h.Kind,
			Path:     h.Path,
			Events:   events,
			Weight:   h.Weight,
			Phase:    string(h.LastRun.Phase),
			Manifest: h.Manifest,
		}
		if !h.LastRun.StartedAt.IsZero() {
			hook.StartedAt = h.LastRun.StartedAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		if !h.LastRun.CompletedAt.IsZero() {
			hook.CompletedAt = h.LastRun.CompletedAt.UTC().Format("2006-01-02T15:04:05Z")
		}
		out = append(out, hook)
	}
	return out, nil
}

// releaseAtRevision finds the requested release revision (or the latest if
// revision <= 0) by listing the release's secrets and decoding each one.
func (b *Backend) releaseAtRevision(ctx context.Context, contextName, namespace, releaseName string, revision int) (*release.Release, error) {
	secrets, _, err := b.secretLister.ListSecrets(ctx, contextName, namespace, "", "name="+releaseName)
	if err != nil {
		return nil, fmt.Errorf("helm release lookup: %w", err)
	}
	if len(secrets) == 0 {
		return nil, fmt.Errorf("%w: %s/%s", ErrReleaseNotFound, namespace, releaseName)
	}
	flat, err := ReassembleContinuation(secrets)
	if err != nil {
		flat = secrets
	}
	var best *release.Release
	for i := range flat {
		rel, err := DecodeRelease(&flat[i])
		if err != nil {
			continue
		}
		if revision > 0 {
			if rel.Version == revision {
				return rel, nil
			}
			continue
		}
		if best == nil || rel.Version > best.Version {
			best = rel
		}
	}
	if best == nil {
		if revision > 0 {
			return nil, fmt.Errorf("%w: %d", ErrRevisionNotFound, revision)
		}
		return nil, fmt.Errorf("%w: %s/%s", ErrReleaseNotFound, namespace, releaseName)
	}
	return best, nil
}

// mergedValues returns chart defaults merged with release config. Errors fall
// back to a copy of release config.
func mergedValues(rel *release.Release) map[string]any {
	if rel == nil {
		return map[string]any{}
	}
	if rel.Chart == nil {
		if cp, ok := deepCopyMap(rel.Config).(map[string]any); ok {
			return cp
		}
		return map[string]any{}
	}
	merged, err := chartutil.CoalesceValues(rel.Chart, rel.Config)
	if err != nil {
		if cp, ok := deepCopyMap(rel.Config).(map[string]any); ok {
			return cp
		}
		return map[string]any{}
	}
	// Deep copy so masking does not mutate the chart-shared map.
	if cp, ok := deepCopyMap(map[string]any(merged)).(map[string]any); ok {
		return cp
	}
	return map[string]any{}
}
