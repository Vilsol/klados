class SelectionStore {
  selectedKeys = $state<Set<string>>(new Set())
  selectedGVR = $state<string>('')
  selectedItems = $state<Map<string, Record<string, any>>>(new Map())
  visibleKeys = $state<Set<string>>(new Set())
  private lastToggled = $state<string | null>(null)

  count = $derived(this.selectedKeys.size)
  notVisibleCount = $derived(
    [...this.selectedKeys].filter((k) => !this.visibleKeys.has(k)).length
  )

  isSelected(key: string): boolean {
    return this.selectedKeys.has(key)
  }

  select(key: string, item: Record<string, any>): void {
    const keys = new Set(this.selectedKeys)
    keys.add(key)
    this.selectedKeys = keys

    const items = new Map(this.selectedItems)
    items.set(key, item)
    this.selectedItems = items

    this.lastToggled = key
  }

  deselect(key: string): void {
    const keys = new Set(this.selectedKeys)
    keys.delete(key)
    this.selectedKeys = keys

    const items = new Map(this.selectedItems)
    items.delete(key)
    this.selectedItems = items

    if (this.lastToggled === key) {
      this.lastToggled = null
    }
  }

  toggle(key: string, item: Record<string, any>): void {
    if (this.isSelected(key)) {
      this.deselect(key)
    } else {
      this.select(key, item)
    }
  }

  selectRange(
    toKey: string,
    orderedKeys: string[],
    itemsByKey: Map<string, Record<string, any>>
  ): void {
    const from = this.lastToggled
    if (!from) {
      const item = itemsByKey.get(toKey)
      if (item) this.select(toKey, item)
      return
    }

    const fromIdx = orderedKeys.indexOf(from)
    const toIdx = orderedKeys.indexOf(toKey)
    if (fromIdx === -1 || toIdx === -1) {
      const item = itemsByKey.get(toKey)
      if (item) this.select(toKey, item)
      return
    }

    const [start, end] = fromIdx <= toIdx ? [fromIdx, toIdx] : [toIdx, fromIdx]
    const rangeKeys = orderedKeys.slice(start, end + 1)

    const keys = new Set(this.selectedKeys)
    const items = new Map(this.selectedItems)

    for (const k of rangeKeys) {
      keys.add(k)
      const item = itemsByKey.get(k)
      if (item) items.set(k, item)
    }

    this.selectedKeys = keys
    this.selectedItems = items
    this.lastToggled = toKey
  }

  selectAll(keys: string[], itemsByKey: Map<string, Record<string, any>>): void {
    const newKeys = new Set(this.selectedKeys)
    const newItems = new Map(this.selectedItems)

    for (const k of keys) {
      newKeys.add(k)
      const item = itemsByKey.get(k)
      if (item) newItems.set(k, item)
    }

    this.selectedKeys = newKeys
    this.selectedItems = newItems
  }

  deselectAll(): void {
    this.selectedKeys = new Set()
    this.selectedItems = new Map()
    this.lastToggled = null
  }

  clear(): void {
    this.selectedKeys = new Set()
    this.selectedItems = new Map()
    this.selectedGVR = ''
    this.lastToggled = null
    this.visibleKeys = new Set()
  }

  items(): Record<string, any>[] {
    return [...this.selectedKeys]
      .map((k) => this.selectedItems.get(k))
      .filter((v): v is Record<string, any> => v !== undefined)
  }

  setVisibleKeys(keys: Set<string>): void {
    this.visibleKeys = keys
  }

  setGVR(gvr: string): void {
    if (this.selectedGVR !== gvr) {
      this.clear()
    }
    this.selectedGVR = gvr
  }

  deselectKeys(keys: string[]): void {
    const newKeys = new Set(this.selectedKeys)
    const newItems = new Map(this.selectedItems)

    for (const k of keys) {
      newKeys.delete(k)
      newItems.delete(k)
    }

    this.selectedKeys = newKeys
    this.selectedItems = newItems
  }
}

export const selectionStore = new SelectionStore()
