package helm

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	release "helm.sh/helm/v4/pkg/release/v1"
	corev1 "k8s.io/api/core/v1"
)

// VirtualGVR is the synthetic GVR for Helm releases. The "helm" pseudo-group
// distinguishes it from any real cluster resource; the dot-separated form
// matches the klados ParseGVR convention.
const VirtualGVR = "helm.klados.io.v1.releases"

// VirtualAPIVersion is the synthetic apiVersion stamped on virtual unstructured
// release objects.
const VirtualAPIVersion = "helm.klados.io/v1"

// VirtualKind is the synthetic kind stamped on virtual unstructured release
// objects.
const VirtualKind = "Release"

// VirtualEvent is a synthesized watch event for a Helm release.
type VirtualEvent struct {
	// Type is one of "ADDED", "MODIFIED", "DELETED".
	Type string
	// Object is the virtual unstructured map for the latest revision of the
	// release. For DELETED events this is the last-known virtual object.
	Object map[string]any
}

type revisionMeta struct {
	rel        *release.Release
	rev        int
	deployedAt time.Time
	rvSecret   string // resourceVersion of the underlying Secret
}

// revisionSnapshot captures the user-visible identity of an emitted MODIFIED
// event so the aggregator can suppress no-op duplicates.
type revisionSnapshot struct {
	rev      int
	status   string
	chartVer string
}

// Aggregator collapses per-revision Helm release Secrets into one virtual
// unstructured row per release (latest revision wins, with deployed-at as
// tiebreak).
//
// Aggregator is safe for concurrent use; the snapshot path takes the same
// lock as the delta path so a Reset can run mid-watch without races.
type Aggregator struct {
	mu sync.Mutex
	// releases: namespace -> name -> revision -> meta
	releases map[string]map[string]map[int]revisionMeta
	// lastEmitted: namespace -> name -> last snapshot we returned to a caller.
	// Used to suppress redundant MODIFIED events when the user-visible state
	// has not changed.
	lastEmitted map[string]map[string]revisionSnapshot
}

// NewAggregator constructs an empty Aggregator.
func NewAggregator() *Aggregator {
	return &Aggregator{
		releases:    map[string]map[string]map[int]revisionMeta{},
		lastEmitted: map[string]map[string]revisionSnapshot{},
	}
}

// CollapseSnapshot replaces the aggregator's internal state with the given
// secrets (typically the result of a fresh ListSecrets call) and returns the
// virtual objects for the latest revision of every release.
func (a *Aggregator) CollapseSnapshot(secrets []corev1.Secret) ([]map[string]any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.releases = map[string]map[string]map[int]revisionMeta{}
	a.lastEmitted = map[string]map[string]revisionSnapshot{}
	for i := range secrets {
		s := &secrets[i]
		rel, err := DecodeRelease(s)
		if err != nil {
			// Skip malformed individual secrets — they shouldn't poison the
			// whole snapshot. Caller logs the slice separately if it cares.
			continue
		}
		a.insertLocked(rel, s.ResourceVersion)
	}
	out := a.snapshotLocked()
	// Prime lastEmitted with what we just returned so subsequent ApplyDelta
	// calls can dedupe against it.
	for ns, byName := range a.releases {
		for name := range byName {
			if m, ok := a.latestLocked(ns, name); ok {
				a.rememberEmittedLocked(ns, name, snapshotOf(m))
			}
		}
	}
	return out, nil
}

// ApplyDelta integrates a single watch event into the aggregator state and
// returns the synthesized event, or nil if the event does not change what
// the consumer sees (e.g. DELETED of a non-latest revision).
//
// eventType is one of "ADDED", "MODIFIED", "DELETED".
func (a *Aggregator) ApplyDelta(eventType string, s *corev1.Secret) (*VirtualEvent, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	rel, err := DecodeRelease(s)
	if err != nil {
		return nil, err
	}
	ns := rel.Namespace
	name := rel.Name
	switch eventType {
	case "ADDED", "MODIFIED":
		// Track whether this release existed before.
		_, existed := a.releases[ns][name]
		a.insertLocked(rel, s.ResourceVersion)
		newLatest, _ := a.latestLocked(ns, name)
		obj := buildVirtualObject(newLatest)
		newSnap := snapshotOf(newLatest)
		if !existed {
			a.rememberEmittedLocked(ns, name, newSnap)
			return &VirtualEvent{Type: "ADDED", Object: obj}, nil
		}
		prevSnap, hadPrev := a.lastEmitted[ns][name]
		if hadPrev && prevSnap == newSnap {
			// User-visible state unchanged — suppress.
			return nil, nil
		}
		a.rememberEmittedLocked(ns, name, newSnap)
		return &VirtualEvent{Type: "MODIFIED", Object: obj}, nil
	case "DELETED":
		_, existed := a.releases[ns][name]
		if !existed {
			return nil, nil
		}
		prevLatest, _ := a.latestLocked(ns, name)
		// Remove the specific revision.
		delete(a.releases[ns][name], rel.Version)
		if len(a.releases[ns][name]) == 0 {
			// Last revision gone — emit DELETED with the last-known object.
			obj := buildVirtualObject(prevLatest)
			delete(a.releases[ns], name)
			if len(a.releases[ns]) == 0 {
				delete(a.releases, ns)
			}
			a.forgetEmittedLocked(ns, name)
			return &VirtualEvent{Type: "DELETED", Object: obj}, nil
		}
		newLatest, _ := a.latestLocked(ns, name)
		if prevLatest != nil && newLatest != nil && prevLatest.rev == newLatest.rev && prevLatest.rvSecret == newLatest.rvSecret {
			// Deleted revision was not the latest — no visible change.
			return nil, nil
		}
		// Latest changed — emit MODIFIED with new latest.
		newSnap := snapshotOf(newLatest)
		prevSnap, hadPrev := a.lastEmitted[ns][name]
		if hadPrev && prevSnap == newSnap {
			return nil, nil
		}
		a.rememberEmittedLocked(ns, name, newSnap)
		obj := buildVirtualObject(newLatest)
		return &VirtualEvent{Type: "MODIFIED", Object: obj}, nil
	default:
		return nil, fmt.Errorf("aggregator: unknown event type %q", eventType)
	}
}

// Reset clears all releases for the given namespace. If namespace is empty,
// clears the entire aggregator (use on full reconnect).
func (a *Aggregator) Reset(namespace string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if namespace == "" {
		a.releases = map[string]map[string]map[int]revisionMeta{}
		a.lastEmitted = map[string]map[string]revisionSnapshot{}
		return
	}
	delete(a.releases, namespace)
	delete(a.lastEmitted, namespace)
}

func (a *Aggregator) rememberEmittedLocked(ns, name string, snap revisionSnapshot) {
	if a.lastEmitted[ns] == nil {
		a.lastEmitted[ns] = map[string]revisionSnapshot{}
	}
	a.lastEmitted[ns][name] = snap
}

func (a *Aggregator) forgetEmittedLocked(ns, name string) {
	if a.lastEmitted[ns] == nil {
		return
	}
	delete(a.lastEmitted[ns], name)
	if len(a.lastEmitted[ns]) == 0 {
		delete(a.lastEmitted, ns)
	}
}

// snapshotOf extracts the user-visible identity bits we dedupe on.
func snapshotOf(m *revisionMeta) revisionSnapshot {
	if m == nil || m.rel == nil {
		return revisionSnapshot{}
	}
	var status, chartVer string
	if m.rel.Info != nil {
		status = string(m.rel.Info.Status)
	}
	if m.rel.Chart != nil && m.rel.Chart.Metadata != nil {
		chartVer = m.rel.Chart.Metadata.Version
	}
	return revisionSnapshot{rev: m.rev, status: status, chartVer: chartVer}
}

func (a *Aggregator) insertLocked(rel *release.Release, rv string) {
	if a.releases[rel.Namespace] == nil {
		a.releases[rel.Namespace] = map[string]map[int]revisionMeta{}
	}
	if a.releases[rel.Namespace][rel.Name] == nil {
		a.releases[rel.Namespace][rel.Name] = map[int]revisionMeta{}
	}
	var deployedAt time.Time
	if rel.Info != nil {
		deployedAt = rel.Info.LastDeployed
	}
	a.releases[rel.Namespace][rel.Name][rel.Version] = revisionMeta{
		rel:        rel,
		rev:        rel.Version,
		deployedAt: deployedAt,
		rvSecret:   rv,
	}
}

// latestLocked returns the latest-revision metadata for (ns, name), or nil if
// no revisions exist. Latest = max revision number, with deployedAt as tiebreak.
func (a *Aggregator) latestLocked(ns, name string) (*revisionMeta, bool) {
	revs, ok := a.releases[ns][name]
	if !ok || len(revs) == 0 {
		return nil, false
	}
	var best revisionMeta
	first := true
	for _, m := range revs {
		if first {
			best = m
			first = false
			continue
		}
		if m.rev > best.rev || (m.rev == best.rev && m.deployedAt.After(best.deployedAt)) {
			best = m
		}
	}
	return &best, true
}

func (a *Aggregator) snapshotLocked() []map[string]any {
	// Stable ordering: namespace asc, name asc.
	var namespaces []string
	for ns := range a.releases {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)
	out := make([]map[string]any, 0)
	for _, ns := range namespaces {
		var names []string
		for n := range a.releases[ns] {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if m, ok := a.latestLocked(ns, n); ok {
				out = append(out, buildVirtualObject(m))
			}
		}
	}
	return out
}

// buildVirtualObject constructs the synthetic unstructured map for the latest
// revision of a release.
func buildVirtualObject(m *revisionMeta) map[string]any {
	if m == nil {
		return nil
	}
	rel := m.rel
	var status string
	var lastDeployed string
	if rel.Info != nil {
		status = string(rel.Info.Status)
		lastDeployed = rel.Info.LastDeployed.UTC().Format(time.RFC3339)
	}
	var chartName, chartVersion, appVersion string
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		chartName = rel.Chart.Metadata.Name
		chartVersion = rel.Chart.Metadata.Version
		appVersion = rel.Chart.Metadata.AppVersion
	}
	return map[string]any{
		"apiVersion": VirtualAPIVersion,
		"kind":       VirtualKind,
		"metadata": map[string]any{
			"name":              rel.Name,
			"namespace":         rel.Namespace,
			"uid":               syntheticUID(rel.Namespace, rel.Name),
			"resourceVersion":   m.rvSecret,
			"creationTimestamp": lastDeployed,
		},
		"spec": map[string]any{
			"chart":        chartName,
			"chartVersion": chartVersion,
			"appVersion":   appVersion,
			"revision":     int64(rel.Version),
			"status":       status,
			"deployedAt":   lastDeployed,
		},
		"status": map[string]any{},
	}
}

// syntheticUID returns a stable hex SHA1 of "namespace/name" so the frontend's
// row-keying works even though Helm releases have no real UID.
func syntheticUID(ns, name string) string {
	h := sha1.Sum([]byte(ns + "/" + name))
	return hex.EncodeToString(h[:])
}
