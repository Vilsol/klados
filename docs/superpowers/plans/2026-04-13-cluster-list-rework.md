# Cluster List Rework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the card-based cluster list with a dense table, using a generic DataTable component extracted from ResourceList.

**Architecture:** Extract a generic `DataTable.svelte` from `ResourceList.svelte` that handles virtual scrolling, grid layout, sorting, and resizing. Both `ResourceList` and `ClusterList` become consumers of `DataTable`, providing cell content via Svelte snippets. `ColumnMenu` becomes prop-driven so both consumers can reuse it.

**Tech Stack:** Svelte 5 (runes, snippets), TanStack Virtual, Tailwind v4 custom tokens, existing `columnStore` + config service for persistence.

**Note:** Auto-reconnect (spec section 5) is already fully implemented in the Go backend (`AppService.ServiceStartup` reconnects from `session.ConnectedClusters`, `ClusterService.Connect/Disconnect` updates the list). No work needed there.

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `frontend/src/lib/components/DataTable.svelte` | **Create** | Generic virtual table: TanStack Virtual, CSS grid, sort headers, resize handles, loading/empty/error states |
| `frontend/src/lib/components/ColumnMenu.svelte` | **Modify** | Change from reading global `columnStore` to accepting props |
| `frontend/src/lib/components/ResourceList.svelte` | **Modify** | Refactor to consume DataTable, provide K8s-specific cell/toolbar/suffix snippets |
| `frontend/src/routes/ClusterList.svelte` | **Modify** | Replace cards with DataTable, local column state, row-click navigation |

---

### Task 1: Create DataTable.svelte and refactor ColumnMenu

Extract the generic virtual table rendering from `ResourceList.svelte` into a new `DataTable.svelte` component, and make `ColumnMenu.svelte` prop-driven.

**Files:**
- Create: `frontend/src/lib/components/DataTable.svelte`
- Modify: `frontend/src/lib/components/ColumnMenu.svelte`

- [ ] **Step 1: Create `DataTable.svelte`**

This component extracts the following from ResourceList:
- TanStack Virtual setup (`createVirtualizer`, `rowHeight`, `scrollContainer`)
- CSS grid layout (`gridTemplateCols` computation from `prefixGridCols` + visible columns + `suffixGridCols`)
- Column header rendering (name, sort icons with `ArrowUpDown`/`ArrowUp`/`ArrowDown`, resize handles)
- Resize logic (`startResize`, `onResizeMove`, `onResizeUp`, `autoFit`, `snapAllColumnsToPixels`)
- Sort toggle (`toggleSort`)
- Row container (absolute positioning, translateY, hover styling)
- Loading / empty / error states

**Props interface:**

```svelte
<script lang="ts" generics="T">
  import {createVirtualizer} from "@tanstack/svelte-virtual";
  import {ArrowUpDown, ArrowUp, ArrowDown} from "lucide-svelte";
  import type {Snippet} from "svelte";

  type DataTableColumn = {
    name: string;
    width?: number;
    align?: "left" | "right" | "center";
  };

  let {
    items,
    visibleColumns,
    sortState = null,
    compact = false,
    loading = false,
    error = null,
    emptyMessage = "No items found",
    prefixGridCols = [],
    suffixGridCols = [],
    toolbar,
    cell,
    headerPrefix,
    headerSuffix,
    rowPrefix,
    rowSuffix,
    selectedRow,
    onsort,
    onresize,
    onrowclick,
    oncontextmenu,
    scrollContainer = $bindable<HTMLDivElement | undefined>(undefined),
  }: {
    items: T[];
    visibleColumns: DataTableColumn[];
    sortState?: {column: string; direction: "asc" | "desc"} | null;
    compact?: boolean;
    loading?: boolean;
    error?: string | null;
    emptyMessage?: string;
    prefixGridCols?: string[];
    suffixGridCols?: string[];
    toolbar?: Snippet;
    cell: Snippet<[{item: T; column: DataTableColumn}]>;
    headerPrefix?: Snippet;
    headerSuffix?: Snippet;
    rowPrefix?: Snippet<[{item: T}]>;
    rowSuffix?: Snippet<[{item: T}]>;
    selectedRow?: (item: T) => boolean;
    onsort?: (column: string, direction: "asc" | "desc") => void;
    onresize?: (column: string, width: number) => void;
    onrowclick?: (item: T) => void;
    oncontextmenu?: (event: MouseEvent, item: T) => void;
    scrollContainer?: HTMLDivElement;
  } = $props();
```

Key details:
- `selectedRow` is a function that returns true if the item should be highlighted (replaces the `isSelected` check in ResourceList that compared `selectedName`)
- `headerPrefix`/`headerSuffix` snippets render header cells for the prefix/suffix grid columns (e.g. the select-all checkbox header, empty action column header)
- The `alignClass` helper moves into DataTable since it's generic (left/right/center)
- DataTable does NOT render cell content — it calls `{@render cell({item, column})}` for each visible column
- Resize state (`resizing`), resize handlers, and `snapAllColumnsToPixels`/`autoFit` move here. `onresize` callback fires when a column is resized, so the consumer can persist the width.

The template structure mirrors ResourceList's current layout:

```svelte
<div class="flex flex-col h-full overflow-hidden isolate">
  <!-- Toolbar -->
  {#if toolbar}
    <div class="flex items-center gap-2 px-3 py-2 border-b border-border shrink-0">
      {@render toolbar()}
    </div>
  {/if}

  <!-- Error -->
  {#if error}
    <div class="p-4 text-sm text-destructive">{error}</div>
  {:else}
    <div bind:this={scrollContainer} class="flex-1 overflow-auto">
      <!-- Header row -->
      <div class="grid text-xs font-semibold uppercase tracking-wider text-muted border-b border-border sticky top-0 z-20 bg-bg px-2"
        style="grid-template-columns: {gridTemplateCols}">
        {#if headerPrefix}{@render headerPrefix()}{/if}
        {#each visibleColumns as col, i}
          <!-- Sort button + resize handle, same markup as current ResourceList -->
        {/each}
        {#if headerSuffix}{@render headerSuffix()}{/if}
      </div>

      <!-- Loading / empty / rows -->
      {#if loading}
        <div class="flex items-center justify-center py-12 text-sm text-muted">Loading...</div>
      {:else if items.length === 0}
        <div class="flex items-center justify-center py-12 text-sm text-muted">{emptyMessage}</div>
      {:else}
        <div style="height: {$virtualizer.getTotalSize()}px; position: relative;">
          {#each $virtualizer.getVirtualItems() as row (row.index)}
            {@const item = items[row.index]}
            <div
              class="absolute top-0 left-0 min-w-full flex items-center px-2 transition-colors group
                {selectedRow?.(item) ? 'bg-accent/10 border-l-2 border-accent' : 'hover:bg-surface-hover border-l-2 border-transparent'}
                {onrowclick ? 'cursor-pointer' : ''}"
              style="transform: translateY({row.start}px); height: {rowHeight}px;"
              tabindex={onrowclick ? 0 : undefined}
              onclick={() => onrowclick?.(item)}
              onkeydown={(e) => { if (e.key === 'Enter') onrowclick?.(item) }}
              oncontextmenu={oncontextmenu ? (e) => { e.preventDefault(); e.stopPropagation(); oncontextmenu?.(e, item) } : undefined}
            >
              <div class="grid flex-1" style="grid-template-columns: {gridTemplateCols}">
                {#if rowPrefix}{@render rowPrefix({item})}{/if}
                {#each visibleColumns as column}
                  <div class="px-1 truncate text-sm {alignClass(column)}" data-col={column.name}>
                    {@render cell({item, column})}
                  </div>
                {/each}
                {#if rowSuffix}{@render rowSuffix({item})}{/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>
```

Export the `DataTableColumn` type from the `<script>` block so consumers can import it.

- [ ] **Step 2: Refactor `ColumnMenu.svelte` to be prop-driven**

Replace the global `columnStore` import with props:

```svelte
<script lang="ts">
  import {ArrowUp, ArrowDown} from "lucide-svelte";

  let {
    visibleColumns,
    allColumns,
    compact,
    onToggle,
    onMove,
    onReset,
    onCompactChange,
    sparklineGvrs = [],
    sparklineColumns = [],
    onSparklineToggle,
    gvr,
  }: {
    visibleColumns: {name: string}[];
    allColumns: {col: {name: string}; visible: boolean}[];
    compact: boolean;
    onToggle: (name: string, visible: boolean) => void;
    onMove: (name: string, direction: "up" | "down") => void;
    onReset: () => void;
    onCompactChange: (value: boolean) => void;
    sparklineGvrs?: string[];
    sparklineColumns?: string[];
    onSparklineToggle?: (columns: string[]) => void;
    gvr?: string;
  } = $props();
```

Replace all `columnStore.xxx` references in the template:
- `columnStore.visibleColumns` → `visibleColumns`
- `columnStore.allColumns` → `allColumns`
- `columnStore.compact` → `compact`
- `columnStore.setColumnVisible(...)` → `onToggle(...)`
- `columnStore.moveColumn(...)` → `onMove(...)`
- `columnStore.reset()` → `onReset()`
- `columnStore.setCompact(...)` → `onCompactChange(...)`

The sparkline section only renders when `gvr` is provided:
```svelte
{#if gvr && sparklineGvrs.includes(gvr)}
```

The `hasSparklines` derived becomes: `const hasSparklines = $derived(gvr ? sparklineGvrs.includes(gvr) : false)`.

- [ ] **Step 3: Verify the app builds**

Run: `cd frontend && pnpm check`

At this point DataTable and ColumnMenu are created/modified but not consumed yet. The build should pass because ColumnMenu is only used inside ResourceList (which still imports the old way — will be updated in Task 2).

- [ ] **Step 4: Commit**

```bash
jj desc -m "Extract generic DataTable component and make ColumnMenu prop-driven"
```

---

### Task 2: Refactor ResourceList to consume DataTable

Wire ResourceList to use DataTable, providing all K8s-specific rendering via snippets. The rendered output should be pixel-identical.

**Files:**
- Modify: `frontend/src/lib/components/ResourceList.svelte`

- [ ] **Step 1: Refactor ResourceList.svelte**

**Remove** from ResourceList (now in DataTable):
- `createVirtualizer` import and virtualizer setup
- `ArrowUpDown`, `ArrowUp`, `ArrowDown` imports (keep `Trash2`, `RefreshCw`, `Columns3`, `Check`, `Minus`, `Download`)
- `rowHeight` derived
- `gridTemplateCols` derived
- `toggleSort` function
- `alignClass` function
- `startResize`, `onResizeMove`, `onResizeUp`, `snapAllColumnsToPixels`, `autoFit` functions
- `resizing` state
- The entire outer `<div class="flex flex-col h-full overflow-hidden isolate">` template — replaced by `<DataTable>`

**Keep** in ResourceList:
- All imports except the ones moved to DataTable
- `itemKey`, `now` timer, keyboard shortcut registration
- `searchTerms`, `searchQuery`, `deleteTarget`, `confirmOpen`, `ctxMenu`, `ctxMenuEl`, `columnMenuOpen`, `exportMenuOpen` state
- `pluginColumns`, `pluginMenuItems`, `basePluginURL` deriveds
- `ctxMenu` effects, `columnMenuOpen` effect, `exportMenuOpen` effect, scroll-to-top effect
- `filtered` derived (search + namespace filter + sort)
- Selection-related deriveds and effects
- `renderCell`, `renderValue`, `badgeClass` functions
- `getSparklinePoints`, `tooManyForSparklines`
- `confirmDelete`, `requestDelete` functions
- Context menu template
- ConfirmDialog

**New structure:**

```svelte
<script lang="ts">
  import DataTable, {type DataTableColumn} from "./DataTable.svelte";
  // ... rest of existing imports minus moved ones ...

  // ... all existing state/deriveds/functions that stay ...

  // Compute suffix grid cols for plugin + sparkline + action columns
  const suffixGridCols = $derived.by(() => {
    const parts: string[] = [];
    for (const _ of pluginColumns) parts.push("1fr");
    for (const _ of sparklineColumns) parts.push("80px");
    parts.push("36px"); // action column
    return parts;
  });

  const prefixGridCols = $derived(canMutate ? ["36px"] : []);
</script>

<DataTable
  items={filtered}
  visibleColumns={columnStore.visibleColumns}
  sortState={columnStore.sortState}
  compact={columnStore.compact}
  {loading}
  {error}
  emptyMessage="No resources found"
  {prefixGridCols}
  {suffixGridCols}
  bind:scrollContainer
  selectedRow={(item) => selectedName === `${item.metadata?.name ?? ''}/${item.metadata?.namespace ?? ''}`}
  onsort={(col, dir) => columnStore.setSort(col, dir)}
  onresize={(col, width) => columnStore.resizeColumn(col, width)}
  onrowclick={onselect ? (item) => onselect?.(item) : undefined}
  oncontextmenu={(e, item) => { ctxMenu = { x: e.clientX, y: e.clientY, item } }}
>
  {#snippet toolbar()}
    <SmartSearch {items} bind:value={searchQuery} ontermschange={(t) => { searchTerms = t }} />
    <SavedFilterDropdown {gvr} {contextName} currentQuery={searchQuery} onapply={(q) => { searchQuery = q }} />
    <span class="text-xs text-muted">{filtered.length} items</span>
    <!-- export menu dropdown (same markup as before) -->
    <div class="relative">
      <button type="button" onclick={() => exportMenuOpen = !exportMenuOpen}
        class="p-1 rounded hover:bg-surface-hover transition-colors" title="Export visible" aria-label="Export visible">
        <Download size={14} />
      </button>
      {#if exportMenuOpen}
        <div class="absolute top-full mt-1 right-0 z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-24">
          <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
            onclick={() => { exportItems(filtered, gvr, 'yaml'); exportMenuOpen = false }}>YAML</button>
          <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
            onclick={() => { exportItems(filtered, gvr, 'json'); exportMenuOpen = false }}>JSON</button>
        </div>
      {/if}
    </div>
    <!-- column menu -->
    <div class="relative">
      <button type="button" onclick={() => columnMenuOpen = !columnMenuOpen}
        class="p-1 rounded hover:bg-surface-hover transition-colors" title="Manage columns" aria-label="Manage columns">
        <Columns3 size={14} />
      </button>
      {#if columnMenuOpen}
        <ColumnMenu
          visibleColumns={columnStore.visibleColumns}
          allColumns={columnStore.allColumns}
          compact={columnStore.compact}
          onToggle={(name, visible) => columnStore.setColumnVisible(name, visible)}
          onMove={(name, dir) => columnStore.moveColumn(name, dir)}
          onReset={() => columnStore.reset()}
          onCompactChange={(v) => columnStore.setCompact(v)}
          {gvr} {sparklineGvrs} {sparklineColumns} {onSparklineToggle}
        />
      {/if}
    </div>
    {#if onrefresh}
      <button type="button" onclick={onrefresh}
        class="p-1 rounded hover:bg-surface-hover transition-colors" title="Refresh" aria-label="Refresh">
        <RefreshCw size={14} class={loading ? 'animate-spin' : ''} />
      </button>
    {/if}
  {/snippet}

  {#snippet headerPrefix()}
    {#if canMutate}
      <div class="flex items-center justify-center {columnStore.compact ? 'py-1' : 'py-2'}">
        <button type="button"
          onclick={() => { if (allVisibleSelected) selectionStore.deselectAll(); else selectionStore.selectAll(filteredKeys, filteredItemsByKey) }}
          class="w-4 h-4 rounded border border-border flex items-center justify-center hover:border-accent transition-colors
            {allVisibleSelected || someVisibleSelected ? 'bg-accent border-accent' : ''}"
          aria-label={allVisibleSelected ? 'Deselect all' : 'Select all'}>
          {#if allVisibleSelected}<Check size={10} class="text-accent-fg" />
          {:else if someVisibleSelected}<Minus size={10} class="text-accent-fg" />{/if}
        </button>
      </div>
    {/if}
  {/snippet}

  {#snippet headerSuffix()}
    <!-- empty headers for plugin cols, sparkline cols, and action col -->
    {#each pluginColumns as pcol (pcol.id)}<div class="{columnStore.compact ? 'py-1' : 'py-2'} px-1">{pcol.label}</div>{/each}
    {#each sparklineColumns as scol}<div class="{columnStore.compact ? 'py-1' : 'py-2'} px-1">{scol}</div>{/each}
    <div></div>
  {/snippet}

  {#snippet rowPrefix({ item })}
    {#if canMutate}
      {@const key = itemKey(item)}
      <div class="flex items-center justify-center" onclick={(e) => e.stopPropagation()} role="none">
        <button type="button"
          onclick={(e) => { e.stopPropagation(); if (e.shiftKey) selectionStore.selectRange(key, filteredKeys, filteredItemsByKey); else selectionStore.toggle(key, item) }}
          class="w-4 h-4 rounded border border-border flex items-center justify-center hover:border-accent transition-colors
            {selectionStore.isSelected(key) ? 'bg-accent border-accent' : ''}"
          aria-label={selectionStore.isSelected(key) ? 'Deselect' : 'Select'}>
          {#if selectionStore.isSelected(key)}<Check size={10} class="text-accent-fg" />{/if}
        </button>
      </div>
    {/if}
  {/snippet}

  {#snippet cell({ item, column: col })}
    {@const value = renderCell(col, item)}
    {#if col.renderType === 'controlledBy'}
      {@const ref = getControllerRef(item)}
      {#if ref}
        {#if onopenowner && clusterStore.resolveOwnerGVR(ref.apiVersion, ref.kind)}
          <button type="button" class="text-accent hover:underline cursor-pointer" title="{ref.kind}/{ref.name}"
            onclick={(e) => { e.stopPropagation(); onopenowner?.(ref, item.metadata?.namespace ?? '') }}>{ref.kind}</button>
        {:else}
          <span title="{ref.kind}/{ref.name}">{ref.kind}</span>
        {/if}
      {/if}
    {:else if col.renderType === 'badge'}
      <span class="px-1.5 py-0.5 text-xs rounded border {badgeClass(value)}" title={renderValue(value, col.renderType)}>
        {renderValue(value, col.renderType)}</span>
    {:else}
      <span class={col.renderType === 'age' ? 'text-muted' : ''} title={renderValue(value, col.renderType)}>
        {renderValue(value, col.renderType)}</span>
    {/if}
    {#if col.name === 'Namespace'}
      <!-- Namespace click-to-filter is handled by wrapping the cell div -->
    {/if}
  {/snippet}

  {#snippet rowSuffix({ item })}
    <!-- Plugin columns -->
    {#each pluginColumns as pcol (pcol.id)}
      <div class="px-1 flex items-center overflow-hidden text-sm">
        {#if basePluginURL}
          {#await loadPluginComponent(pcol.pluginName, pcol.component, basePluginURL) then Cmp}
            {#if Cmp}<Cmp resource={item} />{/if}
          {/await}
        {/if}
      </div>
    {/each}
    <!-- Sparkline columns -->
    {#each sparklineColumns as scol}
      <div class="px-1 flex items-center overflow-hidden">
        {#if tooManyForSparklines}
          <span class="text-xs text-muted" title="Sparklines disabled for >200 resources">Too many</span>
        {:else}
          {@const pts = getSparklinePoints(item.metadata?.name ?? '', scol)}
          {#if pts.length > 0}<Sparkline points={pts} height={20} />
          {:else}<div style="height: 20px;"></div>{/if}
        {/if}
      </div>
    {/each}
    <!-- Action column -->
    <div class="flex items-center justify-end gap-1">
      {#if rowActions}
        {#each rowActions(item) as action}
          <button type="button" onclick={(e) => { e.stopPropagation(); action.onClick() }}
            class="p-1 rounded opacity-0 group-hover:opacity-60 hover:!opacity-100 transition-all {action.variant === 'destructive' ? 'hover:text-destructive' : 'hover:text-fg'}"
            title={action.label} aria-label={action.label}>
            {#if action.icon}<action.icon size={13} />{:else}<span class="text-xs">{action.label}</span>{/if}
          </button>
        {/each}
      {:else if canMutate}
        <button type="button" onclick={(e) => { e.stopPropagation(); requestDelete(item) }}
          class="p-1 rounded opacity-0 group-hover:opacity-60 hover:!opacity-100 hover:text-destructive transition-all"
          title="Delete" aria-label="Delete {item.metadata?.name}">
          <Trash2 size={13} />
        </button>
      {/if}
    </div>
  {/snippet}
</DataTable>

<!-- Context menu (same as before, outside DataTable) -->
{#if ctxMenu}
  <!-- ... existing context menu markup unchanged ... -->
{/if}

<ConfirmDialog bind:open={confirmOpen} title="Delete resource"
  message="Delete {deleteTarget?.name}? This action cannot be undone."
  confirmLabel="Delete" onconfirm={confirmDelete} />
```

**Important detail on Namespace click-to-filter:** In the current ResourceList, the Namespace column `<div>` has a special `onclick` and `cursor-pointer` class. Since DataTable wraps each cell in `<div class="px-1 truncate text-sm {alignClass(col)}" data-col={col.name}>`, the namespace click behavior needs to be inside the `cell` snippet. Handle it by wrapping namespace cell content in a clickable span:

```svelte
{#snippet cell({ item, column: col })}
  {@const value = renderCell(col, item)}
  {#if col.name === 'Namespace'}
    <button type="button" class="hover:text-accent cursor-pointer"
      onclick={(e) => { e.stopPropagation(); clusterStore.setNamespaces(contextName, [String(value)]) }}>
      {renderValue(value, col.renderType)}
    </button>
  {:else if col.renderType === 'controlledBy'}
    <!-- ... rest as above ... -->
```

Note: The `cell` snippet receives `DataTableColumn` but ResourceList needs the full `ColumnDef` (with `expr`, `renderType`). Since `DataTableColumn` is a subset, ResourceList can look up the full column def from `columnStore.visibleColumns` by matching `column.name`. Or simpler: pass `columnStore.visibleColumns` (which are `ColumnDef` objects that satisfy `DataTableColumn`) directly to DataTable — TypeScript structural typing means `ColumnDef` is assignable to `DataTableColumn` since it has `name`, `width`, and `align`. The snippet receives the original `ColumnDef` object back, and ResourceList can cast it.

- [ ] **Step 2: Verify build and visual check**

Run: `cd frontend && pnpm check`

Then start the dev server (`task dev`) and verify:
- Resource list pages render identically — same columns, sorting, resizing, selection, context menu
- Column menu works
- Plugin columns and sparklines render if applicable
- Namespace click-to-filter works
- Delete action works

- [ ] **Step 3: Commit**

```bash
jj new && jj desc -m "Refactor ResourceList to consume DataTable"
```

---

### Task 3: Rework ClusterList to use DataTable

Replace the card-based cluster list with a DataTable, add column visibility and persistence, and wire row-click navigation.

**Files:**
- Modify: `frontend/src/routes/ClusterList.svelte`

- [ ] **Step 1: Rewrite ClusterList.svelte**

Replace the entire file. The new version uses DataTable with a local column state class.

```svelte
<script lang="ts">
  import {onMount} from "svelte";
  import {push} from "svelte-spa-router";
  import {Columns3, Unplug} from "lucide-svelte";
  import {clusterStore, type ConnectionStatusType} from "$lib/stores/cluster.svelte";
  import type {KubeContext} from "$lib/stores/cluster.svelte";
  import DataTable, {type DataTableColumn} from "$lib/components/DataTable.svelte";
  import ColumnMenu from "$lib/components/ColumnMenu.svelte";
  import ConnectionIndicator from "$lib/components/ConnectionIndicator.svelte";
  import KubeconfigImportDialog from "$lib/components/KubeconfigImportDialog.svelte";
  import {
    GetColumnPrefs,
    SetColumnPrefs,
    DeleteColumnPrefs,
  } from "../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js";
  import {GVRColumnPrefs, ColumnSettings, SortPrefs} from "../../../bindings/github.com/Vilsol/klados/internal/config/models.js";

  const PREFS_KEY = "_clusterList";

  const ALL_COLUMNS: DataTableColumn[] = [
    {name: "Name"},
    {name: "Cluster"},
    {name: "User"},
    {name: "Namespace", hidden: true},
    {name: "Version"},
    {name: "Provider"},
    {name: "Status"},
  ];

  // Local column state (same interface shape as columnStore but self-contained)
  let visibleColumns = $state<DataTableColumn[]>([]);
  let allColumns = $state<{col: DataTableColumn; visible: boolean}[]>([]);
  let sortState = $state<{column: string; direction: "asc" | "desc"} | null>(null);
  let columnMenuOpen = $state(false);
  let showImportDialog = $state(false);
  let saveTimer: ReturnType<typeof setTimeout> | null = null;

  function applyPrefs(prefs: GVRColumnPrefs | null) {
    const poolMap = new Map<string, DataTableColumn>(ALL_COLUMNS.map((c) => [c.name, {...c}]));
    if (prefs?.columns) {
      for (const [name, settings] of Object.entries(prefs.columns)) {
        if (settings?.width !== undefined && poolMap.has(name)) {
          const existing = poolMap.get(name)!;
          poolMap.set(name, {...existing, width: settings.width});
        }
      }
    }

    let visibleNames: string[];
    if (prefs?.order && prefs.order.length > 0) {
      visibleNames = prefs.order.filter((name) => poolMap.has(name));
    } else {
      visibleNames = ALL_COLUMNS.filter((c) => !c.hidden).map((c) => c.name);
    }

    const visibleSet = new Set(visibleNames);
    visibleColumns = visibleNames.map((name) => poolMap.get(name)!);
    allColumns = ALL_COLUMNS.map((c) => ({col: poolMap.get(c.name)!, visible: visibleSet.has(c.name)}));
    sortState = prefs?.sort ? {column: prefs.sort.column, direction: prefs.sort.direction as "asc" | "desc"} : null;
  }

  function buildPrefs(): GVRColumnPrefs {
    return new GVRColumnPrefs({
      order: visibleColumns.map((c) => c.name),
      columns: Object.fromEntries(
        allColumns.filter(({col}) => col.width !== undefined).map(({col}) => [col.name, new ColumnSettings({width: col.width})]),
      ),
      sort: sortState ? new SortPrefs({column: sortState.column, direction: sortState.direction}) : null,
    });
  }

  function savePrefs() {
    if (saveTimer) clearTimeout(saveTimer);
    saveTimer = null;
    SetColumnPrefs(PREFS_KEY, buildPrefs());
  }

  function debouncedSavePrefs() {
    if (saveTimer) clearTimeout(saveTimer);
    saveTimer = setTimeout(savePrefs, 300);
  }

  function setColumnVisible(name: string, visible: boolean) {
    if (name === "Name") return;
    const entry = allColumns.find((e) => e.col.name === name);
    if (!entry || entry.visible === visible) return;
    allColumns = allColumns.map((e) => (e.col.name === name ? {...e, visible} : e));
    if (visible) {
      visibleColumns = [...visibleColumns, entry.col];
    } else {
      visibleColumns = visibleColumns.filter((c) => c.name !== name);
    }
    savePrefs();
  }

  function moveColumn(name: string, direction: "up" | "down") {
    const idx = visibleColumns.findIndex((c) => c.name === name);
    if (idx === -1) return;
    if (direction === "up" && idx === 0) return;
    if (direction === "down" && idx === visibleColumns.length - 1) return;
    const next = [...visibleColumns];
    const swapIdx = direction === "up" ? idx - 1 : idx + 1;
    [next[idx], next[swapIdx]] = [next[swapIdx], next[idx]];
    visibleColumns = next;
    savePrefs();
  }

  function resetColumns() {
    if (saveTimer) clearTimeout(saveTimer);
    saveTimer = null;
    DeleteColumnPrefs(PREFS_KEY);
    applyPrefs(null);
  }

  function getCellValue(ctx: KubeContext, colName: string): string {
    const status = clusterStore.connectionStatus[ctx.name] ?? "disconnected";
    switch (colName) {
      case "Name": return ctx.name;
      case "Cluster": return ctx.cluster;
      case "User": return ctx.user;
      case "Namespace": return ctx.namespace;
      case "Version": return ctx.serverVersion;
      case "Provider": return ctx.provider;
      case "Status": return status;
      default: return "";
    }
  }

  function statusBadgeClass(status: string): string {
    switch (status) {
      case "connected": return "bg-accent/20 text-accent border-accent/30";
      case "connecting": return "bg-yellow-500/20 text-yellow-400 border-yellow-500/30";
      case "error": return "bg-destructive/20 text-destructive border-destructive/30";
      default: return "bg-muted/10 text-fg border-border";
    }
  }

  // Sort items client-side
  const sorted = $derived.by(() => {
    let result = [...clusterStore.contexts];
    if (sortState) {
      const {column, direction} = sortState;
      result.sort((a, b) => {
        const av = getCellValue(a, column);
        const bv = getCellValue(b, column);
        const cmp = av.localeCompare(bv);
        return direction === "asc" ? cmp : -cmp;
      });
    }
    return result;
  });

  async function handleRowClick(ctx: KubeContext) {
    const status = clusterStore.connectionStatus[ctx.name] ?? "disconnected";
    if (status !== "connected" && status !== "connecting") {
      clusterStore.connect(ctx.name);
    }
    push(`/c/${encodeURIComponent(ctx.name)}`);
  }

  onMount(() => {
    clusterStore.loadContexts();
    GetColumnPrefs(PREFS_KEY).then((prefs) => applyPrefs(prefs));
  });

  $effect(() => {
    if (!columnMenuOpen) return;
    const close = () => { columnMenuOpen = false };
    const timer = setTimeout(() => window.addEventListener("click", close, {once: true}), 0);
    return () => { clearTimeout(timer); window.removeEventListener("click", close) };
  });
</script>

<KubeconfigImportDialog bind:open={showImportDialog} onsuccess={() => clusterStore.loadContexts()} />

<DataTable
  items={sorted}
  {visibleColumns}
  {sortState}
  compact={false}
  loading={false}
  emptyMessage="No clusters found. Import a kubeconfig to get started."
  suffixGridCols={["36px"]}
  onsort={(col, dir) => { sortState = {column: col, direction: dir}; savePrefs() }}
  onresize={(col, width) => {
    visibleColumns = visibleColumns.map((c) => c.name === col ? {...c, width} : c);
    allColumns = allColumns.map((e) => e.col.name === col ? {...e, col: {...e.col, width}} : e);
    debouncedSavePrefs();
  }}
  onrowclick={handleRowClick}
>
  {#snippet toolbar()}
    <h1 class="text-sm font-semibold">Clusters</h1>
    <span class="text-xs text-muted">{clusterStore.contexts.length} clusters</span>
    <div class="flex-1"></div>
    <div class="relative">
      <button type="button" onclick={() => columnMenuOpen = !columnMenuOpen}
        class="p-1 rounded hover:bg-surface-hover transition-colors" title="Manage columns" aria-label="Manage columns">
        <Columns3 size={14} />
      </button>
      {#if columnMenuOpen}
        <ColumnMenu
          {visibleColumns} {allColumns} compact={false}
          onToggle={setColumnVisible}
          onMove={moveColumn}
          onReset={resetColumns}
          onCompactChange={() => {}}
        />
      {/if}
    </div>
    <button type="button" onclick={() => showImportDialog = true}
      class="px-3 py-1.5 text-sm rounded border border-border hover:bg-surface-hover transition-colors">
      Import Kubeconfig
    </button>
  {/snippet}

  {#snippet headerSuffix()}
    <div></div>
  {/snippet}

  {#snippet cell({ item: ctx, column })}
    {@const value = getCellValue(ctx, column.name)}
    {#if column.name === "Status"}
      <span class="px-1.5 py-0.5 text-xs rounded border {statusBadgeClass(value)}">{value}</span>
    {:else}
      <span>{value}</span>
    {/if}
  {/snippet}

  {#snippet rowSuffix({ item: ctx })}
    {@const status = clusterStore.connectionStatus[ctx.name] ?? "disconnected"}
    <div class="flex items-center justify-end">
      {#if status === "connected" || status === "connecting"}
        <button type="button"
          onclick={(e) => { e.stopPropagation(); clusterStore.disconnect(ctx.name) }}
          class="p-1 rounded opacity-0 group-hover:opacity-60 hover:!opacity-100 hover:text-destructive transition-all"
          title="Disconnect" aria-label="Disconnect {ctx.name}">
          <Unplug size={13} />
        </button>
      {/if}
    </div>
  {/snippet}
</DataTable>
```

Key details:
- Uses `_clusterList` as the config prefs key — reuses existing `GVRColumnPrefs` persistence with no Go changes
- `getCellValue` maps column names to KubeContext fields
- Status column uses badge styling matching ResourceList's badge classes
- Row click connects (if needed) and navigates
- Disconnect button appears on hover for connected clusters
- Column menu reuses the prop-driven `ColumnMenu` (no sparkline section since no `gvr` prop)
- The `Namespace` column is hidden by default (`hidden: true` in `ALL_COLUMNS`)

- [ ] **Step 2: Verify build and visual check**

Run: `cd frontend && pnpm check`

Then `task dev` and verify:
- Cluster list shows as a dense table
- Columns are sortable and resizable
- Column menu shows/hides columns
- Namespace column is hidden by default, can be shown via column menu
- Clicking a row connects and navigates to the cluster
- Disconnect button appears on hover for connected clusters
- Import kubeconfig dialog still works
- Resource list pages still work correctly (regression check)

- [ ] **Step 3: Commit**

```bash
jj new && jj desc -m "Rework ClusterList to use DataTable with column management"
```

---

## Verification Checklist

After all tasks are complete, verify these end-to-end:

- [ ] `cd frontend && pnpm check` passes
- [ ] Cluster list renders as a table with Name, Cluster, User, Version, Provider, Status columns
- [ ] Namespace column is hidden by default, toggleable via column menu
- [ ] Sorting works on all columns
- [ ] Column resizing works (drag + double-click auto-fit)
- [ ] Column show/hide persists across page navigations
- [ ] Clicking a cluster row connects and navigates to `/c/:ctx`
- [ ] Disconnect button appears on hover for connected clusters
- [ ] Import Kubeconfig dialog works
- [ ] Previously connected clusters auto-reconnect on app restart (existing behavior, verify not regressed)
- [ ] Resource list pages render identically (sorting, resizing, selection, context menu, plugin columns, sparklines)
- [ ] ColumnMenu works in both resource list and cluster list
