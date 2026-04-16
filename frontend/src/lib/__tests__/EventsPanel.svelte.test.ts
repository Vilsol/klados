import {describe, it, expect, vi, beforeEach} from "vitest";
import {render, screen, waitFor} from "@testing-library/svelte";

vi.mock("../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js", () => ({
  GetEvents: vi.fn().mockResolvedValue([
    {
      type: "Normal",
      reason: "Pulled",
      message: 'Successfully pulled image "nginx"',
      count: 3,
      lastTimestamp: new Date(Date.now() - 600 * 1000).toISOString(),
      metadata: {creationTimestamp: new Date().toISOString()},
    },
    {
      type: "Warning",
      reason: "BackOff",
      message: "Back-off restarting failed container",
      count: 10,
      lastTimestamp: new Date(Date.now() - 60 * 1000).toISOString(),
      metadata: {},
    },
  ]),
}));

import EventsPanel from "$lib/components/panels/EventsPanel.svelte";
import {GetEvents} from "../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js";

const mockGetEvents = GetEvents as ReturnType<typeof vi.fn>;

describe("EventsPanel", () => {
  const defaultEvents = [
    {
      type: "Normal",
      reason: "Pulled",
      message: 'Successfully pulled image "nginx"',
      count: 3,
      lastTimestamp: new Date(Date.now() - 600 * 1000).toISOString(),
      metadata: {creationTimestamp: new Date().toISOString()},
    },
    {
      type: "Warning",
      reason: "BackOff",
      message: "Back-off restarting failed container",
      count: 10,
      lastTimestamp: new Date(Date.now() - 60 * 1000).toISOString(),
      metadata: {},
    },
  ];

  beforeEach(() => {
    mockGetEvents.mockResolvedValue(defaultEvents);
  });

  it("renders loading state initially", () => {
    render(EventsPanel, {props: {ctxName: "ctx", namespace: "default", uid: "abc-123"}});
    expect(screen.getByText("Loading events...")).toBeTruthy();
  });

  it("renders event rows after load", async () => {
    render(EventsPanel, {props: {ctxName: "ctx", namespace: "default", uid: "abc-123"}});
    await waitFor(() => expect(screen.getByText("Pulled")).toBeTruthy());
    expect(screen.getByText("BackOff")).toBeTruthy();
  });

  it("shows warning badge for Warning events", async () => {
    render(EventsPanel, {props: {ctxName: "ctx", namespace: "default", uid: "abc-123"}});
    await waitFor(() => expect(screen.getByText("Warning")).toBeTruthy());
  });

  it("shows event message", async () => {
    render(EventsPanel, {props: {ctxName: "ctx", namespace: "default", uid: "abc-123"}});
    await waitFor(() => expect(screen.getByText('Successfully pulled image "nginx"')).toBeTruthy());
  });

  it("shows empty state when no events", async () => {
    mockGetEvents.mockResolvedValue([]);
    render(EventsPanel, {props: {ctxName: "ctx", namespace: "default", uid: "abc-123"}});
    await waitFor(() => expect(screen.getByText("No events found.")).toBeTruthy());
  });

  it("calls GetEvents with empty namespace for cluster-scoped resources", async () => {
    render(EventsPanel, {props: {ctxName: "ctx", namespace: "", uid: "cluster-uid"}});
    await waitFor(() => expect(mockGetEvents).toHaveBeenCalledWith("ctx", "", "cluster-uid"));
  });

  it("applies amber tint to Warning event rows", async () => {
    render(EventsPanel, {props: {ctxName: "ctx", namespace: "default", uid: "abc-123"}});
    await waitFor(() => expect(screen.getByText("BackOff")).toBeTruthy());
    const rows = document.querySelectorAll("tbody tr");
    const warningRow = Array.from(rows).find((r) => r.textContent?.includes("BackOff"));
    expect(warningRow?.className).toContain("bg-amber-500/5");
  });

  it("does not apply amber tint to Normal event rows", async () => {
    render(EventsPanel, {props: {ctxName: "ctx", namespace: "default", uid: "abc-123"}});
    await waitFor(() => expect(screen.getByText("Pulled")).toBeTruthy());
    const rows = document.querySelectorAll("tbody tr");
    const normalRow = Array.from(rows).find((r) => r.textContent?.includes("Pulled"));
    expect(normalRow?.className).not.toContain("bg-amber-500/5");
  });
});
