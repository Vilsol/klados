<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    children,
    class: className = 'mb-2',
  }: {
    children: Snippet
    class?: string
  } = $props()
</script>

<h3 class="text-xs font-semibold text-muted uppercase tracking-wide {className}">
  {#if children}{@render children()}{/if}
</h3>
