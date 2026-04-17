import { describe, it, expect } from "vitest";
import { render } from "@testing-library/svelte";
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

  it("renders nothing when only unrecognized conditions", () => {
    const obj = { status: { conditions: [{ type: "A", status: "True" }, { type: "B", status: "False" }] } };
    const { container } = render(HealthBadge, { props: { obj } });
    expect(container.textContent?.trim()).toBe("");
  });

  it("renders nothing when no conditions", () => {
    const { container } = render(HealthBadge, { props: { obj: {} } });
    expect(container.textContent?.trim()).toBe("");
  });

  it("shows green for stable Deployment (Available=True + Progressing=True)", () => {
    const obj = {
      status: {
        conditions: [
          { type: "Available", status: "True" },
          { type: "Progressing", status: "True" },
        ],
      },
    };
    const { container } = render(HealthBadge, { props: { obj } });
    expect(container.querySelector(".bg-emerald-500")).toBeTruthy();
  });

  it("shows green for Succeeded pod despite Ready=False", () => {
    const obj = {
      status: {
        phase: "Succeeded",
        conditions: [{ type: "Ready", status: "False" }],
      },
    };
    const { container } = render(HealthBadge, { props: { obj } });
    expect(container.querySelector(".bg-emerald-500")).toBeTruthy();
  });
});
