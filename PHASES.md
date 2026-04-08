# Resource List Column Improvements — Implementation Phases

## Project Overview

Add user-configurable column management to the resource list view: visibility toggles, reordering, resizing, alignment, compact mode, sticky first column, and sort persistence. Then expand builtin GVR descriptors with additional columns backed by new enrichers. The spec lives in `RESOURCE_LIST_COLUMNS.md`.

## Phase Map

```
Phase 1 — Go Backend (types, RPCs, Namespace columns)
  ├── Phase 2 — Frontend Column Store (depends on 1)
  │     ├── Phase 3 — ResourceList UI Overhaul (depends on 2)
  │     └── Phase 4 — Column Menu Component (depends on 2, parallel with 3)
  └── Phase 5 — Enrichers & Expanded Columns (depends on 1, parallel with 2-4)
```

---

## Phase 1 — Go Backend

> Establishes the data model, config storage, and RPC surface that all frontend work depends on.

| | |
|---|---|
| **Depends on** | none |
| **Parallel with** | nothing |

### Deliverables

- `AlignType` constant and `Align`, `Hidden` fields on `resource.Column` struct in `descriptor.go`
- `ColumnSettings`, `GVRColumnPrefs`, `SortPrefs` types in `config/config.go`
- `ColumnPrefs map[string]*GVRColumnPrefs` and `CompactRows bool` fields on `Config` struct
- `GetColumnPrefs`, `SetColumnPrefs`, `GetCompactRows`, `SetCompactRows` RPC methods on `ConfigService`
- Namespace column (`Hidden: true`) added to every non-cluster-scoped builtin descriptor in `builtin.go`
- Regenerated Wails TypeScript bindings
- Updated `ColumnDef` interface in `frontend/src/lib/registry/index.ts` with `align?` and `hidden?` fields, plus `defaultAlign()` helper

### Tests

- **Go unit test (config)**
  - `TestColumnPrefsRoundTrip` — save GVRColumnPrefs, reload, verify equality
  - `TestMissingColumnPrefsDefaultsGracefully` — load config with no `columnPrefs` key, verify nil/empty map
  - `TestCompactRowsDefault` — verify `CompactRows` defaults to false

- **Go unit test (descriptor)**
  - `TestColumnAlignDefault` — verify `Align` field is empty string when omitted (frontend applies default)
  - `TestNamespaceColumnOnAllNamespacedDescriptors` — iterate builtins, assert every non-`ClusterScoped` descriptor has a Namespace column with `Hidden: true`
  - `TestClusterScopedDescriptorsHaveNoNamespaceColumn` — iterate builtins, assert cluster-scoped descriptors do NOT have a Namespace column

- **Frontend type check**
  - `pnpm check` passes with updated `ColumnDef` interface

### Out of Scope

- Frontend column store and UI — Phase 2-4
- New enrichers and expanded columns — Phase 5
- Column filtering infrastructure — explicitly deferred per spec

### Acceptance Criteria

- [ ] `Column` struct has `Align AlignType` and `Hidden bool` fields with correct JSON tags
- [ ] `Config` struct has `ColumnPrefs` and `CompactRows` fields
- [ ] `ConfigService` has four new RPC methods that read/write column prefs and compact mode
- [ ] Every non-cluster-scoped builtin descriptor has a `Namespace` column with `Hidden: true`
- [ ] No cluster-scoped descriptor has a `Namespace` column
- [ ] `wails3 generate bindings` succeeds
- [ ] `go test ./internal/config/ ./internal/resource/ -v` passes
- [ ] `cd frontend && pnpm check` passes

### Source Documents

- `RESOURCE_LIST_COLUMNS.md` — full spec (storage shape, Go backend changes, Column struct additions)
- `internal/resource/descriptor.go` — `Column` struct to modify
- `internal/resource/builtin.go` — builtin descriptors to add Namespace columns to
- `internal/config/config.go` — `Config` struct to extend
- `internal/services/config.go` — `ConfigService` to add RPCs to
- `frontend/src/lib/registry/index.ts` — `ColumnDef` interface to extend

### Handoff Notes

- `Align` is optional on `Column` — the frontend must apply `defaultAlign(renderType)` when empty. This avoids having to set alignment on every existing column definition.
- `Hidden` columns are present in the descriptor but excluded from default visibility. The frontend column store (Phase 2) must handle the hidden→visible promotion when user enables a column.
- The Wails bindings for the new `ConfigService` methods are the contract Phase 2 depends on. Verify the generated TypeScript signatures match expectations before starting Phase 2.
- `GVRColumnPrefs.Order` only contains visible column names. If the array is empty/nil, fall back to descriptor column order (excluding hidden columns).

---

## Phase 2 — Frontend Column Store

> Reactive Svelte 5 store that merges descriptor defaults with user prefs, exposes visibility/order/width/sort state, and debounce-saves to the Go backend.

| | |
|---|---|
| **Depends on** | Phase 1 |
| **Parallel with** | Phase 5 |

### Deliverables

- New file `frontend/src/lib/stores/columns.svelte.ts` — reactive column store class
- Exposes: `visibleColumns`, `allColumns`, `sortState`, `setColumnVisible()`, `moveColumn()`, `resizeColumn()`, `autoFitColumn()`, `setSort()`, `reset()`, `save()`
- Loads prefs via `ConfigService.GetColumnPrefs(gvr)` when GVR changes
- Merges prefs with descriptor defaults (descriptor = full pool, prefs override visibility/order/widths)
- Debounced save (300-500ms) to `ConfigService.SetColumnPrefs(gvr, prefs)` on any mutation
- Compact mode state exposed (reads/writes `ConfigService.GetCompactRows` / `SetCompactRows`)

### Tests

- **Frontend test (vitest)**
  - `columns store loads descriptor defaults when no prefs exist` — mock `GetColumnPrefs` returning null, verify `visibleColumns` matches descriptor non-hidden columns in order
  - `columns store merges saved prefs with descriptor` — mock prefs with reordered subset, verify `visibleColumns` reflects saved order and widths
  - `setColumnVisible hides a column` — call `setColumnVisible('Status', false)`, verify it's removed from `visibleColumns` and present in `allColumns` as `visible: false`
  - `moveColumn reorders correctly` — move a middle column up, verify new order
  - `resizeColumn updates width` — set width on a column, verify `visibleColumns` entry has new width
  - `reset clears prefs` — set some prefs, call `reset()`, verify columns revert to descriptor defaults
  - `Name column cannot be hidden` — attempt `setColumnVisible('Name', false)`, verify it remains visible

### Out of Scope

- `autoFitColumn` implementation details (DOM measurement) — the store exposes the method signature, but the actual DOM measurement happens in Phase 3's resize handle double-click handler.
- Sort comparison logic — the store only tracks sort column/direction. The actual sort comparator stays in `ResourceList.svelte`.

### Acceptance Criteria

- [ ] `columns.svelte.ts` exports a store class with the specified API
- [ ] Store correctly merges descriptor columns with saved prefs
- [ ] Hidden columns appear in `allColumns` but not in `visibleColumns` by default
- [ ] Mutations trigger debounced save to `ConfigService.SetColumnPrefs`
- [ ] `reset()` clears the GVR's prefs and reverts to descriptor defaults
- [ ] Compact mode getter/setter works via `ConfigService`
- [ ] All vitest tests pass
- [ ] `pnpm check` passes

### Source Documents

- `RESOURCE_LIST_COLUMNS.md` — store API spec (§ Frontend changes, `columns.svelte.ts`)
- `frontend/src/lib/registry/index.ts` — `DescriptorDef` / `ColumnDef` types the store consumes
- `frontend/src/lib/stores/cluster.svelte.ts` — reference for Svelte 5 class-based store pattern
- `frontend/bindings/github.com/Vilsol/klados/internal/services/configservice.js` — generated bindings to import

### Handoff Notes

- The store is a class singleton (following `clusterStore` pattern), not a function factory. One instance, re-initialized when GVR changes.
- `autoFitColumn(name)` in the store should accept an explicit width value (computed by the caller from DOM measurement) rather than trying to measure the DOM itself — the store has no DOM access.
- Plugin-injected columns are NOT managed by this store. They are handled separately by `slotRegistry` and rendered after user-configured columns. The store's `allColumns` only includes descriptor-defined columns.

---

## Phase 3 — ResourceList UI Overhaul

> Wires the column store into the existing ResourceList component: sticky first column, resize handles, alignment, tooltips, compact mode, namespace click, and sort persistence.

| | |
|---|---|
| **Depends on** | Phase 2 |
| **Parallel with** | Phase 4 |

### Deliverables

- `ResourceList.svelte` refactored to consume `columns` from the column store instead of props
- Sticky first column: `position: sticky; left: 0; z-index: 10` with `bg-bg` background and right drop shadow on first header cell and first body cell
- Column resize handles between header cells: thin `<div>` with `cursor-col-resize`, mousedown/mousemove/mouseup tracking, double-click for auto-fit
- Minimum column width enforced at 20px
- Grid template computed from `visibleColumns` with width overrides; columns without explicit width use `minmax(20px, 1fr)`
- Cell alignment via `text-left` / `text-right` / `text-center` classes based on `col.align ?? defaultAlign(col.renderType)`
- Cell content wrapped with `title={renderValue(value, col.renderType)}` for truncation tooltip
- Compact mode: `ROW_HEIGHT` derived from column store's compact state (36px normal, 28px compact)
- Namespace column cells: `onclick` handler calls `clusterStore.setSelectedNamespaces([namespace])` (only on Namespace column)
- Sort column/direction initialized from stored prefs; changes saved via column store's `setSort()`
- `ResourceListPage.svelte` updated to instantiate/provide the column store

### Tests

- **Frontend test (vitest)**
  - `sticky first column has correct classes` — render ResourceList, verify first header cell has `sticky` and `left-0` classes
  - `cell alignment matches render type` — render a row, verify age columns have `text-right`, text columns have `text-left`
  - `cell has title attribute` — render a row, verify each cell's content span has a `title` attribute

- **Manual verification**
  - Column resize: drag header divider, column width changes live
  - Double-click divider: column auto-fits to widest visible content
  - Sticky column: scroll right, first column stays pinned with drop shadow
  - Compact mode: toggle compact, row height visibly decreases
  - Namespace click: click a namespace value, global namespace filter updates

### Out of Scope

- Column menu dropdown (visibility checkboxes, reorder buttons) — Phase 4
- Horizontal scroll container changes that might be needed if sticky breaks with the virtualizer — handle as a gotcha during implementation, not a separate phase

### Acceptance Criteria

- [ ] First column (Name) is sticky with solid background and right drop shadow
- [ ] Resize handles appear between header cells; dragging resizes columns
- [ ] Double-click resize handle auto-fits column to visible content width
- [ ] Column widths cannot go below 20px
- [ ] Cell text alignment follows render type defaults (age=right, others=left)
- [ ] Every cell has a `title` attribute for truncation tooltip
- [ ] Compact mode reduces row height from 36px to 28px
- [ ] Clicking a namespace cell in the Namespace column sets the global namespace filter
- [ ] Sort column/direction persists across GVR switches (loaded from prefs)
- [ ] `pnpm check` passes
- [ ] Existing frontend tests pass

### Source Documents

- `RESOURCE_LIST_COLUMNS.md` — §ResourceList.svelte, sticky column, resize handles, alignment, compact mode, namespace click
- `frontend/src/lib/components/ResourceList.svelte` — the component to modify
- `frontend/src/routes/ResourceListPage.svelte` — parent component that provides props
- `frontend/src/lib/stores/cluster.svelte.ts` — `clusterStore.setSelectedNamespaces()` for namespace click

### Handoff Notes

- **Sticky + CSS grid gotcha**: `position: sticky` inside CSS grid works in modern browsers but requires explicit `background-color` on the sticky cell (use `bg-bg` theme token). Test with horizontal scroll to verify content doesn't show through.
- **Virtual rows + sticky**: The virtualizer uses `transform: translateY()` for row positioning. Horizontal scroll is on the parent container. Verify that the sticky column works within this setup — if not, the inner grid (not the row div) may need to handle horizontal overflow.
- The `scrollContainer` bind and virtualizer setup in ResourceList should remain largely unchanged. The grid template just becomes dynamic based on the column store.
- `ResourceListPage` currently passes `columns` as a prop from the descriptor. After this phase, it should instead ensure the column store is initialized for the current GVR, and ResourceList reads from the store directly.

---

## Phase 4 — Column Menu Component

> Dropdown component replacing the sparkline-only toggle, providing full column management: visibility, reorder, compact mode, sparklines, and reset.

| | |
|---|---|
| **Depends on** | Phase 2 |
| **Parallel with** | Phase 3 |

### Deliverables

- New file `frontend/src/lib/components/ColumnMenu.svelte` — dropdown component
- Visibility checkboxes for each column (Name always checked and disabled)
- Up/down reorder buttons per column (disabled at boundaries)
- Checked (visible) columns at top in current order, unchecked at bottom
- Compact mode toggle calling `ConfigService.SetCompactRows()`
- Sparkline toggles section (only when `sparklineGvrs.includes(gvr)`)
- Reset button clearing the GVR's `columnPrefs` entry
- Replaces the existing `Columns3` button/dropdown in `ResourceList.svelte`

### Tests

- **Frontend test (vitest)**
  - `ColumnMenu renders all columns` — provide a descriptor with 5 columns (1 hidden), verify all 5 appear in the menu
  - `Name column checkbox is disabled` — verify the Name checkbox is checked and has `disabled` attribute
  - `toggling visibility calls setColumnVisible` — click a column checkbox, verify the store method was called
  - `up button disabled for first column` — verify the first column's up arrow is disabled
  - `reset button calls store.reset()` — click reset, verify `reset()` was called

### Out of Scope

- Drag-and-drop reorder — rejected per spec (up/down buttons sufficient)
- Per-column filter UI — architecturally planned but not implemented per spec

### Acceptance Criteria

- [ ] ColumnMenu dropdown opens from the `Columns3` button in the resource list header
- [ ] All descriptor columns (including hidden) appear in the menu
- [ ] Name column checkbox is always checked and disabled
- [ ] Checking/unchecking a column toggles its visibility in the resource list
- [ ] Up/down buttons reorder visible columns; disabled at boundaries
- [ ] Compact mode checkbox toggles global compact mode
- [ ] Sparkline toggles appear only for supported GVRs
- [ ] Reset button reverts columns to descriptor defaults
- [ ] `pnpm check` passes
- [ ] Vitest tests pass

### Source Documents

- `RESOURCE_LIST_COLUMNS.md` — §ColumnMenu.svelte wireframe and behavior spec
- `frontend/src/lib/components/ResourceList.svelte` — existing `Columns3` button to replace
- `frontend/src/lib/stores/columns.svelte.ts` — store API consumed by the menu

### Handoff Notes

- The existing sparkline toggle dropdown in ResourceList should be fully replaced by this component, not layered on top. Remove the old `columnMenuOpen` / `availableSparklineCols` / `toggleSparklineCol` logic from ResourceList.
- The menu should use `onclick={(e) => e.stopPropagation()}` to prevent closing when interacting with checkboxes/buttons inside it (same pattern as the current sparkline dropdown).

---

## Phase 5 — Enrichers & Expanded Columns

> Adds new enrichers and hidden columns to builtin descriptors for richer resource information when users opt in via the column menu.

| | |
|---|---|
| **Depends on** | Phase 1 |
| **Parallel with** | Phase 2, Phase 3, Phase 4 |

### Deliverables

- **Extended enrichers**: `DaemonSetEnricher` (+`nodeSelectorDisplay`), `JobEnricher` (+`statusDisplay`), `NodeEnricher` (+`internalIPDisplay`, `osArchDisplay`)
- **New enrichers**: `ReplicaSetEnricher`, `CronJobEnricher`, `ServiceEnricher`, `IngressEnricher`, `ConfigMapEnricher`, `SecretEnricher`, `PVEnricher`, `PVCEnricher`, `ServiceAccountEnricher`, `RoleEnricher`, `BindingEnricher`
- All new enrichers registered in `RegisterBuiltin()` in `builtin.go`
- New hidden columns added to every affected builtin descriptor per the spec tables
- All new columns have `Hidden: true`

### Tests

- **Go unit test (per enricher)**
  - `TestDaemonSetEnricher_NodeSelectorDisplay` — input with `spec.nodeSelector: {disktype: ssd, zone: us-east}`, verify `status.nodeSelectorDisplay` = `"disktype=ssd,zone=us-east"`
  - `TestReplicaSetEnricher_OwnerDisplay` — input with one ownerReference, verify `status.ownerDisplay` = owner name
  - `TestReplicaSetEnricher_NoOwner` — input with no ownerReferences, verify `status.ownerDisplay` = `"<none>"`
  - `TestJobEnricher_StatusDisplay` — input with Complete condition, verify `status.statusDisplay` = `"Complete"`
  - `TestCronJobEnricher_ActiveCount` — input with 2 active items, verify `status.activeCount` = `2`
  - `TestServiceEnricher_PortsDisplay` — input with two ports, verify `status.portsDisplay` = `"80/TCP, 443/TCP"`
  - `TestServiceEnricher_ExternalIPDisplay` — input with loadBalancer ingress, verify `status.externalIPDisplay`
  - `TestIngressEnricher_HostsDisplay` — input with two rules, verify `status.hostsDisplay` = `"foo.com, bar.com"`
  - `TestIngressEnricher_DefaultBackendDisplay` — input with defaultBackend, verify format
  - `TestConfigMapEnricher_DataKeysCount` — input with 3 data keys, verify `status.dataKeysCount` = `3`
  - `TestSecretEnricher_DataKeysCount` — input with 2 data keys, verify `status.dataKeysCount` = `2`
  - `TestPVEnricher_AccessModesDisplay` — input with `[ReadWriteOnce, ReadOnlyMany]`, verify `status.accessModesDisplay` = `"RWO,ROX"`
  - `TestPVEnricher_ClaimDisplay` — input with claimRef, verify `status.claimDisplay` = `"ns/name"`
  - `TestPVCEnricher_AccessModesDisplay` — similar to PV
  - `TestNodeEnricher_InternalIPDisplay` — input with addresses array, verify correct IP extracted
  - `TestNodeEnricher_OsArchDisplay` — verify `status.osArchDisplay` = `"linux/amd64"`
  - `TestServiceAccountEnricher_SecretsCount` — input with 2 secrets, verify `status.secretsCount` = `2`
  - `TestRoleEnricher_RulesCount` — input with 3 rules, verify `status.rulesCount` = `3`
  - `TestBindingEnricher_RoleRefDisplay` — verify format `"ClusterRole/admin"`
  - `TestBindingEnricher_SubjectsCount` — input with 2 subjects, verify `status.subjectsCount` = `2`

- **Go integration test**
  - `go test ./internal/resource/... -v` passes (all enrichers + descriptors)

### Out of Scope

- Frontend UI for enabling these columns — that's Phase 4 (ColumnMenu)
- Enrichers for CRDs or plugin-injected resources — out of scope entirely

### Acceptance Criteria

- [ ] All 11 new/extended enrichers implemented
- [ ] Each enricher has at least one unit test covering the happy path
- [ ] All new columns added to builtin descriptors with `Hidden: true`
- [ ] Enrichers registered in `RegisterBuiltin()`
- [ ] `go test ./internal/resource/... -v` passes
- [ ] `go test ./internal/resource/enrichers/ -v` passes
- [ ] No existing enricher tests broken

### Source Documents

- `RESOURCE_LIST_COLUMNS.md` — §Phase 2, enricher summary table, per-GVR column tables
- `internal/resource/builtin.go` — descriptors to add columns to, `RegisterBuiltin()` to register enrichers
- `internal/resource/enrichers/` — existing enrichers as reference patterns (e.g. `pod.go`, `job.go`, `node.go`)
- `internal/resource/enricher.go` — `Enricher` interface

### Handoff Notes

- Follow the existing enricher pattern: `Enrich(obj *unstructured.Unstructured) error`, set fields via `unstructured.SetNestedField()`. See `pod.go` or `job.go` for examples.
- Access mode abbreviation map: `ReadWriteOnce→RWO`, `ReadOnlyMany→ROX`, `ReadWriteMany→RWX`, `ReadWriteOncePod→RWOP`.
- The `ServiceEnricher` ports format should match kubectl: `port/protocol` (e.g. `80/TCP, 443/TCP`). Include `nodePort` if non-zero: `80:30080/TCP`.
- For `ConfigMapEnricher` and `SecretEnricher`, use `len(data)` not `len(data) + len(binaryData)` — keep it simple, matching kubectl behavior.
- Some enrichers compute fields under `status.*` even though the source data is in `spec.*`. This is an established pattern in this codebase (enrichers are allowed to put display fields anywhere). The CEL expressions in the column definitions reference the enriched paths.
