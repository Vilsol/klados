# Native Helm Support — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add native Helm v4 support to klados: list Helm releases as a first-class virtual GVR (`helm.v1.releases`) with live watch, secret-aware detail panels, and lifecycle verbs (rollback, uninstall, test).

**Architecture:** Releases are Kubernetes Secrets (`type=helm.sh/release.v1`). A new `internal/helm/` package decodes them, aggregates per-revision Secrets into one virtual row per release, and powers a Wails-bound `HelmService` that exposes verbs and lazy detail fetches. The existing `ResourceEngine` and `WatchManager` gain a `VirtualBackend`/`VirtualWatchSource` extension point so the virtual GVR flows through the standard list/watch/detail pipeline.

**Tech Stack:** Go 1.26, `helm.sh/helm/v4`, `k8s.io/* v0.36.1` (Phase 0 bumps both), Wails v3 alpha.74, Svelte 5 runes, Tailwind v4, slox logging. Tests use `testza` (Go) and `vitest` + `@testing-library/svelte` (frontend).

**Spec reference:** `docs/superpowers/specs/2026-05-19-helm-support-design.md` (jj change `ustkutxt`).

**Testing strategy:**
- Unit tests per Go file as code is written (testza, no CGO needed for `internal/helm`).
- Integration tests under `-tags integration` against kind/minikube — Phase 6 only.
- Frontend component tests via vitest + @testing-library/svelte; mock `@wailsio/runtime` and `HelmService` bindings.
- Storybook stories for each new panel.
- Manual QA checklist (in spec) on a real cluster with Helm releases (kind + bitnami/nginx, kube-prometheus-stack) before declaring v1 done.

**Granularity note:** Coarse tasks aligned with the spec's Phase 0–6. Each task is a coherent slice that can be committed as one jj change. Inside a task, run tests, type-check (`cd frontend && pnpm check`), and commit. TDD is implicit — write tests alongside code, not enumerated as separate steps.

**Commit pattern:** Klados uses Jujutsu. Each task ends with `jj describe -m "..."` then `jj new` to start the next clean commit.

---

## File touchlist

### Phase 0 (dep upgrade)
- **Modify:** `go.mod`, `go.sum`, `mise.toml`
- **Modify:** any Go file that breaks under `k8s.io/* v0.36.1` (audit-driven; expected small)

### Phase 1 (internal/helm)
- **Create:** `internal/helm/release.go`, `release_test.go`
- **Create:** `internal/helm/continuation.go`, `continuation_test.go`
- **Create:** `internal/helm/aggregator.go`, `aggregator_test.go`
- **Create:** `internal/helm/client.go`, `client_test.go`
- **Create:** `internal/helm/list.go`, `list_test.go`
- **Create:** `internal/helm/history.go`, `history_test.go`
- **Create:** `internal/helm/diff.go`, `diff_test.go`
- **Create:** `internal/helm/owned.go`, `owned_test.go`
- **Create:** `internal/helm/enricher.go`, `enricher_test.go`
- **Create:** `internal/helm/actions.go`, `actions_test.go`
- **Create:** `internal/helm/testdata/` fixtures (six Secret YAML files + one chart)

### Phase 2 (engine + watcher plumbing)
- **Modify:** `internal/resource/descriptor.go` — add `IsVirtual bool`
- **Modify:** `internal/resource/engine.go` — `VirtualBackend` interface + dispatch in `List`, `Get`
- **Modify:** `internal/resource/builtin.go` — register `helm.v1.releases` Descriptor + helm enricher
- **Modify:** `internal/watcher/manager.go` — `fieldSelector` parameter on `StartWatch`, `RegisterVirtual` + `VirtualWatchSource`
- **Modify:** `internal/watcher/manager_test.go`, `internal/resource/engine_test.go`, `internal/resource/descriptor_test.go`

### Phase 3 (HelmService)
- **Create:** `internal/services/helm.go`, `internal/services/helm_test.go`
- **Modify:** `internal/services/cluster.go` — invoke `HelmService.OnClusterConnected` on connect (mirror `volumeBrowserSvc` precedent)
- **Modify:** `internal/services/resource.go` — wire helm virtual backend + watch source into engine and watcher at startup
- **Modify:** `internal/cluster/permissions.go` — add `CheckAccess(ctx, ns, verb, gvr)` SAR helper (new function; keep existing `CanMutate` as-is)
- **Modify:** `internal/cluster/permissions_test.go`
- **Modify:** `main.go` — register `HelmService` with Wails app builder
- **Regenerate:** `frontend/bindings/.../HelmService.js` and friends via `wails3 generate bindings`

### Phase 4 (frontend list + sidebar)
- **Modify:** `frontend/src/lib/registry/index.ts` — handle synthetic Helm group label
- **Modify:** `frontend/src/lib/components/Sidebar.svelte` — render Helm group entry
- **Modify:** `frontend/src/lib/components/ResourceList.svelte` — stuck-state badge for `pending-*` / `uninstalling` rows
- **Modify:** `frontend/src/lib/components/BulkActionBar.svelte` — register "Uninstall" for `helm.v1.releases`
- **Create:** `frontend/src/lib/components/__tests__/HelmReleaseRow.svelte.test.ts` (or extend existing ResourceList tests)

### Phase 5 (detail drawer panels)
- **Create:** `frontend/src/lib/helm/secret-masking.ts` — shared secret-masking helper (regex + walker for YAML/JSON objects)
- **Create:** `frontend/src/lib/components/panels/HelmOverviewPanel.svelte`
- **Create:** `frontend/src/lib/components/panels/HelmValuesPanel.svelte`
- **Create:** `frontend/src/lib/components/panels/HelmManifestPanel.svelte`
- **Create:** `frontend/src/lib/components/panels/HelmHistoryPanel.svelte`
- **Create:** `frontend/src/lib/components/panels/HelmNotesPanel.svelte`
- **Create:** `frontend/src/lib/components/panels/HelmResourcesPanel.svelte`
- **Create:** `frontend/src/lib/components/panels/HelmHooksPanel.svelte`
- **Modify:** `frontend/src/lib/components/ResourceDetail.svelte` — register Helm-specific tabs
- **Create:** Storybook stories under `apps/docs/src/stories/Helm*Story.svelte` + `Helm*.stories.ts`
- **Create:** vitest specs under `frontend/src/lib/components/panels/__tests__/`

### Phase 6 (verb UI + integration tests)
- **Create:** `frontend/src/lib/components/HelmRollbackDialog.svelte`
- **Create:** `frontend/src/lib/components/HelmUninstallDialog.svelte`
- **Create:** `frontend/src/lib/components/HelmTestDialog.svelte`
- **Create:** `frontend/src/lib/components/HelmForceDeleteDialog.svelte`
- **Modify:** `frontend/src/lib/components/ResourceList.svelte` and `HelmOverviewPanel.svelte` to invoke the dialogs
- **Create:** `internal/helm/integration_test.go` + `internal/helm/testdata/charts/sample/` chart

---

## Task 1 — Phase 0: Dependency stack upgrade

Bump Go to 1.26 and `k8s.io/*` to v0.36.1 in isolation, before any Helm code lands. Single bisectable commit. Helm itself is NOT added in this task.

**Files:**
- Modify: `go.mod`, `go.sum`, `mise.toml`
- Modify: any Go file that breaks (audit-driven)

- [ ] **Bump tool versions**

In `mise.toml`, set:

```toml
[tools]
go = "1.26"
# leave wails3, go-jsonschema, node, tinygo, pnpm unchanged
```

Run `mise install` to pull Go 1.26.

In `go.mod`, change the `go 1.25` directive to `go 1.26`. Replace every `k8s.io/* v0.35.3` line with `v0.36.1`:

```
k8s.io/api v0.36.1
k8s.io/apiextensions-apiserver v0.36.1
k8s.io/apimachinery v0.36.1
k8s.io/client-go v0.36.1
k8s.io/kubectl v0.36.1
k8s.io/metrics v0.36.1
```

Run `go mod tidy`.

- [ ] **Audit and fix compile breaks**

Run `go build ./...`. For every break, fix in-place. Known historical hotspots between minor k8s versions (verify against current 0.35→0.36 release notes at https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.36.md and `client-go` v0.36 release notes):
- `watch.Interface` and `cache.NewListWatchFromClient` signatures (used in `internal/watcher/manager.go`)
- Discovery API (used in `internal/cluster/discovery_metadata.go`, `internal/cluster/manager.go`)
- RBAC `*ResourceRule` types (used in `internal/cluster/permissions.go`)
- Dynamic client interface (used across `internal/resource/`, `internal/portforward/`, `internal/exec/`, `internal/logs/`)

Each fix is mechanical; do not change behavior.

- [ ] **Run full test suite**

```bash
go test ./internal/... -v
cd frontend && pnpm check && pnpm test
```

All existing tests must pass. Frontend bindings do not change in this task.

- [ ] **Smoke-test on a real cluster**

Run `task dev`. Connect to a kind cluster, list a few resources, watch a Pod, open exec, confirm logs stream. No regressions.

- [ ] **Commit**

```bash
jj describe -m "chore(deps): bump Go 1.26 and k8s.io/* to v0.36.1 (Helm v4 prerequisite)"
jj new
```

---

## Task 2 — Phase 1: `internal/helm` package foundation

Pure Go package, no Wails or frontend coupling. All logic for decoding releases, aggregating per-revision Secrets, building per-cluster `action.Configuration`, and wrapping Helm verbs. Tested end-to-end with fixtures.

**Files:**
- Create: every file under `internal/helm/`

- [ ] **Add Helm v4 dependency**

In `go.mod`, add:

```
require helm.sh/helm/v4 v4.2.x  // pick latest v4.2.* stable
```

Run `go mod tidy`. Confirm with `go mod why helm.sh/helm/v4` that nothing forces a further k8s.io bump.

- [ ] **`release.go` — decoder with size cap + sentinel**

```go
// internal/helm/release.go
package helm

import (
    "bytes"
    "compress/gzip"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"

    "helm.sh/helm/v4/pkg/release"
    corev1 "k8s.io/api/core/v1"
)

const (
    releaseSecretType = "helm.sh/release.v1"
    maxDecompressed   = 50 << 20 // 50 MiB hard cap (zip-bomb guard)
)

// DecodeRelease decodes the `release` data field of a Helm release Secret.
// Returns ErrWrongType if the Secret type doesn't match the v1 sentinel.
// Returns ErrTooLarge if the decompressed payload exceeds maxDecompressed.
func DecodeRelease(s *corev1.Secret) (*release.Release, error) {
    if s.Type != releaseSecretType {
        return nil, fmt.Errorf("%w: %s", ErrWrongType, s.Type)
    }
    raw, ok := s.Data["release"]
    if !ok {
        return nil, ErrMissingPayload
    }
    decoded, err := base64.StdEncoding.DecodeString(string(raw))
    if err != nil {
        return nil, fmt.Errorf("base64: %w", err)
    }
    if len(decoded) >= 3 && decoded[0] == 0x1f && decoded[1] == 0x8b {
        gz, err := gzip.NewReader(bytes.NewReader(decoded))
        if err != nil {
            return nil, fmt.Errorf("gzip: %w", err)
        }
        defer gz.Close()
        decoded, err = io.ReadAll(io.LimitReader(gz, maxDecompressed+1))
        if err != nil {
            return nil, fmt.Errorf("gunzip: %w", err)
        }
        if len(decoded) > maxDecompressed {
            return nil, ErrTooLarge
        }
    }
    var rel release.Release
    if err := json.Unmarshal(decoded, &rel); err != nil {
        return nil, fmt.Errorf("json: %w", err)
    }
    return &rel, nil
}

var (
    ErrWrongType       = fmt.Errorf("helm: secret type mismatch")
    ErrMissingPayload  = fmt.Errorf("helm: secret missing release data")
    ErrTooLarge        = fmt.Errorf("helm: decompressed payload exceeds %d bytes", maxDecompressed)
)
```

Tests cover: good fixture round-trip, missing data, wrong type sentinel, malformed base64, malformed gzip, payload > 50 MiB (synthesize a gzipped payload with `bytes.Repeat`), malformed JSON.

- [ ] **`continuation.go` — multi-Secret chunk reassembly**

Helm v4 splits releases > 1 MB into Secrets named `sh.helm.release.v1.<release>.v<rev>.<chunk>`. Pattern detection: chunk suffix `^\.[0-9]+$` after the revision component. Reassembly: collect all chunks for `(release, rev)`, sort by chunk index, concatenate the `data["release"]` base64 strings **before** base64-decode. Single-chunk releases (no trailing `.N`) bypass.

Tests: 1-chunk pass-through, 2-chunk reassembly, 3-chunk reassembly out-of-order, missing chunk → typed `ErrMissingChunk` error.

- [ ] **`aggregator.go` — snapshot + delta + reconnect**

```go
type Aggregator struct {
    mu       sync.Mutex
    releases map[string]map[string]map[int]*release.Release // ns → name → rev → rel
}

// CollapseSnapshot consumes a full list of Secrets (with continuation already
// reassembled), populates the map, and returns one map[string]any per latest
// revision per release.
func (a *Aggregator) CollapseSnapshot(secrets []corev1.Secret) ([]map[string]any, error)

// ApplyDelta updates the map from a single watch event and returns the
// emitted virtual event (or nil for no-op).
func (a *Aggregator) ApplyDelta(ev watcher.Event) (*VirtualEvent, error)

// Reset clears state for a given namespace (or all) — used on watch reconnect.
func (a *Aggregator) Reset(namespace string)
```

Latest-revision selection: max revision number; deployed-at as tiebreak (rare).

Tests:
- Snapshot reduction: 10 Secrets across 4 releases → 4 virtual rows, latest revision wins.
- Delta: ADDED new revision → MODIFIED virtual (or ADDED if new release).
- Delta: DELETED non-latest revision → no event.
- Delta: DELETED latest revision but other revisions remain → MODIFIED with new latest.
- Delta: DELETED last remaining revision → DELETED virtual.
- Tiebreak: two revisions same rev number, deployed-at distinguishes.
- Reset clears submap atomically.
- **Stress test**: 10k synthetic Secrets across 1000 releases. Assert `CollapseSnapshot` completes in < 500 ms on a normal dev machine. Memory growth bounded (use `runtime.ReadMemStats` before/after, assert delta < 100 MiB).

- [ ] **`client.go` — `action.Configuration` cache**

```go
type ClientCache struct {
    mu    sync.Mutex
    items map[cacheKey]*action.Configuration
}

type cacheKey struct {
    contextName string
    namespace   string
}

// Get returns a cached or freshly-built Configuration for (ctx, ns).
// Uses Helm v4's NewConfiguration builder + ConfigurationSetLogger to route
// SDK logging through slox.
func (c *ClientCache) Get(ctx context.Context, contextName, namespace string,
    restConfig *rest.Config, slogHandler slog.Handler) (*action.Configuration, error)

// Evict drops all entries for a context (called on cluster disconnect).
func (c *ClientCache) Evict(contextName string)
```

Use `action.ConfigurationSetLogger(slogHandler)` — slox provides a slog handler via `slox.Handler(ctx)` or equivalent (check `internal/logging/setup.go`).

Tests: cache hit returns same pointer, cache miss builds new, `Evict(ctx)` drops only that context's entries.

- [ ] **`list.go` — `ListReleases` implementing `VirtualBackend.List`**

```go
func (b *Backend) List(ctx context.Context, namespace string) ([]map[string]any, string, error) {
    secrets, rv, err := b.listSecrets(ctx, namespace, "type="+releaseSecretType)
    if err != nil {
        return nil, "", err
    }
    reassembled, err := ReassembleContinuation(secrets)
    if err != nil {
        return nil, "", err
    }
    items, err := b.aggregator.CollapseSnapshot(reassembled)
    if err != nil {
        return nil, "", err
    }
    return items, rv, nil
}
```

`b.listSecrets` calls `ResourceEngine.List` with a `fieldSelector` option (this requires Phase 2's `fieldSelector` plumbing — for Phase 1, write a small interface stub `secretLister` and inject it; tests use a fake).

Synthetic resourceVersion is the underlying secret list's resourceVersion — passed through unchanged.

Tests: empty list, populated list, all-namespaces (`namespace=""`), continuation chunks present, fieldSelector built correctly (assert via fake).

- [ ] **`history.go` — `GetHistory`**

Lists all Secrets matching `name=sh.helm.release.v1.<release>` (label `name=<release>` is set by Helm; also fallback to name prefix match), decodes each, returns sorted by revision descending. Skip malformed Secrets with a warn log.

Returned type:

```go
type Revision struct {
    Number      int
    Status      string
    ChartName   string
    ChartVersion string
    AppVersion  string
    Description string
    DeployedAt  time.Time
}
```

Tests: descending sort, malformed skipped, empty release name returns empty.

- [ ] **`diff.go` — `RevisionDiff` for values/computed/manifest**

Uses `github.com/google/go-cmp/cmp` or stdlib + custom unified diff (existing klados utility may already exist — check `internal/` for diff helpers). Returns three strings: `Values`, `ComputedValues`, `Manifest`. Equal revisions → empty strings. Apply secret-masking on Go side before diffing values (key-name regex match — same logic as the frontend's `secret-masking.ts`; extract shared constant).

Tests: equal-revisions empty, different-values produces unified diff, missing revision → typed error, masked keys appear as `••••••••` in diff output.

- [ ] **`owned.go` — manifest-primary, label-fallback**

```go
type OwnedRef struct {
    APIVersion       string
    Kind             string
    GVR              string  // dot-separated, klados convention
    Namespace        string
    Name             string
    ResourcePolicy   string  // "" or "keep"
}

func (b *Backend) GetOwnedResources(ctx context.Context, namespace, releaseName string) ([]OwnedRef, error)
```

Steps inside the function:

1. `GetHistory` to find latest revision.
2. Parse its `rel.Manifest` as multi-document YAML. For each non-empty doc, extract `apiVersion`, `kind`, `metadata.namespace` (fallback to release namespace), `metadata.name`, and the `helm.sh/resource-policy` annotation if present.
3. Build a unique set keyed by `(gvk, ns, name)`.
4. Issue one `engine.Get(gvr, ns, name)` per item to verify existence; missing → still include with `Exists: false`.
5. **Label-fallback pass** across the bounded GVR set listed in the spec (Deployments, StatefulSets, DaemonSets, ReplicaSets, Services, ConfigMaps, Secrets, Ingresses, PVCs, ServiceAccounts, Jobs, CronJobs, Roles, RoleBindings, NetworkPolicies, HPAs) with label selector `app.kubernetes.io/managed-by=Helm,meta.helm.sh/release-name=<release>`. Merge results, dedup against primary set.

Tests:
- Multi-doc YAML extraction handles `---`, comments, empty docs.
- Cross-namespace resources (manifest has `metadata.namespace: kube-system` while release is in `apps`) appear.
- `helm.sh/resource-policy: keep` annotation detected.
- Label-fallback finds a resource present in cluster but missing from the manifest (e.g. a hook-installed Job).
- "Scan all GVRs" opt-in path returns the same primary set plus any extra label matches across the full GVR list (test via stub).

- [ ] **`enricher.go` — `resource.Enricher`**

Implements:

```go
func (e *HelmEnricher) Enrich(contextName string, u *unstructured.Unstructured) error
```

Reads the embedded release metadata that `CollapseSnapshot` stashed in the unstructured (use `unstructured.NestedString` etc.) and injects:

- `status.statusDisplay` (e.g. "Deployed", "Failed", "Pending Upgrade", "Superseded", "Uninstalled")
- `status.revisionDisplay` (e.g. "rev 4")
- `status.chartDisplay` (e.g. "nginx-15.4.4")
- `status.appVersion`
- `status.lastDeployedDisplay` (RFC3339 — the frontend `age` renderer handles formatting)
- `status.ownedResourceCount` (computed lazily; populate from `metadata` if pre-counted, else leave 0)

Tests: each field populated from a fixture; missing metadata → safe defaults (empty string, no panic).

- [ ] **`actions.go` — verb wrappers with per-(ctx, ns, release) mutex**

```go
type Actions struct {
    cache *ClientCache
    mu    sync.Mutex
    locks map[lockKey]*sync.Mutex
}

type lockKey struct{ ctx, ns, release string }

func (a *Actions) Rollback(ctx context.Context, contextName, ns, release string, rev int, opts RollbackOpts) error
func (a *Actions) Uninstall(ctx context.Context, contextName, ns, release string, opts UninstallOpts) error
func (a *Actions) Test(ctx context.Context, contextName, ns, release string, opts TestOpts) (TestResult, error)
func (a *Actions) ForceDeleteReleaseSecret(ctx context.Context, contextName, ns, release string, rev int) error
```

The mutex map is itself protected by `a.mu` for safe lazy-init of per-release locks. Acquiring a per-release lock that's already held returns `ErrOperationInProgress` immediately (no blocking).

`Rollback`: builds `action.NewRollback(cfg)`, applies opts (`Wait`, `Timeout`, `DisableHooks` (= --no-hooks), `Version`), calls `.Run(release)`.

`Uninstall`: builds `action.NewUninstall(cfg)`, applies opts (`KeepHistory`, `DisableHooks`, `Wait`, `Timeout`), calls `.Run(release)`.

`Test`: builds `action.NewReleaseTesting(cfg)`, applies opts (`Timeout`, `Filters`), calls `.Run(name)`. **Logs are NOT streaming** — after `.Run` returns, call `client.GetPodLogs(buf, rel)` to bulk-read into a `bytes.Buffer`, return `TestResult{Phase, Logs}`.

`ForceDeleteReleaseSecret`: deletes the underlying Secret directly via the dynamic client. Used for stuck `pending-*` recovery.

Tests: each verb dispatches to a mocked `action.Configuration` (use Helm's own test fakes if available; otherwise write a tiny interface for the action constructors). Concurrent-invocation test: spawn two goroutines calling `Rollback` on the same release; second returns `ErrOperationInProgress` immediately.

- [ ] **Fixtures**

Build fixtures by running `helm install/upgrade/rollback` against kind and copying the resulting Secrets, or generate programmatically with `pkg/storage/driver.Secrets.Create`. Place under `internal/helm/testdata/`:

- `release-deployed.secret.yaml`
- `release-failed.secret.yaml`
- `release-superseded.secret.yaml`
- `release-stuck-pending-upgrade.secret.yaml`
- `release-multi-revision/` (5 revisions × 2 releases)
- `release-large-continuation/` (3-chunk release)
- `release-with-secret-values.secret.yaml` (values containing `password`, `apiToken`)

Include a small `helpers_test.go` to load fixtures via `os.ReadFile` + yaml unmarshal.

- [ ] **Run unit tests**

```bash
go test ./internal/helm/... -v -race
```

All tests pass, including the stress test (10k snapshot).

- [ ] **Commit**

```bash
jj describe -m "feat(helm): internal/helm foundation — decoder, aggregator, actions, enricher"
jj new
```

---

## Task 3 — Phase 2: Engine + watcher plumbing for virtual GVRs

Extend `ResourceEngine` and `WatchManager` with the smallest possible surface for virtual GVRs. Register `helm.v1.releases` Descriptor + helm backend + watch source. After this task, `ResourceService.ListResources("helm.v1.releases", ns)` works end-to-end against a real cluster.

**Files:**
- Modify: `internal/resource/descriptor.go`, `engine.go`, `builtin.go`, related tests
- Modify: `internal/watcher/manager.go`, `manager_test.go`

- [ ] **Add `IsVirtual` to `Descriptor`**

Add the field to the Descriptor struct in `internal/resource/descriptor.go`. Update any descriptor constructors. Update `descriptor_test.go` to cover serialization.

- [ ] **Define `VirtualBackend` and dispatch in Engine**

```go
// internal/resource/engine.go
type VirtualBackend interface {
    List(ctx context.Context, namespace string) ([]map[string]any, string, error)
    Get(ctx context.Context, namespace, name string) (map[string]any, error)
}

func (e *Engine) RegisterVirtual(gvr string, backend VirtualBackend) {
    e.virtuals[gvr] = backend
}
```

In `Engine.List`:

```go
if desc := e.descriptors.Get(gvr); desc != nil && desc.IsVirtual {
    backend, ok := e.virtuals[gvr]
    if !ok {
        return nil, "", fmt.Errorf("virtual backend %q not registered", gvr)
    }
    return backend.List(ctx, namespace)
}
// existing dynamic.Interface path follows
```

Mirror in `Engine.Get`. Apply (and call) the existing enricher chain to the virtual results — virtual backends do NOT bypass enrichers.

Engine tests: dispatch to a fake virtual backend; resourceVersion passthrough; multiple virtuals don't conflict.

- [ ] **Watcher: `fieldSelector` + virtual dispatch**

In `internal/watcher/manager.go`:

1. Add an optional `FieldSelector string` to whatever `StartWatch` accepts (or extend its options struct). Plumb to `metav1.ListOptions.FieldSelector` in the watch call. Existing callers pass `""` — no behavior change.

2. Add:

```go
type VirtualWatchSource interface {
    Watch(ctx context.Context, namespace, resourceVersion string) (<-chan Event, func(), error)
}

func (m *Manager) RegisterVirtual(gvr string, src VirtualWatchSource)
```

3. In `StartWatch`, before constructing the dynamic client, dispatch:

```go
if src, ok := m.virtuals[gvr]; ok {
    return src.Watch(ctx, namespace, resourceVersion)
}
```

4. Define `Event` (likely already exists as `WatchEvent`). Add explicit `SyncStart` and `SyncEnd` event-type constants — these brace a snapshot replay on reconnect. The frontend `ResourceStore` will consume these in Phase 4.

Watcher tests: `fieldSelector` plumbed correctly into the underlying watch; existing call sites with empty selector unchanged; virtual dispatch returns the registered source's events.

- [ ] **Register `helm.v1.releases` Descriptor and backend**

In `internal/resource/builtin.go`, add a Descriptor:

```go
{
    Group:      "helm",
    Version:    "v1",
    Resource:   "releases",
    Namespaced: true,
    IsVirtual:  true,
    DisplayName: "Helm Releases",
    GroupLabel: "Helm", // synthetic sidebar group
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: "text"},
        {Name: "Namespace", Expr: "metadata.namespace", RenderType: "text"},
        {Name: "Status", Expr: "status.statusDisplay", RenderType: "badge"},
        {Name: "Revision", Expr: "status.revisionDisplay", RenderType: "text"},
        {Name: "Chart", Expr: "status.chartDisplay", RenderType: "text"},
        {Name: "App Version", Expr: "status.appVersion", RenderType: "text"},
        {Name: "Last Deployed", Expr: "status.lastDeployedDisplay", RenderType: "age"},
    },
    DetailTabs: []string{"helm-overview", "helm-values", "helm-manifest",
        "helm-history", "helm-notes", "helm-resources", "helm-hooks"},
}
```

Register `helm.Backend` (the struct from Phase 1) as both a `VirtualBackend` on the engine and a `VirtualWatchSource` on the watcher. Wiring lives in Phase 3's service-startup change — for this task, expose constructors that take an injected `secretLister` so the helm package compiles + tests pass without service wiring.

Also register `helm.Enricher` with the existing `EnricherRegistry` for GVR `helm.v1.releases`.

- [ ] **Discovery / availability flag**

Add a field to the serialized descriptor (the one `GetDescriptors()` returns to the frontend): `Available bool`. Set unconditionally to `true` for now; Phase 3 will update it from the `OnClusterConnected` probe.

- [ ] **Run tests**

```bash
go test ./internal/resource/... ./internal/watcher/... ./internal/helm/... -v
```

- [ ] **Commit**

```bash
jj describe -m "feat(resource,watcher): virtual GVR dispatch + helm.v1.releases descriptor"
jj new
```

---

## Task 4 — Phase 3: `HelmService` Wails service + permissions + bindings

Expose verbs and lazy detail-tab fetches as Wails RPCs. Wire the helm backend into engine + watcher at service startup. Add per-namespace SAR-based permission gating.

**Files:**
- Create: `internal/services/helm.go`, `internal/services/helm_test.go`
- Modify: `internal/services/cluster.go` (call `OnClusterConnected`), `internal/services/resource.go` (wire helm backend in `ServiceStartup`)
- Modify: `internal/cluster/permissions.go` (add `CheckAccess` SAR helper)
- Modify: `main.go` (register `HelmService`)
- Regenerate: Wails bindings

- [ ] **`CheckAccess` SAR helper**

```go
// internal/cluster/permissions.go
func (m *Manager) CheckAccess(ctx context.Context, contextName, namespace, verb, group, resource string) (allowed bool, err error)
```

Implementation: build a `SelfSubjectAccessReview` with `ResourceAttributes{Namespace, Verb, Group, Resource}`, call `AuthorizationV1().SelfSubjectAccessReviews().Create(...)`. Return `status.Allowed`.

Keep existing `CanMutate()` untouched — it's used elsewhere.

Tests: stub the auth client; happy path returns true; denied returns false; error propagates.

- [ ] **`HelmService` skeleton**

```go
// internal/services/helm.go
type HelmService struct {
    ctx        context.Context
    cluster    *cluster.Manager
    appSvc     *AppService
    backend    *helm.Backend
    actions    *helm.Actions
    clients    *helm.ClientCache
    available  map[string]bool // contextName → has helm releases
    mu         sync.RWMutex
}

func (s *HelmService) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error
func (s *HelmService) ServiceShutdown() error
func (s *HelmService) OnClusterConnected(ctx context.Context, contextName string)
```

`OnClusterConnected` probes for any Secret with `type=helm.sh/release.v1` (list with `limit=1` across all namespaces or visible ones). Sets `s.available[contextName] = true` on success. Updates the descriptor's `Available` flag via the registry — needs a setter on `descriptor.Registry`; add one.

`ServiceStartup` constructs `helm.Backend`, `helm.Actions`, `helm.ClientCache`, registers backend with engine and watcher.

- [ ] **All RPCs**

Each RPC is a thin pass-through to `helm.Actions` or `helm.Backend`. Method shapes from the spec — keep them stable so bindings stay clean.

For each verb (`Rollback`, `Uninstall`, `Test`, `ForceDeleteReleaseSecret`): call `cluster.CheckAccess` first, return RBAC error if denied. The verb's per-release mutex in `actions.go` handles concurrency.

Add `GetReleasePermissions(ctx, ns, release) (Permissions, error)`:

```go
type Permissions struct {
    CanRollback     bool
    CanUninstall    bool
    CanTest         bool
    CanForceDelete  bool
}
```

Each field reflects one or more `CheckAccess` calls (e.g. `CanUninstall` requires `secrets/delete` + `secrets/update` in ns).

- [ ] **Wire `OnClusterConnected` in `ClusterService.Connect`**

Mirror the existing `volumeBrowserSvc.OnClusterConnected(ctx, name)` invocation in `internal/services/cluster.go` around the existing hook point. The HelmService reference comes from the same DI pattern.

- [ ] **Register service with Wails**

In `main.go`, add `&services.HelmService{}` to the service list passed to `application.NewWithOptions(...)`. Follow whatever pattern the other services use.

- [ ] **Regenerate bindings**

```bash
wails3 generate bindings
```

Commit generated `frontend/bindings/.../HelmService.js` and friends. No hand edits.

- [ ] **Service tests**

`internal/services/helm_test.go` — per RPC: happy path, RBAC-denied (CheckAccess stub returns false), release-not-found, malformed args, concurrent-verb returns `ErrOperationInProgress`. Use a `fakeConnProvider` modeled on the one in `internal/services/resource_test.go`.

- [ ] **Run tests**

```bash
go test ./internal/... -v
cd frontend && pnpm check
```

- [ ] **Commit**

```bash
jj describe -m "feat(services): HelmService with verbs, lazy detail fetches, per-namespace SAR"
jj new
```

---

## Task 5 — Phase 4: Frontend list + sidebar

Render `helm.v1.releases` in the sidebar under a "Helm" group; render the list with default columns and stuck-state badge; wire bulk uninstall.

**Files:**
- Modify: `frontend/src/lib/registry/index.ts` (handle synthetic `GroupLabel`)
- Modify: `frontend/src/lib/components/Sidebar.svelte` (Helm group rendering, availability gating)
- Modify: `frontend/src/lib/components/ResourceList.svelte` (stuck-state badge)
- Modify: `frontend/src/lib/components/BulkActionBar.svelte` (Uninstall action for `helm.v1.releases`)
- Modify: `frontend/src/lib/stores/resource.svelte.ts` (handle `SyncStart`/`SyncEnd` events for atomic snapshot replacement)

- [ ] **Atomic snapshot replacement in `ResourceStore`**

In `frontend/src/lib/stores/resource.svelte.ts`, extend the watch event handler to recognize `type === "SyncStart"` and `type === "SyncEnd"`:

- On `SyncStart`: stash current `items[]` into a hidden buffer, clear `items[]`, set a `syncing` flag.
- On regular `ADDED` events while `syncing`: push into the buffer (not into reactive `items`).
- On `SyncEnd`: atomically swap `items[]` to the buffer's contents. Clear `syncing`.

This closes the ghost-release window for the helm virtual watch. Other GVRs ignore these events (they never fire for them).

Add a unit test for this in `frontend/src/lib/stores/__tests__/resource.svelte.test.ts` (create if needed).

- [ ] **Sidebar Helm group**

In `frontend/src/lib/registry/index.ts`, plumb the descriptor's `GroupLabel` (synthetic group like `"Helm"`) and `Available` flag through to whatever data structure `Sidebar.svelte` reads. Synthetic groups sort at the top.

In `Sidebar.svelte`, filter out descriptors with `Available === false` for the active context.

- [ ] **Stuck-state badge in `ResourceList`**

For rows whose `status.statusDisplay` matches `Pending Install|Pending Upgrade|Pending Rollback|Uninstalling`, render a destructive badge ("Stuck") next to the Status column or as a row-level indicator. Mirror the existing pattern for failed Pods (look in `ResourceList.svelte` for how badge variants are picked).

- [ ] **Bulk uninstall**

In `BulkActionBar.svelte`, register a new bulk action conditionally on `gvr === "helm.v1.releases"`:

```svelte
{#if gvr === "helm.v1.releases"}
  <Button onclick={() => openBulkUninstallDialog(selection)}>Uninstall…</Button>
{/if}
```

Dialog wiring happens in Phase 6; for now stub `openBulkUninstallDialog` to log + show a notification.

- [ ] **Smoke test on a real cluster**

Run `task dev`, connect to a kind cluster with at least one Helm release (`helm install nginx bitnami/nginx`), navigate to `/c/<ctx>/helm.v1.releases`. Confirm:

- Sidebar shows "Helm Releases" under a "Helm" group.
- The release appears in the list with status "Deployed", chart name, revision, app version, last-deployed age.
- `helm install` from CLI in another terminal makes a second row appear without manual refresh.
- `helm uninstall` from CLI removes the row.
- Manually edit the release Secret's status annotation to simulate `pending-upgrade` (or run a failing upgrade) — stuck badge appears.

- [ ] **Frontend tests**

Add or extend tests for sidebar filtering (availability) and stuck-state badge rendering.

```bash
cd frontend && pnpm check && pnpm test
```

- [ ] **Commit**

```bash
jj describe -m "feat(frontend): helm.v1.releases list, sidebar group, stuck-state badge, sync events"
jj new
```

---

## Task 6 — Phase 5: Detail drawer panels

Seven new Svelte panels for the release detail drawer, including the secret-masking helper shared between Values, Manifest, and History (diff view).

**Files:**
- Create: `frontend/src/lib/helm/secret-masking.ts` + test
- Create: 7 `Helm*Panel.svelte` files under `frontend/src/lib/components/panels/`
- Modify: `frontend/src/lib/components/ResourceDetail.svelte` (register Helm tabs)
- Create: storybook stories + component tests

- [ ] **`secret-masking.ts` shared helper**

```ts
// frontend/src/lib/helm/secret-masking.ts
const SECRET_KEY_PATTERN = /password|token|secret|key|cert|credential|apikey|passphrase/i;

export function isSecretKey(key: string): boolean {
  return SECRET_KEY_PATTERN.test(key);
}

/** Walk a parsed YAML/JSON tree and replace string values at secret-keyed paths
 *  with the placeholder. Returns a transform plus a map of (path -> originalValue)
 *  for selective reveal. */
export function maskTree(obj: unknown): {
  masked: unknown;
  reveal: (path: string[]) => string | undefined;
}
```

Tests: nested objects (multi-level), arrays of secrets, mixed types, non-string secret values (number tokens passed through unchanged), unicode keys.

- [ ] **`HelmOverviewPanel.svelte`**

Props: `release` (the virtual unstructured), `permissions` (from `GetReleasePermissions`). Render chips for chart, chartVersion, appVersion, revision, status, lastDeployed. "Source" link if `meta.helm.sh/release-source` annotation present. Stuck-state expander with "Force delete release Secret" button (gated by `permissions.canForceDelete`; wiring to dialog in Phase 6 — stub for now).

- [ ] **`HelmValuesPanel.svelte`**

Props: `ctx`, `ns`, `release`, `revision` (the active revision). Internal state for `mode: "user-supplied" | "computed"` and a revision picker.

Sub-tab switch fires `HelmService.GetValues(ctx, ns, release, computed, revision)`. Apply `maskTree` to the parsed YAML; render via `YAMLEditor` in read-only mode. Each masked field has a "Show" affordance — clicking calls `reveal(path)` and re-renders with the revealed value. A "Copy redacted" button copies the masked YAML; "Copy" copies the live (revealed) YAML.

Persistent banner: *"Values may contain sensitive data."*

- [ ] **`HelmManifestPanel.svelte`**

Same shape as Values but for the rendered manifest. Apply masking to `data` and `stringData` keys of any Secret resource in the multi-doc YAML. Revision picker.

- [ ] **`HelmHistoryPanel.svelte`**

Props: `ctx`, `ns`, `release`, `permissions`. Fetches via `HelmService.GetHistory`. Table columns: Revision, Status, Chart, AppVersion, DeployedAt, Description. Row click previews the revision's values + manifest in a side pane (using the same masking).

"Compare" toggle: picks two revisions, displays a `DiffView` for values and manifest in sub-tabs. `HelmService.DiffRevisions` already masks server-side; the frontend renders the diff as-is.

Past revisions get a "Rollback to this" button — gated by `permissions.canRollback`. Click → opens the Phase 6 dialog (stub for now).

- [ ] **`HelmNotesPanel.svelte`**

Renders `release.notes` (string). If it parses as markdown (heuristic: starts with `#` or contains `**`/`__` markers), render via existing markdown lib (check `package.json` for marked/markdown-it — pick whichever is in tree). Otherwise `<pre>`.

- [ ] **`HelmResourcesPanel.svelte`**

Fetches via `HelmService.GetOwnedResources`. Groups by kind; for each row shows Name, Namespace (always), Status (from existing health helpers), and a link to `/c/:ctx/:gvr/:ns/:name`. Resources with `helm.sh/resource-policy: keep` get a "kept on uninstall" badge.

"Scan all GVRs" button at top for the opt-in fallback path.

- [ ] **`HelmHooksPanel.svelte`**

Fetches via `HelmService.GetHooks`. Table: Hook event (pre-install/post-upgrade/etc.), Kind+Name, Weight, Last Run, Phase. Click → navigate to the underlying resource if still present.

- [ ] **Tab registration in `ResourceDetail.svelte`**

Inspect how Deployments / HPAs register custom tabs today. Mirror that for the seven Helm tabs, keyed off the descriptor's `DetailTabs` array set in Task 3.

- [ ] **Storybook stories**

For each panel, create a `Helm<Name>Story.svelte` + `Helm<Name>.stories.ts` in `apps/docs/src/stories/`. Use fixture data (mock `HelmService` responses). Cover at minimum: empty/loading, populated, secret-masking on/off.

- [ ] **Component tests**

`frontend/src/lib/components/panels/__tests__/Helm*Panel.svelte.test.ts` — at minimum:

- `HelmValuesPanel`: mode switch fires correct RPC; revealing one key doesn't reveal siblings; "Copy redacted" output retains masks.
- `HelmHistoryPanel`: rows in descending order; Compare mode toggle; Rollback button gated by permissions.
- `HelmResourcesPanel`: grouped by kind; namespace shown; "Scan all" triggers the right RPC.
- `HelmOverviewPanel`: stuck-state expander hidden for "Deployed"; visible for "Pending Upgrade".

Mock `HelmService` bindings at the top of each test (`vi.mock('../../../bindings/.../HelmService.js', ...)`).

- [ ] **Run tests + type-check**

```bash
cd frontend && pnpm check && pnpm test
```

- [ ] **Commit**

```bash
jj describe -m "feat(frontend): helm release detail panels with secret-aware masking"
jj new
```

---

## Task 7 — Phase 6: Lifecycle verb UI + integration tests

Wire dialogs for rollback/uninstall/test/force-delete; complete the verb UX; write integration tests against a real cluster; close the QA checklist.

**Files:**
- Create: 4 dialog Svelte components
- Modify: panels and list to invoke dialogs
- Create: `internal/helm/integration_test.go` + sample chart fixtures

- [ ] **`HelmRollbackDialog.svelte`**

Props: `ctx`, `ns`, `release`, `targetRevision`. On open, fetches `HelmService.DiffRevisions(currentRev, targetRevision)`. Renders the diff with `DiffView` (values + manifest sub-tabs). Confirm button calls `HelmService.Rollback`. Show inline progress; disable confirm while in-flight.

Show the residual-RBAC warning: *"You may lack permissions on resources this chart manages; partial state can result on RBAC errors."*

Optional `--no-hooks` checkbox with the explicit inline warning: *"Skipping hooks may leave the cluster in an inconsistent state and is intended for stuck-hook recovery, not routine use."*

- [ ] **`HelmUninstallDialog.svelte`**

Props: `ctx`, `ns`, `releases` (one or many — supports bulk). On open, fetches `HelmService.GetOwnedResources` for each release to enumerate `resource-policy: keep` resources and surface them as a warning list: *"3 resources marked `resource-policy: keep` will remain after uninstall: ..."*

Checkboxes: `KeepHistory`, `--no-hooks` (with same warning copy).

For bulk: confirm once; the dialog fires one `HelmService.Uninstall` RPC per release, with a progress bar.

- [ ] **`HelmTestDialog.svelte`**

Props: `ctx`, `ns`, `release`. Calls `HelmService.Test`. Shows per-pod logs (post-completion bulk read — this is a property of the SDK, not a bug). "Clean up test pods" button at the end calls a small `HelmService.CleanupTestPods` RPC (add it as part of this task — straightforward delete of any pod in the release ns with `helm.sh/hook: test` annotation).

- [ ] **`HelmForceDeleteDialog.svelte`**

Destructive confirm dialog (mirrors `BulkDeleteDialog`). Calls `HelmService.ForceDeleteReleaseSecret`. Banner: *"This deletes the Helm release Secret directly. Resources owned by the release are NOT removed. Use only for stuck `pending-*` releases."*

- [ ] **Wire dialogs**

- `HelmHistoryPanel`: "Rollback to this" → `HelmRollbackDialog`.
- `HelmOverviewPanel`: "Force delete release Secret" expander → `HelmForceDeleteDialog`; "Test release" button → `HelmTestDialog`.
- `ResourceList` row action menu (for `helm.v1.releases`): "Uninstall" → `HelmUninstallDialog`.
- `BulkActionBar` "Uninstall…" → `HelmUninstallDialog` (bulk mode).

- [ ] **Integration tests**

Bundle a sample chart under `internal/helm/testdata/charts/sample/` with: one Deployment, one Service, one ConfigMap, one CRD (in `crds/`), one Job hook (test), one Job hook (pre-install).

`internal/helm/integration_test.go` (build tag `integration`):

```go
//go:build integration
```

Test sequence (one big `TestHelmLifecycle` is fine, with sub-tests via `t.Run`):

1. Install → list shows row, revision 1.
2. Upgrade with new values → revision 2.
3. Rollback to revision 1 → revision 3.
4. History returns 3 entries descending.
5. Owned-resources (manifest-driven) returns Deployment + Service + ConfigMap + CRD — even though CRD has no Helm labels.
6. `helm test` → pass; logs non-empty.
7. CleanupTestPods removes test pods.
8. Simulate stuck `pending-upgrade` (manually craft the Secret) → `ForceDeleteReleaseSecret` cleans it.
9. Uninstall `KeepHistory=true` → row remains with `uninstalled` status.
10. Uninstall again without keep-history → row gone.
11. Continuation: install a chart with large rendered output (synthesize a chart with thousands of ConfigMap keys) → list shows one row; History reads back full payload.

Run with:

```bash
go test ./internal/helm -v -tags integration -timeout 10m
```

Document in the task plan that integration tests require a running cluster (kind or minikube) and are NOT run in CI by default.

- [ ] **Manual QA checklist**

Walk through every item in the spec's "Manual QA checklist" section against a kind cluster with the sample chart + bitnami/nginx + kube-prometheus-stack (for the umbrella-chart owned-resources case). Document any deviations as new issues.

- [ ] **Run full test suite one more time**

```bash
go test ./... -v -race
cd frontend && pnpm check && pnpm test
```

- [ ] **Commit**

```bash
jj describe -m "feat(helm): lifecycle verb UI + integration tests"
jj new
```

---

## Spec coverage check

- ✅ Phase 0 dependency bump → Task 1
- ✅ `internal/helm` package (release, continuation, aggregator, client, list, history, diff, owned, enricher, actions) → Task 2
- ✅ Engine `VirtualBackend` + dispatch → Task 3
- ✅ Watcher `fieldSelector` + `VirtualWatchSource` + `SyncStart/SyncEnd` → Task 3 (Go side) + Task 5 (frontend store)
- ✅ `helm.v1.releases` Descriptor + columns + enricher registration → Task 3
- ✅ `HelmService` RPCs (all 11 listed in spec) → Task 4
- ✅ Per-(ctx, ns, release) mutex → Task 2 (`actions.go`)
- ✅ `action.Configuration` cache → Task 2 (`client.go`) — built early per the spec's revised guidance
- ✅ `SelfSubjectAccessReview`-based permissions + `GetReleasePermissions` RPC → Task 4
- ✅ `OnClusterConnected` probe (cluster-connect hook pattern) → Task 4
- ✅ Atomic snapshot replacement on watch reconnect → Task 5
- ✅ Sidebar Helm group + availability gating + stuck-state badge → Task 5
- ✅ Bulk uninstall → Task 5 (registration) + Task 7 (dialog)
- ✅ Seven detail panels with secret-masking → Task 6
- ✅ Manifest-primary + label-fallback + bounded GVR set + "Scan all" opt-in → Task 2 (`owned.go`) + Task 6 (`HelmResourcesPanel`)
- ✅ Cross-namespace owned resources surfaced → Task 2 + Task 6
- ✅ `resource-policy: keep` detection + uninstall enumeration → Task 2 + Task 7 (`HelmUninstallDialog`)
- ✅ `--no-hooks` with inline warning copy → Task 7
- ✅ Force-delete-release-Secret recovery → Task 4 (RPC) + Task 7 (dialog)
- ✅ `helm test` as post-hoc bulk log read + cleanup button → Task 7
- ✅ Decoder size cap + sentinel + multi-Secret continuation → Task 2
- ✅ Stress-test 10k aggregator → Task 2
- ✅ Integration tests against live cluster → Task 7
- ✅ Manual QA checklist → Task 7

No gaps. Plan is complete.

---

## Execution choice

Plan complete and saved to `docs/superpowers/plans/2026-05-19-helm-support.md`. Two execution options:

1. **Subagent-driven (recommended)** — fresh subagent per task, review between tasks, fastest iteration.
2. **Inline execution** — execute tasks in this session using `superpowers:executing-plans`, batch with checkpoints.

Which approach?
