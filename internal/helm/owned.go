package helm

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Vilsol/slox"
	"sigs.k8s.io/yaml"
)

// resourceGetter abstracts the cluster lookups that owned-resource discovery
// needs. Task 3 wires the real implementation.
type resourceGetter interface {
	// Exists checks whether (gvr, ns, name) currently exists in the cluster.
	Exists(ctx context.Context, contextName, gvr, namespace, name string) (bool, error)

	// ListByLabel lists objects of the given GVR in the given namespace
	// matching the label selector and returns each as an unstructured map.
	ListByLabel(ctx context.Context, contextName, gvr, namespace, labelSelector string) ([]map[string]any, error)

	// KnownGVRs returns every namespaced GVR known to the cluster's discovery
	// for use when "scan all" is requested.
	KnownGVRs(contextName string) []string
}

// OwnedRef is a single resource owned by a Helm release.
type OwnedRef struct {
	APIVersion     string
	Kind           string
	GVR            string
	Namespace      string
	Name           string
	ResourcePolicy string
	Exists         bool
}

// defaultLabelScanGVRs is the bounded set walked by the label-fallback pass.
// Keep this list short — broader scans require scanAll=true.
var defaultLabelScanGVRs = []string{
	"apps.v1.deployments",
	"apps.v1.statefulsets",
	"apps.v1.daemonsets",
	"apps.v1.replicasets",
	"core.v1.services",
	"core.v1.configmaps",
	"core.v1.secrets",
	"core.v1.serviceaccounts",
	"core.v1.persistentvolumeclaims",
	"networking.k8s.io.v1.ingresses",
	"networking.k8s.io.v1.networkpolicies",
	"batch.v1.jobs",
	"batch.v1.cronjobs",
	"rbac.authorization.k8s.io.v1.roles",
	"rbac.authorization.k8s.io.v1.rolebindings",
	"autoscaling.v2.horizontalpodautoscalers",
}

// GetOwnedResources returns the union of (a) the resources declared in the
// release's manifest and (b) anything in the cluster bearing the standard
// Helm management labels for this release. Existence is filled in via the
// injected resourceGetter.
//
// If scanAll is true, the label-fallback pass walks every namespaced GVR
// reported by resourceGetter.KnownGVRs instead of the bounded default set.
func (b *Backend) GetOwnedResources(
	ctx context.Context,
	contextName, namespace, releaseName string,
	scanAll bool,
	getter resourceGetter,
) ([]OwnedRef, error) {
	hist, err := b.GetHistory(ctx, contextName, namespace, releaseName)
	if err != nil {
		return nil, err
	}
	if len(hist) == 0 {
		return nil, fmt.Errorf("%w: %s/%s", ErrReleaseNotFound, namespace, releaseName)
	}

	// Re-fetch the secrets to pull the manifest for the latest revision.
	secrets, _, err := b.secretLister.ListSecrets(ctx, contextName, namespace, "", "name="+releaseName)
	if err != nil {
		return nil, fmt.Errorf("helm owned: %w", err)
	}
	flat, err := ReassembleContinuation(secrets)
	if err != nil {
		flat = secrets
	}
	var manifest string
	latestRev := hist[0].Number
	for i := range flat {
		rel, err := DecodeRelease(&flat[i])
		if err != nil {
			continue
		}
		if rel.Version == latestRev {
			manifest = rel.Manifest
			break
		}
	}

	refs := parseManifest(manifest, namespace)

	// Existence check for manifest refs.
	for i := range refs {
		ok, err := getter.Exists(ctx, contextName, refs[i].GVR, refs[i].Namespace, refs[i].Name)
		if err != nil {
			slox.Warn(ctx, "helm owned: existence check failed", "gvr", refs[i].GVR, "name", refs[i].Name, "err", err)
			continue
		}
		refs[i].Exists = ok
	}

	// Label-fallback pass.
	scanGVRs := defaultLabelScanGVRs
	if scanAll {
		scanGVRs = getter.KnownGVRs(contextName)
	}
	selector := fmt.Sprintf("app.kubernetes.io/managed-by=Helm,meta.helm.sh/release-name=%s", releaseName)
	seen := map[string]bool{}
	for _, r := range refs {
		seen[r.GVR+"|"+r.Namespace+"|"+r.Name] = true
	}
	for _, gvr := range scanGVRs {
		items, err := getter.ListByLabel(ctx, contextName, gvr, namespace, selector)
		if err != nil {
			slox.Warn(ctx, "helm owned: label scan failed", "gvr", gvr, "err", err)
			continue
		}
		for _, obj := range items {
			meta, _ := obj["metadata"].(map[string]any)
			name, _ := meta["name"].(string)
			ns, _ := meta["namespace"].(string)
			if ns == "" {
				ns = namespace
			}
			key := gvr + "|" + ns + "|" + name
			if seen[key] {
				continue
			}
			seen[key] = true
			apiVersion, _ := obj["apiVersion"].(string)
			kind, _ := obj["kind"].(string)
			refs = append(refs, OwnedRef{
				APIVersion: apiVersion,
				Kind:       kind,
				GVR:        gvr,
				Namespace:  ns,
				Name:       name,
				Exists:     true,
			})
		}
	}

	sort.Slice(refs, func(i, j int) bool {
		if refs[i].GVR != refs[j].GVR {
			return refs[i].GVR < refs[j].GVR
		}
		if refs[i].Namespace != refs[j].Namespace {
			return refs[i].Namespace < refs[j].Namespace
		}
		return refs[i].Name < refs[j].Name
	})
	return refs, nil
}

// parseManifest splits a multi-document YAML manifest and returns OwnedRef
// entries for each meaningful document. defaultNS is used when a document
// omits metadata.namespace.
func parseManifest(manifest, defaultNS string) []OwnedRef {
	if strings.TrimSpace(manifest) == "" {
		return nil
	}
	docs := splitYAMLDocs(manifest)
	seen := map[string]bool{}
	out := make([]OwnedRef, 0, len(docs))
	for _, doc := range docs {
		if strings.TrimSpace(doc) == "" {
			continue
		}
		var m map[string]any
		if err := yaml.Unmarshal([]byte(doc), &m); err != nil || m == nil {
			continue
		}
		apiVersion, _ := m["apiVersion"].(string)
		kind, _ := m["kind"].(string)
		meta, _ := m["metadata"].(map[string]any)
		if meta == nil {
			continue
		}
		name, _ := meta["name"].(string)
		ns, _ := meta["namespace"].(string)
		if ns == "" {
			ns = defaultNS
		}
		if name == "" || apiVersion == "" || kind == "" {
			continue
		}
		policy := ""
		if anns, ok := meta["annotations"].(map[string]any); ok {
			if p, ok := anns["helm.sh/resource-policy"].(string); ok {
				policy = p
			}
		}
		gvr := apiVersionKindToGVR(apiVersion, kind)
		key := gvr + "|" + ns + "|" + name
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, OwnedRef{
			APIVersion:     apiVersion,
			Kind:           kind,
			GVR:            gvr,
			Namespace:      ns,
			Name:           name,
			ResourcePolicy: policy,
		})
	}
	return out
}

// splitYAMLDocs splits a manifest on lines that consist solely of "---"
// (optionally with leading/trailing whitespace). It strips empty and
// comment-only documents.
func splitYAMLDocs(manifest string) []string {
	lines := strings.Split(manifest, "\n")
	var docs []string
	var cur []string
	flush := func() {
		text := strings.Join(cur, "\n")
		cur = cur[:0]
		// Strip pure-comment / empty docs.
		hasContent := false
		for _, l := range strings.Split(text, "\n") {
			t := strings.TrimSpace(l)
			if t == "" || strings.HasPrefix(t, "#") {
				continue
			}
			hasContent = true
			break
		}
		if hasContent {
			docs = append(docs, text)
		}
	}
	for _, l := range lines {
		if strings.TrimSpace(l) == "---" {
			flush()
			continue
		}
		cur = append(cur, l)
	}
	flush()
	return docs
}

// apiVersionKindToGVR maps (apiVersion, kind) into the klados dot-separated
// GVR. Resource pluralisation uses a naive English heuristic; consumers
// (resource engine) can normalise via REST mapper later.
func apiVersionKindToGVR(apiVersion, kind string) string {
	group := ""
	version := apiVersion
	if i := strings.Index(apiVersion, "/"); i >= 0 {
		group = apiVersion[:i]
		version = apiVersion[i+1:]
	}
	if group == "" {
		group = "core"
	}
	return fmt.Sprintf("%s.%s.%s", group, version, pluralise(strings.ToLower(kind)))
}

// TODO: replace with REST mapper normalisation when Phase 2 wires the real engine.
func pluralise(s string) string {
	switch {
	case strings.HasSuffix(s, "s"), strings.HasSuffix(s, "x"), strings.HasSuffix(s, "ch"), strings.HasSuffix(s, "sh"):
		return s + "es"
	case strings.HasSuffix(s, "y") && len(s) > 1 && !isVowel(s[len(s)-2]):
		return s[:len(s)-1] + "ies"
	default:
		return s + "s"
	}
}

func isVowel(b byte) bool {
	switch b {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}
