import { describe, it, expect } from "vitest";
import { getOwnerReferences, gvrFromAPIVersion } from "../owners";

describe("getOwnerReferences", () => {
  it("returns [] when missing", () => {
    expect(getOwnerReferences({})).toEqual([]);
    expect(getOwnerReferences({ metadata: {} })).toEqual([]);
  });

  it("returns owner references list", () => {
    const obj = {
      metadata: {
        ownerReferences: [
          { apiVersion: "apps/v1", kind: "ReplicaSet", name: "rs-1", uid: "uid-1", controller: true },
        ],
      },
    };
    const o = getOwnerReferences(obj);
    expect(o.length).toBe(1);
    expect(o[0].kind).toBe("ReplicaSet");
  });
});

describe("gvrFromAPIVersion", () => {
  it("handles core group", () => {
    expect(gvrFromAPIVersion("v1", "Pod")).toBe("core.v1.pods");
  });
  it("handles named group", () => {
    expect(gvrFromAPIVersion("apps/v1", "Deployment")).toBe("apps.v1.deployments");
  });
  it("lowercases and plural-izes simple kinds", () => {
    expect(gvrFromAPIVersion("v1", "Service")).toBe("core.v1.services");
    expect(gvrFromAPIVersion("v1", "ConfigMap")).toBe("core.v1.configmaps");
    expect(gvrFromAPIVersion("policy/v1", "PodDisruptionBudget")).toBe("policy.v1.poddisruptionbudgets");
  });
});
