# Cluster Events Page Rework — Design

**Date:** 2026-04-15
**Status:** Draft for review
**Scope:** `frontend/src/routes/EventStreamPage.svelte` and related shared components.

## Goals

1. **Obvious failure detection.** A glance at the page reveals whether the cluster is healthy or spiking warnings, and roughly when.
2. **Platform consistency.** The page feels like a first-class member of the app — shares the same table, drawer, column, and filter primitives as `ResourceList` and `ClusterList`.
3. **Deep-dive per event.** Every event can be opened for full detail, and its involved object can be opened from there — matching the existing ownership-chain UX.

## Non-goals

- Server-side event aggregation or persistence.
- Alerting, paging, or notification on event spikes.
- Exporting / downloading event streams (can be added later).

## Current state

`EventStreamPage.svelte` predates the `DataTable` extraction (commit `95b21d1b`) and the subsequent `ResourceList` / `ClusterList` refactors (`face09e1`, `249e18a2`). It renders a plain HTML `<table>` with hand-rolled rows, a two-checkbox severity filter, and a free-text `reason` substring filter. It has no virtualization, no sortable columns, no column management, no click-through on the involved object, no detail view, and no visual severity summary.

`EventsPanel.svelte` (used inside the resource detail drawer to show events for a single object) duplicates most of the row rendering.

## Architecture

`EventStreamPage.svelte` becomes a thin orchestrator:

```
EventStreamPage
├── EventSeverityTimeline   (new — top strip, ~40px, sparkline + brush)
├── EventToolbar            (new — filters + grouping + live-tail + column menu)
├── DataTable               (@klados/ui — virtualized, sortable, resizable)
│   └── cell snippet renders event rows via shared column definitions
└── DetailDrawer            (@klados/ui — single slot, swap-in-place)
    └── EventDetailPanel   (new — full event + clickable involved object)
        └── on involved-object click → swap drawer contents to ResourceDetail
```

Reused: `@klados/ui/DataTable`, `@klados/ui/DetailDrawer`, `ResourceDetail`, `createResourceStore` (with `core.v1.events`), session persistence conventions from `ResourceListPage`, `openOwnerDrawer`-style swap handler.

New components (under `frontend/src/lib/event/`):

| File | Purpose |
|---|---|
| `EventSeverityTimeline.svelte` | Severity sparkline + brush selection |
| `EventToolbar.svelte` | Filter chips, grouping toggle, live-tail controls, column menu |
| `EventDetailPanel.svelte` | Full event view inside the drawer |
| `EventTypeBadge.svelte` | Shared Warning / Normal pill |
| `event-columns.ts` | Column descriptors, formatters, severity classification |
| `event-grouping.ts` | Pure `groupBy(items, keyFn)` reducer with first/last-seen aggregation |

## Event Detail Drawer

Matches the existing ownership-chain pattern in `ResourceListPage.svelte:172-181` (single-slot `DetailDrawer`, no back stack — clicks replace contents).

**EventDetailPanel contents:**

- Header: severity badge · reason · age.
- Metadata grid: `Type`, `Reason`, `Count`, `First seen`, `Last seen`, `Source.component`, `Source.host`, `Reporting controller`.
- Involved-object card (clickable): shows `Kind · Namespace/Name`, hover highlight. On click, swap drawer contents to `ResourceDetail` for that object via the same handler shape as `openOwnerDrawer`.
- Full `message` in a mono block with `CopyableValue`.
- Collapsible raw YAML at the bottom, matching the aesthetic of `ResourceDetail`.

Esc closes. Clicking a different event row while the drawer is open swaps contents to that event (consistent with existing behavior elsewhere).

## Severity Timeline

A ~40px strip above the toolbar. Two-row histogram: `Warning` (red, foreground) and `Normal` (muted, background).

- **Bucketing.** ~60 buckets across the visible time range. Bucket size snaps to `{15s, 30s, 1m, 5m, 15m, 1h}` based on range.
- **Filters.** Bars reflect the current filter state (type, reason, kind, namespace, search). A faint unfiltered total overlay renders behind so cluster-wide context stays visible while drilling in.
- **Interactions.**
  - Drag across bars → time-window filter applied to the table, surfaced as a filter chip in the toolbar.
  - Click the chip's × → clears the window.
  - Hover a bar → tooltip `"14:32–14:33 · 12 Warning, 3 Normal"`.
- **Implementation.** Pure client-side aggregation over `store.items`, memoized in `$derived`.

## Toolbar

Replaces the current toolbar:

- **Severity toggles** — `Warning` / `Normal` pills.
- **Kind filter** — multi-select combobox over unique `involvedObject.kind` values in the current stream.
- **Reason filter** — multi-select combobox over unique `reason` values (replaces the free-text reason filter).
- **Search** — free text over `message` + `reason` + `involvedObject.name`.
- **Namespace** — honors sidebar `selectedNamespaces`; also shown as a chip in the toolbar for discoverability. Watch scope is optimized: watch a single namespace when `selectedNamespaces.length === 1`, otherwise watch all and filter client-side (same pattern as `ResourceListPage`).
- **Group by reason + object** toggle.
- **Live tail controls.** When the user scrolls up from the top, a `Paused` indicator appears along with a `Jump to latest` button. Auto-scroll resumes when the user jumps back.
- **Column menu** — the same prop-driven component used by `ResourceList` / `ClusterList`.
- **Count badge.** `342 events · 18 warnings (last 30m)`.

## Columns

All columns participate in the column menu (visibility, width, order persisted to session).

| Column | Default | Sort | Notes |
|---|---|---|---|
| Type | visible | ✓ | badge |
| Reason | visible | ✓ | mono |
| Object | visible | ✓ | `Kind/Name`, click navigates or opens drawer for that object |
| Message | visible | ✗ | truncate + tooltip on hover |
| Count | visible | ✓ | right-aligned numeric |
| First seen | hidden | ✓ | shown by default in group mode |
| Last seen / Age | visible | ✓ | the current "Age" column, renamed for clarity |
| Namespace | hidden | ✓ | useful when browsing cluster-wide |
| Source | hidden | ✓ | `component / host` |

Widths, visibility, order, and current sort persist to session under a dedicated key for the events page.

## Grouping

A pure client-side reducer:

```
groupBy(items, (e) => `${e.reason}|${e.involvedObject.uid ?? e.involvedObject.namespace + '/' + e.involvedObject.name}`)
  → {
      key,
      reason,
      involvedObject,
      count: sum(e.count ?? 1),
      firstSeen: min(lastTimestamp | eventTime | creationTimestamp),
      lastSeen: max(...),
      sample: firstMatchingItem,
      message: sample.message,
      type: max-severity of group,
    }
```

The `DataTable` consumes either raw items or grouped synthetic rows through a common row shape produced by `event-columns.ts`. Opening a grouped row in the drawer opens the `sample` event (most recent), with a note that `N` events are in this group.

## Shared code with EventsPanel

The detail-drawer events panel (`EventsPanel.svelte`) migrates to consume `event-columns.ts`, `EventTypeBadge.svelte`, and the same formatters, so both renderings stay in lock-step. No functional change to `EventsPanel` — only deduplication.

## Correctness

1. **Stable sort.** Secondary key `metadata.uid` when timestamps tie, to avoid row jumping in the virtual list.
2. **Watch-scope optimization.** Single-namespace watch when exactly one namespace is selected, else watch-all with client-side filter. Mirrors `ResourceListPage`.
3. **Age rendering.** Unchanged — the per-second `now` ticker only drives `formatAge` calls in the template and does not invalidate the filtered list.

## Data flow

1. `createResourceStore()` watches `core.v1.events` via the existing watcher machinery; `store.items` holds the current event set.
2. `$derived` filter pipeline: type → reason → kind → namespace → search → time window → group.
3. `EventSeverityTimeline` subscribes to the same pipeline but recomputes its histogram from items (filtered foreground + raw background).
4. `DataTable` receives the final item array; virtualization handles thousands of events without jank.
5. Row click sets `selectedEvent`; drawer renders `EventDetailPanel`.
6. Involved-object click inside the drawer sets `selectedItem` + `selectedGVR` and swaps drawer contents to `ResourceDetail` — identical to the owner-drawer flow in `ResourceListPage`.

## Error handling

- Watch errors are surfaced in the `DataTable`'s `error` prop (existing pattern).
- Failures resolving the GVR for an involved object (e.g. a custom resource the frontend has no descriptor for) keep the card rendered but disable the click, with a tooltip explaining why.
- Empty states for `no events`, `no events matching filters`, and `time window selected has no events` are distinct messages — the empty state reveals what to clear.

## Testing

- **Unit.** `event-grouping.ts` reducer; `event-columns.ts` formatters; severity classification.
- **Component** (vitest + `@testing-library/svelte`).
  - `EventSeverityTimeline` — bucket math for representative ranges; brush-to-window event payload; filter overlay.
  - `EventToolbar` — each filter emits the expected state change; live-tail toggle.
  - `EventDetailPanel` — renders all metadata fields; involved-object click fires the swap handler.
  - `EventStreamPage` integration — filters compose; group toggle flips row set; opening the drawer on a grouped row yields the sample event.
- **Mocking.** `@wailsio/runtime` and the relevant bindings (`GetDescriptors`, `GetEvents`, watcher events) are mocked per project convention (`setup.ts`).
- **Pre-PR smoke.** `cd frontend && pnpm check && pnpm test`.

## Out of scope / future

- Alert rules on event patterns.
- Persistent event buffer across reconnects.
- Per-user saved filter presets.
- Export to JSON / clipboard of the filtered set.
