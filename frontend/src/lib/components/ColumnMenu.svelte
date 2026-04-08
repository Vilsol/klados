<script lang="ts">
  import { ArrowUp, ArrowDown } from 'lucide-svelte'
  import { columnStore } from '$lib/stores/columns.svelte'

  let {
    gvr,
    sparklineGvrs = [],
    sparklineColumns = [],
    onSparklineToggle,
  }: {
    gvr: string
    sparklineGvrs?: string[]
    sparklineColumns?: string[]
    onSparklineToggle?: (columns: string[]) => void
  } = $props()

  const visibleEntries = $derived(
    columnStore.visibleColumns.map((col) => ({ col, visible: true }))
  )
  const hiddenEntries = $derived(columnStore.allColumns.filter((e) => !e.visible))
  const hasSparklines = $derived(sparklineGvrs.includes(gvr))

  const availableSparklineCols = ['CPU', 'Memory']

  function toggleSparklineCol(col: string) {
    const next = sparklineColumns.includes(col)
      ? sparklineColumns.filter((c) => c !== col)
      : [...sparklineColumns, col]
    onSparklineToggle?.(next)
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="absolute right-0 top-full mt-1 z-50 bg-surface border border-border rounded shadow-lg py-2 min-w-56"
  onclick={(e) => e.stopPropagation()}
  onkeydown={(e) => e.stopPropagation()}
>
  <div class="flex items-center justify-between px-3 pb-1.5 mb-1 border-b border-border">
    <span class="text-xs font-semibold uppercase tracking-wider text-muted">Columns</span>
    <button
      onclick={() => columnStore.reset()}
      class="text-xs text-muted hover:text-fg transition-colors"
    >Reset</button>
  </div>

  {#each visibleEntries as entry, i}
    <div class="flex items-center gap-1 px-2 py-1 hover:bg-surface-hover">
      <input
        type="checkbox"
        checked={true}
        disabled={entry.col.name === 'Name'}
        onchange={(e) => columnStore.setColumnVisible(entry.col.name, e.currentTarget.checked)}
        class="rounded border-border shrink-0"
      />
      <span class="flex-1 text-sm truncate">{entry.col.name}</span>
      <div class="flex gap-0.5 shrink-0">
        <button
          onclick={() => columnStore.moveColumn(entry.col.name, 'up')}
          disabled={i <= 1}
          class="p-0.5 rounded text-muted hover:text-fg disabled:opacity-30 disabled:cursor-not-allowed"
          aria-label={`Move ${entry.col.name} up`}
        >
          <ArrowUp size={12} />
        </button>
        <button
          onclick={() => columnStore.moveColumn(entry.col.name, 'down')}
          disabled={entry.col.name === 'Name' || i === visibleEntries.length - 1}
          class="p-0.5 rounded text-muted hover:text-fg disabled:opacity-30 disabled:cursor-not-allowed"
          aria-label={`Move ${entry.col.name} down`}
        >
          <ArrowDown size={12} />
        </button>
      </div>
    </div>
  {/each}

  {#each hiddenEntries as entry}
    <div class="flex items-center gap-1 px-2 py-1 hover:bg-surface-hover">
      <input
        type="checkbox"
        checked={false}
        onchange={(e) => columnStore.setColumnVisible(entry.col.name, e.currentTarget.checked)}
        class="rounded border-border shrink-0"
      />
      <span class="flex-1 text-sm truncate text-muted">{entry.col.name}</span>
    </div>
  {/each}

  <div class="border-t border-border mt-1 pt-1.5 px-2">
    <label class="flex items-center gap-2 py-1 hover:bg-surface-hover cursor-pointer rounded px-0.5">
      <input
        type="checkbox"
        checked={columnStore.compact}
        onchange={(e) => columnStore.setCompact(e.currentTarget.checked)}
        class="rounded border-border"
      />
      <span class="text-sm">Compact rows</span>
    </label>
  </div>

  {#if hasSparklines}
    <div class="border-t border-border mt-1 pt-1.5">
      <div class="px-3 pb-1 text-xs font-semibold uppercase tracking-wider text-muted">Sparklines</div>
      {#each availableSparklineCols as col}
        <label class="flex items-center gap-2 px-2 py-1 hover:bg-surface-hover cursor-pointer">
          <input
            type="checkbox"
            checked={sparklineColumns.includes(col)}
            onchange={() => toggleSparklineCol(col)}
            class="rounded border-border"
          />
          <span class="text-sm">{col}</span>
        </label>
      {/each}
    </div>
  {/if}
</div>
