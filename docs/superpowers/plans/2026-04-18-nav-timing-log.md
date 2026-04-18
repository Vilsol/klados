# Navigation Timing Debug Log — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Emit one debug log entry per frontend navigation containing `firstPaintMs` (double-rAF) and `ttiMs` (longtask quiet-window), so we can measure when the user's screen actually unfreezes after a route change.

**Architecture:** A single module (`frontend/src/lib/navTiming.ts`) subscribes to `svelte-spa-router`'s `location` store. On each emit, it cancels any in-flight measurement, starts a new one using a double `requestAnimationFrame` for first-paint and a `PerformanceObserver({entryTypes:["longtask"]})` + 500ms quiet-window timer for TTI, with a 10s safety ceiling. Results are logged via the existing `getLogger("nav-timing")` (tslog + Wails backend transport). One-line side-effect import in `App.svelte` turns it on.

**Tech Stack:** TypeScript, Svelte 5, `svelte-spa-router`, `tslog` (via `$lib/logger`), `vitest` with fake timers.

**Spec:** `docs/superpowers/specs/2026-04-18-nav-timing-log-design.md`

---

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `frontend/src/lib/navTiming.ts` | Create | Self-initializing side-effect module. Holds measurement state, subscribes to `location`, emits log entries. |
| `frontend/src/App.svelte` | Modify | Add one side-effect import alongside the existing `$lib` imports. |
| `frontend/src/lib/__tests__/navTiming.test.ts` | Create | Vitest suite using fake timers + mocked `getLogger`, `PerformanceObserver`, and `requestAnimationFrame`. |

---

## Task 1: Implement the `navTiming` module and its tests

**Files:**
- Create: `frontend/src/lib/navTiming.ts`
- Create: `frontend/src/lib/__tests__/navTiming.test.ts`

This single task builds the entire module plus its tests together. Keep the two files in sync as you iterate; the test is the primary verification.

### - [ ] Step 1: Create `frontend/src/lib/navTiming.ts`

Full file contents:

```ts
import {location} from "svelte-spa-router";
import {getLogger} from "$lib/logger";

const QUIET_WINDOW_MS = 500;
const CEILING_MS = 10_000;

const log = getLogger("nav-timing");

type Measurement = {
  route: string;
  navStart: number;
  firstPaintMs: number | null;
  firstPaintAbsTime: number | null;
  longTaskCount: number;
  lastLongtaskEnd: number | null;
  observer: PerformanceObserver | null;
  quietTimer: ReturnType<typeof setTimeout> | null;
  ceilingTimer: ReturnType<typeof setTimeout> | null;
  longtaskSupported: boolean;
  done: boolean;
};

let current: Measurement | null = null;

function longtaskSupported(): boolean {
  return (
    typeof PerformanceObserver !== "undefined" &&
    Array.isArray(PerformanceObserver.supportedEntryTypes) &&
    PerformanceObserver.supportedEntryTypes.includes("longtask")
  );
}

function cleanup(m: Measurement): void {
  m.done = true;
  if (m.observer) {
    try {
      m.observer.disconnect();
    } catch {
      /* ignore */
    }
    m.observer = null;
  }
  if (m.quietTimer !== null) {
    clearTimeout(m.quietTimer);
    m.quietTimer = null;
  }
  if (m.ceilingTimer !== null) {
    clearTimeout(m.ceilingTimer);
    m.ceilingTimer = null;
  }
}

function emit(m: Measurement, timedOut: boolean): void {
  if (m.done) {
    return;
  }
  cleanup(m);

  const now = performance.now();
  const endAbs = m.lastLongtaskEnd ?? m.firstPaintAbsTime ?? now;
  const ttiMs = Math.max(0, endAbs - m.navStart);
  const firstPaintMs = m.firstPaintMs ?? 0;

  const payload: Record<string, unknown> = {
    route: m.route,
    firstPaintMs: Number(firstPaintMs.toFixed(2)),
    ttiMs: Number(ttiMs.toFixed(2)),
    longTaskCount: m.longTaskCount,
  };
  if (!m.longtaskSupported) {
    payload.longtaskSupported = false;
  }
  if (timedOut) {
    payload.timedOut = true;
  }
  log.debug("nav", payload);
}

function armQuietWindow(m: Measurement): void {
  if (m.quietTimer !== null) {
    clearTimeout(m.quietTimer);
  }
  m.quietTimer = setTimeout(() => emit(m, false), QUIET_WINDOW_MS);
}

function start(route: string, navStart: number): void {
  if (current && !current.done) {
    cleanup(current);
  }

  const m: Measurement = {
    route,
    navStart,
    firstPaintMs: null,
    firstPaintAbsTime: null,
    longTaskCount: 0,
    lastLongtaskEnd: null,
    observer: null,
    quietTimer: null,
    ceilingTimer: null,
    longtaskSupported: longtaskSupported(),
    done: false,
  };
  current = m;

  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      if (m.done || current !== m) {
        return;
      }
      const t = performance.now();
      m.firstPaintAbsTime = t;
      m.firstPaintMs = Math.max(0, t - m.navStart);
    });
  });

  if (m.longtaskSupported) {
    try {
      m.observer = new PerformanceObserver((list) => {
        if (m.done || current !== m) {
          return;
        }
        for (const entry of list.getEntries()) {
          m.longTaskCount += 1;
          m.lastLongtaskEnd = entry.startTime + entry.duration;
        }
        armQuietWindow(m);
      });
      m.observer.observe({entryTypes: ["longtask"]});
    } catch {
      m.longtaskSupported = false;
      m.observer = null;
    }
  }

  armQuietWindow(m);
  m.ceilingTimer = setTimeout(() => emit(m, true), CEILING_MS);
}

let bootstrapped = false;
function bootstrap(): void {
  if (bootstrapped) {
    return;
  }
  bootstrapped = true;
  let first = true;
  location.subscribe((route) => {
    const navStart = first ? 0 : performance.now();
    first = false;
    start(route ?? "", navStart);
  });
}

bootstrap();

export const __test__ = {start, emit, cleanup, get current() { return current; }};
```

Key invariants (do not relax without updating the spec):

- Only one measurement at a time; `start()` cancels the previous by calling `cleanup()` on it (no log emitted for cancelled measurements).
- Callbacks guard with `m.done || current !== m` so stale rAFs / observer callbacks are no-ops.
- Initial `location` emit uses `navStart = 0` so cold-start is measured from `performance.timeOrigin`.
- `emit()` calls `cleanup()` so the ceiling timer fires exactly once.

### - [ ] Step 2: Create `frontend/src/lib/__tests__/navTiming.test.ts`

The test drives the module via the exported `__test__` helpers to sidestep the `svelte-spa-router` store, and uses fake timers for the quiet window and ceiling. It stubs `performance.now`, `requestAnimationFrame`, and `PerformanceObserver`.

Full file contents:

```ts
import {afterEach, beforeEach, describe, expect, test, vi} from "vitest";

const debugMock = vi.fn();

vi.mock("$lib/logger", () => ({
  getLogger: () => ({
    debug: debugMock,
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  }),
}));

vi.mock("svelte-spa-router", () => ({
  location: {subscribe: vi.fn(() => () => {})},
}));

type RafCb = (t: number) => void;
let rafQueue: RafCb[] = [];
let nowValue = 0;

type ObserverCb = (list: {getEntries: () => PerformanceEntryLike[]}) => void;
type PerformanceEntryLike = {startTime: number; duration: number};
let observers: {cb: ObserverCb; connected: boolean}[] = [];

class MockPerformanceObserver {
  private cb: ObserverCb;
  constructor(cb: ObserverCb) {
    this.cb = cb;
    observers.push({cb, connected: false});
  }
  observe(): void {
    const o = observers.find((x) => x.cb === this.cb);
    if (o) o.connected = true;
  }
  disconnect(): void {
    const o = observers.find((x) => x.cb === this.cb);
    if (o) o.connected = false;
  }
  static supportedEntryTypes = ["longtask"];
}

function flushRafs(): void {
  const q = rafQueue;
  rafQueue = [];
  for (const cb of q) cb(nowValue);
}

function fireLongtask(startTime: number, duration: number): void {
  nowValue = Math.max(nowValue, startTime + duration);
  for (const o of observers) {
    if (o.connected) {
      o.cb({getEntries: () => [{startTime, duration}]});
    }
  }
}

async function loadModule() {
  return await import("../navTiming");
}

beforeEach(() => {
  vi.resetModules();
  vi.useFakeTimers();
  debugMock.mockReset();
  rafQueue = [];
  observers = [];
  nowValue = 0;

  vi.stubGlobal("requestAnimationFrame", (cb: RafCb) => {
    rafQueue.push(cb);
    return rafQueue.length;
  });
  vi.stubGlobal("PerformanceObserver", MockPerformanceObserver);
  vi.spyOn(performance, "now").mockImplementation(() => nowValue);
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("navTiming", () => {
  test("logs firstPaintMs and ttiMs for a clean nav with no long tasks", async () => {
    const {__test__} = await loadModule();
    nowValue = 1000;
    __test__.start("/clean", 1000);

    nowValue = 1016;
    flushRafs(); // outer rAF
    nowValue = 1032;
    flushRafs(); // inner rAF → records firstPaint

    vi.advanceTimersByTime(500); // quiet window

    expect(debugMock).toHaveBeenCalledTimes(1);
    const [msg, payload] = debugMock.mock.calls[0];
    expect(msg).toBe("nav");
    expect(payload.route).toBe("/clean");
    expect(payload.firstPaintMs).toBe(32);
    expect(payload.ttiMs).toBe(32); // falls back to firstPaintAbsTime
    expect(payload.longTaskCount).toBe(0);
    expect(payload.timedOut).toBeUndefined();
    expect(payload.longtaskSupported).toBeUndefined();
  });

  test("ttiMs reflects last long task inside the quiet window", async () => {
    const {__test__} = await loadModule();
    __test__.start("/heavy", 0);

    nowValue = 20;
    flushRafs();
    nowValue = 40;
    flushRafs(); // firstPaintMs = 40

    fireLongtask(100, 80); // ends at 180, arms quiet window
    vi.advanceTimersByTime(300);
    fireLongtask(400, 120); // ends at 520, resets quiet window
    vi.advanceTimersByTime(500); // quiet window elapses

    expect(debugMock).toHaveBeenCalledTimes(1);
    const payload = debugMock.mock.calls[0][1];
    expect(payload.longTaskCount).toBe(2);
    expect(payload.ttiMs).toBe(520);
    expect(payload.firstPaintMs).toBe(40);
  });

  test("rapid successive navs cancel the earlier measurement", async () => {
    const {__test__} = await loadModule();
    __test__.start("/a", 0);
    nowValue = 50;
    __test__.start("/b", 50);

    nowValue = 70;
    flushRafs();
    nowValue = 90;
    flushRafs();
    vi.advanceTimersByTime(500);

    expect(debugMock).toHaveBeenCalledTimes(1);
    expect(debugMock.mock.calls[0][1].route).toBe("/b");
  });

  test("emits timedOut:true when the 10s ceiling is hit", async () => {
    const {__test__} = await loadModule();
    __test__.start("/slow", 0);

    // Keep resetting the quiet window so only the ceiling fires.
    for (let t = 400; t < 10_000; t += 400) {
      fireLongtask(t, 10);
    }
    vi.advanceTimersByTime(10_000);

    expect(debugMock).toHaveBeenCalledTimes(1);
    const payload = debugMock.mock.calls[0][1];
    expect(payload.timedOut).toBe(true);
    expect(payload.route).toBe("/slow");
  });

  test("marks longtaskSupported:false when PerformanceObserver lacks longtask", async () => {
    (MockPerformanceObserver as unknown as {supportedEntryTypes: string[]}).supportedEntryTypes = [];
    const {__test__} = await loadModule();
    __test__.start("/no-lt", 0);

    nowValue = 30;
    flushRafs();
    nowValue = 60;
    flushRafs();
    vi.advanceTimersByTime(500);

    expect(debugMock).toHaveBeenCalledTimes(1);
    expect(debugMock.mock.calls[0][1].longtaskSupported).toBe(false);

    (MockPerformanceObserver as unknown as {supportedEntryTypes: string[]}).supportedEntryTypes = ["longtask"];
  });
});
```

Notes:

- `vi.resetModules()` in `beforeEach` ensures the module's `bootstrapped` guard is fresh per test, so the auto-run `bootstrap()` on import is harmless (the mocked `location.subscribe` is a no-op).
- The tests drive `__test__.start()` directly; this isolates the measurement logic from the store subscription, which is what we actually care about.

### - [ ] Step 3: Run the tests and iterate until green

Command: `cd frontend && npx vitest run src/lib/__tests__/navTiming.test.ts`

Expected: 5 passing tests. If any fail, fix the implementation — do not weaken the tests.

### - [ ] Step 4: Type-check

Command: `cd frontend && pnpm check`

Expected: no new errors attributable to the new files. If the check surfaces errors, fix the offending types (the test file uses `unknown` casts for the `supportedEntryTypes` override — keep it that way; do not introduce `any`).

---

## Task 2: Wire up in `App.svelte` and verify end-to-end

**Files:**
- Modify: `frontend/src/App.svelte`

### - [ ] Step 1: Add the side-effect import

In `frontend/src/App.svelte`, near the other `$lib` imports at the top of the `<script lang="ts">` block (around line 12, next to `import {descriptorRegistry} from "$lib/registry/index";`), add:

```ts
import "$lib/navTiming";
```

Place it with the other side-effect/store imports. No other changes to `App.svelte`.

### - [ ] Step 2: Build and manually verify

Commands:

```bash
cd frontend && pnpm check
cd frontend && npx vitest run src/lib/__tests__/navTiming.test.ts
```

Then start the app (`task dev` from repo root) and navigate between a couple of routes (e.g. cluster list → a cluster overview → a resource list). Open the DevTools console and confirm log lines shaped like:

```
[klados.nav-timing] nav { route: "/c/<ctx>/apps.v1.deployments", firstPaintMs: 18.40, ttiMs: 312.55, longTaskCount: 3 }
```

Also confirm that the same entries show up in the Go log stream (via the Wails backend slog) alongside other frontend-origin logs.

### - [ ] Step 3: Commit via jj

Use the `jj-vcs` skill. Commit with a message like:

```
frontend: add navigation timing debug log
```

The commit should contain exactly three files: `frontend/src/lib/navTiming.ts`, `frontend/src/lib/__tests__/navTiming.test.ts`, and the one-line change to `frontend/src/App.svelte`. Do not bundle this with any unrelated working-copy changes.

---

## Self-review

- **Spec coverage.** Every behavior in the spec (subscribe to `location`, cancel in-flight, double-rAF first-paint, longtask observer + 500ms quiet window, 10s ceiling, `longtaskSupported` fallback, log shape including `route`/`firstPaintMs`/`ttiMs`/`longTaskCount`/`longtaskSupported`/`timedOut`, 2-decimal rounding, single log per nav, logger via `getLogger`) maps to code in Task 1 and is exercised by at least one test. Initial-load `navStart = 0` is built into `bootstrap()`'s `first` flag; it is not exercised by a dedicated test because it is a trivial two-line branch and the module's tests call `start()` directly.
- **Placeholder scan.** No TBD/TODO. All code is complete. Commands are concrete.
- **Type consistency.** Method names (`start`, `emit`, `cleanup`, `armQuietWindow`, `bootstrap`) are consistent between the module and the test. The `__test__` export surface used in the test matches what the module exports.
