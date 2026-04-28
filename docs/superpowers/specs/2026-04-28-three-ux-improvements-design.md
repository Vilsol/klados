# Three UX Improvements — Design

**Date:** 2026-04-28
**Scope:** three independent frontend-only feature slices, bundled because each is too small to warrant its own spec/plan cycle.

1. CTRL+K command palette surfaces CRDs
2. Left sidebar is resizable
3. Bulk-delete dialog supports force delete

No Go/Wails backend changes. No new dependencies. No Wails binding regeneration (the backend RPC `ForceDeleteResource` already exists).

---

## Feature 1 — CRDs in CTRL+K command palette

### Problem

`CommandPalette.svelte` builds its "Navigate" group from `descriptorRegistry.list()`, which returns only explicitly registered descriptors (built-ins + plugin-registered + virtual). Discovered CRDs are populated into a separate `discovery` map and only get a generated descriptor lazily via `get(gvr)` when the user navigates to one. As a result, CRDs are never reachable through CTRL+K today.

### Approach

Expose discovered GVRs from `DescriptorRegistry`, then emit a separate "Custom Resources" category in the palette so they don't drown out built-ins on CRD-heavy clusters (Crossplane/ArgoCD/Istio routinely add 100+).

### Files

- `frontend/src/lib/registry/index.ts` — add `listDiscoveryGVRs()` accessor.
- `frontend/src/lib/components/CommandPalette.svelte` — emit new category.

### Registry change

Add to `DescriptorRegistry`:

```ts
listDiscoveryGVRs(): APIResource[] {
  const out: APIResource[] = [];
  for (const [gvr, r] of this.discovery) {
    if (this.descriptors.has(gvr)) continue;       // already in built-ins/plugins
    if (this.builtins.has(gvr)) continue;          // belt-and-suspenders
    out.push(r);
  }
  out.sort((a, b) => (a.kind || a.resource).localeCompare(b.kind || b.resource));
  return out;
}
```

### Palette item shape

For each entry returned by `listDiscoveryGVRs()` when a context is active:

- `id`: `nav-crd:${ctx}:${gvr}`
- `label`: `r.kind || r.resource`
- `subtitle`: `${ctx} · ${r.group || "core"}/${r.version}`
- `category`: `"Custom Resources"`
- `action`: `push(/c/${encodeURIComponent(ctx)}/${gvr})`

### Stable category order

`CommandPalette.grouped` currently relies on `Map` insertion order. Replace the ad-hoc grouping with an explicit ordered iteration:

```
Navigate → Custom Resources → Actions → Clusters → Plugins
```

Categories not present (e.g. no plugins installed) are skipped.

### Filtering

Existing fuzzy match (`label/subtitle.toLowerCase().includes(q)`) is sufficient. The 20-result cap stays. Built-ins always come first because of the category order, so typing "po" still surfaces `Pods` before any CRD that happens to contain "po".

### Tests

`frontend/src/lib/__tests__/CommandPalette.svelte.test.ts` (or equivalent existing path):

- Mock `descriptorRegistry.list()` to return a single built-in `Pod` descriptor.
- Mock `descriptorRegistry.listDiscoveryGVRs()` to return a fake `VirtualService` CRD.
- Open palette → assert two distinct category headers render in order: `Navigate`, `Custom Resources`.
- Click the CRD entry → assert `push("/c/test-ctx/networking.istio.io.v1.virtualservices")` is called.

---

## Feature 2 — Resizable left sidebar

### Problem

`Sidebar.svelte` uses a fixed Tailwind width class. Users with long namespace/resource names lose readable text; users on small displays want it narrower.

### Approach

Persist width in `sessionStore` (alongside the existing `sidebarCollapsed`), reuse the drag pattern already proven in `BottomPanelResizeHandle.svelte`, keep the existing collapse toggle as the only path to fully-collapsed.

### Files

- `frontend/src/lib/stores/session.svelte.ts` — add `sidebarWidth` field + clamp helpers.
- `frontend/src/lib/components/SidebarResizeHandle.svelte` — new component (small, mirrors `BottomPanelResizeHandle.svelte`).
- `frontend/src/lib/components/Sidebar.svelte` — replace fixed width class with inline `style`, mount the handle.
- `frontend/src/lib/components/Layout.svelte` — only if Layout owns the sidebar wrapper width; otherwise no change.

### Constants

Exported from `session.svelte.ts`:

```ts
export const SIDEBAR_MIN_WIDTH = 180;
export const SIDEBAR_MAX_WIDTH = 480;
export const SIDEBAR_DEFAULT_WIDTH = 240;
```

### Store change

Add to `sessionStore`:

```ts
sidebarWidth: number; // persisted, clamped to [MIN, MAX] on read
```

Default `SIDEBAR_DEFAULT_WIDTH`. Persisted via the existing 500ms debounced save. On load, clamp to bounds in case the file was hand-edited.

### Resize handle

`SidebarResizeHandle.svelte`:

- Absolutely positioned on the sidebar's right edge, 4px wide, full height.
- `cursor: col-resize`. Hover: `bg-border`. Active drag: `bg-accent`.
- `pointerdown` → `e.target.setPointerCapture(e.pointerId)`, record `startX = e.clientX`, `startWidth = sessionStore.sidebarWidth`, set `document.body.dataset.resizing = "true"`.
- `pointermove` → `next = clamp(startWidth + (e.clientX - startX), MIN, MAX)`, write to store directly (debounce in store handles persist).
- `pointerup` / `pointercancel` → release capture, delete `body.dataset.resizing`.
- `dblclick` → reset to `SIDEBAR_DEFAULT_WIDTH`.

Global CSS rule (in `frontend/src/app.css`):

```css
body[data-resizing="true"] {
  user-select: none;
  cursor: col-resize;
}
```

### Sidebar integration

`Sidebar.svelte`:

- When **not** collapsed: wrapping element uses `style="width: {sessionStore.sidebarWidth}px"`.
- When **collapsed**: existing collapsed-width class wins (no change).
- Mount `<SidebarResizeHandle />` only when expanded.

### Tests

- `session.svelte.ts`: unit test that out-of-range values get clamped on read, and that `sidebarWidth` round-trips through save/load.
- Drag DOM behavior: manual verification (covered by the existing `BottomPanelResizeHandle` precedent).

---

## Feature 3 — Bulk force delete

### Problem

`BulkDeleteDialog.svelte` always calls `DeleteResource`, which uses Kubernetes' default graceful termination. When pods/finalized resources are stuck terminating, users have no in-app escape hatch and must drop to `kubectl delete --force --grace-period=0`.

### Approach

Add a "Force delete" checkbox inside the existing dialog. When toggled, the loop calls the already-existing `ForceDeleteResource` Wails RPC (it sets `gracePeriodSeconds=0` + `propagationPolicy=Background` on the backend) instead of `DeleteResource`. No backend changes.

### Files

- `frontend/src/lib/components/BulkDeleteDialog.svelte` — add checkbox state, swap RPC call, swap labels/banners.

### UI changes

Below the resource list, before the action buttons:

```
[ ] Force delete (skip graceful shutdown)
```

When the checkbox is **off** (default):

- Confirm button label: `Delete N items`
- Banner: existing destructive style.

When the checkbox is **on**:

- Confirm button label: `Force Delete N items`
- Banner switches to a louder warning state (existing destructive token, plus a small explanatory line):
  > Bypasses graceful termination. May leave dangling resources (etcd entries, finalizers). Use only for stuck objects.

State is dialog-local; reset to `false` every time the dialog opens.

### Wiring

Existing loop:

```ts
await DeleteResource(contextName, gvr, ns, name);
```

becomes:

```ts
const fn = force ? ForceDeleteResource : DeleteResource;
await fn(contextName, gvr, ns, name);
```

Add the import for `ForceDeleteResource` from the same bindings module.

### Permissions

No new gating. If a user can delete (`canMutate()`), they can force-delete — matches `kubectl delete --force` semantics.

### Tests

`BulkDeleteDialog.svelte.test.ts`:

- Mock both `DeleteResource` and `ForceDeleteResource`.
- Render dialog, click confirm with checkbox unchecked → assert `DeleteResource` called, `ForceDeleteResource` not called.
- Render again, toggle checkbox on, click confirm → assert `ForceDeleteResource` called for each item.
- Assert the confirm button label changes from `Delete N items` to `Force Delete N items` when the checkbox is toggled.

---

## Cross-cutting

### Out of scope

- Snap-to-collapse when the sidebar drags below `MIN_WIDTH`. Collapse stays explicit via the existing toggle.
- Per-cluster sidebar width. It's UI state, not a per-cluster preference.
- Force-delete in single-resource detail view. This spec is bulk-only; that can be added later if requested.
- Pinning frequently-used CRDs to the top of the palette. Categories stay flat for now.

### Sequencing

The three features are independent and can ship in any order or in parallel. A natural single-PR ordering:

1. CRDs in palette (smallest diff, no store changes).
2. Force delete (one component, no store changes).
3. Sidebar resize (touches store + new component + global CSS).

### Verification

Per `superpowers:verification-before-completion`:

- `cd frontend && pnpm check` (typecheck) clean.
- `cd frontend && pnpm test` for the new vitest cases.
- `task dev`, manual smoke for each feature: open CTRL+K with a cluster that has CRDs; drag the sidebar handle and double-click to reset; bulk-select pods on a sandbox cluster and force-delete.
