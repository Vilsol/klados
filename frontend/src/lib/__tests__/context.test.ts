import {describe, it, expect, vi} from "vitest";
import {createPluginContext} from "$lib/plugins/context.js";
import type {PluginManifest} from "$lib/plugins/types/manifest.js";
import type {HostServices} from "$lib/plugins/context.js";

const makeHost = (): HostServices => ({
  clusterName: "test-cluster",
  clusterVersion: "1.29.0",
  namespace: "default",
  listResources: vi.fn().mockResolvedValue([]),
  getResource: vi.fn().mockResolvedValue({}),
});

const manifestWithResources: PluginManifest = {
  schemaVersion: 1,
  name: "my-plugin",
  version: "0.1.0",
  displayName: "My Plugin",
  minHostVersion: "0.1.0",
  permissions: {
    resources: [{group: "apps", version: "v1", resource: "deployments", verbs: ["list", "get"]}],
  },
};

const manifestNoPermissions: PluginManifest = {
  schemaVersion: 1,
  name: "minimal",
  version: "0.1.0",
  displayName: "Minimal",
  minHostVersion: "0.1.0",
};

describe("createPluginContext", () => {
  it("always includes cluster and namespace", () => {
    const ctx = createPluginContext(manifestNoPermissions, makeHost());
    expect(ctx.cluster.name).toBe("test-cluster");
    expect(ctx.namespace).toBe("default");
  });

  it("attaches k8s when resources declared", () => {
    const ctx = createPluginContext(manifestWithResources, makeHost());
    expect(ctx.k8s).toBeDefined();
  });

  it("omits k8s when no resources declared", () => {
    const ctx = createPluginContext(manifestNoPermissions, makeHost());
    expect(ctx.k8s).toBeUndefined();
  });

  it("returns a frozen context", () => {
    const ctx = createPluginContext(manifestWithResources, makeHost());
    expect(() => {
      (ctx as any).newProp = "bad";
    }).toThrow();
  });

  it("k8s.list delegates to host after permission check", async () => {
    const host = makeHost();
    const ctx = createPluginContext(manifestWithResources, host);
    await ctx.k8s!.list("apps.v1.deployments" as any);
    expect(host.listResources).toHaveBeenCalledWith("apps.v1.deployments", undefined);
  });

  it("k8s.list throws for unpermitted GVR", () => {
    const ctx = createPluginContext(manifestWithResources, makeHost());
    expect(() => (ctx.k8s!.list as any)("core.v1.pods")).toThrow();
  });

  it("k8s context is frozen", () => {
    const ctx = createPluginContext(manifestWithResources, makeHost());
    expect(() => {
      (ctx.k8s as any).newProp = "bad";
    }).toThrow();
  });
});
