# Resource List Column Improvements

## Context

The resource list view (`ResourceList.svelte`) displays Kubernetes resources in a virtualized grid. Columns are defined by `Descriptor` structs in Go (`builtin.go`) and rendered via CEL expressions. Currently columns are fixed-width or flex, cannot be resized, hidden, or reordered by the user. Many builtin GVRs are missing useful columns that `kubectl` shows by default. This spec covers two phases: column interaction infrastructure, then expanded builtin columns.

## Decisions

**Global storage, not per-cluster**
Column preferences are stored once globally in `config.json`, keyed by GVR. Configuring per-cluster would be tedious for minimal benefit.

**Namespace column on every namespaced descriptor**
Added directly in `builtin.go` to every non-cluster-scoped descriptor. Hidden by default (not in the default `order`). Frontend shows it when the user enables it or when viewing multiple namespaces. Clicking a namespace value in the column sets the global namespace filter to that single namespace.

**Sticky first column with drop shadow**
The first column (always Name) uses `position: sticky; left: 0` with a solid background and a right-side drop shadow for elevation effect. Works within the existing CSS grid layout.

**Alignment by render type**
Not user-configurable. Defaults: `text` left, `badge` left, `age` right, `progress` left. Applied via a CSS class derived from `renderType`.

**Column menu replaces sparkline toggle**
The existing `Columns3` button expands into a full column management dropdown: visibility checkboxes, up/down reorder buttons, compact toggle, sparkline toggles (when applicable), and a reset button.

**Compact mode is global**
Single toggle in `config.json`, not per-GVR. Reduces row height from 36px to 28px.

**Column filtering architected but not implemented**
The storage shape and column header layout leave room for per-column filters in a future iteration. No filter UI is built in either phase.

## Rejected Alternatives

**Per-cluster column storage**
Rejected because the same user typically wants the same column layout across clusters. Per-cluster would require configuring each cluster separately.

**Drag-and-drop for column reorder**
Would require a DnD library dependency. Up/down buttons in the dropdown are sufficient and keep deps light.

**Per-column alignment configuration**
Alignment by render type covers the practical cases. User-configurable alignment adds complexity for negligible benefit.

## Priorities & Tradeoffs

- **Simplicity over flexibility**: render-type-based alignment, global compact mode, no per-column filter UI yet.
- **Correctness over performance for auto-fit**: auto-fit scans only visible (virtualized) rows, not all rows. Good enough for practical use without measuring thousands of DOM nodes.
- **Infrastructure first**: Phase 1 delivers the interaction framework. Phase 2 adds content (new columns/enrichers) on top.

## Potential Gotchas

- **Sticky + CSS grid**: `position: sticky` inside grid works in modern browsers but requires explicit `background-color` on the sticky cell — otherwise content scrolls visibly behind it. Must use theme token (`bg-bg` or `bg-surface`) not transparent.
- **Virtual rows + sticky**: Rows are absolutely positioned (`transform: translateY`). Horizontal scroll is on the parent container. Sticky applies within the scroll container — verify that the virtualizer's absolute positioning doesn't break sticky behavior. May need the inner grid (not the row div) to handle horizontal overflow.
- **Auto-fit measures visible rows only**: If the widest value is off-screen, auto-fit won't account for it. Acceptable tradeoff.
- **CEL expressions for new columns**: Some new columns (ports summary, ingress hosts) need enrichers to flatten arrays into strings. CEL alone can't do `ports.map(p => p.port + "/" + p.protocol).join(", ")` — the enricher must pre-compute the display string.
- **Config migration**: Existing `config.json` files have no `columnPrefs` or `compact` field. Code must handle missing fields gracefully (default to descriptor defaults).
- **Plugin columns interact with reorder**: Plugin-injected columns should appear after user-configured columns and not be reorderable via this UI (they have their own registration order).
- **Namespace column click**: Must integrate with `clusterStore.setSelectedNamespaces()`. If the user is already filtered to that namespace, the click should be a no-op or deselect (toggle behavior). Decide at implementation time.

## Implementation Details

### Phase 1 — Column Infrastructure

#### Storage shape (`config.json`)

```go
type ColumnSettings struct {
    Width int `json:"width,omitempty"`
}

type GVRColumnPrefs struct {
    Columns map[string]ColumnSettings `json:"columns"`
    Order   []string                  `json:"order"`
    Sort    *SortPrefs               `json:"sort,omitempty"`
}

type SortPrefs struct {
    Column    string `json:"column"`
    Direction string `json:"direction"` // "asc" | "desc"
}

type Config struct {
    // ... existing fields ...
    ColumnPrefs map[string]*GVRColumnPrefs `json:"columnPrefs,omitempty"`
    CompactRows bool                       `json:"compactRows,omitempty"`
}
```

Example JSON:
```json
{
  "columnPrefs": {
    "core.v1.pods": {
      "columns": {
        "Name": { "width": 200 },
        "Namespace": {},
        "Ready": { "width": 80 },
        "Status": {},
        "Age": {}
      },
      "order": ["Name", "Namespace", "Ready", "Status", "Age"],
      "sort": { "column": "Name", "direction": "asc" }
    }
  },
  "compactRows": false
}
```

Semantics:
- Presence in `columns` object = visible. Absence = hidden.
- `width` omitted = use descriptor default or auto-size.
- `order` array contains only visible column names, determines display order.
- If no entry for a GVR, all descriptor columns are visible in descriptor order.
- `compactRows` is global, not per-GVR.

#### Go backend changes

**`internal/config/config.go`** — Add `ColumnPrefs` and `CompactRows` fields to `Config` struct. Add `ColumnSettings`, `GVRColumnPrefs`, `SortPrefs` types.

**`internal/services/config.go`** — Add RPC methods:
```go
func (s *ConfigService) GetColumnPrefs(gvr string) *config.GVRColumnPrefs
func (s *ConfigService) SetColumnPrefs(gvr string, prefs *config.GVRColumnPrefs) error
func (s *ConfigService) GetCompactRows() bool
func (s *ConfigService) SetCompactRows(compact bool) error
```

**`internal/resource/builtin.go`** — Add `Namespace` column to every non-cluster-scoped descriptor:
```go
{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150}
```

Add `Align` field to `Column` struct (see below).

**`internal/resource/descriptor.go`** — Add alignment:
```go
type AlignType string

const (
    AlignLeft   AlignType = "left"
    AlignRight  AlignType = "right"
    AlignCenter AlignType = "center"
)

type Column struct {
    Name       string     `json:"name"`
    Expr       string     `json:"expr"`
    RenderType RenderType `json:"renderType"`
    Width      int        `json:"width,omitempty"`
    Align      AlignType  `json:"align,omitempty"`
    Hidden     bool       `json:"hidden,omitempty"`
}
```

- `Align` defaults based on `RenderType` if empty: `age` → `right`, everything else → `left`.
- `Hidden` marks columns that exist in the descriptor but are not shown by default (e.g., Namespace). The frontend uses this to populate the "available but unchecked" section of the column menu.

#### Frontend changes

**`frontend/src/lib/registry/index.ts`** — Update `ColumnDef`:
```typescript
export type AlignType = 'left' | 'right' | 'center'

export interface ColumnDef {
  name: string
  expr: string
  renderType: RenderType
  width?: number
  align?: AlignType
  hidden?: boolean
}
```

Add helper:
```typescript
export function defaultAlign(renderType: RenderType): AlignType {
  if (renderType === 'age') return 'right'
  return 'left'
}
```

**`frontend/src/lib/stores/columns.svelte.ts`** (new file) — Reactive store that:
- Loads prefs from `ConfigService.GetColumnPrefs(gvr)` when GVR changes.
- Merges with descriptor defaults: descriptor defines the full pool, prefs override visibility/order/widths.
- Exposes:
  - `visibleColumns: ColumnDef[]` — ordered, filtered to visible, with width overrides applied.
  - `allColumns: { col: ColumnDef; visible: boolean }[]` — for the column menu.
  - `sortState: { column: string; direction: 'asc' | 'desc' } | null`
  - `setColumnVisible(name: string, visible: boolean)`
  - `moveColumn(name: string, direction: 'up' | 'down')`
  - `resizeColumn(name: string, width: number)`
  - `autoFitColumn(name: string)` — measures visible rows, sets width.
  - `setSort(column: string, direction: 'asc' | 'desc')`
  - `reset()` — clears prefs for current GVR, reverts to descriptor defaults.
  - `save()` — debounced write to `ConfigService.SetColumnPrefs(gvr, prefs)`.

**`frontend/src/lib/components/ResourceList.svelte`** — Major changes:

1. **Sticky first column**: First cell in header and each row gets:
   ```
   sticky left-0 z-10 bg-bg shadow-[2px_0_4px_rgba(0,0,0,0.08)]
   dark:shadow-[2px_0_4px_rgba(0,0,0,0.3)]
   ```

2. **Column resize handles**: Between header cells, a thin `<div>` with `cursor-col-resize`. Mousedown starts resize tracking, mousemove updates column width live, mouseup saves. Double-click triggers auto-fit. Minimum width: 20px globally.

3. **Grid template**: Computed from `visibleColumns` with width overrides. Columns without explicit width get `minmax(20px, 1fr)`.

4. **Cell alignment**: Each cell gets `text-left`, `text-right`, or `text-center` based on `col.align ?? defaultAlign(col.renderType)`.

5. **Cell tooltips**: Every cell content wrapped with `title={renderValue(value, col.renderType)}` for truncation tooltip.

6. **Compact mode**: Row height derived from global compact setting: `const ROW_HEIGHT = compact ? 28 : 36`. Passed to virtualizer's `estimateSize`.

7. **Namespace click**: Namespace column cells get an `onclick` handler that calls `clusterStore.setSelectedNamespaces([namespace])`.

8. **Sort persistence**: `sortCol`/`sortDir` initialized from stored prefs. Changes saved via debounced `ConfigService.SetColumnPrefs`.

**`frontend/src/lib/components/ColumnMenu.svelte`** (new file) — Dropdown component:

```
┌─────────────────────────────┐
│ Columns               Reset │
│ ┌─────────────────────────┐ │
│ │ ☑ Name           ▲ ▼   │ │
│ │ ☑ Namespace      ▲ ▼   │ │
│ │ ☑ Ready          ▲ ▼   │ │
│ │ ☑ Status         ▲ ▼   │ │
│ │ ☐ Node           ▲ ▼   │ │
│ │ ☐ IP             ▲ ▼   │ │
│ └─────────────────────────┘ │
│ ☐ Compact rows              │
│ ── Sparklines ────────────  │
│  ☐ CPU                      │
│  ☐ Memory                   │
└─────────────────────────────┘
```

- Name column checkbox is always checked and disabled (cannot hide).
- Checked columns appear in current order at top, unchecked at bottom.
- Up/down arrows disabled at boundaries.
- Reset button clears the GVR's `columnPrefs` entry entirely.
- Compact checkbox calls `ConfigService.SetCompactRows()`.
- Sparklines section only rendered when `sparklineGvrs.includes(gvr)`.

#### Wails bindings regeneration

After adding Go service methods, run `wails3 generate bindings`.

---

### Phase 2 — Expanded Builtin Columns

New columns and enrichers for each builtin GVR. All new columns are `Hidden: true` by default (available but not shown unless user enables them).

#### Pods (`core.v1.pods`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Node | `spec.nodeName` | text | No |
| IP | `status.podIP` | text | No |
| QoS | `status.qosClass` | badge | No |

#### Deployments (`apps.v1.deployments`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Up-to-date | `status.updatedReplicas` | text | No |
| Strategy | `spec.strategy.type` | badge | No |

#### StatefulSets (`apps.v1.statefulsets`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Available | `status.availableReplicas` | text | No |
| Current | `status.currentReplicas` | text | No |
| Updated | `status.updatedReplicas` | text | No |

#### DaemonSets (`apps.v1.daemonsets`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Desired | `status.desiredNumberScheduled` | text | No |
| Available | `status.numberAvailable` | text | No |
| Node Selector | `status.nodeSelectorDisplay` | text | Yes — flatten `spec.nodeSelector` map to `key=val,...` string |

#### ReplicaSets (`apps.v1.replicasets`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Replicas | `status.replicas` | text | No |
| Owner | `status.ownerDisplay` | text | Yes — extract first `metadata.ownerReferences[].name` |

#### Jobs (`batch.v1.jobs`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Status | `status.statusDisplay` | badge | Yes — compute "Complete"/"Failed"/"Running" from conditions (enrich in existing `JobEnricher`) |
| Backoff Limit | `spec.backoffLimit` | text | No |

#### CronJobs (`batch.v1.cronjobs`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Suspend | `spec.suspend` | badge | No |
| Last Schedule | `status.lastScheduleTime` | age | No |
| Active | `status.activeCount` | text | Yes — enrich `len(status.active)` to `status.activeCount` |

#### Services (`core.v1.services`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| External IP | `status.externalIPDisplay` | text | Yes — flatten `status.loadBalancer.ingress[].ip` or `spec.externalIPs[]` |
| Ports | `status.portsDisplay` | text | Yes — flatten `spec.ports[]` to `80/TCP, 443/TCP` format |

#### Ingresses (`networking.k8s.io.v1.ingresses`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Class | `spec.ingressClassName` | text | No |
| Hosts | `status.hostsDisplay` | text | Yes — flatten `spec.rules[].host` to comma-separated |
| Default Backend | `status.defaultBackendDisplay` | text | Yes — format `spec.defaultBackend.service.name:port` |

#### ConfigMaps (`core.v1.configmaps`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Keys | `status.dataKeysCount` | text | Yes — `len(data)` |

#### Secrets (`core.v1.secrets`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Keys | `status.dataKeysCount` | text | Yes — `len(data)` |

#### PersistentVolumes (`core.v1.persistentvolumes`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Capacity | `spec.capacity.storage` | text | No |
| Access Modes | `status.accessModesDisplay` | text | Yes — flatten `spec.accessModes[]` to short form `RWO,ROX` |
| Storage Class | `spec.storageClassName` | text | No |
| Claim | `status.claimDisplay` | text | Yes — format `spec.claimRef.namespace/spec.claimRef.name` |

#### PersistentVolumeClaims (`core.v1.persistentvolumeclaims`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Capacity | `status.capacity.storage` | text | No |
| Access Modes | `status.accessModesDisplay` | text | Yes — flatten `spec.accessModes[]` |
| Storage Class | `spec.storageClassName` | text | No |

#### Nodes (`core.v1.nodes`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Version | `status.nodeInfo.kubeletVersion` | text | No |
| Internal IP | `status.internalIPDisplay` | text | Yes — find `status.addresses[]` where `type == InternalIP` |
| OS/Arch | `status.osArchDisplay` | text | Yes — combine `status.nodeInfo.operatingSystem/architecture` |

#### ServiceAccounts (`core.v1.serviceaccounts`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Secrets | `status.secretsCount` | text | Yes — `len(secrets)` |

#### Roles / ClusterRoles (`rbac.authorization.k8s.io.v1.roles`, `clusterroles`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Rules | `status.rulesCount` | text | Yes — `len(rules)` |

#### RoleBindings / ClusterRoleBindings (`rbac.authorization.k8s.io.v1.rolebindings`, `clusterrolebindings`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Role | `status.roleRefDisplay` | text | Yes — format `roleRef.kind/roleRef.name` |
| Subjects | `status.subjectsCount` | text | Yes — `len(subjects)` |

#### StorageClasses (`storage.k8s.io.v1.storageclasses`)

| Column | Expr | Render | Enricher needed |
|--------|------|--------|-----------------|
| Allow Expansion | `allowVolumeExpansion` | badge | No |

#### New enrichers summary (Phase 2)

| Enricher | GVRs | Fields computed |
|----------|------|-----------------|
| `DaemonSetEnricher` (extend) | daemonsets | `status.nodeSelectorDisplay` |
| `ReplicaSetEnricher` (new) | replicasets | `status.ownerDisplay` |
| `JobEnricher` (extend) | jobs | `status.statusDisplay` |
| `CronJobEnricher` (new) | cronjobs | `status.activeCount` |
| `ServiceEnricher` (new) | services | `status.externalIPDisplay`, `status.portsDisplay` |
| `IngressEnricher` (new) | ingresses | `status.hostsDisplay`, `status.defaultBackendDisplay` |
| `ConfigMapEnricher` (new) | configmaps | `status.dataKeysCount` |
| `SecretEnricher` (new) | secrets | `status.dataKeysCount` |
| `PVEnricher` (new) | persistentvolumes | `status.accessModesDisplay`, `status.claimDisplay` |
| `PVCEnricher` (new) | persistentvolumeclaims | `status.accessModesDisplay` |
| `NodeEnricher` (extend) | nodes | `status.internalIPDisplay`, `status.osArchDisplay` |
| `ServiceAccountEnricher` (new) | serviceaccounts | `status.secretsCount` |
| `RoleEnricher` (new) | roles, clusterroles | `status.rulesCount` |
| `BindingEnricher` (new) | rolebindings, clusterrolebindings | `status.roleRefDisplay`, `status.subjectsCount` |

## Definition of Done

### Phase 1 — Column Infrastructure
- [ ] `config.json` stores `columnPrefs` (per-GVR) and `compactRows` (global)
- [ ] `ConfigService` exposes Get/Set RPCs for column prefs and compact mode
- [ ] Wails bindings regenerated
- [ ] `Column` struct has `Align` and `Hidden` fields
- [ ] Every non-cluster-scoped builtin descriptor includes a Namespace column (`Hidden: true`)
- [ ] `columns.svelte.ts` store manages merged prefs + descriptor defaults
- [ ] `ColumnMenu.svelte` dropdown: visibility checkboxes, up/down reorder, compact toggle, sparkline toggles, reset button
- [ ] Column resize handles in header with mousedown/mousemove/mouseup interaction
- [ ] Double-click resize handle triggers auto-fit (measures visible rows)
- [ ] Minimum column width enforced at 20px
- [ ] First column sticky with background and right drop shadow
- [ ] Cell text alignment based on render type
- [ ] Cell content has `title` attribute for truncation tooltip
- [ ] Compact mode toggle changes row height (36px → 28px)
- [ ] Clicking namespace value in Namespace column sets global namespace filter
- [ ] Sort column/direction persisted in column prefs
- [ ] All preferences debounce-saved to config
- [ ] Frontend type-checks cleanly (`pnpm check`)
- [ ] Existing tests pass

### Phase 2 — Expanded Columns
- [ ] All new enrichers implemented with unit tests
- [ ] All new columns added to builtin descriptors (`Hidden: true`)
- [ ] Enrichers registered in `RegisterBuiltin()`
- [ ] Go tests pass (`go test ./internal/resource/... -v`)
- [ ] New columns visible and functional when user enables them via column menu
