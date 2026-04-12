<script lang="ts">
  import {push} from "svelte-spa-router";
  import {clusterStore} from "$lib/stores/cluster.svelte";
</script>

<div class="max-w-2xl space-y-6">
  <h2 class="text-base font-medium text-fg mb-4">Clusters</h2>

  {#if clusterStore.contexts.length === 0}
    <p class="text-sm text-muted-foreground">No clusters found. Add kubeconfig paths to discover clusters.</p>
  {:else}
    <div class="border border-border rounded overflow-hidden divide-y divide-border">
      {#each clusterStore.contexts as ctx}
        <button
          class="w-full flex items-center justify-between px-4 py-3 text-left hover:bg-surface-hover transition-colors"
          onclick={() => push(`/settings/clusters/${encodeURIComponent(ctx.name)}`)}
        >
          <span class="text-sm text-fg">{ctx.name}</span>
          <svg class="w-4 h-4 text-muted-foreground" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </button>
      {/each}
    </div>
  {/if}
</div>
