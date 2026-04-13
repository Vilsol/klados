import {describe, it, expect, vi, beforeEach} from "vitest";
import {render, screen, fireEvent} from "@testing-library/svelte";

import ColumnMenu from "$lib/components/ColumnMenu.svelte";

const col = (name: string) => ({name});

const visibleCols = [col("Name"), col("Namespace"), col("Ready"), col("Age")];
const allCols = [
  {col: col("Name"), visible: true},
  {col: col("Namespace"), visible: true},
  {col: col("Ready"), visible: true},
  {col: col("Age"), visible: true},
  {col: col("Status"), visible: false},
];

function baseProps(overrides: Record<string, unknown> = {}) {
  return {
    visibleColumns: visibleCols,
    allColumns: allCols,
    compact: false,
    onToggle: vi.fn(),
    onMove: vi.fn(),
    onReset: vi.fn(),
    onCompactChange: vi.fn(),
    ...overrides,
  };
}

describe("ColumnMenu", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders all columns", () => {
    render(ColumnMenu, {props: baseProps()});
    expect(screen.getByText("Name")).toBeTruthy();
    expect(screen.getByText("Namespace")).toBeTruthy();
    expect(screen.getByText("Ready")).toBeTruthy();
    expect(screen.getByText("Age")).toBeTruthy();
    expect(screen.getByText("Status")).toBeTruthy();
  });

  it("Name column checkbox is disabled and checked", () => {
    render(ColumnMenu, {props: baseProps()});
    const checkboxes = screen.getAllByRole("checkbox");
    expect((checkboxes[0] as HTMLInputElement).disabled).toBe(true);
    expect((checkboxes[0] as HTMLInputElement).checked).toBe(true);
  });

  it("toggling visibility calls onToggle", async () => {
    const onToggle = vi.fn();
    render(ColumnMenu, {props: baseProps({onToggle})});
    const checkboxes = screen.getAllByRole("checkbox");
    await fireEvent.click(checkboxes[1]);
    expect(onToggle).toHaveBeenCalledWith("Namespace", expect.any(Boolean));
  });

  it("up button is disabled for the second visible column (cannot move above Name)", () => {
    render(ColumnMenu, {props: baseProps()});
    const upBtn = screen.getByRole("button", {name: "Move Namespace up"});
    expect((upBtn as HTMLButtonElement).disabled).toBe(true);
  });

  it("down button is disabled for the last visible column", () => {
    render(ColumnMenu, {props: baseProps()});
    const downBtn = screen.getByRole("button", {name: "Move Age down"});
    expect((downBtn as HTMLButtonElement).disabled).toBe(true);
  });

  it("reset button calls onReset", async () => {
    const onReset = vi.fn();
    render(ColumnMenu, {props: baseProps({onReset})});
    const resetBtn = screen.getByRole("button", {name: /^reset$/i});
    await fireEvent.click(resetBtn);
    expect(onReset).toHaveBeenCalled();
  });

  it("sparkline section is hidden when GVR is not in sparklineGvrs", () => {
    render(ColumnMenu, {
      props: baseProps({
        gvr: "core.v1.configmaps",
        sparklineGvrs: ["core.v1.pods"],
        sparklineColumns: [],
      }),
    });
    expect(screen.queryByText("Sparklines")).toBeNull();
    expect(screen.queryByText("CPU")).toBeNull();
    expect(screen.queryByText("Memory")).toBeNull();
  });

  it("sparkline section appears when GVR is in sparklineGvrs", () => {
    render(ColumnMenu, {
      props: baseProps({
        gvr: "core.v1.pods",
        sparklineGvrs: ["core.v1.pods"],
        sparklineColumns: [],
      }),
    });
    expect(screen.getByText("Sparklines")).toBeTruthy();
    expect(screen.getByText("CPU")).toBeTruthy();
    expect(screen.getByText("Memory")).toBeTruthy();
  });
});
