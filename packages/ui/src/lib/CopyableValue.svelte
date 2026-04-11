<script lang="ts">
  import { Copy, Check } from 'lucide-svelte'

  import type { Snippet } from 'svelte'

  let {
    value,
    rawValue,
    children,
    class: className = '',
  }: {
    value: string
    rawValue?: string
    children?: Snippet
    class?: string
  } = $props()

  let copied = $state(false)
  let flashing = $state(false)

  async function copy() {
    await navigator.clipboard.writeText(rawValue ?? value)
    copied = true
    flashing = true
    setTimeout(() => flashing = false, 300)
    setTimeout(() => copied = false, 1500)
  }
</script>

<button
  onclick={copy}
  title={rawValue ?? value}
  class="group relative inline-flex items-center gap-1 cursor-pointer max-w-full
    hover:underline hover:decoration-dotted hover:decoration-muted
    transition-colors {flashing ? 'bg-accent/10' : ''} rounded px-0.5 -mx-0.5 {className}"
>
  <span class="truncate">{#if children}{@render children()}{:else}{value}{/if}</span>
  <span class="shrink-0 opacity-0 group-hover:opacity-60 transition-opacity">
    {#if copied}
      <Check size={12} />
    {:else}
      <Copy size={12} />
    {/if}
  </span>
</button>
