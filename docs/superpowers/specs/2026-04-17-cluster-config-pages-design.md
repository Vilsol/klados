# Cluster Config Pages

**Date:** 2026-04-17
**Status:** Design

## Goal

Make per-cluster settings discoverable and more useful. Today the page exists at `/settings/clusters/:ctxName` but has no direct entry point from the cluster list, and it exposes only a subset of what's already plumbed in the backend. This iteration adds a discoverable entry point, surfaces metrics configuration, adds read-only inspection info, and adds Disconnect/Forget actions.

No new config features beyond what the backend already supports.

## Scope

1. Gear icon on `ClusterList.svelte` rows → navigates to `/settings/clusters/:ctx`.
2. Expand `ClusterSettings.svelte` with three new sections: **Cluster Info** (read-only), **Metrics**, **Actions** (danger zone).
3. Backend support for "Forget": per-context source kubeconfig path + `RemoveKubeconfigPath`.

## Out of scope

- Per-cluster overrides for insecure-TLS, default namespace, auto-connect, disabled plugins.
- Prometheus connectivity test button.
- Physically editing kubeconfig file contents.

## UI changes

### `ClusterList.svelte`

Add a small gear icon button on each cluster row, placed adjacent to (but visually distinct from) the connect action. Click navigates via `push('/settings/clusters/' + encodeURIComponent(ctx.name))`. Icon uses the existing `Icon` component from `packages/ui` with a settings/gear glyph. Hover state matches existing row action buttons.

### `ClusterSettings.svelte`

Existing sections stay: Display Name, Accent Color, Read-Only, Compact Rows, Favorite Namespaces.

Add three new sections in this order (after existing content):

**Cluster Info (read-only)** — rendered as a definition list:
- Context name
- Cluster name (from kubeconfig `ctx.Cluster`)
- Auth user (from kubeconfig `ctx.AuthInfo`)
- Default namespace (from kubeconfig)
- Server URL
- Server version (when connected)
- Source kubeconfig path — displays the absolute path; when it is the default kubeconfig resolved by `clientcmd.NewDefaultClientConfigLoadingRules()`, append a "(default)" tag
- Connection status
- Detected metrics capabilities: "metrics-server: available/unavailable", "Prometheus: detected at `<url>`" or "Prometheus: not detected"

**Metrics**
- Single text input bound to `ClusterPrefs.Metrics.PrometheusURL`.
- Helper line below showing the currently resolved effective URL (from `ResolveForCluster()`), or "not set" if none.
- Writes are debounced the same way existing fields are (on-change `save()` call).
- Empty value clears the override (set `Metrics` to `nil` when the field is empty, matching the existing pointer-clearing pattern used for `ReadOnly` / `CompactRows`).

**Actions (danger zone)** — visually separated with a top border and red-accented section heading:
- **Disconnect** button — enabled only when connection status is `connected`. Calls existing `ClusterService.Disconnect(ctxName)`.
- **Forget cluster** button — visible only when the context's `SourcePath` is a manually-added path (not the default kubeconfig). Clicking opens a `ConfirmDialog` warning: "This removes all contexts defined in `<path>` from Klados. Your kubeconfig file is not modified." Confirms by calling `ClusterService.RemoveKubeconfigPath(sourcePath)`. After confirmation the user is redirected to `/` since the current context no longer exists.

## Backend changes

### `internal/cluster/manager.go`

Extend `KubeContext` with a source path field:

```go
type KubeContext struct {
    Name        string
    Cluster     string
    User        string
    Namespace   string
    Status      ConnectionStatus
    SourcePath  string  // NEW: kubeconfig file that defined this context
    IsDefault   bool    // NEW: true when SourcePath is part of default loading rules
    // existing fields...
}
```

Update `LoadKubeconfigs` to determine the source path per context. The merged `clientcmd` config does not carry provenance, so the manager must walk the precedence list (default rules + extra paths) and load each file individually to find which file defines each context name. First-match-wins (matches `clientcmd` override precedence).

`IsDefault` is `true` when the source path is one of the paths returned by `clientcmd.NewDefaultClientConfigLoadingRules().Precedence` (before extras are appended).

### `internal/services/cluster.go`

Add:

```go
func (c *ClusterService) RemoveKubeconfigPath(path string) ([]cluster.KubeContext, error)
```

Semantics:
- Removes `path` from `Config.KubeconfigPaths` (no-op if not present).
- Calls `LoadKubeconfigs(cfg.KubeconfigPaths)` to rebuild the context list.
- Emits the same update events as `AddKubeconfigPath`.
- Returns the new context list.
- Returns an error if the path is in the default loading rules (must not attempt to "forget" default kubeconfig entries).

Add:

```go
func (c *ClusterService) GetClusterInfo(ctxName string) (ClusterInfo, error)
```

`ClusterInfo` bundles the read-only data for the new UI section in a single call:

```go
type ClusterInfo struct {
    Context           cluster.KubeContext   // metadata, source path, status, server version
    ServerURL         string                // from raw kubeconfig cluster entry
    MetricsServer     CapabilityState       // available/unavailable/unknown
    PrometheusURL     string                // resolved URL (override or detected)
    PrometheusSource  string                // "detected" | "configured" | ""
}
```

`ServerVersion` is already populated on `KubeContext` by `ListContexts` when connected; not duplicated here.

`PrometheusURL` prefers the configured override (from `ResolveForCluster().Metrics`); falls back to `metrics.DetectPrometheus` for the connected cluster. When disconnected, only metadata is populated.

### Bindings

Regenerate Wails TypeScript bindings after Go changes (`wails3 generate bindings`). New functions appear on the generated `ClusterService` / types.

## Data flow

Forget flow:

```
User clicks Forget → ConfirmDialog → ClusterService.RemoveKubeconfigPath(path)
  → config.Update (remove path) → Manager.LoadKubeconfigs(newPaths)
  → emits cluster:contexts:updated → clusterStore refreshes
  → frontend route redirects to /
```

Info flow:

```
ClusterSettings mounts → GetClusterInfo(ctxName)
  → Manager.ListContexts lookup + Config.ResolveForCluster
  → (if connected) metrics.DetectPrometheus, metrics.MetricsServerProvider.Available
  → returns ClusterInfo
```

## Testing

**Go**
- `TestLoadKubeconfigs_SourcePath_ExtraPath`: a context defined only in an extra path gets that path as `SourcePath` and `IsDefault=false`.
- `TestLoadKubeconfigs_SourcePath_Default`: a context in the default kubeconfig gets `IsDefault=true`.
- `TestLoadKubeconfigs_SourcePath_Precedence`: when a context name exists in multiple files, the first-precedence file wins.
- `TestRemoveKubeconfigPath_RemovesAndReloads`: path removed from config, contexts reloaded.
- `TestRemoveKubeconfigPath_NoOpWhenAbsent`: removing a path not in the list succeeds without error.
- `TestRemoveKubeconfigPath_RejectsDefault`: attempting to remove a default-rules path returns an error.
- `TestGetClusterInfo_Disconnected`: populates metadata only, no server version/metrics detection.
- `TestGetClusterInfo_Connected`: populates server version and metrics capabilities.

**Frontend**
- `ClusterList` test: gear icon click navigates to `/settings/clusters/:ctx`.
- `ClusterSettings` test (mocked bindings): renders Info, Metrics, Actions sections; Prometheus URL input round-trips via `SetClusterPrefs`; Forget button hidden when `IsDefault=true`, visible otherwise.

## Implementation order

1. Backend `KubeContext` + `LoadKubeconfigs` source path tracking + tests.
2. Backend `RemoveKubeconfigPath` + `GetClusterInfo` + tests.
3. Regenerate Wails bindings.
4. Frontend `ClusterSettings` — Cluster Info section.
5. Frontend `ClusterSettings` — Metrics section.
6. Frontend `ClusterSettings` — Actions section + `ConfirmDialog` wiring.
7. Frontend `ClusterList` — gear icon.
8. Frontend tests.
