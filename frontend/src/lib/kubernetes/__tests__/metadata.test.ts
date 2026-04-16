import { describe, it, expect } from "vitest";
import {
  getLabels,
  getAnnotations,
  getFinalizers,
  stripServerFields,
  getLastAppliedConfig,
  LAST_APPLIED_ANNOTATION,
} from "../metadata";

describe("getLabels / getAnnotations", () => {
  it("returns empty objects when missing", () => {
    expect(getLabels({})).toEqual({});
    expect(getAnnotations({ metadata: {} })).toEqual({});
  });
  it("extracts label and annotation maps", () => {
    const obj = { metadata: { labels: { app: "x" }, annotations: { a: "1" } } };
    expect(getLabels(obj)).toEqual({ app: "x" });
    expect(getAnnotations(obj)).toEqual({ a: "1" });
  });
});

describe("getFinalizers", () => {
  it("returns [] when missing", () => {
    expect(getFinalizers({})).toEqual([]);
    expect(getFinalizers({ metadata: {} })).toEqual([]);
  });
  it("returns the finalizers list", () => {
    const obj = { metadata: { finalizers: ["foregroundDeletion", "example.com/cleanup"] } };
    expect(getFinalizers(obj)).toEqual(["foregroundDeletion", "example.com/cleanup"]);
  });
});

describe("stripServerFields", () => {
  it("removes server-managed fields", () => {
    const obj = {
      apiVersion: "v1",
      kind: "Pod",
      metadata: {
        name: "p",
        namespace: "default",
        resourceVersion: "123",
        uid: "abc",
        creationTimestamp: "2026-04-16T00:00:00Z",
        generation: 1,
        selfLink: "/api/v1/pods/p",
        managedFields: [{ manager: "kubectl" }],
      },
      spec: { x: 1 },
      status: { ready: true },
    };
    const out = stripServerFields(obj);
    expect(out.metadata.resourceVersion).toBeUndefined();
    expect(out.metadata.uid).toBeUndefined();
    expect(out.metadata.creationTimestamp).toBeUndefined();
    expect(out.metadata.generation).toBeUndefined();
    expect(out.metadata.selfLink).toBeUndefined();
    expect(out.metadata.managedFields).toBeUndefined();
    expect(out.status).toBeUndefined();
    expect(out.metadata.name).toBe("p");
    expect(out.spec).toEqual({ x: 1 });
  });
  it("does not mutate the input", () => {
    const obj = { metadata: { uid: "x" } };
    stripServerFields(obj);
    expect(obj.metadata.uid).toBe("x");
  });
});

describe("getLastAppliedConfig", () => {
  it("returns null when annotation missing", () => {
    expect(getLastAppliedConfig({})).toBeNull();
  });
  it("parses the annotation as JSON", () => {
    const obj = { metadata: { annotations: { [LAST_APPLIED_ANNOTATION]: '{"spec":{"x":1}}' } } };
    expect(getLastAppliedConfig(obj)).toEqual({ spec: { x: 1 } });
  });
  it("returns null when JSON is malformed", () => {
    const obj = { metadata: { annotations: { [LAST_APPLIED_ANNOTATION]: "{not json" } } };
    expect(getLastAppliedConfig(obj)).toBeNull();
  });
});
