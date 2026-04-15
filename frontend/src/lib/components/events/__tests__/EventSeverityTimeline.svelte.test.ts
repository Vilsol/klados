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

  it("fires onBrush once after mousedown+mouseup on svg", async () => {
    const spy = vi.fn();
    const {container} = render(EventSeverityTimeline, {
      props: {filteredItems: [], allItems: [], rangeMs, now, selectedWindow: null, onBrush: spy},
    });
    const svg = container.querySelector("svg") as SVGSVGElement;

    // Mock getBoundingClientRect so bucketIndexFromEvent returns a valid index
    vi.spyOn(svg, "getBoundingClientRect").mockReturnValue({
      left: 0, top: 0, right: 240, bottom: 40, width: 240, height: 40,
      x: 0, y: 0, toJSON: () => {},
    } as DOMRect);

    // Click in the middle of the SVG — both events at the same x → single bucket
    await fireEvent.mouseDown(svg, {clientX: 20, clientY: 20});
    await fireEvent.mouseUp(svg, {clientX: 20, clientY: 20});

    expect(spy).toHaveBeenCalledTimes(1);
    const arg = spy.mock.calls[0][0];
    expect(typeof arg.from).toBe("number");
    expect(typeof arg.to).toBe("number");
    expect(arg.from).toBeLessThan(arg.to);
  });

  it("does not fire onBrush without any mouse interaction", () => {
    const spy = vi.fn();
    render(EventSeverityTimeline, {
      props: {filteredItems: [], allItems: [], rangeMs, now, selectedWindow: null, onBrush: spy},
    });
    expect(spy).not.toHaveBeenCalled();
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
