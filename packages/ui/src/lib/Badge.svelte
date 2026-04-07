<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    variant = 'default',
    children,
  }: {
    variant?: 'default' | 'success' | 'warning' | 'destructive'
    children?: Snippet
  } = $props()

  const variantClasses: Record<string, string> = {
    default: 'bg-surface text-muted',
    success: 'bg-accent text-accent-fg',
    warning: 'bg-surface-hover text-fg',
    destructive: 'bg-destructive text-accent-fg',
  }
</script>

<span class="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full {variantClasses[variant]}">
  {#if children}{@render children()}{/if}
</span>
