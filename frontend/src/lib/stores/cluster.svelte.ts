import {Events} from "@wailsio/runtime";
import * as ClusterService from "../../../bindings/github.com/Vilsol/klados/internal/services/clusterservice.js";
import * as AppService from "../../../bindings/github.com/Vilsol/klados/internal/services/appservice.js";
import * as ConfigService from "../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js";
import {getLogger} from "$lib/logger";
import {preferencesStore} from "./preferences.svelte";

const log = getLogger("cluster");
import {type KubeContext, ConnectionStatus} from "../../../bindings/github.com/Vilsol/klados/internal/cluster/models.js";
import {buildKindGVRMap, resolveGVR, type APIResource} from "$lib/utils/relationships";

export type {KubeContext};
export {ConnectionStatus};

// PermissionSet mirrors internal/cluster/permissions.go — emitted via event, not a service return type.
export interface PermissionSet {
  rules: Array<{verbs: string[]; resources: string[]; apiGroups: string[]}>;
  inferred: boolean;
}

export type ConnectionStatusType = "disconnected" | "connecting" | "connected" | "error";

const statusToString: Record<number, ConnectionStatusType> = {
  [ConnectionStatus.StatusDisconnected]: "disconnected",
  [ConnectionStatus.StatusConnecting]: "connecting",
  [ConnectionStatus.StatusConnected]: "connected",
  [ConnectionStatus.StatusError]: "error",
};

class ClusterStore {
  contexts = $state<KubeContext[]>([]);
  // activeContext = the cluster currently being viewed (set by routing components)
  activeContext = $state<string | null>(null);
  // Per-cluster namespace selection and list
  selectedNamespaces = $state<Record<string, string[]>>({});
  namespaces = $state<Record<string, string[]>>({});
  connectionStatus = $state<Record<string, ConnectionStatusType>>({});
  permissions = $state<Record<string, PermissionSet>>({});
  isReadOnly = $state<boolean>(false);
  kindGVRMap = $state<Map<string, string>>(new Map());

  private statusUnsubs: Array<() => void> = [];
  private metaUnsubs: Array<() => void> = [];
  private permUnsubs: Array<() => void> = [];

  /** Returns false when either the global read-only toggle is on or detected RBAC permits no writes. */
  canMutate(): boolean {
    if (preferencesStore.prefs.readOnly || this.isReadOnly) {
      return false;
    }
    const perms = this.permissions[this.activeContext ?? ""];
    if (!perms) {
      return true; // not yet fetched — optimistic
    }
    if (perms.inferred) {
      return true;
    }
    return perms.rules.some((r) => r.verbs.some((v) => v === "*" || v === "delete" || v === "patch" || v === "update" || v === "create"));
  }

  async setReadOnly(enabled: boolean) {
    this.isReadOnly = enabled;
    try {
      await AppService.SetReadOnly(enabled);
    } catch (e) {
      log.error("Failed to save read-only state", {error: String(e)});
    }
  }

  setDiscoveryResources(resources: APIResource[]): void {
    this.kindGVRMap = buildKindGVRMap(resources);
  }

  resolveOwnerGVR(apiVersion: string, kind: string): string | undefined {
    return resolveGVR(this.kindGVRMap, apiVersion, kind);
  }

  /** Set the currently-viewed cluster context (called by route components on mount) */
  setActiveContext(ctxName: string) {
    this.activeContext = ctxName;
  }

  /** Namespace list for the given context (falls back to empty array) */
  getNamespaces(ctxName: string): string[] {
    return this.namespaces[ctxName] ?? [];
  }

  /** Selected namespaces for the given context (empty = all namespaces) */
  getSelectedNamespaces(ctxName: string): string[] {
    return this.selectedNamespaces[ctxName] ?? [];
  }

  async loadContexts() {
    try {
      const result = await ClusterService.ListContexts();
      this.contexts = result ?? [];

      this.statusUnsubs.forEach((u) => u());
      this.statusUnsubs = [];
      this.metaUnsubs.forEach((u) => u());
      this.metaUnsubs = [];
      this.permUnsubs.forEach((u) => u());
      this.permUnsubs = [];

      // Load read-only state from persisted config
      try {
        const cfg = await ConfigService.GetConfig();
        this.isReadOnly = cfg?.readOnly ?? false;
      } catch (e) {
        log.warn("Failed to load persisted config", {error: String(e)});
      }

      for (const ctx of this.contexts) {
        this.connectionStatus[ctx.name] = statusToString[ctx.status] ?? "disconnected";
        const unsub = Events.On(`status:${ctx.name}:connection`, (wailsEvent: any) => {
          const status = (wailsEvent.data ?? wailsEvent) as string;
          this.connectionStatus[ctx.name] = (status as ConnectionStatusType) ?? "disconnected";
          if (status === "connected" && !this.activeContext) {
            this.restoreContext(ctx.name);
          }
        });
        this.statusUnsubs.push(unsub);
        const metaUnsub = Events.On(`metadata:${ctx.name}:cluster`, () => {
          this.loadContexts();
        });
        this.metaUnsubs.push(metaUnsub);
        const permUnsub = Events.On(`cluster:${ctx.name}:permissions`, (wailsEvent: any) => {
          const perms = (wailsEvent.data ?? wailsEvent) as PermissionSet;
          this.permissions[ctx.name] = perms;
        });
        this.permUnsubs.push(permUnsub);
      }

      const connected = this.contexts.find((c) => (statusToString[c.status] ?? "disconnected") === "connected");
      if (connected && !this.activeContext) {
        await this.restoreContext(connected.name);
      } else if (this.activeContext) {
        // activeContext already set by routing (e.g. page refresh) — still load namespaces
        try {
          const saved = await ClusterService.GetActiveNamespace(this.activeContext);
          if (saved) {
            this.selectedNamespaces[this.activeContext] = [saved];
          }
        } catch (e) {
          log.debug("Could not restore saved namespace", {error: String(e)});
        }
        await this.loadNamespaces(this.activeContext);
      }
    } catch (e) {
      log.error("Failed to load contexts", {error: String(e)});
    }
  }

  private async restoreContext(ctxName: string) {
    this.activeContext = ctxName;
    try {
      const saved = await ClusterService.GetActiveNamespace(ctxName);
      if (saved) {
        this.selectedNamespaces[ctxName] = [saved];
      }
    } catch (e) {
      log.debug("Could not restore saved namespace", {error: String(e)});
    }
    await this.loadNamespaces(ctxName);
  }

  async connect(ctxName: string) {
    this.connectionStatus[ctxName] = "connecting";
    try {
      await ClusterService.Connect(ctxName);
      this.connectionStatus[ctxName] = "connected";
      log.info("Cluster connected", {ctxName});
      // Only set activeContext if nothing is currently active
      if (!this.activeContext) {
        this.activeContext = ctxName;
      }
      await this.loadNamespaces(ctxName);
    } catch (e) {
      log.error("Cluster connect failed", {ctxName, error: String(e)});
      this.connectionStatus[ctxName] = "error";
    }
  }

  async disconnect(ctxName: string) {
    try {
      await ClusterService.Disconnect(ctxName);
      this.connectionStatus[ctxName] = "disconnected";
      log.info("Cluster disconnected", {ctxName});
      delete this.namespaces[ctxName];
      delete this.selectedNamespaces[ctxName];
      if (this.activeContext === ctxName) {
        // Find another connected cluster to switch to
        const other = Object.entries(this.connectionStatus).find(([name, s]) => name !== ctxName && s === "connected");
        this.activeContext = other ? other[0] : null;
      }
    } catch (e) {
      log.error("Failed to disconnect", {error: String(e)});
    }
  }

  async loadNamespaces(ctxName: string) {
    try {
      const result = await ClusterService.ListNamespaces(ctxName);
      this.namespaces[ctxName] = result ?? [];
    } catch (e) {
      log.error("Failed to load namespaces", {error: String(e)});
    }
  }

  async createNamespace(ctxName: string, name: string) {
    await ClusterService.CreateNamespace(ctxName, name);
    await this.loadNamespaces(ctxName);
  }

  async deleteNamespace(ctxName: string, name: string) {
    await ClusterService.DeleteNamespace(ctxName, name);
    this.namespaces[ctxName] = (this.namespaces[ctxName] ?? []).filter((n) => n !== name);
    this.selectedNamespaces[ctxName] = (this.selectedNamespaces[ctxName] ?? []).filter((n) => n !== name);
  }

  async setNamespaces(ctxName: string, namespaces: string[]) {
    this.selectedNamespaces[ctxName] = namespaces;
    const persist = namespaces.length === 1 ? namespaces[0] : "";
    try {
      await ClusterService.SwitchNamespace(ctxName, persist);
    } catch (e) {
      log.error("Failed to switch namespace", {error: String(e)});
    }
  }
}

export const clusterStore = new ClusterStore();
