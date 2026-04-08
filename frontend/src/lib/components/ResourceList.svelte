<script lang="ts">
  import { createVirtualizer } from '@tanstack/svelte-virtual'
  import { ArrowUpDown, ArrowUp, ArrowDown, Trash2, RefreshCw, Columns3 } from 'lucide-svelte'
  import { ConfirmDialog } from '@klados/ui'
  import { notificationStore } from '$lib/stores/notification.svelte'
  import { evalExpr, defaultAlign, type ColumnDef, type RenderType } from '$lib/registry/index'
  import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import { formatAge } from '$lib/utils/age'
  import { onMount } from 'svelte'
  import { slotRegistry } from '$lib/plugins/slots.svelte.js'
  import { loadPluginComponent } from '$lib/plugins/loader.js'
  import { streamingStore } from '$lib/stores/streaming.svelte.js'
  import Sparkline from './charts/Sparkline.svelte'
  import type { MetricResult } from './charts/types'
  import { columnStore } from '$lib/stores/columns.svelte'
  import { clusterStore } from '$lib/stores/cluster.svelte'

  let now = $state(Date.now())
  onMount(() => {
    const id = setInterval(() => { now = Date.now() }, 1_000)
    return () => clearInterval(id)
  })

  let {
    items,
    contextName,
    gvr,
    selectedNamespaces = [],
    loading = false,
    error = null,
    selectedName = null,
    scrollContainer = $bindable<HTMLDivElement | undefined>(undefined),
    onrefresh,
    onselect,
    sparklineGvrs = [],
    sparklineData = {},
    sparklineColumns = [],
    onSparklineToggle,
  }: {
    items: Record<string, any>[]
    contextName: string
    gvr: string
    selectedNamespaces?: string[]
    loading?: boolean
    error?: string | null
    selectedName?: string | null
    scrollContainer?: HTMLDivElement
    onrefresh?: () => void
    onselect?: (item: Record<string, any>) => void
    sparklineGvrs?: string[]
    sparklineData?: Record<string, MetricResult[]>
    sparklineColumns?: string[]
    onSparklineToggle?: (columns: string[]) => void
  } = $props()

  let filterText = $state('')
  let deleteTarget = $state<{ namespace: string; name: string } | null>(null)
  let confirmOpen = $state(false)
  let ctxMenu = $state<{ x: number; y: number; item: Record<string, any> } | null>(null)
  let columnMenuOpen = $state(false)
  let resizing = $state<{ name: string; startX: number; startWidth: number } | null>(null)

  const hasSparklines = $derived(sparklineGvrs.includes(gvr))
  const availableSparklineCols = ['CPU', 'Memory']

  function toggleSparklineCol(col: string) {
    const current = sparklineColumns
    const next = current.includes(col) ? current.filter(c => c !== col) : [...current, col]
    onSparklineToggle?.(next)
  }

  function getSparklinePoints(itemName: string, metricName: string): { t: number; v: number }[] {
    const metrics = sparklineData[itemName]
    if (!metrics) return []
    const metric = metrics.find(m => m.name === metricName)
    if (!metric?.series?.[0]?.points) return []
    return metric.series[0].points
  }

  const pluginColumns = $derived(slotRegistry.getListColumns(gvr))
  const pluginMenuItems = $derived(slotRegistry.getContextMenuItems(gvr))
  const basePluginURL = $derived(
    streamingStore.config
      ? `http://127.0.0.1:${streamingStore.config.port}/${streamingStore.config.token}/plugins`
      : null
  )

  $effect(() => {
    if (!ctxMenu) return
    const close = () => { ctxMenu = null }
    window.addEventListener('click', close, { once: true })
    return () => window.removeEventListener('click', close)
  })

  $effect(() => {
    if (!columnMenuOpen) return
    const close = () => { columnMenuOpen = false }
    const timer = setTimeout(() => window.addEventListener('click', close, { once: true }), 0)
    return () => { clearTimeout(timer); window.removeEventListener('click', close) }
  })

  // Scroll to top when GVR changes
  $effect(() => {
    gvr
    filterText = ''
    if (scrollContainer) scrollContainer.scrollTop = 0
  })

  const filtered = $derived.by(() => {
    let result = items
    if (selectedNamespaces.length > 1) {
      result = result.filter((item) => selectedNamespaces.includes(item.metadata?.namespace ?? ''))
    }
    if (filterText.trim()) {
      const q = filterText.trim().toLowerCase()
      result = result.filter((item) => {
        const labels = item.metadata?.labels ?? {}
        const labelsStr = Object.entries(labels)
          .map(([k, v]) => `${k}=${v}`)
          .join(',')
        return labelsStr.includes(q) || (item.metadata?.name ?? '').toLowerCase().includes(q)
      })
    }
    if (columnStore.sortState) {
      const { column, direction } = columnStore.sortState
      const col = columnStore.visibleColumns.find((c) => c.name === column)
      if (col?.expr) {
        result = [...result].sort((a, b) => {
          const av = String(evalExpr(col.expr, a) ?? '')
          const bv = String(evalExpr(col.expr, b) ?? '')
          return direction === 'asc' ? av.localeCompare(bv) : bv.localeCompare(av)
        })
      }
    }
    return result
  })

  const tooManyForSparklines = $derived(filtered.length > 200)

  const rowHeight = $derived(columnStore.compact ? 28 : 36)

  const virtualizer = $derived(
    createVirtualizer({
      count: filtered.length,
      getScrollElement: () => scrollContainer ?? null,
      estimateSize: () => rowHeight,
      overscan: 10,
    }),
  )

  function toggleSort(name: string) {
    const current = columnStore.sortState
    if (current?.column === name) {
      columnStore.setSort(name, current.direction === 'asc' ? 'desc' : 'asc')
    } else {
      columnStore.setSort(name, 'asc')
    }
  }

  function renderCell(col: ColumnDef, item: Record<string, any>) {
    return evalExpr(col.expr, item)
  }

  function renderValue(value: any, renderType: RenderType): string {
    if (value == null) return ''
    if (renderType === 'age') return formatAge(String(value), now)
    return String(value)
  }

  function badgeClass(value: any): string {
    const v = String(value ?? '').toLowerCase()
    if (['running', 'active', 'bound', 'available', 'true'].includes(v))
      return 'bg-accent/20 text-accent border-accent/30'
    if (['error', 'crashloopbackoff', 'failed', 'oomkilled'].includes(v))
      return 'bg-destructive/20 text-destructive border-destructive/30'
    if (['pending', 'terminating'].includes(v))
      return 'bg-muted/20 text-muted border-muted/30'
    return 'bg-muted/10 text-fg border-border'
  }

  function alignClass(col: ColumnDef): string {
    const align = col.align ?? defaultAlign(col.renderType)
    return align === 'right' ? 'text-right' : align === 'center' ? 'text-center' : 'text-left'
  }

  async function confirmDelete() {
    if (!deleteTarget) return
    const { namespace, name } = deleteTarget
    try {
      await ResourceService.DeleteResource(contextName, gvr, namespace, name)
      notificationStore.push(`Deleted ${name}`, 'success')
    } catch (e: any) {
      notificationStore.push(`Failed to delete: ${e?.message ?? e}`, 'error')
    }
    deleteTarget = null
  }

  function requestDelete(item: Record<string, any>) {
    deleteTarget = {
      namespace: item.metadata?.namespace ?? '',
      name: item.metadata?.name ?? '',
    }
    confirmOpen = true
  }

  const gridTemplateCols = $derived(
    columnStore.visibleColumns
      .map((c) => c.width ? `${c.width}px` : 'minmax(20px, 1fr)')
      .join(' ')
    + (pluginColumns.length ? ' ' + pluginColumns.map(() => '1fr').join(' ') : '')
    + (sparklineColumns.length ? ' ' + sparklineColumns.map(() => '80px').join(' ') : '')
    + ' 36px'
  )

  function startResize(e: MouseEvent, col: ColumnDef) {
    e.preventDefault()
    resizing = { name: col.name, startX: e.clientX, startWidth: col.width ?? 100 }
    window.addEventListener('mousemove', onResizeMove)
    window.addEventListener('mouseup', onResizeUp, { once: true })
  }

  function onResizeMove(e: MouseEvent) {
    if (!resizing) return
    const delta = e.clientX - resizing.startX
    const newWidth = Math.max(20, resizing.startWidth + delta)
    columnStore.resizeColumn(resizing.name, newWidth)
  }

  function onResizeUp() {
    window.removeEventListener('mousemove', onResizeMove)
    resizing = null
  }

  function autoFit(name: string) {
    const cells = scrollContainer?.querySelectorAll(`[data-col="${name}"]`)
    if (!cells) return
    let max = 60
    for (const cell of cells) {
      max = Math.max(max, (cell as HTMLElement).scrollWidth)
    }
    columnStore.autoFitColumn(name, max)
  }
</script>

<div class="flex flex-col h-full">
  <div class="flex items-center gap-2 px-3 py-2 border-b border-border shrink-0">
    <input
      type="text"
      placeholder="Filter by name or label (key=value)..."
      bind:value={filterText}
      class="flex-1 text-sm bg-transparent outline-none placeholder-muted"
    />
    <span class="text-xs text-muted">{filtered.length} items</span>
    {#if hasSparklines}
      <div class="relative">
        <button
          onclick={() => columnMenuOpen = !columnMenuOpen}
          class="p-1 rounded hover:bg-surface-hover transition-colors"
          title="Toggle sparkline columns"
          aria-label="Toggle columns"
        >
          <Columns3 size={14} />
        </button>
        {#if columnMenuOpen}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div class="absolute right-0 top-full mt-1 z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-40"
            onclick={(e) => e.stopPropagation()}
            onkeydown={(e) => e.stopPropagation()}>
            {#each availableSparklineCols as col}
              <label class="flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-surface-hover cursor-pointer">
                <input
                  type="checkbox"
                  checked={sparklineColumns.includes(col)}
                  onchange={() => toggleSparklineCol(col)}
                  class="rounded border-border"
                />
                {col} Sparkline
              </label>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
    {#if onrefresh}
      <button
        onclick={onrefresh}
        class="p-1 rounded hover:bg-surface-hover transition-colors"
        title="Refresh"
        aria-label="Refresh"
      >
        <RefreshCw size={14} class={loading ? 'animate-spin' : ''} />
      </button>
    {/if}
  </div>

  {#if error}
    <div class="p-4 text-sm text-destructive">{error}</div>
  {:else}
    <div class="grid text-xs font-semibold uppercase tracking-wider text-muted border-b border-border shrink-0 px-2"
      style="grid-template-columns: {gridTemplateCols}"
    >
      {#each columnStore.visibleColumns as col, i}
        <div class="relative">
          <button
            onclick={() => toggleSort(col.name)}
            class="flex items-center gap-1 py-2 px-1 hover:text-fg transition-colors text-left w-full
              {i === 0 ? 'sticky left-0 z-10 bg-bg shadow-[2px_0_4px_rgba(0,0,0,0.08)] dark:shadow-[2px_0_4px_rgba(0,0,0,0.3)]' : ''}"
          >
            {col.name}
            {#if columnStore.sortState?.column === col.name}
              {#if columnStore.sortState.direction === 'asc'}<ArrowUp size={10} />{:else}<ArrowDown size={10} />{/if}
            {:else}
              <ArrowUpDown size={10} class="opacity-30" />
            {/if}
          </button>
          {#if i < columnStore.visibleColumns.length - 1}
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <div
              class="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-accent/50 z-20"
              onmousedown={(e) => startResize(e, col)}
              ondblclick={() => autoFit(col.name)}
            ></div>
          {/if}
        </div>
      {/each}
      {#each pluginColumns as pcol (pcol.id)}
        <div class="py-2 px-1">{pcol.label}</div>
      {/each}
      {#each sparklineColumns as scol}
        <div class="py-2 px-1">{scol}</div>
      {/each}
      <div></div>
    </div>

    <div bind:this={scrollContainer} class="flex-1 overflow-y-auto">
      {#if loading}
        <div class="flex items-center justify-center py-12 text-sm text-muted">Loading...</div>
      {:else if filtered.length === 0}
        <div class="flex items-center justify-center py-12 text-sm text-muted">No resources found</div>
      {:else}
        <div style="height: {$virtualizer.getTotalSize()}px; position: relative;">
          {#each $virtualizer.getVirtualItems() as row (row.index)}
            {@const item = filtered[row.index]}
            {@const isSelected = selectedName === `${item.metadata?.name ?? ''}/${item.metadata?.namespace ?? ''}`}
            <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
            <div
              class="absolute top-0 left-0 right-0 flex items-center px-2 transition-colors group
                {isSelected ? 'bg-accent/10 border-l-2 border-accent' : 'hover:bg-surface-hover border-l-2 border-transparent'}
                {onselect ? 'cursor-pointer' : ''}"
              style="transform: translateY({row.start}px); height: {rowHeight}px;"
              role={onselect ? 'button' : undefined}
              tabindex={onselect ? 0 : undefined}
              onclick={() => onselect?.(item)}
              onkeydown={(e) => { if (e.key === 'Enter') onselect?.(item) }}
              oncontextmenu={(e) => { e.preventDefault(); e.stopPropagation(); ctxMenu = { x: e.clientX, y: e.clientY, item } }}
            >
              <div
                class="grid flex-1 min-w-0"
                style="grid-template-columns: {gridTemplateCols}"
              >
                {#each columnStore.visibleColumns as col, i}
                  {@const value = renderCell(col, item)}
                  <div
                    class="px-1 truncate text-sm {alignClass(col)}
                      {i === 0 ? 'sticky left-0 z-10 bg-bg shadow-[2px_0_4px_rgba(0,0,0,0.08)] dark:shadow-[2px_0_4px_rgba(0,0,0,0.3)]' : ''}
                      {col.name === 'Namespace' ? 'cursor-pointer hover:text-accent' : ''}"
                    data-col={col.name}
                    onclick={col.name === 'Namespace' ? (e) => { e.stopPropagation(); clusterStore.setNamespaces(contextName, [String(value)]) } : undefined}
                    role={col.name === 'Namespace' ? 'button' : undefined}
                  >
                    {#if col.renderType === 'badge'}
                      <span class="px-1.5 py-0.5 text-xs rounded border {badgeClass(value)}"
                            title={renderValue(value, col.renderType)}>
                        {renderValue(value, col.renderType)}
                      </span>
                    {:else}
                      <span class={col.renderType === 'age' ? 'text-muted' : ''}
                            title={renderValue(value, col.renderType)}>
                        {renderValue(value, col.renderType)}
                      </span>
                    {/if}
                  </div>
                {/each}
                {#each pluginColumns as pcol (pcol.id)}
                  <div class="px-1 flex items-center overflow-hidden text-sm">
                    {#if basePluginURL}
                      {#await loadPluginComponent(pcol.pluginName, pcol.component, basePluginURL) then Cmp}
                        {#if Cmp}<Cmp resource={item} />{/if}
                      {/await}
                    {/if}
                  </div>
                {/each}
                {#each sparklineColumns as scol}
                  <div class="px-1 flex items-center overflow-hidden">
                    {#if tooManyForSparklines}
                      <span class="text-xs text-muted" title="Sparklines disabled for >200 resources">Too many</span>
                    {:else}
                      {@const pts = getSparklinePoints(item.metadata?.name ?? '', scol)}
                      {#if pts.length > 0}
                        <Sparkline points={pts} height={20} />
                      {:else}
                        <div style="height: 20px;"></div>
                      {/if}
                    {/if}
                  </div>
                {/each}
                <div class="flex items-center justify-end">
                  <button
                    onclick={(e) => { e.stopPropagation(); requestDelete(item) }}
                    class="p-1 rounded opacity-0 group-hover:opacity-60 hover:!opacity-100 hover:text-destructive transition-all"
                    title="Delete"
                    aria-label="Delete {item.metadata?.name}"
                  >
                    <Trash2 size={13} />
                  </button>
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>

{#if ctxMenu}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-36"
    style="left:{ctxMenu.x}px; top:{ctxMenu.y}px"
    onclick={(e) => e.stopPropagation()}
    onkeydown={(e) => e.stopPropagation()}
  >
    {#each pluginMenuItems as mi (mi.id)}
      {#if basePluginURL}
        {#await loadPluginComponent(mi.pluginName, mi.component, basePluginURL) then Cmp}
          {#if Cmp}
            {@const menuItem = ctxMenu}
            <Cmp resource={menuItem.item} onclose={() => { ctxMenu = null }} />
          {/if}
        {/await}
      {/if}
    {/each}
    <button
      class="w-full text-left px-3 py-1.5 text-sm text-destructive hover:bg-surface-hover"
      onclick={() => { requestDelete(ctxMenu!.item); ctxMenu = null }}
    >
      Delete
    </button>
  </div>
{/if}

<ConfirmDialog
  bind:open={confirmOpen}
  title="Delete resource"
  message="Delete {deleteTarget?.name}? This action cannot be undone."
  confirmLabel="Delete"
  onconfirm={confirmDelete}
/>
