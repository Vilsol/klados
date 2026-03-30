<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { X } from 'lucide-svelte'
  import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import ResourceDetail from './ResourceDetail.svelte'
  import type { DescriptorDef } from '$lib/registry/index'

  let {
    item,
    descriptor,
    ctxName,
    gvr,
    onclose,
  }: {
    item: Record<string, any>
    descriptor: DescriptorDef
    ctxName: string
    gvr: string
    onclose: () => void
  } = $props()

  const name = $derived<string>(item.metadata?.name ?? '')
  const namespace = $derived<string>(item.metadata?.namespace ?? '')

  let obj = $state<Record<string, any>>(item)
  $effect(() => { obj = item })

  async function refresh() {
    try {
      const fresh = await ResourceService.GetResource(ctxName, gvr, namespace, name)
      if (fresh) obj = fresh
    } catch {
      // keep stale data on error
    }
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose()
  }

  // Resize state
  let drawerWidth = $state(640)
  let dragging = $state(false)
  let dragStartX = 0
  let dragStartWidth = 0

  let containerEl: HTMLElement | undefined

  function onResizeStart(e: MouseEvent) {
    dragging = true
    dragStartX = e.clientX
    dragStartWidth = drawerWidth
    document.body.style.cursor = 'ew-resize'
    document.body.style.userSelect = 'none'
    e.preventDefault()
  }

  function onResizeMove(e: MouseEvent) {
    if (!dragging) return
    const containerWidth = containerEl?.parentElement?.clientWidth ?? window.innerWidth
    const delta = dragStartX - e.clientX
    drawerWidth = Math.min(
      Math.max(dragStartWidth + delta, 280),
      containerWidth - 60,
    )
  }

  function onResizeEnd() {
    if (!dragging) return
    dragging = false
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }

  onMount(() => {
    document.addEventListener('keydown', onkeydown)
    document.addEventListener('mousemove', onResizeMove)
    document.addEventListener('mouseup', onResizeEnd)
  })

  onDestroy(() => {
    document.removeEventListener('keydown', onkeydown)
    document.removeEventListener('mousemove', onResizeMove)
    document.removeEventListener('mouseup', onResizeEnd)
  })
</script>

<!-- Drawer panel -->
<div
  bind:this={containerEl}
  class="absolute top-0 right-0 bottom-0 z-30 flex flex-col bg-bg border-l border-border shadow-2xl"
  style="width: {drawerWidth}px"
  role="dialog"
  aria-label="Resource detail: {name}"
>
  <!-- Resize handle -->
  <div
    class="absolute top-0 left-0 bottom-0 w-1 z-10 cursor-ew-resize transition-colors
      {dragging ? 'bg-accent/50' : 'hover:bg-accent/30'}"
    role="separator"
    aria-label="Resize drawer"
    onmousedown={onResizeStart}
  ></div>

  <!-- Drawer header -->
  <div class="flex items-center gap-2 px-4 py-2.5 border-b border-border shrink-0">
    <span class="text-sm font-semibold truncate flex-1">{name}</span>
    {#if namespace}
      <span class="text-xs text-muted border border-border rounded px-1.5 py-0.5 shrink-0">{namespace}</span>
    {/if}
    <button
      onclick={onclose}
      class="p-1 rounded hover:bg-surface-hover transition-colors shrink-0"
      aria-label="Close"
    >
      <X size={15} />
    </button>
  </div>

  <!-- Detail content -->
  <div class="flex-1 overflow-hidden">
    <ResourceDetail
      bind:obj
      {descriptor}
      {ctxName}
      {gvr}
      {namespace}
      {name}
      onrefresh={refresh}
    />
  </div>
</div>

