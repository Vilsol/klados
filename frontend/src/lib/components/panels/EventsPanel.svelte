<script lang="ts">
  import {GetEvents} from "../../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js";
  import {Events as WailsEvents} from "@wailsio/runtime";
  import {formatAge} from "$lib/utils/age";
  import type {KubernetesResource} from "$lib/types";
  import EventTypeBadge from "$lib/event/EventTypeBadge.svelte";
  import {classifySeverity, eventTimestamp} from "$lib/event/event-columns";
  import type {EventItem} from "$lib/event/event-types";

  let {
    ctxName,
    namespace,
    uid,
  }: {
    ctxName: string;
    namespace: string; // "" for cluster-scoped resources (all-namespaces search)
    uid: string;
  } = $props();

  let events = $state<Record<string, KubernetesResource>[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  async function refresh(ctx: string, ns: string, uidVal: string) {
    try {
      loading = true;
      events = (await GetEvents(ctx, ns, uidVal)) ?? [];
      error = null;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    const currentCtx = ctxName;
    const currentNs = namespace;
    const currentUid = uid;

    void refresh(currentCtx, currentNs, currentUid);

    // Watch for new events in the relevant namespace (empty = all namespaces for cluster-scoped)
    const watchKey = `watch:${currentCtx}:core.v1.events:${currentNs}`;
    const unsub = WailsEvents.On(watchKey, (wailsEvent: unknown) => {
      const data = (wailsEvent as {data?: {object?: {involvedObject?: {uid?: string}}}}).data;
      if (data?.object?.involvedObject?.uid === currentUid) {
        void refresh(currentCtx, currentNs, currentUid);
      }
    });

    return () => {
      unsub?.();
    };
  });
</script>

<div class="flex flex-col h-full overflow-auto">
  {#if loading}
    <div class="p-4 text-sm text-muted">Loading events...</div>
  {:else if error}
    <div class="p-4 text-sm text-destructive">{error}</div>
  {:else if events.length === 0}
    <div class="p-4 text-sm text-muted">No events found.</div>
  {:else}
    <table class="w-full text-xs">
      <thead class="sticky top-0 bg-surface border-b border-border">
        <tr>
          <th class="text-left px-3 py-2 font-medium text-muted w-16">Type</th>
          <th class="text-left px-3 py-2 font-medium text-muted w-32">Reason</th>
          <th class="text-left px-3 py-2 font-medium text-muted">Message</th>
          <th class="text-left px-3 py-2 font-medium text-muted w-12">Count</th>
          <th class="text-left px-3 py-2 font-medium text-muted w-16">Age</th>
        </tr>
      </thead>
      <tbody>
        {#each events as event}
          {@const ev = event as EventItem}
          {@const reason = ev.reason ?? ''}
          {@const message = ev.message ?? ''}
          {@const count = ev.count ?? 1}
          {@const ts = eventTimestamp(ev)}
          {@const isWarning = classifySeverity(ev) === 'Warning'}
          <tr class="border-b border-border hover:bg-surface-hover {isWarning ? 'bg-amber-500/5' : ''}">
            <td class="px-3 py-1.5">
              <EventTypeBadge severity={classifySeverity(ev)} />
            </td>
            <td class="px-3 py-1.5 font-mono {isWarning ? 'text-amber-500' : 'text-muted'}">{reason}</td>
            <td class="px-3 py-1.5 text-muted">{message}</td>
            <td class="px-3 py-1.5 text-muted">{count}</td>
            <td class="px-3 py-1.5 text-muted">{ts ? formatAge(ts) : '—'}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>
