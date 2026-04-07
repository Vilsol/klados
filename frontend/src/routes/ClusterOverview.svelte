<script lang="ts">
  import { untrack } from 'svelte'
  import { clusterStore } from '$lib/stores/cluster.svelte'
  import * as MetricsService from '../../bindings/github.com/Vilsol/klados/internal/services/metricsservice.js'
  import MetricsChart from '$lib/components/charts/MetricsChart.svelte'
  import TimeRangeSelector from '$lib/components/charts/TimeRangeSelector.svelte'
  import { Combobox } from '@klados/ui'
  import type { MetricsCapability, MetricsResponse, TimeSeries } from '$lib/components/charts/types'

  let { params = {} }: { params?: Record<string, string> } = $props()

  let ctxName = $derived(params.ctx ?? 'unknown')

  $effect(() => { if (ctxName) clusterStore.setActiveContext(ctxName) })

  let namespaces = $derived(clusterStore.getNamespaces(ctxName))
  let selectedNamespace = $state('')

  // Auto-select first namespace when list loads and nothing is selected
  $effect(() => {
    const ns = namespaces
    untrack(() => {
      if (selectedNamespace === '' && ns.length > 0) selectedNamespace = ns[0]
    })
  })

  let capability: MetricsCapability | null = $state(null)
  let rangeMinutes: number = $state(15)
  let response: MetricsResponse | null = $state(null)
  let zoomRange: { min: number; max: number } | null = $state(null)
  let fetchError: string | null = $state(null)

  $effect(() => {
    MetricsService.GetCapabilities(ctxName)
      .then((cap) => { capability = cap as unknown as MetricsCapability })
      .catch(() => {})
  })

  $effect(() => {
    const ns = selectedNamespace
    void [ctxName, ns]
    untrack(() => { response = null; fetchError = null; zoomRange = null })
  })

  $effect(() => {
    const ns = selectedNamespace
    if (!ns || !capability?.hasPrometheus) return

    const interval = rangeMinutes <= 60 ? 15_000 : 60_000

    async function fetchMetrics() {
      try {
        const res = await MetricsService.GetNamespaceMetrics(ctxName, ns!, rangeMinutes)
        fetchError = null
        if (res) response = res as unknown as MetricsResponse
      } catch (err: unknown) {
        fetchError = err instanceof Error ? err.message : String(err)
      }
    }

    fetchMetrics()
    const id = setInterval(fetchMetrics, interval)
    return () => clearInterval(id)
  })

  function getSeriesByUnit(unit: string): TimeSeries[] {
    if (!response) return []
    const metric = response.metrics.find((m) => m.unit === unit || m.name.toLowerCase().includes(unit === 'cores' ? 'cpu' : 'mem'))
    return metric?.series ?? []
  }
</script>

<div class="p-6 flex flex-col gap-6">
  <div class="flex items-center justify-between">
    <h1 class="text-2xl font-semibold">Cluster: {ctxName}</h1>

    {#if namespaces.length > 0}
      <div class="w-48">
        <Combobox
          bind:value={selectedNamespace}
          options={namespaces.map((ns) => ({ value: ns, label: ns }))}
          placeholder="Select namespace"
        />
      </div>
    {/if}
  </div>

  {#if selectedNamespace}
    {#if !capability?.hasPrometheus}
      <p class="text-xs text-muted">Configure a Prometheus endpoint to view namespace metrics.</p>
    {:else}
      <div class="flex items-center justify-between">
        <span class="text-sm font-medium">Namespace: {selectedNamespace}</span>
        <div class="flex items-center gap-2">
          <TimeRangeSelector value={rangeMinutes} hasPrometheus={true} onchange={(v) => (rangeMinutes = v)} />
          <span class="text-xs text-muted">prometheus</span>
        </div>
      </div>

      {#if fetchError}
        <div class="text-xs text-destructive font-mono">{fetchError}</div>
      {/if}

      <MetricsChart
        title="CPU Usage"
        unit="cores"
        series={getSeriesByUnit('cores')}
        loading={!response && !fetchError}
        {zoomRange}
        onzoom={(r) => (zoomRange = r)}
      />
      <MetricsChart
        title="Memory Usage"
        unit="bytes"
        series={getSeriesByUnit('bytes')}
        loading={!response && !fetchError}
        {zoomRange}
        onzoom={(r) => (zoomRange = r)}
      />
    {/if}
  {:else}
    <p class="text-muted text-sm">No namespaces found.</p>
  {/if}
</div>
