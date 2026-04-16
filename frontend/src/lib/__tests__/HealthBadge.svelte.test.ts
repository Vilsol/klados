import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/svelte";
import HealthBadge from "../components/HealthBadge.svelte";

describe("HealthBadge", () => {
  it("shows green dot for healthy", () => {
    const obj = { status: { conditions: [{ type: "Ready", status: "True" }] } };
    const { container } = render(HealthBadge, { props: { obj } });
    expect(container.querySelector(".bg-emerald-500")).toBeTruthy();
  });

  it("shows red dot for unhealthy", () => {
    const obj = { status: { conditions: [{ type: "Ready", status: "False" }] } };
    const { container } = render(HealthBadge, { props: { obj } });
    expect(container.querySelector(".bg-destructive")).toBeTruthy();
  });

  it("shows ratio for unrecognized conditions", () => {
    const obj = { status: { conditions: [{ type: "A", status: "True" }, { type: "B", status: "False" }] } };
    render(HealthBadge, { props: { obj } });
    expect(screen.getByText("1/2 True")).toBeTruthy();
  });

  it("renders nothing when no conditions", () => {
    const { container } = render(HealthBadge, { props: { obj: {} } });
    expect(container.textContent?.trim()).toBe("");
  });
});
