# PVC Volume Browser — Design

**Status:** draft
**Date:** 2026-04-19

## Goal

Give users a one-click way to open an ephemeral shell with any PVC mounted. The button spawns a helper pod in the PVC's namespace, opens the new pod's detail drawer, and opens a bottom-panel terminal tab that attaches the moment the pod becomes ready. For RWO/ROX volumes currently attached to a node, the helper pod is scheduled onto that same node to avoid `Multi-Attach` errors.

## Non-Goals

- Bulk browsing (one pod per click).
- Editing volume contents through a UI (use the shell).
- File-transfer UI — out of scope for v1; the shell covers `cp`/`tar`.
- Managing helper pods created outside Klados.

## User-Facing Flow

1. User finds a PVC via list, drawer, or command palette.
2. User clicks **Browse Volume** (row context menu, detail-drawer action, or keyboard shortcut).
3. If `Shift` is held or `PromptBeforeSpawn` is enabled, a dialog appears pre-filled from the resolved config. The user can tweak image, mount path, read-only flag, resources, node selector, tolerations, and deadline for this one invocation.
4. Klados creates the helper pod and returns immediately with `{podName, namespace}`.
5. The frontend:
   - opens the `ResourceDetail` drawer for the new pod, and
   - opens a bottom-panel tab in `terminal-pending` mode.
6. The terminal panel watches the pod; when phase is `Running` and the container is `Ready`, it transitions to the normal exec session.
7. Closing the terminal tab deletes the pod.

### Collision: an existing managed pod is already present for this PVC

A dialog offers **Attach / Replace / Cancel**:
- Attach: skip creation, reuse the existing pod.
- Replace: delete the existing pod, then create a fresh one.
- Cancel: abort.

### Error surface

- PVC is not `Bound` → toast error, no pod created.
- `canMutate === false` or `Pods:create` RBAC denied in that namespace → button disabled, tooltip explains why.
- Image pull fails or pod stuck `ContainerCreating > 60s` → terminal panel shows the events tail with **[Delete & Retry] [Delete]**.

## Architecture

```
PVC row / drawer / shortcut
        │
        ▼
frontend: volumeBrowser store
  ├─ optional Shift-override dialog
  └─ VolumeBrowserService.Spawn(ctx, pvcNs, pvcName, overrides?)
        │
        ▼
internal/services/volumebrowser.go     (thin Wails service)
        │
        ▼
internal/volumebrowser/                (domain logic)
  ├─ spawner.go     — builds Pod spec, resolves node, creates pod
  ├─ discovery.go   — VolumeAttachment lookup + pod-scan fallback
  ├─ tracker.go     — in-memory map of managed pods per context
  ├─ orphans.go     — startup scan for pods from prior sessions
  └─ cleanup.go     — deletes pods on tab close / disconnect / shutdown
        │
        ▼
Returns {podName, namespace, managedId}; emits volumebrowser:{ctx}:{id}
        │
        ▼
frontend:
  ├─ opens ResourceDetail drawer for the new pod
  └─ opens bottom-panel terminal tab in "waiting for ready" state
                    │
                    ▼ pod watch fires: phase=Running, containerReady=true
                    │
                    ▼ auto-attach via existing ExecService
```

This mirrors the layout of `internal/portforward`, `internal/exec`, and `internal/logs`: a self-contained domain package with a thin Wails service wrapper.

## Configuration

New block in `internal/config/config.go`, resolvable per-cluster through `internal/config/resolve.go`:

```go
type VolumeBrowserConfig struct {
    Image                  string              // default "alpine:edge"
    MountPath              string              // default "/mnt/volume"
    ReadOnly               bool                // default false
    ActiveDeadlineSeconds  *int64              // default ptr(3600); nil = no deadline
    Resources              *ResourceReqs       // nil = no resource requests/limits set on the container
    NodeSelector           map[string]string   // default empty
    Tolerations            []corev1.Toleration // default empty
    PromptBeforeSpawn      bool                // default false; Shift always prompts regardless
    OrphanCleanupOnStartup string              // "prompt" | "auto" | "ignore"; default "prompt"
}

type ResourceReqs struct {
    Requests map[string]string // e.g. {"cpu": "10m", "memory": "32Mi"}
    Limits   map[string]string
}
```

Settings UI:
- **Global:** new "Volume Browser" section in `GeneralSettings.svelte`.
- **Per-cluster override:** corresponding section in `ClusterSettings.svelte`, using the existing override-row pattern.

## Pod Spec

- **Name:** `klados-pvc-<pvcname-truncated-to-40>-<4hex>` (deterministic prefix so users spot klados pods in `kubectl`).
- **Namespace:** same as the PVC (required — PVCs are namespaced).
- **Labels:**
  - `app.kubernetes.io/managed-by=klados`
  - `klados.io/purpose=pvc-browser`
  - `klados.io/pvc=<name>`
  - `klados.io/session=<uuid>` — per-app-start UUID, used by orphan detection to distinguish "this session's pods" from "previous session's pods"
- **Annotations:**
  - `klados.io/created-at=<RFC3339>`
  - `klados.io/created-by=<os.Hostname>/<os user>`
- **Container:**
  - `image` from resolved config
  - `command: ["sh","-c","trap 'exit 0' TERM; sleep infinity & wait"]` — clean shutdown on SIGTERM so `terminationGracePeriodSeconds: 0` finishes immediately
  - `resources` from config (omitted entirely when `Resources == nil`)
  - `securityContext`: no `privileged`, no extra capabilities, `runAsNonRoot: false` (many volumes have root-owned data)
  - one `volumeMount` at the configured path with the configured `readOnly`
- **Pod spec:**
  - `restartPolicy: Never`
  - `terminationGracePeriodSeconds: 0`
  - `activeDeadlineSeconds` from config (nil = omitted)
  - `nodeSelector` / `tolerations` from config
  - `nodeName` set directly from discovery result for RWO (faster than affinity rules; bypasses scheduler corner cases)
  - one `volume` of type `persistentVolumeClaim` referencing the source PVC

## Node Discovery

```
resolveNode(ctx, pvc):
  if pvc.accessModes contains RWX:
    return ""                             // any node is fine
  try VolumeAttachment list:
    find where spec.source.persistentVolumeName == pvc.spec.volumeName
    if found: return spec.nodeName
  (if RBAC denied, fall through silently)
  list pods in pvc.namespace:
    keep those whose .spec.volumes[*].persistentVolumeClaim.claimName == pvc.name
    filter to phase == Running
    if any: return .spec.nodeName of first
  return ""                               // detached volume — schedule anywhere
```

## Lifecycle Tracking

`internal/volumebrowser/tracker.go` holds:

```go
type ManagedPod struct {
    ID            string    // UUID
    ContextName   string
    Namespace     string
    PodName       string
    PVCName       string
    CreatedAt     time.Time
    TerminalTabID string    // set by frontend after tab creation
}

type Tracker struct {
    mu    sync.RWMutex
    items map[string][]*ManagedPod  // key = contextName
}
```

Cleanup triggers:
- Terminal tab closed → service calls `tracker.Remove(id)` and deletes pod.
- Cluster disconnected → delete all tracked pods for that context.
- App shutdown (`ServiceShutdown`) → delete all tracked pods across contexts, best-effort with a short timeout.

## Orphan Detection

On cluster connect, `orphans.go` runs once per context:

1. List pods cluster-wide with label selector `klados.io/purpose=pvc-browser`.
2. Filter to those whose `klados.io/session` label does NOT equal the current app-session UUID.
3. If any found, honour `OrphanCleanupOnStartup`:
   - `prompt` (default) → single toast: "Found N leftover Klados browser pods from a previous session. [Clean up] [Dismiss]"
   - `auto` → delete them silently; log one info line.
   - `ignore` → do nothing.

## Frontend Components

New files:

- `frontend/src/lib/stores/volumeBrowser.svelte.ts` — small store exposing `spawn(ctx, ns, name, overrides?)`, plus the "pending managed pods" list used by the terminal panel's waiting state.
- `frontend/src/lib/components/VolumeBrowserDialog.svelte` — pre-filled override dialog (image, mount path, RO, resources, selectors, deadline).
- Extension to `frontend/src/lib/components/panels/TerminalPanel.svelte` — handles the `terminal-pending` phase: watches the pod, shows "Waiting for pod to be ready…" with a live status line (pulled from pod phase + container state), transitions to the existing exec flow when ready, renders an error block with event tail on timeout.

Wiring into existing UI:

- `ResourceList.svelte` row context menu adds **Browse Volume** when `gvr === "core.v1.persistentvolumeclaims"`, gated by `canMutate`.
- `ResourceDetail.svelte` actions area shows a **Browse Volume** button with the same gating.
- `shortcutStore` registers `Browse Volume` command, default unbound; appears in the command palette.

Bottom-panel tab kind is extended:

```ts
export type PanelKind = "logs" | "terminal" | "terminal-pending" | "aggregate-logs" | "yaml";
```

`terminal-pending` tabs carry an extra `managedId` field referencing the `ManagedPod`.

## Error Handling

| Case | Behaviour |
|---|---|
| PVC not `Bound` | Service returns error; toast "PVC is not bound — nothing to mount"; no pod created. |
| `canMutate === false` | Button hidden/disabled with tooltip. |
| `Pods:create` RBAC denied in PVC namespace | Button disabled with tooltip "You don't have permission to create Pods in `<ns>`". |
| Existing managed pod for same PVC in cluster | Dialog: Attach / Replace / Cancel. |
| `VolumeAttachment` list RBAC denied | Silent fallback to pod-scan. |
| Pod stuck `ContainerCreating` > 60s | Terminal panel shows events tail with [Delete & Retry] [Delete]. |
| Image pull fails | Same as above — surfaced through the pod's events. |
| Cluster disconnected while pod running | Tracker deletes pod on the disconnect hook. |
| App crashed without cleanup | `activeDeadlineSeconds` terminates the pod; next startup's orphan scan offers cleanup. |

## Testing

Go:

- `internal/volumebrowser/spawner_test.go` — pod spec generation: labels/annotations/mount path/readOnly/resources-omitted-when-nil/nodeName-when-set.
- `internal/volumebrowser/discovery_test.go` — table-driven with fake dynamic client: RWX short-circuit, VolumeAttachment hit, VolumeAttachment RBAC-denied → pod-scan hit, pod-scan empty, all-empty detached case.
- `internal/volumebrowser/tracker_test.go` — concurrent Add/Remove, per-context isolation, disconnect flush.
- `internal/volumebrowser/orphans_test.go` — session UUID filter correctness; label-selector list.
- `internal/services/volumebrowser_test.go` — bound-check, existing-pod collision dispatch, error paths surface through the service.

Frontend:

- `VolumeBrowserDialog.svelte.test.ts` — renders, validates inputs, emits submit with overrides.
- `TerminalPanel` waiting-state transition test — given a mocked pod watch, panel transitions to exec when `Running + ready`.

Manual integration:

- RWO with an attached running pod → helper pod schedules on same node and attaches.
- RWX → helper pod schedules anywhere.
- RWO detached → helper pod schedules anywhere and the CSI attaches cleanly.
- RBAC-restricted user (no VolumeAttachment read) → falls back to pod scan.
- Image pull failure → error surface works; Retry works.
- Disconnect with live helper pod → pod is deleted.
- Restart app with a leftover pod → orphan toast appears.

## Decisions Log

- **Why `spec.nodeName` instead of `nodeAffinity`?** Faster (no scheduler round trip), deterministic, and avoids corner cases where a drain/cordon could still land the pod elsewhere. We already know the exact node from discovery.
- **Why mount RW by default?** The common case is "let me poke at this volume" which often needs writes; the Shift-dialog RO path is one keypress away.
- **Why `alpine:edge`?** Best ergonomics-to-size ratio; `apk add` covers anything missing. `edge` (not `3`) picks up newer tooling the user explicitly asked for.
- **Why a new service instead of extending `ResourceService`?** `ResourceService` is already ~5.6K tokens; the `portforward`/`exec`/`logs` pattern is the established idiom for long-lived resource lifecycles.
- **Why a session UUID in labels?** So orphan detection can distinguish "this process's pods" from "pods from a prior crashed session" without timestamp guesswork.
