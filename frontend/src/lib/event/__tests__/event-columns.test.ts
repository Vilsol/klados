import {describe, it, expect} from "vitest";
import {
  classifySeverity,
  eventTimestamp,
  involvedObjectKey,
  formatObject,
  involvedObjectOf,
} from "../event-columns";

describe("event-columns helpers", () => {
  it("classifies Warning vs Normal", () => {
    expect(classifySeverity({type: "Warning"} as any)).toBe("Warning");
    expect(classifySeverity({type: "Normal"} as any)).toBe("Normal");
    expect(classifySeverity({} as any)).toBe("Normal");
  });
  it("falls back through timestamp fields", () => {
    expect(eventTimestamp({lastTimestamp: "L", eventTime: "E"} as any)).toBe("L");
    expect(eventTimestamp({eventTime: "E"} as any)).toBe("E");
    expect(eventTimestamp({metadata: {creationTimestamp: "C"}} as any)).toBe("C");
    expect(eventTimestamp({} as any)).toBe("");
  });
  it("uses uid as involvedObjectKey when present, falls back to ns/kind/name", () => {
    expect(involvedObjectKey({involvedObject: {uid: "u1"}} as any)).toBe("u1");
    expect(involvedObjectKey({involvedObject: {kind: "Pod", name: "p", namespace: "ns"}} as any)).toBe("ns/Pod/p");
  });
  it("formatObject renders Kind/Name", () => {
    expect(formatObject(involvedObjectOf({involvedObject: {kind: "Pod", name: "p1"}} as any))).toBe("Pod/p1");
  });
});
