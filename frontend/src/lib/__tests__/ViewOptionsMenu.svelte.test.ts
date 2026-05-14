import {describe, it, expect, vi} from "vitest";
import {render, screen, fireEvent} from "@testing-library/svelte";
import ViewOptionsMenu from "$lib/components/ViewOptionsMenu.svelte";

describe("ViewOptionsMenu", () => {
  it("renders compact toggle and calls onCompactChange", async () => {
    const onCompactChange = vi.fn();
    render(ViewOptionsMenu, {props: {compact: false, onCompactChange, hasSparklines: false}});
    const toggle = screen.getByLabelText("Compact rows") as HTMLInputElement;
    await fireEvent.click(toggle);
    expect(onCompactChange).toHaveBeenCalledWith(true);
  });

  it("does not render sparkline toggles when hasSparklines is false", () => {
    render(ViewOptionsMenu, {props: {compact: false, onCompactChange: vi.fn(), hasSparklines: false}});
    expect(screen.queryByLabelText("CPU")).toBeNull();
    expect(screen.queryByLabelText("Memory")).toBeNull();
  });

  it("renders sparkline toggles when hasSparklines is true", () => {
    render(ViewOptionsMenu, {
      props: {
        compact: false,
        onCompactChange: vi.fn(),
        hasSparklines: true,
        sparklineColumns: [],
        onSparklineToggle: vi.fn(),
      },
    });
    expect(screen.getByLabelText("CPU")).toBeTruthy();
    expect(screen.getByLabelText("Memory")).toBeTruthy();
  });

  it("toggles a sparkline column on click", async () => {
    const onSparklineToggle = vi.fn();
    render(ViewOptionsMenu, {
      props: {
        compact: false,
        onCompactChange: vi.fn(),
        hasSparklines: true,
        sparklineColumns: [],
        onSparklineToggle,
      },
    });
    await fireEvent.click(screen.getByLabelText("CPU"));
    expect(onSparklineToggle).toHaveBeenCalledWith(["CPU"]);
  });
});
