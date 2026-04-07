<script lang="ts">
  import { SectionHeader, KeyValueBadge, StatusBadge, DataTable } from '@klados/ui'

  let { obj }: { obj: Record<string, any> } = $props()

  const strategy = $derived(obj.spec?.strategy ?? {})
  const selector = $derived(obj.spec?.selector?.matchLabels ?? {})
  const conditions = $derived<any[]>(obj.status?.conditions ?? [])
  const replicas = $derived({
    desired: obj.spec?.replicas ?? 0,
    ready: obj.status?.readyReplicas ?? 0,
    available: obj.status?.availableReplicas ?? 0,
    updated: obj.status?.updatedReplicas ?? 0,
  })
</script>

<div class="flex flex-col gap-6 p-4 overflow-auto">
  <!-- Replica counts -->
  <section>
    <SectionHeader>Replicas</SectionHeader>
    <div class="grid grid-cols-4 gap-3">
      {#each [
        { label: 'Desired', value: replicas.desired },
        { label: 'Ready', value: replicas.ready },
        { label: 'Available', value: replicas.available },
        { label: 'Updated', value: replicas.updated },
      ] as r}
        <div class="bg-surface border border-border rounded-lg p-3 text-center">
          <div class="text-2xl font-semibold">{r.value}</div>
          <div class="text-xs text-muted mt-0.5">{r.label}</div>
        </div>
      {/each}
    </div>
  </section>

  <!-- Strategy -->
  <section>
    <SectionHeader>Update Strategy</SectionHeader>
    <div class="grid grid-cols-[auto_1fr] gap-x-6 gap-y-1.5 text-xs">
      <span class="text-muted">Type</span>
      <span class="font-mono font-medium">{strategy.type ?? '—'}</span>
      {#if strategy.rollingUpdate}
        <span class="text-muted">Max Surge</span>
        <span class="font-mono">{strategy.rollingUpdate.maxSurge ?? '—'}</span>
        <span class="text-muted">Max Unavailable</span>
        <span class="font-mono">{strategy.rollingUpdate.maxUnavailable ?? '—'}</span>
      {/if}
    </div>
  </section>

  <!-- Selector labels -->
  {#if Object.keys(selector).length > 0}
    <section>
      <SectionHeader>Selector</SectionHeader>
      <KeyValueBadge entries={Object.entries(selector)} />
    </section>
  {/if}

  <!-- Conditions -->
  {#if conditions.length > 0}
    <section>
      <SectionHeader>Conditions</SectionHeader>
      <DataTable
        columns={[{ label: 'Type' }, { label: 'Status' }, { label: 'Reason' }, { label: 'Message' }]}
        items={conditions}
      >
        {#snippet row(cond)}
          <td class="px-2 py-1.5 font-mono">{cond.type}</td>
          <td class="px-2 py-1.5">
            <StatusBadge status={cond.status}>{cond.status}</StatusBadge>
          </td>
          <td class="px-2 py-1.5 text-muted">{cond.reason ?? '—'}</td>
          <td class="px-2 py-1.5 text-muted max-w-xs truncate">{cond.message ?? '—'}</td>
        {/snippet}
      </DataTable>
    </section>
  {/if}
</div>
