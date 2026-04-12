import {describe, it, expect, vi, beforeEach} from "vitest";
import {clusterStore, ConnectionStatus} from "$lib/stores/cluster.svelte";

// Mock the binding module
vi.mock("../../../bindings/github.com/Vilsol/klados/internal/services/clusterservice.js", () => ({
  ListContexts: vi.fn(),
  Connect: vi.fn(),
  Disconnect: vi.fn(),
  ListNamespaces: vi.fn(),
  SwitchNamespace: vi.fn(),
  GetActiveNamespace: vi.fn().mockResolvedValue(""),
  GetStatus: vi.fn(),
}));

import {
  ListContexts,
  Connect,
  Disconnect,
  ListNamespaces,
} from "../../../bindings/github.com/Vilsol/klados/internal/services/clusterservice";

const mockedListContexts = vi.mocked(ListContexts);
const mockedConnect = vi.mocked(Connect);
const mockedDisconnect = vi.mocked(Disconnect);
const mockedListNamespaces = vi.mocked(ListNamespaces);

describe("clusterStore", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clusterStore.contexts = [];
    clusterStore.activeContext = null;
    clusterStore.selectedNamespaces = {};
    clusterStore.namespaces = {};
    clusterStore.connectionStatus = {};
  });

  it("loadContexts populates contexts and status", async () => {
    mockedListContexts.mockResolvedValue([
      {name: "ctx1", cluster: "c1", user: "u1", namespace: "default", status: ConnectionStatus.StatusConnected},
      {name: "ctx2", cluster: "c2", user: "u2", namespace: "ns2", status: ConnectionStatus.StatusDisconnected},
    ] as unknown[]);

    await clusterStore.loadContexts();

    expect(clusterStore.contexts).toHaveLength(2);
    expect(clusterStore.connectionStatus.ctx1).toBe("connected");
    expect(clusterStore.connectionStatus.ctx2).toBe("disconnected");
  });

  it("connect updates status and sets active context", async () => {
    mockedConnect.mockResolvedValue(undefined);
    mockedListNamespaces.mockResolvedValue(["default", "kube-system"] as unknown[]);

    await clusterStore.connect("ctx1");

    expect(mockedConnect).toHaveBeenCalledWith("ctx1");
    expect(clusterStore.connectionStatus.ctx1).toBe("connected");
    expect(clusterStore.activeContext).toBe("ctx1");
    expect(clusterStore.getNamespaces("ctx1")).toEqual(["default", "kube-system"]);
  });

  it("connect does not override activeContext when already set", async () => {
    clusterStore.activeContext = "ctx1";
    mockedConnect.mockResolvedValue(undefined);
    mockedListNamespaces.mockResolvedValue([] as unknown[]);

    await clusterStore.connect("ctx2");

    expect(clusterStore.activeContext).toBe("ctx1");
    expect(clusterStore.connectionStatus.ctx2).toBe("connected");
  });

  it("connect sets error status on failure", async () => {
    mockedConnect.mockRejectedValue(new Error("fail"));

    await clusterStore.connect("ctx1");

    expect(clusterStore.connectionStatus.ctx1).toBe("error");
  });

  it("disconnect clears active context", async () => {
    clusterStore.activeContext = "ctx1";
    clusterStore.namespaces = {ctx1: ["default"]};
    mockedDisconnect.mockResolvedValue(undefined);

    await clusterStore.disconnect("ctx1");

    expect(mockedDisconnect).toHaveBeenCalledWith("ctx1");
    expect(clusterStore.connectionStatus.ctx1).toBe("disconnected");
    expect(clusterStore.activeContext).toBeNull();
    expect(clusterStore.getNamespaces("ctx1")).toEqual([]);
  });

  it("disconnect preserves other connected context as active", async () => {
    clusterStore.activeContext = "ctx1";
    clusterStore.connectionStatus = {ctx1: "connected", ctx2: "connected"};
    mockedDisconnect.mockResolvedValue(undefined);

    await clusterStore.disconnect("ctx1");

    expect(clusterStore.activeContext).toBe("ctx2");
  });

  it("setNamespaces updates selected namespaces per context", async () => {
    await clusterStore.setNamespaces("ctx1", ["kube-system"]);

    expect(clusterStore.getSelectedNamespaces("ctx1")).toEqual(["kube-system"]);
    expect(clusterStore.getSelectedNamespaces("ctx2")).toEqual([]);
  });
});
