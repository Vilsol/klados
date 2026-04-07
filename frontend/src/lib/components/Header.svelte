<script lang="ts">
  import { Sun, Moon, Monitor, ChevronDown, Check, Plus, Trash2 } from 'lucide-svelte'
  import { getTheme, setTheme } from '$lib/theme.svelte'
  import { clusterStore } from '$lib/stores/cluster.svelte'
  import ConnectionIndicator from './ConnectionIndicator.svelte'
  import { ConfirmDialog } from '@klados/ui'
  import { push } from 'svelte-spa-router'
  import { slotRegistry } from '$lib/plugins/slots.svelte.js'
  import { loadPluginComponent } from '$lib/plugins/loader.js'
  import { streamingStore } from '$lib/stores/streaming.svelte.js'
  import { Events } from '@wailsio/runtime'
  import * as DrainService from '../../../bindings/github.com/Vilsol/klados/internal/services/drainservice.js'

  function cycleTheme() {
    const current = getTheme()
    const next = current === 'system' ? 'dark' : current === 'dark' ? 'light' : 'system'
    setTheme(next)
  }

  let currentTheme = $derived(getTheme())
  let nsDropdownOpen = $state(false)
  let nsSearch = $state('')
  let newNsName = $state('')
  let deleteTarget = $state<string | null>(null)
  let confirmDeleteOpen = $state(false)
  let nsCreateError = $state('')

  async function createNs() {
    if (!ctx || !newNsName.trim()) return
    nsCreateError = ''
    try {
      await clusterStore.createNamespace(ctx, newNsName.trim())
      newNsName = ''
    } catch (e: any) {
      nsCreateError = e?.message ?? String(e)
    }
  }

  async function confirmDelete() {
    if (!ctx || !deleteTarget) return
    await clusterStore.deleteNamespace(ctx, deleteTarget)
    deleteTarget = null
    confirmDeleteOpen = false
  }

  const ctx = $derived(clusterStore.activeContext)
  const selected = $derived(ctx ? clusterStore.getSelectedNamespaces(ctx) : [])

  let activeDrains = $state<string[]>([])

  $effect(() => {
    const currentCtx = ctx
    if (!currentCtx) { activeDrains = []; return }

    DrainService.ListActive(currentCtx).then((nodes: string[]) => {
      activeDrains = nodes ?? []
    })

    const unsub = Events.On(`drain:${currentCtx}:updated`, () => {
      DrainService.ListActive(currentCtx).then((nodes: string[]) => {
        activeDrains = nodes ?? []
      })
    })
    return unsub
  })

  const label = $derived(
    selected.length === 0
      ? 'All Namespaces'
      : selected.length === 1
        ? selected[0]
        : `${selected.length} namespaces`,
  )

  function selectOnly(ns: string) {
    if (ctx) clusterStore.setNamespaces(ctx, [ns])
    nsDropdownOpen = false
  }

  function toggleNs(ns: string) {
    if (!ctx) return
    const next = selected.includes(ns) ? selected.filter((n) => n !== ns) : [...selected, ns]
    clusterStore.setNamespaces(ctx, next)
  }

  const filteredNamespaces = $derived(
    nsSearch === ''
      ? (ctx ? clusterStore.getNamespaces(ctx) : [])
      : (ctx ? clusterStore.getNamespaces(ctx) : []).filter((ns) =>
          ns.toLowerCase().includes(nsSearch.toLowerCase()),
        ),
  )

  function selectAll() {
    if (ctx) clusterStore.setNamespaces(ctx, [])
    nsDropdownOpen = false
  }

  function handleClickOutside(e: MouseEvent) {
    if (!(e.target as HTMLElement).closest('[data-ns-dropdown]')) {
      nsDropdownOpen = false
      nsSearch = ''
    }
  }

  const basePluginURL = $derived(
    streamingStore.config
      ? `http://127.0.0.1:${streamingStore.config.port}/${streamingStore.config.token}/plugins`
      : null
  )
</script>

<svelte:document onclick={handleClickOutside} />

<ConfirmDialog
  bind:open={confirmDeleteOpen}
  title="Delete namespace"
  message="Delete namespace &quot;{deleteTarget}&quot;? This cannot be undone."
  onconfirm={confirmDelete}
/>

<header class="flex items-center h-12 px-4 border-b border-border bg-surface shrink-0 gap-4">
  <span class="font-semibold text-sm tracking-wide">Klados</span>

  <div class="flex items-center gap-2 ml-4">
    {#if ctx}
      <ConnectionIndicator
        status={clusterStore.connectionStatus[ctx] ?? 'disconnected'}
        clusterName={ctx}
      />
      <button
        onclick={() => push('/clusters')}
        class="text-sm font-medium hover:underline"
      >{ctx}</button>
    {:else}
      <button onclick={() => push('/clusters')} class="text-sm text-muted hover:underline">
        No cluster selected
      </button>
    {/if}
  </div>

  {#if ctx && clusterStore.getNamespaces(ctx).length > 0}
    <div class="relative ml-2" data-ns-dropdown>
      <button
        onclick={() => (nsDropdownOpen = !nsDropdownOpen)}
        class="flex items-center gap-1 text-sm bg-bg text-fg border border-border rounded px-2 py-1 hover:bg-surface-hover transition-colors"
      >
        {label}
        <ChevronDown size={14} />
      </button>

      {#if nsDropdownOpen}
        <div
          class="absolute top-full left-0 mt-1 w-64 max-h-72 overflow-y-auto rounded border border-border bg-bg shadow-lg z-50"
        >
          <!-- Search input -->
          <div class="px-2 py-1.5">
            <input
              type="text"
              bind:value={nsSearch}
              placeholder="Search namespaces…"
              class="w-full text-xs px-2 py-1 rounded border border-border bg-surface focus:outline-none focus:border-accent"
            />
          </div>

          <div class="border-t border-border my-0.5"></div>

          <!-- All Namespaces row -->
          <button
            onclick={selectAll}
            class="w-full flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-surface-hover transition-colors
              {selected.length === 0 ? 'font-medium text-fg' : 'text-muted'}"
          >
            <span class="w-4 shrink-0 flex items-center justify-center">
              {#if selected.length === 0}<Check size={12} />{/if}
            </span>
            All Namespaces
          </button>

          <div class="border-t border-border my-0.5"></div>

          {#each filteredNamespaces as ns}
            {@const isSelected = selected.includes(ns)}
            <div class="flex items-stretch hover:bg-surface-hover transition-colors group">
              <!-- checkbox area — toggles multi-select -->
              <button
                onclick={(e) => { e.stopPropagation(); toggleNs(ns) }}
                class="shrink-0 w-8 flex items-center justify-center"
                title="Add to selection"
                aria-label="{isSelected ? 'Deselect' : 'Select'} namespace {ns}"
              >
                <span
                  class="w-3.5 h-3.5 rounded border flex items-center justify-center transition-colors
                    {isSelected ? 'bg-accent border-accent text-bg' : 'border-border'}"
                >
                  {#if isSelected}<Check size={10} />{/if}
                </span>
              </button>

              <!-- name area — select only this namespace -->
              <button
                onclick={() => selectOnly(ns)}
                class="flex-1 text-left px-2 py-1.5 text-sm {isSelected ? 'font-medium' : ''}"
              >
                {ns}
              </button>

              <!-- delete button -->
              <button
                onclick={(e) => { e.stopPropagation(); deleteTarget = ns; confirmDeleteOpen = true }}
                class="shrink-0 w-7 flex items-center justify-center opacity-0 group-hover:opacity-100 text-muted hover:text-destructive transition-all"
                title="Delete namespace {ns}"
                aria-label="Delete namespace {ns}"
              >
                <Trash2 size={12} />
              </button>
            </div>
          {/each}

          <div class="border-t border-border my-0.5"></div>

          <!-- Create namespace -->
          <div class="px-2 py-1.5">
            {#if nsCreateError}
              <p class="text-xs text-destructive mb-1">{nsCreateError}</p>
            {/if}
            <div class="flex gap-1">
              <input
                type="text"
                bind:value={newNsName}
                placeholder="New namespace..."
                onkeydown={(e) => { if (e.key === 'Enter') createNs() }}
                class="flex-1 text-xs px-2 py-1 rounded border border-border bg-surface focus:outline-none focus:border-accent"
              />
              <button
                onclick={createNs}
                disabled={!newNsName.trim()}
                class="shrink-0 p-1 rounded border border-border hover:bg-surface-hover disabled:opacity-40 transition-colors"
                title="Create namespace"
              >
                <Plus size={12} />
              </button>
            </div>
          </div>
        </div>
      {/if}
    </div>
  {/if}

  {#if basePluginURL}
    {#each slotRegistry.getHeaderWidgets() as widget (widget.id)}
      {#await loadPluginComponent(widget.pluginName, widget.component, basePluginURL) then Cmp}
        {#if Cmp}
          <Cmp />
        {/if}
      {/await}
    {/each}
  {/if}

  {#if activeDrains.length > 0}
    <div class="flex items-center gap-1.5 text-xs px-2 py-1 rounded bg-amber-500/20 text-amber-400 border border-amber-500/30">
      <span class="inline-block w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse"></span>
      Draining {activeDrains.length} node{activeDrains.length === 1 ? '' : 's'}
    </div>
  {/if}

  <div class="ml-auto">
    <button
      onclick={cycleTheme}
      class="p-1.5 rounded hover:bg-surface-hover transition-colors"
      title="Theme: {currentTheme}"
      aria-label="Switch theme (current: {currentTheme})"
    >
      {#if currentTheme === 'dark'}
        <Moon size={16} />
      {:else if currentTheme === 'light'}
        <Sun size={16} />
      {:else}
        <Monitor size={16} />
      {/if}
    </button>
  </div>
</header>
