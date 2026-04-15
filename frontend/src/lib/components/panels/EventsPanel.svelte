<script lang="ts">
  import {GetEvents} from "../../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js";
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
    namespace: string;
    uid: string;
  } = $props();

  let events = $state<Record<string, KubernetesResource>[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  $effect(() => {
    const currentCtx = ctxName;
    const currentNs = namespace;
    const currentUid = uid;
    let cancelled = false;
    loading = true;
    error = null;
    GetEvents(currentCtx, currentNs, currentUid)
      .then((result) => {
        if (!cancelled) {
          events = result ?? [];
          loading = false;
        }
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          error = e instanceof Error ? e.message : String(e);
          loading = false;
        }
      });
    return () => {
      cancelled = true;
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
          <tr class="border-b border-border hover:bg-surface-hover">
            <td class="px-3 py-1.5">
              <EventTypeBadge severity={classifySeverity(ev)} />
            </td>
            <td class="px-3 py-1.5 font-mono text-muted">{reason}</td>
            <td class="px-3 py-1.5 text-muted">{message}</td>
            <td class="px-3 py-1.5 text-muted">{count}</td>
            <td class="px-3 py-1.5 text-muted">{ts ? formatAge(ts) : '—'}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>
