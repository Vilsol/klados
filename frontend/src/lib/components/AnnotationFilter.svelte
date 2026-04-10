<script lang="ts">
  import { X, Plus } from 'lucide-svelte'

  let { filters = $bindable<{ key: string; value: string }[]>([]) }: {
    filters: { key: string; value: string }[]
  } = $props()

  let popoverOpen = $state(false)
  let keyInput = $state('')
  let valueInput = $state('')

  $effect(() => {
    if (!popoverOpen) return
    const close = () => { popoverOpen = false }
    const timer = setTimeout(() => window.addEventListener('click', close, { once: true }), 0)
    return () => { clearTimeout(timer); window.removeEventListener('click', close) }
  })

  function addFilter() {
    const k = keyInput.trim()
    if (!k) return
    filters = [...filters, { key: k, value: valueInput.trim() }]
    keyInput = ''
    valueInput = ''
    popoverOpen = false
  }

  function removeFilter(index: number) {
    filters = filters.filter((_, i) => i !== index)
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') addFilter()
  }
</script>

<div class="flex items-center gap-1 flex-wrap">
  {#each filters as f, i}
    <span class="inline-flex items-center gap-1 text-xs px-1.5 py-0.5 rounded border border-purple-500/30 bg-purple-500/10 text-purple-300">
      {f.key}{f.value ? `=${f.value}` : ''}
      <button onclick={() => removeFilter(i)} class="hover:text-purple-100 transition-colors" aria-label="Remove filter">
        <X size={10} />
      </button>
    </span>
  {/each}
  <div class="relative">
    <button
      onclick={(e) => { e.stopPropagation(); popoverOpen = !popoverOpen }}
      class="inline-flex items-center gap-0.5 text-xs text-muted hover:text-fg transition-colors"
      title="Add annotation filter"
    >
      <Plus size={12} />
      Annotation
    </button>
    {#if popoverOpen}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="absolute top-full mt-1 left-0 z-50 bg-surface border border-border rounded shadow-lg p-2 flex flex-col gap-1.5 min-w-48"
        onclick={(e) => e.stopPropagation()}
        onkeydown={(e) => e.stopPropagation()}
      >
        <input
          type="text"
          placeholder="Key"
          bind:value={keyInput}
          onkeydown={handleKeydown}
          class="text-xs bg-transparent border border-border rounded px-2 py-1 outline-none placeholder-muted focus:border-accent"
        />
        <input
          type="text"
          placeholder="Value (empty = any)"
          bind:value={valueInput}
          onkeydown={handleKeydown}
          class="text-xs bg-transparent border border-border rounded px-2 py-1 outline-none placeholder-muted focus:border-accent"
        />
        <button
          onclick={addFilter}
          class="text-xs px-2 py-1 rounded bg-accent/20 text-accent hover:bg-accent/30 transition-colors"
        >
          Add
        </button>
      </div>
    {/if}
  </div>
</div>
