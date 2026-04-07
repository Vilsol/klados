<script lang="ts">
  let {
    pairs = $bindable(),
    addLabel = '+ Add',
    keyPlaceholder = 'key',
    valuePlaceholder = 'value',
  }: {
    pairs: [string, string][]
    addLabel?: string
    keyPlaceholder?: string
    valuePlaceholder?: string
  } = $props()
</script>

<div class="flex flex-col gap-1.5">
  {#each pairs as pair, i}
    <div class="flex gap-2 items-center">
      <input
        bind:value={pairs[i][0]}
        placeholder={keyPlaceholder}
        class="text-xs font-mono flex-1 bg-bg border border-border rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-accent"
      />
      <span class="text-muted text-xs">=</span>
      <input
        bind:value={pairs[i][1]}
        placeholder={valuePlaceholder}
        class="text-xs font-mono flex-1 bg-bg border border-border rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-accent"
      />
      <button
        onclick={() => pairs.splice(i, 1)}
        class="text-xs text-muted hover:text-destructive transition-colors"
      >✕</button>
    </div>
  {/each}
  <button
    onclick={() => pairs.push(['', ''])}
    class="text-xs text-accent hover:underline self-start"
  >{addLabel}</button>
</div>
