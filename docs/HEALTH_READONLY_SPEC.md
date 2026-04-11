# Cluster Health Dashboard & Read-only Mode

## Context

Two related features: a cluster-level health dashboard showing infrastructure signals that aren't tied to any specific resource, and a read-only mode that can be toggled globally or inferred from detected RBAC permissions. Both inform the same underlying question — "what can this user do in this cluster?" — so they share data flow and UI conventions.

## Decisions

**Health polling as a separate ticker, not WatchManager**
Most health signals (`/healthz`, `/readyz`, componentstatuses, node count) are either HTTP endpoints or non-watchable resources. The WatchManager's watch-event model doesn't apply. A dedicated goroutine with a 10s ticker per connected context is simpler and correct. On disconnect, the ticker goroutine is cancelled via the existing connection context.

**Cluster health emitted as Wails events**
Rather than a pull RPC, the backend emits `cluster:{ctx}:health` every 10 seconds (and immediately on connect). The frontend subscribes and updates a reactive store. This keeps the health page live without the frontend managing its own polling timer.

**`SelfSubjectRulesReview` on connect, fallback to full access**
Called once per cluster connect against namespace `kube-system` (covers both namespaced and cluster-scoped verbs). If the call fails (403, or distro doesn't support it), `Inferred: true` is set and the frontend assumes full access — no partial lockout from false negatives.

**Read-only is a global config flag, not per-cluster**
It's a deliberate user choice ("I'm browsing production clusters, lock me out of writes"). Stored in `config.json`, loaded on startup, toggled from the header. It applies to the entire app regardless of which cluster is active.

**Layered read-only: config OR detected permissions → same gate**
`isReadOnly := configReadOnly || !permissionSet.CanMutate`. A single computed property drives the UI. The manual toggle overrides; detected permissions add an additional layer when the toggle is off.

**API group unavailability → whole sidebar section grayed out**
If a GVR's API group is not in `DiscoveredResources`, the sidebar entry is rendered but non-interactive with a tooltip: "API group `autoscaling/v2` not available on this cluster." The list page itself shouldn't be navigable — querying an unavailable API would return 404. This signal already exists in `DiscoverResources()` output; it just needs to be surfaced in the sidebar and descriptor registry.

**componentstatuses kept as best-effort**
Known deprecated since 1.19, absent on some distros. If the API returns 404 or non-2xx, the component is shown as "Unavailable (not exposed by this cluster)" rather than "Unhealthy." No error toast.

**Port-forwarding is exempt from read-only mode**
It's a local tunnel, not a mutating API call. Blocking it in read-only would be surprising and unhelpful.

## Rejected Alternatives

**Per-cluster read-only setting**
Would require UI to configure per-cluster and complicate the config schema. The global flag covers the primary use case (cautious browsing of production) without that complexity.

**`SelfSubjectAccessReview` per-verb-per-resource**
Requires N calls to check N resources. `SelfSubjectRulesReview` returns all rules in one call. The tradeoff is that rules-review is namespace-scoped and some distros don't support it — handled by the fallback.

**Health dashboard as part of ClusterOverview**
The signal count is large enough to warrant its own page. ClusterOverview can show a summary badge/widget that links to it.

## Library Selections

| Library | Purpose | Why chosen | Alternatives considered |
|---------|---------|------------|------------------------|
| `k8s.io/client-go/kubernetes` (already present) | `SelfSubjectRulesReview`, componentstatuses, node list | Already in go.mod; typed client for core APIs | dynamic client (would need manual unstructured parsing) |
| `net/http` via REST client | `/healthz`, `/readyz`, `/livez` | Already accessible via `client.Discovery().RESTClient()` | Separate HTTP client (unnecessary) |

## Priorities & Tradeoffs

Optimizing for **correctness over comprehensiveness** — only emit health signals we can reliably interpret. A false "healthy" is better than a false "unhealthy" (hence the componentstatuses fallback). Optimizing for **simplicity of the permission model** — one computed boolean rather than fine-grained verb checks per action. Deprioritizing cert expiry detection (requires cluster-admin access to `secrets` in `kube-system`, too privileged to assume).

## Potential Gotchas

- `SelfSubjectRulesReview` is namespace-scoped. Checking `kube-system` covers most verbs but misses namespace-specific grants. Acceptable for the current use case (coarse read/write detection), but document this limitation.
- `/healthz` etc. must be called via `client.Discovery().RESTClient().Get().AbsPath("/healthz").DoRaw(ctx)` — not via the typed or dynamic client. The response is plain text `ok`, not JSON.
- Node list to count Ready/NotReady nodes requires a `LIST nodes` permission. If the user lacks it, the node gauge should show "Insufficient permissions" rather than failing silently or crashing.
- The 10s ticker goroutine must be started after the metrics-server / API server is confirmed reachable, or the first N ticks will all fail and produce noisy logs. Gate start behind the existing `DiscoverResources` completion.
- `componentstatuses` returns an item per component; the `conditions` field holds status. The `type: Healthy` condition with `status: "True"` is the healthy signal. Some clusters return zero items (not 404) — treat empty list as "not exposed."
- Read-only mode must propagate to plugin command dispatch. `WasmRuntime.CallCommand` should check the flag before invoking; the error message should be "App is in read-only mode."
- API group graying in the sidebar: `DiscoverResources` runs async after connect. There's a brief window where sidebar entries appear before the discovery completes. Show a loading state rather than briefly showing all entries as available.

## Implementation Details

### New types — `internal/cluster/health.go`

```go
type HealthStatus int

const (
    HealthOK HealthStatus = iota
    HealthDegraded
    HealthUnknown // e.g. componentstatuses not exposed
)

type APIServerHealth struct {
    Livez   HealthStatus
    Readyz  HealthStatus
    Healthz HealthStatus
}

type ComponentHealth struct {
    Name    string
    Status  HealthStatus
    Message string
}

type NodeSummary struct {
    Total              int
    Ready              int
    NotReady           int
    SchedulingDisabled int
    PermissionDenied   bool // LIST nodes was denied
}

type ClusterHealth struct {
    APIServer  APIServerHealth
    Components []ComponentHealth // empty = not exposed by cluster
    Nodes      NodeSummary
    CheckedAt  time.Time
}

func CheckHealth(ctx context.Context, conn *Connection) ClusterHealth
```

### New types — `internal/cluster/permissions.go`

```go
type PermissionSet struct {
    Rules    []authv1.ResourceRule
    Inferred bool // true = SelfSubjectRulesReview failed, assume full access
}

func (p PermissionSet) CanMutate() bool {
    if p.Inferred {
        return true
    }
    for _, rule := range p.Rules {
        for _, verb := range rule.Verbs {
            if verb == "*" || verb == "delete" || verb == "patch" || verb == "update" || verb == "create" {
                return true
            }
        }
    }
    return false
}

func FetchPermissions(ctx context.Context, client kubernetes.Interface) PermissionSet
```

### Changes to `internal/cluster/manager.go`

```go
// Added to Connection:
type Connection struct {
    // ... existing fields ...
    Permissions PermissionSet
    healthStop  context.CancelFunc
}

// Manager gains:
func (m *Manager) startHealthPoller(ctx context.Context, connCtx string, conn *Connection)
// - ticker every 10s
// - calls CheckHealth, emits "cluster:{connCtx}:health" via wailsruntime.EventsEmit
// - stopped via conn.healthStop on disconnect
```

### Config change — `internal/config/config.go`

```go
type Config struct {
    // ... existing fields ...
    ReadOnly bool `json:"readOnly,omitempty"`
}
```

### New Wails service methods — `services/AppService`

```go
func (s *AppService) SetReadOnly(ctx context.Context, enabled bool) error
func (s *AppService) GetClusterHealth(ctx context.Context, connCtx string) (cluster.ClusterHealth, error)
// GetClusterHealth is for initial load; subsequent updates arrive via event
```

### Frontend data flow

```
cluster connect
  → FetchPermissions()          → stored in Connection.Permissions
  → startHealthPoller()         → emits cluster:{ctx}:health every 10s
  → DiscoverResources()         → emits discovery:{ctx}:resources (existing)

clusterStore.svelte.ts additions:
  permissions: Record<string, PermissionSet>   // keyed by ctx
  isReadOnly: boolean                          // from config (loaded on init)

computed (per active context):
  canMutate = !isReadOnly && permissions[activeCtx]?.canMutate ?? true
```

### New route

```
/c/:ctx/health  →  ClusterHealthPage.svelte
```

`ClusterOverview` (`/c/:ctx`) gets a `HealthSummaryWidget` that shows a single OK/degraded badge linking to the full page.

### Sidebar grayed-out logic

```typescript
// In sidebar entry rendering:
const available = descriptorRegistry.isGVRAvailable(gvr) // checks discoveredResources
const disabled = !available

// Rendered as:
<SidebarItem
  disabled={disabled}
  tooltip={disabled ? `API group not available on this cluster` : undefined}
/>
```

`DescriptorRegistry` gains `isGVRAvailable(gvr: string): boolean` backed by the set populated from `discovery:{ctx}:resources`.

### Read-only action gate

Single utility used everywhere a mutating action is rendered:

```typescript
// lib/stores/cluster.svelte.ts
canMutate(): boolean  // !isReadOnly && permissions[activeCtx].canMutate

// Usage in any action button:
<Button disabled={!clusterStore.canMutate()} title={!clusterStore.canMutate() ? "Read-only mode" : undefined}>
  Delete
</Button>
```

## Definition of Done

- Health page at `/c/:ctx/health` shows API server (livez/readyz/healthz), component statuses, and node gauge; auto-refreshes every 10s
- componentstatuses showing 404/empty renders "not exposed by this cluster" — no error state
- `SelfSubjectRulesReview` called on connect; failure silently falls back to full-access assumption
- Header toggle persists to `config.json`; survives app restart
- `canMutate()` gates all delete/scale/edit/apply actions across the UI
- Plugin `CallCommand` checks `isReadOnly` and returns an error without invoking Wasm
- Sidebar entries for unavailable API groups are grayed out with tooltip; clicking them does nothing
- Node gauge shows "Insufficient permissions" when LIST nodes is denied rather than showing zeros
- Port-forward is unaffected by read-only mode
