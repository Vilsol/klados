<script lang="ts">
  import {onDestroy} from "svelte";
  import {createResourceStore} from "$lib/stores/resource.svelte";
  import {clusterStore} from "$lib/stores/cluster.svelte";
  import {formatAge} from "$lib/utils/age";
  import {RefreshCw} from "lucide-svelte";

  let {params = {}}: {params?: Record<string, string>} = $props();

  const ctxName = $derived(params.ctx ?? "");

  $effect(() => {
    if (ctxName) {
      clusterStore.setActiveContext(ctxName);
    }
  });

  const store = createResourceStore();
  const EVENTS_GVR = "core.v1.events";

  $effect(() => {
    if (ctxName) {
      store.start(ctxName, EVENTS_GVR, "");
    }
    return () => store.stop();
  });

  let showWarning = $state(true);
  let showNormal = $state(true);
  let reasonFilter = $state("");

  const selectedNs = $derived(clusterStore.getSelectedNamespaces(ctxName));

  let now = $state(Date.now());
  const ticker = setInterval(() => {
    now = Date.now();
  }, 1_000);
  onDestroy(() => clearInterval(ticker));

  const filtered = $derived.by(() => {
    return store.items
      .filter((e) => {
        const type = e.type ?? "Normal";
        if (type === "Warning" && !showWarning) {
          return false;
        }
        if (type === "Normal" && !showNormal) {
          return false;
        }
        if (reasonFilter && !(e.reason ?? "").toLowerCase().includes(reasonFilter.toLowerCase())) {
          return false;
        }
        if (selectedNs.length > 0 && !selectedNs.includes(e.metadata?.namespace ?? "")) {
          return false;
        }
        return true;
      })
      .sort((a, b) => {
        const ta = a.lastTimestamp ?? a.eventTime ?? a.metadata?.creationTimestamp ?? "";
        const tb = b.lastTimestamp ?? b.eventTime ?? b.metadata?.creationTimestamp ?? "";
        return tb.localeCompare(ta);
      });
  });
</script>

<div class="flex flex-col h-full">
  <div class="shrink-0 px-4 py-3 border-b border-border flex items-center gap-3 flex-wrap">
    <h1 class="text-sm font-semibold">Event Stream</h1>
    <span class="text-xs text-muted">{ctxName}</span>

    <div class="flex items-center gap-2 ml-2">
      <label class="flex items-center gap-1 text-xs cursor-pointer">
        <input type="checkbox" bind:checked={showWarning} class="accent-destructive">
        <span class="text-destructive font-medium">Warning</span>
      </label>
      <label class="flex items-center gap-1 text-xs cursor-pointer">
        <input type="checkbox" bind:checked={showNormal} class="accent-accent">
        <span>Normal</span>
      </label>
    </div>

    <input
      type="text"
      placeholder="Filter reason…"
      bind:value={reasonFilter}
      class="text-xs bg-bg border border-border rounded px-2 py-1 outline-none focus:ring-1 focus:ring-accent w-36"
    >

    <span class="text-xs text-muted ml-auto">{filtered.length} events</span>

    {#if store.loading}
      <RefreshCw size={14} class="animate-spin text-muted" />
    {/if}
  </div>

  <div class="flex-1 overflow-auto">
    {#if store.error}
      <div class="p-4 text-sm text-destructive">{store.error}</div>
    {:else if filtered.length === 0 && !store.loading}
      <div class="flex items-center justify-center py-16 text-sm text-muted">No events found</div>
    {:else}
      <table class="w-full text-xs">
        <thead class="sticky top-0 bg-surface border-b border-border z-10">
          <tr>
            <th class="text-left px-3 py-2 font-medium text-muted w-20">Type</th>
            <th class="text-left px-3 py-2 font-medium text-muted w-32">Reason</th>
            <th class="text-left px-3 py-2 font-medium text-muted w-40">Object</th>
            <th class="text-left px-3 py-2 font-medium text-muted">Message</th>
            <th class="text-left px-3 py-2 font-medium text-muted w-12">Count</th>
            <th class="text-left px-3 py-2 font-medium text-muted w-24">Age</th>
          </tr>
        </thead>
        <tbody>
          {#each filtered as event (event.metadata?.uid ?? event.metadata?.name)}
            {@const type = event.type ?? 'Normal'}
            {@const reason = event.reason ?? ''}
            {@const objName = event.involvedObject?.name ?? ''}
            {@const objKind = event.involvedObject?.kind ?? ''}
            {@const message = event.message ?? ''}
            {@const count = event.count ?? 1}
            {@const ts = event.lastTimestamp ?? event.eventTime ?? event.metadata?.creationTimestamp ?? ''}
            <tr
              class="border-b border-border hover:bg-surface-hover
              {type === 'Warning' ? 'bg-destructive/5' : ''}"
            >
              <td class="px-3 py-1.5">
                <span
                  class="px-1.5 py-0.5 rounded text-xs font-medium
                  {type === 'Warning' ? 'bg-destructive/15 text-destructive' : 'bg-accent/15 text-accent'}"
                >
                  {type}
                </span>
              </td>
              <td class="px-3 py-1.5 font-mono text-muted truncate max-w-[128px]">{reason}</td>
              <td class="px-3 py-1.5 truncate max-w-[160px]"><span class="text-muted">{objKind}/</span>{objName}</td>
              <td class="px-3 py-1.5 text-muted max-w-xs truncate">{message}</td>
              <td class="px-3 py-1.5 text-muted">{count}</td>
              <td class="px-3 py-1.5 text-muted">{ts ? formatAge(ts, now) : '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>
