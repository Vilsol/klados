import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, screen } from "@testing-library/svelte";
import EventToolbar from "../EventToolbar.svelte";

vi.mock("@klados/ui", () => {
  const Stub = () => ({});
  return { Combobox: Stub };
});

function baseProps(overrides = {}) {
  return {
    showWarning: true,
    showNormal: true,
    onSeverityChange: vi.fn(),
    availableKinds: ["Pod", "Deployment"],
    selectedKinds: [],
    onKindsChange: vi.fn(),
    availableReasons: ["Scheduled", "BackOff"],
    selectedReasons: [],
    onReasonsChange: vi.fn(),
    search: "",
    onSearchChange: vi.fn(),
    grouped: false,
    onGroupedChange: vi.fn(),
    paused: false,
    onJumpToLatest: vi.fn(),
    totalCount: 10,
    warningCount: 2,
    rangeLabel: "last 30m",
    columnMenuOpen: false,
    onColumnMenuToggle: vi.fn(),
    timeWindow: null as null | { from: number; to: number },
    onClearTimeWindow: vi.fn(),
    ...overrides,
  };
}

describe("EventToolbar", () => {
  it("toggling Warning pill emits onSeverityChange with showWarning flipped", async () => {
    const props = baseProps();
    render(EventToolbar, { props });
    await fireEvent.click(screen.getByText("Warning"));
    expect(props.onSeverityChange).toHaveBeenCalledWith({ showWarning: false, showNormal: true });
  });

  it("typing in the search input fires onSearchChange", async () => {
    const props = baseProps();
    render(EventToolbar, { props });
    const input = screen.getByPlaceholderText(/Search/i) as HTMLInputElement;
    await fireEvent.input(input, { target: { value: "oom" } });
    expect(props.onSearchChange).toHaveBeenCalledWith("oom");
  });

  it("Jump to latest is visible only when paused", () => {
    const { rerender } = render(EventToolbar, { props: baseProps({ paused: false }) });
    expect(screen.queryByTestId("jump-to-latest")).toBeNull();
    rerender(baseProps({ paused: true }));
    expect(screen.getByTestId("jump-to-latest")).toBeTruthy();
  });

  it("Jump to latest button fires onJumpToLatest", async () => {
    const props = baseProps({ paused: true });
    render(EventToolbar, { props });
    await fireEvent.click(screen.getByTestId("jump-to-latest"));
    expect(props.onJumpToLatest).toHaveBeenCalledTimes(1);
  });

  it("time-window chip × fires onClearTimeWindow", async () => {
    const props = baseProps({ timeWindow: { from: Date.now() - 60_000, to: Date.now() } });
    render(EventToolbar, { props });
    await fireEvent.click(screen.getByTestId("time-window-chip"));
    expect(props.onClearTimeWindow).toHaveBeenCalledTimes(1);
  });

  it("column-menu button fires onColumnMenuToggle", async () => {
    const props = baseProps();
    render(EventToolbar, { props });
    await fireEvent.click(screen.getByTestId("column-menu-button"));
    expect(props.onColumnMenuToggle).toHaveBeenCalledTimes(1);
  });

  it("grouped toggle fires onGroupedChange with flipped value", async () => {
    const props = baseProps({ grouped: false });
    render(EventToolbar, { props });
    await fireEvent.click(screen.getByTestId("grouped-toggle"));
    expect(props.onGroupedChange).toHaveBeenCalledWith(true);
  });

  it("shows count badge text", () => {
    render(EventToolbar, { props: baseProps({ totalCount: 42, warningCount: 7, rangeLabel: "last 30m" }) });
    expect(screen.getByText(/42 events/)).toBeTruthy();
    expect(screen.getByText(/7 warnings/)).toBeTruthy();
    expect(screen.getByText(/last 30m/)).toBeTruthy();
  });
});
