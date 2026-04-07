<script lang="ts">
  let {
    entries,
    sorted = true,
  }: {
    entries: [string, unknown][]
    sorted?: boolean
  } = $props()

  const display = $derived(
    sorted ? [...entries].sort(([a], [b]) => a.localeCompare(b)) : entries,
  )
</script>

{#if display.length === 0}
  <p class="text-xs text-muted">None</p>
{:else}
  <div class="flex flex-wrap gap-1.5">
    {#each display as [k, v]}
      <span class="text-xs font-mono bg-bg border border-border rounded px-2 py-0.5">
        <span class="text-accent">{k}</span><span class="text-muted">=</span>{v}
      </span>
    {/each}
  </div>
{/if}
