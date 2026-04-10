<script lang="ts">
  import { Dialog } from 'bits-ui'
  import { push } from 'svelte-spa-router'
  import { Search } from 'lucide-svelte'
  import { clusterStore } from '$lib/stores/cluster.svelte'
  import { descriptorRegistry } from '$lib/registry/index'
  import { slotRegistry } from '$lib/plugins/slots.svelte.js'
  import { createResourceStore } from '$lib/stores/createResource.svelte'

  let { open = $bindable(false) }: { open: boolean } = $props()

  let query = $state('')
  let selectedIndex = $state(0)
  let inputEl: HTMLInputElement

  interface PaletteItem {
    id: string
    label: string
    subtitle?: string
    category: string
    action: () => void
  }

  function buildItems(): PaletteItem[] {
    const items: PaletteItem[] = []
    const ctx = clusterStore.activeContext

    if (ctx) {
      for (const desc of descriptorRegistry.list()) {
        items.push({
          id: `nav:${ctx}:${desc.gvr}`,
          label: desc.kind || desc.resource,
          subtitle: `${ctx} · ${desc.gvr}`,
          category: 'Navigate',
          action: () => {
            push(`/c/${encodeURIComponent(ctx)}/${desc.gvr}`)
            open = false
          },
        })
      }

      items.push({
        id: `cluster:overview:${ctx}`,
        label: 'Cluster Overview',
        subtitle: ctx,
        category: 'Navigate',
        action: () => {
          push(`/c/${encodeURIComponent(ctx)}`)
          open = false
        },
      })

      items.push({
        id: `events:${ctx}`,
        label: 'Event Stream',
        subtitle: ctx,
        category: 'Navigate',
        action: () => {
          push(`/c/${encodeURIComponent(ctx)}/events`)
          open = false
        },
      })

      items.push({
        id: 'create:resource',
        label: 'Create Resource',
        subtitle: 'Open template picker',
        category: 'Actions',
        action: () => {
          open = false
          createResourceStore.openDialog()
        },
      })
    }

    for (const c of clusterStore.contexts) {
      const status = clusterStore.connectionStatus[c.name] ?? 'disconnected'
      if (status !== 'connected') {
        items.push({
          id: `connect:${c.name}`,
          label: `Connect to ${c.name}`,
          category: 'Clusters',
          action: async () => {
            open = false
            await clusterStore.connect(c.name)
          },
        })
      }
    }

    for (const cmd of slotRegistry.getCommands()) {
      items.push({
        id: `plugin:${cmd.pluginName}:${cmd.id}`,
        label: cmd.label,
        subtitle: cmd.pluginName,
        category: 'Plugins',
        action: () => {
          open = false
          cmd.action()
        },
      })
    }

    return items
  }

  const filtered = $derived.by(() => {
    const all = buildItems()
    if (!query.trim()) return all.slice(0, 20)
    const q = query.toLowerCase()
    return all
      .filter((i) => i.label.toLowerCase().includes(q) || (i.subtitle ?? '').toLowerCase().includes(q))
      .slice(0, 20)
  })

  $effect(() => {
    if (open) {
      query = ''
      selectedIndex = 0
      // Focus input after dialog opens
      requestAnimationFrame(() => inputEl?.focus())
    }
  })

  // Reset selection when results change
  $effect(() => {
    void filtered
    selectedIndex = 0
  })

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      selectedIndex = Math.min(selectedIndex + 1, filtered.length - 1)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      selectedIndex = Math.max(selectedIndex - 1, 0)
    } else if (e.key === 'Enter') {
      e.preventDefault()
      filtered[selectedIndex]?.action()
    } else if (e.key === 'Escape') {
      open = false
    }
  }

  // Group items by category for display
  const grouped = $derived.by(() => {
    const map = new Map<string, PaletteItem[]>()
    for (const item of filtered) {
      const group = map.get(item.category) ?? []
      group.push(item)
      map.set(item.category, group)
    }
    return map
  })

  // Flat index for keyboard selection
  const flatItems = $derived(filtered)
</script>

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/60 z-40" />
    <Dialog.Content
      class="fixed left-1/2 top-[20%] -translate-x-1/2 z-50 w-[560px] max-w-[92vw] bg-surface border border-border rounded-xl shadow-2xl overflow-hidden"
      onkeydown={onKeydown}
    >
      <div class="flex items-center gap-2 px-4 py-3 border-b border-border">
        <Search size={16} class="text-muted shrink-0" />
        <input
          bind:this={inputEl}
          bind:value={query}
          placeholder="Search resources, navigate, run actions…"
          class="flex-1 bg-transparent text-sm outline-none placeholder:text-muted"
          aria-label="Command palette search"
        />
        <kbd class="text-xs text-muted bg-bg border border-border rounded px-1.5 py-0.5">Esc</kbd>
      </div>

      <div class="max-h-96 overflow-y-auto py-1" role="listbox" aria-label="Results">
        {#if filtered.length === 0}
          <p class="text-sm text-muted text-center py-8">No results</p>
        {:else}
          {#each grouped as [category, items] (category)}
            <div>
              <div class="px-3 py-1.5 text-xs font-medium text-muted uppercase tracking-wide">
                {category}
              </div>
              {#each items as item (item.id)}
                {@const idx = flatItems.indexOf(item)}
                <button
                  role="option"
                  aria-selected={idx === selectedIndex}
                  class="w-full text-left px-3 py-2 flex items-center gap-3 hover:bg-surface-hover transition-colors {idx === selectedIndex ? 'bg-surface-hover' : ''}"
                  onclick={item.action}
                  onmouseenter={() => { selectedIndex = idx }}
                >
                  <span class="text-sm flex-1 truncate">{item.label}</span>
                  {#if item.subtitle}
                    <span class="text-xs text-muted truncate max-w-[200px]">{item.subtitle}</span>
                  {/if}
                </button>
              {/each}
            </div>
          {/each}
        {/if}
      </div>

      <div class="px-4 py-2 border-t border-border flex items-center gap-4 text-xs text-muted">
        <span><kbd class="bg-bg border border-border rounded px-1">↑↓</kbd> navigate</span>
        <span><kbd class="bg-bg border border-border rounded px-1">↵</kbd> select</span>
        <span><kbd class="bg-bg border border-border rounded px-1">Ctrl+K</kbd> close</span>
      </div>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
