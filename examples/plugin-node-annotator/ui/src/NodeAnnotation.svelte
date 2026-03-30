<script lang="ts">
  interface K8sContext {
    list(gvr: string, ns?: string): Promise<any[]>
  }
  interface PluginContext {
    cluster: { name: string; version: string }
    namespace: string
    k8s?: K8sContext
  }

  let { resource, ctx }: { resource: Record<string, any>; ctx: PluginContext } = $props()

  const status = $derived(resource?.status ?? {})
  const spec = $derived(resource?.spec ?? {})

  const taints = $derived<any[]>(spec.taints ?? [])
  const taintCount = $derived<number>(status.taintCount ?? taints.length)
  const readinessSummary = $derived<string>(status.readinessSummary ?? 'Unknown')
  const ready = $derived(readinessSummary === 'Ready')

  interface NodeRow {
    name: string
    taintCount: number
    readiness: string
  }

  let nodes = $state<NodeRow[]>([])
  let loading = $state(true)
  let errorMsg = $state<string | null>(null)

  $effect(() => {
    if (!ctx?.k8s) {
      loading = false
      return
    }
    ctx.k8s
      .list('core.v1.nodes')
      .then((items) => {
        nodes = items.map((n) => ({
          name: n.metadata?.name ?? '',
          taintCount: n.status?.taintCount ?? n.spec?.taints?.length ?? 0,
          readiness: n.status?.readinessSummary ?? 'Unknown',
        }))
        loading = false
      })
      .catch((e: unknown) => {
        errorMsg = e instanceof Error ? e.message : String(e)
        loading = false
      })
  })
</script>

<div class="p-4 overflow-auto h-full" style="font-family: inherit;">
  <!-- Current node summary -->
  <div class="mb-4 rounded border border-border bg-surface p-3">
    <h3 class="text-xs font-semibold mb-2">This Node</h3>
    <div class="flex flex-wrap gap-4 text-xs">
      <div>
        <span class="text-muted">Readiness</span>
        <div class="mt-0.5 font-medium" class:text-green-500={ready} class:text-red-500={!ready}>
          {readinessSummary}
        </div>
      </div>
      <div>
        <span class="text-muted">Taint Count</span>
        <div class="mt-0.5 font-medium">{taintCount}</div>
      </div>
    </div>

    {#if taints.length > 0}
      <div class="mt-3">
        <p class="text-xs text-muted mb-1">Taints:</p>
        <div class="flex flex-wrap gap-1">
          {#each taints as taint}
            <span class="text-xs bg-surface border border-border rounded px-1.5 py-0.5 font-mono">
              {taint.key}{taint.value ? '=' + taint.value : ''}:{taint.effect}
            </span>
          {/each}
        </div>
      </div>
    {/if}
  </div>

  <!-- All nodes table -->
  <h3 class="text-xs font-semibold mb-2">All Nodes ({nodes.length})</h3>

  {#if loading}
    <div class="flex items-center gap-2 text-sm text-muted">
      <div class="w-4 h-4 border-2 border-accent border-t-transparent rounded-full animate-spin"></div>
      Loading…
    </div>
  {:else if errorMsg}
    <div class="rounded border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
      {errorMsg}
    </div>
  {:else}
    <table class="w-full text-xs border-collapse">
      <thead>
        <tr class="border-b border-border text-muted">
          <th class="text-left py-1.5 px-2 font-medium">Name</th>
          <th class="text-left py-1.5 px-2 font-medium">Readiness</th>
          <th class="text-right py-1.5 px-2 font-medium">Taints</th>
        </tr>
      </thead>
      <tbody>
        {#each nodes as node (node.name)}
          {@const nodeReady = node.readiness === 'Ready'}
          <tr class="border-b border-border hover:bg-surface-hover transition-colors">
            <td class="py-1.5 px-2 font-mono">{node.name}</td>
            <td
              class="py-1.5 px-2 font-medium"
              class:text-green-500={nodeReady}
              class:text-red-500={!nodeReady && node.readiness !== 'Unknown'}
              class:text-muted={node.readiness === 'Unknown'}
            >
              {node.readiness}
            </td>
            <td class="py-1.5 px-2 text-right">{node.taintCount}</td>
          </tr>
        {/each}
        {#if nodes.length === 0}
          <tr>
            <td colspan="3" class="py-4 px-2 text-center text-muted">No nodes found</td>
          </tr>
        {/if}
      </tbody>
    </table>
  {/if}
</div>
