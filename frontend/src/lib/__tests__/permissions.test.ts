import {describe, it, expect} from "vitest";
import {assertGVRPermission} from "$lib/plugins/permissions.js";
import type {PluginManifest} from "$lib/plugins/types/manifest.js";

const manifest: PluginManifest = {
  schemaVersion: 1,
  name: "test-plugin",
  version: "0.1.0",
  displayName: "Test Plugin",
  minHostVersion: "0.1.0",
  permissions: {
    resources: [
      {group: "apps", version: "v1", resource: "deployments", verbs: ["list", "get"]},
      {group: "", version: "v1", resource: "pods", verbs: ["list", "get", "watch"]},
    ],
  },
};

describe("assertGVRPermission", () => {
  it("allows permitted GVR and verb", () => {
    expect(() => assertGVRPermission(manifest, "apps.v1.deployments", "list")).not.toThrow();
    expect(() => assertGVRPermission(manifest, "apps.v1.deployments", "get")).not.toThrow();
  });

  it("throws for unpermitted verb", () => {
    expect(() => assertGVRPermission(manifest, "apps.v1.deployments", "delete")).toThrow();
  });

  it("throws for unpermitted GVR", () => {
    expect(() => assertGVRPermission(manifest, "apps.v1.statefulsets", "list")).toThrow();
  });

  it('maps "core" group to empty string', () => {
    expect(() => assertGVRPermission(manifest, "core.v1.pods", "list")).not.toThrow();
    expect(() => assertGVRPermission(manifest, "core.v1.pods", "watch")).not.toThrow();
  });

  it("handles GVRs with dots in the group (e.g. cert-manager)", () => {
    const m: PluginManifest = {
      ...manifest,
      permissions: {
        resources: [{group: "cert-manager.io", version: "v1", resource: "certificates", verbs: ["list"]}],
      },
    };
    expect(() => assertGVRPermission(m, "cert-manager.io.v1.certificates", "list")).not.toThrow();
    expect(() => assertGVRPermission(m, "cert-manager.io.v1.certificates", "delete")).toThrow();
  });

  it("throws when manifest has no permissions", () => {
    const empty: PluginManifest = {schemaVersion: 1, name: "x", version: "0.1.0", displayName: "X", minHostVersion: "0.1.0"};
    expect(() => assertGVRPermission(empty, "apps.v1.deployments", "list")).toThrow();
  });
});
