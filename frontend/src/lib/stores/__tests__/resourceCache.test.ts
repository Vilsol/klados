import { describe, it, expect, beforeEach } from "vitest";
import { SvelteMap } from "svelte/reactivity";
import { resourceCache } from "../resourceCache.svelte";

describe("resourceCache", () => {
  beforeEach(() => {
    (resourceCache as any).cache = new SvelteMap();
  });

  it("upsert + findByOwnerUID", () => {
    resourceCache.upsert("c", "apps.v1.replicasets", {
      metadata: { uid: "rs-1", ownerReferences: [{ uid: "deploy-1", kind: "Deployment" }] },
    });
    resourceCache.upsert("c", "core.v1.pods", {
      metadata: { uid: "pod-1", ownerReferences: [{ uid: "rs-1", kind: "ReplicaSet" }] },
    });
    resourceCache.upsert("c", "core.v1.pods", {
      metadata: { uid: "pod-2", ownerReferences: [{ uid: "other", kind: "ReplicaSet" }] },
    });

    const byDeploy = resourceCache.findByOwnerUID("c", "deploy-1");
    expect(byDeploy.length).toBe(1);
    expect(byDeploy[0].gvr).toBe("apps.v1.replicasets");
    expect(byDeploy[0].items.length).toBe(1);

    const byRS = resourceCache.findByOwnerUID("c", "rs-1");
    expect(byRS.length).toBe(1);
    expect(byRS[0].items.length).toBe(1);
    expect((byRS[0].items[0] as any).metadata.uid).toBe("pod-1");
  });

  it("remove evicts object", () => {
    resourceCache.upsert("c", "core.v1.pods", {
      metadata: { uid: "pod-1", ownerReferences: [{ uid: "rs-1" }] },
    });
    resourceCache.remove("c", "core.v1.pods", "pod-1");
    expect(resourceCache.findByOwnerUID("c", "rs-1")).toEqual([]);
  });
});
