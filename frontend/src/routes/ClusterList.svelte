<script lang="ts">
  import { onMount } from 'svelte'
  import { clusterStore } from '$lib/stores/cluster.svelte'
  import ConnectionIndicator from '$lib/components/ConnectionIndicator.svelte'
  import KubeconfigImportDialog from '$lib/components/KubeconfigImportDialog.svelte'

  let showImportDialog = $state(false)

  onMount(() => {
    clusterStore.loadContexts()
  })
</script>

<KubeconfigImportDialog
  bind:open={showImportDialog}
  onsuccess={() => clusterStore.loadContexts()}
/>

<div class="p-6">
  <div class="flex items-center justify-between mb-6">
    <h1 class="text-2xl font-semibold">Clusters</h1>
    <button
      onclick={() => showImportDialog = true}
      class="px-3 py-1.5 text-sm rounded border border-border hover:bg-surface-hover transition-colors"
    >
      Import Kubeconfig
    </button>
  </div>

  {#if clusterStore.contexts.length === 0}
    <div class="text-muted text-center py-12">
      <p class="text-lg">No clusters found</p>
      <p class="text-sm mt-2">Configure kubeconfig paths in settings</p>
    </div>
  {:else}
    <div class="grid gap-4 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
      {#each clusterStore.contexts as ctx}
        {@const status = clusterStore.connectionStatus[ctx.name] ?? 'disconnected'}
        <div class="rounded-lg border border-border bg-surface p-4 hover:bg-surface-hover transition-colors">
          <div class="flex items-center justify-between mb-2">
            <h2 class="font-medium truncate">{ctx.name}</h2>
            <div class="flex items-center gap-1.5 shrink-0">
              {#if ctx.provider}
                <span class="text-xs px-1.5 py-0.5 rounded border border-border text-muted">{ctx.provider}</span>
              {/if}
              <ConnectionIndicator {status} clusterName={ctx.name} />
            </div>
          </div>
          <div class="text-sm text-muted space-y-1">
            <p>Cluster: {ctx.cluster}</p>
            <p>User: {ctx.user}</p>
            <p>Namespace: {ctx.namespace}</p>
            {#if ctx.serverVersion}
              <p>Version: {ctx.serverVersion}</p>
            {/if}
          </div>
          <div class="mt-3">
            {#if status === 'connected'}
              <button
                onclick={() => clusterStore.disconnect(ctx.name)}
                class="px-3 py-1.5 text-sm rounded bg-destructive text-white hover:opacity-90 transition-opacity"
              >
                Disconnect
              </button>
            {:else}
              <button
                onclick={() => clusterStore.connect(ctx.name)}
                disabled={status === 'connecting'}
                class="px-3 py-1.5 text-sm rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity disabled:opacity-50"
              >
                {status === 'connecting' ? 'Connecting...' : 'Connect'}
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
