import { Events } from '@wailsio/runtime'
import * as PluginService from '../../../bindings/github.com/Vilsol/klados/internal/services/pluginservice.js'
import { notificationStore } from '$lib/stores/notification.svelte.js'

export interface ResourcePerm {
  group: string
  version: string
  resource: string
  verbs: string[]
}

export interface RegisteredDetailTab {
  pluginName: string
  gvr: string
  id: string
  label: string
  component: string
  resourcePerms: ResourcePerm[]
}

export interface RegisteredCommand {
  pluginName: string
  id: string
  label: string
  icon?: string
  action: () => void
}

export interface RegisteredOverviewField {
  pluginName: string
  gvr: string
  id: string
  label: string
  component: string
}

export interface RegisteredListColumn {
  pluginName: string
  gvr: string
  id: string
  label: string
  component: string
}

export interface RegisteredContextMenuItem {
  pluginName: string
  gvr: string
  id: string
  label: string
  component: string
}

export interface RegisteredHeaderWidget {
  pluginName: string
  id: string
  component: string
}

export interface RegisteredStatusBarWidget {
  pluginName: string
  id: string
  component: string
}

class SlotRegistry {
  detailTabs = $state<RegisteredDetailTab[]>([])
  commands = $state<RegisteredCommand[]>([])
  overviewFields = $state<RegisteredOverviewField[]>([])
  listColumns = $state<RegisteredListColumn[]>([])
  contextMenuItems = $state<RegisteredContextMenuItem[]>([])
  headerWidgets = $state<RegisteredHeaderWidget[]>([])
  statusBarWidgets = $state<RegisteredStatusBarWidget[]>([])

  getDetailTabs(gvr: string): RegisteredDetailTab[] {
    return this.detailTabs.filter((t) => t.gvr === gvr)
  }

  getCommands(): RegisteredCommand[] {
    return this.commands
  }

  getOverviewFields(gvr: string): RegisteredOverviewField[] {
    return this.overviewFields.filter((f) => f.gvr === gvr)
  }

  getListColumns(gvr: string): RegisteredListColumn[] {
    return this.listColumns.filter((c) => c.gvr === gvr)
  }

  getContextMenuItems(gvr: string): RegisteredContextMenuItem[] {
    return this.contextMenuItems.filter((c) => c.gvr === gvr)
  }

  getHeaderWidgets(): RegisteredHeaderWidget[] {
    return this.headerWidgets
  }

  getStatusBarWidgets(): RegisteredStatusBarWidget[] {
    return this.statusBarWidgets
  }

  registerCommand(cmd: RegisteredCommand): void {
    this.commands = [...this.commands, cmd]
  }

  unregisterPlugin(pluginName: string): void {
    this.detailTabs = this.detailTabs.filter((t) => t.pluginName !== pluginName)
    this.commands = this.commands.filter((c) => c.pluginName !== pluginName)
    this.overviewFields = this.overviewFields.filter((f) => f.pluginName !== pluginName)
    this.listColumns = this.listColumns.filter((c) => c.pluginName !== pluginName)
    this.contextMenuItems = this.contextMenuItems.filter((c) => c.pluginName !== pluginName)
    this.headerWidgets = this.headerWidgets.filter((w) => w.pluginName !== pluginName)
    this.statusBarWidgets = this.statusBarWidgets.filter((w) => w.pluginName !== pluginName)
  }

  async initFromBackend(): Promise<void> {
    try {
      const [tabs, cmds, overviewFields, listColumns, contextMenuItems, headerWidgets, statusBarWidgets] = await Promise.all([
        PluginService.GetPluginDetailTabs(),
        PluginService.GetPluginCommands(),
        PluginService.GetPluginOverviewFields(''),
        PluginService.GetPluginListColumns(''),
        PluginService.GetPluginContextMenuItems(''),
        PluginService.GetPluginHeaderWidgets(),
        PluginService.GetPluginStatusBarWidgets(),
      ])
      this.detailTabs = (tabs ?? []).map((t) => ({
        pluginName: t.pluginName ?? '',
        gvr: t.gvr ?? '',
        id: t.id ?? '',
        label: t.label ?? '',
        component: t.component ?? '',
        resourcePerms: (t.resourcePerms ?? []).map((p) => ({
          group: p.group ?? '',
          version: p.version ?? '',
          resource: p.resource ?? '',
          verbs: p.verbs ?? [],
        })),
      }))
      this.commands = (cmds ?? []).map((c) => ({
        pluginName: c.pluginName ?? '',
        id: c.id ?? '',
        label: c.label ?? '',
        icon: c.icon ?? undefined,
        action: () => {},
      }))
      this.overviewFields = (overviewFields ?? []).map((f) => ({
        pluginName: f.pluginName ?? '',
        gvr: f.gvr ?? '',
        id: f.id ?? '',
        label: f.label ?? '',
        component: f.component ?? '',
      }))
      this.listColumns = (listColumns ?? []).map((c) => ({
        pluginName: c.pluginName ?? '',
        gvr: c.gvr ?? '',
        id: c.id ?? '',
        label: c.label ?? '',
        component: c.component ?? '',
      }))
      this.contextMenuItems = (contextMenuItems ?? []).map((c) => ({
        pluginName: c.pluginName ?? '',
        gvr: c.gvr ?? '',
        id: c.id ?? '',
        label: c.label ?? '',
        component: c.component ?? '',
      }))
      this.headerWidgets = (headerWidgets ?? []).map((w) => ({
        pluginName: w.pluginName ?? '',
        id: w.id ?? '',
        component: w.component ?? '',
      }))
      this.statusBarWidgets = (statusBarWidgets ?? []).map((w) => ({
        pluginName: w.pluginName ?? '',
        id: w.id ?? '',
        component: w.component ?? '',
      }))
    } catch {
      // No plugins loaded or backend unavailable — non-fatal
    }
  }
}

export const slotRegistry = new SlotRegistry()

if (typeof window !== 'undefined') {
  Events.On('plugins:loaded', () => {
    slotRegistry.initFromBackend()
  })

  Events.On('plugin:reloading', (wailsEvent: any) => {
    const name = wailsEvent?.data?.name ?? wailsEvent?.name
    if (name) {
      slotRegistry.unregisterPlugin(name)
      notificationStore.push(`Reloading plugin "${name}"...`, 'info')
    }
  })

  Events.On('plugin:loaded', (wailsEvent: any) => {
    const name = wailsEvent?.data?.name ?? wailsEvent?.name
    slotRegistry.initFromBackend()
    if (name) {
      notificationStore.push(`Plugin "${name}" reloaded`, 'success')
    }
  })

  Events.On('plugin:error', (wailsEvent: any) => {
    const data = wailsEvent?.data ?? wailsEvent
    const name = data?.name
    const error = data?.error
    if (name) {
      notificationStore.error(`Plugin "${name}" failed`, error)
    }
  })

  slotRegistry.initFromBackend()
}
