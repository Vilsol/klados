<script lang="ts">
  import {Pin} from "lucide-svelte";

  let {
    allColumns,
    visibleColumns,
    pinnedNames = [],
    onToggle,
    onReset,
  }: {
    allColumns: {col: {name: string}; visible: boolean}[];
    visibleColumns: {name: string}[];
    pinnedNames?: string[];
    onToggle: (name: string, visible: boolean) => void;
    onReset: () => void;
  } = $props();

  let filter = $state("");

  const pinnedSet = $derived(new Set(pinnedNames));
  const visibleOrder = $derived(visibleColumns.map((c) => c.name));

  const ordered = $derived.by(() => {
    const byName = new Map(allColumns.map((e) => [e.col.name, e]));
    const visibleEntries = visibleOrder
      .map((n) => byName.get(n))
      .filter((e): e is {col: {name: string}; visible: boolean} => e !== undefined);
    const hiddenEntries = allColumns.filter((e) => !e.visible);
    return [...visibleEntries, ...hiddenEntries];
  });

  const filtered = $derived.by(() => {
    if (!filter) return ordered;
    const q = filter.toLowerCase();
    return ordered.filter((e) => e.col.name.toLowerCase().includes(q));
  });
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="absolute right-0 top-full mt-1 z-50 bg-surface border border-border rounded shadow-lg py-2 min-w-64"
  onclick={(e) => e.stopPropagation()}
  onkeydown={(e) => e.stopPropagation()}
>
  <div class="flex items-center justify-between px-3 pb-1.5 mb-1 border-b border-border">
    <span class="text-xs font-semibold uppercase tracking-wider text-muted">Columns</span>
    <button type="button" onclick={() => onReset()} class="text-xs text-muted hover:text-fg transition-colors">Reset</button>
  </div>
  <div class="px-2 pb-1.5">
    <input
      type="text"
      bind:value={filter}
      placeholder="Filter…"
      class="w-full px-2 py-1 text-sm bg-bg border border-border rounded focus:outline-none focus:border-accent"
    />
  </div>
  <div class="max-h-72 overflow-y-auto">
    {#each filtered as entry (entry.col.name)}
      {@const isPinned = pinnedSet.has(entry.col.name)}
      <label class="flex items-center gap-2 px-3 py-1 hover:bg-surface-hover cursor-pointer">
        <input
          type="checkbox"
          checked={entry.visible}
          disabled={isPinned}
          onchange={(e) => onToggle(entry.col.name, e.currentTarget.checked)}
          class="rounded border-border shrink-0"
        />
        <span class="flex-1 text-sm truncate {entry.visible ? '' : 'text-muted'}">
          {entry.col.name}
        </span>
        {#if isPinned}
          <Pin size={11} class="text-muted shrink-0" />
        {/if}
      </label>
    {/each}
    {#if filtered.length === 0}
      <div class="px-3 py-2 text-xs text-muted">No matches</div>
    {/if}
  </div>
</div>
