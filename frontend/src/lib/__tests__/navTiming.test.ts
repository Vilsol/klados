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

type PerformanceEntryLike = {startTime: number; duration: number};
type ObserverCb = (list: {getEntries: () => PerformanceEntryLike[]}) => void;
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
  static supportedEntryTypes: string[] = ["longtask"];
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
  MockPerformanceObserver.supportedEntryTypes = ["longtask"];

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
    flushRafs();
    nowValue = 1032;
    flushRafs();

    vi.advanceTimersByTime(500);

    expect(debugMock).toHaveBeenCalledTimes(1);
    const [msg, payload] = debugMock.mock.calls[0];
    expect(msg).toBe("nav");
    expect(payload.route).toBe("/clean");
    expect(payload.firstPaintMs).toBe(32);
    expect(payload.ttiMs).toBe(32);
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
    flushRafs();

    fireLongtask(100, 80);
    vi.advanceTimersByTime(300);
    fireLongtask(400, 120);
    vi.advanceTimersByTime(500);

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

    for (let t = 400; t < 10_000; t += 400) {
      vi.advanceTimersByTime(400);
      fireLongtask(t, 10);
    }
    vi.advanceTimersByTime(1000);

    expect(debugMock).toHaveBeenCalledTimes(1);
    const payload = debugMock.mock.calls[0][1];
    expect(payload.timedOut).toBe(true);
    expect(payload.route).toBe("/slow");
  });

  test("marks longtaskSupported:false when PerformanceObserver lacks longtask", async () => {
    MockPerformanceObserver.supportedEntryTypes = [];
    const {__test__} = await loadModule();
    __test__.start("/no-lt", 0);

    nowValue = 30;
    flushRafs();
    nowValue = 60;
    flushRafs();
    vi.advanceTimersByTime(500);

    expect(debugMock).toHaveBeenCalledTimes(1);
    expect(debugMock.mock.calls[0][1].longtaskSupported).toBe(false);
  });
});
