<script lang="ts">
  import {CodeBlock, CopyableValue} from "@klados/ui";
  import EventTypeBadge from "$lib/event/EventTypeBadge.svelte";
  import {
    classifySeverity,
    rowInvolvedObject,
    rowMessage,
    rowSample,
    rowLastSeen,
    rowFirstSeen,
    rowCount,
    rowReason,
  } from "$lib/event/event-columns";
  import {clusterStore} from "$lib/stores/cluster.svelte";
  import {formatAge} from "$lib/utils/age";
  import {isGrouped, type EventRow, type InvolvedObjectRef} from "$lib/event/event-types";
  import {stringify as yamlStringify} from "yaml";

  let {
    event,
    now,
    onOpenInvolvedObject,
  }: {
    event: EventRow;
    now: number;
    onOpenInvolvedObject?: (ref: InvolvedObjectRef, gvr: string) => void;
  } = $props();

  const severity = $derived(classifySeverity(event));
  const io = $derived(rowInvolvedObject(event));
  const sample = $derived(rowSample(event));
  const lastSeen = $derived(rowLastSeen(event));
  const firstSeen = $derived(rowFirstSeen(event));
  const count = $derived(rowCount(event));
  const resolvedGVR = $derived(io.kind ? clusterStore.resolveOwnerGVR(io.apiVersion, io.kind) : undefined);
  const canNavigate = $derived(Boolean(resolvedGVR && onOpenInvolvedObject));
  const yaml = $derived(yamlStringify(sample));
  let yamlOpen = $state(false);
</script>

<div class="flex flex-col h-full overflow-auto">
  <div class="flex items-center gap-2 px-4 py-3 border-b border-border shrink-0">
    <EventTypeBadge {severity} />
    <span class="font-mono text-sm">{rowReason(event)}</span>
    <span class="text-xs text-muted ml-auto">{lastSeen ? `${formatAge(lastSeen, now)} ago` : '—'}</span>
  </div>

  {#if isGrouped(event)}
    <div class="px-4 py-1.5 text-xs text-muted border-b border-border">
      Grouped: {count} occurrences
    </div>
  {/if}

  <button
    type="button"
    disabled={!canNavigate}
    onclick={() => { if (canNavigate && resolvedGVR) onOpenInvolvedObject?.(io, resolvedGVR) }}
    class="mx-4 mt-3 text-left border border-border rounded p-3 flex items-center gap-3 transition-colors
      {canNavigate ? 'hover:bg-surface-hover cursor-pointer' : 'opacity-70 cursor-not-allowed'}"
    title={canNavigate ? 'Open involved object' : `No descriptor registered for Kind ${io.kind}`}
    data-testid="involved-object-card"
  >
    <span class="text-xs text-muted">{io.kind}</span>
    <span class="text-sm font-medium">{io.name}</span>
    {#if io.namespace}
      <span class="text-xs text-muted ml-auto">{io.namespace}</span>
    {/if}
  </button>

  <div class="px-4 py-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 text-xs">
    <span class="text-muted">Count</span><span>{count}</span>
    <span class="text-muted">First seen</span><span>{firstSeen ? `${formatAge(firstSeen, now)} ago` : '—'}</span>
    <span class="text-muted">Last seen</span><span>{lastSeen ? `${formatAge(lastSeen, now)} ago` : '—'}</span>
    <span class="text-muted">Source</span>
    <span>{(sample.source?.component ?? '') + (sample.source?.host ? ' @ ' + sample.source.host : '') || '—'}</span>
    <span class="text-muted">Reporting ctrl</span>
    <span>{sample.reportingController ?? '—'}</span>
  </div>

  <div class="px-4 py-3 border-t border-border">
    <div class="text-xs text-muted mb-1">Message</div>
    <CopyableValue value={rowMessage(event)} class="font-mono text-xs whitespace-pre-wrap" />
  </div>

  <div class="px-4 py-3 border-t border-border">
    <button type="button" onclick={() => yamlOpen = !yamlOpen} class="text-xs text-muted hover:text-fg">
      {yamlOpen ? '▾' : '▸'} Raw YAML
    </button>
    {#if yamlOpen}
      <div class="mt-2">
        <CodeBlock lang="yaml" value={yaml} />
      </div>
    {/if}
  </div>
</div>
