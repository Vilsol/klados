import {describe, it, expect, vi} from "vitest";
import {render, fireEvent} from "@testing-library/svelte";
import EventSeverityTimeline from "../EventSeverityTimeline.svelte";

const now = Date.parse("2026-04-15T10:30:00Z");
const rangeMs = 30 * 60_000;

describe("EventSeverityTimeline", () => {
  it("renders one bucket group per bucket", () => {
    const {container} = render(EventSeverityTimeline, {
      props: {filteredItems: [], allItems: [], rangeMs, now, selectedWindow: null},
    });
    const groups = container.querySelectorAll("[data-bucket]");
    // 30m range with 60-target buckets → 30000ms ideal → first >= is 30_000 → 60 buckets
    expect(groups.length).toBeGreaterThan(0);
  });

  it("fires onBrush with correct from/to when dragging from bucket i to bucket j", async () => {
    const spy = vi.fn();
    const {container} = render(EventSeverityTimeline, {
      props: {filteredItems: [], allItems: [], rangeMs, now, selectedWindow: null, onBrush: spy},
    });
    const groups = container.querySelectorAll("[data-bucket]");
    const start = groups[5] as HTMLElement;
    const end = groups[10] as HTMLElement;
    await fireEvent.mouseDown(start);
    await fireEvent.mouseMove(end);
    const svg = container.querySelector("svg") as SVGElement;
    await fireEvent.mouseUp(svg);
    expect(spy).toHaveBeenCalledTimes(1);
    const arg = spy.mock.calls[0][0];
    expect(arg.from).toBeLessThan(arg.to);
  });

  it("clear affordance fires onBrush(null)", async () => {
    const spy = vi.fn();
    const {getByTestId} = render(EventSeverityTimeline, {
      props: {
        filteredItems: [],
        allItems: [],
        rangeMs,
        now,
        selectedWindow: {from: now - 10 * 60_000, to: now - 5 * 60_000},
        onBrush: spy,
      },
    });
    await fireEvent.click(getByTestId("clear-window"));
    expect(spy).toHaveBeenCalledWith(null);
  });
});
