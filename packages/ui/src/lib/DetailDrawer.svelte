<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import type { Snippet } from 'svelte'
  import { X } from 'lucide-svelte'
  import CopyableValue from './CopyableValue.svelte'

  let {
    item,
    ctxName,
    gvr,
    onclose,
    onFetchResource,
    children,
  }: {
    item: Record<string, any>
    ctxName: string
    gvr: string
    onclose: () => void
    onFetchResource?: (ctx: string, gvr: string, ns: string, name: string) => Promise<Record<string, any> | null>
    children: Snippet<[{ obj: Record<string, any>; onrefresh: () => void; onupdate: (updated: Record<string, any>) => void }]>
  } = $props()

  const name = $derived<string>(item.metadata?.name ?? '')
  const namespace = $derived<string>(item.metadata?.namespace ?? '')

  // svelte-ignore state_referenced_locally
  let obj = $state<Record<string, any>>(item)
  $effect(() => { obj = item })

  async function refresh() {
    try {
      const fresh = await onFetchResource?.(ctxName, gvr, namespace, name)
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
  class="absolute top-0 right-0 bottom-0 flex flex-col bg-bg border-l border-border shadow-2xl"
  style="width: {drawerWidth}px"
  role="dialog"
  aria-label="Resource detail: {name}"
>
  <!-- Resize handle -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <div
    class="absolute top-0 left-0 bottom-0 w-2 z-10 cursor-ew-resize transition-colors group
      {dragging ? 'bg-accent/40' : 'hover:bg-accent/30'}"
    role="separator"
    aria-label="Resize drawer"
    onmousedown={onResizeStart}
  >
    <div class="absolute inset-y-0 left-0.5 w-px bg-border group-hover:bg-accent/60 {dragging ? 'bg-accent/60' : ''}"></div>
  </div>

  <!-- Drawer header -->
  <div class="flex items-center gap-2 px-4 py-2.5 border-b border-border shrink-0">
    <CopyableValue value={name} class="text-sm font-semibold truncate flex-1" />
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
    {@render children({ obj, onrefresh: refresh, onupdate: (updated) => { obj = updated } })}
  </div>
</div>
