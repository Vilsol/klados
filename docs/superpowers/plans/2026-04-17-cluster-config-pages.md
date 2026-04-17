# Cluster Config Pages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make per-cluster settings discoverable (gear icon on cluster list) and expose Metrics config, read-only cluster info, and Disconnect/Forget actions in the existing `ClusterSettings.svelte` page.

**Architecture:** Extend `cluster.KubeContext` with a source kubeconfig path, add `ClusterService.RemoveKubeconfigPath` and `ClusterService.GetClusterInfo` bindings, then surface it all in a restructured `ClusterSettings.svelte`. No new preference fields — everything routes through existing `ClusterPrefs` / `ResolveForCluster`.

**Tech Stack:** Go 1.25 (client-go, clientcmd), Wails v3 bindings, Svelte 5 runes, Tailwind v4, vitest + testza.

**Spec:** `docs/superpowers/specs/2026-04-17-cluster-config-pages-design.md`

---

## File map

**Backend (create/modify):**
- Modify `internal/cluster/manager.go` — `KubeContext` gains `SourcePath`, `IsDefault`; `LoadKubeconfigs` populates them.
- Modify `internal/cluster/manager_test.go` — unit tests for source-path detection.
- Modify `internal/services/cluster.go` — `RemoveKubeconfigPath`, `GetClusterInfo`, `ClusterInfo` type.
- Modify `internal/services/cluster_test.go` — tests for the two new RPC methods.

**Frontend (create/modify):**
- Modify `frontend/src/routes/ClusterList.svelte` — add gear icon in `rowSuffix` snippet.
- Modify `frontend/src/routes/settings/ClusterSettings.svelte` — add Cluster Info, Metrics, Actions sections.
- Regenerate `frontend/bindings/**` via `wails3 generate bindings` (one-shot after backend lands).

**Docs/artifacts:** none beyond this plan + the spec.

---

## Coding conventions for this plan

- Go tests use `testza`; frontend tests use vitest + `@testing-library/svelte` and must mock `@wailsio/runtime` plus any bindings the component imports (see existing `frontend/src/lib/__tests__/HealthBadge.svelte.test.ts` / `setup.ts`).
- Commit after each task using the `jj-vcs` skill (per repo CLAUDE.md — this is a Jujutsu repo, never use raw git).
- Log events through slox (context-carried). Struct fields store `ctx context.Context`, not `*slog.Logger`.
- Wails binding imports use `.js` extension; regenerate after every exported Go signature change.

---

## Task 1: Track source kubeconfig path per context

**Files:**
- Modify: `internal/cluster/manager.go`
- Modify: `internal/cluster/manager_test.go`

**What to do**

1. Extend `KubeContext` with two new fields:

```go
type KubeContext struct {
    Name          string           `json:"name"`
    Cluster       string           `json:"cluster"`
    User          string           `json:"user"`
    Namespace     string           `json:"namespace"`
    Status        ConnectionStatus `json:"status"`
    ServerVersion string           `json:"serverVersion"`
    Provider      string           `json:"provider"`
    SourcePath    string           `json:"sourcePath"`
    IsDefault     bool             `json:"isDefault"`
}
```

2. Rewrite `LoadKubeconfigs(extraPaths []string)` so provenance is captured. The merged `clientcmd` config loses file origin, so walk each candidate file individually:

   - Capture `defaultPaths := clientcmd.NewDefaultClientConfigLoadingRules().Precedence` (compute once, before appending extras — use to fill `IsDefault`).
   - Build the merged config the current way (for `m.rawConfig`).
   - Separately, iterate `append(defaultPaths, extraPaths...)` in order. For each existing readable file, load it with `clientcmd.LoadFromFile` and record the first file that contains each context name in a `map[string]sourceEntry{path, isDefault}`.
   - When building `m.contexts`, set `SourcePath` / `IsDefault` from that map. If a name wasn't found via per-file walk (shouldn't normally happen), leave both zero-valued.
   - Non-existent paths in the precedence list are skipped silently (matches clientcmd behaviour).

3. Add tests to `manager_test.go` covering:

   - A context defined only in an extra path carries that path as `SourcePath` and `IsDefault=false`.
   - A context defined in the default kubeconfig (simulate by pointing `KUBECONFIG` env at a tmp file for the test — see existing `TestLoadKubeconfigs` helpers) gets `IsDefault=true`.
   - Precedence: same context name defined in two files → the earlier-precedence file wins.

   Use `testza` assertions like the existing tests in this file. Use `t.TempDir()` + `os.Setenv("KUBECONFIG", …)` per test; restore afterward.

**Verify**

Run: `go test ./internal/cluster/ -run 'TestLoadKubeconfigs' -v`
Expected: all new tests pass; existing tests still pass.

**Commit** via `jj-vcs` skill: `cluster: track source kubeconfig path per context`

---

## Task 2: ClusterService RemoveKubeconfigPath + GetClusterInfo

**Files:**
- Modify: `internal/services/cluster.go`
- Modify: `internal/services/cluster_test.go`
- Possibly modify: `internal/cluster/manager.go` (add `RawConfig()` accessor if missing)

**What to do**

1. Add `RemoveKubeconfigPath` at the bottom of `cluster.go`, mirroring `AddKubeconfigPath` style:

```go
func (c *ClusterService) RemoveKubeconfigPath(path string) ([]cluster.KubeContext, error) {
    for _, p := range clientcmd.NewDefaultClientConfigLoadingRules().Precedence {
        if p == path {
            return nil, fmt.Errorf("cannot forget default kubeconfig path")
        }
    }

    cfg := c.appService.Config()
    if err := cfg.Update(func(cfg *config.Config) {
        filtered := cfg.KubeconfigPaths[:0]
        for _, p := range cfg.KubeconfigPaths {
            if p != path {
                filtered = append(filtered, p)
            }
        }
        cfg.KubeconfigPaths = append([]string(nil), filtered...)
    }); err != nil {
        return nil, err
    }

    if err := c.manager().LoadKubeconfigs(cfg.KubeconfigPaths); err != nil {
        return nil, err
    }

    newContexts := c.manager().ListContexts()
    alive := make(map[string]struct{}, len(newContexts))
    for _, kc := range newContexts {
        alive[kc.Name] = struct{}{}
    }
    _ = cfg.Update(func(cfg *config.Config) {
        for name := range cfg.Clusters {
            if _, ok := alive[name]; !ok {
                delete(cfg.Clusters, name)
            }
        }
    })

    return newContexts, nil
}
```

2. Add a `ClusterInfo` struct + `GetClusterInfo` method:

```go
type CapabilityState string

const (
    CapabilityAvailable   CapabilityState = "available"
    CapabilityUnavailable CapabilityState = "unavailable"
    CapabilityUnknown     CapabilityState = "unknown"
)

type ClusterInfo struct {
    Context          cluster.KubeContext `json:"context"`
    ServerURL        string              `json:"serverUrl"`
    MetricsServer    CapabilityState     `json:"metricsServer"`
    PrometheusURL    string              `json:"prometheusUrl"`
    PrometheusSource string              `json:"prometheusSource"` // "detected" | "configured" | ""
}

func (c *ClusterService) GetClusterInfo(ctxName string) (ClusterInfo, error) {
    info := ClusterInfo{MetricsServer: CapabilityUnknown}

    var found *cluster.KubeContext
    for _, kc := range c.manager().ListContexts() {
        if kc.Name == ctxName {
            kcCopy := kc
            found = &kcCopy
            break
        }
    }
    if found == nil {
        return info, fmt.Errorf("context not found: %s", ctxName)
    }
    info.Context = *found

    if raw := c.manager().RawConfig(); raw != nil {
        if kctx, ok := raw.Contexts[ctxName]; ok {
            if clst, ok := raw.Clusters[kctx.Cluster]; ok {
                info.ServerURL = clst.Server
            }
        }
    }

    resolved := c.appService.Config().ResolveForCluster(ctxName)
    if resolved.Metrics != nil && resolved.Metrics.PrometheusURL != "" {
        info.PrometheusURL = resolved.Metrics.PrometheusURL
        info.PrometheusSource = "configured"
    }

    conn, err := c.manager().GetConnection(ctxName)
    if err == nil && conn.Status == cluster.StatusConnected {
        switch conn.MetricsCapability {
        case metrics.MetricsCapabilityAvailable:
            info.MetricsServer = CapabilityAvailable
        case metrics.MetricsCapabilityUnavailable:
            info.MetricsServer = CapabilityUnavailable
        default:
            info.MetricsServer = CapabilityUnknown
        }
        if info.PrometheusURL == "" {
            if url := metrics.DetectPrometheus(c.ctx, conn.Clientset, conn.Dynamic); url != "" {
                info.PrometheusURL = url
                info.PrometheusSource = "detected"
            }
        }
    }
    return info, nil
}
```

3. If `Manager.RawConfig()` does not already exist, add a small accessor in `cluster/manager.go`:

```go
func (m *Manager) RawConfig() *clientcmdapi.Config {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.rawConfig
}
```

4. Confirm `metrics.DetectPrometheus` signature and `metrics.MetricsCapability*` constant names before committing (`internal/metrics/detect.go`, `internal/metrics/provider.go`) — adjust if the exported names differ.

5. Tests in `cluster_test.go`:

   - `TestRemoveKubeconfigPath_RemovesAndReloads`: add a path, call Remove, assert path is gone, manager's contexts list shrinks.
   - `TestRemoveKubeconfigPath_NoOpWhenAbsent`: removing unknown path succeeds, list unchanged.
   - `TestRemoveKubeconfigPath_RejectsDefault`: returns error when path is in default precedence.
   - `TestRemoveKubeconfigPath_PrunesClusterPrefs`: seed `config.Clusters[name]`, remove the file defining that context, assert entry was deleted from config.
   - `TestGetClusterInfo_Disconnected`: populates Context + ServerURL; MetricsServer is `CapabilityUnknown`; PrometheusURL empty unless configured.
   - `TestGetClusterInfo_ConfiguredPrometheus`: sets `ClusterPrefs.Metrics.PrometheusURL`; `GetClusterInfo` returns `PrometheusSource="configured"` and that URL.

   Use the existing fake/stub provider patterns from `cluster_test.go`. Do not invent new test infra.

**Verify**

Run: `go test ./internal/services/ -run 'ClusterService|RemoveKubeconfigPath|GetClusterInfo' -v`
Expected: all new tests pass.

**Commit:** `services: cluster RemoveKubeconfigPath + GetClusterInfo`

---

## Task 3: Regenerate Wails bindings

**Files:**
- Regenerate: `frontend/bindings/**`

**What to do**

Run from repo root:

```bash
wails3 generate bindings
```

Inspect the diff — new entries expected:
- `frontend/bindings/github.com/Vilsol/klados/internal/services/clusterservice.js` gains `RemoveKubeconfigPath`, `GetClusterInfo`.
- `frontend/bindings/github.com/Vilsol/klados/internal/cluster/models.js` — `KubeContext` gets `sourcePath`, `isDefault`.
- A `ClusterInfo` / `CapabilityState` model appears under `internal/services/` models.

Run `cd frontend && pnpm check` to confirm no TypeScript errors surface downstream.

**Commit:** `bindings: regenerate for cluster info + forget`

---

## Task 4: ClusterList gear icon

**Files:**
- Modify: `frontend/src/routes/ClusterList.svelte`
- Create: `frontend/src/routes/__tests__/ClusterList.gear.svelte.test.ts`

**What to do**

1. Import `Settings` from `lucide-svelte` alongside the existing icons:

```ts
import {Columns3, Unplug, Settings} from "lucide-svelte";
```

2. Inside the existing `rowSuffix` snippet, add a gear button next to the disconnect button. Gear is **always visible** (not gated by status). Stop propagation so the row click doesn't also fire:

```svelte
{#snippet rowSuffix({ item: ctx })}
  {@const status = clusterStore.connectionStatus[ctx.name] ?? "disconnected"}
  <div class="flex items-center justify-end gap-1">
    <button
      type="button"
      onclick={(e) => { e.stopPropagation(); push(`/settings/clusters/${encodeURIComponent(ctx.name)}`) }}
      class="p-1 rounded opacity-0 group-hover:opacity-60 hover:!opacity-100 transition-all"
      title="Cluster settings"
      aria-label="Settings for {ctx.name}"
    >
      <Settings size={13} />
    </button>
    {#if status !== "disconnected"}
      <button …>…existing disconnect button…</button>
    {/if}
  </div>
{/snippet}
```

3. Test: new file `ClusterList.gear.svelte.test.ts` that mounts `ClusterList`, renders one fake context via the `clusterStore` mock, clicks the "Settings for …" button, and asserts `push` was called with the expected path. Model after existing `HealthBadge.svelte.test.ts`; mock `svelte-spa-router`, `@wailsio/runtime`, and every binding `ClusterList` imports.

   If the mocking footprint is disproportionate, an acceptable lighter assertion is: gear button exists per rendered row with the correct `aria-label`. Do not skip the test entirely.

**Verify**

Run:
```bash
cd frontend && pnpm test -- ClusterList.gear
```
Expected: new test passes.

Manual smoke: `task dev`, load the cluster list, hover a row, click the gear → lands on `/settings/clusters/<name>`.

**Commit:** `cluster-list: gear icon links to cluster settings`

---

## Task 5: ClusterSettings — Cluster Info section

**Files:**
- Modify: `frontend/src/routes/settings/ClusterSettings.svelte`

**What to do**

1. Import the new binding and types:

```ts
import {GetClusterInfo} from "../../../bindings/github.com/Vilsol/klados/internal/services/clusterservice.js";
import type {ClusterInfo} from "../../../bindings/github.com/Vilsol/klados/internal/services/models.js";
```

(Exact path for `ClusterInfo` depends on where `wails3 generate` placed it — verify and adjust.)

2. Add state and loader:

```ts
let info = $state<ClusterInfo | null>(null);

onMount(() => {
  (async () => {
    // existing prefs load stays as-is
    try {
      info = await GetClusterInfo(ctxName);
    } catch (e) {
      console.warn("GetClusterInfo failed", e);
    }
  })();
});
```

3. Insert a **new top section** above the existing Display Name section, rendered only when `info` is populated:

```svelte
{#if info}
  <section class="space-y-2">
    <h3 class="text-sm font-semibold text-fg">Cluster Info</h3>
    <dl class="grid grid-cols-[140px_1fr] gap-y-1 text-sm">
      <dt class="text-muted">Context</dt><dd class="text-fg">{info.context.name}</dd>
      <dt class="text-muted">Cluster</dt><dd class="text-fg">{info.context.cluster || "—"}</dd>
      <dt class="text-muted">User</dt><dd class="text-fg">{info.context.user || "—"}</dd>
      <dt class="text-muted">Default namespace</dt><dd class="text-fg">{info.context.namespace || "—"}</dd>
      <dt class="text-muted">Server URL</dt><dd class="text-fg break-all">{info.serverUrl || "—"}</dd>
      {#if info.context.serverVersion}
        <dt class="text-muted">Server version</dt><dd class="text-fg">{info.context.serverVersion}</dd>
      {/if}
      <dt class="text-muted">Kubeconfig</dt>
      <dd class="text-fg break-all">
        {info.context.sourcePath || "—"}
        {#if info.context.isDefault}<span class="ml-1 text-xs text-muted">(default)</span>{/if}
      </dd>
      <dt class="text-muted">Status</dt><dd class="text-fg">{statusLabel(info.context.status)}</dd>
      <dt class="text-muted">metrics-server</dt><dd class="text-fg">{info.metricsServer}</dd>
      <dt class="text-muted">Prometheus</dt>
      <dd class="text-fg break-all">
        {#if info.prometheusUrl}
          {info.prometheusUrl}
          <span class="ml-1 text-xs text-muted">({info.prometheusSource})</span>
        {:else}
          not detected
        {/if}
      </dd>
    </dl>
  </section>
{/if}
```

4. Add a local helper inside `<script>`:

```ts
function statusLabel(s: number | string): string {
  const map: Record<number, string> = {0: "disconnected", 1: "connecting", 2: "connected", 3: "error"};
  return typeof s === "number" ? (map[s] ?? "unknown") : String(s);
}
```

   (If the generated binding serializes status as a string, drop the helper and render directly. Inspect the generated model first.)

**Verify**

Manual: `task dev`, open Settings → Clusters → pick any cluster → info block renders. Try both connected and disconnected states; assert capability lines update after connecting.

**Commit:** `cluster-settings: add read-only cluster info section`

---

## Task 6: ClusterSettings — Metrics section

**Files:**
- Modify: `frontend/src/routes/settings/ClusterSettings.svelte`

**What to do**

1. Add state for the Prometheus override and wire it to existing `save()`:

```ts
let prometheusUrl = $state<string>("");
```

   In the existing `onMount` prefs loader, hydrate it: `prometheusUrl = prefs.metrics?.prometheusUrl ?? "";`

2. Extend `save()` to include the metrics field:

```ts
metrics: prometheusUrl ? { prometheusUrl } : undefined,
```

3. Render the Metrics section under the existing Favorite Namespaces section:

```svelte
<section class="space-y-2">
  <h3 class="text-sm font-semibold text-fg">Metrics</h3>
  <label class="block text-sm font-medium text-fg mb-1">
    Prometheus URL
    <input
      type="text"
      value={prometheusUrl}
      oninput={(e) => { prometheusUrl = (e.target as HTMLInputElement).value; save(); }}
      placeholder={info?.prometheusUrl && info.prometheusSource === "detected" ? info.prometheusUrl : "https://prometheus.example/api/v1"}
      class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
    >
  </label>
  <p class="text-xs text-muted">
    {#if info?.prometheusUrl}
      Effective: {info.prometheusUrl} <span class="text-muted">({info.prometheusSource})</span>
    {:else}
      No Prometheus endpoint detected or configured.
    {/if}
  </p>
</section>
```

4. Confirm the generated `ClusterPrefs` model accepts the field as a nested object (check `frontend/bindings/.../config/models.js`). If there's a class constructor (e.g. `new MetricsConfig(...)`), use it for consistency with existing `SetClusterPrefs` call sites.

**Verify**

Manual: set a Prometheus URL in the input → it persists across a refresh. Clear it → effective line falls back to detected or "not detected".

**Commit:** `cluster-settings: add metrics (prometheus url) section`

---

## Task 7: ClusterSettings — Actions section (Disconnect / Forget)

**Files:**
- Modify: `frontend/src/routes/settings/ClusterSettings.svelte`

**What to do**

1. Add imports:

```ts
import {push} from "svelte-spa-router";
import {Disconnect, RemoveKubeconfigPath} from "../../../bindings/github.com/Vilsol/klados/internal/services/clusterservice.js";
import ConfirmDialog from "$lib/components/ConfirmDialog.svelte"; // verify path; else packages/ui version
```

   Check how other settings pages import dialogs; use whichever path is already in use in this repo rather than inventing one.

2. Add state:

```ts
let forgetConfirmOpen = $state(false);
```

3. Reactive helpers:

```ts
const canForget = $derived(Boolean(info && info.context.sourcePath && !info.context.isDefault));
const isConnected = $derived(info?.context.status === 2); // adjust if status is stringified
```

4. Render the Actions section last:

```svelte
<section class="space-y-3 pt-6 border-t border-destructive/30">
  <h3 class="text-sm font-semibold text-destructive">Actions</h3>

  <div class="flex flex-col gap-2">
    <button
      type="button"
      disabled={!isConnected}
      onclick={async () => { try { await Disconnect(ctxName); info = await GetClusterInfo(ctxName); } catch (e) { console.warn(e); } }}
      class="self-start px-3 py-1.5 text-sm rounded border border-border hover:bg-surface-hover disabled:opacity-40 disabled:cursor-not-allowed"
    >
      Disconnect
    </button>

    {#if canForget}
      <button
        type="button"
        onclick={() => forgetConfirmOpen = true}
        class="self-start px-3 py-1.5 text-sm rounded border border-destructive/50 text-destructive hover:bg-destructive/10"
      >
        Forget cluster
      </button>
    {/if}
  </div>
</section>

<ConfirmDialog
  bind:open={forgetConfirmOpen}
  title="Forget cluster?"
  message={`This removes all contexts defined in ${info?.context.sourcePath} from Klados. Your kubeconfig file is not modified.`}
  confirmLabel="Forget"
  destructive
  onconfirm={async () => {
    if (!info) return;
    try {
      await RemoveKubeconfigPath(info.context.sourcePath);
      forgetConfirmOpen = false;
      push("/");
    } catch (e) {
      console.warn("forget failed", e);
    }
  }}
/>
```

   Inspect `ConfirmDialog.svelte` for the exact prop names (`open`, `title`, `message`, `onconfirm`, `destructive`) and adjust — match existing call sites if they differ.

5. After a successful Disconnect or Forget, the cluster list store must refresh. If there is an event (`cluster:contexts:updated`) that `clusterStore` already listens for, no extra work is needed — confirm by grepping. Otherwise, call `clusterStore.loadContexts()` manually after the mutation.

**Verify**

Manual:
- Open a cluster you've imported via `KubeconfigImportDialog` → Actions shows both buttons; Disconnect is enabled only when connected; clicking Forget shows confirm, confirming navigates to `/` and the cluster disappears from the list.
- Open a cluster from the default kubeconfig → Forget button is hidden; Disconnect still works.

**Commit:** `cluster-settings: add disconnect + forget actions`

---

## Task 8: Polish pass & final verify

**Files:** any of the above as needed.

**What to do**

1. Run the full local test battery:

   ```bash
   go test ./internal/cluster/ ./internal/services/ ./internal/config/ -v
   cd frontend && pnpm check && pnpm test
   ```

2. Boot `task dev` and walk the full flow once:
   - Gear icon on cluster list → lands on settings page.
   - Info section shows correct source path and "(default)" tag for default contexts.
   - Metrics URL persists across reload; helper line matches detection/configuration.
   - Disconnect clears connection state; info block updates.
   - Forget removes manually-added cluster and all its contexts; bounces to `/`.

3. Fix any visual/behavioral gaps discovered during the walk. Likely candidates:
   - Dark-mode contrast on the Actions section.
   - Info loader race when `ctxName` prop changes while the page is mounted (wrap the loader in `$effect(() => { … info fetch … })` keyed on `ctxName` if you find the page is reused across contexts).
   - `push('/')` firing before `clusterStore.loadContexts()` completes — await it if needed.

4. Update `.wolf/memory.md` with a one-line entry per OpenWolf protocol.

**Commit:** `cluster-settings: polish pass (info refresh, a11y, dark mode)`

---

## Self-review notes (author)

- **Spec coverage:** every section of the spec maps to Task 1-7; Task 8 is the integration sanity check the spec implies.
- **Type consistency:** `ClusterInfo`, `CapabilityState`, `sourcePath`, `isDefault`, `prometheusUrl`, `prometheusSource` used identically across Go struct tags, spec, and Svelte bindings.
- **Known risk:** `ConnectionStatus` JSON shape (int vs string). Task 5 calls this out — inspect the generated binding before committing.
- **Known risk:** `metrics.DetectPrometheus` / `MetricsCapability*` exact names. Task 2 flags this — confirm before committing.
