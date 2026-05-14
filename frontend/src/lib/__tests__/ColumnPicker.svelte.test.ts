import {describe, it, expect, vi} from "vitest";
import {render, screen, fireEvent} from "@testing-library/svelte";
import ColumnPicker from "$lib/components/ColumnPicker.svelte";

const col = (name: string) => ({name});

function baseProps(overrides: Record<string, unknown> = {}) {
  return {
    allColumns: [
      {col: col("Name"), visible: true},
      {col: col("Ready"), visible: true},
      {col: col("Status"), visible: true},
      {col: col("Restarts"), visible: false},
      {col: col("IP"), visible: false},
    ],
    visibleColumns: [col("Name"), col("Ready"), col("Status")],
    pinnedNames: ["Name"],
    onToggle: vi.fn(),
    onReset: vi.fn(),
    ...overrides,
  };
}

describe("ColumnPicker", () => {
  it("renders all columns, pinned first, with pinned checkbox disabled", () => {
    render(ColumnPicker, {props: baseProps()});
    const items = screen.getAllByRole("checkbox") as HTMLInputElement[];
    expect(items[0].disabled).toBe(true);
    expect(items[1].checked).toBe(true);
    expect(items[2].checked).toBe(true);
    expect(items[3].checked).toBe(false);
  });

  it("filters by name (case-insensitive substring)", async () => {
    render(ColumnPicker, {props: baseProps()});
    const input = screen.getByPlaceholderText("Filter…");
    await fireEvent.input(input, {target: {value: "re"}});
    expect(screen.queryByText("Name")).toBeNull();
    expect(screen.getByText("Ready")).toBeTruthy();
    expect(screen.getByText("Restarts")).toBeTruthy();
    expect(screen.queryByText("Status")).toBeNull();
  });

  it("calls onToggle when a non-pinned checkbox flips", async () => {
    const props = baseProps();
    render(ColumnPicker, {props});
    const restartsCheckbox = screen.getAllByRole("checkbox")[3];
    await fireEvent.click(restartsCheckbox);
    expect(props.onToggle).toHaveBeenCalledWith("Restarts", true);
  });

  it("calls onReset when the Reset button is clicked", async () => {
    const props = baseProps();
    render(ColumnPicker, {props});
    await fireEvent.click(screen.getByText("Reset"));
    expect(props.onReset).toHaveBeenCalled();
  });
});
