# Native Helm Support — Design

**Date:** 2026-05-19
**Scope:** First-party Helm integration for listing and managing existing releases (read-only views + lifecycle verbs: rollback, uninstall, test). No install, no upgrade, no repo management in v1.

This spec was revised after a multi-angle review. The revision integrates:
- Corrected `ResourceEngine` / `WatchManager` dispatch points (signatures match real code).
- Cluster-connect hook routed through the existing `OnClusterConnected` precedent.
- Secret-aware Values/Manifest panels with the existing `SecretPanel` reveal-gate pattern.
- Per-namespace `SelfSubjectAccessReview` for verb gating (the existing `PermissionSet.CanMutate()` is too coarse).
- Honest `helm test` UX — post-hoc bulk log read; the SDK does not stream live.
- Decoder size cap + format sentinel + multi-Secret continuation handling.
- Manifest-driven owned-resources walk, with labels as fallback only.
- Client-side verb mutex; atomic snapshot replacement on watch reconnect.
- `--no-hooks` and `resource-policy: keep` warnings.
- Helm SDK **v4** with the rolled-in `k8s.io/* v0.36.1` and Go 1.26 upgrades as Phase 0.

## Motivation

Helm is the dominant Kubernetes package manager, and clusters managed via Klados almost always have Helm releases on them. Today users have to drop to a terminal (`helm list`, `helm history`, `helm rollback`) and lose all of Klados's affordances: live watch, smart search, column pinning, bulk actions, deep-linking to owned resources, side-by-side YAML diffs.

A native integration brings Helm release management into the same surface as every other resource — same table, same filters, same detail drawer — so users never have to context-switch for the most common Helm operations.

## Goals

- List Helm releases (latest revision per release) as a first-class virtual resource type.
- Live watch — installs, upgrades, rollbacks from the CLI appear without manual refresh.
- Detail drawer parity with `helm get all`: Overview, Values (user + computed), Manifest, History, Notes, Resources (owned k8s objects), Hooks.
- Side-by-side revision diff (values and manifest) inside the History tab.
- Lifecycle verbs through existing UI patterns: rollback (with diff preview), uninstall (with keep-history / no-hooks / resource-policy-keep awareness), test (post-hoc bulk log view).
- Bulk uninstall using the existing `BulkActionBar`.
- Cross-link from a release to the resources it owns, deep-linked to standard resource detail pages.
- Treat sensitive data (values, manifests) with the same care as `SecretPanel`.

## Non-goals

- **Install / repo browsing** — out of scope.
- **Upgrade an existing release** — deferred but acknowledged as the highest-priority follow-up (see Future Work). Requires a values editor and a chart-version picker — its own design.
- **Full drift detection** (`helm template` ↔ cluster diff) — deferred to v2.
- **Values JSON-schema form** — YAML-only in v1.
- **Multi-cluster Helm view** — per-cluster, matching every other resource view.
- **Non-Secret storage drivers** — Secret only. Helm 3+ default.

## SDK and dependency choice — Helm v4

We target **`helm.sh/helm/v4`** (currently v4.2.x). This pulls forward:

- `k8s.io/* v0.35.3` → **v0.36.1** across the entire backend (`api`, `apimachinery`, `client-go`, `kubectl`, `metrics`).
- Go directive `1.25` → **1.26** in `go.mod`; `mise.toml` updated to match.
- Helm SDK uses `log/slog` natively in v4 — fits cleanly with `slox`. The new `action.NewConfiguration(action.ConfigurationSetLogger(slogHandler))` builder replaces the v3 `cfg.Log = ...` field assignment.
- `chartutil.Capabilities` moved to `common.Capabilities` — minor cosmetic change.

Phase 0 (below) handles the upgrade in isolation, before any Helm code lands. The bump is invasive on paper (every backend package imports `k8s.io/*`), but 0.35→0.36 has historically had a narrow API delta; the audit is bounded and benefits from running on its own commit so a regression bisects cleanly.

Release Secret format is still `helm.sh/release.v1` in v4 — no decoder branching needed for v3-installed releases.

## Approach: synthetic GVR + Helm SDK

Helm releases are stored as Kubernetes Secrets (`type=helm.sh/release.v1`, gzipped+base64-encoded JSON). Klados already has live-watched, columnar, filterable lists of Secrets. The integration treats `helm.v1.releases` as a **virtual GVR** that flows through the existing list/watch/detail pipeline, with an aggregator collapsing per-revision Secrets to one row per release.

Lifecycle verbs and detail-tab fetches go through a separate Wails-bound `HelmService` that wraps `helm.sh/helm/v4/pkg/action`. This ensures correct hook invocation (pre-rollback, pre-delete, post-test) — reimplementing Helm semantics is not on the table.

## Engine and watcher integration

The actual surface of `ResourceEngine` and `WatchManager` does not match a single tidy "backend" interface. Dispatch for virtual GVRs lives in **two places**, each with the existing function signatures preserved:

### `ResourceEngine.List` / `Get`

Current signatures (`internal/resource/engine.go`):

```go
func (e *Engine) List(ctx, gvr, namespace) ([]map[string]any, resourceVersion string, err error)
func (e *Engine) Get(ctx, gvr, namespace, name) (map[string]any, error)
```

The `resourceVersion` return is load-bearing — `ResourceService.ListResourcesWithVersion` seeds the subsequent watch with it. We preserve both signatures and add an early dispatch:

```go
func (e *Engine) List(ctx, gvr, ns) (...) {
    if desc := e.descriptors.Get(gvr); desc != nil && desc.IsVirtual {
        return e.virtuals[gvr].List(ctx, ns) // returns ([]map[string]any, string, error)
    }
    // existing dynamic.Interface path
}
```

The virtual backend interface (singular, lives in `internal/resource`):

```go
type VirtualBackend interface {
    List(ctx context.Context, namespace string) ([]map[string]any, string, error)
    Get(ctx context.Context, namespace, name string) (map[string]any, error)
}
```

The virtual `List` returns a synthetic `resourceVersion`. For Helm it's the **highest underlying Secret resourceVersion** observed in the snapshot — passed through unchanged so the seeded watch can start cleanly.

### Watch path — `WatchManager`

`WatchManager` (`internal/watcher/manager.go`) keys watches by `{contextName, gvr, namespace}` and constructs a `dynamic.ResourceInterface` directly. It has no field-selector support and no extension point for virtual GVRs. We change it as follows:

1. Add an optional `fieldSelector string` to `StartWatch`. Existing call sites pass `""` (no behavior change). Internally `WatchManager` plumbs this into `metav1.ListOptions.FieldSelector` for the watch call.

2. Add a `VirtualWatch` registry on `WatchManager`:

```go
type VirtualWatchSource interface {
    Watch(ctx context.Context, namespace, resourceVersion string) (<-chan WatchEvent, func(), error)
}

func (wm *WatchManager) RegisterVirtual(gvr string, source VirtualWatchSource)
```

3. In `StartWatch`, dispatch on virtual GVR before constructing the dynamic client. The virtual source is responsible for its own underlying watch (for Helm: an internal Secret watch with `fieldSelector=type=helm.sh/release.v1`) and emits `WatchEvent{Type, Object map[string]any}` in the existing format.

`WatchEvent` stays in `internal/watcher`; `internal/helm` imports it. No type duplication.

The Helm releases view and the standard `core.v1.secrets` view both opening simultaneously results in **two independent watch streams** to the API server (one filtered, one not). This is acceptable — it's how the API server works, and the cost is bounded.

## Cluster-connect hook (discovery)

There is no generic "discovery hook" on `cluster.Manager`. The existing precedent is `ClusterService.Connect` calling `volumeBrowserSvc.OnClusterConnected(ctx, name)` after successful activation (`internal/services/cluster.go:67-69`). We follow the same pattern:

```go
// internal/services/helm.go
func (s *HelmService) OnClusterConnected(ctx context.Context, name string) {
    // Probe for type=helm.sh/release.v1 Secrets, limit=1.
    // If found AND user has secrets/list in any visible namespace,
    // mark `helm.v1.releases` as available for this context.
    // Availability is read by GetDescriptors() to filter what the sidebar shows.
}
```

The synthetic GVR is injected into the frontend's resource list via the existing `GetDescriptors()` channel (the frontend already uses it to seed the registry). `DiscoverResources` is **not** modified — virtual GVRs flow through descriptor metadata, not the discovery API.

## Package layout

```
internal/helm/
  client.go         Per-cluster, per-namespace action.Configuration factory.
                    Cached: map[clusterCtx + ns]*action.Configuration (REST
                    mapper + discovery init is expensive). Eviction on
                    cluster disconnect.
  release.go        Decode helm.sh/release.v1 Secret payloads. Bounded size
                    (50 MB decompressed cap). Sentinel check: refuse decode
                    if Secret type != "helm.sh/release.v1".
  continuation.go   Detect multi-Secret release continuation (chunked payloads
                    for releases >1 MB compressed) and reassemble before
                    decode.
  aggregator.go     Snapshot + delta logic. Map per (ctx, ns). Latest revision
                    wins (by revision number, deployed-at as tiebreak).
  list.go           ListReleases(ctx, namespace) — implements VirtualBackend.List.
  history.go        GetHistory(ctx, ns, release).
  diff.go           RevisionDiff for values, computed values, manifest.
  actions.go        Rollback / Uninstall / Test wrappers. Per-RPC mutex
                    keyed by (ctx, ns, release).
  enricher.go       resource.Enricher implementation.
  owned.go          OwnedResources — manifest-driven, label-driven fallback.

internal/resource/
  descriptor.go     + IsVirtual bool on Descriptor.
  engine.go         + VirtualBackend interface, dispatch in List/Get.
  builtin.go        + helm.v1.releases Descriptor (columns, virtual flag,
                    enricher hookup, supported detail tabs, fieldSelector
                    hint for the underlying Secret watch).

internal/watcher/
  manager.go        + StartWatch accepts fieldSelector; + RegisterVirtual
                    + VirtualWatchSource interface.

internal/services/
  helm.go           HelmService — Wails-bound RPCs for verbs + lazy
                    detail-tab fetches + cluster-connect probe.
```

### `HelmService` RPCs

```go
Rollback(ctx, ns, release, revision int, opts RollbackOpts) error
Uninstall(ctx, ns, release, opts UninstallOpts) error
Test(ctx, ns, release, opts TestOpts) (TestResult, error)
                                       // TestResult includes per-pod bulk logs

GetValues(ctx, ns, release, computed bool, revision int) (string, error)
GetManifest(ctx, ns, release, revision int) (string, error)
GetHistory(ctx, ns, release) ([]Revision, error)
GetNotes(ctx, ns, release, revision int) (string, error)
GetHooks(ctx, ns, release, revision int) ([]Hook, error)
GetOwnedResources(ctx, ns, release) ([]OwnedRef, error)
DiffRevisions(ctx, ns, release, from, to int) (RevisionDiff, error)

ForceDeleteReleaseSecret(ctx, ns, release, revision int) error
                          // Recovery path for stuck pending-* states.
```

`RollbackOpts`, `UninstallOpts`, `TestOpts` all carry `Wait`, `Timeout`, `DisableHooks`. `UninstallOpts` also carries `KeepHistory`.

Each verb RPC takes a per-(ctx, ns, release) mutex (in `actions.go`) to prevent a double-clicked button from racing itself. The mutex is released on RPC return.

## Aggregator + decoder

### Decoder

`release.go`:
1. Read `secret.Data["release"]`.
2. Reject if `secret.Type != "helm.sh/release.v1"`.
3. Base64-decode into a fixed-size scratch buffer; reject if base64 length implies >50 MB after decode.
4. If gzip magic header present (`0x1f 0x8b 0x08`), gunzip with a `io.LimitReader` capped at 50 MB. Reject on cap exceeded.
5. JSON-decode into a `release.Release` (Helm SDK type).
6. Errors bubble up wrapped with `slox.With(ctx, "helm.secret", secret.Name, "helm.namespace", secret.Namespace)`.

The 50 MB cap protects against zip-bombs from any user with `secrets/create`. Releases above the cap are rare and the failure mode is "row appears with `status=unreadable`" — same as a corrupt release. Bound is constant in code; revisit only on real-world feedback.

### Continuation (multi-Secret releases)

Helm splits releases >1 MB compressed across multiple Secrets named `sh.helm.release.v1.<release>.v<rev>.<chunk>` (see Helm v4 `pkg/storage/driver/secrets.go`). `continuation.go`:

1. Scans Secrets matching the chunked-name pattern in the aggregator's input set.
2. Groups by `(release, rev)`, sorts chunks, concatenates the base64 payload before handing to the decoder.
3. Single-chunk releases (the common case) bypass this entirely.

### Aggregator

`aggregator.go` keeps `map[namespace]map[releaseName]map[revisionNumber]revisionMeta` per (ctx, ns). On every delta:
- Decode the Secret (or assemble continuation chunks first).
- Update the map.
- Recompute latest: max revision number; deployed-at as tiebreak.
- Emit `ADDED` / `MODIFIED` / `DELETED` virtual event when the latest changes.
- No-op when an old revision is deleted but latest is unchanged.

Reconnect path (watch died, reconnect): aggregator clears its (ctx, ns) submap, lists fresh, replays all entries, and emits a **single snapshot event** (new field `WatchEvent.Type = "SYNC"` or — to avoid touching the watcher contract — emits explicit ADDED events for the new set after the frontend has cleared its `items[]` for this GVR). We pick the second option: emit an `out-of-band` `SyncStart`/`SyncEnd` pair via the Wails event channel that `ResourceStore` interprets as "drop current items[] and rebuild from incoming ADDEDs." This is the **atomic snapshot replacement** that closes the list-watch ghost-release window.

## Data flow

### List + watch

```
helm.v1.releases list opens
        ↓
ResourceService.ListResourcesWithVersion(ctx, "helm.v1.releases", ns)
        ↓
ResourceEngine.List → desc.IsVirtual=true → helm.virtualBackend.List(ctx, ns)
        ↓
underlying Secret list with fieldSelector=type=helm.sh/release.v1
        ↓
continuation reassembly → aggregator.collapseToLatest(secrets)
        ↓
[]map[string]any  (one per release; embeds chart, version, status, revision,
                   lastDeployed, appVersion)
        ↓
EnricherRegistry → helm.Enricher.Enrich() adds display fields
        ↓
ResourceStore items[]  →  ResourceList table

— and in parallel —

WatchManager.StartWatch(ctx, "helm.v1.releases", ns, resourceVersion=rv)
        ↓
helm.VirtualWatchSource.Watch(ctx, ns, rv)
        ↓
internal Secret watch (fieldSelector=type=helm.sh/release.v1, resumed from rv)
        ↓
aggregator.applyDelta(ev) → ADDED / MODIFIED / DELETED virtualRelease
        ↓
emit watch:{ctx}:helm.v1.releases:{ns}  {type, object}
        ↓
ResourceStore reconciles

— on watch reconnect —

SyncStart event → ResourceStore clears items[] for this gvr+ns
ADDED for each release in fresh snapshot → ResourceStore rebuilds items[]
SyncEnd event → ResourceStore re-renders
```

### Detail-tab fetches

| Tab | RPC | Source |
|---|---|---|
| Overview | (in row) | virtual unstructured |
| Values | `GetValues(rev, computed=false)` | active revision's Secret |
| Computed Values | `GetValues(rev, computed=true)` | merged defaults + overrides |
| Manifest | `GetManifest(rev)` | active revision's Secret |
| Notes | `GetNotes(rev)` | active revision's Secret |
| History | `GetHistory(release)` | all revisions |
| Resources | `GetOwnedResources(release)` | manifest-driven (see below) |
| Hooks | `GetHooks(rev)` | active revision's hook defs + recent cluster state |

### Lifecycle verbs

```
User clicks "Rollback to revision 3"
        ↓
ConfirmDialog with preview-diff via HelmService.DiffRevisions(current, 3)
        ↓
(disable Rollback button; show spinner)
        ↓
HelmService.Rollback(ctx, ns, release, 3, opts)
        ↓
acquire (ctx, ns, release) mutex; reject if held
        ↓
action.NewRollback(actionConfig).Run(release)  (runs hooks unless disabled)
        ↓
new revision Secret appears
        ↓
secret watch → aggregator → ResourceStore → row reflects new revision
        ↓
release mutex; (re-enable button)
        ↓
notification "Rolled back <release> to revision 3 (new revision N)"
```

Uninstall mirrors this, with `KeepHistory`, `DisableHooks`, and a `resource-policy: keep` advisory (see Error handling).

## Frontend integration

### Sidebar

`helm.v1.releases` is registered as a Descriptor with synthetic group label `"Helm"`. Sidebar entry is hidden on clusters where `OnClusterConnected`'s probe found no Helm Secrets OR where `secrets/list` is denied in every visible namespace.

### List page

Existing `ResourceListPage` — zero changes. Default columns:

| Column | Source | Render |
|---|---|---|
| Name | `metadata.name` | text |
| Namespace | `metadata.namespace` | text |
| Status | `helm.statusDisplay` | badge (deployed=ok, failed/pending-*=destructive for failed/warning for pending, superseded/uninstalled=muted) |
| Revision | `helm.revisionDisplay` | text |
| Chart | `helm.chartDisplay` (e.g. `nginx-15.4.4`) | text |
| App Version | `helm.appVersion` | text |
| Last Deployed | `helm.lastDeployedDisplay` | age |

Smart search, column pinning, saved filters, bulk select all work via the standard CEL pipeline. Bulk uninstall is a new `BulkActionBar` entry, registered conditionally for this GVR.

Stuck-state indicator: rows with `status=pending-install`, `pending-upgrade`, `pending-rollback`, `uninstalling` show a destructive badge. The detail drawer's Overview panel includes a "Release stuck?" expander with a `ForceDeleteReleaseSecret` action behind a destructive confirm dialog — the documented recovery path.

### Detail drawer panels

New panels in `frontend/src/lib/components/panels/`:

- **HelmOverviewPanel.svelte** — chips for chart name, chart version, app version, revision, status. "Source" link to OCI ref from `meta.helm.sh/release-source` annotation when present. Stuck-state expander with `ForceDeleteReleaseSecret`.
- **HelmValuesPanel.svelte** — read-only `YAMLEditor`, sub-tabs `user-supplied` / `computed`, revision picker.
  **Secret-handling**: a per-key reveal-gate identical to `SecretPanel.svelte`'s pattern. Any key whose name matches `password|token|secret|key|cert|credential|apikey|passphrase` (case-insensitive, applied at any nesting depth) is masked as `••••••••` with a per-key "Show" button. The panel header carries a persistent banner: *"Values may contain sensitive data."* Copy actions copy decoded (revealed) values only; a separate "Copy redacted" action preserves the masking for safe sharing.
- **HelmManifestPanel.svelte** — read-only YAML, revision picker. Same secret-handling banner. `stringData` and `data` fields of any Secret resource in the rendered manifest are masked; the rest of the manifest renders verbatim.
- **HelmHistoryPanel.svelte** — revisions table; row click previews; "Compare" mode picks two revisions, shows `DiffView` for values and manifest in tabs (same secret-masking); "Rollback to this" verb on past revisions.
- **HelmNotesPanel.svelte** — NOTES.txt (markdown if it parses, else preformatted).
- **HelmResourcesPanel.svelte** — owned resources grouped by kind. Each row deep-links to `/c/:ctx/:gvr/:ns/:name`. Cross-namespace resources show their namespace per row (releases commonly own kube-system resources).
- **HelmHooksPanel.svelte** — hooks table (event, kind/name, weight, last-run, phase). Click → open the underlying resource if still present.

Tabs are registered via the existing per-Descriptor mechanism in `ResourceDetail.svelte`.

Verb buttons are disabled while their RPC is in-flight (the existing pattern for Delete in `BulkActionBar` — a per-action `pending` flag on the store).

### Wails bindings

After adding `HelmService`, run `wails3 generate bindings`. Frontend imports `bindings/.../HelmService.js` with `.js` ESM pattern.

## Permissions

The existing `PermissionSet.CanMutate()` hardcodes its `SelfSubjectRulesReview` to `kube-system` and is namespace-blind — it can't gate per-namespace Helm verbs honestly. We extend rather than rely on it:

- **Sidebar visibility**: `OnClusterConnected` probe — coarse, fine.
- **Per-verb gating**: a new lightweight `internal/cluster.CheckAccess(ctx, namespace, verb, gvr)` helper that issues a `SelfSubjectAccessReview` (single API call, cheap). Called by `HelmService` at the start of each RPC and surfaced to the frontend via a `GetReleasePermissions(ns, release)` RPC that returns `{canRollback, canUninstall, canTest, canForceDelete}` — these gate button enablement honestly.
- **Caching**: per-(ctx, ns, verb) for the lifetime of the detail-drawer session. Invalidated on cluster disconnect.

This adds a few SAR calls per detail-drawer open. SAR is designed to be cheap; we accept the cost.

We still document the residual risk: rollback/uninstall write to many resources the chart manages, not just the release Secret. A user may have Secret write but lack write on, say, ClusterRoles the chart creates. SAR can only check what we ask it; we don't pre-walk the rendered manifest. Mid-operation RBAC failures remain possible. The Confirm dialog includes a one-line note: *"You may lack permissions on resources this chart manages; partial state can result on RBAC errors."*

## Owned resources — manifest-driven primary, labels as fallback

The label heuristic (`app.kubernetes.io/managed-by=Helm` + `meta.helm.sh/release-name=<n>`) misses CRDs (Helm doesn't label `crds/` resources), pre-3.2 releases, hand-rolled charts, and Flux/Argo-wrapped releases. We invert the approach:

1. **Primary**: parse the release's rendered manifest (already fetched for the Manifest tab; cache per release-detail session). For each YAML document, extract `apiVersion`, `kind`, `metadata.namespace`, `metadata.name`. Group by `(gvk, namespace)` and issue one `List` per group — bounded by the chart's footprint, not the cluster's GVR count.
2. **Fallback / supplement**: a second pass using the label selector across a **bounded default GVR set** — Deployments, StatefulSets, DaemonSets, ReplicaSets, Services, ConfigMaps, Secrets, Ingresses, PVCs, ServiceAccounts, Jobs, CronJobs, Roles, RoleBindings, NetworkPolicies, HPAs. This catches hook-created resources whose manifest may not appear in the active revision's stored manifest. Deduplicated with the primary set.
3. **Opt-in "scan all"**: a per-detail button "Scan all GVRs" runs the label query across every discovered GVR for users on clusters with non-standard CRDs the release owns. Single user action, single time per session — bounded.

Cross-namespace owned resources (releases in `apps` owning `kube-system` ClusterRoles) surface because step 1 reads the namespace from the manifest, not from a label filter. The Resources panel shows the namespace column per row.

Charts that mark resources with `helm.sh/resource-policy: keep` are highlighted in the panel. Uninstall's confirm dialog enumerates them: *"3 resources marked `resource-policy: keep` will remain after uninstall."*

## Error handling

| Failure | Surface | Recovery |
|---|---|---|
| Secret decode fails (corrupt / size cap / wrong type sentinel) | Aggregator skips that revision, logs warn, emits virtual row with `status=unreadable`, banner in detail drawer. | Inspect History; manually delete bad Secret via `core.v1.secrets`. |
| Verb RPC fails (Helm SDK error) | Toast notification with the Helm error verbatim; error preserved in detail-drawer status area until next watch update. | Retry; partial state visible in History. |
| Mid-operation hook failure | Release left in `pending-*`. List shows destructive badge. | History tab shows failing hook; Confirm dialog offers `--no-hooks` escape hatch (off by default, accompanied by an inline warning: *"Skipping hooks may leave the cluster in an inconsistent state and is intended for stuck-hook recovery, not routine use."*); or `ForceDeleteReleaseSecret` for fully-stuck releases. |
| User double-clicks Rollback | Button disables on first click; second click is a no-op. RPC layer has a per-(ctx, ns, release) mutex as defense-in-depth. | None needed. |
| External CLI runs `helm upgrade` mid-Klados-verb | Helm refuses with "another operation is in progress". Toast propagates verbatim. | User waits, retries. No automatic retry. |
| Test pods fail | Test panel shows pass/fail and per-pod logs (bulk read after completion — Helm SDK does not stream). "Clean up test pods" button removes them. | User decides whether to keep. |
| Watch reconnect | Aggregator rebuilds; SyncStart/SyncEnd brackets ensure `ResourceStore` replaces `items[]` atomically — no ghost releases. | Automatic. |
| Continuation Secret chunk missing | Decode fails; `status=unreadable`. | User investigates; rare. |
| OCI source URL unreachable | Overview link is external; no fetch. | N/A. |

### Logging

Every Helm SDK call gets `slox.With(ctx, "helm.release", name, "helm.namespace", ns)`. Helm v4's `slog` integration routes the SDK's own logs through our `slox`-backed handler via `action.ConfigurationSetLogger(slogHandler)`.

## Testing

### Go unit tests (no CGO)

| Package | Coverage |
|---|---|
| `internal/helm/release.go` | Good fixtures; size-cap rejection (decompressed >50 MB); wrong-type sentinel; truncated base64; corrupt gzip; malformed JSON. |
| `internal/helm/continuation.go` | Two-chunk and three-chunk reassembly; out-of-order chunks; missing chunk → error. |
| `internal/helm/aggregator.go` | **Critical.** Snapshot reduction. Delta: ADDED new revision, DELETED old revision (no-op), DELETED latest (emits new latest), DELETED last revision (emits DELETED virtual). Revision-number ties broken by deployed-at. Reconnect emits SyncStart/SyncEnd. Stress test with 10k synthetic Secrets. |
| `internal/helm/list.go` | Field selector correct; empty result; all-namespaces. Synthetic resourceVersion is the max underlying RV. |
| `internal/helm/history.go` | Descending sort; malformed Secrets skipped. |
| `internal/helm/diff.go` | Unified-diff text; equal revisions → empty; missing revision → typed error; secret-masking applied to diff output. |
| `internal/helm/enricher.go` | All display fields populated; missing metadata → safe defaults. |
| `internal/helm/actions.go` | Verb wrappers call mocked `action.Configuration`. Per-(ctx, ns, release) mutex serializes concurrent verb invocations. |
| `internal/helm/owned.go` | Manifest-driven extraction handles all major YAML shapes (single-doc, multi-doc, --- separators, comments, empty docs). Label-fallback path against stub engine. Cross-namespace resources are returned. `resource-policy: keep` is detected. |
| `internal/helm/client.go` | `action.Configuration` cache keyed by (ctx, ns); eviction on cluster disconnect. |
| `internal/resource/engine_test.go` (additions) | `IsVirtual=true` dispatches to registered backend; preserves resourceVersion return; multiple virtuals don't conflict. |
| `internal/watcher/manager_test.go` (additions) | `fieldSelector` plumbed correctly; `RegisterVirtual` dispatch; existing call sites with empty selector unchanged. |
| `internal/services/helm_test.go` | Per RPC: happy path, RBAC-denied, release-not-found, malformed-args, concurrent-verb-rejection. Fake `ConnectionProvider`. SAR-based permissions. |

Fixtures in `internal/helm/testdata/`:
- `release-deployed.secret.yaml`
- `release-failed.secret.yaml`
- `release-superseded.secret.yaml`
- `release-stuck-pending-upgrade.secret.yaml`
- `release-multi-revision/` — five revisions across two release names.
- `release-large-continuation/` — three-chunk release for continuation tests.
- `release-with-secret-values.secret.yaml` — values containing password/token keys.

### Integration tests (`-tags integration`)

`internal/helm/integration_test.go`, bundled sample chart under `testdata/charts/sample/`:

1. Install chart → list shows one virtual row.
2. Upgrade with new values → row updates, revision incremented.
3. Rollback to v1 → row updates, revision 3.
4. History returns three revisions, descending.
5. Owned-resources walk (manifest-driven) returns all three chart resources, including a CRD that has no Helm labels.
6. `helm test` against chart with test hook → expect pass, logs returned.
7. Test pod cleanup succeeds.
8. Stuck-state recovery: simulate `pending-upgrade` Secret, `ForceDeleteReleaseSecret` removes it cleanly.
9. Uninstall with `KeepHistory=true` → row shows `uninstalled` status.
10. Uninstall again without keep-history → row gone.
11. Multi-Secret continuation: install a chart with large rendered output → list correctly shows one row, History reads back the full payload.

Run manually with `go test ./internal/helm -v -tags integration` against a kind/minikube cluster.

### Frontend tests

- `HelmHistoryPanel.svelte.test.ts` — descending order; Compare mode toggle; Rollback opens ConfirmDialog; secret-masking propagates into the diff view.
- `HelmValuesPanel.svelte.test.ts` — user-supplied / computed switch; revision picker re-fetches; per-key reveal gate matches `SecretPanel`'s pattern; "Copy redacted" preserves masking.
- `HelmResourcesPanel.svelte.test.ts` — grouped by kind; rows link to correct GVR routes; cross-namespace resources show namespace.
- `HelmOverviewPanel.svelte.test.ts` — stuck-state expander hidden for healthy releases, visible+actionable for `pending-*`.
- `HelmService.test.ts` — mocks `@wailsio/runtime`; RPC call shapes; verb buttons disable while in-flight.

Storybook stories for each new panel under `apps/docs/src/stories/`.

### Manual QA checklist

- [ ] List shows latest revision only when a release has multiple revisions.
- [ ] CLI `helm install` appears in UI within ~1 s.
- [ ] CLI `helm upgrade` updates row revision/status without re-list.
- [ ] Rollback via UI matches `helm rollback` outcome.
- [ ] Uninstall with keep-history shows row in `uninstalled` status.
- [ ] Uninstall with `resource-policy: keep` resources warns in dialog.
- [ ] Test panel shows pass/fail and bulk logs (post-completion).
- [ ] Test pod cleanup button removes test pods.
- [ ] Stuck `pending-upgrade` release shows destructive badge + force-delete affordance.
- [ ] Cluster with no Helm releases hides the sidebar entry.
- [ ] Cluster with `secrets/list` denied in every namespace hides the sidebar entry.
- [ ] Smart search filters by chart name, status, revision count.
- [ ] Bulk uninstall confirms once for N selected releases.
- [ ] Values panel masks `password`/`token`/`secret`/etc. keys; "Show" reveals each.
- [ ] Manifest panel masks Secret resource `stringData`/`data` fields.
- [ ] Watch reconnect after network blip does not show ghost releases.
- [ ] Cross-namespace owned resources (release in `apps`, owns `kube-system` ClusterRole) appear in the Resources tab.
- [ ] Double-clicking Rollback fires the RPC exactly once.

## Build sequence (Phase 0 — Phase 6)

### Phase 0 — Dependency stack upgrade (no Helm code yet)

- `go.mod`: bump `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/client-go`, `k8s.io/kubectl`, `k8s.io/metrics`, `k8s.io/apiextensions-apiserver` from `v0.35.3` → `v0.36.1`. Bump Go directive `1.25` → `1.26`.
- `mise.toml`: bump Go to 1.26.
- Run full test suite (`go test ./internal/... -v`) — fix any 0.35→0.36 API breaks. Likely candidates: discovery API signatures, watch.Interface, RBAC types. Each fix is its own jj commit.
- Verify Wails CGO build succeeds.
- Smoke-test the app against a live cluster: connect, list a few resources, watch, exec.

**Deliverable:** klados runs unchanged on Go 1.26 + k8s.io/* v0.36.1. Single PR / single bisectable jj branch.

### Phase 1 — Foundation: `internal/helm` package

- Add `helm.sh/helm/v4 v4.2.x` to `go.mod`. Run `go mod tidy`; confirm no further bumps.
- Implement `release.go`, `continuation.go`, `aggregator.go`, `client.go`, `list.go`, `history.go`, `diff.go`, `owned.go`, `enricher.go`, `actions.go` with unit tests.
- `helm.Enricher` registered with `EnricherRegistry`.

**Deliverable:** `internal/helm/*` tested in isolation. No frontend impact.

### Phase 2 — Engine + watcher plumbing for virtual GVRs

- `Descriptor.IsVirtual` field.
- `VirtualBackend` interface in `internal/resource`; `Engine.List/Get` dispatch.
- `WatchManager.StartWatch` accepts `fieldSelector`; `RegisterVirtual` + `VirtualWatchSource`.
- `helm.v1.releases` Descriptor in `builtin.go`; helm virtual backend + watch source wired up.
- `OnClusterConnected` probe in `HelmService` (added in Phase 3, but interface and call site land here).
- Update `GetDescriptors()` to emit virtual descriptors with availability flag.

**Deliverable:** `ResourceService.ListResources("helm.v1.releases", ns)` works end-to-end on a real cluster.

### Phase 3 — Wails service for verbs + lazy fetches

- `internal/services/helm.go` exposing all RPCs.
- Per-(ctx, ns, release) mutex in `actions.go`.
- `action.Configuration` cache in `client.go`, evicted on cluster disconnect.
- `SelfSubjectAccessReview` helper + `GetReleasePermissions` RPC.
- Service tests with fake `ConnectionProvider`.
- `wails3 generate bindings` + commit generated TS.

**Deliverable:** All Helm RPCs callable from frontend; permissions resolved per namespace.

### Phase 4 — Frontend list + sidebar

- Sidebar registration (Helm group).
- Default columns for `helm.v1.releases`.
- Stuck-state destructive badge in the list.
- `BulkActionBar` "Uninstall" verb.
- Smart-search compatibility test.

**Deliverable:** `/c/:ctx/helm.v1.releases` renders live; bulk uninstall works.

### Phase 5 — Detail drawer panels

- Seven new `Helm*Panel.svelte` components.
- `HelmValuesPanel` and `HelmManifestPanel` integrate the `SecretPanel`-style reveal-gate (shared helper extracted if needed).
- Tab registration in `ResourceDetail.svelte`.
- Storybook stories.
- Component tests for History, Values, Resources, Overview.

**Deliverable:** Full detail drawer experience including secret-aware redaction.

### Phase 6 — Lifecycle verb UI + integration tests

- Rollback confirm dialog with preview diff.
- Uninstall confirm dialog (`KeepHistory`, `--no-hooks` with warning, `resource-policy: keep` enumeration).
- Test verb side panel with post-hoc bulk logs + cleanup button.
- Force-delete-release-Secret recovery flow.
- Integration tests in `internal/helm/integration_test.go`.
- Manual QA checklist completion.

**Deliverable:** v1 ship-ready.

## Risks

- **Phase 0 k8s minor bump audit**: 0.35→0.36 API delta is historically small but not zero. Allocate buffer time for the audit; do it in a dedicated PR/branch so regressions bisect cleanly.
- **Helm SDK dependency weight** — v4 pulls ~50 transitive modules. Confirm no surprising further k8s.io bumps with `go mod why`.
- **Custom release decoder vs unstable SDK internal**: format is `v1` and Helm has not bumped it across v3 and v4, but the `encodeRelease`/`decodeRelease` functions remain unexported. Type sentinel + size cap + golden-fixture tests keep this manageable; revisit if Helm ever ships `v2`.
- **SAR call cost on detail-drawer open**: four calls per release. Cached per session. Cheap in practice; revisit if profiling shows otherwise.
- **Aggregator memory on huge clusters**: 10k-release stress test is part of Phase 1.

## Future work (v2 candidates, ranked)

1. **`helm upgrade`** — values editor (YAML + JSON-schema form when chart provides `values.schema.json`), chart-version picker, dry-run preview, post-upgrade diff. This is the verb that makes the integration useful daily; it should follow v1 closely, not be deferred indefinitely.
2. **Per-resource drift hint** — on the Resources tab, a "diff vs live" affordance per row that runs `helm template <release> | yq` against the live object for that single resource. 80% of full drift's value at 20% of the implementation cost.
3. **Full drift detection** — `helm template` ↔ cluster reconciliation with field-manager-aware normalization.
4. **Chart metadata sub-view** — Chart.yaml, Chart.lock, dependency tree (umbrella charts like kube-prometheus-stack).
5. **Install from OCI** — when (4) is in place and we have a values editor from (1), install becomes a natural extension.
6. **Values JSON-schema form** — render `values.schema.json` via the existing `SchemaForm.svelte`.
