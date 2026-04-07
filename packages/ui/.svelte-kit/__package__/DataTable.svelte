<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    columns,
    items,
    row,
    sticky = false,
  }: {
    columns: { label: string; width?: string }[]
    items: any[]
    row: Snippet<[item: any]>
    sticky?: boolean
  } = $props()
</script>

{#if items.length > 0}
  <table class="w-full text-xs">
    <thead class="bg-surface {sticky ? 'sticky top-0 z-10' : ''}">
      <tr>
        {#each columns as col}
          <th
            class="text-left px-2 py-1.5 font-medium text-muted"
            style={col.width ? `width: ${col.width}` : ''}
          >{col.label}</th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#each items as item}
        <tr class="border-t border-border">
          {@render row(item)}
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
