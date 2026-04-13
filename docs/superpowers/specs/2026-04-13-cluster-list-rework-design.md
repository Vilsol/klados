# Cluster List Rework Design

## Goal

Replace the card-based cluster list page with a dense table view using a new generic `DataTable` component extracted from `ResourceList`. Add auto-reconnect for previously connected clusters. Add column visibility controls.

## Components

### 1. DataTable.svelte (new)

Generic virtual scrolling table extracted from ResourceList. Location: `frontend/src/lib/components/DataTable.svelte`.

**Owns:**
- TanStack Virtual scrolling with configurable row height (compact/normal)
- CSS grid column layout with `gridTemplateCols` computed from column definitions
- Column headers with sort toggle (asc/desc/none) and resize handles (drag + double-click auto-fit)
- Row positioning (absolute + translateY), click handling, hover styling
- Loading / empty / error states

**Does not own:** CEL evaluation, render types, plugin columns, selection, search, export, keyboard shortcuts.

**Props:**

```ts
type DataTableColumn = {
  name: string;
  width?: number;
  align?: "left" | "right" | "center";
  hidden?: boolean;
};

items: T[];
columns: DataTableColumn[];          // all columns (visible + hidden)
visibleColumns: DataTableColumn[];   // ordered visible subset
sortState: { column: string; direction: "asc" | "desc" } | null;
compact: boolean;
loading: boolean;
error: string | null;
rowHeight?: number;                  // override, otherwise 28 (compact) / 36

// Extra grid columns injected before/after data columns
prefixGridCols?: string[];           // e.g. ["36px"] for checkbox column
suffixGridCols?: string[];           // e.g. ["36px"] for action column

// Snippets
toolbar: Snippet;                    // rendered above the table
cell: Snippet<[{ item: T; column: DataTableColumn }]>;
rowPrefix?: Snippet<[{ item: T }]>;
rowSuffix?: Snippet<[{ item: T }]>;

// Callbacks
onsort?: (column: string, direction: "asc" | "desc") => void;
onresize?: (column: string, width: number) => void;
onrowclick?: (item: T) => void;
oncontextmenu?: (event: MouseEvent, item: T) => void;

// Bindable
scrollContainer?: HTMLDivElement;
```

**Rendering contract:**
- DataTable renders the header row (column names, sort icons, resize handles) and the virtual row container.
- For each visible row, DataTable renders the grid wrapper and calls `rowPrefix`, then `cell` for each visible column, then `rowSuffix`.
- The consumer controls all cell content via snippets.

### 2. ResourceList.svelte (refactored)

Becomes a DataTable consumer. All Kubernetes-specific logic stays here:

- CEL expression evaluation via `evalExpr` in the `cell` snippet
- Render types: text, badge, age, controlledBy, progress
- Plugin columns and context menu items (rendered after data columns in the `cell` snippet via extra iteration)
- Sparkline columns (same approach)
- SmartSearch + SavedFilterDropdown in the toolbar snippet
- Export menu (YAML/JSON) in the toolbar snippet
- ColumnMenu in the toolbar snippet
- Refresh button in the toolbar snippet
- Selection store (checkboxes in `rowPrefix`, bulk actions)
- Keyboard shortcuts (delete, select-all, copy names, refresh)
- Namespace click-to-filter
- Confirm delete dialog
- Context menu (right-click)
- `selectedName` highlighting (accent border)

ResourceList continues to use `columnStore` for column state, passing `columnStore.visibleColumns` and `columnStore.sortState` as DataTable props.

Plugin columns and sparkline columns are rendered inside the `rowSuffix` snippet (after the data columns, before the action button). Their grid track sizes are included in `suffixGridCols` so DataTable allocates space for them in the CSS grid. This keeps DataTable unaware of plugin/sparkline concepts — it just renders the extra grid tracks and lets the suffix snippet fill them.

### 3. ColumnMenu.svelte (refactored)

Currently reads from the global `columnStore` singleton. Changed to accept props:

```ts
visibleColumns: { name: string }[];
allColumns: { col: { name: string }; visible: boolean }[];
compact: boolean;
onToggle: (name: string, visible: boolean) => void;
onMove: (name: string, direction: "up" | "down") => void;
onReset: () => void;
onCompactChange: (value: boolean) => void;

// Optional — only used by ResourceList
sparklineGvrs?: string[];
sparklineColumns?: string[];
onSparklineToggle?: (columns: string[]) => void;
gvr?: string;
```

ResourceList passes `columnStore` methods. ClusterList passes its own local state methods. The sparkline section only renders when `gvr` is provided.

### 4. ClusterList.svelte (reworked)

Replaces card grid with a DataTable. Location: same file `frontend/src/routes/ClusterList.svelte`.

**Columns:**

| Column | Default visible | Source |
|---|---|---|
| Name | yes | `ctx.name` |
| Cluster | yes | `ctx.cluster` |
| User | yes | `ctx.user` |
| Namespace | no (hidden) | `ctx.namespace` |
| Version | yes | `ctx.serverVersion` |
| Provider | yes | `ctx.provider` |
| Status | yes | `connectionStatus[ctx.name]` — badge render |

**Cell rendering:**
- Status column: rendered as a colored badge (green=connected, yellow=connecting, red=error, gray=disconnected) matching the badge styling in ResourceList.
- All other columns: plain text.

**Row click:** Connects if not already connected, then navigates to `/c/:ctx`.

**Row suffix:** Disconnect button, visible on hover, only for connected/connecting clusters. Styled like ResourceList's delete-on-hover button.

**Toolbar:** "Import Kubeconfig" button + item count + ColumnMenu. No search, no export.

**Column state:** Local class with same shape as columnStore (`visibleColumns`, `allColumns`, `sortState`, `setColumnVisible()`, `moveColumn()`, `setSort()`, `resizeColumn()`). Persisted to config via a new `clusterListColumns` key in `GVRColumnPrefs` (reusing the existing column prefs structure with a synthetic key like `_clusterList`).

**Sort:** Client-side, comparing string values of the active sort column.

### 5. Auto-reconnect

**Session state change:** Add `lastConnectedContexts: string[]` to the session store (`session.json`).

**On connect:** Add context name to `lastConnectedContexts`, save session.

**On disconnect:** Remove context name from `lastConnectedContexts`, save session.

**On app launch:** After `loadContexts()` completes, read `lastConnectedContexts` from session. For each name that exists in the loaded contexts, call `connect()`. Failures are silently ignored (the status will show as disconnected/error).

## Files changed

| File | Change |
|---|---|
| `frontend/src/lib/components/DataTable.svelte` | **New** — generic virtual table |
| `frontend/src/lib/components/ResourceList.svelte` | Refactor to use DataTable |
| `frontend/src/lib/components/ColumnMenu.svelte` | Prop-driven instead of global store |
| `frontend/src/routes/ClusterList.svelte` | Cards → DataTable with column visibility |
| `frontend/src/lib/stores/cluster.svelte.ts` | Auto-reconnect logic |
| `frontend/src/lib/stores/session.svelte.ts` | Add `lastConnectedContexts` field |

## Not in scope

- No Go backend changes (all RPCs exist)
- No changes to `columnStore` internals
- No changes to descriptor registry
- No search/export for cluster list (can be added later)

## Risk

The ResourceList refactor is the primary risk — it's the most-used component. Mitigated by keeping all K8s logic in ResourceList and only extracting pure table rendering. The rendered output should be pixel-identical after the refactor.
