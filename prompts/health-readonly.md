# Cluster Health Dashboard & Read-only Mode

Add a live cluster health dashboard page and a global read-only toggle, with RBAC-inferred permissions gating all mutating actions across the UI.

## First Action

Read `internal/cluster/manager.go` lines 163–308 — specifically the `Connect()` function (line 163) and the existing `healthMonitor` goroutine (line 455). You'll add two new goroutines at the same call sites in `Connect()`: one that runs `FetchPermissions` once, and one that starts the 10s health polling ticker. Understanding how `healthMonitor` uses `conn.Clientset.Discovery().RESTClient().Get().AbsPath("/healthz").Do(ctx).Raw()` gives you the exact pattern for all three new API server health probes.

## Context

Klados has a cluster connection pipeline (`cluster.Manager`) that already starts per-connection goroutines for health monitoring, discovery, server version, and metrics detection. This work extends that pipeline with two new concerns: (1) a richer cluster health signal (API server livez/readyz, componentstatuses, node readiness) emitted as a Wails event every 10 seconds so a new dashboard page stays live, and (2) RBAC permission detection via `SelfSubjectRulesReview` that, combined with a global config flag, drives a unified `canMutate()` gate across the entire UI.

## Files to Read

- `internal/cluster/manager.go` — **what to look for**: `Connection` struct (line 76, fields to extend with `Permissions` and `healthStop`), `Connect()` (line 163, where to add the two new goroutines alongside `go m.healthMonitor`), and `healthMonitor` (line 455, the RESTClient AbsPath pattern to reuse for `/livez`/`/readyz`/`/healthz`)
- `internal/config/config.go` — **what to look for**: the `Config` struct (line 31) and its existing fields — `ReadOnly bool` goes at the end following the same `json:"...,omitempty"` pattern
- `internal/services/app.go` — **what to look for**: `AppService` struct and the existing `Config()` accessor (line 100) — `SetReadOnly` and `GetClusterHealth` are added here as new exported Wails-bound methods
- `internal/services/cluster.go` — **what to look for**: how existing cluster Wails methods are structured (parameter order, error returns, slox logging) — `GetClusterHealth` follows the same pattern
- `frontend/src/lib/stores/cluster.svelte.ts` — **what to look for**: the class structure, how `$state` fields are added (activeContext, selectedNamespaces pattern), and the Wails event subscription pattern in `onContextConnected` — `permissions` and `isReadOnly` are new `$state` fields; `canMutate()` is a method
- `frontend/src/lib/components/Sidebar.svelte` — **what to look for**: lines 115–210, specifically `handleDiscovery()` (line 182) which populates `customResources` from `discovery:{ctx}:resources` events — the `discoveredGVRs` Set built inside here is the source for `isGVRAvailable()`; the `gvrGroups` entries (line 266) are where `disabled` prop and tooltip are added
- `frontend/src/lib/registry/index.ts` — **what to look for**: the `DescriptorRegistry` class — `isGVRAvailable(gvr: string): boolean` is a new method backed by a `Set<string>` that Sidebar populates via a new exported function

## Source Documents

- `HEALTH_READONLY_SPEC.md` — full spec: all decisions, rejected alternatives, type definitions, data flow, sidebar grayed-out logic, and the complete Definition of Done

## What Exists

- `cluster.Connection` struct with `KubeContext`, `Config`, `Clientset`, `Dynamic`, `Discovery`, `MetricsCapability`, `cancel` — no `Permissions` or `healthStop` yet
- `cluster.Manager.Connect()` already starts `go m.healthMonitor`, `go m.emitDiscovery`, server version goroutine, and metrics detection goroutine — two more goroutines slot in here
- `healthMonitor` uses `conn.Clientset.Discovery().RESTClient().Get().AbsPath("/healthz").Do(ctx).Raw()` — exact pattern for the new health checker
- `config.Config` struct with all existing fields — no `ReadOnly` field yet
- `AppService` with `Config()`, `ClusterManager()` accessors and Wails lifecycle methods
- `clusterStore` singleton with `activeContext`, `selectedNamespaces`, namespace loading, and Wails event subscriptions — no `permissions` or `isReadOnly` state
- `Sidebar.svelte` already subscribes to `discovery:{ctx}:resources` events and maintains a `customResources` list — no grayed-out logic yet
- `DescriptorRegistry` in `registry/index.ts` with `get(gvr)`, `set()`, `reloadPlugins()` — no `isGVRAvailable()` method

## Deliverables

1. **`internal/cluster/health.go`** — `HealthStatus` (int const: OK/Degraded/Unknown), `APIServerHealth`, `ComponentHealth`, `NodeSummary`, `ClusterHealth` types; `CheckHealth(ctx, conn) ClusterHealth` function that probes `/livez`, `/readyz`, `/healthz` via RESTClient AbsPath, queries componentstatuses (treats 404 and empty list as Unknown, not Degraded), and lists nodes (sets `NodeSummary.PermissionDenied = true` on 403)

2. **`internal/cluster/permissions.go`** — `PermissionSet` struct with `Rules []authv1.ResourceRule` and `Inferred bool`; `CanMutate() bool` method (returns true if Inferred, else scans Rules for `*`/`delete`/`patch`/`update`/`create` verb); `FetchPermissions(ctx, client kubernetes.Interface) PermissionSet` that calls `SelfSubjectRulesReview` in `kube-system` namespace and sets `Inferred: true` on any error

3. **`internal/cluster/manager.go`** — extend `Connection` with `Permissions PermissionSet` and `healthStop context.CancelFunc`; add `startHealthPoller(ctx, connCtx string, conn *Connection)` method (10s ticker, calls `CheckHealth`, emits `cluster:{connCtx}:health` via `m.emitEvent`); add `fetchAndStorePermissions(ctx, contextName string, conn *Connection)` method (calls `FetchPermissions`, stores result in `conn.Permissions`, emits `cluster:{contextName}:permissions` event); wire both as goroutines in `Connect()` after `go m.healthMonitor`

4. **`internal/config/config.go`** — add `ReadOnly bool \`json:"readOnly,omitempty"\`` field to `Config` struct

5. **`internal/services/app.go`** — add `SetReadOnly(ctx context.Context, enabled bool) error` (updates `a.config.ReadOnly`, calls `config.Save()`); add `GetClusterHealth(ctx context.Context, connCtx string) (cluster.ClusterHealth, error)` (calls `cluster.CheckHealth` once for initial page load — live updates come via event)

6. **`frontend/src/lib/stores/cluster.svelte.ts`** — add `permissions = $state<Record<string, PermissionSet>>({})` and `isReadOnly = $state<boolean>(false)`; add `canMutate(): boolean` method (`!this.isReadOnly && (this.permissions[this.activeContext ?? '']?.canMutate() ?? true)`); subscribe to `cluster:{ctx}:permissions` event in `onContextConnected`; load `isReadOnly` from config on init via `AppService.GetConfig()`; wire `SetReadOnly` call from the header toggle

7. **`frontend/src/lib/registry/index.ts`** — add `availableGVRs = new Set<string>()` field; add `setAvailableGVRs(gvrs: string[]): void`; add `isGVRAvailable(gvr: string): boolean` (returns `true` if set is empty — discovery not yet complete — or if gvr is in the set)

8. **`frontend/src/lib/components/Sidebar.svelte`** — call `descriptorRegistry.setAvailableGVRs(...)` inside `handleDiscovery()`; add `disabled` and `tooltip` props to sidebar GVR entries based on `!descriptorRegistry.isGVRAvailable(gvr)` and `!clusterStore.canMutate()`; navigation click handler is a no-op when `disabled`

9. **`frontend/src/routes/c/[ctx]/health/+page.svelte`** — new route; subscribes to `cluster:{ctx}:health` Wails event; renders API server status (livez/readyz/healthz as colored badges), component statuses table (Unknown shown as grey "not exposed"), and node gauge (total/ready/not-ready; shows permission warning when `NodeSummary.PermissionDenied`)

10. **`frontend/src/lib/components/HealthSummaryWidget.svelte`** — compact OK/degraded badge for the ClusterOverview page that links to `/c/:ctx/health`; derives status from the last `cluster:{ctx}:health` event

11. **Header read-only toggle** — toggle switch in the app header (top-right); bound to `clusterStore.isReadOnly`; calls `AppService.SetReadOnly` on change; visually distinct (lock icon, muted label)

12. **Mutating action gates** — all delete, scale, edit YAML, apply, and drain action buttons gain `disabled={!clusterStore.canMutate()}` and `title="Read-only mode"` when disabled; `WasmRuntime.CallCommand` in `internal/services/plugin.go` checks `a.config.ReadOnly` before invoking and returns an error if set

## Tests

**Unit — Go**
- `internal/cluster/permissions_test.go`: `FetchPermissions` returns `Inferred: false` with rules when `SelfSubjectRulesReview` succeeds; returns `Inferred: true` on 403; `CanMutate()` returns false when rules contain only `get`/`list`/`watch`; returns true when rules contain `delete`
- `internal/cluster/health_test.go`: `CheckHealth` marks `NodeSummary.PermissionDenied = true` when node LIST returns 403; componentstatuses 404 response sets all components to `HealthUnknown`; empty componentstatuses list also sets `HealthUnknown`
- `internal/config/config_test.go`: `ReadOnly` field round-trips through JSON marshal/unmarshal

**Manual verification**
- Connect to a cluster → health page auto-populates within 1s, then refreshes every 10s
- Toggle read-only in header → delete buttons gray out immediately; toggle off → buttons re-enable; reopen app → toggle remains in same position
- Connect to a cluster where `SelfSubjectRulesReview` is blocked (simulate by connecting with a restricted kubeconfig) → app assumes full access, no error shown
- Connect to k3s or a cluster that doesn't expose componentstatuses → component section shows "not exposed by this cluster" for all components, no error toast
- Sidebar GVR entry for an unavailable API group → grayed out, cursor not-allowed, tooltip explains reason, click does nothing

## Acceptance Criteria

- [ ] `/c/:ctx/health` renders API server livez/readyz/healthz badges and refreshes every 10s without manual interaction
- [ ] componentstatuses 404 or empty response shows "not exposed by this cluster" — not an error state
- [ ] Read-only toggle in header persists across app restarts (saved to `config.json`)
- [ ] `canMutate()` returns false when either config `readOnly: true` or when `SelfSubjectRulesReview` reports no mutating verbs
- [ ] All delete/scale/edit/apply/drain buttons are `disabled` and show tooltip when `!canMutate()`
- [ ] Plugin `CallCommand` returns error (without invoking Wasm) when `config.ReadOnly` is true
- [ ] Sidebar entries for GVRs whose API group is absent from `discovery:{ctx}:resources` are visually disabled with tooltip
- [ ] Port-forward actions are unaffected by read-only mode
- [ ] Node gauge shows "Insufficient permissions" text when LIST nodes is denied, not zeros

## Definition of Done

A developer connects to any cluster and sees the health page populate within one second showing API server probe results, a component status table (or a clear "not exposed" state for each component on distros like k3s), and a node ready/total gauge. The page refreshes live every 10 seconds. Enabling the read-only toggle in the header immediately disables all mutating action buttons app-wide, the toggle state survives an app restart, and plugin commands that would mutate state return a clear error. Any sidebar entry whose API group isn't available on the connected cluster is visibly grayed out with a tooltip explaining why.

## Known Gotchas

- **The trap**: `/healthz`, `/livez`, `/readyz` responses are plain text `ok` or a list of check names, not JSON. The `Do(ctx).Raw()` call returns `[]byte` — don't attempt JSON unmarshal.
  **Why**: These are Kubernetes diagnostic endpoints, not API endpoints.
  **What to do instead**: `strings.TrimSpace(string(body)) == "ok"` for the simple case; for `/livez?verbose` the response is multi-line named checks.

- **The trap**: `SelfSubjectRulesReview` is namespace-scoped. Calling it with `kube-system` namespace covers most verbs but misses grants specific to other namespaces.
  **Why**: The API is designed to answer "what can I do in namespace X?" — there's no cluster-wide equivalent.
  **What to do instead**: Accept this limitation. The permission check is coarse (read vs. write) not fine-grained. Document in a code comment.

- **The trap**: `isGVRAvailable()` returning `false` before discovery completes would flash the entire sidebar as disabled on every connect.
  **Why**: `emitDiscovery` runs async; the first event arrives ~100-500ms after `Connect()` returns.
  **What to do instead**: `availableGVRs` starts empty; `isGVRAvailable()` returns `true` when the set is empty (pre-discovery = optimistic). Only after the first discovery event does the set become non-empty and start gating entries.

- **The trap**: `componentstatuses` returns items with a `conditions` array where each condition has `type: "Healthy"` and `status: "True"/"False"`. Some clusters return zero items (not a 404) when the API exists but nothing is registered.
  **Why**: The API is deprecated and vendors don't populate it consistently.
  **What to do instead**: Treat both 404 and empty-item-list as `HealthUnknown` with message "not exposed by this cluster". Only trust a non-empty response.

- **The trap**: `AppService.SetReadOnly` must regenerate Wails bindings after being added, and the method signature must not use types that cause model name collisions.
  **Why**: Wails v3 alpha.74 generates `.js` bindings; any exported method that takes/returns a struct it hasn't seen can trigger duplicate identifier issues in `index.js`.
  **What to do instead**: `SetReadOnly(ctx, bool) error` and `GetClusterHealth(ctx, string) (ClusterHealth, error)` are safe — `bool`/`string`/`error` are primitives. Run `wails3 generate bindings` after adding methods and check `frontend/bindings/index.js` for duplicates before moving on.
