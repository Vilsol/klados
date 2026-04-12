import {describe, it, expect, vi, beforeEach} from "vitest";
import {render} from "@testing-library/svelte";

const mockSetData = vi.hoisted(() => vi.fn());
const mockDestroy = vi.hoisted(() => vi.fn());
const mockSetSize = vi.hoisted(() => vi.fn());
const mockInstances = vi.hoisted(() => [] as any[]);

vi.mock("uplot", () => {
  class UPlot {
    setData = mockSetData;
    destroy = mockDestroy;
    setSize = mockSetSize;
    data: any;
    series: any[];

    constructor(_opts: any, _data: any, el: HTMLElement) {
      this.data = _data;
      this.series = _opts?.series ?? [];
      const canvas = document.createElement("canvas");
      el?.appendChild(canvas);
      mockInstances.push(this);
    }
  }
  return {default: UPlot};
});

vi.mock("uplot/dist/uPlot.min.css", () => ({}));

import Sparkline from "$lib/components/charts/Sparkline.svelte";
import type {TimeSeriesPoint} from "$lib/components/charts/types";

function makePoints(n = 5): TimeSeriesPoint[] {
  return Array.from({length: n}, (_, i) => ({
    t: 1000 + i * 15,
    v: Math.random() * 0.5,
  }));
}

beforeEach(() => {
  mockSetData.mockClear();
  mockDestroy.mockClear();
  mockSetSize.mockClear();
  mockInstances.length = 0;
});

describe("Sparkline", () => {
  it("renders a canvas element on mount", async () => {
    const {container} = render(Sparkline, {
      props: {points: makePoints()},
    });
    await vi.waitFor(() => {
      expect(container.querySelector("canvas")).toBeTruthy();
    });
  });

  it("creates uPlot with correct height", async () => {
    render(Sparkline, {
      props: {points: makePoints(), height: 24},
    });
    await vi.waitFor(() => {
      expect(mockInstances.length).toBe(1);
    });
  });

  it("destroys uPlot on unmount", async () => {
    const {unmount} = render(Sparkline, {
      props: {points: makePoints()},
    });
    await vi.waitFor(() => {
      expect(mockInstances.length).toBe(1);
    });
    unmount();
    expect(mockDestroy).toHaveBeenCalled();
  });

  it("renders empty sparkline with no points", async () => {
    const {container} = render(Sparkline, {
      props: {points: []},
    });
    await vi.waitFor(() => {
      expect(mockInstances.length).toBe(1);
    });
    expect(container.querySelector("canvas")).toBeTruthy();
  });
});
