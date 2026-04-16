import {describe, it, expect, beforeEach, vi} from "vitest";
import {render, screen} from "@testing-library/svelte";
import {SvelteMap} from "svelte/reactivity";
import RelatedResourcesPanel from "../components/panels/RelatedResourcesPanel.svelte";
import {resourceCache} from "../stores/resourceCache.svelte";

vi.mock("svelte-spa-router", () => ({push: vi.fn()}));

describe("RelatedResourcesPanel", () => {
  beforeEach(() => {
    (resourceCache as any).cache = new SvelteMap();
  });

  it("shows empty state when no related resources in cache", () => {
    render(RelatedResourcesPanel, {
      props: {
        contextName: "c",
        obj: {metadata: {uid: "x"}},
      },
    });
    expect(screen.getByText(/No related resources/)).toBeTruthy();
  });

  it("groups related items by GVR", () => {
    resourceCache.upsert("c", "core.v1.pods", {
      metadata: {uid: "p1", name: "p1", namespace: "default", ownerReferences: [{uid: "owner", kind: "ReplicaSet"}]},
    });
    resourceCache.upsert("c", "core.v1.pods", {
      metadata: {uid: "p2", name: "p2", namespace: "default", ownerReferences: [{uid: "owner", kind: "ReplicaSet"}]},
    });
    render(RelatedResourcesPanel, {
      props: {
        contextName: "c",
        obj: {metadata: {uid: "owner"}},
      },
    });
    expect(screen.getByText(/core\.v1\.pods \(2\)/)).toBeTruthy();
    expect(screen.getByText("default/p1")).toBeTruthy();
    expect(screen.getByText("default/p2")).toBeTruthy();
  });
});
