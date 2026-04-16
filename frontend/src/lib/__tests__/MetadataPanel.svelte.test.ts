import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/svelte";
import MetadataPanel from "../components/panels/MetadataPanel.svelte";
import { LAST_APPLIED_ANNOTATION } from "../kubernetes/metadata";

describe("MetadataPanel", () => {
  it("renders labels and annotations", () => {
    const obj = {
      metadata: {
        labels: { app: "x" },
        annotations: { foo: "bar" },
      },
    };
    render(MetadataPanel, { props: { obj } });
    expect(screen.getByText("app=x")).toBeTruthy();
    expect(screen.getByText("foo")).toBeTruthy();
    expect(screen.getByText("bar")).toBeTruthy();
  });

  it("excludes last-applied annotation", () => {
    const obj = {
      metadata: {
        annotations: { [LAST_APPLIED_ANNOTATION]: "{}", other: "visible" },
      },
    };
    render(MetadataPanel, { props: { obj } });
    expect(screen.queryByText(LAST_APPLIED_ANNOTATION)).toBeNull();
    expect(screen.getByText("other")).toBeTruthy();
  });
});
