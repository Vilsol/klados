# PVC Volume Browser Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a one-click "Browse Volume" action that spawns a helper Pod with any PVC mounted, opens its detail drawer, and attaches a bottom-panel terminal when the Pod is ready — handling RWO node affinity transparently.

**Architecture:** New self-contained domain package `internal/volumebrowser/` (spawner, discovery, tracker, orphan scanner, cleanup) wrapped by a thin `VolumeBrowserService` Wails service, mirroring `internal/portforward`/`internal/exec`/`internal/logs`. Frontend adds a dialog, store, new `terminal-pending` bottom-panel kind, and entry points in the PVC list/detail/shortcut/command-palette surfaces. Config is per-cluster-resolvable via the existing `config.Resolve` pattern.

**Tech Stack:** Go 1.25 + Wails v3 alpha.74 + k8s.io/client-go dynamic client + Svelte 5 runes + Tailwind v4 + vitest + testza.

**Spec:** `docs/superpowers/specs/2026-04-19-pvc-volume-browser-design.md`

---

### Task 1: Config schema + per-cluster resolution

Add the `VolumeBrowserConfig` block to the global config, wire it through the per-cluster resolver, and unit-test both layers before touching anything else. This is the foundation every later task reads from.

**Files:**
- Modify: `internal/config/config.go` — add `VolumeBrowserConfig` type and field on `Config`, wire default values
- Modify: `internal/config/resolve.go` — add `VolumeBrowser` field on `ResolvedPrefs`, add `ResolveForCluster` merge logic
- Modify: `internal/config/config_test.go` — add defaults + roundtrip tests
- Modify: `internal/config/resolve_test.go` — add global-default, per-cluster-override, nil-resources, nil-deadline cases

**Implementation:**

In `internal/config/config.go`, add:

```go
type ResourceReqs struct {
    Requests map[string]string `json:"requests,omitempty"`
    Limits   map[string]string `json:"limits,omitempty"`
}

type VolumeBrowserConfig struct {
    Image                  string              `json:"image,omitempty"`
    MountPath              string              `json:"mountPath,omitempty"`
    ReadOnly               bool                `json:"readOnly,omitempty"`
    ActiveDeadlineSeconds  *int64              `json:"activeDeadlineSeconds,omitempty"`
    Resources              *ResourceReqs       `json:"resources,omitempty"`
    NodeSelector           map[string]string   `json:"nodeSelector,omitempty"`
    Tolerations            []map[string]any    `json:"tolerations,omitempty"` // raw JSON to avoid corev1 dep in config
    PromptBeforeSpawn      bool                `json:"promptBeforeSpawn,omitempty"`
    OrphanCleanupOnStartup string              `json:"orphanCleanupOnStartup,omitempty"` // "prompt"|"auto"|"ignore"
}
```

Add `VolumeBrowser VolumeBrowserConfig` to the global `Config` struct and a matching `*VolumeBrowserConfig` on any per-cluster override struct that already exists. Defaults are set in the existing `DefaultConfig()` (or equivalent):

```go
func defaultVolumeBrowser() VolumeBrowserConfig {
    deadline := int64(3600)
    return VolumeBrowserConfig{
        Image:                  "alpine:edge",
        MountPath:              "/mnt/volume",
        ReadOnly:               false,
        ActiveDeadlineSeconds:  &deadline,
        Resources:              nil, // unset by default per Q5
        NodeSelector:           nil,
        Tolerations:            nil,
        PromptBeforeSpawn:      false,
        OrphanCleanupOnStartup: "prompt",
    }
}
```

In `internal/config/resolve.go`, add a field `VolumeBrowser VolumeBrowserConfig` to `ResolvedPrefs` and merge logic that mirrors the existing metrics/theme resolution (global → per-cluster override; nil fields fall through to global).

**TDD:**
1. Write failing tests first covering: default values present; JSON round-trip preserves all fields; `ResolveForCluster` returns global when no override; per-cluster override replaces only set fields; `ActiveDeadlineSeconds = nil` is preserved (means "no deadline"); empty override does not clobber global.
2. Implement until green.

**Verify:**
```bash
go test ./internal/config/ -v
```
Expected: all new + existing tests pass.

**Commit:** `feat(config): add VolumeBrowserConfig with per-cluster resolution`

---

### Task 2: `internal/volumebrowser/` domain package

The whole lifecycle in one package: pod spec builder, node discovery, in-memory tracker, orphan scanner. No Wails dependency; all I/O goes through injected interfaces so tests use fake dynamic clients. This is the single largest task; budget a full session.

**Files:**
- Create: `internal/volumebrowser/spawner.go` — `Spawner` with `Spawn(ctx, req) (*ManagedPod, error)`; builds pod spec, calls discovery, creates pod via dynamic client
- Create: `internal/volumebrowser/spawner_test.go`
- Create: `internal/volumebrowser/discovery.go` — `ResolveNode(ctx, conn, pvc)`: RWX short-circuit → VolumeAttachment lookup → pod-scan fallback
- Create: `internal/volumebrowser/discovery_test.go`
- Create: `internal/volumebrowser/tracker.go` — `Tracker` thread-safe `map[ctxName][]*ManagedPod`
- Create: `internal/volumebrowser/tracker_test.go`
- Create: `internal/volumebrowser/orphans.go` — `ScanOrphans(ctx, conn, sessionUUID) ([]OrphanPod, error)` using label selector
- Create: `internal/volumebrowser/orphans_test.go`
- Create: `internal/volumebrowser/manager.go` — `Manager` stitches the four together, holds session UUID, exposes `Spawn`, `Stop`, `StopAll`, `StopForContext`, `ListManaged`
- Create: `internal/volumebrowser/manager_test.go`
- Create: `internal/volumebrowser/types.go` — `SpawnRequest`, `SpawnOverrides`, `ManagedPod`, `OrphanPod`

**Key type definitions (authoritative — later tasks must match):**

```go
package volumebrowser

type SpawnRequest struct {
    ContextName string
    Namespace   string
    PVCName     string
    Overrides   *SpawnOverrides // nil = use resolved config as-is
}

type SpawnOverrides struct {
    Image                 *string
    MountPath             *string
    ReadOnly              *bool
    ActiveDeadlineSeconds *int64 // pointer-to-nil = explicitly unset
    Resources             *config.ResourceReqs
    NodeSelector          map[string]string
    Tolerations           []map[string]any
}

type ManagedPod struct {
    ID            string    // UUID (the tracker key)
    ContextName   string
    Namespace     string
    PodName       string
    PVCName       string
    CreatedAt     time.Time
    SessionUUID   string
    TerminalTabID string    // set via Manager.AttachTab(id, tabID)
}

type OrphanPod struct {
    ContextName string
    Namespace   string
    PodName     string
    PVCName     string
    CreatedAt   time.Time
    SessionUUID string
}
```

**Labels (authoritative, repeated in later tasks — do not change):**
- `app.kubernetes.io/managed-by=klados`
- `klados.io/purpose=pvc-browser`
- `klados.io/pvc=<pvc-name>`
- `klados.io/session=<sessionUUID>`

**Pod spec rules:**
- Name: `fmt.Sprintf("klados-pvc-%s-%s", truncate(pvc, 40), randHex(4))`
- `spec.restartPolicy = "Never"`
- `spec.terminationGracePeriodSeconds = ptr.To(int64(1))` (1s, not 0 — lets kubelet send SIGTERM before SIGKILL)
- `spec.activeDeadlineSeconds` from config/override (omit if nil)
- `spec.nodeName` from `ResolveNode` when non-empty; otherwise omitted
- Single container named `browser`, `command: ["sh","-c","sleep infinity"]`
- Single volume of type PVC referencing the source PVC name
- Single volumeMount at configured path with configured readOnly
- `securityContext`: no `privileged`, no capabilities, `runAsNonRoot: false`
- Resources: when `Resources != nil`, set `requests`/`limits` from the map; when `nil`, omit the resources block entirely

**Discovery rules (`discovery.go`):**
```
ResolveNode(ctx, conn, pvc) →
  if pvc.accessModes contains "ReadWriteMany" or "ReadOnlyMany": return ""
  list VolumeAttachment (GVR: storage.k8s.io/v1/volumeattachments) cluster-scoped
    for each: if .spec.source.persistentVolumeName == pvc.spec.volumeName → return .spec.nodeName
  (on RBAC forbidden, log debug, fall through; don't propagate)
  list pods in pvc.namespace
    keep pods with .spec.volumes[*].persistentVolumeClaim.claimName == pvc.name
    keep phase == "Running"
    return .spec.nodeName of first (if any)
  return ""
```

**Orphan rules (`orphans.go`):**
- List Pods cluster-wide with label selector `klados.io/purpose=pvc-browser`
- Filter to those whose `klados.io/session` label != current sessionUUID
- Return the list; caller decides what to do per `OrphanCleanupOnStartup`

**Use the existing `cluster.Manager` abstractions:** the package takes a `ConnectionProvider` interface (match the pattern used by `internal/portforward`, `internal/logs`, `internal/exec`). Read one of those packages for the exact shape, and mirror it.

**TDD plan (write tests first for each file):**

- `spawner_test.go`: pod name pattern, all labels present, nil-resources omits container.resources, non-nil resources populates both requests/limits, readOnly respected, nodeName populated when discovery returns a node, RWX → nodeName empty, PVC not Bound returns typed error, collision with existing managed pod returns typed `ErrCollision` (sentinel exported in `types.go`).
- `discovery_test.go`: table-driven with a fake dynamic client: RWX short-circuit, RWO VolumeAttachment hit, VolumeAttachment RBAC forbidden → pod-scan hit, no running pods → empty, detached-volume (neither source) → empty.
- `tracker_test.go`: concurrent `Add`/`Remove` safe (use `-race`), `Get` by id, `ListForContext`, `RemoveAll` clears everything.
- `orphans_test.go`: sessionUUID filter excludes own pods, label selector correctness.
- `manager_test.go`: `Spawn` → tracker has entry, `Stop(id)` deletes pod and removes from tracker, `StopForContext(ctx)` deletes only that context's pods, `StopAll` drains across contexts.

**Verify:**
```bash
go test ./internal/volumebrowser/ -v -race
```
Expected: all pass under `-race`.

**Commit:** `feat(volumebrowser): add spawner, discovery, tracker, and orphan scanner`

---

### Task 3: Wails service + AppService wiring + bindings

Thin service exposing `Spawn`, `Stop`, `AttachTab`, `ListManaged`, `ScanOrphans`, `CleanupOrphans`. Lifecycle hooks do the disconnect/shutdown cleanup.

**Files:**
- Create: `internal/services/volumebrowser.go`
- Create: `internal/services/volumebrowser_test.go`
- Modify: `internal/services/app.go` — construct `volumebrowser.Manager` (passing the cluster connection provider and resolved-config accessor), expose via `VolumeBrowserManager()` getter
- Modify: `main.go` — register `VolumeBrowserService` alongside existing services
- Modify: `internal/services/cluster.go` — on cluster disconnect, call `manager.StopForContext(ctxName)` (follow the pattern already used by port-forward cleanup)
- Regenerate: `frontend/bindings/...` via `wails3 generate bindings`

**Service surface:**

```go
type VolumeBrowserService struct {
    appService *AppService
    manager    *volumebrowser.Manager
    ctx        context.Context
}

func NewVolumeBrowserService(appSvc *AppService) *VolumeBrowserService
func (s *VolumeBrowserService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error
func (s *VolumeBrowserService) ServiceShutdown() error // calls manager.StopAll() with 5s timeout

// The DTOs below are pure-Go shapes with no corev1 deps so Wails binding gen works.
type SpawnRequestDTO struct {
    ContextName string              `json:"contextName"`
    Namespace   string              `json:"namespace"`
    PVCName     string              `json:"pvcName"`
    Overrides   *SpawnOverridesDTO  `json:"overrides,omitempty"`
}

type SpawnOverridesDTO struct {
    Image                 *string              `json:"image,omitempty"`
    MountPath             *string              `json:"mountPath,omitempty"`
    ReadOnly              *bool                `json:"readOnly,omitempty"`
    ActiveDeadlineSeconds *int64               `json:"activeDeadlineSeconds,omitempty"`
    Resources             *config.ResourceReqs `json:"resources,omitempty"`
    NodeSelector          map[string]string    `json:"nodeSelector,omitempty"`
    Tolerations           []map[string]any     `json:"tolerations,omitempty"`
}

type SpawnResult struct {
    ID        string `json:"id"`
    Namespace string `json:"namespace"`
    PodName   string `json:"podName"`
}

type CollisionError struct {
    ExistingPodName string `json:"existingPodName"`
    ExistingID      string `json:"existingId"`
}

func (s *VolumeBrowserService) Spawn(req SpawnRequestDTO) (SpawnResult, error)
func (s *VolumeBrowserService) Stop(id string) error
func (s *VolumeBrowserService) Replace(id string, req SpawnRequestDTO) (SpawnResult, error) // Stop then Spawn
func (s *VolumeBrowserService) AttachTab(id, tabID string) error
func (s *VolumeBrowserService) ListManaged(contextName string) []ManagedPodDTO
func (s *VolumeBrowserService) ScanOrphans(contextName string) ([]OrphanPodDTO, error)
func (s *VolumeBrowserService) CleanupOrphans(contextName string) error
```

On `Spawn`, before calling `manager.Spawn`, the service:
1. Reads the PVC via the engine — if `status.phase != "Bound"`, return `errors.New("pvc not bound")`.
2. Checks for an existing managed pod for the same `(ctx, ns, pvc)` — if found, return a `*CollisionError`.
3. Resolves config via `config.ResolveForCluster(ctxName)` and merges overrides.
4. Calls `manager.Spawn(...)`.

**TDD:**
- Fake `volumebrowser.Manager` (interface extracted if needed for isolation) covering: bound-check rejects Pending, collision returns `*CollisionError` and does not call `manager.Spawn`, happy path returns `SpawnResult`, `Replace` calls Stop then Spawn in order, `ServiceShutdown` calls `StopAll` within the timeout.

**Verify:**
```bash
go test ./internal/services/ -v
wails3 generate bindings
cd frontend && pnpm check
```
Expected: tests pass, bindings regenerate cleanly, frontend type-check passes (the new `bindings/.../volumebrowserservice.ts` is consumed in later tasks).

**Commit:** `feat(services): add VolumeBrowserService and lifecycle hooks`

---

### Task 4: Settings UI

Expose every `VolumeBrowserConfig` field in both global and per-cluster settings, following the existing metrics/appearance patterns exactly.

**Files:**
- Create: `frontend/src/routes/settings/VolumeBrowserSettings.svelte` — reusable section component accepting `{ bind:value, defaults? }` props so both Global and Cluster settings reuse it
- Modify: `frontend/src/routes/settings/GeneralSettings.svelte` — render `<VolumeBrowserSettings />` under a new "Volume Browser" heading
- Modify: `frontend/src/routes/settings/ClusterSettings.svelte` — render the same component inside the existing per-cluster override UI, using the "override" row pattern already used for metrics
- Modify: `frontend/src/lib/stores/preferences.svelte.ts` — extend the resolved-prefs shape to include `volumeBrowser: VolumeBrowserConfig`
- Create: `frontend/src/routes/settings/__tests__/VolumeBrowserSettings.svelte.test.ts`

**Form fields:**
- Image (text)
- Mount path (text, validated: must start with `/`)
- Read-only by default (checkbox)
- Active deadline seconds (number, empty = no deadline; surfaced as a "Kill after N seconds" toggle with number input)
- Resources block: toggle "Set container resources"; when on, CPU request, CPU limit, memory request, memory limit
- Node selector (KeyValuePairEditor)
- Tolerations (JSON textarea — parse on blur)
- Prompt before spawn (checkbox)
- Orphan cleanup on startup (Select: Prompt / Auto-delete / Ignore)

**Bindings:** use the generated `ConfigService.GetConfig()` / `ConfigService.UpdateConfig(...)` paths (same pattern used by metrics/theme settings today). Per-cluster variant uses whichever existing per-cluster update call that the metrics override uses — read `ClusterSettings.svelte` first and mirror.

**Test:** one Svelte test proving bidirectional binding for at least the tricky fields (`ActiveDeadlineSeconds` nil vs 0, `Resources` nil vs set), plus validation of mount-path prefix.

**Verify:**
```bash
cd frontend && pnpm check && npx vitest run src/routes/settings/__tests__/VolumeBrowserSettings.svelte.test.ts
```

**Commit:** `feat(settings): add Volume Browser settings panel (global + per-cluster)`

---

### Task 5: Spawn UX — store, dialog, entry points

Wire the three entry points (row context menu, detail-drawer action, command palette shortcut) to a single store function, with a Shift-modifier (or config flag) that routes through a pre-filled dialog.

**Files:**
- Create: `frontend/src/lib/stores/volumeBrowser.svelte.ts` — store exposing `spawn(ctx, ns, pvcName, opts?)`; handles collision dialog, drawer open, bottom-panel tab creation
- Create: `frontend/src/lib/components/VolumeBrowserDialog.svelte` — pre-filled override form (same fields as Task 4 but collected into `SpawnOverridesDTO`); emits `submit` with overrides or `cancel`
- Create: `frontend/src/lib/components/VolumeBrowserCollisionDialog.svelte` — Attach / Replace / Cancel buttons
- Modify: `frontend/src/lib/components/ResourceList.svelte` — in the row context menu, when `gvr === "core.v1.persistentvolumeclaims"` and `canMutate`, add a "Browse Volume" item that calls `volumeBrowser.spawn(...)`
- Modify: `frontend/src/lib/components/panels/ActionsToolbar.svelte` — when the displayed resource is a PVC, render a "Browse Volume" button with the same gating
- Modify: `frontend/src/lib/stores/shortcuts.svelte.ts` — register a `browse-volume` shortcut (default unbound) that triggers the store against the currently-selected PVC
- Modify: `frontend/src/lib/components/CommandPalette.svelte` — add a command entry so users can search "Browse Volume" and run it on the focused PVC

**Store API (authoritative):**

```ts
class VolumeBrowserStore {
  async spawn(
    ctxName: string,
    namespace: string,
    pvcName: string,
    opts?: { forceDialog?: boolean }
  ): Promise<void>;
}
export const volumeBrowserStore = new VolumeBrowserStore();
```

Flow inside `spawn`:
1. Resolve prefs for `ctxName` via `preferencesStore`.
2. If `opts.forceDialog || prefs.volumeBrowser.promptBeforeSpawn || shiftHeld`, open `VolumeBrowserDialog` pre-filled from prefs; await overrides or cancel.
3. Call `VolumeBrowserService.Spawn({ contextName, namespace, pvcName, overrides })`.
4. On `CollisionError`, open `VolumeBrowserCollisionDialog`; on Attach, skip to step 6 with the existing pod; on Replace, call `VolumeBrowserService.Replace(existingId, req)`; on Cancel, return.
5. On other errors, push an error toast via `notificationStore` and return.
6. Open the new pod's detail drawer (there is an existing `sessionStore.openTab` / drawer-open helper — read `ResourceList` for the exact call).
7. Add a bottom-panel tab via `bottomPanelStore.addTab({ kind: "terminal-pending", ctxName, gvr: "core.v1.pods", namespace, name: result.podName, resourceKind: "Pod", resourceName: result.podName, obj: {} })`.
8. Call `VolumeBrowserService.AttachTab(result.id, tabId)` so the backend can clean up on tab close.

**Shift detection:** use the existing `event.shiftKey` pattern; where the entry point has no event (command palette), pass `forceDialog: true`.

**Gating:** the button is hidden/disabled whenever `clusterStore.canMutate === false` OR the user lacks `pods:create` in the PVC namespace (extend the existing permissions store if needed; the data is already on `cluster.PermissionSet`).

**TDD:**
- `VolumeBrowserDialog.svelte.test.ts` — renders with pre-filled values, emits `submit` with correct shape, validates required fields.
- `volumeBrowser.svelte.ts` test (mock the binding) — happy path, collision → Replace path, cancel path, error path toasts.

**Verify:**
```bash
cd frontend && pnpm check && npx vitest run
task dev  # manual: click PVC row → Browse Volume; with Shift; via command palette
```

**Commit:** `feat(frontend): add Browse Volume action with dialog and entry points`

---

### Task 6: `terminal-pending` bottom-panel state

Extend `PanelKind`, make `TerminalPanel` handle the two-phase lifecycle (waiting → attached), render errors on stuck-creating pods.

**Files:**
- Modify: `frontend/src/lib/stores/bottom-panel.svelte.ts` — extend `PanelKind` union with `"terminal-pending"`; extend `PanelTab` with optional `managedId?: string`
- Modify: `frontend/src/lib/components/panels/TerminalPanel.svelte` — when `tab.kind === "terminal-pending"`, subscribe to the pod watch (existing `resourceCache` gives this) and:
  - Show a spinner + status line "Waiting for pod <name>: <phase>/<container-state>…" pulled from pod phase + `status.containerStatuses[0].state`
  - If `phase === "Running"` and `containerStatuses[0].ready === true`, flip `tab.kind = "terminal"` and let the existing exec flow take over
  - If `phase === "Pending"` for > 60 seconds AND any `containerStatuses[0].state.waiting.reason` in (`ImagePullBackOff`, `ErrImagePull`, `CreateContainerConfigError`), render an error block with the last 10 events for the pod (reuse `EventsPanel` or inline) + `[Delete & Retry] [Delete]` buttons
  - `[Delete]` calls `VolumeBrowserService.Stop(managedId)` then closes the tab
  - `[Delete & Retry]` calls `VolumeBrowserService.Replace(managedId, lastRequest)` — the store stashes the last request in the tab metadata so retry is possible
- Modify: `frontend/src/lib/components/BottomPanel.svelte` — render `TerminalPanel` for both `terminal` and `terminal-pending` kinds
- Modify: closing a `terminal` or `terminal-pending` tab with a `managedId` must call `VolumeBrowserService.Stop(managedId)` — hook into `bottomPanelStore.closeTab` (add a `beforeClose` hook or intercept at the close-button site)

**Test:**
- Snapshot-style test: given a mocked pod-watch store transitioning Pending→Running+Ready, panel transitions from spinner to exec-terminal mount.
- Given stuck `ImagePullBackOff`, error state renders with both buttons.

**Verify:**
```bash
cd frontend && pnpm check && npx vitest run
task dev  # manual: spawn browser, observe waiting → terminal; kill kubelet / bad image to test error path
```

**Commit:** `feat(terminal-panel): support terminal-pending lifecycle for volume browser`

---

### Task 7: Orphan detection + startup cleanup flow

Glue the startup scan, per-cluster-connect trigger, and notification UI together.

**Files:**
- Modify: `internal/services/cluster.go` — on successful `Connect`, emit a `volumebrowser:orphans:{ctx}` event carrying the scan result list
- Modify: `internal/services/volumebrowser.go` — add `OnClusterConnected(ctxName string)` internal hook (called from cluster service) that runs `ScanOrphans` and, per `OrphanCleanupOnStartup` setting, either auto-cleans, emits the event (prompt), or skips
- Create: `frontend/src/lib/components/OrphanCleanupToast.svelte` — consumes the event, renders a toast with "[Clean up] [Dismiss]" actions; Clean up calls `VolumeBrowserService.CleanupOrphans(ctxName)`
- Modify: `frontend/src/App.svelte` — mount `OrphanCleanupToast` once globally
- Add one backend test covering the three `OrphanCleanupOnStartup` modes.
- Update `CLAUDE.md` / `.wolf/anatomy.md` entries via the OpenWolf flow on file creation.

**Manual test checklist (run before closing the plan):**
- RWO PVC with an attached running pod: browser schedules on same node, attaches.
- RWX PVC: browser schedules anywhere.
- RWO detached PVC: browser schedules, CSI attaches cleanly.
- RBAC-restricted user with no VolumeAttachment read: falls back to pod scan silently.
- Bad image (`alpine:nonexistent`): panel shows error + retry.
- Disconnect cluster with live browser: pod is deleted on the backend.
- Restart the app with a leftover browser pod: orphan toast appears; Clean up removes it.
- Shift-click opens dialog; normal click spawns with resolved config.
- Collision: existing pod + second click → Attach / Replace / Cancel dialog works all three ways.
- Cluster read-only mode: button is disabled with tooltip.

**Verify:**
```bash
go test ./internal/services/ -v
cd frontend && pnpm check && pnpm test
task dev  # work through the manual checklist above
```

**Commit:** `feat(volumebrowser): add startup orphan detection and cleanup UI`

---

## Self-Review

- **Spec coverage check:** goal, non-goals, user flow, collision dialog, error surface, config block, pod spec rules, node discovery, lifecycle tracking, orphan detection, frontend components, error table, testing section — each maps to a task (1: config; 2: spawner/discovery/tracker/orphans/pod spec/node rules; 3: service + error surface + collision error + lifecycle hooks; 4: settings UI; 5: spawn UX + collision dialog + entry points + button gating; 6: terminal-pending + error surface + retry; 7: orphan detection + cleanup UI + manual QA).
- **Placeholder scan:** no TBD/TODO left; every code shape that later tasks depend on is declared (DTOs, labels, pod name format, resolve rules).
- **Type consistency:** `ManagedPod`, `SpawnRequest`, `SpawnResult`, `CollisionError`, `PanelKind` extension, label set, pod name pattern, and `OrphanCleanupOnStartup` values are declared once and reused verbatim in later tasks.
- **Scope:** one cohesive feature, seven coarse tasks matching repo conventions.
