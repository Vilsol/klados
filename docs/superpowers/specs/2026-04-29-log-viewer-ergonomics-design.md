# Log Viewer Ergonomics

**Date:** 2026-04-29
**Component:** `packages/ui/package/VirtualLogViewer.svelte`
**Scope:** Search quality bugs + ergonomic improvements. No backend changes.

## Motivation

Two bugs and several gaps in the log viewer make searching and navigating long logs awkward:

1. The search "highlight" colors entire rows yellow rather than the matching substrings, making it hard to see *what* matched.
2. Row backgrounds (search match, level highlight) only span the parent container width, so when a line is long enough to require horizontal scrolling, the background visibly cuts off mid-line.

Beyond the bugs, search has no match counter, no current-vs-other-match distinction, no feedback on invalid regex, no keyboard shortcuts, no filter mode, no case-sensitivity toggle, and no debounce. There is also no quick way to re-tail after scrolling up.

## Goals

- Substring-level search highlighting that survives the existing syntax highlighters (JSON / logfmt / klog / CLF).
- Row backgrounds extend across the full content width, regardless of horizontal scroll position.
- Search feels like a "real" find UI: counter, current-match emphasis, regex validity feedback, keyboard shortcuts, case-sensitive option.
- Filter mode to hide non-matching lines on demand.
- Floating "jump to bottom" affordance when auto-tail is paused.
- Search input is debounced (300ms) so typing on large logs doesn't stutter.

## Non-goals

- Improving level-color detection (`levelClass`) — out of scope.
- Server-side / streaming search.
- Unit tests for new logic — covered by storybook scenarios this round.

## Design

All work is in `packages/ui/package/VirtualLogViewer.svelte`. The package is consumed by the frontend's logs panels (`LogsPanel.svelte`, `AggregateLogsPanel.svelte`); no consumer changes expected.

### Bug 1 — Substring search highlighting

The renderers (`highlightJSON`, `highlightLogfmt`, `highlightKlog`, `highlightCLF`) emit HTML with `<span class="hl-…">` wrappers. We do not modify those renderers. Instead, after a row mounts (or when its underlying line index / search pattern changes), a Svelte action walks the row's text nodes and wraps regex matches in `<mark>` elements.

Approach details:

- New action `markMatches(el, params)` invoked alongside the existing `measureEl` action on each row.
- Params: `{ pattern: RegExp | null, isCurrent: boolean }`. Params change reactively; the action re-runs on update.
- On each run: remove any existing `<mark>` descendants (replace with their text-node content, normalize), then if `pattern` is set, walk text nodes via `TreeWalker` and wrap matches.
- The active match (the one `matchCursor` points at) gets `<mark class="match active">`; others get `<mark class="match">`.
- The row no longer gets a row-wide `search-match` background. Instead, rows that contain at least one match get a thin left border accent (CSS class `has-match`) — a low-noise indicator in the gutter while the actual match is painted on the substring.

Performance: only ~30 rows are mounted at any time (virtualized), so the DOM walk runs over a small DOM each search update.

### Bug 2 — Full-width row background

Today each virtualized row is `width: 100%`, which equals the scroll viewport width, not the content width. When the user scrolls right, the row background (level highlight, has-match border, etc.) does not extend.

Fix: set `min-width: max-content` on the row's inner element so the row sizes to its content. The absolute-positioned wrapper provided by the virtualizer needs to allow content overflow; `width: 100%` is replaced by `min-width: 100%` to keep the row at least as wide as the viewport (so empty rows still look right).

### A — Match counter

Display `{matchCursor + 1} / {matchIndices.length}` between the previous and next buttons in the toolbar. When there are no matches, render `0 / 0` in muted color.

`matchCursor` is repurposed to be a 0-indexed position into `matchIndices` (the array of matching line indices), not a line index. `findNext` / `findPrev` increment / decrement this cursor with wrap-around. The line index used for `scrollToLine` becomes `matchIndices[matchCursor]`.

### B — Active-match emphasis

Covered by the substring highlighting design — the active match gets a distinct color via `<mark class="match active">`. Suggested colors: yellow translucent for non-active, orange for active. Existing scroll-into-view behavior in `findNext` / `findPrev` already brings the active match into view.

### C — Invalid regex feedback

`searchPattern` already returns `null` when `new RegExp(...)` throws. Add a separate `$derived` signal `regexInvalid` that is `true` when `regexSearch` is on, the query is non-empty, and the construction would throw. Use that to apply a red border on the search input.

### D — Keyboard shortcuts

Component-scoped, attached to the root container via `keydown`:

- `Ctrl`/`Cmd` + `F` — focus the search input.
- `Enter` (already implemented for next) — keep.
- `Shift` + `Enter` — previous.
- `Escape` while search input is focused — clear query and blur.
- `F3` — next; `Shift` + `F3` — previous.

Bindings live on the root `div` so they activate when the log viewer (or any descendant) has focus, avoiding global hijacking. The `Find` shortcut uses `event.preventDefault()` to suppress the WebView's default find dialog.

### E — Filter mode

New `Filter` checkbox in the toolbar, state `filterMode`. When on:

- A new derived `visibleLines: ProcessedLine[]` is `processedLines.filter((_, i) => matchIndices.includes(i))`. For efficiency, derive a `Set` of match indices and a `visibleIndices: number[]` once.
- The virtualizer count switches to `visibleIndices.length`.
- Each rendered row resolves its content via `processedLines[visibleIndices[row.index]]` so the line content and timestamp remain correct.
- Substring highlighting and current-match navigation operate against `visibleIndices` so that "next match" still moves through the visible set.
- Auto-tail interaction: while filter is on, only auto-scroll on append if the new line matches the current pattern. Otherwise stay put.

When the user toggles filter off, the virtualizer count returns to `processedLines.length`.

### F — Floating "jump to bottom" arrow

A circular floating button overlaid on the scroll area, bottom-right, ~12px inset:

- Visible when `sticky === false` and there is at least one line.
- Click handler sets `sticky = true` and scrolls to the last index. The existing `$effect` watching `sticky` keeps it tailing afterwards.
- Implementation: an absolutely positioned `<button>` inside the scroll wrapper (or its parent) with `pointer-events: auto` and a high `z-index` to sit above rows.

### G — Case-sensitive toggle

A small `Aa` button next to the existing `.*` regex toggle, state `caseSensitive`. When on, drop the `'i'` flag from the constructed `RegExp`. Default: off (current behavior).

### I — Search debounce (300ms)

The search input continues to bind directly to `searchQuery` so typing feels responsive. A new `$state` `debouncedQuery` is updated 300ms after the last `searchQuery` change via a small debounce helper inside an `$effect`. `searchPattern` derives from `debouncedQuery` instead of `searchQuery`. The toggles (`regexSearch`, `caseSensitive`) bypass the debounce — flipping them updates the pattern immediately.

### Toolbar layout

From left to right:

```
[search input] [Aa] [.*] [↑] [n/N] [↓]  |  [HL] [Wrap] [Filter]  |  [EOF?]
```

The `n/N` counter sits between the prev/next arrows and shows position in `matchIndices`. Existing keyboard nav and click bindings stay.

## Testing

Manual testing via storybook (`apps/docs/src/stories/VirtualLogViewerStory.svelte`). Add scenarios:

- Long single-line logs that require horizontal scroll — verify highlight extends across full width.
- Logs with many matches — verify counter, active-match emphasis, and prev/next wrap-around.
- Invalid regex — verify red border, no crash.
- Filter mode toggled on a noisy log — verify only matches render and counter still tracks.
- Paused tail — verify floating arrow appears and re-tails on click.
- Case-sensitive toggle interaction with regex toggle.
- Large log (10k+ lines) with rapid typing — verify debounce keeps typing smooth.

No new unit tests this round. Existing tests in `frontend/src/lib/__tests__/LogsPanel.svelte.test.ts` should continue to pass since the panel-level API does not change.

## File touchlist

- `packages/ui/package/VirtualLogViewer.svelte` — all logic, markup, and style changes.
- `packages/ui/.svelte-kit/__package__/VirtualLogViewer.svelte` — regenerated build artifact (do not hand-edit).
- `apps/docs/src/stories/VirtualLogViewerStory.svelte` — add storybook scenarios listed above.

## Open considerations

- The virtualizer caches measured row heights when `wrap` is on. Adding `<mark>` wrappers should not change the measured height (inline elements), but verify under wrap mode at small font sizes.
- `markMatches` must be careful not to wrap inside existing `<mark>` (no-op) and not inside elements that own meaningful semantics. `<span class="hl-…">` is fine to descend into.
- When toggling filter mode while the cursor sits on a match, the cursor should be preserved if the same match is still visible; otherwise reset to 0.
