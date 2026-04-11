# ResourceList Improvements Design

**Date:** 2026-04-11
**Scope:** `frontend/src/lib/components/ResourceList.svelte`
**Issues:** 3 user-reported + 4 discovered during review

## 1. Sticky Column Styling

**Problem:** The first column uses `bg-bg` with a `shadow-[2px_0_4px_rgba(0,0,0,0.08)]` to visually separate it as a sticky column. The shadow renders smaller than the cell height, creating a visible gap that looks broken.

**Fix:** Replace `bg-bg shadow-[...]` with `bg-bg border-r border-border` on both the header cell and body cell for column index 0. The sticky positioning (`sticky left-0 z-10`) remains for horizontal scroll pinning.

**Locations:** Two places in ResourceList.svelte — the header `{#each columnStore.visibleColumns}` block and the body `{#each columnStore.visibleColumns}` block, both checking `i === 0`.

## 2. Column Resize Jump & Horizontal Overflow

**Problem (jump):** `startResize` uses `col.width ?? 100` as the drag baseline. Columns without an explicit width render at their natural CSS size (often much wider than 100px), so the first drag frame snaps the column to ~100px plus the mouse delta.

**Fix (jump):** On `mousedown`, read `getBoundingClientRect().width` from the header cell DOM element (`e.currentTarget.parentElement`) and use that as `startWidth`. This always starts from the real rendered width regardless of whether CSS used `fr`, `auto`, or a stored pixel value.

**Problem (overflow):** The grid parent has `overflow-hidden`, so columns can never extend past the viewport. Users cannot see or access columns that would require horizontal scrolling.

**Fix (overflow):** Change the scroll container from `overflow-y-auto` to `overflow-auto`. Move the header grid row inside the scroll container with `sticky top-0 z-20 bg-bg` so it stays pinned vertically but scrolls horizontally in sync with the body rows. Grid rows use `min-w-max` or `width: max-content` so they overflow naturally when columns have explicit pixel widths.

## 3. Number Sorting

**Problem:** All column sorting goes through `String(evalExpr(...))` with `localeCompare`, so numeric columns like Restarts, Available, and Replicas sort lexicographically ("10" < "2").

**Fix:** Replace the sort comparator in the `filtered` derived block:

- For `age` renderType: compare raw ISO timestamp strings (ISO 8601 sorts lexicographically correctly).
- For all other renderTypes: try `parseFloat()` on both values. If both are finite numbers, compare numerically (`an - bn`). Otherwise fall back to `localeCompare`.

No schema changes needed. This infers sort behavior from the actual data values.

## 4. Context Menu Viewport Clamping

**Problem:** The right-click context menu renders at raw `clientX`/`clientY`. Near screen edges, it overflows off-screen.

**Fix:** Add a `bind:this` ref to the context menu div. After the menu mounts (via `$effect` when `ctxMenu` is set), read its `getBoundingClientRect()` and clamp:

- `left = Math.min(ctxMenu.x, window.innerWidth - menuWidth - 8)`
- `top = Math.min(ctxMenu.y, window.innerHeight - menuHeight - 8)`

## 5. gridTemplateCols Cleanup

**Problem:** The `gridTemplateCols` derived builds its value via string concatenation with conditional prefixes/suffixes. Fragile and hard to read.

**Fix:** Build as an array, then `.join(' ')`:

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

## 6. Delete Button canMutate Guard

**Problem:** The fallback delete trash icon (when `rowActions` is not provided) shows on every row regardless of whether the cluster connection is read-only. The checkbox column is already gated behind `canMutate`, but the delete button is not.

**Fix:** Change `{:else}` to `{:else if canMutate}` in the row actions block. When `rowActions` is provided, the caller is responsible for their own guards.

## Files Changed

Only `frontend/src/lib/components/ResourceList.svelte` is modified. No Go backend changes, no new dependencies, no binding regeneration needed.
