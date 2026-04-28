# Three UX Improvements — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Surface CRDs in CTRL+K palette, make the left sidebar resizable, and add a force-delete option to the bulk-delete dialog.

**Architecture:** Three independent frontend-only slices. No Go/Wails changes — `ForceDeleteResource` already exists on the backend. Each task is a self-contained PR-sized unit that ends with a `jj` commit.

**Tech Stack:** Svelte 5 runes, Vitest, Tailwind v4, existing Wails bindings.

**Spec:** [`docs/superpowers/specs/2026-04-28-three-ux-improvements-design.md`](../specs/2026-04-28-three-ux-improvements-design.md)

**VCS:** This repo uses `jj`. Each task starts with `jj st`; if `@` already has changes from a previous task, run `jj new` first. Then `jj desc -m "<message>"`. Make changes. Do **not** run `jj new` at the end — let the next task's first step do it.

---

## Task 1 — CRDs in CTRL+K command palette

**Scope:** Expose discovered GVRs from `DescriptorRegistry`, then have `CommandPalette.svelte` render them in a separate "Custom Resources" category with stable category ordering.

**Files:**
- Modify: `frontend/src/lib/registry/index.ts` — add `listDiscoveryGVRs()` accessor.
- Modify: `frontend/src/lib/components/CommandPalette.svelte` — emit new category, force category order.
- Modify or create: `frontend/src/lib/components/__tests__/CommandPalette.svelte.test.ts` — vitest coverage.

### Steps

- [ ] **1.1 — `jj` setup**
  - Run `jj st`. If `@` has changes, `jj new`.
  - `jj desc -m "feat(palette): surface CRDs in command palette under Custom Resources category"`

- [ ] **1.2 — Add `listDiscoveryGVRs()` to `DescriptorRegistry`**

  In `frontend/src/lib/registry/index.ts`, add this method (place near `list()`):

  ```ts
  /**
   * Discovered GVRs not already covered by an explicit descriptor (built-in,
   * plugin-registered, or virtual). Used by the command palette to surface CRDs.
   * Sorted alphabetically by kind, then group.
   */
  listDiscoveryGVRs(): APIResource[] {
    const out: APIResource[] = [];
    for (const [gvr, r] of this.discovery) {
      if (this.descriptors.has(gvr)) continue;
      if (this.builtins.has(gvr)) continue;
      out.push(r);
    }
    out.sort((a, b) => {
      const kindCmp = (a.kind || a.resource).localeCompare(b.kind || b.resource);
      if (kindCmp !== 0) return kindCmp;
      return (a.group || "").localeCompare(b.group || "");
    });
    return out;
  }
  ```

- [ ] **1.3 — Emit "Custom Resources" entries in CommandPalette**

  In `frontend/src/lib/components/CommandPalette.svelte`, inside `buildItems()` immediately after the existing built-in `descriptorRegistry.list()` loop, add:

  ```ts
  for (const r of descriptorRegistry.listDiscoveryGVRs()) {
    const groupLabel = r.group || "core";
    items.push({
      id: `nav-crd:${ctx}:${r.gvr}`,
      label: r.kind || r.resource,
      subtitle: `${ctx} · ${groupLabel}/${r.version}`,
      category: "Custom Resources",
      action: () => {
        push(`/c/${encodeURIComponent(ctx)}/${r.gvr}`);
        open = false;
      },
    });
  }
  ```

  Note: `APIResource` is the type already imported by the registry; CommandPalette doesn't need to import it because it consumes plain shape via the registry method.

- [ ] **1.4 — Force stable category order**

  Replace the `grouped` derivation in `CommandPalette.svelte` with one that iterates a fixed order, skipping empty categories:

  ```ts
  const CATEGORY_ORDER = ["Navigate", "Custom Resources", "Actions", "Clusters", "Plugins"] as const;

  const grouped = $derived.by(() => {
    const byCategory = new Map<string, PaletteItem[]>();
    for (const item of filtered) {
      const arr = byCategory.get(item.category) ?? [];
      arr.push(item);
      byCategory.set(item.category, arr);
    }
    const ordered: [string, PaletteItem[]][] = [];
    for (const cat of CATEGORY_ORDER) {
      const arr = byCategory.get(cat);
      if (arr && arr.length > 0) ordered.push([cat, arr]);
    }
    // Any unexpected categories appended at the end (defensive)
    for (const [cat, arr] of byCategory) {
      if (!CATEGORY_ORDER.includes(cat as (typeof CATEGORY_ORDER)[number])) {
        ordered.push([cat, arr]);
      }
    }
    return ordered;
  });
  ```

  Update the template's `{#each grouped as [category, items] (category)}` — already iterating an array of tuples, so it works with `Map.entries()` today and continues to work with the array.

- [ ] **1.5 — Test**

  Add to `frontend/src/lib/components/__tests__/CommandPalette.svelte.test.ts` (create if it doesn't exist; mock pattern is the same as other component tests in that directory — they mock `@wailsio/runtime` in `setup.ts`).

  Test cases:
  1. Mock `descriptorRegistry.list()` → `[Pod descriptor]` and `descriptorRegistry.listDiscoveryGVRs()` → `[VirtualService APIResource]`. Mock `clusterStore.activeContext` to a fixed value. Render `<CommandPalette open={true} />`. Assert both `Navigate` and `Custom Resources` headers render in that order.
  2. Click the `VirtualService` entry → assert `push` from `svelte-spa-router` is called with `/c/test-ctx/networking.istio.io.v1.virtualservices`.

  Use `vi.mock(...)` for `$lib/registry/index` and `svelte-spa-router`. If existing tests already mock these, follow that pattern.

- [ ] **1.6 — Verify and commit**

  ```bash
  cd frontend && pnpm check
  cd frontend && pnpm test -- CommandPalette
  ```

  Both must pass. Then `jj st` to confirm only intended files changed. The commit description from step 1.1 already covers it; no extra command needed (jj auto-snapshots).

---

## Task 2 — Bulk force delete

**Scope:** Add a "Force delete" checkbox to `BulkDeleteDialog.svelte` that swaps the per-item RPC from `DeleteResource` to the existing `ForceDeleteResource`.

**Files:**
- Modify: `frontend/src/lib/components/BulkDeleteDialog.svelte`
- Modify or create: `frontend/src/lib/components/__tests__/BulkDeleteDialog.svelte.test.ts`

### Steps

- [ ] **2.1 — `jj` setup**
  - `jj new` (Task 1's commit is finalized).
  - `jj desc -m "feat(bulk-delete): add force delete option to bulk delete dialog"`

- [ ] **2.2 — Add force checkbox + label/banner swaps**

  In `frontend/src/lib/components/BulkDeleteDialog.svelte`:

  - Add to the imports:
    ```ts
    import {DeleteResource, ForceDeleteResource} from "../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js";
    ```
    (Replace the existing `DeleteResource`-only import.)
  - Add local state:
    ```ts
    let force = $state(false);
    ```
  - Reset `force = false` in whatever lifecycle currently resets the dialog when it opens (an `$effect` keyed on `open`, or the existing `onOpen` handler — match the established reset pattern in the file).
  - Replace the delete call:
    ```ts
    const fn = force ? ForceDeleteResource : DeleteResource;
    await fn(contextName, gvr, ns, name);
    ```
  - Confirm button label becomes `${force ? "Force Delete" : "Delete"} ${count} item${count === 1 ? "" : "s"}`.
  - Render the checkbox below the resource list, before the action buttons:
    ```svelte
    <label class="flex items-start gap-2 mt-3 text-sm cursor-pointer">
      <input type="checkbox" bind:checked={force} class="mt-0.5" />
      <span>
        <span class="font-medium">Force delete</span>
        <span class="text-muted">(skip graceful shutdown)</span>
      </span>
    </label>
    {#if force}
      <p class="mt-2 text-xs text-destructive">
        Bypasses graceful termination. May leave dangling resources (etcd entries, finalizers). Use only for stuck objects.
      </p>
    {/if}
    ```
    (Use existing dialog spacing/typography conventions from the surrounding code; the snippet above uses tokens already in use elsewhere in the codebase.)

- [ ] **2.3 — Test**

  In `frontend/src/lib/components/__tests__/BulkDeleteDialog.svelte.test.ts`:

  - Mock both bindings:
    ```ts
    vi.mock("../../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js", () => ({
      DeleteResource: vi.fn().mockResolvedValue(undefined),
      ForceDeleteResource: vi.fn().mockResolvedValue(undefined),
    }));
    ```
  - Test 1: render dialog with two items, click confirm without toggling the checkbox → assert `DeleteResource` called twice, `ForceDeleteResource` not called.
  - Test 2: render dialog, toggle checkbox, click confirm → assert `ForceDeleteResource` called twice, `DeleteResource` not called.
  - Test 3: assert button label transitions from `Delete 2 items` to `Force Delete 2 items` when the checkbox toggles.

  Match the existing test setup pattern in the same directory (the wailsio runtime mock is in `setup.ts`).

- [ ] **2.4 — Verify and commit**

  ```bash
  cd frontend && pnpm check
  cd frontend && pnpm test -- BulkDeleteDialog
  ```

  Then `jj st` to confirm scope.

---

## Task 3 — Resizable left sidebar

**Scope:** Add `sidebarWidth` to `sessionStore`, build a resize handle component, wire `Sidebar.svelte` to use the dynamic width, add the global `data-resizing` CSS rule.

**Files:**
- Modify: `frontend/src/lib/stores/session.svelte.ts` — add `sidebarWidth` field, constants, `setSidebarWidth`, `resetSidebarWidth`. Update `restore()` and any persistence helper that mirrors the store's serialized shape.
- Create: `frontend/src/lib/components/SidebarResizeHandle.svelte` — drag handle.
- Modify: `frontend/src/lib/components/Sidebar.svelte` — inline width style + mount the handle.
- Modify: `frontend/src/app.css` — add `body[data-resizing="true"]` rule.
- Modify or create: `frontend/src/lib/stores/__tests__/session.svelte.test.ts` — store tests.

### Steps

- [ ] **3.1 — `jj` setup**
  - `jj new`
  - `jj desc -m "feat(sidebar): make left sidebar resizable with drag handle"`

- [ ] **3.2 — Extend `sessionStore`**

  In `frontend/src/lib/stores/session.svelte.ts`:

  Add at module scope:

  ```ts
  export const SIDEBAR_MIN_WIDTH = 180;
  export const SIDEBAR_MAX_WIDTH = 480;
  export const SIDEBAR_DEFAULT_WIDTH = 240;

  function clampSidebarWidth(n: number): number {
    if (!Number.isFinite(n)) return SIDEBAR_DEFAULT_WIDTH;
    return Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, Math.round(n)));
  }
  ```

  Add a field on `SessionStore`:

  ```ts
  sidebarWidth = $state(SIDEBAR_DEFAULT_WIDTH);
  ```

  Add methods:

  ```ts
  setSidebarWidth(width: number) {
    this.sidebarWidth = clampSidebarWidth(width);
  }

  resetSidebarWidth() {
    this.sidebarWidth = SIDEBAR_DEFAULT_WIDTH;
  }
  ```

  Update `restore()` signature and body to accept and clamp `sidebarWidth`:

  ```ts
  restore(
    tabs: TabState[],
    activeTab: number,
    sidebarCollapsed: boolean,
    terminalFontSize?: number,
    sidebarWidth?: number,
  ) {
    this.tabs = tabs;
    this.activeTabIndex = activeTab < tabs.length ? activeTab : 0;
    this.sidebarCollapsed = sidebarCollapsed;
    this.terminalFontSize = terminalFontSize ?? 13;
    this.sidebarWidth = clampSidebarWidth(sidebarWidth ?? SIDEBAR_DEFAULT_WIDTH);
  }
  ```

  **Find all callers of `sessionStore.restore(...)`** (`rg -n "sessionStore.restore" frontend/src` and similar) and update them to pass through `sidebarWidth` from the persisted blob. Look for the matching Go-side struct or JSON shape used for session persistence — `internal/session/session.go` defines the on-disk schema; add a `sidebarWidth` field there mirroring how `sidebarCollapsed` is handled. The Go change is small and parallel to the existing field; keep its zero-value as 0 and treat 0 as "use default" on the frontend (already covered by `clampSidebarWidth`).

- [ ] **3.3 — Add global `data-resizing` CSS**

  In `frontend/src/app.css`, append:

  ```css
  body[data-resizing="true"] {
    user-select: none;
    cursor: col-resize;
  }
  ```

- [ ] **3.4 — Create `SidebarResizeHandle.svelte`**

  Create `frontend/src/lib/components/SidebarResizeHandle.svelte`. Mirror `BottomPanelResizeHandle.svelte` but for horizontal:

  ```svelte
  <script lang="ts">
    import {onMount, onDestroy} from "svelte";
    import {sessionStore} from "$lib/stores/session.svelte";

    let dragging = $state(false);
    let dragStartX = 0;
    let dragStartWidth = 0;

    function onResizeStart(e: MouseEvent) {
      dragging = true;
      dragStartX = e.clientX;
      dragStartWidth = sessionStore.sidebarWidth;
      document.body.dataset.resizing = "true";
      e.preventDefault();
    }

    function onResizeMove(e: MouseEvent) {
      if (!dragging) return;
      const delta = e.clientX - dragStartX;
      sessionStore.setSidebarWidth(dragStartWidth + delta);
    }

    function onResizeEnd() {
      if (!dragging) return;
      dragging = false;
      delete document.body.dataset.resizing;
    }

    function onDoubleClick() {
      sessionStore.resetSidebarWidth();
    }

    onMount(() => {
      document.addEventListener("mousemove", onResizeMove);
      document.addEventListener("mouseup", onResizeEnd);
    });

    onDestroy(() => {
      document.removeEventListener("mousemove", onResizeMove);
      document.removeEventListener("mouseup", onResizeEnd);
      delete document.body.dataset.resizing;
    });
  </script>

  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="absolute top-0 right-0 h-full w-1 cursor-col-resize transition-colors group z-10
      {dragging ? 'bg-accent/40' : 'hover:bg-accent/30'}"
    role="separator"
    aria-orientation="vertical"
    aria-valuenow={sessionStore.sidebarWidth}
    aria-valuemin={180}
    aria-valuemax={480}
    onmousedown={onResizeStart}
    ondblclick={onDoubleClick}
    title="Drag to resize · double-click to reset"
  >
    <div class="w-px h-full mx-auto bg-border group-hover:bg-accent/60 {dragging ? 'bg-accent/60' : ''}"></div>
  </div>
  ```

- [ ] **3.5 — Wire `Sidebar.svelte`**

  In `frontend/src/lib/components/Sidebar.svelte`:

  - Import the new component and store:
    ```ts
    import SidebarResizeHandle from "./SidebarResizeHandle.svelte";
    import {sessionStore} from "$lib/stores/session.svelte";
    ```
    (Almost certainly `sessionStore` is already imported. Don't duplicate.)
  - The sidebar's outer wrapper currently sets a fixed Tailwind width (e.g. `w-60`). Replace that class with `relative` for handle positioning and an inline style binding:
    ```svelte
    <aside
      class="relative ..."
      style={sessionStore.sidebarCollapsed ? "" : `width: ${sessionStore.sidebarWidth}px`}
    >
      ...
      {#if !sessionStore.sidebarCollapsed}
        <SidebarResizeHandle />
      {/if}
    </aside>
    ```
    Keep the existing collapsed-width class for the collapsed state. If the existing wrapper isn't an `<aside>`, use whatever element is already there — only the `class`/`style`/handle-mount changes.

- [ ] **3.6 — Test the store**

  In `frontend/src/lib/stores/__tests__/session.svelte.test.ts` (create if missing), add:

  - `setSidebarWidth(50)` clamps up to `SIDEBAR_MIN_WIDTH` (180).
  - `setSidebarWidth(9999)` clamps down to `SIDEBAR_MAX_WIDTH` (480).
  - `setSidebarWidth(NaN)` falls back to `SIDEBAR_DEFAULT_WIDTH`.
  - `resetSidebarWidth()` returns the value to `SIDEBAR_DEFAULT_WIDTH`.
  - `restore(..., undefined)` yields `SIDEBAR_DEFAULT_WIDTH`.
  - `restore(..., 9999)` yields `SIDEBAR_MAX_WIDTH`.

  Match existing store-test conventions in the repo (no Wails mocks needed for the store itself).

- [ ] **3.7 — Manual smoke**

  Run `task dev`. Verify:
  - Drag the handle: width changes smoothly within [180, 480].
  - Double-click the handle: resets to 240.
  - Toggle collapse: width is preserved on re-expand.
  - Refresh the app: width persists.

- [ ] **3.8 — Verify and commit**

  ```bash
  cd frontend && pnpm check
  cd frontend && pnpm test -- session
  ```

  If the Go session schema was modified, also run:
  ```bash
  go test ./internal/session/ -v
  ```

  Then `jj st` to confirm scope.

---

## Self-Review

**Spec coverage check:**

- Feature 1 (CRDs in palette): registry accessor (1.2), palette wiring (1.3), category order (1.4), tests (1.5). ✓
- Feature 2 (force delete): RPC swap (2.2), checkbox UI (2.2), button label change (2.2), tests (2.3). ✓
- Feature 3 (sidebar resize): store + constants (3.2), CSS (3.3), handle component (3.4), Sidebar wiring (3.5), tests (3.6), persistence on Go side (3.2). ✓
- Cross-cutting: each task has typecheck + targeted vitest run. Task 3 includes manual smoke for the drag mechanic since DOM events resist clean unit tests.

**Type consistency:** `setSidebarWidth` / `resetSidebarWidth` / `SIDEBAR_DEFAULT_WIDTH` / `SIDEBAR_MIN_WIDTH` / `SIDEBAR_MAX_WIDTH` used identically in 3.2, 3.4, 3.6. `listDiscoveryGVRs` consistent across 1.2, 1.3, 1.5. `force` boolean and `ForceDeleteResource` consistent in 2.2 and 2.3.

**Out-of-scope items left out:** snap-to-collapse, per-cluster width, force-delete in detail view, palette pinning. All deliberate per the spec.
