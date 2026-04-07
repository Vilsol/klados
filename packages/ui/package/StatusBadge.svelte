<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    status,
    mode = 'text',
    children,
  }: {
    status: 'True' | 'False' | 'Unknown' | 'Warning' | 'Normal' | boolean
    mode?: 'text' | 'pill'
    children?: Snippet
  } = $props()

  const isPositive = $derived(
    status === 'True' || status === 'Normal' || status === true,
  )
  const isNegative = $derived(
    status === 'False' || status === 'Warning' || status === false,
  )

  const textClass = $derived(
    isPositive
      ? 'text-green-600 dark:text-green-400'
      : isNegative
        ? 'text-red-600 dark:text-red-400'
        : 'text-muted',
  )

  const pillClass = $derived(
    isPositive
      ? 'bg-green-500/15 text-green-600 dark:text-green-400'
      : isNegative
        ? 'bg-red-500/15 text-red-600 dark:text-red-400'
        : 'bg-surface text-muted border border-border',
  )
</script>

{#if mode === 'pill'}
  <span class="text-xs px-2 py-0.5 rounded-full font-medium {pillClass}">
    {#if children}{@render children()}{:else}{status}{/if}
  </span>
{:else}
  <span class="{textClass}">
    {#if children}{@render children()}{:else}{status}{/if}
  </span>
{/if}
