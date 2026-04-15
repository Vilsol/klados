import {describe, it, expect, vi, beforeEach} from "vitest";
import {render, fireEvent, screen} from "@testing-library/svelte";
import EventDetailPanel from "../EventDetailPanel.svelte";

// CodeMirror DOM operations don't work in jsdom — mock CodeBlock and CopyableValue
vi.mock("@klados/ui", () => ({
  CodeBlock: vi.fn(),
  CopyableValue: vi.fn(),
}));

vi.mock("$lib/stores/cluster.svelte", () => ({
  clusterStore: {
    resolveOwnerGVR: vi.fn(),
  },
}));

import {clusterStore} from "$lib/stores/cluster.svelte";

const baseEvent = {
  metadata: {uid: "e1", namespace: "default", creationTimestamp: "2026-04-15T10:00:00Z"},
  type: "Warning",
  reason: "BackOff",
  message: "Back-off restarting failed container",
  count: 3,
  lastTimestamp: "2026-04-15T10:00:00Z",
  involvedObject: {kind: "Pod", apiVersion: "v1", name: "my-pod", namespace: "default", uid: "pod-1"},
  source: {component: "kubelet", host: "node-1"},
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("EventDetailPanel", () => {
  it("renders severity badge, reason, and count", () => {
    vi.mocked(clusterStore.resolveOwnerGVR).mockReturnValue("core.v1.pods");
    render(EventDetailPanel, {
      props: {event: baseEvent, now: Date.parse("2026-04-15T10:05:00Z")},
    });
    expect(screen.getByText("Warning")).toBeTruthy();
    expect(screen.getByText("BackOff")).toBeTruthy();
    expect(screen.getByText("3")).toBeTruthy();
  });

  it("fires onOpenInvolvedObject with ref + gvr when GVR resolves", async () => {
    vi.mocked(clusterStore.resolveOwnerGVR).mockReturnValue("core.v1.pods");
    const spy = vi.fn();
    render(EventDetailPanel, {
      props: {event: baseEvent, now: Date.now(), onOpenInvolvedObject: spy},
    });
    const card = screen.getByTestId("involved-object-card");
    await fireEvent.click(card);
    expect(spy).toHaveBeenCalledTimes(1);
    expect(spy.mock.calls[0][1]).toBe("core.v1.pods");
    expect(spy.mock.calls[0][0].kind).toBe("Pod");
    expect(spy.mock.calls[0][0].name).toBe("my-pod");
  });

  it("disables the involved-object card when no GVR resolves", () => {
    vi.mocked(clusterStore.resolveOwnerGVR).mockReturnValue(undefined);
    render(EventDetailPanel, {
      props: {event: baseEvent, now: Date.now(), onOpenInvolvedObject: vi.fn()},
    });
    const card = screen.getByTestId("involved-object-card") as HTMLButtonElement;
    expect(card.disabled).toBe(true);
    expect(card.title).toContain("No descriptor registered for Kind Pod");
  });

  it("shows the grouped count strip for GroupedEvent inputs", () => {
    vi.mocked(clusterStore.resolveOwnerGVR).mockReturnValue("core.v1.pods");
    const grouped = {
      key: "BackOff|pod-1",
      reason: "BackOff",
      involvedObject: {kind: "Pod", apiVersion: "v1", name: "my-pod", namespace: "default", uid: "pod-1"},
      severity: "Warning" as const,
      count: 17,
      firstSeen: "2026-04-15T09:30:00Z",
      lastSeen: "2026-04-15T10:00:00Z",
      message: "repeating",
      sample: baseEvent,
    };
    render(EventDetailPanel, {props: {event: grouped, now: Date.now()}});
    expect(screen.getByText(/Grouped: 17 occurrences/)).toBeTruthy();
  });
});
