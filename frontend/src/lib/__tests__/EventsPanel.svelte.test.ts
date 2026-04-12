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

describe("EventsPanel", () => {
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
});
