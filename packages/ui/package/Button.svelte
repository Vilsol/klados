<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    variant = 'primary',
    disabled = false,
    type = 'button',
    onclick,
    children,
  }: {
    variant?: 'primary' | 'destructive' | 'ghost' | 'outline'
    disabled?: boolean
    type?: 'button' | 'submit' | 'reset'
    onclick?: () => void
    children?: Snippet
  } = $props()

  const variantClasses: Record<string, string> = {
    primary: 'bg-accent text-accent-fg hover:opacity-90',
    destructive: 'bg-destructive text-accent-fg hover:opacity-90',
    ghost: 'text-fg hover:bg-surface-hover',
    outline: 'border border-border text-fg hover:bg-surface-hover',
  }
</script>

<button
  {type}
  {disabled}
  {onclick}
  class="inline-flex items-center justify-center gap-1.5 px-3 py-1.5 text-sm rounded transition-all disabled:opacity-50 disabled:pointer-events-none {variantClasses[variant]}"
>
  {#if children}{@render children()}{/if}
</button>
