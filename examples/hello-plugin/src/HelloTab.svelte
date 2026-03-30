<script lang="ts">
  interface K8SContext {
    list(gvr: string, ns?: string): Promise<any[]>
    get(gvr: string, ns: string, name: string): Promise<any>
  }

  interface PluginContext {
    cluster: { name: string; version: string }
    namespace: string
    k8s?: K8SContext
  }

  let { resource, ctx }: { resource: Record<string, any>; ctx: PluginContext } = $props()

  interface DeploymentRow {
    name: string
    namespace: string
    replicas: number
    ready: number
    available: number
  }

  let rows = $state<DeploymentRow[]>([])
  let loading = $state(true)
  let errorMsg = $state<string | null>(null)

  $effect(() => {
    if (!ctx?.k8s) {
      errorMsg = 'k8s context not available — check plugin permissions'
      loading = false
      return
    }
    const ns = resource?.metadata?.namespace ?? ''
    ctx.k8s
      .list('apps.v1.deployments', ns)
      .then((items) => {
        rows = items.map((d) => ({
          name: d.metadata?.name ?? '',
          namespace: d.metadata?.namespace ?? '',
          replicas: d.spec?.replicas ?? 0,
          ready: d.status?.readyReplicas ?? 0,
          available: d.status?.availableReplicas ?? 0,
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
  <div class="mb-4">
    <p class="text-xs text-muted mb-1">
      <span class="font-medium">Cluster:</span> {ctx.cluster.name}
      &nbsp;·&nbsp;
      <span class="font-medium">Namespace:</span> {resource?.metadata?.namespace ?? '—'}
    </p>
    <p class="text-xs text-muted">
      Listing all Deployments in namespace via <code>ctx.k8s.list('apps.v1.deployments')</code>
    </p>
  </div>

  {#if loading}
    <div class="flex items-center gap-2 text-sm text-muted">
      <div class="w-4 h-4 border-2 border-accent border-t-transparent rounded-full animate-spin"></div>
      Loading deployments…
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
          <th class="text-left py-1.5 px-2 font-medium">Namespace</th>
          <th class="text-right py-1.5 px-2 font-medium">Replicas</th>
          <th class="text-right py-1.5 px-2 font-medium">Ready</th>
          <th class="text-right py-1.5 px-2 font-medium">Available</th>
        </tr>
      </thead>
      <tbody>
        {#each rows as row (row.name + '/' + row.namespace)}
          <tr class="border-b border-border hover:bg-surface-hover transition-colors">
            <td class="py-1.5 px-2 font-mono">{row.name}</td>
            <td class="py-1.5 px-2 text-muted">{row.namespace}</td>
            <td class="py-1.5 px-2 text-right">{row.replicas}</td>
            <td
              class="py-1.5 px-2 text-right"
              class:text-destructive={row.ready < row.replicas}
              class:text-accent={row.ready >= row.replicas && row.replicas > 0}
            >{row.ready}</td>
            <td class="py-1.5 px-2 text-right">{row.available}</td>
          </tr>
        {/each}
        {#if rows.length === 0}
          <tr>
            <td colspan="5" class="py-4 px-2 text-center text-muted">No deployments found</td>
          </tr>
        {/if}
      </tbody>
    </table>
    <p class="mt-2 text-xs text-muted">{rows.length} deployment{rows.length === 1 ? '' : 's'} total</p>
  {/if}
</div>
