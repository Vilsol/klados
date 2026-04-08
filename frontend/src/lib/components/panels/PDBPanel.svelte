<script lang="ts">
  import { SectionHeader, KeyValueBadge, StatusBadge, DataTable } from '@klados/ui'

  let { obj }: { obj: Record<string, any> } = $props()

  const selector = $derived<Record<string, string>>(obj.spec?.selector?.matchLabels ?? {})
  const conditions = $derived<any[]>(obj.status?.conditions ?? [])
  const disruptionsAllowed = $derived<number>(obj.status?.disruptionsAllowed ?? 0)
  const expectedPods = $derived<number>(obj.status?.expectedPods ?? 0)
  const currentHealthy = $derived<number>(obj.status?.currentHealthy ?? 0)
  const desiredHealthy = $derived<number>(obj.status?.desiredHealthy ?? 0)
  const healthPct = $derived(expectedPods > 0 ? Math.min(100, (currentHealthy / expectedPods) * 100) : 0)
</script>

<div class="flex flex-col gap-6 p-4 overflow-auto">
  {#if Object.keys(selector).length > 0}
    <section>
      <SectionHeader>Selector</SectionHeader>
      <KeyValueBadge entries={Object.entries(selector)} />
    </section>
  {/if}

  <section>
    <SectionHeader>Budget</SectionHeader>
    <div class="text-xs">
      {#if obj.spec?.minAvailable !== undefined}
        <span class="text-muted">Min Available:</span>
        <span class="font-mono ml-2">{obj.spec.minAvailable}</span>
      {:else if obj.spec?.maxUnavailable !== undefined}
        <span class="text-muted">Max Unavailable:</span>
        <span class="font-mono ml-2">{obj.spec.maxUnavailable}</span>
      {:else}
        <span class="text-muted">—</span>
      {/if}
    </div>
  </section>

  <section>
    <SectionHeader>Health</SectionHeader>
    <div class="flex items-center gap-3 mb-3">
      <div class="flex-1 h-3 bg-border rounded-full overflow-hidden">
        <div
          class="{disruptionsAllowed > 0 ? 'bg-green-500' : 'bg-red-500'} h-full rounded-full transition-all"
          style="width: {healthPct}%"
        ></div>
      </div>
      <span class="text-xs text-muted">{currentHealthy} / {expectedPods}</span>
    </div>
    <div class="grid grid-cols-4 gap-3">
      {#each [
        { label: 'Expected', value: expectedPods },
        { label: 'Healthy', value: currentHealthy },
        { label: 'Desired', value: desiredHealthy },
        { label: 'Disruptions Allowed', value: disruptionsAllowed },
      ] as stat}
        <div class="bg-surface border border-border rounded-lg p-3 text-center">
          <div class="text-2xl font-semibold">{stat.value}</div>
          <div class="text-xs text-muted mt-0.5">{stat.label}</div>
        </div>
      {/each}
    </div>
  </section>

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
