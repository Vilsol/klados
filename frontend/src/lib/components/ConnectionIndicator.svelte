<script lang="ts">
  import type {ConnectionStatusType} from "$lib/stores/cluster.svelte";

  let {status, clusterName}: {status: ConnectionStatusType; clusterName: string} = $props();

  function getDotClass(s: ConnectionStatusType): string {
    if (s === "connected") {
      return "bg-green-500";
    }
    if (s === "connecting") {
      return "bg-yellow-500 animate-pulse";
    }
    if (s === "error") {
      return "bg-red-500 animate-pulse";
    }
    return "bg-gray-400";
  }

  function getLabel(s: ConnectionStatusType, name: string): string {
    if (s === "connected") {
      return `Connected to ${name}`;
    }
    if (s === "connecting") {
      return `Connecting to ${name}...`;
    }
    if (s === "error") {
      return `Error connecting to ${name}`;
    }
    return `Disconnected from ${name}`;
  }

  let dotClass = $derived(getDotClass(status));
  let label = $derived(getLabel(status, clusterName));
</script>

<span class="relative inline-flex items-center" title={label}>
  <span class="inline-block w-2.5 h-2.5 rounded-full {dotClass}"></span>
</span>
