# Bulk Operations & List Enhancements — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add multi-select, bulk actions (delete/label/annotate/scale), annotation filtering, and export to the resource list.

**Architecture:** A new `SelectionStore` singleton manages selection state, decoupled from `ResourceStore` for plugin extensibility. `ResourceList.svelte` gets a checkbox column. A floating `BulkActionBar.svelte` appears when items are selected, rendered in `Layout.svelte`. Bulk operations are orchestrated client-side with sequential calls to existing backend methods (`DeleteResource`, `UpdateResource`, `ScaleResource`). Annotation filtering extends the existing filter bar with chip-based UI. Export serializes client-side data to YAML/JSON file downloads.

**Tech Stack:** Svelte 5 runes, bits-ui Dialog, `yaml` npm package (already installed), existing `@klados/ui` components (`KeyValuePairEditor`, `ConfirmDialog`), existing `ResourceService` Wails bindings.

---

## File Structure

**New files:**
| File | Responsibility |
|---|---|
| `frontend/src/lib/stores/selection.svelte.ts` | SelectionStore singleton — selection state, keys, items, visible tracking |
| `frontend/src/lib/components/BulkActionBar.svelte` | Floating bottom bar — shows when selection > 0, action buttons |
| `frontend/src/lib/components/BulkDeleteDialog.svelte` | Progress dialog for bulk delete |
| `frontend/src/lib/components/BulkMetadataDialog.svelte` | Shared label/annotation editor for bulk patch |
| `frontend/src/lib/components/BulkScaleDialog.svelte` | Scale dialog with set/increase/decrease modes |
| `frontend/src/lib/components/AnnotationFilter.svelte` | Chip-based annotation filter popover |
| `frontend/src/lib/utils/export.ts` | Export helpers — serialize items to YAML/JSON, trigger download |

**Modified files:**
| File | Changes |
|---|---|
| `frontend/src/lib/components/ResourceList.svelte` | Add checkbox column, shift+click, expose visible keys, integrate annotation filter |
| `frontend/src/lib/components/Layout.svelte` | Render BulkActionBar |
| `frontend/src/routes/ResourceListPage.svelte` | Clear selection on GVR change, pass context to BulkActionBar |

---

## Task 1: SelectionStore

**Files:**
- Create: `frontend/src/lib/stores/selection.svelte.ts`

- [ ] **Step 1: Create SelectionStore**

```typescript
// frontend/src/lib/stores/selection.svelte.ts

function resourceKey(obj: Record<string, any>): string {
  const ns = obj.metadata?.namespace ?? ''
  const name = obj.metadata?.name ?? ''
  return ns ? `${ns}/${name}` : name
}

class SelectionStore {
  selectedKeys = $state<Set<string>>(new Set())
  selectedGVR = $state('')
  selectedItems = $state<Map<string, Record<string, any>>>(new Map())
  visibleKeys = $state<Set<string>>(new Set())

  private lastToggled = $state<string | null>(null)

  get count(): number {
    return this.selectedKeys.size
  }

  get notVisibleCount(): number {
    let count = 0
    for (const key of this.selectedKeys) {
      if (!this.visibleKeys.has(key)) count++
    }
    return count
  }

  toggle(key: string, item: Record<string, any>) {
    const next = new Set(this.selectedKeys)
    const nextItems = new Map(this.selectedItems)
    if (next.has(key)) {
      next.delete(key)
      nextItems.delete(key)
    } else {
      next.add(key)
      nextItems.set(key, item)
    }
    this.selectedKeys = next
    this.selectedItems = nextItems
    this.lastToggled = key
  }

  select(key: string, item: Record<string, any>) {
    if (this.selectedKeys.has(key)) return
    const next = new Set(this.selectedKeys)
    const nextItems = new Map(this.selectedItems)
    next.add(key)
    nextItems.set(key, item)
    this.selectedKeys = next
    this.selectedItems = nextItems
    this.lastToggled = key
  }

  deselect(key: string) {
    if (!this.selectedKeys.has(key)) return
    const next = new Set(this.selectedKeys)
    const nextItems = new Map(this.selectedItems)
    next.delete(key)
    nextItems.delete(key)
    this.selectedKeys = next
    this.selectedItems = nextItems
  }

  selectRange(toKey: string, orderedKeys: string[], itemsByKey: Map<string, Record<string, any>>) {
    const fromKey = this.lastToggled
    if (!fromKey) return
    const fromIdx = orderedKeys.indexOf(fromKey)
    const toIdx = orderedKeys.indexOf(toKey)
    if (fromIdx < 0 || toIdx < 0) return
    const [start, end] = fromIdx < toIdx ? [fromIdx, toIdx] : [toIdx, fromIdx]
    const next = new Set(this.selectedKeys)
    const nextItems = new Map(this.selectedItems)
    for (let i = start; i <= end; i++) {
      const k = orderedKeys[i]
      next.add(k)
      const item = itemsByKey.get(k)
      if (item) nextItems.set(k, item)
    }
    this.selectedKeys = next
    this.selectedItems = nextItems
    this.lastToggled = toKey
  }

  selectAll(keys: string[], itemsByKey: Map<string, Record<string, any>>) {
    const next = new Set(this.selectedKeys)
    const nextItems = new Map(this.selectedItems)
    for (const k of keys) {
      next.add(k)
      const item = itemsByKey.get(k)
      if (item) nextItems.set(k, item)
    }
    this.selectedKeys = next
    this.selectedItems = nextItems
  }

  deselectAll() {
    this.selectedKeys = new Set()
    this.selectedItems = new Map()
    this.lastToggled = null
  }

  clear() {
    this.selectedKeys = new Set()
    this.selectedItems = new Map()
    this.selectedGVR = ''
    this.lastToggled = null
    this.visibleKeys = new Set()
  }

  isSelected(key: string): boolean {
    return this.selectedKeys.has(key)
  }

  items(): Record<string, any>[] {
    return Array.from(this.selectedItems.values())
  }

  setVisibleKeys(keys: Set<string>) {
    this.visibleKeys = keys
  }

  setGVR(gvr: string) {
    if (this.selectedGVR && this.selectedGVR !== gvr) {
      this.clear()
    }
    this.selectedGVR = gvr
  }

  /** Deselect specific keys (e.g. after successful bulk operation) */
  deselectKeys(keys: string[]) {
    const next = new Set(this.selectedKeys)
    const nextItems = new Map(this.selectedItems)
    for (const k of keys) {
      next.delete(k)
      nextItems.delete(k)
    }
    this.selectedKeys = next
    this.selectedItems = nextItems
  }
}

export const selectionStore = new SelectionStore()
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd /home/vilsol/Projects/Vilsol/klados/frontend && npx svelte-check --tsconfig ./tsconfig.json 2>&1 | head -30`

Expected: No errors in `selection.svelte.ts`

- [ ] **Step 3: Commit**

```bash
jj desc -m "feat: add SelectionStore for multi-select state management"
jj new
```

---

## Task 2: Checkbox Column in ResourceList

**Files:**
- Modify: `frontend/src/lib/components/ResourceList.svelte`

- [ ] **Step 1: Import selectionStore and add helper**

At the top of `ResourceList.svelte`, add import and key helper:

```typescript
import { selectionStore } from '$lib/stores/selection.svelte'
import { Check, Minus } from 'lucide-svelte'

function itemKey(obj: Record<string, any>): string {
  const ns = obj.metadata?.namespace ?? ''
  const name = obj.metadata?.name ?? ''
  return ns ? `${ns}/${name}` : name
}
```

- [ ] **Step 2: Add visible keys tracking and select-all state**

After the `filtered` derived, add:

```typescript
// Build ordered keys and lookup map for the current filtered view
const filteredKeys = $derived(filtered.map(item => itemKey(item)))
const filteredItemsByKey = $derived(() => {
  const map = new Map<string, Record<string, any>>()
  for (const item of filtered) {
    map.set(itemKey(item), item)
  }
  return map
})

// Update selectionStore's visible keys whenever filtered changes
$effect(() => {
  selectionStore.setVisibleKeys(new Set(filteredKeys))
})

// Select-all checkbox state
const allVisibleSelected = $derived(
  filtered.length > 0 && filteredKeys.every(k => selectionStore.isSelected(k))
)
const someVisibleSelected = $derived(
  !allVisibleSelected && filteredKeys.some(k => selectionStore.isSelected(k))
)
const canMutate = $derived(clusterStore.canMutate())
```

- [ ] **Step 3: Update gridTemplateCols to include checkbox column**

Replace the `gridTemplateCols` derived:

```typescript
const gridTemplateCols = $derived(
  (canMutate ? '36px ' : '')
  + columnStore.visibleColumns
    .map((c) => c.width ? `${c.width}px` : 'minmax(20px, 1fr)')
    .join(' ')
  + (pluginColumns.length ? ' ' + pluginColumns.map(() => '1fr').join(' ') : '')
  + (sparklineColumns.length ? ' ' + sparklineColumns.map(() => '80px').join(' ') : '')
  + ' 36px'
)
```

- [ ] **Step 4: Add select-all checkbox to header**

In the grid header (the `<div class="grid text-xs font-semibold...">` section), add before the `{#each columnStore.visibleColumns}` loop:

```svelte
{#if canMutate}
  <div class="flex items-center justify-center {columnStore.compact ? 'py-1' : 'py-2'}">
    <button
      onclick={() => {
        if (allVisibleSelected) {
          selectionStore.deselectAll()
        } else {
          selectionStore.selectAll(filteredKeys, filteredItemsByKey())
        }
      }}
      class="w-4 h-4 rounded border border-border flex items-center justify-center hover:border-accent transition-colors
        {allVisibleSelected || someVisibleSelected ? 'bg-accent border-accent' : ''}"
      aria-label={allVisibleSelected ? 'Deselect all' : 'Select all'}
    >
      {#if allVisibleSelected}
        <Check size={10} class="text-accent-fg" />
      {:else if someVisibleSelected}
        <Minus size={10} class="text-accent-fg" />
      {/if}
    </button>
  </div>
{/if}
```

- [ ] **Step 5: Add row checkbox**

In the row template (inside the `<div class="grid flex-1 min-w-0">`), add before the `{#each columnStore.visibleColumns}` loop:

```svelte
{#if canMutate}
  {@const key = itemKey(item)}
  <div class="flex items-center justify-center"
    onclick={(e) => e.stopPropagation()}
  >
    <button
      onclick={(e) => {
        e.stopPropagation()
        if (e.shiftKey) {
          selectionStore.selectRange(key, filteredKeys, filteredItemsByKey())
        } else {
          selectionStore.toggle(key, item)
        }
      }}
      class="w-4 h-4 rounded border border-border flex items-center justify-center hover:border-accent transition-colors
        {selectionStore.isSelected(key) ? 'bg-accent border-accent' : ''}"
      aria-label={selectionStore.isSelected(key) ? 'Deselect' : 'Select'}
    >
      {#if selectionStore.isSelected(key)}
        <Check size={10} class="text-accent-fg" />
      {/if}
    </button>
  </div>
{/if}
```

- [ ] **Step 6: Add empty header cell for checkbox column in header**

In the grid header, also add an empty `<div></div>` matching the checkbox column before the column headers (already covered by the select-all button above — but also add a matching empty cell in the plugin/sparkline header area if needed, or verify alignment is correct).

- [ ] **Step 7: Verify visually**

Run: `cd /home/vilsol/Projects/Vilsol/klados && task dev`

Expected: Checkbox column appears as the first column when connected to a cluster. Clicking toggles selection. Shift+click selects range. Select-all works. Checkboxes hidden in read-only mode.

- [ ] **Step 8: Commit**

```bash
jj desc -m "feat: add checkbox column to ResourceList for multi-select"
jj new
```

---

## Task 3: BulkActionBar

**Files:**
- Create: `frontend/src/lib/components/BulkActionBar.svelte`
- Modify: `frontend/src/lib/components/Layout.svelte`

- [ ] **Step 1: Create BulkActionBar component**

```svelte
<!-- frontend/src/lib/components/BulkActionBar.svelte -->
<script lang="ts">
  import { selectionStore } from '$lib/stores/selection.svelte'
  import { Trash2, Tag, StickyNote, Scale, Download, X } from 'lucide-svelte'
  import BulkDeleteDialog from './BulkDeleteDialog.svelte'
  import BulkMetadataDialog from './BulkMetadataDialog.svelte'
  import BulkScaleDialog from './BulkScaleDialog.svelte'
  import { stringify } from 'yaml'

  let {
    contextName = '',
    gvr = '',
  }: {
    contextName?: string
    gvr?: string
  } = $props()

  const count = $derived(selectionStore.count)
  const notVisible = $derived(selectionStore.notVisibleCount)
  const showScale = $derived(
    selectionStore.selectedGVR === 'apps.v1.deployments' ||
    selectionStore.selectedGVR === 'apps.v1.statefulsets'
  )

  let deleteOpen = $state(false)
  let labelOpen = $state(false)
  let annotateOpen = $state(false)
  let scaleOpen = $state(false)

  function exportAs(format: 'yaml' | 'json') {
    const items = selectionStore.items()
    if (items.length === 0) return

    let content: string
    let ext: string
    if (format === 'yaml') {
      content = items.map(item => stringify(item)).join('---\n')
      ext = 'yaml'
    } else {
      content = JSON.stringify(items, null, 2)
      ext = 'json'
    }

    const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
    const filename = `${selectionStore.selectedGVR}-${timestamp}.${ext}`
    const blob = new Blob([content], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  }

  let exportMenuOpen = $state(false)
</script>

{#if count > 0}
  <div
    class="fixed bottom-6 left-1/2 -translate-x-1/2 z-30 bg-surface border border-border rounded-lg shadow-xl px-4 py-2.5 flex items-center gap-3 transition-all animate-slide-up"
  >
    <span class="text-sm font-medium whitespace-nowrap">
      {count} selected{#if notVisible > 0} <span class="text-muted">({notVisible} not visible)</span>{/if}
    </span>

    <button
      onclick={() => selectionStore.deselectAll()}
      class="p-1 rounded hover:bg-surface-hover transition-colors text-muted hover:text-fg"
      title="Clear selection"
      aria-label="Clear selection"
    >
      <X size={14} />
    </button>

    <div class="w-px h-5 bg-border"></div>

    <button
      onclick={() => deleteOpen = true}
      class="flex items-center gap-1.5 px-2.5 py-1 text-sm rounded hover:bg-destructive/10 hover:text-destructive transition-colors"
      title="Delete selected"
    >
      <Trash2 size={13} />
      Delete
    </button>

    <button
      onclick={() => labelOpen = true}
      class="flex items-center gap-1.5 px-2.5 py-1 text-sm rounded hover:bg-surface-hover transition-colors"
      title="Edit labels"
    >
      <Tag size={13} />
      Labels
    </button>

    <button
      onclick={() => annotateOpen = true}
      class="flex items-center gap-1.5 px-2.5 py-1 text-sm rounded hover:bg-surface-hover transition-colors"
      title="Edit annotations"
    >
      <StickyNote size={13} />
      Annotations
    </button>

    {#if showScale}
      <button
        onclick={() => scaleOpen = true}
        class="flex items-center gap-1.5 px-2.5 py-1 text-sm rounded hover:bg-surface-hover transition-colors"
        title="Scale selected"
      >
        <Scale size={13} />
        Scale
      </button>
    {/if}

    <div class="relative">
      <button
        onclick={() => exportMenuOpen = !exportMenuOpen}
        class="flex items-center gap-1.5 px-2.5 py-1 text-sm rounded hover:bg-surface-hover transition-colors"
        title="Export selected"
      >
        <Download size={13} />
        Export
      </button>
      {#if exportMenuOpen}
        <div class="absolute bottom-full mb-1 right-0 bg-surface border border-border rounded shadow-lg py-1 min-w-28">
          <button
            class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
            onclick={() => { exportAs('yaml'); exportMenuOpen = false }}
          >YAML</button>
          <button
            class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
            onclick={() => { exportAs('json'); exportMenuOpen = false }}
          >JSON</button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<BulkDeleteDialog bind:open={deleteOpen} {contextName} />
<BulkMetadataDialog bind:open={labelOpen} mode="labels" {contextName} {gvr} />
<BulkMetadataDialog bind:open={annotateOpen} mode="annotations" {contextName} {gvr} />
<BulkScaleDialog bind:open={scaleOpen} {contextName} />
```

- [ ] **Step 2: Add slide-up animation to app.css**

In `frontend/src/app.css`, add:

```css
@keyframes slide-up {
  from { transform: translateX(-50%) translateY(20px); opacity: 0; }
  to { transform: translateX(-50%) translateY(0); opacity: 1; }
}
.animate-slide-up {
  animation: slide-up 0.15s ease-out;
}
```

- [ ] **Step 3: Render BulkActionBar in Layout.svelte**

In `frontend/src/lib/components/Layout.svelte`, add the import and render:

```svelte
<script lang="ts">
  // ... existing imports ...
  import BulkActionBar from './BulkActionBar.svelte'
  import { clusterStore } from '$lib/stores/cluster.svelte'

  // ... existing code ...
  const activeCtx = $derived(clusterStore.activeContext ?? '')
</script>
```

Then add `<BulkActionBar contextName={activeCtx} />` just before the closing `</div>` of the root flex container (after the status bar widget section).

- [ ] **Step 4: Verify visually**

Select some items in the resource list. The floating bar should appear at bottom center with action buttons. Clear selection should dismiss it.

- [ ] **Step 5: Commit**

```bash
jj desc -m "feat: add BulkActionBar floating component in Layout"
jj new
```

---

## Task 4: BulkDeleteDialog

**Files:**
- Create: `frontend/src/lib/components/BulkDeleteDialog.svelte`

- [ ] **Step 1: Create BulkDeleteDialog**

```svelte
<!-- frontend/src/lib/components/BulkDeleteDialog.svelte -->
<script lang="ts">
  import { Dialog } from 'bits-ui'
  import { selectionStore } from '$lib/stores/selection.svelte'
  import { notificationStore } from '$lib/stores/notification.svelte'
  import { Check, X, Loader2 } from 'lucide-svelte'
  import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'

  let {
    open = $bindable(false),
    contextName,
  }: {
    open: boolean
    contextName: string
  } = $props()

  type ItemStatus = 'pending' | 'deleting' | 'success' | 'error'
  let statuses = $state<Map<string, { status: ItemStatus; error?: string }>>(new Map())
  let running = $state(false)

  const selectedItems = $derived(selectionStore.items())
  const gvr = $derived(selectionStore.selectedGVR)

  function itemKey(obj: Record<string, any>): string {
    const ns = obj.metadata?.namespace ?? ''
    const name = obj.metadata?.name ?? ''
    return ns ? `${ns}/${name}` : name
  }

  async function run() {
    running = true
    const items = [...selectedItems]
    statuses = new Map(items.map(item => [itemKey(item), { status: 'pending' as ItemStatus }]))

    const succeeded: string[] = []
    let failCount = 0

    for (const item of items) {
      const key = itemKey(item)
      const ns = item.metadata?.namespace ?? ''
      const name = item.metadata?.name ?? ''

      statuses = new Map(statuses).set(key, { status: 'deleting' })

      try {
        await ResourceService.DeleteResource(contextName, gvr, ns, name)
        statuses = new Map(statuses).set(key, { status: 'success' })
        succeeded.push(key)
      } catch (e: any) {
        statuses = new Map(statuses).set(key, { status: 'error', error: e?.message ?? String(e) })
        failCount++
      }
    }

    selectionStore.deselectKeys(succeeded)
    running = false

    if (failCount === 0) {
      notificationStore.push(`Deleted ${succeeded.length}/${items.length} resources`, 'success')
      open = false
    } else {
      notificationStore.push(`Deleted ${succeeded.length}/${items.length} — ${failCount} failed`, 'error')
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl p-6 w-[480px] max-w-[90vw] max-h-[70vh] flex flex-col">
      <Dialog.Title class="text-base font-semibold mb-2">Delete {selectedItems.length} resources</Dialog.Title>
      <Dialog.Description class="text-sm text-muted mb-4">This action cannot be undone.</Dialog.Description>

      <div class="flex-1 overflow-auto mb-4 border border-border rounded">
        {#each selectedItems as item}
          {@const key = itemKey(item)}
          {@const st = statuses.get(key)}
          <div class="flex items-center gap-2 px-3 py-1.5 text-sm border-b border-border last:border-b-0">
            <span class="w-4 flex-shrink-0">
              {#if st?.status === 'success'}
                <Check size={14} class="text-accent" />
              {:else if st?.status === 'error'}
                <X size={14} class="text-destructive" />
              {:else if st?.status === 'deleting'}
                <Loader2 size={14} class="animate-spin text-muted" />
              {/if}
            </span>
            <span class="truncate flex-1">{item.metadata?.namespace ? `${item.metadata.namespace}/` : ''}{item.metadata?.name}</span>
            {#if st?.error}
              <span class="text-xs text-destructive truncate max-w-48" title={st.error}>{st.error}</span>
            {/if}
          </div>
        {/each}
      </div>

      <div class="flex justify-end gap-2">
        <Dialog.Close
          class="px-3 py-1.5 text-sm rounded border border-border hover:bg-surface-hover transition-colors"
          disabled={running}
        >Cancel</Dialog.Close>
        <button
          onclick={run}
          disabled={running}
          class="px-3 py-1.5 text-sm rounded bg-destructive text-destructive-fg hover:opacity-90 transition-opacity disabled:opacity-50"
        >
          {running ? 'Deleting…' : 'Delete'}
        </button>
      </div>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
```

- [ ] **Step 2: Verify visually**

Select items, click Delete in the bulk bar. Dialog should list items. Click Delete — items should show progress, then success/error icons.

- [ ] **Step 3: Commit**

```bash
jj desc -m "feat: add BulkDeleteDialog with per-item progress"
jj new
```

---

## Task 5: BulkMetadataDialog (Labels & Annotations)

**Files:**
- Create: `frontend/src/lib/components/BulkMetadataDialog.svelte`

- [ ] **Step 1: Create BulkMetadataDialog**

```svelte
<!-- frontend/src/lib/components/BulkMetadataDialog.svelte -->
<script lang="ts">
  import { Dialog } from 'bits-ui'
  import { KeyValuePairEditor } from '@klados/ui'
  import { selectionStore } from '$lib/stores/selection.svelte'
  import { notificationStore } from '$lib/stores/notification.svelte'
  import { Check, X, Loader2 } from 'lucide-svelte'
  import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'

  let {
    open = $bindable(false),
    mode,
    contextName,
    gvr,
  }: {
    open: boolean
    mode: 'labels' | 'annotations'
    contextName: string
    gvr: string
  } = $props()

  const title = $derived(mode === 'labels' ? 'Edit Labels' : 'Edit Annotations')
  const metadataField = $derived(mode === 'labels' ? 'labels' : 'annotations')

  let pairs = $state<[string, string][]>([])
  let removeKeys = $state<string[]>([])

  type ItemStatus = 'pending' | 'patching' | 'success' | 'error'
  let statuses = $state<Map<string, { status: ItemStatus; error?: string }>>(new Map())
  let running = $state(false)

  function itemKey(obj: Record<string, any>): string {
    const ns = obj.metadata?.namespace ?? ''
    const name = obj.metadata?.name ?? ''
    return ns ? `${ns}/${name}` : name
  }

  // Compute common keys across all selected items
  const commonEntries = $derived.by(() => {
    const items = selectionStore.items()
    if (items.length === 0) return [] as [string, string][]
    const first = items[0].metadata?.[metadataField] ?? {}
    const common: [string, string][] = []
    for (const [k, v] of Object.entries(first)) {
      if (items.every(item => (item.metadata?.[metadataField] ?? {})[k] === v)) {
        common.push([k, String(v)])
      }
    }
    return common
  })

  // All unique keys across selected items
  const allKeys = $derived.by(() => {
    const keys = new Set<string>()
    for (const item of selectionStore.items()) {
      for (const k of Object.keys(item.metadata?.[metadataField] ?? {})) {
        keys.add(k)
      }
    }
    return Array.from(keys).sort()
  })

  // Reset state when dialog opens
  $effect(() => {
    if (open) {
      pairs = [...commonEntries]
      removeKeys = []
      statuses = new Map()
      running = false
    }
  })

  async function run() {
    running = true
    const items = [...selectionStore.items()]
    statuses = new Map(items.map(item => [itemKey(item), { status: 'pending' as ItemStatus }]))

    const addEntries = Object.fromEntries(pairs.filter(([k]) => k.trim()))
    const succeeded: string[] = []
    let failCount = 0

    for (const item of items) {
      const key = itemKey(item)
      const ns = item.metadata?.namespace ?? ''
      const name = item.metadata?.name ?? ''

      statuses = new Map(statuses).set(key, { status: 'patching' })

      try {
        const updated = JSON.parse(JSON.stringify(item))
        const current = updated.metadata[metadataField] ?? {}
        // Remove keys
        for (const rk of removeKeys) {
          delete current[rk]
        }
        // Add/overwrite keys
        Object.assign(current, addEntries)
        updated.metadata[metadataField] = current

        await ResourceService.UpdateResource(contextName, gvr, ns, updated)
        statuses = new Map(statuses).set(key, { status: 'success' })
        succeeded.push(key)
      } catch (e: any) {
        statuses = new Map(statuses).set(key, { status: 'error', error: e?.message ?? String(e) })
        failCount++
      }
    }

    selectionStore.deselectKeys(succeeded)
    running = false

    if (failCount === 0) {
      notificationStore.push(`Updated ${mode} on ${succeeded.length} resources`, 'success')
      open = false
    } else {
      notificationStore.push(`Updated ${succeeded.length}/${items.length} — ${failCount} failed`, 'error')
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl p-6 w-[560px] max-w-[90vw] max-h-[80vh] flex flex-col">
      <Dialog.Title class="text-base font-semibold mb-1">{title}</Dialog.Title>
      <Dialog.Description class="text-sm text-muted mb-4">
        Editing {selectionStore.count} resources. Changes apply to all selected items.
      </Dialog.Description>

      <div class="flex-1 overflow-auto mb-4 flex flex-col gap-4">
        <section>
          <h3 class="text-xs font-medium mb-2">Add / Update</h3>
          <KeyValuePairEditor bind:pairs addLabel="+ Add {mode === 'labels' ? 'label' : 'annotation'}" />
        </section>

        {#if allKeys.length > 0}
          <section>
            <h3 class="text-xs font-medium mb-2">Remove Keys</h3>
            <div class="flex flex-wrap gap-1.5">
              {#each allKeys as key}
                <button
                  onclick={() => {
                    if (removeKeys.includes(key)) {
                      removeKeys = removeKeys.filter(k => k !== key)
                    } else {
                      removeKeys = [...removeKeys, key]
                      pairs = pairs.filter(([k]) => k !== key)
                    }
                  }}
                  class="px-2 py-0.5 text-xs rounded border transition-colors
                    {removeKeys.includes(key)
                      ? 'bg-destructive/10 text-destructive border-destructive/30 line-through'
                      : 'border-border hover:bg-surface-hover'}"
                >
                  {key}
                </button>
              {/each}
            </div>
          </section>
        {/if}

        {#if running || statuses.size > 0}
          <section>
            <h3 class="text-xs font-medium mb-2">Progress</h3>
            <div class="border border-border rounded max-h-40 overflow-auto">
              {#each selectionStore.items() as item}
                {@const key = itemKey(item)}
                {@const st = statuses.get(key)}
                <div class="flex items-center gap-2 px-3 py-1 text-sm border-b border-border last:border-b-0">
                  <span class="w-4 flex-shrink-0">
                    {#if st?.status === 'success'}
                      <Check size={14} class="text-accent" />
                    {:else if st?.status === 'error'}
                      <X size={14} class="text-destructive" />
                    {:else if st?.status === 'patching'}
                      <Loader2 size={14} class="animate-spin text-muted" />
                    {/if}
                  </span>
                  <span class="truncate flex-1">{item.metadata?.name}</span>
                  {#if st?.error}
                    <span class="text-xs text-destructive truncate max-w-48" title={st.error}>{st.error}</span>
                  {/if}
                </div>
              {/each}
            </div>
          </section>
        {/if}
      </div>

      <div class="flex justify-end gap-2">
        <Dialog.Close
          class="px-3 py-1.5 text-sm rounded border border-border hover:bg-surface-hover transition-colors"
          disabled={running}
        >Cancel</Dialog.Close>
        <button
          onclick={run}
          disabled={running || (pairs.filter(([k]) => k.trim()).length === 0 && removeKeys.length === 0)}
          class="px-3 py-1.5 text-sm rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity disabled:opacity-50"
        >
          {running ? 'Applying…' : 'Apply'}
        </button>
      </div>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
```

- [ ] **Step 2: Verify visually**

Select items, click Labels or Annotations in the bulk bar. Dialog should show common entries, allow add/remove, and show progress on apply.

- [ ] **Step 3: Commit**

```bash
jj desc -m "feat: add BulkMetadataDialog for bulk label/annotation editing"
jj new
```

---

## Task 6: BulkScaleDialog

**Files:**
- Create: `frontend/src/lib/components/BulkScaleDialog.svelte`

- [ ] **Step 1: Create BulkScaleDialog**

```svelte
<!-- frontend/src/lib/components/BulkScaleDialog.svelte -->
<script lang="ts">
  import { Dialog } from 'bits-ui'
  import { selectionStore } from '$lib/stores/selection.svelte'
  import { notificationStore } from '$lib/stores/notification.svelte'
  import { Check, X, Loader2 } from 'lucide-svelte'
  import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'

  let {
    open = $bindable(false),
    contextName,
  }: {
    open: boolean
    contextName: string
  } = $props()

  type ScaleMode = 'set' | 'increase' | 'decrease'
  let mode = $state<ScaleMode>('set')
  let value = $state(1)

  type ItemStatus = 'pending' | 'scaling' | 'success' | 'error'
  let statuses = $state<Map<string, { status: ItemStatus; error?: string }>>(new Map())
  let running = $state(false)

  function itemKey(obj: Record<string, any>): string {
    const ns = obj.metadata?.namespace ?? ''
    const name = obj.metadata?.name ?? ''
    return ns ? `${ns}/${name}` : name
  }

  function currentReplicas(item: Record<string, any>): number {
    return item.spec?.replicas ?? 0
  }

  function targetReplicas(item: Record<string, any>): number {
    const current = currentReplicas(item)
    switch (mode) {
      case 'set': return Math.max(0, value)
      case 'increase': return current + value
      case 'decrease': return Math.max(0, current - value)
    }
  }

  const gvr = $derived(selectionStore.selectedGVR)
  const selectedItems = $derived(selectionStore.items())

  $effect(() => {
    if (open) {
      mode = 'set'
      value = 1
      statuses = new Map()
      running = false
    }
  })

  async function run() {
    running = true
    const items = [...selectedItems]
    statuses = new Map(items.map(item => [itemKey(item), { status: 'pending' as ItemStatus }]))

    const succeeded: string[] = []
    let failCount = 0

    for (const item of items) {
      const key = itemKey(item)
      const ns = item.metadata?.namespace ?? ''
      const name = item.metadata?.name ?? ''
      const target = targetReplicas(item)

      statuses = new Map(statuses).set(key, { status: 'scaling' })

      try {
        await ResourceService.ScaleResource(contextName, gvr, ns, name, target)
        statuses = new Map(statuses).set(key, { status: 'success' })
        succeeded.push(key)
      } catch (e: any) {
        statuses = new Map(statuses).set(key, { status: 'error', error: e?.message ?? String(e) })
        failCount++
      }
    }

    selectionStore.deselectKeys(succeeded)
    running = false

    if (failCount === 0) {
      notificationStore.push(`Scaled ${succeeded.length}/${items.length} resources`, 'success')
      open = false
    } else {
      notificationStore.push(`Scaled ${succeeded.length}/${items.length} — ${failCount} failed`, 'error')
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl p-6 w-[480px] max-w-[90vw] max-h-[80vh] flex flex-col">
      <Dialog.Title class="text-base font-semibold mb-1">Scale {selectedItems.length} resources</Dialog.Title>
      <Dialog.Description class="text-sm text-muted mb-4">Adjust replica count for selected {gvr.split('.').at(-1)}.</Dialog.Description>

      <div class="flex-1 overflow-auto mb-4 flex flex-col gap-4">
        <div class="flex gap-2">
          {#each [['set', 'Set to'], ['increase', 'Increase by'], ['decrease', 'Decrease by']] as [m, label]}
            <button
              onclick={() => mode = m as ScaleMode}
              class="px-3 py-1 text-sm rounded border transition-colors
                {mode === m ? 'bg-accent text-accent-fg border-accent' : 'border-border hover:bg-surface-hover'}"
            >{label}</button>
          {/each}
        </div>

        <input
          type="number"
          min="0"
          bind:value={value}
          class="w-24 px-2 py-1 text-sm border border-border rounded bg-transparent outline-none focus:border-accent"
        />

        <div class="border border-border rounded max-h-52 overflow-auto">
          <div class="grid grid-cols-[1fr_80px_20px_80px] gap-2 px-3 py-1 text-xs font-medium text-muted border-b border-border">
            <span>Name</span>
            <span class="text-right">Current</span>
            <span></span>
            <span class="text-right">Target</span>
          </div>
          {#each selectedItems as item}
            {@const key = itemKey(item)}
            {@const current = currentReplicas(item)}
            {@const target = targetReplicas(item)}
            {@const st = statuses.get(key)}
            <div class="grid grid-cols-[1fr_80px_20px_80px] gap-2 px-3 py-1.5 text-sm border-b border-border last:border-b-0 items-center">
              <span class="truncate flex items-center gap-1.5">
                {#if st?.status === 'success'}
                  <Check size={12} class="text-accent flex-shrink-0" />
                {:else if st?.status === 'error'}
                  <X size={12} class="text-destructive flex-shrink-0" />
                {:else if st?.status === 'scaling'}
                  <Loader2 size={12} class="animate-spin text-muted flex-shrink-0" />
                {/if}
                {item.metadata?.name}
              </span>
              <span class="text-right text-muted">{current}</span>
              <span class="text-center text-muted">&rarr;</span>
              <span class="text-right {target !== current ? 'text-accent font-medium' : ''}">{target}</span>
            </div>
          {/each}
        </div>
      </div>

      <div class="flex justify-end gap-2">
        <Dialog.Close
          class="px-3 py-1.5 text-sm rounded border border-border hover:bg-surface-hover transition-colors"
          disabled={running}
        >Cancel</Dialog.Close>
        <button
          onclick={run}
          disabled={running}
          class="px-3 py-1.5 text-sm rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity disabled:opacity-50"
        >
          {running ? 'Scaling…' : 'Scale'}
        </button>
      </div>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
```

- [ ] **Step 2: Verify visually**

Select Deployments, click Scale. Toggle between Set to / Increase by / Decrease by. Verify preview shows correct target replicas. Decrease by should floor at 0.

- [ ] **Step 3: Commit**

```bash
jj desc -m "feat: add BulkScaleDialog with set/increase/decrease modes"
jj new
```

---

## Task 7: Annotation Filter

**Files:**
- Create: `frontend/src/lib/components/AnnotationFilter.svelte`
- Modify: `frontend/src/lib/components/ResourceList.svelte`

- [ ] **Step 1: Create AnnotationFilter component**

```svelte
<!-- frontend/src/lib/components/AnnotationFilter.svelte -->
<script lang="ts">
  import { X, Plus } from 'lucide-svelte'

  let {
    filters = $bindable<{ key: string; value: string }[]>([]),
  }: {
    filters: { key: string; value: string }[]
  } = $props()

  let popoverOpen = $state(false)
  let newKey = $state('')
  let newValue = $state('')

  function add() {
    if (!newKey.trim()) return
    filters = [...filters, { key: newKey.trim(), value: newValue.trim() }]
    newKey = ''
    newValue = ''
    popoverOpen = false
  }

  function remove(index: number) {
    filters = filters.filter((_, i) => i !== index)
  }
</script>

<div class="flex items-center gap-1.5 flex-wrap">
  {#each filters as filter, i}
    <span class="flex items-center gap-1 px-2 py-0.5 text-xs rounded-full border border-purple-500/30 bg-purple-500/10 text-purple-300">
      <span class="font-mono">{filter.key}={filter.value}</span>
      <button
        onclick={() => remove(i)}
        class="hover:text-fg transition-colors"
        aria-label="Remove annotation filter"
      >
        <X size={10} />
      </button>
    </span>
  {/each}

  <div class="relative">
    <button
      onclick={() => popoverOpen = !popoverOpen}
      class="flex items-center gap-1 px-1.5 py-0.5 text-xs rounded border border-border hover:bg-surface-hover transition-colors text-muted"
      title="Add annotation filter"
    >
      <Plus size={10} />
      Annotation
    </button>

    {#if popoverOpen}
      <div class="absolute top-full mt-1 left-0 z-50 bg-surface border border-border rounded shadow-lg p-3 flex flex-col gap-2 min-w-48">
        <input
          type="text"
          placeholder="Key"
          bind:value={newKey}
          class="px-2 py-1 text-sm border border-border rounded bg-transparent outline-none focus:border-accent"
          onkeydown={(e) => { if (e.key === 'Enter') add() }}
        />
        <input
          type="text"
          placeholder="Value"
          bind:value={newValue}
          class="px-2 py-1 text-sm border border-border rounded bg-transparent outline-none focus:border-accent"
          onkeydown={(e) => { if (e.key === 'Enter') add() }}
        />
        <button
          onclick={add}
          disabled={!newKey.trim()}
          class="px-2 py-1 text-sm rounded bg-accent text-accent-fg hover:opacity-90 disabled:opacity-50"
        >Add</button>
      </div>
    {/if}
  </div>
</div>
```

- [ ] **Step 2: Integrate annotation filter into ResourceList.svelte**

Add import at top:

```typescript
import AnnotationFilter from './AnnotationFilter.svelte'
```

Add state:

```typescript
let annotationFilters = $state<{ key: string; value: string }[]>([])
```

Update the `filtered` derived to include annotation filtering. Replace the existing filter logic:

```typescript
const filtered = $derived.by(() => {
  let result = items
  if (selectedNamespaces.length > 1) {
    result = result.filter((item) => selectedNamespaces.includes(item.metadata?.namespace ?? ''))
  }
  if (filterText.trim()) {
    const q = filterText.trim().toLowerCase()
    result = result.filter((item) => {
      const labels = item.metadata?.labels ?? {}
      const labelsStr = Object.entries(labels)
        .map(([k, v]) => `${k}=${v}`)
        .join(',')
      return labelsStr.includes(q) || (item.metadata?.name ?? '').toLowerCase().includes(q)
    })
  }
  if (annotationFilters.length > 0) {
    result = result.filter((item) => {
      const annotations = item.metadata?.annotations ?? {}
      return annotationFilters.every(f =>
        f.value ? annotations[f.key] === f.value : f.key in annotations
      )
    })
  }
  if (columnStore.sortState) {
    const { column, direction } = columnStore.sortState
    const col = columnStore.visibleColumns.find((c) => c.name === column)
    if (col?.expr) {
      result = [...result].sort((a, b) => {
        const av = String(evalExpr(col.expr, a) ?? '')
        const bv = String(evalExpr(col.expr, b) ?? '')
        return direction === 'asc' ? av.localeCompare(bv) : bv.localeCompare(av)
      })
    }
  }
  return result
})
```

- [ ] **Step 3: Add AnnotationFilter to the filter bar**

In the filter bar area of ResourceList.svelte (the `<div class="flex items-center gap-2 px-3 py-2 border-b ...">` section), add after the `<input>` element:

```svelte
<AnnotationFilter bind:filters={annotationFilters} />
```

- [ ] **Step 4: Verify visually**

Add annotation filters. Verify the list filters correctly. Verify chips appear and are removable. Verify value-less filter matches any resource with that annotation key present.

- [ ] **Step 5: Commit**

```bash
jj desc -m "feat: add chip-based annotation filtering to ResourceList"
jj new
```

---

## Task 8: Export (Filter Bar)

**Files:**
- Create: `frontend/src/lib/utils/export.ts`
- Modify: `frontend/src/lib/components/ResourceList.svelte`

- [ ] **Step 1: Create export utility**

```typescript
// frontend/src/lib/utils/export.ts
import { stringify } from 'yaml'

export function exportItems(items: Record<string, any>[], gvr: string, format: 'yaml' | 'json') {
  if (items.length === 0) return

  let content: string
  let ext: string
  if (format === 'yaml') {
    content = items.map(item => stringify(item)).join('---\n')
    ext = 'yaml'
  } else {
    content = JSON.stringify(items, null, 2)
    ext = 'json'
  }

  const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
  const filename = `${gvr}-${timestamp}.${ext}`
  const blob = new Blob([content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
```

- [ ] **Step 2: Update BulkActionBar to use shared export**

In `BulkActionBar.svelte`, replace the inline `exportAs` function:

```typescript
import { exportItems } from '$lib/utils/export'

function exportAs(format: 'yaml' | 'json') {
  exportItems(selectionStore.items(), selectionStore.selectedGVR, format)
}
```

Remove the `import { stringify } from 'yaml'` line from BulkActionBar.

- [ ] **Step 3: Add export button to ResourceList filter bar**

Import in ResourceList.svelte:

```typescript
import { exportItems } from '$lib/utils/export'
import { Download } from 'lucide-svelte'
```

Add state:

```typescript
let exportMenuOpen = $state(false)
```

Add in the filter bar (before the column menu button):

```svelte
<div class="relative">
  <button
    onclick={() => exportMenuOpen = !exportMenuOpen}
    class="p-1 rounded hover:bg-surface-hover transition-colors"
    title="Export visible"
    aria-label="Export visible"
  >
    <Download size={14} />
  </button>
  {#if exportMenuOpen}
    <div class="absolute top-full mt-1 right-0 z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-24">
      <button
        class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
        onclick={() => { exportItems(filtered, gvr, 'yaml'); exportMenuOpen = false }}
      >YAML</button>
      <button
        class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
        onclick={() => { exportItems(filtered, gvr, 'json'); exportMenuOpen = false }}
      >JSON</button>
    </div>
  {/if}
</div>
```

Add click-outside handler for export menu:

```typescript
$effect(() => {
  if (!exportMenuOpen) return
  const close = () => { exportMenuOpen = false }
  const timer = setTimeout(() => window.addEventListener('click', close, { once: true }), 0)
  return () => { clearTimeout(timer); window.removeEventListener('click', close) }
})
```

- [ ] **Step 4: Verify**

Click export button in filter bar — dropdown with YAML/JSON. Click one — browser should download a file with the correct content. Also verify export from bulk action bar works.

- [ ] **Step 5: Commit**

```bash
jj desc -m "feat: add YAML/JSON export for visible and selected resources"
jj new
```

---

## Task 9: Selection Lifecycle (Clear on Namespace/GVR Change)

**Files:**
- Modify: `frontend/src/routes/ResourceListPage.svelte`

- [ ] **Step 1: Import and wire up selection clearing**

In `ResourceListPage.svelte`, add import:

```typescript
import { selectionStore } from '$lib/stores/selection.svelte'
```

Add effects for clearing selection:

```typescript
// Set GVR on selectionStore (auto-clears if GVR changed)
$effect(() => {
  if (gvr) selectionStore.setGVR(gvr)
})

// Clear selection on namespace change
$effect(() => {
  selectedNamespaces  // track
  selectionStore.deselectAll()
})
```

Also update the existing GVR change effect to clear selection:

The existing line `$effect(() => { gvr; selectedItem = null; selectedGVR = gvr })` already handles GVR change. The `selectionStore.setGVR(gvr)` call above handles clearing selection on GVR change since `setGVR` calls `clear()` when the GVR differs.

- [ ] **Step 2: Pass GVR to BulkActionBar in Layout**

Since BulkActionBar is in Layout and needs the GVR, it can read it directly from `selectionStore.selectedGVR` (which is already set by ResourceListPage). No additional prop threading needed — the bar already derives `showScale` from `selectionStore.selectedGVR`.

However, `contextName` and `gvr` are needed for the dialogs. Update Layout's BulkActionBar render:

```svelte
<BulkActionBar contextName={activeCtx} gvr={selectionStore.selectedGVR} />
```

Add import in Layout.svelte:

```typescript
import { selectionStore } from '$lib/stores/selection.svelte'
```

- [ ] **Step 3: Verify behavior**

1. Select items on Deployments page
2. Switch namespace → selection clears
3. Navigate to Pods → selection clears
4. Select items, apply name filter → selection preserved
5. Add annotation filter → selection preserved

- [ ] **Step 4: Commit**

```bash
jj desc -m "feat: wire selection lifecycle — clear on namespace/GVR change"
jj new
```

---

## Task 10: Close Export Menus on Outside Click

**Files:**
- Modify: `frontend/src/lib/components/BulkActionBar.svelte`

- [ ] **Step 1: Add click-outside handler for export dropdown in BulkActionBar**

Add this effect (matching the existing pattern in ResourceList for `columnMenuOpen`):

```typescript
$effect(() => {
  if (!exportMenuOpen) return
  const close = () => { exportMenuOpen = false }
  const timer = setTimeout(() => window.addEventListener('click', close, { once: true }), 0)
  return () => { clearTimeout(timer); window.removeEventListener('click', close) }
})
```

- [ ] **Step 2: Add click-outside for AnnotationFilter popover**

In `AnnotationFilter.svelte`, add:

```typescript
$effect(() => {
  if (!popoverOpen) return
  const close = () => { popoverOpen = false }
  const timer = setTimeout(() => window.addEventListener('click', close, { once: true }), 0)
  return () => { clearTimeout(timer); window.removeEventListener('click', close) }
})
```

Add `onclick={(e) => e.stopPropagation()}` on the popover `<div>` to prevent it closing when interacting with the inputs.

- [ ] **Step 3: Commit**

```bash
jj desc -m "fix: close export and annotation filter dropdowns on outside click"
jj new
```

---

## Task 11: Final Type-Check and Integration Test

**Files:**
- All new and modified files

- [ ] **Step 1: Run TypeScript check**

Run: `cd /home/vilsol/Projects/Vilsol/klados/frontend && pnpm check`

Fix any type errors.

- [ ] **Step 2: Run existing frontend tests**

Run: `cd /home/vilsol/Projects/Vilsol/klados/frontend && pnpm test`

Fix any regressions.

- [ ] **Step 3: Manual integration test**

Connect to a cluster and verify the full flow:
1. Checkbox column appears, select-all works
2. Shift+click range select works
3. Floating bar appears with correct actions
4. Bulk delete works with progress
5. Bulk label/annotate works
6. Bulk scale works with set/increase/decrease modes
7. Annotation filter chips work
8. Export (both from filter bar and bulk bar) downloads correct files
9. Selection clears on namespace change and GVR change
10. Selection preserved on filter change
11. Read-only mode hides checkboxes and bulk bar

- [ ] **Step 4: Commit any fixes**

```bash
jj desc -m "fix: address type-check and integration issues"
jj new
```
