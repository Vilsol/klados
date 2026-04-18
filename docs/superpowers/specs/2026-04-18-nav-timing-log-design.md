# Navigation Timing Debug Log

Date: 2026-04-18

## Goal

Emit a single debug log entry per frontend route navigation containing:

- `firstPaintMs` — time from navigation start until the first frame is painted (double-rAF).
- `ttiMs` — time from navigation start until the main thread is quiet (no `longtask` entries for a 500ms window), i.e. when the user's screen "unfreezes."

This is a dev/diagnostic tool, not user-facing. No UI.

## Non-Goals

- No data-loaded / skeleton-cleared signal. TTI here is main-thread readiness, not semantic "page finished loading."
- No settings toggle, no ring buffer, no HUD overlay.
- No programmatic `markNavStart()` for non-route transitions (can be added later).

## Architecture

Single new module, one subscription, one init call.

**New file:** `frontend/src/lib/navTiming.ts`
**Modified:** `frontend/src/App.svelte` — add `import "$lib/navTiming";` alongside existing side-effect imports.

### Data flow

```
svelte-spa-router location store
        │ (hash change)
        ▼
navTiming.ts: onNav(route)
        │
        ├── cancel in-flight measurement (if any)
        ├── navStart = performance.now()
        ├── schedule double-rAF → firstPaintMs
        └── start PerformanceObserver(longtask) + 500ms quiet-window timer
                │
                │ each longtask resets the quiet-window timer
                │
                ▼ (quiet window elapses OR 10s ceiling hit)
        log.debug("nav", { route, firstPaintMs, ttiMs, longTaskCount, timedOut })
```

## Detailed behavior

### Navigation boundary

Subscribe once to the `location` store exported from `svelte-spa-router`. Each emit, including the initial one at app load, is a navigation boundary. For the initial emit, `navStart = 0` so cold-start TTI is measured relative to `performance.timeOrigin`.

### Measurement state

A single module-level `Measurement | null` is held. Starting a new measurement cancels the previous:

- disconnect the `PerformanceObserver`
- clear the quiet-window `setTimeout`
- clear the 10s ceiling `setTimeout`
- the two pending rAFs are left to fire into a no-op (their callbacks check `current === this` and bail)

Only one nav is measured at a time; rapid navs discard earlier in-flight measurements without logging them.

### First-paint (D)

Nested `requestAnimationFrame` pair. Inner callback records `firstPaintMs = performance.now() - navStart`. Stored on the measurement; logged at the end alongside `ttiMs`.

### TTI (B)

1. `PerformanceObserver` with `entryTypes: ["longtask"]`. Each observed entry increments `longTaskCount` and updates `lastLongtaskEnd = entry.startTime + entry.duration`.
2. A 500ms quiet-window `setTimeout` starts at nav boundary. Every longtask resets this timer (`clearTimeout` + `setTimeout(500)`).
3. When the timer fires: `ttiMs = (lastLongtaskEnd ?? firstPaintAbsTime ?? now) - navStart`. Emit log, clean up.

If `PerformanceObserver.supportedEntryTypes` does not include `"longtask"`, skip the observer; the quiet-window timer still fires at 500ms after nav, giving `ttiMs ≈ firstPaintMs + 500` (noted via `longtaskSupported: false` in the log payload).

### Safety ceiling

A 10000ms `setTimeout` scheduled at nav boundary. If it fires before TTI resolves, log with `timedOut: true` using whatever data is available, then clean up. Prevents observer leaks on pathological pages.

## Log shape

```ts
getLogger("nav-timing").debug("nav", {
  route: string,              // e.g. "/c/kind-dev/apps.v1.deployments"
  firstPaintMs: number,       // 2 decimal places
  ttiMs: number,              // 2 decimal places
  longTaskCount: number,
  longtaskSupported: boolean, // omitted when true
  timedOut?: true,            // omitted when false
});
```

Emitted via the existing `getLogger` from `frontend/src/lib/logger.ts`, so it lands in both the DevTools console (tslog) and the Go slog stream (`LogFrontend`).

## Edge cases

| Case | Handling |
|---|---|
| Initial app load | First `location` emit with `navStart = 0`; measures cold-start TTI from `timeOrigin`. |
| Rapid sequential navs | Previous measurement cancelled silently, no log. Latest nav wins. |
| No `longtask` support | Observer skipped; quiet-window timer still fires. `longtaskSupported: false` in payload. |
| Nav with zero long tasks | Quiet window expires at navStart + 500ms; `ttiMs ≈ firstPaintMs` or 500, whichever is larger. |
| 10s ceiling | `timedOut: true`, cleanup runs, no observer leak. |
| Duplicate location emits with same hash | Treated as a new nav boundary; acceptable (rare and cheap). |

## Testing

Unit test `frontend/src/lib/__tests__/navTiming.test.ts` using `vitest` fake timers + a manual `PerformanceObserver` mock:

- logs `firstPaintMs` and `ttiMs` on a clean nav (no long tasks).
- `ttiMs` reflects the last long task when multiple fire within the quiet window.
- rapid successive navs cancel the earlier measurement (only one log emitted).
- `timedOut: true` emitted when the 10s ceiling hits.
- `longtaskSupported: false` when the entry type is unavailable.

Mock `getLogger` to capture `debug` calls.

## Out of scope

- Measuring non-route transitions (panel opens, tab switches).
- Persisted history UI.
- Any settings or toggle.
- Any change to backend logging.
