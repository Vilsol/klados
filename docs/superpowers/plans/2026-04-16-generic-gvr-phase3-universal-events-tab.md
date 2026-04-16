# Generic GVR Phase 3 — Universal Events Tab Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the `"events"` detail panel available for every GVR, fetching and displaying events that reference the current resource via `involvedObject`. Built-in descriptors that currently lack it have it added; auto-generated descriptors from Phase 2 already include it.

**Architecture:** `EventsPanel.svelte` already exists and is used by Pods/Deployments — it fetches via `GetEvents(ctx, ns, uid)`. The panel is already wired into the descriptor-to-component map in `ResourceDetail.svelte`. This phase:
1. Adds the `"events"` panel to any built-in descriptor that omits it.
2. Ensures cluster-scoped resources query events across all namespaces.
3. Adds a real-time watch subscription so new events appear while the tab is open.

**Tech Stack:** Go (`ResourceService.GetEvents`), Svelte 5, Wails events, vitest.

**Depends on:** Phases 1-2 complete.

**Reference spec:** `docs/superpowers/specs/2026-04-16-generic-gvr-capabilities-design.md` §4.

---

## File Structure

- Modify: `internal/resource/builtin.go` — add `"events"` to any built-in descriptor missing it
- Modify: `internal/services/resource.go` — extend `GetEvents` to support cluster-scoped (empty namespace) lookups
- Modify: `frontend/src/lib/components/panels/EventsPanel.svelte` — handle cluster-scoped resources; add watch subscription; severity color coding
- Modify: `frontend/src/lib/__tests__/EventsPanel.svelte.test.ts` (create if absent) — unit tests for the panel
- Regenerate: Wails bindings if `GetEvents` signature changes

---

## Task 1: Audit which built-in descriptors lack "events"

**Files:**
- Reference: `internal/resource/builtin.go`

- [ ] **Step 1: List descriptors and their detailPanels**

Run: `grep -n 'DetailPanels:\|"events"' internal/resource/builtin.go | head -80`
Inspect the output to identify descriptors whose `DetailPanels` slice does NOT contain `"events"`.

- [ ] **Step 2: Record the list**

Create a short note in the implementation branch (not committed): every GVR missing `"events"`. Typical candidates: ConfigMaps, Secrets, ServiceAccounts, RoleBindings, NetworkPolicies, ResourceQuotas, LimitRanges, CRDs themselves. (The spec doesn't require ALL built-ins to gain Events — some resources like Secrets genuinely don't emit events — but the spec says "universal". Proceed with adding to all unless the resource is cluster-scoped system meta like Node.)

- [ ] **Step 3: No commit yet — this task produces the edit list**

---

## Task 2: Add "events" to missing built-in descriptors

**Files:**
- Modify: `internal/resource/builtin.go`

- [ ] **Step 1: Add `"events"` to `DetailPanels` for each missing descriptor**

For each descriptor identified in Task 1, add `"events"` to the `DetailPanels` slice. Place it after `"overview"` and before `"yaml"`. Example edit pattern:

Before:
```go
DetailPanels: []string{"overview", "labels", "yaml"},
```

After:
```go
DetailPanels: []string{"overview", "labels", "events", "yaml"},
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/resource/`
Expected: exits 0.

- [ ] **Step 3: Run existing tests**

Run: `go test ./internal/resource/ -v`
Expected: all PASS (builtin descriptors have validation tests for CEL expressions which remain unchanged).

- [ ] **Step 4: Commit**

```bash
jj desc -m "resource: add Events tab to all built-in descriptors"
```

---

## Task 3: Support cluster-scoped events in ResourceService.GetEvents

**Files:**
- Modify: `internal/services/resource.go` — `GetEvents` method
- Modify: `internal/services/resource_test.go`

- [ ] **Step 1: Read current implementation**

Find the current `GetEvents(contextName, namespace, uid string)` method in `internal/services/resource.go`. Observe how it queries events today — likely a field-selector like `involvedObject.uid=<uid>` against `core.v1.events` in the given namespace.

- [ ] **Step 2: Write failing test**

Append to `internal/services/resource_test.go` (use the existing test patterns in that file — they already have a fake clientset setup; reuse it):

```go
func TestGetEvents_ClusterScoped_SearchesAllNamespaces(t *testing.T) {
	svc, kfake := newTestResourceService(t) // use existing helper; if absent, mirror setup from makeDeployment test
	// Create events in different namespaces, both referencing the same UID
	_, _ = kfake.CoreV1().Events("ns-a").Create(context.Background(),
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "e1", Namespace: "ns-a"},
			InvolvedObject: corev1.ObjectReference{UID: "cluster-scoped-uid"},
			Reason:         "Created",
			Type:           "Normal",
		}, metav1.CreateOptions{})
	_, _ = kfake.CoreV1().Events("ns-b").Create(context.Background(),
		&corev1.Event{
			ObjectMeta:     metav1.ObjectMeta{Name: "e2", Namespace: "ns-b"},
			InvolvedObject: corev1.ObjectReference{UID: "cluster-scoped-uid"},
			Reason:         "Updated",
			Type:           "Normal",
		}, metav1.CreateOptions{})

	events, err := svc.GetEvents("ctx", "", "cluster-scoped-uid")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 2, len(events))
}
```

If `newTestResourceService` doesn't exist, extract the setup from an existing test (e.g. `TestResourceService_ScaleResource`) into a helper.

- [ ] **Step 3: Run to verify failure**

Run: `go test ./internal/services/ -run TestGetEvents_ClusterScoped -v`
Expected: FAIL — empty namespace likely returns a namespace-scoped result with 0 events (or panics).

- [ ] **Step 4: Update `GetEvents` to handle empty namespace**

In `internal/services/resource.go`, change `GetEvents` to call `Events("")` when the namespace argument is `""` — the fake clientset (and real k8s API) treats empty namespace on `CoreV1().Events("")` as all-namespaces. Filter by `involvedObject.uid` client-side OR via field selector:

```go
func (s *ResourceService) GetEvents(contextName, namespace, uid string) ([]map[string]any, error) {
	conn, err := s.appService.ClusterManager().GetConnection(contextName)
	if err != nil {
		return nil, err
	}

	// Empty namespace => cluster-wide search (used for cluster-scoped resources).
	sel := fields.OneTermEqualSelector("involvedObject.uid", uid)
	list, err := conn.Clientset.CoreV1().Events(namespace).List(s.ctx, metav1.ListOptions{
		FieldSelector: sel.String(),
	})
	if err != nil {
		return nil, err
	}

	out := make([]map[string]any, 0, len(list.Items))
	for i := range list.Items {
		u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&list.Items[i])
		if err != nil {
			continue
		}
		out = append(out, u)
	}
	return out, nil
}
```

Ensure imports: `"k8s.io/apimachinery/pkg/fields"`, `"k8s.io/apimachinery/pkg/runtime"`.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/services/ -run TestGetEvents -v`
Expected: PASS.

- [ ] **Step 6: Regenerate bindings (signature unchanged, but safe)**

Run: `wails3 generate bindings`
Expected: no diff (or trivial diff).

- [ ] **Step 7: Commit**

```bash
jj new && jj desc -m "services: support cluster-scoped GetEvents via empty namespace"
```

---

## Task 4: EventsPanel — pass cluster scope and wire real-time watch

**Files:**
- Modify: `frontend/src/lib/components/panels/EventsPanel.svelte`
- Modify: `frontend/src/lib/components/ResourceDetail.svelte` — pass `clusterScoped` flag as a prop

- [ ] **Step 1: Read current `EventsPanel.svelte`**

Open `frontend/src/lib/components/panels/EventsPanel.svelte`. Note its props (likely `{ contextName, namespace, uid }`) and how it calls `GetEvents`.

- [ ] **Step 2: Add cluster-scoped awareness + watch**

Replace the component body (adapt the following to match existing styling; preserve the table rendering):

```svelte
<script lang="ts">
  import { GetEvents } from "../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js";
  import { Events as WailsEvents } from "@wailsio/runtime";

  interface Props {
    contextName: string;
    namespace: string; // "" for cluster-scoped resources
    uid: string;
    kind: string;
    name: string;
  }
  let { contextName, namespace, uid, kind, name }: Props = $props();

  let items = $state<Record<string, unknown>[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  async function refresh() {
    try {
      loading = true;
      items = (await GetEvents(contextName, namespace, uid)) ?? [];
      error = null;
    } catch (e) {
      error = (e as Error)?.message ?? String(e);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void refresh();

    // Subscribe to event watches. For cluster-scoped, watch across all
    // namespaces; for namespaced, watch the resource's namespace.
    const ns = namespace || "";
    const eventKey = `watch:${contextName}:core.v1.events:${ns}`;
    const unsub = WailsEvents.On(eventKey, (evt: { data: { type: string; object: any } }) => {
      const obj = evt.data?.object;
      if (!obj) return;
      if (obj.involvedObject?.uid !== uid) return;
      void refresh();
    });

    return () => unsub?.();
  });

  function severity(type: unknown): "warning" | "normal" {
    return String(type).toLowerCase() === "warning" ? "warning" : "normal";
  }
</script>

{#if loading}
  <div class="p-4 text-muted text-sm">Loading events…</div>
{:else if error}
  <div class="p-4 text-destructive text-sm">Failed: {error}</div>
{:else if items.length === 0}
  <div class="p-4 text-muted text-sm">No events for {kind}/{name}.</div>
{:else}
  <table class="w-full text-sm">
    <thead class="text-muted text-left">
      <tr>
        <th class="p-2">Type</th>
        <th class="p-2">Reason</th>
        <th class="p-2">Message</th>
        <th class="p-2 w-16 text-right">Count</th>
        <th class="p-2 w-24 text-right">Age</th>
      </tr>
    </thead>
    <tbody>
      {#each items as e (e.metadata?.uid ?? `${e.metadata?.name}`)}
        {@const sev = severity(e.type)}
        <tr class={sev === "warning" ? "bg-amber-500/5" : ""}>
          <td class="p-2">
            <span class={sev === "warning" ? "text-amber-500" : "text-muted"}>{e.type}</span>
          </td>
          <td class="p-2">{e.reason}</td>
          <td class="p-2">{e.message}</td>
          <td class="p-2 text-right">{e.count ?? 1}</td>
          <td class="p-2 text-right text-muted">
            {/* use your existing age helper; if `formatAge` is in scope, use it */}
            {e.lastTimestamp ?? e.eventTime ?? ""}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
```

Replace the age cell's `{e.lastTimestamp ?? e.eventTime ?? ""}` with the project's existing age formatter (search `formatAge` / `ageFrom` in `frontend/src/lib/utils/` to locate it).

- [ ] **Step 3: Ensure ResourceDetail passes all required props**

Open `frontend/src/lib/components/ResourceDetail.svelte`. Find where panels are rendered (the panel component map). The `EventsPanel` invocation should pass: `contextName`, `namespace` (empty string for cluster-scoped), `uid`, `kind`, `name`. If the current invocation is missing `kind` or `name`, add them.

Also, because `EventsPanel` now subscribes to a watch, we need to ensure the `core.v1.events` watch is running for the relevant namespace. Check if `ResourceDetail.svelte` already starts watches for other resources — if so, follow the same pattern to add:

```typescript
// near the component's mount logic
onMount(() => {
  const ns = clusterScoped ? "" : namespace;
  void StartWatch(contextName, "core.v1.events", ns);
  return () => {
    void StopWatch(contextName, "core.v1.events", ns);
  };
});
```

(If `StartWatch` is already invoked centrally, skip this — check `frontend/src/lib/stores/resource.svelte.ts` or a top-level layout component.)

- [ ] **Step 4: Unit test for EventsPanel**

Create/Modify `frontend/src/lib/__tests__/EventsPanel.svelte.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/svelte";
import EventsPanel from "../components/panels/EventsPanel.svelte";

vi.mock("../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js", () => ({
  GetEvents: vi.fn(),
}));

import { GetEvents } from "../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js";

describe("EventsPanel", () => {
  beforeEach(() => vi.clearAllMocks());

  it("shows empty state when no events", async () => {
    (GetEvents as any).mockResolvedValue([]);
    render(EventsPanel, {
      props: { contextName: "c", namespace: "ns", uid: "u", kind: "Pod", name: "p" },
    });
    await waitFor(() => {
      expect(screen.getByText(/No events/)).toBeInTheDocument();
    });
  });

  it("renders rows with warning class on Warning events", async () => {
    (GetEvents as any).mockResolvedValue([
      { metadata: { uid: "e1", name: "e1" }, type: "Warning", reason: "Failed", message: "oops", count: 2 },
      { metadata: { uid: "e2", name: "e2" }, type: "Normal", reason: "Created", message: "ok", count: 1 },
    ]);
    render(EventsPanel, {
      props: { contextName: "c", namespace: "ns", uid: "u", kind: "Pod", name: "p" },
    });
    await waitFor(() => {
      expect(screen.getByText("Failed")).toBeInTheDocument();
      expect(screen.getByText("Created")).toBeInTheDocument();
    });
  });

  it("passes empty namespace for cluster-scoped resources", async () => {
    (GetEvents as any).mockResolvedValue([]);
    render(EventsPanel, {
      props: { contextName: "c", namespace: "", uid: "u", kind: "Node", name: "n1" },
    });
    await waitFor(() => {
      expect(GetEvents).toHaveBeenCalledWith("c", "", "u");
    });
  });
});
```

- [ ] **Step 5: Run tests**

Run: `cd frontend && npx vitest run src/lib/__tests__/EventsPanel.svelte.test.ts`
Expected: all PASS.

- [ ] **Step 6: Type-check**

Run: `cd frontend && pnpm check`
Expected: 0 errors.

- [ ] **Step 7: Commit**

```bash
jj new && jj desc -m "EventsPanel: real-time watch, cluster-scoped support, severity color"
```

---

## Task 5: Manual verification

- [ ] **Step 1: Launch dev mode, open a CRD detail page**

Run: `task dev`

Connect to a cluster, trigger an event on a CR (e.g. kick a reconciliation), navigate to `Events` tab. Confirm:
- Events appear for the CR.
- Warning events have amber tint.
- New events appear without manual refresh (watch works).
- Cluster-scoped resources (e.g. a ClusterRole) show events across namespaces.

- [ ] **Step 2: Commit the phase marker**

```bash
jj new && jj desc -m "docs: phase 3 universal Events tab complete"
```

---

## Self-Review Checklist

- [x] Events tab is now universal via auto-generated descriptors (Phase 2) + added to all built-ins (Task 2).
- [x] Cluster-scoped resources handled via empty-namespace path.
- [x] Real-time updates via watch subscription.
- [x] Severity color coding (Warning events tinted).
- [x] All tests precede implementation.
