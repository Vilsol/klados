<script lang="ts">
  import { X } from 'lucide-svelte'
  import { sessionStore } from './stores/session.svelte'

  let dragFrom = $state<number | null>(null)
  let dragOver = $state<number | null>(null)

  function onDragStart(i: number) {
    dragFrom = i
  }

  function onDragOver(e: DragEvent, i: number) {
    e.preventDefault()
    dragOver = i
  }

  function onDrop(i: number) {
    if (dragFrom !== null && dragFrom !== i) {
      sessionStore.reorderTabs(dragFrom, i)
    }
    dragFrom = null
    dragOver = null
  }

  function onDragEnd() {
    dragFrom = null
    dragOver = null
  }
</script>

{#if sessionStore.tabs.length > 0}
  <div class="flex items-center border-b border-border bg-surface overflow-x-auto shrink-0" role="tablist">
    {#each sessionStore.tabs as tab, i}
      <div
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm border-r border-border whitespace-nowrap transition-colors cursor-pointer select-none
          {i === sessionStore.activeTabIndex ? 'bg-bg font-medium' : 'hover:bg-surface-hover text-muted'}
          {dragOver === i && dragFrom !== i ? 'border-l-2 border-l-accent' : ''}"
        role="tab"
        tabindex="0"
        aria-selected={i === sessionStore.activeTabIndex}
        draggable="true"
        onclick={() => sessionStore.setActiveTab(i)}
        onkeydown={(e) => { if (e.key === 'Enter') sessionStore.setActiveTab(i) }}
        ondragstart={() => onDragStart(i)}
        ondragover={(e) => onDragOver(e, i)}
        ondrop={() => onDrop(i)}
        ondragend={onDragEnd}
      >
        <span class="truncate max-w-40">
          {tab.name || tab.gvr || 'Untitled'}
        </span>
        <button
          onclick={(e) => { e.stopPropagation(); sessionStore.closeTab(i) }}
          class="p-0.5 rounded hover:bg-border transition-colors"
          aria-label="Close tab"
        >
          <X size={12} />
        </button>
      </div>
    {/each}
  </div>
{/if}
