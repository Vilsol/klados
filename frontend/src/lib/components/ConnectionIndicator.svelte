<script lang="ts">
  import type { ConnectionStatusType } from '$lib/stores/cluster.svelte'

  let { status, clusterName }: { status: ConnectionStatusType; clusterName: string } = $props()

  let dotClass = $derived(
    status === 'connected'
      ? 'bg-green-500'
      : status === 'connecting'
        ? 'bg-yellow-500 animate-pulse'
        : status === 'error'
          ? 'bg-red-500 animate-pulse'
          : 'bg-gray-400',
  )

  let label = $derived(
    status === 'connected'
      ? `Connected to ${clusterName}`
      : status === 'connecting'
        ? `Connecting to ${clusterName}...`
        : status === 'error'
          ? `Error connecting to ${clusterName}`
          : `Disconnected from ${clusterName}`,
  )
</script>

<span class="relative inline-flex items-center" title={label}>
  <span class="inline-block w-2.5 h-2.5 rounded-full {dotClass}"></span>
</span>
