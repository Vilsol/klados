<script lang="ts">
  import { ChevronDown } from 'lucide-svelte'

  let {
    options,
    value = $bindable(),
    size = 'sm',
  }: {
    options: { value: string; label: string }[]
    value: string
    size?: 'xs' | 'sm'
  } = $props()

  let open = $state(false)
  const selected = $derived(options.find((o) => o.value === value))

  function pick(v: string) {
    value = v
    open = false
  }

  function onOutsideClick(e: MouseEvent) {
    if (!(e.target as HTMLElement).closest('[data-select]')) open = false
  }
</script>

<svelte:window onclick={onOutsideClick} />

<div class="relative" data-select>
  <button
    type="button"
    onclick={() => (open = !open)}
    class="flex items-center gap-1 w-full bg-bg text-fg border border-border rounded px-2 py-1 hover:bg-surface-hover transition-colors
      {size === 'xs' ? 'text-xs' : 'text-sm'}"
  >
    <span class="flex-1 text-left truncate">{selected?.label ?? ''}</span>
    <ChevronDown size={size === 'xs' ? 12 : 14} class="shrink-0 text-muted" />
  </button>

  {#if open}
    <div class="absolute top-full left-0 mt-1 w-full min-w-max rounded border border-border bg-bg shadow-lg z-50">
      {#each options as opt}
        <button
          type="button"
          onclick={() => pick(opt.value)}
          class="w-full text-left px-3 py-1.5 hover:bg-surface-hover transition-colors
            {size === 'xs' ? 'text-xs' : 'text-sm'}
            {opt.value === value ? 'font-medium text-fg' : 'text-muted'}"
        >
          {opt.label}
        </button>
      {/each}
    </div>
  {/if}
</div>
