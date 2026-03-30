import { Events } from '@wailsio/runtime'
import * as ClusterService from '../../../bindings/github.com/Vilsol/klados/internal/services/clusterservice.js'
import { KubeContext, ConnectionStatus } from '../../../bindings/github.com/Vilsol/klados/internal/cluster/models.js'

export type { KubeContext }
export { ConnectionStatus }

export type ConnectionStatusType = 'disconnected' | 'connecting' | 'connected' | 'error'

const statusToString: Record<number, ConnectionStatusType> = {
  [ConnectionStatus.StatusDisconnected]: 'disconnected',
  [ConnectionStatus.StatusConnecting]: 'connecting',
  [ConnectionStatus.StatusConnected]: 'connected',
  [ConnectionStatus.StatusError]: 'error',
}

class ClusterStore {
  contexts = $state<KubeContext[]>([])
  // activeContext = the cluster currently being viewed (set by routing components)
  activeContext = $state<string | null>(null)
  // Per-cluster namespace selection and list
  selectedNamespaces = $state<Record<string, string[]>>({})
  namespaces = $state<Record<string, string[]>>({})
  connectionStatus = $state<Record<string, ConnectionStatusType>>({})

  private statusUnsubs: Array<() => void> = []
  private metaUnsubs: Array<() => void> = []

  /** Set the currently-viewed cluster context (called by route components on mount) */
  setActiveContext(ctxName: string) {
    this.activeContext = ctxName
  }

  /** Namespace list for the given context (falls back to empty array) */
  getNamespaces(ctxName: string): string[] {
    return this.namespaces[ctxName] ?? []
  }

  /** Selected namespaces for the given context (empty = all namespaces) */
  getSelectedNamespaces(ctxName: string): string[] {
    return this.selectedNamespaces[ctxName] ?? []
  }

  async loadContexts() {
    try {
      const result = await ClusterService.ListContexts()
      this.contexts = result ?? []

      this.statusUnsubs.forEach((u) => u())
      this.statusUnsubs = []
      this.metaUnsubs.forEach((u) => u())
      this.metaUnsubs = []
      for (const ctx of this.contexts) {
        this.connectionStatus[ctx.name] = statusToString[ctx.status] ?? 'disconnected'
        const unsub = Events.On(`status:${ctx.name}:connection`, (wailsEvent: any) => {
          const status = (wailsEvent.data ?? wailsEvent) as string
          this.connectionStatus[ctx.name] = (status as ConnectionStatusType) ?? 'disconnected'
          if (status === 'connected' && !this.activeContext) {
            this.restoreContext(ctx.name)
          }
        })
        this.statusUnsubs.push(unsub)
        const metaUnsub = Events.On(`metadata:${ctx.name}:cluster`, () => {
          this.loadContexts()
        })
        this.metaUnsubs.push(metaUnsub)
      }

      const connected = this.contexts.find(
        (c) => (statusToString[c.status] ?? 'disconnected') === 'connected',
      )
      if (connected && !this.activeContext) {
        await this.restoreContext(connected.name)
      }
    } catch (e) {
      console.error('Failed to load contexts:', e)
    }
  }

  private async restoreContext(ctxName: string) {
    this.activeContext = ctxName
    try {
      const saved = await ClusterService.GetActiveNamespace(ctxName)
      if (saved) this.selectedNamespaces[ctxName] = [saved]
    } catch {}
    await this.loadNamespaces(ctxName)
  }

  async connect(ctxName: string) {
    this.connectionStatus[ctxName] = 'connecting'
    try {
      await ClusterService.Connect(ctxName)
      this.connectionStatus[ctxName] = 'connected'
      // Only set activeContext if nothing is currently active
      if (!this.activeContext) this.activeContext = ctxName
      await this.loadNamespaces(ctxName)
    } catch {
      this.connectionStatus[ctxName] = 'error'
    }
  }

  async disconnect(ctxName: string) {
    try {
      await ClusterService.Disconnect(ctxName)
      this.connectionStatus[ctxName] = 'disconnected'
      delete this.namespaces[ctxName]
      delete this.selectedNamespaces[ctxName]
      if (this.activeContext === ctxName) {
        // Find another connected cluster to switch to
        const other = Object.entries(this.connectionStatus).find(
          ([name, s]) => name !== ctxName && s === 'connected'
        )
        this.activeContext = other ? other[0] : null
      }
    } catch (e) {
      console.error('Failed to disconnect:', e)
    }
  }

  async loadNamespaces(ctxName: string) {
    try {
      const result = await ClusterService.ListNamespaces(ctxName)
      this.namespaces[ctxName] = result ?? []
    } catch (e) {
      console.error('Failed to load namespaces:', e)
    }
  }

  async createNamespace(ctxName: string, name: string) {
    await ClusterService.CreateNamespace(ctxName, name)
    await this.loadNamespaces(ctxName)
  }

  async deleteNamespace(ctxName: string, name: string) {
    await ClusterService.DeleteNamespace(ctxName, name)
    this.namespaces[ctxName] = (this.namespaces[ctxName] ?? []).filter((n) => n !== name)
    this.selectedNamespaces[ctxName] = (this.selectedNamespaces[ctxName] ?? []).filter((n) => n !== name)
  }

  async setNamespaces(ctxName: string, namespaces: string[]) {
    this.selectedNamespaces[ctxName] = namespaces
    const persist = namespaces.length === 1 ? namespaces[0] : ''
    try {
      await ClusterService.SwitchNamespace(ctxName, persist)
    } catch (e) {
      console.error('Failed to switch namespace:', e)
    }
  }
}

export const clusterStore = new ClusterStore()
