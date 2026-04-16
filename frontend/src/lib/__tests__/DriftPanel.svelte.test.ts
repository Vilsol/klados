import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/svelte";

vi.mock("@klados/ui", () => ({
  DiffView: vi.fn(),
}));

import DriftPanel from "../components/panels/DriftPanel.svelte";
import { LAST_APPLIED_ANNOTATION } from "../kubernetes/metadata";

describe("DriftPanel", () => {
  it("shows empty state when annotation missing", () => {
    render(DriftPanel, { props: { obj: {} } });
    expect(screen.getAllByText((_, el) => el?.tagName === "CODE" && el.textContent === "last-applied-configuration").length).toBeGreaterThan(0);
  });

  it("does not show empty state when annotation present", () => {
    const obj = {
      metadata: {
        annotations: { [LAST_APPLIED_ANNOTATION]: '{"spec":{"x":1}}' },
      },
      spec: { x: 1 },
    };
    render(DriftPanel, { props: { obj } });
    expect(screen.queryByText(/No.*last-applied/)).toBeNull();
  });
});
