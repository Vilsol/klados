<script lang="ts">
  import { Combobox } from 'bits-ui'
  import { ChevronDown, Check, X } from 'lucide-svelte'

  type BaseOption = { value: string; label: string }

  type SingleProps = {
    type?: 'single'
    options: BaseOption[]
    value: string
    placeholder?: string
    searchPlaceholder?: string
    emptyMessage?: string
    size?: 'xs' | 'sm'
    disabled?: boolean
  }

  type MultipleProps = {
    type: 'multiple'
    options: BaseOption[]
    value: string[]
    placeholder?: string
    searchPlaceholder?: string
    emptyMessage?: string
    size?: 'xs' | 'sm'
    disabled?: boolean
  }

  type Props = SingleProps | MultipleProps

  let {
    type = 'single',
    options,
    value = $bindable(),
    placeholder = 'Select…',
    searchPlaceholder = 'Search…',
    emptyMessage = 'No results found.',
    size = 'sm',
    disabled = false,
  }: Props = $props()

  let searchValue = $state('')
  let open = $state(false)

  const filtered = $derived(
    searchValue === ''
      ? options
      : options.filter((o) =>
          o.label.toLowerCase().includes(searchValue.toLowerCase()),
        ),
  )

  const displayLabel = $derived.by(() => {
    if (type === 'multiple') {
      const vals = value as string[]
      if (vals.length === 0) return placeholder
      if (vals.length === 1) {
        const match = options.find((o) => o.value === vals[0])
        return match?.label ?? vals[0]
      }
      return `${vals.length} selected`
    }
    const v = value as string
    if (!v) return placeholder
    const match = options.find((o) => o.value === v)
    return match?.label ?? v
  })

  const textSize = $derived(size === 'xs' ? 'text-xs' : 'text-sm')
  const iconSize = $derived(size === 'xs' ? 12 : 14)
  const itemPy = $derived(size === 'xs' ? 'py-1' : 'py-1.5')

  function removeTag(val: string) {
    if (type === 'multiple') {
      value = (value as string[]).filter((v) => v !== val) as any
    }
  }
</script>

<Combobox.Root
  {type}
  bind:value={value as any}
  bind:open
  onOpenChangeComplete={(o) => { if (!o) searchValue = '' }}
  {disabled}
>
  <div class="relative">
    <Combobox.Input
      oninput={(e) => (searchValue = e.currentTarget.value)}
      onclick={() => { if (!open) open = true }}
      class="flex items-center gap-1 w-full bg-bg text-fg border border-border rounded
        px-2 {itemPy} {textSize} placeholder:text-muted
        hover:bg-surface-hover focus:outline-none focus:ring-1 focus:ring-accent
        transition-colors disabled:opacity-50 disabled:cursor-not-allowed pr-7"
      placeholder={open ? searchPlaceholder : displayLabel}
      aria-label={placeholder}
      {disabled}
    />
    {#if type === 'multiple' && (value as string[]).length > 0 && !open}
      <button
        type="button"
        class="absolute end-7 top-1/2 -translate-y-1/2 text-muted hover:text-fg transition-colors"
        onclick={(e) => { e.stopPropagation(); value = [] as any }}
        aria-label="Clear all"
      >
        <X size={iconSize} />
      </button>
    {/if}
    <Combobox.Trigger
      class="absolute end-1.5 top-1/2 -translate-y-1/2 text-muted"
      {disabled}
    >
      <ChevronDown size={iconSize} />
    </Combobox.Trigger>
  </div>

  <Combobox.Portal>
    <Combobox.Content
      class="border border-border bg-bg rounded shadow-lg z-50 py-1
        max-h-[min(20rem,var(--bits-combobox-content-available-height))]
        w-[var(--bits-combobox-anchor-width)] min-w-[var(--bits-combobox-anchor-width)]
        select-none overflow-hidden"
      sideOffset={4}
    >
      <Combobox.Viewport class="p-0.5">
        {#each filtered as opt (opt.value)}
          <Combobox.Item
            value={opt.value}
            label={opt.label}
            class="flex items-center gap-2 w-full rounded px-2 {itemPy} {textSize}
              text-fg cursor-pointer outline-none
              data-[highlighted]:bg-surface-hover transition-colors"
          >
            {#snippet children({ selected })}
              {#if type === 'multiple'}
                <span
                  class="shrink-0 w-3.5 h-3.5 rounded border flex items-center justify-center transition-colors
                    {selected ? 'bg-accent border-accent text-bg' : 'border-border'}"
                >
                  {#if selected}<Check size={10} />{/if}
                </span>
              {/if}
              <span class="flex-1 truncate">{opt.label}</span>
              {#if type === 'single' && selected}
                <Check size={iconSize} class="shrink-0 text-accent" />
              {/if}
            {/snippet}
          </Combobox.Item>
        {:else}
          <span class="block px-3 py-2 {textSize} text-muted">{emptyMessage}</span>
        {/each}
      </Combobox.Viewport>
    </Combobox.Content>
  </Combobox.Portal>
</Combobox.Root>
