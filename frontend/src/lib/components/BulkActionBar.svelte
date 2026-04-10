<script lang="ts">
  import { Trash2, Tag, StickyNote, Scale, Download, X } from 'lucide-svelte'
  import { selectionStore } from '$lib/stores/selection.svelte'
  import { exportItems } from '$lib/utils/export'
  import BulkDeleteDialog from './BulkDeleteDialog.svelte'
  import BulkMetadataDialog from './BulkMetadataDialog.svelte'
  import BulkScaleDialog from './BulkScaleDialog.svelte'

  let { contextName, gvr }: { contextName: string; gvr: string } = $props()

  let deleteOpen = $state(false)
  let labelOpen = $state(false)
  let annotateOpen = $state(false)
  let scaleOpen = $state(false)
  let exportMenuOpen = $state(false)

  const SCALABLE_GVRS = ['apps.v1.deployments', 'apps.v1.statefulsets']
  const canScale = $derived(SCALABLE_GVRS.includes(gvr))

  $effect(() => {
    if (!exportMenuOpen) return
    const close = () => { exportMenuOpen = false }
    const timer = setTimeout(() => window.addEventListener('click', close, { once: true }), 0)
    return () => { clearTimeout(timer); window.removeEventListener('click', close) }
  })

  function doExport(format: 'yaml' | 'json') {
    exportItems(selectionStore.items(), gvr, format)
    exportMenuOpen = false
  }
</script>

{#if selectionStore.count > 0}
  <div class="animate-slide-up fixed bottom-6 left-1/2 -translate-x-1/2 z-30 flex items-center gap-2 rounded-lg border border-border bg-surface px-4 py-2 shadow-lg">
    <span class="text-sm font-medium text-fg whitespace-nowrap">
      {selectionStore.count} selected{#if selectionStore.notVisibleCount > 0}
        <span class="text-muted-foreground"> ({selectionStore.notVisibleCount} not visible)</span>
      {/if}
    </span>

    <button
      class="ml-1 p-1 rounded hover:bg-surface-hover text-muted-foreground"
      onclick={() => selectionStore.deselectAll()}
      title="Clear selection"
    >
      <X class="w-4 h-4" />
    </button>

    <div class="w-px h-5 bg-border mx-1"></div>

    <button
      class="flex items-center gap-1.5 rounded px-2.5 py-1.5 text-sm font-medium hover:bg-surface-hover text-destructive"
      onclick={() => (deleteOpen = true)}
      title="Delete selected"
    >
      <Trash2 class="w-4 h-4" />
      Delete
    </button>

    <button
      class="flex items-center gap-1.5 rounded px-2.5 py-1.5 text-sm font-medium hover:bg-surface-hover text-fg"
      onclick={() => (labelOpen = true)}
      title="Edit labels"
    >
      <Tag class="w-4 h-4" />
      Labels
    </button>

    <button
      class="flex items-center gap-1.5 rounded px-2.5 py-1.5 text-sm font-medium hover:bg-surface-hover text-fg"
      onclick={() => (annotateOpen = true)}
      title="Edit annotations"
    >
      <StickyNote class="w-4 h-4" />
      Annotations
    </button>

    {#if canScale}
      <button
        class="flex items-center gap-1.5 rounded px-2.5 py-1.5 text-sm font-medium hover:bg-surface-hover text-fg"
        onclick={() => (scaleOpen = true)}
        title="Scale"
      >
        <Scale class="w-4 h-4" />
        Scale
      </button>
    {/if}

    <div class="relative">
      <button
        class="flex items-center gap-1.5 rounded px-2.5 py-1.5 text-sm font-medium hover:bg-surface-hover text-fg"
        onclick={() => (exportMenuOpen = !exportMenuOpen)}
        title="Export"
      >
        <Download class="w-4 h-4" />
        Export
      </button>
      {#if exportMenuOpen}
        <div class="absolute bottom-full mb-1 left-0 rounded-md border border-border bg-surface shadow-lg py-1 min-w-[100px]">
          <button
            class="block w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover text-fg"
            onclick={() => doExport('yaml')}
          >
            YAML
          </button>
          <button
            class="block w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover text-fg"
            onclick={() => doExport('json')}
          >
            JSON
          </button>
        </div>
      {/if}
    </div>
  </div>

  <BulkDeleteDialog bind:open={deleteOpen} {contextName} />
  <BulkMetadataDialog bind:open={labelOpen} mode="labels" {contextName} {gvr} />
  <BulkMetadataDialog bind:open={annotateOpen} mode="annotations" {contextName} {gvr} />
  <BulkScaleDialog bind:open={scaleOpen} {contextName} />
{/if}
