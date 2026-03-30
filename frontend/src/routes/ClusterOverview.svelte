<script lang="ts">
  import { clusterStore } from '$lib/stores/cluster.svelte'

  let { params = {} }: { params?: Record<string, string> } = $props()

  let ctxName = $derived(params.ctx ?? 'unknown')

  $effect(() => { if (ctxName) clusterStore.setActiveContext(ctxName) })
</script>

<div class="p-6">
  <h1 class="text-2xl font-semibold mb-4">Cluster: {ctxName}</h1>
  <p class="text-muted">Cluster overview — resource browsing coming in Phase 2</p>
</div>
