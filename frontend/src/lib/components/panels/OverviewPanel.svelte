<script lang="ts">
  import { evalExpr } from '$lib/registry/index'
  import type { DescriptorDef } from '$lib/registry/index'
  import { formatAge } from '$lib/utils/age'
  import { slotRegistry } from '$lib/plugins/slots.svelte.js'
  import { loadPluginComponent } from '$lib/plugins/loader.js'
  import { streamingStore } from '$lib/stores/streaming.svelte.js'

  let {
    obj,
    descriptor,
    gvr = '',
  }: {
    obj: Record<string, any>
    descriptor: DescriptorDef
    gvr?: string
  } = $props()

  const basePluginURL = $derived(
    streamingStore.config
      ? `http://127.0.0.1:${streamingStore.config.port}/${streamingStore.config.token}/plugins`
      : null
  )

  function renderValue(expr: string, renderType: string): string {
    const raw = evalExpr(expr, obj)
    if (renderType === 'age' && raw) {
      return formatAge(String(raw))
    }
    if (raw === null || raw === undefined) return '—'
    return String(raw)
  }
</script>

<div class="p-4 grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 max-w-2xl">
  {#each descriptor.overviewFields as field}
    <span class="text-xs text-muted self-center">{field.label}</span>
    {#if field.renderType === 'badge'}
      <span class="text-xs font-mono bg-surface border border-border rounded px-2 py-0.5 w-fit">
        {renderValue(field.expr, field.renderType)}
      </span>
    {:else}
      <span class="text-xs font-mono">{renderValue(field.expr, field.renderType)}</span>
    {/if}
  {/each}
  {#if basePluginURL}
    {#each slotRegistry.getOverviewFields(gvr) as field (field.id)}
      {#await loadPluginComponent(field.pluginName, field.component, basePluginURL) then Cmp}
        {#if Cmp}
          <svelte:component this={Cmp} resource={obj} />
        {/if}
      {/await}
    {/each}
  {/if}
</div>
