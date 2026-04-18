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

export const __test__ = {
  start,
  emit,
  cleanup,
  get current() {
    return current;
  },
};
