# Cluster Events Page Rework — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rework the `core.v1.events` stream page to match the rest of the app: shared `DataTable`/`DetailDrawer` primitives, proper filtering and grouping, a severity sparkline, and click-through to per-event and per-involved-object detail.

**Architecture:** Thin orchestrator (`EventStreamPage.svelte`) composing existing `DataTable` + `DetailDrawer` primitives, an events descriptor driving column management through `columnStore`, and three new Svelte components (timeline, toolbar, detail panel). Shared logic lives in `frontend/src/lib/event/` and is reused by the existing drawer-contextual `EventsPanel.svelte`.

**Tech Stack:** Svelte 5 runes, Tailwind v4, TanStack Virtual (via shared `DataTable`), `@klados/ui` primitives, `vitest` + `@testing-library/svelte`, Go descriptor + CEL column exprs.

**Spec:** `docs/superpowers/specs/2026-04-15-cluster-events-page-rework-design.md`

---

## File Map

**Backend (Go)**

- Modify `internal/resource/builtin.go` — add a `core.v1.events` descriptor so `columnStore` can load column definitions for events; regenerate via `wails3 generate bindings` is not needed (descriptor changes flow through existing `GetDescriptors()`).

**Frontend — new shared event modules (`frontend/src/lib/event/`)**

- `event-types.ts` — TypeScript interfaces for an event, grouped-event, severity, involved-object reference.
- `event-columns.ts` — column descriptor helpers, severity classifier, formatters (`formatEventType`, `formatObject`, `involvedObjectKey`).
- `event-grouping.ts` — pure reducer that turns raw events into grouped rows keyed by `reason + involvedObject`.
- `event-timeline.ts` — pure bucketing helper that turns an event array + time range into `[{t, warn, normal}]` buckets.
- `EventTypeBadge.svelte` — colored Warning / Normal pill.

**Frontend — new page components**

- `frontend/src/lib/components/events/EventSeverityTimeline.svelte` — sparkline + brush selection.
- `frontend/src/lib/components/events/EventToolbar.svelte` — filter chips, grouping toggle, live-tail controls, column menu.
- `frontend/src/lib/components/events/EventDetailPanel.svelte` — drawer contents for one event, clickable involved-object card.

**Frontend — rewritten / wired**

- Rewrite `frontend/src/routes/EventStreamPage.svelte` to compose the above.
- Migrate `frontend/src/lib/components/panels/EventsPanel.svelte` to consume `event-columns.ts` and `EventTypeBadge.svelte` (no UX change, just dedup).

**Tests**

- `frontend/src/lib/event/__tests__/event-grouping.test.ts`
- `frontend/src/lib/event/__tests__/event-timeline.test.ts`
- `frontend/src/lib/event/__tests__/event-columns.test.ts`
- `frontend/src/lib/components/events/__tests__/EventSeverityTimeline.svelte.test.ts`
- `frontend/src/lib/components/events/__tests__/EventToolbar.svelte.test.ts`
- `frontend/src/lib/components/events/__tests__/EventDetailPanel.svelte.test.ts`
- `frontend/src/lib/__tests__/EventStreamPage.svelte.test.ts`
- Existing `frontend/src/lib/__tests__/EventsPanel.svelte.test.ts` updated if selectors shift.

---

## Task 1 — Events descriptor + pure shared modules

Establish the data model and all pure logic first. Everything in this task is unit-testable without DOM.

**Files:**
- Modify: `internal/resource/builtin.go` (add events descriptor)
- Create: `frontend/src/lib/event/event-types.ts`
- Create: `frontend/src/lib/event/event-columns.ts`
- Create: `frontend/src/lib/event/event-grouping.ts`
- Create: `frontend/src/lib/event/event-timeline.ts`
- Create: `frontend/src/lib/event/__tests__/event-grouping.test.ts`
- Create: `frontend/src/lib/event/__tests__/event-timeline.test.ts`
- Create: `frontend/src/lib/event/__tests__/event-columns.test.ts`

### Step 1.1 — Add events descriptor to `builtin.go`

Add this descriptor entry alongside the existing ones (match the surrounding style). Check `builtin.go:20-40` for the Pod descriptor as the canonical example of the struct shape.

```go
{
    GVR: "core.v1.events",
    Columns: []Column{
        {Label: "Type",       Expr: "type",                                                           RenderType: RenderBadge},
        {Label: "Reason",     Expr: "reason",                                                         RenderType: RenderText},
        {Label: "Object",     Expr: "involvedObject.kind + '/' + involvedObject.name",                RenderType: RenderText},
        {Label: "Message",    Expr: "message",                                                        RenderType: RenderText},
        {Label: "Count",      Expr: "has(count) ? count : 1",                                         RenderType: RenderText},
        {Label: "First seen", Expr: "has(firstTimestamp) ? firstTimestamp : metadata.creationTimestamp", RenderType: RenderAge, Hidden: true},
        {Label: "Last seen",  Expr: "has(lastTimestamp) ? lastTimestamp : (has(eventTime) ? eventTime : metadata.creationTimestamp)", RenderType: RenderAge},
        {Label: "Namespace",  Expr: "metadata.namespace",                                             RenderType: RenderText, Hidden: true},
        {Label: "Source",     Expr: "(has(source.component) ? source.component : '') + (has(source.host) ? ' @ ' + source.host : '')", RenderType: RenderText, Hidden: true},
    },
    DetailPanels: []string{}, // events page uses its own drawer
}
```

Notes:
- `RenderAge` plus the `formatAge` helper will render `2m`, `5h`, etc.
- The `Hidden: true` entries appear in the Column Menu but are off by default.
- `DetailPanels` stays empty because the event drawer has a bespoke panel.

Verify: `go test ./internal/resource/ -v` still passes; `pnpm check` in `frontend/` still passes (new columns are additive, not breaking).

### Step 1.2 — Type definitions (`event-types.ts`)

```ts
import type {KubernetesResource} from "$lib/types";

export type Severity = "Warning" | "Normal";

export interface InvolvedObjectRef {
  kind: string;
  apiVersion: string;
  name: string;
  namespace: string;
  uid: string;
}

export interface EventItem {
  metadata?: {name?: string; namespace?: string; uid?: string; creationTimestamp?: string};
  type?: string;
  reason?: string;
  message?: string;
  count?: number;
  firstTimestamp?: string;
  lastTimestamp?: string;
  eventTime?: string;
  involvedObject?: Partial<InvolvedObjectRef>;
  source?: {component?: string; host?: string};
  reportingController?: string;
  [k: string]: KubernetesResource | undefined;
}

export interface GroupedEvent {
  key: string;
  reason: string;
  involvedObject: InvolvedObjectRef;
  severity: Severity;
  count: number;
  firstSeen: string;
  lastSeen: string;
  message: string;
  sample: EventItem; // the most recent contributing event
}

export type EventRow = EventItem | GroupedEvent;

export function isGrouped(row: EventRow): row is GroupedEvent {
  return (row as GroupedEvent).key !== undefined;
}
```

### Step 1.3 — Column & formatter helpers (`event-columns.ts`)

```ts
import type {EventItem, EventRow, GroupedEvent, InvolvedObjectRef, Severity} from "./event-types";

export function classifySeverity(e: EventItem | GroupedEvent): Severity {
  if ("severity" in e && e.severity) return e.severity;
  return (e as EventItem).type === "Warning" ? "Warning" : "Normal";
}

export function eventTimestamp(e: EventItem): string {
  return e.lastTimestamp ?? e.eventTime ?? e.metadata?.creationTimestamp ?? "";
}

export function eventFirstTimestamp(e: EventItem): string {
  return e.firstTimestamp ?? eventTimestamp(e);
}

export function involvedObjectOf(e: EventItem): InvolvedObjectRef {
  const io = e.involvedObject ?? {};
  return {
    kind: io.kind ?? "",
    apiVersion: io.apiVersion ?? "",
    name: io.name ?? "",
    namespace: io.namespace ?? e.metadata?.namespace ?? "",
    uid: io.uid ?? "",
  };
}

export function involvedObjectKey(e: EventItem): string {
  const io = involvedObjectOf(e);
  return io.uid ? io.uid : `${io.namespace}/${io.kind}/${io.name}`;
}

export function formatObject(io: InvolvedObjectRef): string {
  if (!io.kind && !io.name) return "";
  return `${io.kind}/${io.name}`;
}

export function rowReason(row: EventRow): string {
  return (row as EventItem).reason ?? (row as GroupedEvent).reason ?? "";
}

export function rowMessage(row: EventRow): string {
  return (row as EventItem).message ?? (row as GroupedEvent).message ?? "";
}

export function rowCount(row: EventRow): number {
  if ("count" in row && typeof row.count === "number") return row.count;
  return 1;
}

export function rowLastSeen(row: EventRow): string {
  if ("lastSeen" in row && row.lastSeen) return row.lastSeen;
  return eventTimestamp(row as EventItem);
}

export function rowFirstSeen(row: EventRow): string {
  if ("firstSeen" in row && row.firstSeen) return row.firstSeen;
  return eventFirstTimestamp(row as EventItem);
}

export function rowInvolvedObject(row: EventRow): InvolvedObjectRef {
  if ("involvedObject" in row && (row as GroupedEvent).involvedObject) {
    return (row as GroupedEvent).involvedObject;
  }
  return involvedObjectOf(row as EventItem);
}

export function rowSample(row: EventRow): EventItem {
  return (row as GroupedEvent).sample ?? (row as EventItem);
}
```

### Step 1.4 — Grouping reducer (`event-grouping.ts`)

```ts
import type {EventItem, GroupedEvent} from "./event-types";
import {
  classifySeverity,
  eventTimestamp,
  eventFirstTimestamp,
  involvedObjectKey,
  involvedObjectOf,
} from "./event-columns";

export function groupEvents(items: EventItem[]): GroupedEvent[] {
  const groups = new Map<string, GroupedEvent>();
  for (const e of items) {
    const key = `${e.reason ?? ""}|${involvedObjectKey(e)}`;
    const ts = eventTimestamp(e);
    const fts = eventFirstTimestamp(e);
    const count = e.count ?? 1;
    const sev = classifySeverity(e);
    const existing = groups.get(key);
    if (!existing) {
      groups.set(key, {
        key,
        reason: e.reason ?? "",
        involvedObject: involvedObjectOf(e),
        severity: sev,
        count,
        firstSeen: fts,
        lastSeen: ts,
        message: e.message ?? "",
        sample: e,
      });
      continue;
    }
    existing.count += count;
    if (fts && (!existing.firstSeen || fts < existing.firstSeen)) {
      existing.firstSeen = fts;
    }
    if (ts && ts > existing.lastSeen) {
      existing.lastSeen = ts;
      existing.message = e.message ?? existing.message;
      existing.sample = e;
    }
    // Escalate severity to Warning if any contributor is Warning
    if (sev === "Warning") existing.severity = "Warning";
  }
  return Array.from(groups.values());
}
```

### Step 1.5 — Timeline bucketing (`event-timeline.ts`)

```ts
import type {EventItem} from "./event-types";
import {classifySeverity, eventTimestamp} from "./event-columns";

export interface TimelineBucket {
  t0: number; // bucket start (ms)
  t1: number; // bucket end (ms)
  warn: number;
  normal: number;
}

export const BUCKET_SIZES_MS = [
  15_000,   // 15s
  30_000,   // 30s
  60_000,   // 1m
  300_000,  // 5m
  900_000,  // 15m
  3_600_000, // 1h
];

export function pickBucketSize(rangeMs: number, targetBuckets = 60): number {
  const ideal = rangeMs / targetBuckets;
  for (const size of BUCKET_SIZES_MS) {
    if (size >= ideal) return size;
  }
  return BUCKET_SIZES_MS[BUCKET_SIZES_MS.length - 1];
}

export function bucketize(
  items: EventItem[],
  fromMs: number,
  toMs: number,
  bucketSizeMs: number,
): TimelineBucket[] {
  const buckets: TimelineBucket[] = [];
  for (let t = fromMs; t < toMs; t += bucketSizeMs) {
    buckets.push({t0: t, t1: Math.min(t + bucketSizeMs, toMs), warn: 0, normal: 0});
  }
  for (const e of items) {
    const ts = Date.parse(eventTimestamp(e));
    if (!Number.isFinite(ts) || ts < fromMs || ts >= toMs) continue;
    const idx = Math.min(buckets.length - 1, Math.floor((ts - fromMs) / bucketSizeMs));
    if (classifySeverity(e) === "Warning") buckets[idx].warn++;
    else buckets[idx].normal++;
  }
  return buckets;
}
```

### Step 1.6 — Tests for all three pure modules

`event-grouping.test.ts`:

```ts
import {describe, it, expect} from "vitest";
import {groupEvents} from "../event-grouping";
import type {EventItem} from "../event-types";

function ev(overrides: Partial<EventItem>): EventItem {
  return {
    metadata: {uid: Math.random().toString(36).slice(2), namespace: "default", creationTimestamp: "2026-04-15T10:00:00Z"},
    type: "Normal",
    reason: "Scheduled",
    message: "",
    count: 1,
    involvedObject: {kind: "Pod", name: "p1", namespace: "default", uid: "pod-1"},
    lastTimestamp: "2026-04-15T10:00:00Z",
    ...overrides,
  };
}

describe("groupEvents", () => {
  it("merges events by reason + involvedObject", () => {
    const a = ev({reason: "BackOff", lastTimestamp: "2026-04-15T10:00:00Z", count: 3});
    const b = ev({reason: "BackOff", lastTimestamp: "2026-04-15T10:02:00Z", count: 2, message: "latest"});
    const result = groupEvents([a, b]);
    expect(result).toHaveLength(1);
    expect(result[0].count).toBe(5);
    expect(result[0].lastSeen).toBe("2026-04-15T10:02:00Z");
    expect(result[0].message).toBe("latest");
  });

  it("keeps different reasons separate", () => {
    const a = ev({reason: "BackOff"});
    const b = ev({reason: "Failed"});
    expect(groupEvents([a, b])).toHaveLength(2);
  });

  it("keeps different involved objects separate", () => {
    const a = ev({involvedObject: {kind: "Pod", name: "p1", uid: "u1", namespace: "default"}});
    const b = ev({involvedObject: {kind: "Pod", name: "p2", uid: "u2", namespace: "default"}});
    expect(groupEvents([a, b])).toHaveLength(2);
  });

  it("escalates severity to Warning when any contributor is Warning", () => {
    const a = ev({type: "Normal"});
    const b = ev({type: "Warning"});
    const result = groupEvents([a, b]);
    expect(result[0].severity).toBe("Warning");
  });

  it("tracks firstSeen as the earliest firstTimestamp fallback", () => {
    const a = ev({firstTimestamp: "2026-04-15T09:55:00Z", lastTimestamp: "2026-04-15T10:00:00Z"});
    const b = ev({firstTimestamp: "2026-04-15T09:50:00Z", lastTimestamp: "2026-04-15T10:02:00Z"});
    expect(groupEvents([a, b])[0].firstSeen).toBe("2026-04-15T09:50:00Z");
  });
});
```

`event-timeline.test.ts`:

```ts
import {describe, it, expect} from "vitest";
import {bucketize, pickBucketSize, BUCKET_SIZES_MS} from "../event-timeline";

describe("pickBucketSize", () => {
  it("returns a bucket size that fits the range/target ratio", () => {
    expect(pickBucketSize(60 * 60 * 1000, 60)).toBe(60_000); // 1h → 1m
    expect(pickBucketSize(24 * 60 * 60 * 1000, 60)).toBe(900_000); // 24h → 15m
    expect(pickBucketSize(10 * 60 * 1000, 60)).toBe(15_000); // 10m → 15s
  });
  it("clamps to the max for very long ranges", () => {
    expect(pickBucketSize(Number.MAX_SAFE_INTEGER, 60)).toBe(BUCKET_SIZES_MS.at(-1));
  });
});

describe("bucketize", () => {
  it("counts warnings and normals into correct buckets", () => {
    const from = Date.parse("2026-04-15T10:00:00Z");
    const to = from + 5 * 60_000;
    const items = [
      {type: "Warning", lastTimestamp: "2026-04-15T10:00:30Z", involvedObject: {}},
      {type: "Normal",  lastTimestamp: "2026-04-15T10:02:15Z", involvedObject: {}},
      {type: "Warning", lastTimestamp: "2026-04-15T10:04:59Z", involvedObject: {}},
    ];
    const result = bucketize(items, from, to, 60_000);
    expect(result).toHaveLength(5);
    expect(result[0].warn).toBe(1);
    expect(result[2].normal).toBe(1);
    expect(result[4].warn).toBe(1);
  });
  it("drops events outside the range", () => {
    const from = Date.parse("2026-04-15T10:00:00Z");
    const to = from + 60_000;
    const items = [{type: "Warning", lastTimestamp: "2026-04-15T09:00:00Z", involvedObject: {}}];
    expect(bucketize(items, from, to, 60_000)[0].warn).toBe(0);
  });
});
```

`event-columns.test.ts`:

```ts
import {describe, it, expect} from "vitest";
import {
  classifySeverity,
  eventTimestamp,
  involvedObjectKey,
  formatObject,
  involvedObjectOf,
} from "../event-columns";

describe("event-columns helpers", () => {
  it("classifies Warning vs Normal", () => {
    expect(classifySeverity({type: "Warning"} as any)).toBe("Warning");
    expect(classifySeverity({type: "Normal"} as any)).toBe("Normal");
    expect(classifySeverity({} as any)).toBe("Normal");
  });
  it("falls back through timestamp fields", () => {
    expect(eventTimestamp({lastTimestamp: "L", eventTime: "E"} as any)).toBe("L");
    expect(eventTimestamp({eventTime: "E"} as any)).toBe("E");
    expect(eventTimestamp({metadata: {creationTimestamp: "C"}} as any)).toBe("C");
    expect(eventTimestamp({} as any)).toBe("");
  });
  it("uses uid as involvedObjectKey when present, falls back to ns/kind/name", () => {
    expect(involvedObjectKey({involvedObject: {uid: "u1"}} as any)).toBe("u1");
    expect(involvedObjectKey({involvedObject: {kind: "Pod", name: "p", namespace: "ns"}} as any)).toBe("ns/Pod/p");
  });
  it("formatObject renders Kind/Name", () => {
    expect(formatObject(involvedObjectOf({involvedObject: {kind: "Pod", name: "p1"}} as any))).toBe("Pod/p1");
  });
});
```

- [ ] **Step 1a** — Write all new files above (Go + TS + test files) exactly as specified.

- [ ] **Step 1b** — Create `EventTypeBadge.svelte` at `frontend/src/lib/event/EventTypeBadge.svelte`:

```svelte
<script lang="ts">
  import type {Severity} from "./event-types";
  let {severity}: {severity: Severity} = $props();
</script>

<span
  class="px-1.5 py-0.5 rounded text-xs font-medium
  {severity === 'Warning' ? 'bg-destructive/15 text-destructive' : 'bg-accent/15 text-accent'}"
>
  {severity}
</span>
```

- [ ] **Step 1c** — Verify:
  - `go test ./internal/resource/ -v` passes.
  - `cd frontend && pnpm test src/lib/event/__tests__/` passes.
  - `cd frontend && pnpm check` passes.

- [ ] **Step 1d** — Commit via the `jj-vcs` skill with message: `Add events descriptor and shared event modules`.

---

## Task 2 — Event detail drawer panel + EventsPanel migration

Build the per-event detail view that lives inside `DetailDrawer`, and migrate the existing drawer-contextual `EventsPanel` to consume the shared modules. This task groups them because both render "one event" visually and should stay consistent.

**Files:**
- Create: `frontend/src/lib/components/events/EventDetailPanel.svelte`
- Create: `frontend/src/lib/components/events/__tests__/EventDetailPanel.svelte.test.ts`
- Modify: `frontend/src/lib/components/panels/EventsPanel.svelte`
- Modify (if needed): `frontend/src/lib/__tests__/EventsPanel.svelte.test.ts`

### Step 2.1 — `EventDetailPanel.svelte`

Contract:
- Props: `event: EventItem | GroupedEvent`, `ctxName: string`, `now: number`, `onOpenInvolvedObject?: (ref: InvolvedObjectRef, gvr: string) => void`.
- Content layout (top to bottom): header row (`EventTypeBadge`, reason, age-since-lastSeen), clickable involved-object card (disabled + tooltip if GVR can't be resolved), metadata grid (Type, Reason, Count, First seen, Last seen, Source, Reporting controller), full message as a mono `CopyableValue` block, collapsible raw YAML dump of `rowSample(event)` at the bottom using the same YAML aesthetic as `ResourceDetail` (reuse `CodeBlock` from `@klados/ui`).
- Clicking the involved-object card calls `onOpenInvolvedObject` if a GVR resolves via `clusterStore.resolveOwnerGVR(io.apiVersion, io.kind)`. If it does not resolve, render the card but disable the click and show a `title` tooltip: `"No descriptor registered for Kind {kind}"`.

Complete component skeleton (fill in the classes to match the existing `ResourceDetail` look; use `text-xs` for labels, `text-sm` for values, `gap-2`, `grid-cols-[auto_1fr]`):

```svelte
<script lang="ts">
  import {CodeBlock, CopyableValue} from "@klados/ui";
  import EventTypeBadge from "$lib/event/EventTypeBadge.svelte";
  import {
    classifySeverity,
    rowInvolvedObject,
    rowMessage,
    rowSample,
    rowLastSeen,
    rowFirstSeen,
    rowCount,
    rowReason,
    formatObject,
  } from "$lib/event/event-columns";
  import {clusterStore} from "$lib/stores/cluster.svelte";
  import {formatAge} from "$lib/utils/age";
  import {isGrouped, type EventRow, type InvolvedObjectRef} from "$lib/event/event-types";
  import {stringify as yamlStringify} from "yaml";

  let {
    event,
    now,
    onOpenInvolvedObject,
  }: {
    event: EventRow;
    now: number;
    onOpenInvolvedObject?: (ref: InvolvedObjectRef, gvr: string) => void;
  } = $props();

  const severity = $derived(classifySeverity(event as any));
  const io = $derived(rowInvolvedObject(event));
  const sample = $derived(rowSample(event));
  const lastSeen = $derived(rowLastSeen(event));
  const firstSeen = $derived(rowFirstSeen(event));
  const count = $derived(rowCount(event));
  const resolvedGVR = $derived(io.kind ? clusterStore.resolveOwnerGVR(io.apiVersion, io.kind) : null);
  const canNavigate = $derived(Boolean(resolvedGVR && onOpenInvolvedObject));
  const yaml = $derived(yamlStringify(sample));
  let yamlOpen = $state(false);
</script>

<div class="flex flex-col h-full overflow-auto">
  <div class="flex items-center gap-2 px-4 py-3 border-b border-border shrink-0">
    <EventTypeBadge {severity} />
    <span class="font-mono text-sm">{rowReason(event)}</span>
    <span class="text-xs text-muted ml-auto">{formatAge(lastSeen, now)} ago</span>
  </div>

  {#if isGrouped(event)}
    <div class="px-4 py-1.5 text-xs text-muted border-b border-border">
      Grouped: {count} occurrences
    </div>
  {/if}

  <!-- Involved object card -->
  <button
    type="button"
    disabled={!canNavigate}
    onclick={() => { if (canNavigate && resolvedGVR) onOpenInvolvedObject?.(io, resolvedGVR) }}
    class="mx-4 mt-3 text-left border border-border rounded p-3 flex items-center gap-3 transition-colors
      {canNavigate ? 'hover:bg-surface-hover cursor-pointer' : 'opacity-70 cursor-not-allowed'}"
    title={canNavigate ? 'Open involved object' : `No descriptor registered for Kind ${io.kind}`}
  >
    <span class="text-xs text-muted">{io.kind}</span>
    <span class="text-sm font-medium">{io.name}</span>
    {#if io.namespace}
      <span class="text-xs text-muted ml-auto">{io.namespace}</span>
    {/if}
  </button>

  <!-- Metadata grid -->
  <div class="px-4 py-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 text-xs">
    <span class="text-muted">Count</span><span>{count}</span>
    <span class="text-muted">First seen</span><span>{firstSeen ? `${formatAge(firstSeen, now)} ago` : '—'}</span>
    <span class="text-muted">Last seen</span><span>{lastSeen ? `${formatAge(lastSeen, now)} ago` : '—'}</span>
    <span class="text-muted">Source</span>
    <span>{(sample.source?.component ?? '') + (sample.source?.host ? ' @ ' + sample.source.host : '') || '—'}</span>
    <span class="text-muted">Reporting ctrl</span>
    <span>{sample.reportingController ?? '—'}</span>
  </div>

  <!-- Message -->
  <div class="px-4 py-3 border-t border-border">
    <div class="text-xs text-muted mb-1">Message</div>
    <CopyableValue value={rowMessage(event)} class="font-mono text-xs whitespace-pre-wrap" />
  </div>

  <!-- Raw YAML -->
  <div class="px-4 py-3 border-t border-border">
    <button type="button" onclick={() => yamlOpen = !yamlOpen} class="text-xs text-muted hover:text-fg">
      {yamlOpen ? '▾' : '▸'} Raw YAML
    </button>
    {#if yamlOpen}
      <div class="mt-2">
        <CodeBlock language="yaml" value={yaml} />
      </div>
    {/if}
  </div>
</div>
```

Implementation notes:
- `yaml` import: the app already depends on `yaml` (check `frontend/package.json` — `ResourceDetail.svelte` uses it). If not, add it. Do not add a new top-level dependency without confirming.
- `formatAge(ts, now)` matches the signature used by `ResourceList` and `EventStreamPage` today.
- `CodeBlock` and `CopyableValue` are exported by `@klados/ui`.

### Step 2.2 — `EventDetailPanel` test

File: `frontend/src/lib/components/events/__tests__/EventDetailPanel.svelte.test.ts`

Cover:
1. Renders severity badge, reason, count, last/first seen.
2. Involved-object card is clickable when `resolveOwnerGVR` returns a GVR; fires `onOpenInvolvedObject(ref, gvr)`.
3. Involved-object card is disabled with a tooltip when `resolveOwnerGVR` returns null.
4. Grouped events show the "Grouped: N occurrences" strip.

Mock `$lib/stores/cluster.svelte` with a stub exposing `resolveOwnerGVR` that the test controls per case. Also mock `@wailsio/runtime` per project convention (see `setup.ts`).

### Step 2.3 — Migrate `EventsPanel.svelte`

Replace its hand-rolled row rendering with `EventTypeBadge` + the shared formatters. The behavior and the `GetEvents` backend call stay the same; only the row render changes. After the migration, `EventsPanel` should not reference `.type === 'Warning'` directly — severity comes from `classifySeverity`.

Minimum migration: swap the severity `<span>` for `<EventTypeBadge severity={classifySeverity(event)} />`, replace `{formatAge(ts)}` with the existing ts fallback via `eventTimestamp(event)`. Confirm the existing `EventsPanel.svelte.test.ts` still passes; adjust selectors only if necessary.

- [ ] **Step 2a** — Write `EventDetailPanel.svelte` per Step 2.1. The `yaml` package is already a dependency of `frontend/` (used by `ResourceDetail.svelte`); import it directly.

- [ ] **Step 2b** — Write `EventDetailPanel.svelte.test.ts` per Step 2.2.

- [ ] **Step 2c** — Migrate `EventsPanel.svelte` per Step 2.3.

- [ ] **Step 2d** — Verify: `cd frontend && pnpm test` passes for the affected files; `pnpm check` passes.

- [ ] **Step 2e** — Commit via `jj-vcs` with message: `Add event detail panel and migrate EventsPanel to shared modules`.

---

## Task 3 — Severity timeline component

**Files:**
- Create: `frontend/src/lib/components/events/EventSeverityTimeline.svelte`
- Create: `frontend/src/lib/components/events/__tests__/EventSeverityTimeline.svelte.test.ts`

### Step 3.1 — Component contract

Props:
```ts
{
  filteredItems: EventItem[];
  allItems: EventItem[];                // for faint unfiltered overlay
  rangeMs: number;                      // e.g. 30*60_000 for "last 30m"
  now: number;                          // tick source from parent
  selectedWindow: {from: number; to: number} | null;
  onBrush?: (window: {from: number; to: number} | null) => void;
}
```

Behavior:
- Compute `from = now - rangeMs`, `to = now`, `bucketSize = pickBucketSize(rangeMs)`.
- Compute `filteredBuckets = bucketize(filteredItems, from, to, bucketSize)` and `totalBuckets = bucketize(allItems, ...)`.
- Render a 40px SVG with a row per bucket: faint grey background bars for `totalBuckets.warn + totalBuckets.normal`, foreground red bars for `filteredBuckets.warn`, foreground muted bars for `filteredBuckets.normal` (normal stacked below baseline or tinted differently — your call, keep it readable).
- Mouse interactions: `mousedown` records `brushStartIdx`; `mousemove` while pressed sets `brushEndIdx`; `mouseup` emits `onBrush({from: buckets[min].t0, to: buckets[max].t1})` or clears the brush if start === end with a modifier, and clears `brushStartIdx` state. If `selectedWindow` is non-null, render a translucent overlay over the matching bars and a small × affordance to clear.
- Tooltip on hover: `"HH:MM–HH:MM · W warnings, N normal"`.

### Step 3.2 — Tests

Cover:
1. Given a fixed `filteredItems` and `rangeMs`, it renders the correct number of bars (the snapshot need not be pixel-perfect — assert on `<rect>` count or `data-bucket` attributes).
2. Dragging from bucket i to bucket j emits `onBrush({from, to})` matching `buckets[i].t0` and `buckets[j].t1`.
3. When `selectedWindow` is set, clicking the clear affordance fires `onBrush(null)`.

Use `@testing-library/svelte`'s `fireEvent.mouseDown/mouseMove/mouseUp` to simulate the brush. For time-dependent assertions pin `now` to a constant.

- [ ] **Step 3a** — Write the component and tests.
- [ ] **Step 3b** — Verify: `pnpm test` for the new file passes; `pnpm check` passes.
- [ ] **Step 3c** — Commit via `jj-vcs` with message: `Add event severity timeline`.

---

## Task 4 — Toolbar component

**Files:**
- Create: `frontend/src/lib/components/events/EventToolbar.svelte`
- Create: `frontend/src/lib/components/events/__tests__/EventToolbar.svelte.test.ts`

### Step 4.1 — Props

```ts
{
  // Severity
  showWarning: boolean;
  showNormal: boolean;
  onSeverityChange: (next: {showWarning: boolean; showNormal: boolean}) => void;

  // Multi-select filters
  availableKinds: string[];
  selectedKinds: string[];
  onKindsChange: (v: string[]) => void;

  availableReasons: string[];
  selectedReasons: string[];
  onReasonsChange: (v: string[]) => void;

  // Search (reason + message + involvedObject.name)
  search: string;
  onSearchChange: (v: string) => void;

  // Grouping
  grouped: boolean;
  onGroupedChange: (v: boolean) => void;

  // Live tail
  paused: boolean;
  onJumpToLatest: () => void;

  // Counts for the right-side badge
  totalCount: number;
  warningCount: number;
  rangeLabel: string; // e.g. "last 30m"

  // Column menu plumbing (identical to ResourceList)
  columnMenuOpen: boolean;
  onColumnMenuToggle: () => void;

  // Time window chip, if set
  timeWindow: {from: number; to: number} | null;
  onClearTimeWindow: () => void;
}
```

Layout: row with severity pills · kinds combobox · reasons combobox · search input · `Group` toggle · `Paused`/`Jump to latest` cluster · flex spacer · count badge · column menu button. Mirror the styling of `ResourceList`'s toolbar (see `ResourceList.svelte:392-459`). When `timeWindow` is non-null, render a chip `⟶ 14:32–14:35 ×` immediately to the right of the search input.

Use `@klados/ui`'s `Combobox` for the multi-selects (see `packages/ui/src/lib/Combobox.svelte`). If its API doesn't support multi-select, fall back to a dropdown of checkboxes in the same layout as `ColumnMenu` (see `ColumnMenu.svelte`); the toolbar component owns the open/close state for each dropdown via `$state`.

### Step 4.2 — Tests

Cover:
1. Toggling severity pills emits `onSeverityChange` with the correct payload.
2. Entering text fires `onSearchChange` (debounced if you add a debounce — matches `SmartSearch.svelte`; mirror that behavior).
3. Picking a kind emits `onKindsChange` with the union.
4. `Jump to latest` button is visible iff `paused` is true.
5. Time-window chip × click fires `onClearTimeWindow`.

- [ ] **Step 4a** — Write the component and tests.
- [ ] **Step 4b** — Verify: `pnpm test` for the new file; `pnpm check`.
- [ ] **Step 4c** — Commit via `jj-vcs` with message: `Add event toolbar`.

---

## Task 5 — Rewrite `EventStreamPage.svelte`

Compose all the pieces. This is the largest task — no point splitting because every piece is interdependent.

**Files:**
- Rewrite: `frontend/src/routes/EventStreamPage.svelte`
- Create: `frontend/src/lib/__tests__/EventStreamPage.svelte.test.ts`

### Step 5.1 — New `EventStreamPage.svelte`

Structure mirrors `ResourceListPage.svelte`. Keep the existing `createResourceStore` + watch bootstrap; everything else is new.

State the page owns:
```ts
let showWarning = $state(true);
let showNormal = $state(true);
let selectedKinds = $state<string[]>([]);
let selectedReasons = $state<string[]>([]);
let search = $state("");
let grouped = $state(false);
let timeWindow = $state<{from: number; to: number} | null>(null);
let paused = $state(false);             // true when user scrolled up
let selectedRow = $state<EventRow | null>(null);
let selectedInvolvedItem = $state<Record<string, unknown> | null>(null);
let selectedInvolvedGVR = $state<string>("");
let columnMenuOpen = $state(false);
let now = $state(Date.now());
const rangeMs = 30 * 60_000;
```

Pipeline (all `$derived`):
1. `rawItems = store.items as EventItem[]`
2. `selectedNs = clusterStore.getSelectedNamespaces(ctxName)`
3. `nsFiltered = selectedNs.length ? rawItems.filter((e) => selectedNs.includes(e.metadata?.namespace ?? '')) : rawItems`
4. `severityFiltered` applies `showWarning` / `showNormal`.
5. `kindFiltered` applies `selectedKinds`.
6. `reasonFiltered` applies `selectedReasons`.
7. `searchFiltered` — lowercase substring match of `search` across `reason`, `message`, `involvedObject.name`.
8. `windowFiltered` applies `timeWindow` by `Date.parse(eventTimestamp(e))`.
9. `rowsRaw = windowFiltered` **sorted** by `lastTimestamp` desc, tiebreak on `metadata.uid`.
10. `rows = grouped ? groupEvents(windowFiltered) : rowsRaw`

Column plumbing: `columnStore.loadForGVR("core.v1.events")` on mount. Pass `columnStore.visibleColumns` to `DataTable`. Cell rendering reads `column.name` and picks the right helper from `event-columns.ts` — no CEL evaluation here, because `EventRow` may be a synthetic group row the descriptor CEL doesn't understand.

```ts
function renderCell(name: string, row: EventRow): {kind: "badge" | "text" | "age", value: string} {
  switch (name) {
    case "Type":       return {kind: "badge", value: classifySeverity(row as any)};
    case "Reason":     return {kind: "text",  value: rowReason(row)};
    case "Object":     return {kind: "text",  value: formatObject(rowInvolvedObject(row))};
    case "Message":    return {kind: "text",  value: rowMessage(row)};
    case "Count":      return {kind: "text",  value: String(rowCount(row))};
    case "First seen": return {kind: "age",   value: rowFirstSeen(row)};
    case "Last seen":  return {kind: "age",   value: rowLastSeen(row)};
    case "Namespace":  return {kind: "text",  value: rowInvolvedObject(row).namespace};
    case "Source":     return {kind: "text",  value: rowSample(row).source?.component ?? ""};
    default:           return {kind: "text",  value: ""};
  }
}
```

Watch scope: mirror `ResourceListPage`:
```ts
const rawWatchNamespace = $derived(selectedNs.length === 1 ? selectedNs[0] : "");
$effect(() => {
  if (ctxName) store.start(ctxName, "core.v1.events", rawWatchNamespace);
  return () => store.stop();
});
```

Live-tail: when `scrollContainer.scrollTop > 0`, set `paused = true`. `Jump to latest` scrolls to top.

Drawer pattern (same as `ResourceListPage`):
```svelte
{#if selectedRow}
  <DetailDrawer
    item={selectedInvolvedItem ?? (rowSample(selectedRow) as any)}
    {ctxName}
    gvr={selectedInvolvedGVR || "core.v1.events"}
    onclose={() => { selectedRow = null; selectedInvolvedItem = null; selectedInvolvedGVR = "" }}
    onFetchResource={async (c, g, ns, n) => { try { return await GetResource(c, g, ns, n) } catch { return null } }}
  >
    {#snippet children({ obj, onrefresh })}
      {#if selectedInvolvedItem}
        <ResourceDetail
          {obj}
          descriptor={descriptorRegistry.get(selectedInvolvedGVR)}
          {ctxName}
          gvr={selectedInvolvedGVR}
          namespace={obj.metadata?.namespace ?? ''}
          name={obj.metadata?.name ?? ''}
          {onrefresh}
          onopenowner={openOwnerDrawer}
        />
      {:else if selectedRow}
        <EventDetailPanel
          event={selectedRow}
          {now}
          onOpenInvolvedObject={openInvolvedObject}
        />
      {/if}
    {/snippet}
  </DetailDrawer>
{/if}
```

Handlers:
```ts
async function openInvolvedObject(ref: InvolvedObjectRef, gvr: string) {
  try {
    const obj = await GetResource(ctxName, gvr, ref.namespace, ref.name);
    if (obj) {
      selectedInvolvedItem = obj as any;
      selectedInvolvedGVR = gvr;
    }
  } catch {
    notificationStore.push("Involved object not found", "error");
  }
}

async function openOwnerDrawer(ref: ControllerRef, namespace: string) {
  const ownerGVR = clusterStore.resolveOwnerGVR(ref.apiVersion, ref.kind);
  if (!ownerGVR) return;
  const owner = await GetResource(ctxName, ownerGVR, namespace, ref.name);
  if (owner) {
    selectedInvolvedItem = owner as any;
    selectedInvolvedGVR = ownerGVR;
  }
}
```

Clicking a row: `onrowclick={(row) => { selectedRow = row; selectedInvolvedItem = null; selectedInvolvedGVR = "" }}`.

Replace-contents behavior comes naturally — the drawer swaps between event-mode and object-mode based on which of `selectedInvolvedItem` vs `selectedRow` is present.

### Step 5.2 — Integration test

File: `frontend/src/lib/__tests__/EventStreamPage.svelte.test.ts`. Mock `@wailsio/runtime` per convention and mock the event watch by pre-seeding `createResourceStore().items` (dig into `resource.svelte.ts` for the test-friendly path — see the pattern used by `EventsPanel.svelte.test.ts`).

Cover:
1. Filters compose: flipping off `Warning` removes warning rows.
2. Grouping toggle collapses identical `reason + involvedObject`.
3. Clicking a row opens the drawer with `EventDetailPanel`.
4. Clicking the involved-object card when a GVR resolves swaps drawer contents to `ResourceDetail`.

- [ ] **Step 5a** — Write the new `EventStreamPage.svelte` per Step 5.1.
- [ ] **Step 5b** — Write the integration test.
- [ ] **Step 5c** — Verify: `pnpm test` on all new and changed event tests; `pnpm check` passes.
- [ ] **Step 5d** — Manual smoke: `task dev`, open the events page on a real cluster (k3d or whatever the user has), verify: filters, grouping toggle, severity timeline renders, brush-to-window narrows rows, row click opens event drawer, involved-object click swaps to `ResourceDetail`, Esc closes, Paused indicator + Jump to latest works when scrolled.
- [ ] **Step 5e** — Commit via `jj-vcs` with message: `Rework EventStreamPage to compose DataTable, toolbar, timeline, and drawer`.

---

## Task 6 — Finalize & anatomy update

- [ ] **Step 6a** — Run the full frontend suite: `cd frontend && pnpm check && pnpm test`.
- [ ] **Step 6b** — Run Go tests: `go test ./internal/resource/ -v` (the only backend package touched).
- [ ] **Step 6c** — Update `.wolf/anatomy.md` entries for the new files and the rewritten page per the OpenWolf protocol.
- [ ] **Step 6d** — Commit via `jj-vcs` with message: `Update anatomy for events page rework`.

---

## Testing Strategy Summary

- **Pure reducers** (`event-grouping`, `event-timeline`, `event-columns`) — unit tests cover grouping merge rules, severity escalation, bucketing math, bucket-size selection.
- **Components** (`EventTypeBadge`, `EventSeverityTimeline`, `EventToolbar`, `EventDetailPanel`) — component tests via `@testing-library/svelte` with `@wailsio/runtime` mocked per project convention. Each test asserts *one* contract: event emission on interaction, conditional rendering, or filter gating.
- **Integration** (`EventStreamPage`) — one test file exercising the filter pipeline end-to-end and the drawer swap behavior.
- **Manual** — smoke run on a live cluster covers the pieces unit tests can't: brush selection, sparkline visual, live-tail pause/resume, YAML rendering.

## Out of Scope

Per the spec: alerting, persistent event buffer across reconnects, per-user saved filter presets, export to JSON/clipboard. Do not add these.
