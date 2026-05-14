# Resource List UX Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace single-step column reorder + mixed-concern dropdown with header drag-reorder, a visibility-only column picker, pinned `Name` column, header/row right-click context menus, loading skeletons, and a smart empty state.

**Architecture:** Frontend-heavy. Add one Go field (`GVRColumnPrefs.Pinned`) for persistence, regenerate Wails bindings. The DataTable header gets restructured into pinned + main grids, with `svelte-dnd-action` driving the reorder on the main grid. ColumnMenu is replaced by two focused popovers (ColumnPicker + ViewOptionsMenu) plus a HeaderContextMenu. ResourceList wires the row-level context menu to the existing `bottomPanelStore.addTab({kind: ...})` API for logs/terminal/yaml.

**Tech Stack:** Go 1.25 + Wails v3 alpha.74, Svelte 5 runes, Tailwind v4, TanStack Virtual, `svelte-dnd-action`, vitest + @testing-library/svelte.

**Reference spec:** `docs/superpowers/specs/2026-05-14-resource-list-ux-design.md`

---

## File Map

**New files:**
- `frontend/src/lib/components/ColumnPicker.svelte` — checkbox list, replaces ColumnMenu
- `frontend/src/lib/components/ViewOptionsMenu.svelte` — compact + sparkline toggles
- `frontend/src/lib/components/HeaderContextMenu.svelte` — sort/auto-fit/pin/hide popover
- `frontend/src/lib/utils/yamlClipboard.ts` — single-item YAML serialization helper
- `frontend/src/lib/__tests__/ColumnPicker.svelte.test.ts`
- `frontend/src/lib/__tests__/ViewOptionsMenu.svelte.test.ts`
- `frontend/src/lib/__tests__/HeaderContextMenu.svelte.test.ts`
- `frontend/src/lib/__tests__/yamlClipboard.test.ts`

**Modified files:**
- `internal/config/config.go` — add `Pinned []string` to `GVRColumnPrefs`
- `frontend/src/lib/stores/columns.svelte.ts` — pinning + reorder API
- `frontend/src/lib/stores/bottom-panel.svelte.ts` — optional `editable?: boolean` on `PanelTab`
- `frontend/src/lib/components/BottomPanel.svelte` — pass `editable` to YAML panel
- `frontend/src/lib/components/DataTable.svelte` — pinned layout, header dnd, header ctx menu, skeleton, empty action
- `frontend/src/lib/components/ResourceList.svelte` — swap menus, expand row ctx menu, wire reorder, empty state
- `frontend/src/routes/settings/ColumnSettings.svelte` — display pinned columns
- `frontend/src/lib/__tests__/columns.svelte.test.ts` — cover new store methods
- `frontend/package.json` — add `svelte-dnd-action`

**Removed (requires explicit user confirmation before deletion per repo rule):**
- `frontend/src/lib/components/ColumnMenu.svelte`
- `frontend/src/lib/__tests__/ColumnMenu.svelte.test.ts`

---

## Phase 1 — Persistence & Store

### Task 1: Add `Pinned` to `GVRColumnPrefs` and regenerate bindings

**Files:**
- Modify: `internal/config/config.go:58-62`
- Modify: `internal/config/config_test.go` (round-trip test if present)
- Regenerate: `frontend/bindings/github.com/Vilsol/klados/internal/config/models.ts`

- [ ] **Step 1: Add `Pinned []string` field**

Edit `internal/config/config.go` lines 58-62:

```go
type GVRColumnPrefs struct {
    Columns map[string]ColumnSettings `json:"columns"`
    Order   []string                  `json:"order"`
    Sort    *SortPrefs                `json:"sort,omitempty"`
    Pinned  []string                  `json:"pinned,omitempty"`
}
```

- [ ] **Step 2: Run Go tests for the config package**

```bash
go test ./internal/config/ -v
```
Expected: PASS. Old configs without `pinned` decode fine (zero-value slice).

- [ ] **Step 3: Regenerate Wails bindings**

```bash
wails3 generate bindings
```

Verify `frontend/bindings/github.com/Vilsol/klados/internal/config/models.ts` (or `.js` per CLAUDE.md note) now contains a `pinned` property on `GVRColumnPrefs`.

- [ ] **Step 4: Type-check frontend**

```bash
cd frontend && pnpm check
```
Expected: PASS (no consumers reference `pinned` yet; the new property is optional).

- [ ] **Step 5: Commit via jj**

```bash
jj new
jj desc -m "feat(config): add Pinned to GVRColumnPrefs for sticky-left columns"
```

---

### Task 2: Extend `ColumnStore` with reorder + pinning

**Files:**
- Modify: `frontend/src/lib/stores/columns.svelte.ts`
- Modify: `frontend/src/lib/__tests__/columns.svelte.test.ts`

- [ ] **Step 1: Write failing tests for the new store methods**

Append to `frontend/src/lib/__tests__/columns.svelte.test.ts`:

```ts
describe("ColumnStore — pinning & reorder", () => {
  beforeEach(() => {
    mockGetColumnPrefs.mockReset();
    mockSetColumnPrefs.mockReset();
    mockGetDescriptor.mockReturnValue(podDescriptor);
  });

  it("pins Name by default when no prefs are saved", async () => {
    mockGetColumnPrefs.mockResolvedValue(null);
    await columnStore.loadForGVR("core.v1.pods");
    expect(columnStore.isPinned("Name")).toBe(true);
    expect(columnStore.pinnedNames()).toEqual(["Name"]);
  });

  it("setPinned(true) moves the column to the front of visibleColumns", async () => {
    mockGetColumnPrefs.mockResolvedValue(null);
    await columnStore.loadForGVR("core.v1.pods");
    columnStore.setPinned("Status", true);
    expect(columnStore.visibleColumns[0].name).toBe("Name");
    expect(columnStore.visibleColumns[1].name).toBe("Status");
    expect(columnStore.isPinned("Status")).toBe(true);
  });

  it("setPinned(false) removes the column from the pinned set", async () => {
    mockGetColumnPrefs.mockResolvedValue({order: ["Name", "Status", "Age"], columns: {}, pinned: ["Name", "Status"]});
    await columnStore.loadForGVR("core.v1.pods");
    columnStore.setPinned("Status", false);
    expect(columnStore.isPinned("Status")).toBe(false);
    expect(columnStore.pinnedNames()).toEqual(["Name"]);
  });

  it("reorderVisible(names) replaces visibleColumns and persists", async () => {
    mockGetColumnPrefs.mockResolvedValue(null);
    await columnStore.loadForGVR("core.v1.pods");
    columnStore.reorderVisible(["Name", "Age", "Status"]);
    expect(columnStore.visibleColumns.map((c) => c.name)).toEqual(["Name", "Age", "Status"]);
    expect(mockSetColumnPrefs).toHaveBeenCalled();
  });

  it("reorderVisible ignores names not currently visible", async () => {
    mockGetColumnPrefs.mockResolvedValue(null);
    await columnStore.loadForGVR("core.v1.pods");
    const before = columnStore.visibleColumns.map((c) => c.name);
    columnStore.reorderVisible(["Name", "DoesNotExist", "Age"]);
    expect(columnStore.visibleColumns.map((c) => c.name)).toEqual(["Name", "Age"]);
    // unrelated columns are dropped from visible — that's the documented behavior
    expect(before).not.toEqual(columnStore.visibleColumns.map((c) => c.name));
  });
});
```

- [ ] **Step 2: Run tests, confirm failures**

```bash
cd frontend && pnpm vitest run src/lib/__tests__/columns.svelte.test.ts
```
Expected: 5 new tests FAIL ("`columnStore.isPinned is not a function`", etc.).

- [ ] **Step 3: Implement pinning + reorder in `columns.svelte.ts`**

Replace the body of `columns.svelte.ts` with the additions below. Three changes: add `pinnedSet` state, augment `#applyPrefs` to default-pin Name, and add the new methods. Existing methods unchanged.

Add at the top of the class (after `compact = $state<boolean>(false);`):

```ts
#pinnedSet = $state<Set<string>>(new Set());
```

Update `#applyPrefs` after computing `visibleNames` (before the sort section). Insert:

```ts
// Pinned set: explicit prefs OR default to ["Name"] when Name is visible
const explicitPinned = prefs?.pinned ?? [];
const pinned = explicitPinned.length > 0
  ? explicitPinned.filter((n) => visibleSet.has(n))
  : (visibleSet.has("Name") ? ["Name"] : []);
this.#pinnedSet = new Set(pinned);

// Reorder visibleNames so pinned columns are first (preserving pinned order)
const pinnedFirst = [...pinned, ...visibleNames.filter((n) => !this.#pinnedSet.has(n))];
this.visibleColumns = pinnedFirst.map((name) => poolMap.get(name)).filter((c): c is ColumnDef => c !== undefined);
```

Replace `#buildPrefs` to include pinned:

```ts
#buildPrefs(): GVRColumnPrefs {
  return new GVRColumnPrefs({
    order: this.visibleColumns.map((c) => c.name),
    columns: Object.fromEntries(
      this.allColumns.filter(({col}) => col.width !== undefined).map(({col}) => [col.name, new ColumnSettings({width: col.width})]),
    ),
    sort: this.sortState ? new SortPrefs({column: this.sortState.column, direction: this.sortState.direction}) : null,
    pinned: [...this.#pinnedSet],
  });
}
```

Add the new methods at the bottom of the class (before the closing brace):

```ts
reorderVisible(names: string[]): void {
  const currentSet = new Set(this.visibleColumns.map((c) => c.name));
  const filtered = names.filter((n) => currentSet.has(n));
  // Pin order must be preserved: pinned names always lead, then the rest in `filtered` order
  const pinned = [...this.#pinnedSet].filter((n) => currentSet.has(n));
  const rest = filtered.filter((n) => !this.#pinnedSet.has(n));
  const finalOrder = [...pinned, ...rest];
  const byName = new Map(this.visibleColumns.map((c) => [c.name, c]));
  this.visibleColumns = finalOrder.map((n) => byName.get(n)).filter((c): c is ColumnDef => c !== undefined);
  this.#save();
}

setPinned(name: string, pinned: boolean): void {
  if (!this.visibleColumns.some((c) => c.name === name)) {
    return;
  }
  const next = new Set(this.#pinnedSet);
  if (pinned) {
    next.add(name);
  } else {
    next.delete(name);
  }
  this.#pinnedSet = next;

  if (pinned) {
    // Move to end of pinned section
    const without = this.visibleColumns.filter((c) => c.name !== name);
    const pinnedCount = without.filter((c) => this.#pinnedSet.has(c.name)).length;
    const col = this.visibleColumns.find((c) => c.name === name);
    if (col) {
      this.visibleColumns = [...without.slice(0, pinnedCount), col, ...without.slice(pinnedCount)];
    }
  }
  this.#save();
}

isPinned(name: string): boolean {
  return this.#pinnedSet.has(name);
}

pinnedNames(): string[] {
  return this.visibleColumns.filter((c) => this.#pinnedSet.has(c.name)).map((c) => c.name);
}
```

Also update `setColumnVisible` to refuse hiding a pinned column:

```ts
setColumnVisible(name: string, visible: boolean): void {
  if (name === "Name") {
    return;
  }
  if (!visible && this.#pinnedSet.has(name)) {
    return; // unpin first
  }
  // ... rest unchanged
}
```

- [ ] **Step 4: Run tests, confirm pass**

```bash
cd frontend && pnpm vitest run src/lib/__tests__/columns.svelte.test.ts
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
jj new
jj desc -m "feat(columns): reorderVisible + setPinned/isPinned, default-pin Name"
```

---

## Phase 2 — Building blocks

### Task 3: Add `svelte-dnd-action` dependency and spike test

**Files:**
- Modify: `frontend/package.json`

- [ ] **Step 1: Install**

```bash
cd frontend && pnpm add svelte-dnd-action
```

- [ ] **Step 2: Verify it loads inside Wails WebView**

In `frontend/src/App.svelte` (temporarily, will revert), at the top of the script, add:

```ts
import {dndzone as _spike} from "svelte-dnd-action";
void _spike;
```

Then:

```bash
cd frontend && pnpm build
```
Expected: build succeeds. Revert the temporary import before committing.

- [ ] **Step 3: Confirm types resolve**

```bash
cd frontend && pnpm check
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
jj new
jj desc -m "build(frontend): add svelte-dnd-action for column reorder"
```

---

### Task 4: `yamlClipboard.ts` utility

**Files:**
- Create: `frontend/src/lib/utils/yamlClipboard.ts`
- Create: `frontend/src/lib/__tests__/yamlClipboard.test.ts`

- [ ] **Step 1: Inspect existing serialization**

Look at `frontend/src/lib/utils/export.ts` to find the YAML serializer used by `exportItems`. If it's `js-yaml` directly, the new helper reuses it. Note the library name for the next step.

- [ ] **Step 2: Write failing test**

Create `frontend/src/lib/__tests__/yamlClipboard.test.ts`:

```ts
import {describe, it, expect} from "vitest";
import {itemToYaml} from "$lib/utils/yamlClipboard";

describe("itemToYaml", () => {
  it("serializes a Kubernetes resource to YAML", () => {
    const pod = {
      apiVersion: "v1",
      kind: "Pod",
      metadata: {name: "test-pod", namespace: "default"},
      spec: {containers: [{name: "main", image: "nginx"}]},
    };
    const yaml = itemToYaml(pod);
    expect(yaml).toContain("apiVersion: v1");
    expect(yaml).toContain("kind: Pod");
    expect(yaml).toContain("name: test-pod");
    expect(yaml).toContain("image: nginx");
  });

  it("strips managed fields and status if present (matches export behavior)", () => {
    const obj = {
      apiVersion: "v1",
      kind: "Pod",
      metadata: {name: "p", managedFields: [{manager: "k"}]},
      status: {phase: "Running"},
    };
    const yaml = itemToYaml(obj);
    expect(yaml).not.toContain("managedFields");
    // status is preserved (View YAML should show observed state); only managedFields stripped
    expect(yaml).toContain("phase: Running");
  });
});
```

- [ ] **Step 3: Run, confirm failure**

```bash
cd frontend && pnpm vitest run src/lib/__tests__/yamlClipboard.test.ts
```
Expected: FAIL ("Cannot find module yamlClipboard").

- [ ] **Step 4: Implement `yamlClipboard.ts`**

```ts
import {dump} from "js-yaml";
import type {KubernetesResource} from "$lib/types";

export function itemToYaml(item: Record<string, KubernetesResource> | Record<string, unknown>): string {
  const clone = structuredClone(item) as Record<string, unknown>;
  const meta = clone.metadata as Record<string, unknown> | undefined;
  if (meta && "managedFields" in meta) {
    delete meta.managedFields;
  }
  return dump(clone, {indent: 2, lineWidth: -1, noRefs: true});
}
```

If `export.ts` uses a different YAML lib, swap accordingly; the test still asserts the same shape.

- [ ] **Step 5: Run, confirm pass**

```bash
cd frontend && pnpm vitest run src/lib/__tests__/yamlClipboard.test.ts
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
jj new
jj desc -m "feat(utils): itemToYaml helper for single-item YAML clipboard"
```

---

### Task 5: `editable` flag on `PanelTab`

**Files:**
- Modify: `frontend/src/lib/stores/bottom-panel.svelte.ts:5-17`
- Modify: `frontend/src/lib/components/BottomPanel.svelte` (YAML panel render branch)
- Modify: `frontend/src/lib/__tests__/session.svelte.test.ts` if it asserts on the full `PanelTab` shape (likely no change needed since `editable` is optional)

- [ ] **Step 1: Add the optional field**

In `bottom-panel.svelte.ts`, update the interface (line 5):

```ts
export interface PanelTab {
  id: string;
  kind: PanelKind;
  resourceKind: string;
  resourceName: string;
  ctxName: string;
  gvr: string;
  namespace: string;
  name: string;
  obj: Record<string, unknown>;
  poppedOut: boolean;
  managedId?: string;
  editable?: boolean;
}
```

- [ ] **Step 2: Pass `editable` to the YAML panel**

In `BottomPanel.svelte`, find the `kind === "yaml"` branch (or equivalent — the existing YAML panel rendering). Pass `editable={tab.editable ?? false}` as a prop. If the YAML panel component doesn't accept an `editable` prop yet, add it as `let {editable = false, ...} = $props()` and gate the edit UI on it. (Search the YAML panel file for read-only vs edit-mode controls.)

- [ ] **Step 3: Type-check**

```bash
cd frontend && pnpm check
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
jj new
jj desc -m "feat(panel): optional editable flag on PanelTab for Edit YAML"
```

---

## Phase 3 — Focused popovers

### Task 6: `ColumnPicker.svelte` (replaces ColumnMenu, visibility-only)

**Files:**
- Create: `frontend/src/lib/components/ColumnPicker.svelte`
- Create: `frontend/src/lib/__tests__/ColumnPicker.svelte.test.ts`

- [ ] **Step 1: Write failing tests**

Create `frontend/src/lib/__tests__/ColumnPicker.svelte.test.ts`:

```ts
import {describe, it, expect, vi} from "vitest";
import {render, screen, fireEvent} from "@testing-library/svelte";
import ColumnPicker from "$lib/components/ColumnPicker.svelte";

const col = (name: string) => ({name});

function baseProps(overrides: Record<string, unknown> = {}) {
  return {
    allColumns: [
      {col: col("Name"), visible: true},
      {col: col("Ready"), visible: true},
      {col: col("Status"), visible: true},
      {col: col("Restarts"), visible: false},
      {col: col("IP"), visible: false},
    ],
    visibleColumns: [col("Name"), col("Ready"), col("Status")],
    pinnedNames: ["Name"],
    onToggle: vi.fn(),
    onReset: vi.fn(),
    ...overrides,
  };
}

describe("ColumnPicker", () => {
  it("renders all columns, pinned first, with pinned checkbox disabled", () => {
    render(ColumnPicker, {props: baseProps()});
    const items = screen.getAllByRole("checkbox");
    // Name is first and disabled
    expect(items[0]).toBeDisabled();
    // Other visible checkboxes are checked
    expect(items[1]).toBeChecked();
    expect(items[2]).toBeChecked();
    // Hidden are unchecked
    expect(items[3]).not.toBeChecked();
  });

  it("filters by name (case-insensitive substring)", async () => {
    render(ColumnPicker, {props: baseProps()});
    const input = screen.getByPlaceholderText("Filter…");
    await fireEvent.input(input, {target: {value: "re"}});
    expect(screen.queryByText("Name")).toBeNull();
    expect(screen.getByText("Ready")).toBeTruthy();
    expect(screen.getByText("Restarts")).toBeTruthy();
    expect(screen.queryByText("Status")).toBeNull();
  });

  it("calls onToggle when a non-pinned checkbox flips", async () => {
    const props = baseProps();
    render(ColumnPicker, {props});
    const restartsCheckbox = screen.getAllByRole("checkbox")[3];
    await fireEvent.click(restartsCheckbox);
    expect(props.onToggle).toHaveBeenCalledWith("Restarts", true);
  });

  it("calls onReset when the Reset button is clicked", async () => {
    const props = baseProps();
    render(ColumnPicker, {props});
    await fireEvent.click(screen.getByText("Reset"));
    expect(props.onReset).toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run, confirm failure**

```bash
cd frontend && pnpm vitest run src/lib/__tests__/ColumnPicker.svelte.test.ts
```
Expected: FAIL ("Cannot find module ColumnPicker").

- [ ] **Step 3: Implement `ColumnPicker.svelte`**

```svelte
<script lang="ts">
  import {Pin} from "lucide-svelte";

  let {
    allColumns,
    visibleColumns,
    pinnedNames = [],
    onToggle,
    onReset,
  }: {
    allColumns: {col: {name: string}; visible: boolean}[];
    visibleColumns: {name: string}[];
    pinnedNames?: string[];
    onToggle: (name: string, visible: boolean) => void;
    onReset: () => void;
  } = $props();

  let filter = $state("");

  const pinnedSet = $derived(new Set(pinnedNames));
  const visibleOrder = $derived(visibleColumns.map((c) => c.name));

  const ordered = $derived.by(() => {
    const byName = new Map(allColumns.map((e) => [e.col.name, e]));
    const visibleEntries = visibleOrder
      .map((n) => byName.get(n))
      .filter((e): e is {col: {name: string}; visible: boolean} => e !== undefined);
    const hiddenEntries = allColumns.filter((e) => !e.visible);
    return [...visibleEntries, ...hiddenEntries];
  });

  const filtered = $derived.by(() => {
    if (!filter) return ordered;
    const q = filter.toLowerCase();
    return ordered.filter((e) => e.col.name.toLowerCase().includes(q));
  });
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="absolute right-0 top-full mt-1 z-50 bg-surface border border-border rounded shadow-lg py-2 min-w-64"
  onclick={(e) => e.stopPropagation()}
  onkeydown={(e) => e.stopPropagation()}
>
  <div class="flex items-center justify-between px-3 pb-1.5 mb-1 border-b border-border">
    <span class="text-xs font-semibold uppercase tracking-wider text-muted">Columns</span>
    <button type="button" onclick={() => onReset()} class="text-xs text-muted hover:text-fg transition-colors">Reset</button>
  </div>
  <div class="px-2 pb-1.5">
    <input
      type="text"
      bind:value={filter}
      placeholder="Filter…"
      class="w-full px-2 py-1 text-sm bg-bg border border-border rounded focus:outline-none focus:border-accent"
    />
  </div>
  <div class="max-h-72 overflow-y-auto">
    {#each filtered as entry (entry.col.name)}
      {@const isPinned = pinnedSet.has(entry.col.name)}
      <label class="flex items-center gap-2 px-3 py-1 hover:bg-surface-hover cursor-pointer">
        <input
          type="checkbox"
          checked={entry.visible}
          disabled={isPinned}
          onchange={(e) => onToggle(entry.col.name, e.currentTarget.checked)}
          class="rounded border-border shrink-0"
        />
        <span class="flex-1 text-sm truncate {entry.visible ? '' : 'text-muted'}">
          {entry.col.name}
        </span>
        {#if isPinned}
          <Pin size={11} class="text-muted shrink-0" />
        {/if}
      </label>
    {/each}
    {#if filtered.length === 0}
      <div class="px-3 py-2 text-xs text-muted">No matches</div>
    {/if}
  </div>
</div>
```

- [ ] **Step 4: Run, confirm pass**

```bash
cd frontend && pnpm vitest run src/lib/__tests__/ColumnPicker.svelte.test.ts
```
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
jj new
jj desc -m "feat(ui): ColumnPicker — visibility-only popover with filter"
```

---

### Task 7: `ViewOptionsMenu.svelte`

**Files:**
- Create: `frontend/src/lib/components/ViewOptionsMenu.svelte`
- Create: `frontend/src/lib/__tests__/ViewOptionsMenu.svelte.test.ts`

- [ ] **Step 1: Write failing tests**

```ts
import {describe, it, expect, vi} from "vitest";
import {render, screen, fireEvent} from "@testing-library/svelte";
import ViewOptionsMenu from "$lib/components/ViewOptionsMenu.svelte";

describe("ViewOptionsMenu", () => {
  it("renders compact toggle and calls onCompactChange", async () => {
    const onCompactChange = vi.fn();
    render(ViewOptionsMenu, {props: {compact: false, onCompactChange, hasSparklines: false}});
    const toggle = screen.getByLabelText(/compact/i);
    await fireEvent.click(toggle);
    expect(onCompactChange).toHaveBeenCalledWith(true);
  });

  it("renders sparkline toggles only when hasSparklines is true", () => {
    render(ViewOptionsMenu, {props: {compact: false, onCompactChange: vi.fn(), hasSparklines: false}});
    expect(screen.queryByText("CPU")).toBeNull();

    render(ViewOptionsMenu, {props: {
      compact: false, onCompactChange: vi.fn(),
      hasSparklines: true, sparklineColumns: [], onSparklineToggle: vi.fn(),
    }});
    expect(screen.getByText("CPU")).toBeTruthy();
    expect(screen.getByText("Memory")).toBeTruthy();
  });

  it("toggles a sparkline column on click", async () => {
    const onSparklineToggle = vi.fn();
    render(ViewOptionsMenu, {props: {
      compact: false, onCompactChange: vi.fn(),
      hasSparklines: true, sparklineColumns: [], onSparklineToggle,
    }});
    await fireEvent.click(screen.getByLabelText("CPU"));
    expect(onSparklineToggle).toHaveBeenCalledWith(["CPU"]);
  });
});
```

- [ ] **Step 2: Run, confirm failure**

```bash
cd frontend && pnpm vitest run src/lib/__tests__/ViewOptionsMenu.svelte.test.ts
```
Expected: FAIL.

- [ ] **Step 3: Implement `ViewOptionsMenu.svelte`**

```svelte
<script lang="ts">
  let {
    compact,
    onCompactChange,
    hasSparklines = false,
    sparklineColumns = [],
    onSparklineToggle,
  }: {
    compact: boolean;
    onCompactChange: (value: boolean) => void;
    hasSparklines?: boolean;
    sparklineColumns?: string[];
    onSparklineToggle?: (columns: string[]) => void;
  } = $props();

  const availableSparklineCols = ["CPU", "Memory"];

  function toggleSparkline(col: string) {
    const next = sparklineColumns.includes(col)
      ? sparklineColumns.filter((c) => c !== col)
      : [...sparklineColumns, col];
    onSparklineToggle?.(next);
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="absolute right-0 top-full mt-1 z-50 bg-surface border border-border rounded shadow-lg py-2 min-w-44"
  onclick={(e) => e.stopPropagation()}
  onkeydown={(e) => e.stopPropagation()}
>
  <div class="px-3 pb-1 text-xs font-semibold uppercase tracking-wider text-muted">View</div>
  <label class="flex items-center gap-2 px-3 py-1 hover:bg-surface-hover cursor-pointer">
    <input
      type="checkbox"
      checked={compact}
      aria-label="Compact rows"
      onchange={(e) => onCompactChange(e.currentTarget.checked)}
      class="rounded border-border"
    />
    <span class="text-sm">Compact rows</span>
  </label>
  {#if hasSparklines}
    <div class="border-t border-border mt-1 pt-1.5">
      <div class="px-3 pb-1 text-xs font-semibold uppercase tracking-wider text-muted">Sparklines</div>
      {#each availableSparklineCols as col}
        <label class="flex items-center gap-2 px-3 py-1 hover:bg-surface-hover cursor-pointer">
          <input
            type="checkbox"
            checked={sparklineColumns.includes(col)}
            aria-label={col}
            onchange={() => toggleSparkline(col)}
            class="rounded border-border"
          />
          <span class="text-sm">{col}</span>
        </label>
      {/each}
    </div>
  {/if}
</div>
```

- [ ] **Step 4: Run, confirm pass**

```bash
cd frontend && pnpm vitest run src/lib/__tests__/ViewOptionsMenu.svelte.test.ts
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
jj new
jj desc -m "feat(ui): ViewOptionsMenu for compact + sparkline toggles"
```

---

### Task 8: `HeaderContextMenu.svelte`

**Files:**
- Create: `frontend/src/lib/components/HeaderContextMenu.svelte`
- Create: `frontend/src/lib/__tests__/HeaderContextMenu.svelte.test.ts`

- [ ] **Step 1: Write failing tests**

```ts
import {describe, it, expect, vi} from "vitest";
import {render, screen, fireEvent} from "@testing-library/svelte";
import HeaderContextMenu from "$lib/components/HeaderContextMenu.svelte";

function props(overrides: Record<string, unknown> = {}) {
  return {
    x: 100,
    y: 100,
    columnName: "Status",
    isPinned: false,
    canHide: true,
    onSort: vi.fn(),
    onAutoFit: vi.fn(),
    onTogglePin: vi.fn(),
    onHide: vi.fn(),
    onClose: vi.fn(),
    ...overrides,
  };
}

describe("HeaderContextMenu", () => {
  it("renders Sort asc/desc, Auto-fit, Pin, Hide for a normal column", () => {
    render(HeaderContextMenu, {props: props()});
    expect(screen.getByText(/sort ascending/i)).toBeTruthy();
    expect(screen.getByText(/sort descending/i)).toBeTruthy();
    expect(screen.getByText(/auto-?fit/i)).toBeTruthy();
    expect(screen.getByText(/pin to left/i)).toBeTruthy();
    expect(screen.getByText(/hide column/i)).toBeTruthy();
  });

  it("shows Unpin when isPinned=true and never offers Hide", () => {
    render(HeaderContextMenu, {props: props({isPinned: true, canHide: false})});
    expect(screen.getByText(/unpin/i)).toBeTruthy();
    expect(screen.queryByText(/hide column/i)).toBeNull();
  });

  it("calls onSort with 'asc' / 'desc'", async () => {
    const p = props();
    render(HeaderContextMenu, {props: p});
    await fireEvent.click(screen.getByText(/sort ascending/i));
    expect(p.onSort).toHaveBeenCalledWith("asc");
    await fireEvent.click(screen.getByText(/sort descending/i));
    expect(p.onSort).toHaveBeenCalledWith("desc");
  });

  it("calls onTogglePin and onClose on pin click", async () => {
    const p = props();
    render(HeaderContextMenu, {props: p});
    await fireEvent.click(screen.getByText(/pin to left/i));
    expect(p.onTogglePin).toHaveBeenCalled();
    expect(p.onClose).toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run, confirm failure**

```bash
cd frontend && pnpm vitest run src/lib/__tests__/HeaderContextMenu.svelte.test.ts
```
Expected: FAIL.

- [ ] **Step 3: Implement `HeaderContextMenu.svelte`**

```svelte
<script lang="ts">
  let {
    x,
    y,
    columnName,
    isPinned = false,
    canHide = true,
    onSort,
    onAutoFit,
    onTogglePin,
    onHide,
    onClose,
  }: {
    x: number;
    y: number;
    columnName: string;
    isPinned?: boolean;
    canHide?: boolean;
    onSort: (direction: "asc" | "desc") => void;
    onAutoFit: () => void;
    onTogglePin: () => void;
    onHide: () => void;
    onClose: () => void;
  } = $props();

  let menuEl = $state<HTMLDivElement | null>(null);

  $effect(() => {
    if (!menuEl) return;
    const rect = menuEl.getBoundingClientRect();
    const maxX = window.innerWidth - rect.width - 8;
    const maxY = window.innerHeight - rect.height - 8;
    if (x > maxX) menuEl.style.left = `${Math.max(0, maxX)}px`;
    if (y > maxY) menuEl.style.top = `${Math.max(0, maxY)}px`;
  });

  function clickAndClose(fn: () => void) {
    fn();
    onClose();
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  bind:this={menuEl}
  class="fixed z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-44"
  style="left:{x}px; top:{y}px"
  onclick={(e) => e.stopPropagation()}
  onkeydown={(e) => e.stopPropagation()}
>
  <div class="px-3 py-1 text-xs font-semibold text-muted truncate">{columnName}</div>
  <div class="border-t border-border my-1"></div>
  <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover" onclick={() => clickAndClose(() => onSort("asc"))}>Sort ascending</button>
  <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover" onclick={() => clickAndClose(() => onSort("desc"))}>Sort descending</button>
  <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover" onclick={() => clickAndClose(onAutoFit)}>Auto-fit width</button>
  <div class="border-t border-border my-1"></div>
  <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover" onclick={() => clickAndClose(onTogglePin)}>
    {isPinned ? "Unpin" : "Pin to left"}
  </button>
  {#if canHide}
    <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover" onclick={() => clickAndClose(onHide)}>Hide column</button>
  {/if}
</div>
```

- [ ] **Step 4: Run, confirm pass**

```bash
cd frontend && pnpm vitest run src/lib/__tests__/HeaderContextMenu.svelte.test.ts
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
jj new
jj desc -m "feat(ui): HeaderContextMenu for sort/auto-fit/pin/hide"
```

---

## Phase 4 — DataTable rebuild

### Task 9: Split DataTable header into pinned + main grids

**Files:**
- Modify: `frontend/src/lib/components/DataTable.svelte`

This is the highest-risk task. Do it standalone and visually verify before adding dnd.

- [ ] **Step 1: Add `pinnedColumns` and `mainColumns` derived from props**

Inside the `<script>` block, add a `pinnedNames` prop:

```ts
// Inside the props destructure
pinnedNames = [] as string[],
```

Then below `let resizing = ...`:

```ts
const pinnedColumns = $derived(visibleColumns.filter((c) => pinnedNames.includes(c.name)));
const mainColumns = $derived(visibleColumns.filter((c) => !pinnedNames.includes(c.name)));
```

Replace the existing single `gridTemplateCols` with two derived grids:

```ts
const pinnedGridCols = $derived.by(() => {
  const parts: string[] = [...prefixGridCols];
  for (const c of pinnedColumns) {
    parts.push(c.width ? `${c.width}px` : "minmax(20px, max-content)");
  }
  return parts.join(" ");
});

const mainGridCols = $derived.by(() => {
  const parts: string[] = [];
  for (const c of mainColumns) {
    parts.push(c.width ? `${c.width}px` : "minmax(20px, 1fr)");
  }
  parts.push(...suffixGridCols);
  return parts.join(" ");
});
```

- [ ] **Step 2: Restructure the header markup**

Replace the existing header div (line 167-205) with two siblings inside a flex row:

```svelte
<div class="flex sticky top-0 z-20 bg-bg text-xs font-semibold uppercase tracking-wider text-muted border-b border-border">
  <div
    class="grid sticky left-0 z-30 bg-bg pl-2"
    style="grid-template-columns: {pinnedGridCols}"
  >
    {#if headerPrefix}
      {@render headerPrefix()}
    {/if}
    {#each pinnedColumns as col}
      {@render headerCell(col)}
    {/each}
  </div>
  <div
    class="grid flex-1 pr-2"
    style="grid-template-columns: {mainGridCols}"
  >
    {#each mainColumns as col, i (col.name)}
      {@render headerCell(col, i === mainColumns.length - 1)}
    {/each}
    {#if headerSuffix}
      {@render headerSuffix()}
    {/if}
  </div>
</div>
```

Extract the per-header cell into a snippet at the top of the template:

```svelte
{#snippet headerCell(col, isLast = false)}
  <div class="relative" data-header-col={col.name}>
    <button
      type="button"
      onclick={() => toggleSort(col.name)}
      class="flex items-center gap-1 px-1 hover:text-fg transition-colors text-left w-full {compact ? 'py-1' : 'py-2'}"
    >
      {col.name}
      {#if sortState?.column === col.name}
        {#if sortState.direction === 'asc'}
          <ArrowUp size={10} />
        {:else}
          <ArrowDown size={10} />
        {/if}
      {:else}
        <ArrowUpDown size={10} class="opacity-30" />
      {/if}
    </button>
    {#if !isLast}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize bg-border/50 hover:bg-accent/70 z-20"
        onmousedown={(e) => startResize(e, col)}
        ondblclick={() => autoFit(col.name)}
      ></div>
    {/if}
  </div>
{/snippet}
```

- [ ] **Step 3: Mirror the structure in body rows**

Replace the inner row content (line 227-239) with the same flex layout:

```svelte
<div class="flex flex-1">
  <div
    class="grid sticky left-0 z-10 pl-2 {selectedRow?.(item) ? 'bg-accent/10' : 'bg-bg group-hover:bg-surface-hover'}"
    style="grid-template-columns: {pinnedGridCols}"
  >
    {#if rowPrefix}
      {@render rowPrefix({item})}
    {/if}
    {#each pinnedColumns as column}
      <div class="px-1 truncate text-sm {alignClass(column)}" data-col={column.name}>
        {@render cell({item, column})}
      </div>
    {/each}
  </div>
  <div
    class="grid flex-1 pr-2"
    style="grid-template-columns: {mainGridCols}"
  >
    {#each mainColumns as column}
      <div class="px-1 truncate text-sm {alignClass(column)}" data-col={column.name}>
        {@render cell({item, column})}
      </div>
    {/each}
    {#if rowSuffix}
      {@render rowSuffix({item})}
    {/if}
  </div>
</div>
```

Note the change: the row's outer `<div>` still has `display: flex` from `flex items-center`, but its child becomes a `.flex` div instead of `.grid`. Remove the existing `<div class="grid flex-1" ...>` wrapper.

- [ ] **Step 4: Type-check + build**

```bash
cd frontend && pnpm check && pnpm build
```
Expected: PASS.

- [ ] **Step 5: Smoke-test in dev**

```bash
task dev
```

Open the app, navigate to Pods. Verify: Name stays sticky on horizontal scroll, columns align, sort/resize still works. Try Deployments and a CRD. **Do not proceed until visually verified.**

- [ ] **Step 6: Commit**

```bash
jj new
jj desc -m "refactor(DataTable): split header/body into pinned + main grids"
```

---

### Task 10: Header drag-reorder with `svelte-dnd-action`

**Files:**
- Modify: `frontend/src/lib/components/DataTable.svelte`

- [ ] **Step 1: Add `onreorder` callback prop**

In props destructure, add:

```ts
onreorder,
```

with the type:

```ts
onreorder?: (names: string[]) => void;
```

- [ ] **Step 2: Wrap mainColumns header in dndzone**

Import at the top:

```ts
import {dndzone, type DndEvent} from "svelte-dnd-action";
```

Replace the `<div class="grid flex-1 pr-2" ...>` block in the header with a dndzone-wrapped version. Note `svelte-dnd-action` requires the items to have a unique `id` per item — use `name` as the id:

```svelte
<div
  class="grid flex-1 pr-2"
  style="grid-template-columns: {mainGridCols}"
  use:dndzone={{
    items: mainColumns.map((c) => ({id: c.name, ...c})),
    type: "table-columns",
    flipDurationMs: 150,
    dropTargetStyle: {outline: "2px dashed var(--color-accent, currentColor)"},
  }}
  onconsider={(e: CustomEvent<DndEvent>) => {
    // Visual-only intermediate state — we don't mutate visibleColumns here; the picker is the source of truth via store
    // To get smooth animations, we'd need a local mirror; for v1 we keep it simple and only react on finalize
  }}
  onfinalize={(e: CustomEvent<DndEvent>) => {
    const names = (e.detail.items as Array<{id: string}>).map((i) => i.id);
    onreorder?.(names);
  }}
>
  {#each mainColumns as col, i (col.name)}
    {@render headerCell(col, i === mainColumns.length - 1)}
  {/each}
  {#if headerSuffix}
    {@render headerSuffix()}
  {/if}
</div>
```

Note: the simple `consider` no-op means columns don't animate in flight. That's acceptable for v1. If it feels jarring during smoke-test, add a local `let liveColumns = $state<DataTableColumn[]>(...)` mirror that updates on `consider` and is reset from `mainColumns` on `finalize` (the store update will round-trip).

Mark the resize handle and sort button with `data-no-dnd` so dragging them doesn't initiate a column drag. In the `headerCell` snippet:

```svelte
<button type="button" onclick={...} data-no-dnd ...>
```

```svelte
<div data-no-dnd class="absolute right-0 top-0 bottom-0 w-1 ..." onmousedown={...} />
```

`svelte-dnd-action` ignores drag starts on elements with `data-no-dnd` set.

- [ ] **Step 3: Type-check + smoke test**

```bash
cd frontend && pnpm check
task dev
```

In the running app, drag a column header. Verify: drop indicator appears, drop reorders, `onreorder` callback fires (add a temporary `console.log` if needed). Verify resize and sort still work — clicking them should NOT initiate drag.

- [ ] **Step 4: Commit**

```bash
jj new
jj desc -m "feat(DataTable): drag-reorder column headers via svelte-dnd-action"
```

---

### Task 11: Header right-click menu integration

**Files:**
- Modify: `frontend/src/lib/components/DataTable.svelte`

- [ ] **Step 1: Add new props for pin/hide callbacks**

```ts
onTogglePin?: (name: string) => void;
onHideColumn?: (name: string) => void;
```

- [ ] **Step 2: Add `headerCtxMenu` state**

In the script:

```ts
import HeaderContextMenu from "./HeaderContextMenu.svelte";

let headerCtxMenu = $state<{x: number; y: number; columnName: string} | null>(null);

$effect(() => {
  if (!headerCtxMenu) return;
  const close = () => { headerCtxMenu = null; };
  const t = setTimeout(() => window.addEventListener("click", close, {once: true}), 0);
  return () => {
    clearTimeout(t);
    window.removeEventListener("click", close);
  };
});
```

- [ ] **Step 3: Wire `oncontextmenu` on the header button**

Update the `headerCell` snippet's button:

```svelte
<button
  type="button"
  onclick={() => toggleSort(col.name)}
  oncontextmenu={(e) => { e.preventDefault(); headerCtxMenu = { x: e.clientX, y: e.clientY, columnName: col.name } }}
  data-no-dnd
  ...
>
```

- [ ] **Step 4: Render `HeaderContextMenu` at the bottom of the template**

After the main `</div>` closing the table wrapper:

```svelte
{#if headerCtxMenu}
  <HeaderContextMenu
    x={headerCtxMenu.x}
    y={headerCtxMenu.y}
    columnName={headerCtxMenu.columnName}
    isPinned={pinnedNames.includes(headerCtxMenu.columnName)}
    canHide={headerCtxMenu.columnName !== "Name" && !pinnedNames.includes(headerCtxMenu.columnName)}
    onSort={(dir) => onsort?.(headerCtxMenu!.columnName, dir)}
    onAutoFit={() => autoFit(headerCtxMenu!.columnName)}
    onTogglePin={() => onTogglePin?.(headerCtxMenu!.columnName)}
    onHide={() => onHideColumn?.(headerCtxMenu!.columnName)}
    onClose={() => { headerCtxMenu = null }}
  />
{/if}
```

- [ ] **Step 5: Type-check + smoke test**

```bash
cd frontend && pnpm check
```

In `task dev`, right-click a column header. Menu appears. Each item triggers the right callback (verify with a temporary log if needed). Verify clicking outside dismisses the menu.

- [ ] **Step 6: Commit**

```bash
jj new
jj desc -m "feat(DataTable): right-click column header context menu"
```

---

### Task 12: Loading skeleton rows

**Files:**
- Modify: `frontend/src/lib/components/DataTable.svelte`

- [ ] **Step 1: Replace the loading branch**

Find the current:

```svelte
{#if loading}
  <div class="flex items-center justify-center py-12 text-sm text-muted">Loading...</div>
```

Replace with conditional skeleton (only on initial load — `items.length === 0`):

```svelte
{#if loading && items.length === 0}
  <div>
    {#each Array(8) as _, i}
      <div
        class="flex items-center px-2 border-b border-border/40"
        style="height: {rowHeight}px;"
      >
        <div class="grid flex-1" style="grid-template-columns: {pinnedGridCols}">
          {#each pinnedColumns as _, j}
            <div class="px-1"><div class="h-3 rounded bg-surface-hover animate-pulse" style="width: {50 + ((i + j) % 4) * 10}%"></div></div>
          {/each}
        </div>
        <div class="grid flex-1" style="grid-template-columns: {mainGridCols}">
          {#each mainColumns as _, j}
            <div class="px-1"><div class="h-3 rounded bg-surface-hover animate-pulse" style="width: {40 + ((i + j) % 5) * 10}%"></div></div>
          {/each}
        </div>
      </div>
    {/each}
  </div>
{:else if items.length === 0}
  ...
{/if}
```

- [ ] **Step 2: Smoke test**

```bash
task dev
```

Switch to a context with slow API. Verify skeletons render only on initial load; subsequent refreshes leave existing rows visible.

- [ ] **Step 3: Commit**

```bash
jj new
jj desc -m "feat(DataTable): skeleton rows during initial load"
```

---

### Task 13: `emptyAction` snippet

**Files:**
- Modify: `frontend/src/lib/components/DataTable.svelte`

- [ ] **Step 1: Add `emptyAction` snippet prop**

In props:

```ts
emptyAction?: Snippet;
```

- [ ] **Step 2: Render it in the empty branch**

Replace:

```svelte
{:else if items.length === 0}
  <div class="flex items-center justify-center py-12 text-sm text-muted">{emptyMessage}</div>
```

with:

```svelte
{:else if items.length === 0}
  <div class="flex flex-col items-center justify-center py-12 gap-3">
    <div class="text-sm text-muted">{emptyMessage}</div>
    {#if emptyAction}
      {@render emptyAction()}
    {/if}
  </div>
```

- [ ] **Step 3: Type-check**

```bash
cd frontend && pnpm check
```

- [ ] **Step 4: Commit**

```bash
jj new
jj desc -m "feat(DataTable): emptyAction snippet for actionable empty states"
```

---

## Phase 5 — ResourceList wiring

### Task 14: Swap menus + wire reorder in `ResourceList.svelte`

**Files:**
- Modify: `frontend/src/lib/components/ResourceList.svelte`

- [ ] **Step 1: Update imports**

Replace:

```ts
import ColumnMenu from "./ColumnMenu.svelte";
```

with:

```ts
import ColumnPicker from "./ColumnPicker.svelte";
import ViewOptionsMenu from "./ViewOptionsMenu.svelte";
import {Eye} from "lucide-svelte";
```

- [ ] **Step 2: Add `viewMenuOpen` state and effect**

```ts
let viewMenuOpen = $state(false);

$effect(() => {
  if (!viewMenuOpen) return;
  const close = () => { viewMenuOpen = false; };
  const timer = setTimeout(() => window.addEventListener("click", close, {once: true}), 0);
  return () => {
    clearTimeout(timer);
    window.removeEventListener("click", close);
  };
});
```

- [ ] **Step 3: Swap the toolbar Columns button and add View button**

Replace the existing `{#if columnMenuOpen}` block (lines ~437-449) with:

```svelte
{#if columnMenuOpen}
  <ColumnPicker
    visibleColumns={columnStore.visibleColumns}
    allColumns={columnStore.allColumns}
    pinnedNames={columnStore.pinnedNames()}
    onToggle={(name, visible) => columnStore.setColumnVisible(name, visible)}
    onReset={() => columnStore.reset()}
  />
{/if}
```

After the Columns button, add a View Options button:

```svelte
<div class="relative">
  <button
    type="button"
    onclick={() => viewMenuOpen = !viewMenuOpen}
    class="p-1 rounded hover:bg-surface-hover transition-colors"
    title="View options"
    aria-label="View options"
  >
    <Eye size={14} />
  </button>
  {#if viewMenuOpen}
    <ViewOptionsMenu
      compact={columnStore.compact}
      onCompactChange={(v) => columnStore.setCompact(v)}
      hasSparklines={sparklineGvrs.includes(gvr)}
      {sparklineColumns}
      {onSparklineToggle}
    />
  {/if}
</div>
```

- [ ] **Step 4: Pass `pinnedNames` + new callbacks to DataTable**

Update the `<DataTable>` opening tag:

```svelte
<DataTable
  ...existing props...
  pinnedNames={columnStore.pinnedNames()}
  onreorder={(names) => columnStore.reorderVisible(names)}
  onTogglePin={(name) => columnStore.setPinned(name, !columnStore.isPinned(name))}
  onHideColumn={(name) => columnStore.setColumnVisible(name, false)}
>
```

- [ ] **Step 5: Type-check + smoke test**

```bash
cd frontend && pnpm check
task dev
```

Verify: Columns popover shows checkboxes only (no arrows); View Options button opens compact + sparkline toggles; drag-reordering a header persists across navigation; right-click "Pin to left" / "Unpin" / "Hide column" work.

- [ ] **Step 6: Commit**

```bash
jj new
jj desc -m "feat(ResourceList): swap ColumnMenu for ColumnPicker + ViewOptionsMenu, wire reorder + pin/hide"
```

---

### Task 15: Row context menu — Copy name / Copy YAML / View YAML / Edit YAML

**Files:**
- Modify: `frontend/src/lib/components/ResourceList.svelte`

- [ ] **Step 1: Import helpers**

```ts
import {itemToYaml} from "$lib/utils/yamlClipboard";
import {bottomPanelStore} from "$lib/stores/bottom-panel.svelte";
```

- [ ] **Step 2: Add ctx menu items above the existing Browse Volume / plugin / Delete items**

Find the `{#if ctxMenu}` block (line ~625). Insert new buttons at the top of the menu div, before plugin items:

```svelte
<button
  type="button"
  class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
  onclick={() => {
    if (ctxMenu) {
      const name = ctxMenu.item.metadata?.name as string | undefined;
      if (name) {
        navigator.clipboard.writeText(name);
        notificationStore.push("Copied name", "info");
      }
      ctxMenu = null;
    }
  }}
>Copy name</button>

<button
  type="button"
  class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
  onclick={() => {
    if (ctxMenu) {
      navigator.clipboard.writeText(itemToYaml(ctxMenu.item));
      notificationStore.push("Copied YAML", "info");
      ctxMenu = null;
    }
  }}
>Copy YAML</button>

<button
  type="button"
  class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
  onclick={() => {
    if (ctxMenu) {
      const item = ctxMenu.item;
      bottomPanelStore.addTab({
        kind: "yaml",
        resourceKind: (item.kind as string) ?? "",
        resourceName: (item.metadata?.name as string) ?? "",
        ctxName: contextName,
        gvr,
        namespace: (item.metadata?.namespace as string) ?? "",
        name: (item.metadata?.name as string) ?? "",
        obj: item as Record<string, unknown>,
        editable: false,
      });
      ctxMenu = null;
    }
  }}
>View YAML</button>

{#if canMutate}
  <button
    type="button"
    class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
    onclick={() => {
      if (ctxMenu) {
        const item = ctxMenu.item;
        bottomPanelStore.addTab({
          kind: "yaml",
          resourceKind: (item.kind as string) ?? "",
          resourceName: (item.metadata?.name as string) ?? "",
          ctxName: contextName,
          gvr,
          namespace: (item.metadata?.namespace as string) ?? "",
          name: (item.metadata?.name as string) ?? "",
          obj: item as Record<string, unknown>,
          editable: true,
        });
        ctxMenu = null;
      }
    }}
  >Edit YAML</button>
{/if}

<div class="border-t border-border my-1"></div>
```

- [ ] **Step 3: Smoke test**

```bash
task dev
```

Right-click a pod row → Copy name (verify clipboard); Copy YAML; View YAML opens the bottom panel YAML tab; Edit YAML opens it with edit enabled.

- [ ] **Step 4: Commit**

```bash
jj new
jj desc -m "feat(ResourceList): Copy name/YAML, View/Edit YAML in row ctx menu"
```

---

### Task 16: Row context menu — View logs / Open terminal

**Files:**
- Modify: `frontend/src/lib/components/ResourceList.svelte`

- [ ] **Step 1: Add pod-like detection**

Above the toolbar snippet, add:

```ts
const POD_OWNER_GVRS = new Set([
  "apps.v1.deployments",
  "apps.v1.statefulsets",
  "apps.v1.daemonsets",
  "apps.v1.replicasets",
  "batch.v1.jobs",
  "batch.v1.cronjobs",
]);

const isPod = $derived(gvr === "core.v1.pods");
const isPodOwner = $derived(POD_OWNER_GVRS.has(gvr));
const canViewLogs = $derived(isPod || isPodOwner);
const canOpenTerminal = $derived(isPod && canMutate);
```

- [ ] **Step 2: Add View logs and Open terminal items to the ctx menu**

After the YAML group (and the `<div class="border-t border-border my-1"></div>` you added in Task 15), insert:

```svelte
{#if canViewLogs}
  <button
    type="button"
    class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
    onclick={() => {
      if (ctxMenu) {
        const item = ctxMenu.item;
        bottomPanelStore.addTab({
          kind: isPod ? "logs" : "aggregate-logs",
          resourceKind: (item.kind as string) ?? "",
          resourceName: (item.metadata?.name as string) ?? "",
          ctxName: contextName,
          gvr,
          namespace: (item.metadata?.namespace as string) ?? "",
          name: (item.metadata?.name as string) ?? "",
          obj: item as Record<string, unknown>,
        });
        ctxMenu = null;
      }
    }}
  >View logs</button>
{/if}

{#if canOpenTerminal}
  <button
    type="button"
    class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
    onclick={() => {
      if (ctxMenu) {
        const item = ctxMenu.item;
        bottomPanelStore.addTab({
          kind: "terminal",
          resourceKind: (item.kind as string) ?? "",
          resourceName: (item.metadata?.name as string) ?? "",
          ctxName: contextName,
          gvr,
          namespace: (item.metadata?.namespace as string) ?? "",
          name: (item.metadata?.name as string) ?? "",
          obj: item as Record<string, unknown>,
        });
        ctxMenu = null;
      }
    }}
  >Open terminal</button>
{/if}

{#if canViewLogs || canOpenTerminal}
  <div class="border-t border-border my-1"></div>
{/if}
```

- [ ] **Step 3: Smoke test**

```bash
task dev
```

Pods page: right-click → View logs and Open terminal both render and open bottom panel tabs of the right kind. Deployments page: View logs renders (aggregate-logs), Open terminal does not. ConfigMaps page: neither renders.

- [ ] **Step 4: Commit**

```bash
jj new
jj desc -m "feat(ResourceList): View logs + Open terminal in row ctx menu"
```

---

### Task 17: Smart empty state with Clear filters

**Files:**
- Modify: `frontend/src/lib/components/ResourceList.svelte`

- [ ] **Step 1: Compute dynamic empty message**

In the script section, near other `$derived`:

```ts
const hasActiveFilters = $derived(searchTerms.length > 0);
const emptyMessage = $derived(hasActiveFilters ? "No resources match these filters" : "No resources found");
```

- [ ] **Step 2: Pass `emptyMessage` and `emptyAction` snippet to DataTable**

Update the DataTable opening tag:

```svelte
<DataTable
  ...existing props...
  {emptyMessage}
>
```

Add the snippet inside the `<DataTable>` element (alongside the existing `{#snippet toolbar()}`):

```svelte
{#snippet emptyAction()}
  {#if hasActiveFilters}
    <button
      type="button"
      onclick={() => { searchQuery = ''; searchTerms = []; }}
      class="px-3 py-1.5 text-sm border border-border rounded hover:bg-surface-hover transition-colors"
    >
      Clear filters
    </button>
  {/if}
{/snippet}
```

- [ ] **Step 3: Smoke test**

```bash
task dev
```

Type a non-matching search term — empty message shows "No resources match these filters" with a Clear filters button. Click it — list returns. Without filters, button does not appear.

- [ ] **Step 4: Commit**

```bash
jj new
jj desc -m "feat(ResourceList): smart empty state with Clear filters action"
```

---

## Phase 6 — Settings + cleanup

### Task 18: Settings page shows pinned columns

**Files:**
- Modify: `frontend/src/routes/settings/ColumnSettings.svelte`

- [ ] **Step 1: Render pinned alongside order/sort**

Inside the `{#each gvrKeys as gvr}` loop, after the existing sort line, add:

```svelte
{#if prefs?.pinned && prefs.pinned.length > 0}
  <div class="text-xs text-muted-foreground">
    Pinned: {#each prefs.pinned as p, i}
      <span class="text-fg font-mono">{p}</span>{#if i < prefs.pinned.length - 1}, {/if}
    {/each}
  </div>
{/if}
```

- [ ] **Step 2: Type-check**

```bash
cd frontend && pnpm check
```

- [ ] **Step 3: Smoke test**

```bash
task dev
```

Pin a column in any resource list. Navigate to Settings → Column Preferences. Verify the pinned column is shown.

- [ ] **Step 4: Commit**

```bash
jj new
jj desc -m "feat(settings): show pinned columns in ColumnSettings"
```

---

### Task 19: Confirm + delete legacy `ColumnMenu` files

Per repo rule: **never delete files without explicit user confirmation.** This task is a checkpoint, not a free action.

**Files (pending confirmation):**
- Delete: `frontend/src/lib/components/ColumnMenu.svelte`
- Delete: `frontend/src/lib/__tests__/ColumnMenu.svelte.test.ts`

- [ ] **Step 1: Verify no remaining imports**

```bash
command grep -rn "ColumnMenu" frontend/src
```
Expected: zero results.

- [ ] **Step 2: Ask user to confirm deletion**

> "ColumnMenu.svelte and its test file are no longer imported anywhere. OK to delete?"

- [ ] **Step 3: On approval, delete and run the full frontend test suite**

```bash
rm frontend/src/lib/components/ColumnMenu.svelte
rm frontend/src/lib/__tests__/ColumnMenu.svelte.test.ts
cd frontend && pnpm test
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
jj new
jj desc -m "chore(ui): remove legacy ColumnMenu (replaced by ColumnPicker + ViewOptionsMenu)"
```

---

## Final verification

After Task 19 (or Task 18 if deletion is deferred):

- [ ] **Run full frontend test suite**

```bash
cd frontend && pnpm test
```
Expected: PASS.

- [ ] **Run frontend type check**

```bash
cd frontend && pnpm check
```
Expected: PASS.

- [ ] **Run Go tests touching config**

```bash
go test ./internal/config/ -v
```
Expected: PASS.

- [ ] **Visual smoke pass**

`task dev`. Walk through: Pods, Deployments, a CRD with many printer columns. Confirm:
- Drag a header → reorders.
- Right-click header → menu, Pin/Unpin/Hide/Sort/Auto-fit all work.
- Name stays sticky on horizontal scroll.
- ColumnPicker filter + checkbox toggle.
- ViewOptionsMenu compact + sparkline (on Pods).
- Right-click row → all menu items in correct visibility per resource kind.
- Empty state with filters shows Clear filters.
- Skeleton on first navigation to a new GVR; existing items don't flash on refetch.

---

## Self-review notes

- **Spec coverage:** All 7 goals from spec §Goals have at least one task; each new file in spec §12 has a creation task; persistence change (§10) covered in Task 1; library decision (§11) executed in Task 3.
- **Naming consistency:** `reorderVisible`, `setPinned`, `isPinned`, `pinnedNames` consistent across Task 2 (store) and Tasks 9, 10, 11, 14 (consumers). `onreorder`, `onTogglePin`, `onHideColumn` consistent between Tasks 10, 11 (DataTable) and Task 14 (ResourceList).
- **No placeholders detected.**
- **TDD where high-value** (store, helpers, focused components). Smoke-only for DataTable (it's a structural refactor where E2E behavior matters more than unit assertions). Existing `ResourceList.svelte.test.ts` will need a small update if it asserts on the old `ColumnMenu` import — engineer to verify at Task 14.
