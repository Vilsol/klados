import {describe, it, expect} from "vitest";
import {groupEvents} from "../event-grouping";
import type {EventItem} from "../event-types";

function ev(overrides: Partial<EventItem>): EventItem {
  return {
    metadata: {uid: Math.random().toString(36).slice(2), namespace: "default", creationTimestamp: "2026-04-15T10:00:00Z"},
    type: "Normal",
    reason: "Scheduled",
    message: "",
    count: 1,
    involvedObject: {kind: "Pod", name: "p1", namespace: "default", uid: "pod-1"},
    lastTimestamp: "2026-04-15T10:00:00Z",
    ...overrides,
  };
}

describe("groupEvents", () => {
  it("merges events by reason + involvedObject", () => {
    const a = ev({reason: "BackOff", lastTimestamp: "2026-04-15T10:00:00Z", count: 3});
    const b = ev({reason: "BackOff", lastTimestamp: "2026-04-15T10:02:00Z", count: 2, message: "latest"});
    const result = groupEvents([a, b]);
    expect(result).toHaveLength(1);
    expect(result[0].count).toBe(5);
    expect(result[0].lastSeen).toBe("2026-04-15T10:02:00Z");
    expect(result[0].message).toBe("latest");
  });

  it("keeps different reasons separate", () => {
    const a = ev({reason: "BackOff"});
    const b = ev({reason: "Failed"});
    expect(groupEvents([a, b])).toHaveLength(2);
  });

  it("keeps different involved objects separate", () => {
    const a = ev({involvedObject: {kind: "Pod", name: "p1", uid: "u1", namespace: "default"}});
    const b = ev({involvedObject: {kind: "Pod", name: "p2", uid: "u2", namespace: "default"}});
    expect(groupEvents([a, b])).toHaveLength(2);
  });

  it("escalates severity to Warning when any contributor is Warning", () => {
    const a = ev({type: "Normal"});
    const b = ev({type: "Warning"});
    const result = groupEvents([a, b]);
    expect(result[0].severity).toBe("Warning");
  });

  it("tracks firstSeen as the earliest firstTimestamp fallback", () => {
    const a = ev({firstTimestamp: "2026-04-15T09:55:00Z", lastTimestamp: "2026-04-15T10:00:00Z"});
    const b = ev({firstTimestamp: "2026-04-15T09:50:00Z", lastTimestamp: "2026-04-15T10:02:00Z"});
    expect(groupEvents([a, b])[0].firstSeen).toBe("2026-04-15T09:50:00Z");
  });

  it("returns empty array for empty input", () => {
    expect(groupEvents([])).toEqual([]);
  });
});
