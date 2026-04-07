<script lang="ts">
  let { obj }: { obj: Record<string, any> } = $props()

  const params = $derived<Record<string, string>>(obj.spec?.parameters ?? obj.parameters ?? {})
  const entries = $derived(Object.entries(params))
</script>

{#if entries.length === 0}
  <p class="text-sm text-muted p-4">No parameters</p>
{:else}
  <div class="divide-y divide-border">
    {#each entries as [k, v]}
      <div class="flex px-4 py-2 text-sm">
        <span class="w-1/2 text-muted font-mono break-all">{k}</span>
        <span class="w-1/2 font-mono break-all">{v}</span>
      </div>
    {/each}
  </div>
{/if}
