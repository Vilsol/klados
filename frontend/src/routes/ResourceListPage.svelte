<script lang="ts">
  import { onDestroy } from 'svelte'
  import ResourceList from '$lib/components/ResourceList.svelte'
  import DetailDrawer from '$lib/components/DetailDrawer.svelte'
  import CreateResourceDialog from '$lib/components/CreateResourceDialog.svelte'
  import { createResourceStore } from '$lib/stores/resource.svelte'
  import { descriptorRegistry } from '$lib/registry/index'
  import { registryLoaded } from '$lib/registry/loaded.svelte'
  import { clusterStore } from '$lib/stores/cluster.svelte'
  import { sessionStore } from '$lib/stores/session.svelte'
  import { Plus } from 'lucide-svelte'

  let { params = {} }: { params?: Record<string, string> } = $props()

  const ctxName = $derived(params.ctx ?? '')
  const gvr = $derived(params.gvr ?? '')
  const selectedNamespaces = $derived(clusterStore.getSelectedNamespaces(ctxName))
  const watchNamespace = $derived(selectedNamespaces.length === 1 ? selectedNamespaces[0] : '')

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
    if (ctxName && gvr) {
      store.start(ctxName, gvr, watchNamespace)
    }
    return () => store.stop()
  })

  // Close drawer when GVR changes
  $effect(() => { gvr; selectedItem = null })

  const descriptor = $derived(registryLoaded() ? descriptorRegistry.get(gvr) : null)

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

  async function refresh() {
    if (ctxName && gvr) {
      await store.start(ctxName, gvr, watchNamespace)
    }
  }
</script>

<div class="flex flex-col h-full">
  <div class="shrink-0 px-4 py-3 border-b border-border flex items-center gap-2">
    <h1 class="text-sm font-semibold">{gvr.split('.').at(-1) ?? gvr}</h1>
    {#if selectedNamespaces.length === 1}
      <span class="text-xs text-muted border border-border rounded px-1.5 py-0.5">{selectedNamespaces[0]}</span>
    {:else if selectedNamespaces.length > 1}
      <span class="text-xs text-muted border border-border rounded px-1.5 py-0.5">{selectedNamespaces.length} namespaces</span>
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

  <div class="flex-1 relative overflow-hidden">
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
      />

      {#if selectedItem}
        <DetailDrawer
          item={selectedItem}
          {descriptor}
          {ctxName}
          {gvr}
          onclose={() => selectedItem = null}
        />
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
