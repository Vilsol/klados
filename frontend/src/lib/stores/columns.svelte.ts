import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'
import { GVRColumnPrefs, ColumnSettings, SortPrefs } from '../../../bindings/github.com/Vilsol/klados/internal/config/models.js'
import { descriptorRegistry } from '../registry/index.js'
import type { ColumnDef } from '../registry/index.js'

class ColumnStore {
  visibleColumns = $state<ColumnDef[]>([])
  allColumns = $state<{ col: ColumnDef; visible: boolean }[]>([])
  sortState = $state<{ column: string; direction: 'asc' | 'desc' } | null>(null)
  compact = $state<boolean>(false)

  #gvr = ''
  #saveTimer: ReturnType<typeof setTimeout> | null = null

  async loadForGVR(gvr: string): Promise<void> {
    if (this.#saveTimer !== null) {
      clearTimeout(this.#saveTimer)
      this.#saveTimer = null
    }
    this.#gvr = gvr

    const [prefs, compact] = await Promise.all([
      ConfigService.GetColumnPrefs(gvr),
      ConfigService.GetCompactRows(),
    ])
    this.compact = compact
    this.#applyPrefs(prefs)
  }

  #applyPrefs(prefs: GVRColumnPrefs | null): void {
    const descriptor = descriptorRegistry.get(this.#gvr)
    const pool = descriptor.columns

    // Build name → ColumnDef map with width overrides applied
    const poolMap = new Map<string, ColumnDef>(pool.map((c) => [c.name, { ...c }]))
    if (prefs?.columns) {
      for (const [name, settings] of Object.entries(prefs.columns)) {
        if (settings?.width !== undefined && poolMap.has(name)) {
          poolMap.set(name, { ...poolMap.get(name)!, width: settings.width })
        }
      }
    }

    // Determine visible order
    let visibleNames: string[]
    if (prefs?.order && prefs.order.length > 0) {
      visibleNames = prefs.order.filter((name) => poolMap.has(name))
    } else {
      visibleNames = pool.filter((c) => !c.hidden).map((c) => c.name)
    }

    const visibleSet = new Set(visibleNames)
    this.visibleColumns = visibleNames
      .map((name) => poolMap.get(name))
      .filter((c): c is ColumnDef => c !== undefined)
    this.allColumns = pool.map((c) => ({
      col: poolMap.get(c.name)!,
      visible: visibleSet.has(c.name),
    }))

    this.sortState =
      prefs?.sort
        ? { column: prefs.sort.column, direction: prefs.sort.direction as 'asc' | 'desc' }
        : null
  }

  setColumnVisible(name: string, visible: boolean): void {
    if (name === 'Name') return

    const entry = this.allColumns.find((e) => e.col.name === name)
    if (!entry || entry.visible === visible) return

    const col = entry.col
    this.allColumns = this.allColumns.map((e) =>
      e.col.name === name ? { ...e, visible } : e,
    )
    if (visible) {
      this.visibleColumns = [...this.visibleColumns, col]
    } else {
      this.visibleColumns = this.visibleColumns.filter((c) => c.name !== name)
    }
    this.#save()
  }

  moveColumn(name: string, direction: 'up' | 'down'): void {
    const idx = this.visibleColumns.findIndex((c) => c.name === name)
    if (idx === -1) return
    if (direction === 'up' && idx === 0) return
    if (direction === 'down' && idx === this.visibleColumns.length - 1) return

    const next = [...this.visibleColumns]
    const swapIdx = direction === 'up' ? idx - 1 : idx + 1
    ;[next[idx], next[swapIdx]] = [next[swapIdx], next[idx]]
    this.visibleColumns = next
    this.#save()
  }

  resizeColumn(name: string, width: number): void {
    this.#setWidth(name, width)
    this.#debouncedSave()
  }

  autoFitColumn(name: string, width: number): void {
    this.#setWidth(name, width)
    this.#debouncedSave()
  }

  #setWidth(name: string, width: number): void {
    this.visibleColumns = this.visibleColumns.map((c) =>
      c.name === name ? { ...c, width } : c,
    )
    this.allColumns = this.allColumns.map((e) =>
      e.col.name === name ? { ...e, col: { ...e.col, width } } : e,
    )
  }

  setSort(column: string, direction: 'asc' | 'desc'): void {
    this.sortState = { column, direction }
    this.#save()
  }

  reset(): void {
    if (this.#saveTimer !== null) {
      clearTimeout(this.#saveTimer)
      this.#saveTimer = null
    }
    ConfigService.DeleteColumnPrefs(this.#gvr)
    this.#applyPrefs(null)
  }

  async setCompact(value: boolean): Promise<void> {
    this.compact = value
    await ConfigService.SetCompactRows(value)
  }

  #buildPrefs(): GVRColumnPrefs {
    return new GVRColumnPrefs({
      order: this.visibleColumns.map((c) => c.name),
      columns: Object.fromEntries(
        this.allColumns
          .filter(({ col }) => col.width !== undefined)
          .map(({ col }) => [col.name, new ColumnSettings({ width: col.width })]),
      ),
      sort: this.sortState
        ? new SortPrefs({ column: this.sortState.column, direction: this.sortState.direction })
        : null,
    })
  }

  #save(): void {
    if (this.#saveTimer !== null) {
      clearTimeout(this.#saveTimer)
      this.#saveTimer = null
    }
    ConfigService.SetColumnPrefs(this.#gvr, this.#buildPrefs())
  }

  #debouncedSave(): void {
    if (this.#saveTimer !== null) clearTimeout(this.#saveTimer)
    this.#saveTimer = setTimeout(() => {
      this.#saveTimer = null
      ConfigService.SetColumnPrefs(this.#gvr, this.#buildPrefs())
    }, 300)
  }
}

export const columnStore = new ColumnStore()
