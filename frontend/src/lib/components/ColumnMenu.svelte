<script lang="ts">
  import {ArrowUp, ArrowDown} from "lucide-svelte";

  let {
    visibleColumns,
    allColumns,
    compact,
    onToggle,
    onMove,
    onReset,
    onCompactChange,
    sparklineGvrs = [],
    sparklineColumns = [],
    onSparklineToggle,
    gvr,
  }: {
    visibleColumns: {name: string}[];
    allColumns: {col: {name: string}; visible: boolean}[];
    compact: boolean;
    onToggle: (name: string, visible: boolean) => void;
    onMove: (name: string, direction: "up" | "down") => void;
    onReset: () => void;
    onCompactChange: (value: boolean) => void;
    sparklineGvrs?: string[];
    sparklineColumns?: string[];
    onSparklineToggle?: (columns: string[]) => void;
    gvr?: string;
  } = $props();

  const visibleEntries = $derived(visibleColumns.map((col) => ({col, visible: true})));
  const hiddenEntries = $derived(allColumns.filter((e) => !e.visible));
  const hasSparklines = $derived(gvr ? sparklineGvrs.includes(gvr) : false);

  const availableSparklineCols = ["CPU", "Memory"];

  function toggleSparklineCol(col: string) {
    const next = sparklineColumns.includes(col) ? sparklineColumns.filter((c) => c !== col) : [...sparklineColumns, col];
    onSparklineToggle?.(next);
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
    <button type="button" onclick={() => onReset()} class="text-xs text-muted hover:text-fg transition-colors">Reset</button>
  </div>

  {#each visibleEntries as entry, i (entry.col.name)}
    <div class="flex items-center gap-1 px-2 py-1 hover:bg-surface-hover">
      <input
        type="checkbox"
        checked={true}
        disabled={entry.col.name === 'Name'}
        onchange={(e) => onToggle(entry.col.name, e.currentTarget.checked)}
        class="rounded border-border shrink-0"
      >
      <span class="flex-1 text-sm truncate">{entry.col.name}</span>
      <div class="flex gap-0.5 shrink-0">
        <button
          type="button"
          onclick={() => onMove(entry.col.name, 'up')}
          disabled={i <= 1}
          class="p-0.5 rounded text-muted hover:text-fg disabled:opacity-30 disabled:cursor-not-allowed"
          aria-label={`Move ${entry.col.name} up`}
        >
          <ArrowUp size={12} />
        </button>
        <button
          type="button"
          onclick={() => onMove(entry.col.name, 'down')}
          disabled={entry.col.name === 'Name' || i === visibleEntries.length - 1}
          class="p-0.5 rounded text-muted hover:text-fg disabled:opacity-30 disabled:cursor-not-allowed"
          aria-label={`Move ${entry.col.name} down`}
        >
          <ArrowDown size={12} />
        </button>
      </div>
    </div>
  {/each}

  {#each hiddenEntries as entry (entry.col.name)}
    <div class="flex items-center gap-1 px-2 py-1 hover:bg-surface-hover">
      <input
        type="checkbox"
        checked={false}
        onchange={(e) => onToggle(entry.col.name, e.currentTarget.checked)}
        class="rounded border-border shrink-0"
      >
      <span class="flex-1 text-sm truncate text-muted">{entry.col.name}</span>
    </div>
  {/each}

  <div class="border-t border-border mt-1 pt-1.5 px-2">
    <label class="flex items-center gap-2 py-1 hover:bg-surface-hover cursor-pointer rounded px-0.5">
      <input
        type="checkbox"
        checked={compact}
        onchange={(e) => onCompactChange(e.currentTarget.checked)}
        class="rounded border-border"
      >
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
          >
          <span class="text-sm">{col}</span>
        </label>
      {/each}
    </div>
  {/if}
</div>
