<script lang="ts">
  import { onDestroy } from 'svelte'
  import { untrack } from 'svelte'
  import ResourceList from '$lib/components/ResourceList.svelte'
  import ResourceDetail from '$lib/components/ResourceDetail.svelte'
  import { DetailDrawer } from '@klados/ui'
  import CreateResourceDialog from '$lib/components/CreateResourceDialog.svelte'
  import * as ResourceService from '../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import * as MetricsService from '../../bindings/github.com/Vilsol/klados/internal/services/metricsservice.js'
  import { createResourceStore } from '$lib/stores/resource.svelte'
  import { descriptorRegistry } from '$lib/registry/index'
  import { registryLoaded } from '$lib/registry/loaded.svelte'
  import { clusterStore } from '$lib/stores/cluster.svelte'
  import { sessionStore } from '$lib/stores/session.svelte'
  import { Plus } from 'lucide-svelte'
  import type { MetricResult } from '$lib/components/charts/types'

  let { params = {} }: { params?: Record<string, string> } = $props()

  const ctxName = $derived(params.ctx ?? '')
  const gvr = $derived(params.gvr ?? '')
  const selectedNamespaces = $derived(clusterStore.getSelectedNamespaces(ctxName))
  const descriptor = $derived(registryLoaded() ? descriptorRegistry.get(gvr) : null)
  const rawWatchNamespace = $derived(selectedNamespaces.length === 1 ? selectedNamespaces[0] : '')
  const watchNamespace = $derived(descriptor?.clusterScoped ? '' : rawWatchNamespace)

  // Keep activeContext in sync with the current tab's context for the header
  $effect(() => { if (ctxName) clusterStore.setActiveContext(ctxName) })

  let listScrollContainer = $state<HTMLDivElement | undefined>()

  // Restore scroll position after items load
  $effect(() => {
    if (!listScrollContainer || store.loading) return
    const tab = sessionStore.tabs[sessionStore.activeTabIndex]
    const saved = tab?.scrollPosition
    if (saved) requestAnimationFrame(() => {
      if (listScrollContainer) listScrollContainer.scrollTop = saved
    })
  })

  onDestroy(() => {
    if (listScrollContainer) {
      sessionStore.saveScrollPosition(sessionStore.activeTabIndex, listScrollContainer.scrollTop)
    }
  })

  const store = createResourceStore()

  $effect(() => {
    if (ctxName && gvr && descriptor) {
      store.start(ctxName, gvr, watchNamespace)
    }
    return () => store.stop()
  })

  // Close drawer when GVR changes
  $effect(() => { gvr; selectedItem = null })

  let createOpen = $state(false)
  let selectedItem = $state<Record<string, any> | null>(null)
  const selectedName = $derived<string | null>(
    selectedItem ? `${selectedItem.metadata?.name ?? ''}/${selectedItem.metadata?.namespace ?? ''}` : null
  )

  // Keep selected item in sync with live watch updates
  $effect(() => {
    if (!selectedItem) return
    const name = selectedItem.metadata?.name
    const ns = selectedItem.metadata?.namespace
    const fresh = store.items.find(
      (i) => i.metadata?.name === name && i.metadata?.namespace === ns
    )
    if (fresh) selectedItem = fresh
  })

  const sparklineGvrs = ['core.v1.pods', 'core.v1.nodes']
  let sparklineColumns = $state<string[]>([])
  let sparklineData = $state<Record<string, MetricResult[]>>({})

  // Reset sparkline columns when GVR changes
  $effect(() => { gvr; sparklineColumns = []; sparklineData = {} })

  // Sparkline polling
  $effect(() => {
    const enabled = sparklineColumns.length > 0
    const ctx = ctxName
    const g = gvr
    const ns = watchNamespace
    if (!enabled || !ctx || !g || !sparklineGvrs.includes(g)) return

    async function poll() {
      try {
        const result = await MetricsService.GetListMetrics(ctx, g, ns)
        untrack(() => {
          const data: Record<string, MetricResult[]> = {}
          if (result) {
            for (const [k, v] of Object.entries(result)) {
              if (v) data[k] = v as MetricResult[]
            }
          }
          sparklineData = data
        })
      } catch {
        untrack(() => { sparklineData = {} })
      }
    }

    poll()
    const id = setInterval(poll, 15_000)
    return () => clearInterval(id)
  })

  async function refresh() {
    if (ctxName && gvr) {
      await store.start(ctxName, gvr, watchNamespace)
    }
  }
</script>

<div class="flex flex-col h-full">
  <div class="shrink-0 px-4 py-3 border-b border-border flex items-center gap-2">
    <h1 class="text-sm font-semibold">{gvr.split('.').at(-1) ?? gvr}</h1>
    {#if !descriptor?.clusterScoped}
      {#if selectedNamespaces.length === 1}
        <span class="text-xs text-muted border border-border rounded px-1.5 py-0.5">{selectedNamespaces[0]}</span>
      {:else if selectedNamespaces.length > 1}
        <span class="text-xs text-muted border border-border rounded px-1.5 py-0.5">{selectedNamespaces.length} namespaces</span>
      {/if}
    {/if}
    <div class="flex-1"></div>
    <button
      onclick={() => createOpen = true}
      class="flex items-center gap-1 text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
      title="Create resource"
    >
      <Plus size={12} />
      Create
    </button>
  </div>

  <div class="flex-1 overflow-hidden relative">
    {#if descriptor}
      <ResourceList
        items={store.items}
        columns={descriptor.columns}
        contextName={ctxName}
        {gvr}
        {selectedNamespaces}
        loading={store.loading}
        error={store.error}
        {selectedName}
        bind:scrollContainer={listScrollContainer}
        onrefresh={refresh}
        onselect={(item) => selectedItem = item}
        {sparklineGvrs}
        {sparklineData}
        {sparklineColumns}
        onSparklineToggle={(cols) => sparklineColumns = cols}
      />

      {#if selectedItem}
        <DetailDrawer
          item={selectedItem}
          {ctxName}
          {gvr}
          onclose={() => selectedItem = null}
          onFetchResource={async (ctx, g, ns, n) => {
            try { return await ResourceService.GetResource(ctx, g, ns, n) } catch { return null }
          }}
        >
          {#snippet children({ obj, onrefresh })}
            <ResourceDetail
              {obj}
              {descriptor}
              {ctxName}
              {gvr}
              namespace={obj.metadata?.namespace ?? ''}
              name={obj.metadata?.name ?? ''}
              {onrefresh}
            />
          {/snippet}
        </DetailDrawer>
      {/if}
    {:else}

      <div class="flex-1 flex items-center justify-center text-sm text-muted">Loading...</div>
    {/if}
  </div>
</div>

<CreateResourceDialog
  bind:open={createOpen}
  {ctxName}
  {gvr}
  defaultNamespace={selectedNamespaces[0] ?? 'default'}
  onsuccess={refresh}
/>
