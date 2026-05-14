<script lang="ts">
  let {
    compact,
    onCompactChange,
    hasSparklines = false,
    sparklineColumns = [],
    onSparklineToggle,
  }: {
    compact: boolean;
    onCompactChange: (value: boolean) => void;
    hasSparklines?: boolean;
    sparklineColumns?: string[];
    onSparklineToggle?: (columns: string[]) => void;
  } = $props();

  const availableSparklineCols = ["CPU", "Memory"];

  function toggleSparkline(col: string) {
    const next = sparklineColumns.includes(col)
      ? sparklineColumns.filter((c) => c !== col)
      : [...sparklineColumns, col];
    onSparklineToggle?.(next);
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="absolute right-0 top-full mt-1 z-50 bg-surface border border-border rounded shadow-lg py-2 min-w-44"
  onclick={(e) => e.stopPropagation()}
  onkeydown={(e) => e.stopPropagation()}
>
  <div class="px-3 pb-1 text-xs font-semibold uppercase tracking-wider text-muted">View</div>
  <label class="flex items-center gap-2 px-3 py-1 hover:bg-surface-hover cursor-pointer">
    <input
      type="checkbox"
      checked={compact}
      aria-label="Compact rows"
      onchange={(e) => onCompactChange(e.currentTarget.checked)}
      class="rounded border-border"
    />
    <span class="text-sm">Compact rows</span>
  </label>
  {#if hasSparklines}
    <div class="border-t border-border mt-1 pt-1.5">
      <div class="px-3 pb-1 text-xs font-semibold uppercase tracking-wider text-muted">Sparklines</div>
      {#each availableSparklineCols as col}
        <label class="flex items-center gap-2 px-3 py-1 hover:bg-surface-hover cursor-pointer">
          <input
            type="checkbox"
            checked={sparklineColumns.includes(col)}
            aria-label={col}
            onchange={() => toggleSparkline(col)}
            class="rounded border-border"
          />
          <span class="text-sm">{col}</span>
        </label>
      {/each}
    </div>
  {/if}
</div>
