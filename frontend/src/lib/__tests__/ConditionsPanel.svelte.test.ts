import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/svelte";
import ConditionsPanel from "../components/panels/ConditionsPanel.svelte";

describe("ConditionsPanel", () => {
  it("shows empty state when object has no conditions", () => {
    render(ConditionsPanel, { props: { obj: { status: {} } } });
    expect(screen.getByText(/No conditions reported/i)).toBeTruthy();
  });

  it("renders each condition as a row with colored status badge", () => {
    const obj = {
      status: {
        conditions: [
          { type: "Ready", status: "True", reason: "Ok", message: "all good", lastTransitionTime: "2026-04-16T12:00:00Z" },
          { type: "Degraded", status: "False", reason: "", message: "", lastTransitionTime: "2026-04-16T12:00:00Z" },
        ],
      },
    };
    render(ConditionsPanel, { props: { obj } });
    expect(screen.getByText("Ready")).toBeTruthy();
    expect(screen.getByText("Degraded")).toBeTruthy();
    expect(screen.getByText("all good")).toBeTruthy();
  });
});
