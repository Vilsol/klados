<script lang="ts">
  import type {Suggestion} from "$lib/search/autocomplete";

  let {
    suggestions = [],
    visible = false,
    selectedIndex = 0,
    onselect,
  }: {
    suggestions: Suggestion[];
    visible: boolean;
    selectedIndex: number;
    onselect?: (suggestion: Suggestion) => void;
  } = $props();
</script>

{#if visible && suggestions.length > 0}
  <div
    class="absolute left-0 top-full mt-1 z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-48 max-h-64 overflow-y-auto"
    role="listbox"
  >
    {#each suggestions as suggestion, i}
      <button
        class="w-full flex items-center justify-between gap-4 px-3 py-1.5 text-sm text-left hover:bg-surface-hover {i === selectedIndex ? 'bg-surface-hover' : ''}"
        role="option"
        aria-selected={i === selectedIndex}
        onmousedown={(e) => { e.preventDefault(); onselect?.(suggestion) }}
      >
        <span class="text-fg">{suggestion.value}</span>
        <span class="flex items-center gap-2">
          {#if suggestion.description}
            <span class="text-muted text-xs">{suggestion.description}</span>
          {/if}
          {#if suggestion.count !== undefined}
            <span class="text-muted text-xs tabular-nums">{suggestion.count}</span>
          {/if}
        </span>
      </button>
    {/each}
  </div>
{/if}
