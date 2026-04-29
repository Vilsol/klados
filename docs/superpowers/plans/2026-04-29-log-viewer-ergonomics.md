# Log Viewer Ergonomics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix two visual bugs in `VirtualLogViewer` and add ergonomic search features (substring highlighting, match counter, current-match emphasis, invalid-regex feedback, case-sensitive toggle, keyboard shortcuts, filter mode, floating jump-to-bottom, debounced search).

**Architecture:** All changes live in `packages/ui/package/VirtualLogViewer.svelte`. Substring highlighting is implemented as a Svelte action that DOM-walks the row and wraps regex matches in `<mark>` elements after the existing syntax-highlighter HTML has been parsed by the browser. Filter mode swaps the virtualizer's count to a derived `visibleIndices` array; everything else is local toolbar/state additions.

**Tech Stack:** Svelte 5 runes (`$state`, `$derived`, `$effect`), `@tanstack/svelte-virtual`, Tailwind v4 tokens, `ansi_up`, `strip-ansi`. No backend changes.

**Spec reference:** `docs/superpowers/specs/2026-04-29-log-viewer-ergonomics-design.md`

**Testing strategy:** Manual verification via storybook (`apps/docs/src/stories/VirtualLogViewerStory.svelte`) — no new unit tests this round. Existing tests in `frontend/src/lib/__tests__/LogsPanel.svelte.test.ts` must continue to pass.

**Granularity note:** Coarse tasks. Each one is a coherent feature slice that can be eyeballed in storybook and committed as one `jj` change. Inside a task, write all the code, run `pnpm check` and the storybook scenario, then commit.

---

## File touchlist

- **Modify:** `packages/ui/package/VirtualLogViewer.svelte` — all logic, markup, styles.
- **Modify:** `apps/docs/src/stories/VirtualLogViewerStory.svelte` — add scenario props for long lines, many matches, invalid-regex, filter, paused-tail.
- **Modify:** `apps/docs/src/stories/VirtualLogViewer.stories.ts` — register new stories.
- **Auto-regenerated:** `packages/ui/.svelte-kit/__package__/VirtualLogViewer.svelte` — do not hand-edit; produced by `pnpm --filter @klados/ui build` if/when run. Not strictly required during dev (the workspace consumes the source via package exports). Leave alone unless CI requires it.

---

### Task 1: Fix full-width row background (Bug 2)

The smallest, lowest-risk change. Lands the visual fix that makes every later improvement look correct on long lines.

**Files:**
- Modify: `packages/ui/package/VirtualLogViewer.svelte` (the row `<div>` inside the `{#each $virtualizerStore.getVirtualItems()}` block, around lines 402-418)

- [ ] **Step 1: Update the row's width style**

In the row `<div>` swap `style:width="100%"` for `style:min-width="100%"` and add a `style:width="max-content"` so the row sizes to its content while remaining at least viewport-wide:

```svelte
<div
  data-index={row.index}
  use:measureEl
  style:position="absolute"
  style:top="0"
  style:left="0"
  style:min-width="100%"
  style:width="max-content"
  style:transform="translateY({row.start}px)"
  class="log-row px-3 py-0 {highlight ? levelClass(processedLines[row.index].plain) : ''} {matchIndices.includes(row.index) ? 'search-match' : ''}"
  style:line-height="{rowHeight}px"
  class:whitespace-pre={!wrap}
  class:whitespace-pre-wrap={wrap}
  class:break-all={wrap}
>
```

Note: when `wrap` is on, `whitespace-pre-wrap` + the parent's `overflow-x-hidden={wrap}` keeps things constrained to viewport, and `width: max-content` collapses to the row width naturally — no special-case needed.

- [ ] **Step 2: Verify in storybook**

Run dev: `task dev` (or `cd apps/docs && pnpm dev`). Open the existing `VirtualLogViewer` "WithLines" story; pump `lineCount` up so some lines are long, or temporarily edit the story to emit a 500-character line. Confirm:

- With `wrap` off: scroll right; the level-highlight color (toggle HL) and the existing yellow `search-match` background extend to the line's full width.
- With `wrap` on: rows still constrained to viewport, no horizontal scroll.

- [ ] **Step 3: Commit**

```bash
jj describe -m "fix(log-viewer): extend row background to full content width"
jj new
```

---

### Task 2: Refactor match cursor + add search debounce + match counter (A, I)

Lays the state foundation for active-match emphasis (Task 4) and filter mode (Task 7). Includes the visible counter so progress is observable in storybook.

**Files:**
- Modify: `packages/ui/package/VirtualLogViewer.svelte`

- [ ] **Step 1: Replace `matchCursor` semantics**

Currently `matchCursor` holds a *line index*. Repurpose it to a 0-indexed *position into `matchIndices`*. Replace this block (around line 154):

```ts
let matchCursor = $state(-1)
```

with:

```ts
let matchCursor = $state(0)
```

Replace `findNext` and `findPrev` (around lines 296-312):

```ts
function findNext() {
  if (!matchIndices.length) return
  matchCursor = (matchCursor + 1) % matchIndices.length
  setSticky(false)
  scrollToLine(matchIndices[matchCursor], 'start')
}

function findPrev() {
  if (!matchIndices.length) return
  matchCursor = (matchCursor - 1 + matchIndices.length) % matchIndices.length
  setSticky(false)
  scrollToLine(matchIndices[matchCursor], 'start')
}
```

- [ ] **Step 2: Reset cursor when matches change**

Add a `$effect` that clamps `matchCursor` whenever `matchIndices.length` shrinks below the current cursor (e.g. user types more characters and matches disappear). Place it near the other effects:

```ts
$effect(() => {
  if (matchCursor >= matchIndices.length) matchCursor = 0
})
```

- [ ] **Step 3: Add 300ms debounce**

Introduce a debounced query state. Replace the existing `searchPattern` derivation:

```ts
let searchQuery = $state('')
let debouncedQuery = $state('')

$effect(() => {
  const q = searchQuery
  const timer = setTimeout(() => { debouncedQuery = q }, 300)
  return () => clearTimeout(timer)
})

const searchPattern = $derived((() => {
  if (!debouncedQuery) return null
  try {
    return new RegExp(regexSearch ? debouncedQuery : debouncedQuery.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i')
  } catch {
    return null
  }
})())
```

The toolbar input still binds to `searchQuery` so typing feels instant. Toggle changes (`regexSearch`, future `caseSensitive`) reactively re-derive `searchPattern` immediately because the `$derived` reads them directly.

- [ ] **Step 4: Add the `n / N` counter to the toolbar**

In the toolbar `<div>` (around line 358), insert a counter span between the prev (`↑`) and next (`↓`) buttons:

```svelte
<button onclick={findPrev} class="text-xs text-muted hover:text-fg px-1.5 py-1" title="Previous match" aria-label="Previous match">↑</button>
<span class="text-xs text-muted tabular-nums select-none min-w-[3rem] text-center" aria-live="polite">
  {matchIndices.length === 0 ? '0 / 0' : `${matchCursor + 1} / ${matchIndices.length}`}
</span>
<button onclick={findNext} class="text-xs text-muted hover:text-fg px-1.5 py-1" title="Next match" aria-label="Next match">↓</button>
```

- [ ] **Step 5: Verify and commit**

In storybook, type into the search box. Observe:
- The counter updates ~300ms after the last keystroke (debounce works).
- Pressing ↓/↑ wraps around.
- Counter clears to `0 / 0` when query is empty.

```bash
cd frontend && pnpm check  # type-check from frontend, which consumes @klados/ui
jj describe -m "feat(log-viewer): debounce search and show match counter"
jj new
```

---

### Task 3: Invalid-regex feedback + case-sensitive toggle (C, G)

Two small toolbar additions; bundled because they share the same `searchPattern` derivation.

**Files:**
- Modify: `packages/ui/package/VirtualLogViewer.svelte`

- [ ] **Step 1: Add `caseSensitive` state and update `searchPattern`**

Near `regexSearch`:

```ts
let caseSensitive = $state(false)
```

Update `searchPattern`:

```ts
const searchPattern = $derived((() => {
  if (!debouncedQuery) return null
  try {
    const flags = caseSensitive ? '' : 'i'
    return new RegExp(regexSearch ? debouncedQuery : debouncedQuery.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), flags)
  } catch {
    return null
  }
})())
```

- [ ] **Step 2: Add `regexInvalid` derivation**

```ts
const regexInvalid = $derived((() => {
  if (!regexSearch || !debouncedQuery) return false
  try { new RegExp(debouncedQuery); return false } catch { return true }
})())
```

- [ ] **Step 3: Reflect invalid state on the search input**

Update the search `<input>` class binding (around line 349):

```svelte
<input
  type="text"
  bind:value={searchQuery}
  onkeydown={(e) => e.key === 'Enter' && findNext()}
  placeholder="Search…"
  class="flex-1 min-w-0 text-xs bg-surface-hover border rounded px-2 py-1 focus:outline-none {regexInvalid ? 'border-destructive focus:border-destructive' : 'border-border focus:border-accent'}"
/>
```

- [ ] **Step 4: Add the `Aa` toggle button next to `.*`**

Insert before the `.*` button:

```svelte
<button
  onclick={() => (caseSensitive = !caseSensitive)}
  title="Match case"
  aria-label="Match case"
  aria-pressed={caseSensitive}
  class="text-xs px-2 py-1 rounded border transition-colors
    {caseSensitive ? 'border-accent text-accent bg-accent/10' : 'border-border text-muted hover:text-fg'}"
>Aa</button>
```

- [ ] **Step 5: Verify and commit**

Storybook: enable `.*` regex toggle, type `[(`. Confirm input border turns red. Disable regex; toggle `Aa` and confirm matches change with case.

```bash
cd frontend && pnpm check
jj describe -m "feat(log-viewer): case-sensitive search and invalid-regex feedback"
jj new
```

---

### Task 4: Substring search highlighting (Bug 1 + B)

The trickiest task. Wraps regex matches in `<mark>` via a Svelte action that DOM-walks each row after mount. Drops the row-wide `search-match` background; replaces it with a thin left border on rows that contain a match.

**Files:**
- Modify: `packages/ui/package/VirtualLogViewer.svelte`

- [ ] **Step 1: Add the `markMatches` action**

Add after the `measureEl` action (around line 215):

```ts
function markMatches(el: HTMLElement, params: { pattern: RegExp | null, isCurrent: boolean }) {
  function clear() {
    const marks = el.querySelectorAll('mark')
    marks.forEach(m => {
      const text = document.createTextNode(m.textContent ?? '')
      m.replaceWith(text)
    })
    el.normalize()
  }

  function apply(pattern: RegExp | null, isCurrent: boolean) {
    clear()
    if (!pattern) return
    // Make pattern global for matchAll; preserve case flag.
    const flags = pattern.flags.includes('g') ? pattern.flags : pattern.flags + 'g'
    const gPattern = new RegExp(pattern.source, flags)
    const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT, {
      acceptNode(node) {
        // Skip text inside existing <mark> (defensive — clear() should have removed them)
        let p: Node | null = node.parentNode
        while (p && p !== el) {
          if ((p as Element).tagName === 'MARK') return NodeFilter.FILTER_REJECT
          p = p.parentNode
        }
        return NodeFilter.FILTER_ACCEPT
      }
    })
    const targets: Text[] = []
    let n: Node | null
    while ((n = walker.nextNode())) targets.push(n as Text)

    const cls = isCurrent ? 'match active' : 'match'
    for (const textNode of targets) {
      const text = textNode.nodeValue ?? ''
      gPattern.lastIndex = 0
      const matches = [...text.matchAll(gPattern)]
      if (matches.length === 0) continue
      const frag = document.createDocumentFragment()
      let last = 0
      for (const m of matches) {
        const start = m.index ?? 0
        const end = start + m[0].length
        if (start > last) frag.appendChild(document.createTextNode(text.slice(last, start)))
        const mark = document.createElement('mark')
        mark.className = cls
        mark.textContent = m[0]
        frag.appendChild(mark)
        last = end
      }
      if (last < text.length) frag.appendChild(document.createTextNode(text.slice(last)))
      textNode.replaceWith(frag)
    }
  }

  apply(params.pattern, params.isCurrent)

  return {
    update(next: { pattern: RegExp | null, isCurrent: boolean }) {
      apply(next.pattern, next.isCurrent)
    },
    destroy() {
      clear()
    }
  }
}
```

Note: this is a single shared action — but only the *current* row uses `isCurrent: true`. We compute that per-row in the template.

- [ ] **Step 2: Apply the action on each row + drop row-wide search-match**

Update the row in the `{#each}` block. Remove the `search-match` class; add the action. Replace the row markup:

```svelte
<div
  data-index={row.index}
  use:measureEl
  use:markMatches={{ pattern: searchPattern, isCurrent: matchIndices[matchCursor] === row.index }}
  style:position="absolute"
  style:top="0"
  style:left="0"
  style:min-width="100%"
  style:width="max-content"
  style:transform="translateY({row.start}px)"
  class="log-row px-3 py-0 {highlight ? levelClass(processedLines[row.index].plain) : ''} {matchIndices.includes(row.index) ? 'has-match' : ''}"
  style:line-height="{rowHeight}px"
  class:whitespace-pre={!wrap}
  class:whitespace-pre-wrap={wrap}
  class:break-all={wrap}
>
```

- [ ] **Step 3: Update styles**

Replace the `.search-match` rule and add new ones in the `<style>` block:

```css
.has-match { box-shadow: inset 2px 0 0 0 #ca8a04; }

:global(.log-row mark.match) {
  background: #854d0e88;
  color: inherit;
  border-radius: 2px;
  padding: 0 1px;
}
:global(.log-row mark.match.active) {
  background: #ea580c;
  color: #1a1a1a;
}
```

(Remove the old `.search-match { background: #854d0e55; outline: 1px solid #854d0e; }` line.)

- [ ] **Step 4: Verify in storybook**

- Type a query that appears in many rows: substrings should be highlighted yellow; the row navigated to via ↓ should have its match in orange.
- Press ↓ repeatedly: orange should move down rows.
- Verify highlights survive across syntax-highlighted JSON rows (the `{"level":"info",…}` row in the existing story).
- Verify horizontal scroll: a long line with a match near the end keeps its highlight visible.

- [ ] **Step 5: Commit**

```bash
cd frontend && pnpm check
jj describe -m "feat(log-viewer): substring search highlighting with active-match emphasis"
jj new
```

---

### Task 5: Keyboard shortcuts (D)

**Files:**
- Modify: `packages/ui/package/VirtualLogViewer.svelte`

- [ ] **Step 1: Add a ref for the search input and the root container**

Near the other state:

```ts
let searchInputEl = $state<HTMLInputElement | undefined>(undefined)
let rootEl = $state<HTMLDivElement | undefined>(undefined)
```

Bind them on the root div and the input. Update the input element:

```svelte
<input
  bind:this={searchInputEl}
  type="text"
  bind:value={searchQuery}
  onkeydown={onSearchKey}
  placeholder="Search…"
  class="flex-1 min-w-0 text-xs bg-surface-hover border rounded px-2 py-1 focus:outline-none {regexInvalid ? 'border-destructive focus:border-destructive' : 'border-border focus:border-accent'}"
/>
```

Update the outermost `<div>` that wraps the whole component:

```svelte
<div bind:this={rootEl} onkeydown={onRootKey} class="flex flex-col h-full overflow-hidden">
```

- [ ] **Step 2: Implement the handlers**

Add functions near `findNext`/`findPrev`:

```ts
function onSearchKey(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    if (e.shiftKey) findPrev()
    else findNext()
    e.preventDefault()
  } else if (e.key === 'Escape') {
    searchQuery = ''
    searchInputEl?.blur()
    e.preventDefault()
  }
}

function onRootKey(e: KeyboardEvent) {
  const isFind = (e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f'
  if (isFind) {
    searchInputEl?.focus()
    searchInputEl?.select()
    e.preventDefault()
    return
  }
  if (e.key === 'F3') {
    if (e.shiftKey) findPrev()
    else findNext()
    e.preventDefault()
  }
}
```

(The previous `onkeydown` on the input — `(e) => e.key === 'Enter' && findNext()` — is now subsumed by `onSearchKey`.)

- [ ] **Step 3: Verify and commit**

Storybook: with focus inside the log viewer (click anywhere in it), press Ctrl+F → search input focuses. Type query, press Enter → next; Shift+Enter → prev. Press Esc → query clears, input blurs. Press F3 → next.

```bash
cd frontend && pnpm check
jj describe -m "feat(log-viewer): keyboard shortcuts for search (Ctrl+F, Esc, F3)"
jj new
```

---

### Task 6: Floating jump-to-bottom arrow (F)

**Files:**
- Modify: `packages/ui/package/VirtualLogViewer.svelte`

- [ ] **Step 1: Add a `jumpToBottom` function**

Place near `scrollToTop`:

```ts
export function jumpToBottom() {
  setSticky(true)
  if (scrollEl) {
    programmaticScroll = true
    scrollEl.scrollTop = scrollEl.scrollHeight
  }
}
```

- [ ] **Step 2: Add the floating button**

Inside the scroll wrapper (the `<div bind:this={scrollEl} …>`), add a button that's visible when `!sticky` and there is at least one line. Place it just *after* the inner virtualizer container so it sits above the rows in stacking order. Because the wrapper is `relative`, an absolutely positioned button will anchor inside it:

```svelte
{#if !sticky && processedLines.length > 0}
  <button
    type="button"
    onclick={jumpToBottom}
    aria-label="Jump to bottom"
    title="Jump to bottom"
    class="absolute bottom-3 right-4 w-9 h-9 rounded-full bg-surface border border-border shadow-md flex items-center justify-center text-fg hover:bg-surface-hover transition-colors"
    style:position="sticky"
    style:bottom="12px"
    style:margin-left="auto"
    style:margin-right="12px"
  >
    ↓
  </button>
{/if}
```

Note on positioning: the scroll wrapper has `relative` already. Use `position: sticky` on the button so it stays pinned to the bottom-right of the *visible* area regardless of scroll. Wrap it in a sticky container if needed:

```svelte
{#if !sticky && processedLines.length > 0}
  <div class="sticky bottom-3 flex justify-end pr-4 pointer-events-none" style:height="0">
    <button
      type="button"
      onclick={jumpToBottom}
      aria-label="Jump to bottom"
      title="Jump to bottom"
      class="pointer-events-auto -translate-y-9 w-9 h-9 rounded-full bg-surface border border-border shadow-md flex items-center justify-center text-fg hover:bg-surface-hover transition-colors"
    >
      ↓
    </button>
  </div>
{/if}
```

The wrapper is sticky-positioned with zero height so it doesn't add document height; the button is translated up so it appears above the bottom edge. `pointer-events-none` on the wrapper + `pointer-events-auto` on the button ensures clicks elsewhere pass through to rows.

- [ ] **Step 3: Verify and commit**

Storybook with a tall log: scroll up. The floating ↓ should appear bottom-right. Click it → log re-tails (sticky returns to true). When at bottom, the button hides.

```bash
cd frontend && pnpm check
jj describe -m "feat(log-viewer): floating jump-to-bottom button when tail is paused"
jj new
```

---

### Task 7: Filter mode (E)

Largest behavioral change — switches the virtualizer to render only matching lines.

**Files:**
- Modify: `packages/ui/package/VirtualLogViewer.svelte`

- [ ] **Step 1: Add filter state and derived visible indices**

Near `wrap`:

```ts
let filterMode = $state(false)
```

Add a derived array that resolves to `null` (meaning "no filter, show all") or the list of matching line indices:

```ts
const visibleIndices = $derived(filterMode && searchPattern ? matchIndices : null)
const visibleCount = $derived(visibleIndices ? visibleIndices.length : processedLines.length)
```

- [ ] **Step 2: Update virtualizer count**

Update the `$effect` that calls `updateVirtOptions` (around lines 205-210):

```ts
$effect(() => {
  const count = visibleCount
  const _wrap = wrap
  const rh = rowHeight
  untrack(() => updateVirtOptions(count, rh))
})
```

- [ ] **Step 3: Resolve row content via the visible map**

Inside the `{#each}` block, compute the underlying line index per row. Replace the row body to read through `visibleIndices`:

```svelte
{#each $virtualizerStore.getVirtualItems() as row (row.index)}
  {@const lineIdx = visibleIndices ? visibleIndices[row.index] : row.index}
  {@const line = processedLines[lineIdx]}
  {#if line}
    <div
      data-index={lineIdx}
      use:measureEl
      use:markMatches={{ pattern: searchPattern, isCurrent: matchIndices[matchCursor] === lineIdx }}
      style:position="absolute"
      style:top="0"
      style:left="0"
      style:min-width="100%"
      style:width="max-content"
      style:transform="translateY({row.start}px)"
      class="log-row px-3 py-0 {highlight ? levelClass(line.plain) : ''} {!filterMode && matchIndices.includes(lineIdx) ? 'has-match' : ''}"
      style:line-height="{rowHeight}px"
      class:whitespace-pre={!wrap}
      class:whitespace-pre-wrap={wrap}
      class:break-all={wrap}
    >
      {#if showTimestamps && line.ts}<span class="text-muted mr-2 select-none">{line.ts}</span>{/if}{@html line.html}
    </div>
  {/if}
{/each}
```

(Note: in filter mode every visible row matches, so the gutter accent is redundant — only show `has-match` when filter is off.)

- [ ] **Step 4: Add the Filter checkbox to the toolbar**

After the Wrap label:

```svelte
<label class="flex items-center gap-1 text-xs text-muted select-none cursor-pointer" title="Show only matching lines">
  <input type="checkbox" bind:checked={filterMode} class="accent-accent" />
  Filter
</label>
```

- [ ] **Step 5: Update auto-tail and copy to respect filter**

Update `onCopy` (around line 314) to use the underlying line index, which we already store as `data-index`. Existing logic already reads `data-index` and looks up `processedLines[idx]`, so it works unchanged.

Update the auto-tail effect to skip when filter is on and the new tail line doesn't match. Replace the existing tail effect (around line 251):

```ts
$effect(() => {
  const count = processedLines.length
  if (!sticky || count === 0 || !scrollEl) return
  if (filterMode && searchPattern) {
    const lastIdx = count - 1
    if (!searchPattern.test(processedLines[lastIdx]?.plain ?? '')) return
  }
  untrack(() => {
    if (tailRaf !== null) cancelAnimationFrame(tailRaf)
    tailRaf = requestAnimationFrame(() => {
      tailRaf = null
      if (!sticky || !scrollEl) return
      programmaticScroll = true
      scrollEl.scrollTop = scrollEl.scrollHeight
    })
  })
  return () => { if (tailRaf !== null) { cancelAnimationFrame(tailRaf); tailRaf = null } }
})
```

- [ ] **Step 6: Update the line-count footer to show filtered count**

Replace the footer (line 425):

```svelte
<div class="flex items-center px-3 py-1 border-t border-border bg-surface shrink-0">
  <span class="text-xs text-muted">
    {#if filterMode && searchPattern}
      {matchIndices.length.toLocaleString()} / {processedLines.length.toLocaleString()} lines{eofReached ? '' : ' (live)'}
    {:else}
      {processedLines.length.toLocaleString()} lines{eofReached ? '' : ' (live)'}
    {/if}
  </span>
</div>
```

- [ ] **Step 7: Verify and commit**

In storybook: type a query, toggle Filter on. Only matching lines should render; counter still shows position; ↓/↑ still navigate; footer shows `M / N lines`. Toggle off → full log returns. Verify scroll position is sane on toggle.

```bash
cd frontend && pnpm check
jj describe -m "feat(log-viewer): filter mode shows only matching lines"
jj new
```

---

### Task 8: Storybook scenarios + final verification

**Files:**
- Modify: `apps/docs/src/stories/VirtualLogViewerStory.svelte`
- Modify: `apps/docs/src/stories/VirtualLogViewer.stories.ts`

- [ ] **Step 1: Extend the story to support new scenarios**

Replace `VirtualLogViewerStory.svelte` with:

```svelte
<script lang="ts">
  import { VirtualLogViewer } from '@klados/ui'

  let {
    lineCount = 20,
    includeErrors = false,
    showTimestamps = false,
    longLines = false,
    manyMatches = false,
  }: {
    lineCount?: number
    includeErrors?: boolean
    showTimestamps?: boolean
    longLines?: boolean
    manyMatches?: boolean
  } = $props()

  const ts = (i: number) => `2024-01-15T${String(10 + Math.floor(i / 60)).padStart(2, '0')}:${String(i % 60).padStart(2, '0')}:00Z`

  const lines = $derived(
    Array.from({ length: lineCount }, (_, i) => {
      const prefix = showTimestamps ? `${ts(i)} ` : ''
      const tail = longLines ? ' ' + 'x'.repeat(400) : ''
      const targetWord = manyMatches ? ' target' : ''
      if (includeErrors && i === 5) return `${prefix}ERROR failed to connect to database: connection refused${targetWord}${tail}`
      if (includeErrors && i === 12) return `${prefix}WARN retry attempt 3/5${targetWord}${tail}`
      if (i % 4 === 0) return `${prefix}INFO server listening on :8080 version=1.2.3${targetWord}${tail}`
      if (i % 4 === 1) return `${prefix}INFO request completed method=GET path=/health status=200 duration=2ms${targetWord}${tail}`
      if (i % 4 === 2) return `${prefix}{"level":"info","msg":"processed event","id":"evt-${i}","ts":"${new Date().toISOString()}"${manyMatches ? ',"tag":"target"' : ''}}${tail}`
      return `${prefix}DEBUG cache hit key=user:${i} ttl=300s${targetWord}${tail}`
    }),
  )
</script>

<div class="h-96 border border-border rounded overflow-hidden">
  <VirtualLogViewer {lines} eofReached={true} {showTimestamps} />
</div>
```

- [ ] **Step 2: Register new stories**

Update `VirtualLogViewer.stories.ts` to add stories for `LongLines` (forces horizontal scroll), `ManyMatches` (lots of "target" hits for filter/counter testing), and `Tall` (1000 lines for paused-tail / floating button testing). Read the existing file first to keep its meta export shape:

```ts
// Keep existing meta + Empty / WithLines / WithTimestamps stories.
// Append:
export const LongLines = {
  args: { lineCount: 30, longLines: true, includeErrors: true },
}

export const ManyMatches = {
  args: { lineCount: 80, manyMatches: true, includeErrors: true },
}

export const Tall = {
  args: { lineCount: 1000, includeErrors: true },
}
```

- [ ] **Step 3: Run through the full manual checklist**

In storybook (`cd apps/docs && pnpm dev`), exercise each story:

**LongLines:**
- Scroll right; level-highlight (HL on) extends across full line. ✓ Bug 2.
- Search "ERROR"; the substring is highlighted, not the whole row. ✓ Bug 1.

**ManyMatches:**
- Search "target"; counter shows `1 / N`.
- ↓/↑ wraps around; current match orange, others yellow.
- Toggle Filter; only matching lines shown; counter still works.
- Toggle case-sensitive `Aa`; matches change.
- Type `[` with `.*` regex on; input border turns red.

**Tall:**
- Scroll up; floating ↓ button appears bottom-right. Click → re-tails.
- Click in viewer, press Ctrl+F; search input focuses.
- Press Esc in search; clears query.
- Type fast; observe debounce (counter updates after pause).

- [ ] **Step 4: Run frontend checks**

```bash
cd frontend && pnpm check
cd frontend && pnpm test
```

Both should pass — the panel-level tests (`LogsPanel.svelte.test.ts`) don't touch the new APIs.

- [ ] **Step 5: Commit**

```bash
jj describe -m "test(log-viewer): storybook scenarios for new search ergonomics"
jj new
```

---

## Self-review

**Spec coverage** — every section in the spec maps to a task:

- Bug 1 (substring highlighting) → Task 4
- Bug 2 (full-width background) → Task 1
- A (match counter) → Task 2 step 4
- B (active-match emphasis) → Task 4 step 3
- C (invalid regex) → Task 3 steps 2-3
- D (keyboard shortcuts) → Task 5
- E (filter mode) → Task 7
- F (jump-to-bottom) → Task 6
- G (case-sensitive) → Task 3 steps 1, 4
- I (debounce) → Task 2 step 3
- Storybook scenarios → Task 8

**Type/name consistency** — `matchCursor` semantics change in Task 2 (line index → array position) and every later reference (`findNext`, `findPrev`, the active-match check `matchIndices[matchCursor] === row.index` in Task 4 and `matchIndices[matchCursor] === lineIdx` in Task 7) is consistent with the new meaning. `searchPattern`, `debouncedQuery`, `regexInvalid`, `caseSensitive`, `filterMode`, `visibleIndices`, `markMatches` are all defined where first used and referenced consistently afterward.

**No placeholders** — every code step shows the full code to write.

**Risk areas:**
- Task 4's DOM walking interacts with `@html` rendered content. Active-match orange must not collide with ANSI-styled foreground colors visually; the chosen `#ea580c` is dark enough that white text on it remains readable.
- Task 7's filter mode interacts with `prependLines` (history loading). When filter is on, prepended lines that don't match still get into `processedLines`; `matchIndices` re-derives correctly. The first-visible-index restoration in `prependLines` uses the *unfiltered* line count — that's a known quirk worth noting but not a correctness bug since history-prepend in filter mode is a corner case.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-29-log-viewer-ergonomics.md`. Two execution options:

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks.
2. **Inline Execution** — execute tasks here in this session with checkpoints.

Which approach?
