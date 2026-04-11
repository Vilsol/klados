# ResourceList Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix 7 issues in ResourceList.svelte — sticky column styling, column resize jump, number sorting, horizontal scrolling, context menu overflow, gridTemplateCols cleanup, and delete button guard.

**Architecture:** All changes are in a single file (`frontend/src/lib/components/ResourceList.svelte`). No backend changes. The existing test file covers basic rendering; we'll update it for the new sticky class and add a sort comparator test.

**Tech Stack:** Svelte 5, Tailwind v4, TanStack Virtual, TypeScript

---

### Task 1: gridTemplateCols cleanup, sticky column styling, and delete button guard

These are the three simplest changes — straightforward find-and-replace edits with no behavioral complexity.

**Files:**
- Modify: `frontend/src/lib/components/ResourceList.svelte:238-246` (gridTemplateCols)
- Modify: `frontend/src/lib/components/ResourceList.svelte:372` (header sticky class)
- Modify: `frontend/src/lib/components/ResourceList.svelte:455` (body sticky class)
- Modify: `frontend/src/lib/components/ResourceList.svelte:526` (delete button)
- Modify: `frontend/src/lib/__tests__/ResourceList.svelte.test.ts:113-131` (update sticky test)

- [ ] **Step 1: Replace gridTemplateCols with array-based builder**

Replace lines 238-246:

```ts
const gridTemplateCols = $derived.by(() => {
  const parts: string[] = []
  if (canMutate) parts.push('36px')
  for (const c of columnStore.visibleColumns) {
    parts.push(c.width ? `${c.width}px` : 'minmax(20px, 1fr)')
  }
  for (const _ of pluginColumns) parts.push('1fr')
  for (const _ of sparklineColumns) parts.push('80px')
  parts.push('36px')
  return parts.join(' ')
})
```

- [ ] **Step 2: Fix sticky column classes in header and body**

In the header button (line 372), replace:
```
'sticky left-0 z-10 bg-bg shadow-[2px_0_4px_rgba(0,0,0,0.08)] dark:shadow-[2px_0_4px_rgba(0,0,0,0.3)]'
```
with:
```
'sticky left-0 z-10 bg-bg border-r border-border'
```

In the body cell (line 455), replace the same shadow pattern:
```
'sticky left-0 z-10 bg-bg shadow-[2px_0_4px_rgba(0,0,0,0.08)] dark:shadow-[2px_0_4px_rgba(0,0,0,0.3)]'
```
with:
```
'sticky left-0 z-10 bg-bg border-r border-border'
```

- [ ] **Step 3: Guard delete button with canMutate**

On line 526, change `{:else}` to `{:else if canMutate}`.

- [ ] **Step 4: Update existing sticky column test**

In `ResourceList.svelte.test.ts`, the test at line 113 checks for `sticky` and `left-0`. It still passes since those classes remain. Optionally verify the shadow is gone and border is present:

```ts
expect(first.className).not.toContain('shadow')
```

- [ ] **Step 5: Run tests and commit**

```bash
cd frontend && npx vitest run src/lib/__tests__/ResourceList.svelte.test.ts
```

Expected: all 3 tests pass.

Commit message: `fix(resourcelist): clean up gridTemplateCols, fix sticky column styling, guard delete button`

---

### Task 2: Column resize jump fix and horizontal scrolling

These are coupled — horizontal scrolling requires moving the header into the scroll container, which changes the DOM structure around the resize handles.

**Files:**
- Modify: `frontend/src/lib/components/ResourceList.svelte:248-253` (startResize function)
- Modify: `frontend/src/lib/components/ResourceList.svelte:278` (outer container)
- Modify: `frontend/src/lib/components/ResourceList.svelte:342-398` (header row — move into scroll container)
- Modify: `frontend/src/lib/components/ResourceList.svelte:400` (scroll container class)

- [ ] **Step 1: Fix startResize to measure real DOM width**

Replace the `startResize` function (lines 248-253):

```ts
function startResize(e: MouseEvent, col: ColumnDef) {
  e.preventDefault()
  const cell = (e.currentTarget as HTMLElement).parentElement
  const measuredWidth = cell ? cell.getBoundingClientRect().width : (col.width ?? 100)
  resizing = { name: col.name, startX: e.clientX, startWidth: measuredWidth }
  window.addEventListener('mousemove', onResizeMove)
  window.addEventListener('mouseup', onResizeUp, { once: true })
}
```

- [ ] **Step 2: Move header into scroll container and enable horizontal scroll**

This is a structural DOM change. The goal is:
1. The scroll container gets `overflow-auto` (was `overflow-y-auto`)
2. The header grid row moves inside the scroll container, before the loading/empty/virtualized content
3. The header gets `sticky top-0 z-20 bg-bg` so it pins vertically but scrolls horizontally with the body
4. Remove `shrink-0` from the header since it's now inside the scrollable area (sticky handles pinning)

The resulting structure inside the `{:else}` (after the error check) becomes:

```svelte
<div bind:this={scrollContainer} class="flex-1 overflow-auto">
  <!-- Header row (was outside scroll container) -->
  <div class="grid text-xs font-semibold uppercase tracking-wider text-muted border-b border-border sticky top-0 z-20 bg-bg px-2"
    style="grid-template-columns: {gridTemplateCols}"
  >
    <!-- ...existing header content unchanged... -->
  </div>

  {#if loading}
    <!-- ...existing loading/empty/virtualized content unchanged... -->
```

Remove the old header div that was between `{:else}` (line 341) and the scroll container div (line 400). The header content itself (select-all checkbox, column headers with sort buttons and resize handles, plugin/sparkline headers, empty action column) stays identical.

- [ ] **Step 3: Run tests and verify**

```bash
cd frontend && npx vitest run src/lib/__tests__/ResourceList.svelte.test.ts
```

Expected: all tests pass. The test queries `container.querySelectorAll('.grid button')` which still matches the header buttons inside the scroll container.

Manual verification (with `task dev`):
- Resize a column that has no explicit width — should start from its actual rendered size, no jump
- Resize a column wider than the viewport — horizontal scrollbar should appear
- Header should scroll horizontally in sync with body
- Vertical scroll should keep header pinned at top

Commit message: `fix(resourcelist): fix column resize jump, enable horizontal scrolling`

---

### Task 3: Smart sorting and context menu clamping

**Files:**
- Modify: `frontend/src/lib/components/ResourceList.svelte:128-146` (filtered sort block)
- Modify: `frontend/src/lib/components/ResourceList.svelte:78` (add ctxMenuEl ref)
- Modify: `frontend/src/lib/components/ResourceList.svelte:546-571` (context menu template)
- Add to: `frontend/src/lib/__tests__/ResourceList.svelte.test.ts` (sort comparator test)

- [ ] **Step 1: Replace sort comparator with smart sorting**

Replace the sort block inside `filtered` (lines 134-144):

```ts
if (columnStore.sortState) {
  const { column, direction } = columnStore.sortState
  const col = columnStore.visibleColumns.find((c) => c.name === column)
  if (col?.expr) {
    result = [...result].sort((a, b) => {
      const rawA = evalExpr(col.expr, a)
      const rawB = evalExpr(col.expr, b)
      const av = String(rawA ?? '')
      const bv = String(rawB ?? '')
      let cmp: number
      if (col.renderType === 'age') {
        cmp = av.localeCompare(bv)
      } else {
        const an = parseFloat(av)
        const bn = parseFloat(bv)
        cmp = Number.isFinite(an) && Number.isFinite(bn)
          ? an - bn
          : av.localeCompare(bv)
      }
      return direction === 'asc' ? cmp : -cmp
    })
  }
}
```

Key behaviors:
- `age` columns: ISO timestamps sort correctly with `localeCompare` (lexicographic order matches chronological for ISO 8601)
- Numeric values: `parseFloat` on both sides, numeric compare if both parse
- Mixed/string values: falls back to `localeCompare`
- Direction is applied by negating the comparison result

- [ ] **Step 2: Add context menu viewport clamping**

Add a new state variable near line 78:

```ts
let ctxMenuEl = $state<HTMLDivElement | null>(null)
```

Add a new `$effect` after the existing `ctxMenu` close effect (after line 104):

```ts
$effect(() => {
  if (!ctxMenu || !ctxMenuEl) return
  const rect = ctxMenuEl.getBoundingClientRect()
  const maxX = window.innerWidth - rect.width - 8
  const maxY = window.innerHeight - rect.height - 8
  if (ctxMenu.x > maxX || ctxMenu.y > maxY) {
    ctxMenu = {
      ...ctxMenu,
      x: Math.max(0, Math.min(ctxMenu.x, maxX)),
      y: Math.max(0, Math.min(ctxMenu.y, maxY)),
    }
  }
})
```

Add `bind:this={ctxMenuEl}` to the context menu div (line 548):

```svelte
<div
  bind:this={ctxMenuEl}
  class="fixed z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-36"
  style="left:{ctxMenu.x}px; top:{ctxMenu.y}px"
  ...
```

- [ ] **Step 3: Add sort comparator test**

Add to the test file:

```ts
it('sorts numeric values numerically', async () => {
  const restartsCol: ColumnDef = { name: 'Restarts', expr: 'status.restartCount', renderType: 'text', width: 80 }
  mockVisibleColumns.value = [textCol, restartsCol]
  mockSortState.value = { column: 'Restarts', direction: 'asc' }

  const items = [
    { metadata: { name: 'pod-a' }, spec: {}, status: { restartCount: 10 } },
    { metadata: { name: 'pod-b' }, spec: {}, status: { restartCount: 2 } },
    { metadata: { name: 'pod-c' }, spec: {}, status: { restartCount: 1 } },
  ]

  const { container } = render(ResourceList, {
    props: { items, contextName: 'test-ctx', gvr: 'core.v1.pods' },
  })

  const nameCells = container.querySelectorAll('[data-col="Name"] span')
  const names = Array.from(nameCells).map((el) => el.textContent)
  expect(names).toEqual(['pod-c', 'pod-b', 'pod-a'])
})
```

- [ ] **Step 4: Run all tests and commit**

```bash
cd frontend && npx vitest run src/lib/__tests__/ResourceList.svelte.test.ts
```

Expected: all tests pass including the new sort test.

Commit message: `fix(resourcelist): add smart numeric sorting, clamp context menu to viewport`
