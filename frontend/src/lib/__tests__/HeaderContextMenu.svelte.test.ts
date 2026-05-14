import {describe, it, expect, vi} from "vitest";
import {render, screen, fireEvent} from "@testing-library/svelte";
import HeaderContextMenu from "$lib/components/HeaderContextMenu.svelte";

function props(overrides: Record<string, unknown> = {}) {
  return {
    x: 100,
    y: 100,
    columnName: "Status",
    isPinned: false,
    canHide: true,
    onSort: vi.fn(),
    onAutoFit: vi.fn(),
    onTogglePin: vi.fn(),
    onHide: vi.fn(),
    onClose: vi.fn(),
    ...overrides,
  };
}

describe("HeaderContextMenu", () => {
  it("renders Sort asc/desc, Auto-fit, Pin, Hide for a normal column", () => {
    render(HeaderContextMenu, {props: props()});
    expect(screen.getByText(/sort ascending/i)).toBeTruthy();
    expect(screen.getByText(/sort descending/i)).toBeTruthy();
    expect(screen.getByText(/auto-?fit/i)).toBeTruthy();
    expect(screen.getByText(/pin to left/i)).toBeTruthy();
    expect(screen.getByText(/hide column/i)).toBeTruthy();
  });

  it("shows Unpin when isPinned=true and never offers Hide when canHide=false", () => {
    render(HeaderContextMenu, {props: props({isPinned: true, canHide: false})});
    expect(screen.getByText(/unpin/i)).toBeTruthy();
    expect(screen.queryByText(/hide column/i)).toBeNull();
  });

  it("calls onSort with 'asc' / 'desc'", async () => {
    const p = props();
    render(HeaderContextMenu, {props: p});
    await fireEvent.click(screen.getByText(/sort ascending/i));
    expect(p.onSort).toHaveBeenCalledWith("asc");
    await fireEvent.click(screen.getByText(/sort descending/i));
    expect(p.onSort).toHaveBeenCalledWith("desc");
  });

  it("calls onTogglePin and onClose on pin click", async () => {
    const p = props();
    render(HeaderContextMenu, {props: p});
    await fireEvent.click(screen.getByText(/pin to left/i));
    expect(p.onTogglePin).toHaveBeenCalled();
    expect(p.onClose).toHaveBeenCalled();
  });
});
