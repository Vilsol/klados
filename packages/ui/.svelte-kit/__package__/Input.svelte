<script lang="ts">
  let {
    label,
    error,
    disabled = false,
    value = $bindable(''),
    placeholder = '',
    type = 'text',
    id,
  }: {
    label?: string
    error?: string
    disabled?: boolean
    value?: string
    placeholder?: string
    type?: string
    id?: string
  } = $props()

  const inputId = $derived(id ?? (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined))
</script>

<div class="flex flex-col gap-1">
  {#if label}
    <label for={inputId} class="text-sm font-medium text-fg">{label}</label>
  {/if}
  <input
    id={inputId}
    {type}
    {disabled}
    {placeholder}
    bind:value
    class="px-3 py-1.5 text-sm bg-bg text-fg border rounded transition-colors
      {error ? 'border-destructive' : 'border-border'}
      hover:border-accent focus:outline-none focus:border-accent
      disabled:opacity-50 disabled:pointer-events-none"
  />
  {#if error}
    <span class="text-xs text-destructive">{error}</span>
  {/if}
</div>
