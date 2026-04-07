<script lang="ts">
  interface K8sContext {
    list(gvr: string, ns?: string): Promise<any[]>
  }
  interface PluginContext {
    cluster: { name: string; version: string }
    namespace: string
    k8s?: K8sContext
  }

  let { ctx }: { ctx: PluginContext } = $props()

  interface TaintKey {
    key: string
    count: number
    effects: string[]
  }

  interface NodeRow {
    name: string
    ready: boolean
    readiness: string
    taintCount: number
  }

  let overlay: HTMLElement
  let loading = $state(true)
  let errorMsg = $state<string | null>(null)
  let nodes = $state<NodeRow[]>([])
  let taintKeys = $state<TaintKey[]>([])

  $effect(() => {
    if (!ctx?.k8s) {
      loading = false
      return
    }
    ;(async () => {
      try {
        const items = await ctx.k8s!.list('core.v1.nodes')
        nodes = items.map((n: any) => ({
          name: n.metadata?.name ?? '',
          readiness: n.status?.readinessSummary ?? 'Unknown',
          ready: (n.status?.readinessSummary ?? '') === 'Ready',
          taintCount: n.status?.taintCount ?? n.spec?.taints?.length ?? 0,
        }))

        const keyMap = new Map<string, { count: number; effects: Set<string> }>()
        for (const n of items) {
          for (const t of (n as any).spec?.taints ?? []) {
            const entry = keyMap.get(t.key) ?? { count: 0, effects: new Set() }
            entry.count++
            if (t.effect) entry.effects.add(t.effect)
            keyMap.set(t.key, entry)
          }
        }
        taintKeys = [...keyMap.entries()].map(([key, v]) => ({
          key,
          count: v.count,
          effects: [...v.effects],
        }))

        loading = false
      } catch (e: unknown) {
        errorMsg = e instanceof Error ? e.message : String(e)
        loading = false
      }
    })()
  })

  const readyCount = $derived(nodes.filter((n) => n.ready).length)

  function dismiss() {
    overlay?.remove()
  }
</script>

<!-- Full-screen backdrop -->
<div
  bind:this={overlay}
  class="fixed inset-0 z-50 flex items-center justify-center"
  style="background: rgba(0,0,0,0.5);"
  role="dialog"
  aria-modal="true"
>
  <!-- Panel -->
  <div
    class="bg-surface border border-border rounded-lg shadow-xl w-[560px] max-h-[80vh] flex flex-col overflow-hidden"
    style="font-family: inherit;"
  >
    <!-- Header -->
    <div class="flex items-center justify-between px-4 py-3 border-b border-border shrink-0">
      <h2 class="text-sm font-semibold">Cluster Node Health</h2>
      <button
        onclick={dismiss}
        class="text-muted hover:text-fg transition-colors text-lg leading-none"
        aria-label="Close"
      >✕</button>
    </div>

    <div class="flex-1 overflow-y-auto p-4 space-y-4">
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
        <!-- Summary bar -->
        <div class="rounded border border-border bg-surface p-3 flex gap-6 text-xs">
          <div>
            <div class="text-muted mb-0.5">Nodes Ready</div>
            <div class="font-semibold text-sm" class:text-green-500={readyCount === nodes.length} class:text-red-500={readyCount < nodes.length}>
              {readyCount} / {nodes.length}
            </div>
          </div>
          <div>
            <div class="text-muted mb-0.5">Unique Taint Keys</div>
            <div class="font-semibold text-sm">{taintKeys.length}</div>
          </div>
          <div>
            <div class="text-muted mb-0.5">Cluster</div>
            <div class="font-semibold text-sm font-mono">{ctx.cluster.name || '—'}</div>
          </div>
        </div>

        <!-- Node table -->
        <div>
          <h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Nodes</h3>
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
                <tr class="border-b border-border hover:bg-surface-hover">
                  <td class="py-1.5 px-2 font-mono">{node.name}</td>
                  <td
                    class="py-1.5 px-2 font-medium"
                    class:text-green-500={node.ready}
                    class:text-red-500={!node.ready && node.readiness !== 'Unknown'}
                    class:text-muted={node.readiness === 'Unknown'}
                  >{node.readiness}</td>
                  <td class="py-1.5 px-2 text-right">{node.taintCount}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <!-- Taint key breakdown -->
        {#if taintKeys.length > 0}
          <div>
            <h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Taint Key Breakdown</h3>
            <table class="w-full text-xs border-collapse">
              <thead>
                <tr class="border-b border-border text-muted">
                  <th class="text-left py-1.5 px-2 font-medium">Key</th>
                  <th class="text-left py-1.5 px-2 font-medium">Effects</th>
                  <th class="text-right py-1.5 px-2 font-medium">Nodes</th>
                </tr>
              </thead>
              <tbody>
                {#each taintKeys as tk (tk.key)}
                  <tr class="border-b border-border hover:bg-surface-hover">
                    <td class="py-1.5 px-2 font-mono text-destructive">{tk.key}</td>
                    <td class="py-1.5 px-2 text-muted">{tk.effects.join(', ') || '—'}</td>
                    <td class="py-1.5 px-2 text-right">{tk.count}</td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      {/if}
    </div>
  </div>
</div>
