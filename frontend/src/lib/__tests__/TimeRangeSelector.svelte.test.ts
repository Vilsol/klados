import {describe, it, expect, vi} from "vitest";
import {render, screen, fireEvent} from "@testing-library/svelte";
import TimeRangeSelector from "$lib/components/charts/TimeRangeSelector.svelte";

describe("TimeRangeSelector", () => {
  it("is hidden when hasPrometheus is false", () => {
    const {container} = render(TimeRangeSelector, {
      props: {value: 15, onchange: vi.fn(), hasPrometheus: false},
    });
    expect(container.querySelector("button")).toBeNull();
  });

  it("renders preset buttons when hasPrometheus is true", () => {
    render(TimeRangeSelector, {
      props: {value: 15, onchange: vi.fn(), hasPrometheus: true},
    });
    expect(screen.getByText("15m")).toBeTruthy();
    expect(screen.getByText("1h")).toBeTruthy();
    expect(screen.getByText("6h")).toBeTruthy();
    expect(screen.getByText("24h")).toBeTruthy();
    expect(screen.getByText("7d")).toBeTruthy();
  });

  it.each([
    ["15m", 15],
    ["1h", 60],
    ["6h", 360],
    ["24h", 1440],
    ["7d", 10_080],
  ])("emits %s → %i on click", (label, expected) => {
    const onchange = vi.fn();
    render(TimeRangeSelector, {
      props: {value: 15, onchange, hasPrometheus: true},
    });
    fireEvent.click(screen.getByText(label));
    expect(onchange).toHaveBeenCalledWith(expected);
  });
});
