import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
import * as PluginService from '../../../bindings/github.com/Vilsol/klados/internal/services/pluginservice.js'
import { evaluate } from 'cel-js'
import { setRegistryLoaded } from './loaded.svelte'

export type RenderType = 'text' | 'badge' | 'age' | 'progress'

export interface ColumnDef {
  name: string
  expr: string
  renderType: RenderType
  width?: number
}

export interface OverviewFieldDef {
  label: string
  expr: string
  renderType: RenderType
}

export interface DescriptorDef {
  group: string
  version: string
  resource: string
  kind: string
  gvr: string
  columns: ColumnDef[]
  overviewFields: OverviewFieldDef[]
  detailPanels: string[]
  actions: string[]
}

class DescriptorRegistry {
  private descriptors = new Map<string, DescriptorDef>()
  private builtins = new Map<string, DescriptorDef>()

  async load() {
    try {
      const defs = await ResourceService.GetDescriptors()
      this.builtins.clear()
      for (const d of defs ?? []) {
        if (!d) continue
        const gKey = d.group === '' ? 'core' : d.group
        const gvr = `${gKey}.${d.version}.${d.resource}`
        this.builtins.set(gvr, {
          group: d.group ?? '',
          version: d.version ?? '',
          resource: d.resource ?? '',
          kind: d.kind ?? '',
          gvr,
          columns: (d.columns ?? []).map((c: any) => ({
            name: c.name ?? '',
            expr: c.expr ?? '',
            renderType: (c.renderType ?? 'text') as RenderType,
            width: c.width ?? undefined,
          })),
          overviewFields: (d.overviewFields ?? []).map((f: any) => ({
            label: f.label ?? '',
            expr: f.expr ?? '',
            renderType: (f.renderType ?? 'text') as RenderType,
          })),
          detailPanels: d.detailPanels ?? [],
          actions: d.actions ?? [],
        })
      }
      this.descriptors = new Map(this.builtins)
      await this.mergePluginDescriptors()
    } catch (e) {
      console.error('Failed to load descriptors:', e)
    } finally {
      setRegistryLoaded()
    }
  }

  // reloadPlugins resets to builtins then re-merges from GetPluginDescriptors().
  // Call this when the plugins:loaded event fires (hot-reload).
  async reloadPlugins() {
    this.descriptors = new Map(this.builtins)
    await this.mergePluginDescriptors()
  }

  private async mergePluginDescriptors() {
    try {
      const pluginDefs = await PluginService.GetPluginDescriptors()
      for (const d of pluginDefs ?? []) {
        if (!d) continue
        const gKey = d.group === '' ? 'core' : d.group
        const gvr = `${gKey}.${d.version}.${d.resource}`
        if (this.descriptors.has(gvr)) {
          const existing = this.descriptors.get(gvr)!
          const pluginColumns = (d.columns ?? []).map((c: any) => ({
            name: c.name ?? '',
            expr: c.expr ?? '',
            renderType: (c.renderType ?? 'text') as RenderType,
            width: c.width ?? undefined,
          }))
          const addedOverview = (d.overviewFields ?? []).map((f: any) => ({
            label: f.label ?? '',
            expr: f.expr ?? '',
            renderType: (f.renderType ?? 'text') as RenderType,
          }))
          const panelSet = new Set(existing.detailPanels)
          for (const p of d.detailPanels ?? []) panelSet.add(p)
          const actionSet = new Set(existing.actions)
          for (const a of d.actions ?? []) actionSet.add(a)
          // Create a new object — never mutate builtins references.
          // Plugin columns replace built-in columns; panels/actions are additive.
          this.descriptors.set(gvr, {
            ...existing,
            columns: pluginColumns.length > 0 ? pluginColumns : existing.columns,
            overviewFields: [...existing.overviewFields, ...addedOverview],
            detailPanels: [...panelSet],
            actions: [...actionSet],
          })
        } else {
          this.descriptors.set(gvr, {
            group: d.group ?? '',
            version: d.version ?? '',
            resource: d.resource ?? '',
            kind: d.kind ?? '',
            gvr,
            columns: (d.columns ?? []).map((c: any) => ({
              name: c.name ?? '',
              expr: c.expr ?? '',
              renderType: (c.renderType ?? 'text') as RenderType,
              width: c.width ?? undefined,
            })),
            overviewFields: (d.overviewFields ?? []).map((f: any) => ({
              label: f.label ?? '',
              expr: f.expr ?? '',
              renderType: (f.renderType ?? 'text') as RenderType,
            })),
            detailPanels: d.detailPanels ?? [],
            actions: d.actions ?? [],
          })
        }
      }
    } catch (e) {
      console.error('Failed to load plugin descriptors:', e)
    }
  }

  get(gvr: string): DescriptorDef {
    return this.descriptors.get(gvr) ?? this.fallback(gvr)
  }

  private fallback(gvr: string): DescriptorDef {
    const parts = gvr.split('.')
    const resource = parts.at(-1) ?? gvr
    const version = parts.at(-2) ?? ''
    const group = parts.slice(0, -2).join('.')
    return {
      group: group === 'core' ? '' : group,
      version,
      resource,
      kind: '',
      gvr,
      columns: [
        { name: 'Name', expr: 'metadata.name', renderType: 'text' },
        { name: 'Namespace', expr: 'metadata.namespace', renderType: 'text' },
        { name: 'Age', expr: 'metadata.creationTimestamp', renderType: 'age' },
      ],
      overviewFields: [
        { label: 'Namespace', expr: 'metadata.namespace', renderType: 'text' },
        { label: 'Age', expr: 'metadata.creationTimestamp', renderType: 'age' },
      ],
      detailPanels: ['overview', 'yaml'],
      actions: ['delete'],
    }
  }

  list(): DescriptorDef[] {
    return Array.from(this.descriptors.values())
  }
}

export const descriptorRegistry = new DescriptorRegistry()

export function evalExpr(expr: string, obj: Record<string, any>): any {
  const ctx = {
    metadata: obj.metadata ?? {},
    spec: obj.spec ?? {},
    status: obj.status ?? {},
    type: obj.type ?? '',
  }
  try {
    const result = evaluate(expr, ctx)
    return result ?? ''
  } catch {
    // fall back to simple path access
    return getPath(obj, expr)
  }
}

function getPath(obj: any, path: string): any {
  const parts = path.split('.')
  let cur = obj
  for (const p of parts) {
    if (cur == null) return ''
    cur = cur[p]
  }
  return cur ?? ''
}
