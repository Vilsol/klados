<script lang="ts">
  import {SectionHeader, EmptyState, StatusBadge, DataTable} from "@klados/ui";
  import type {KubernetesResource} from "$lib/types";

  let {obj}: {obj: Record<string, KubernetesResource>} = $props();

  const conditions = $derived<KubernetesResource[]>(obj.status?.conditions ?? []);
  const taints = $derived<KubernetesResource[]>(obj.spec?.taints ?? []);
</script>

<div class="p-4 space-y-6">
  <section>
    <SectionHeader>Conditions</SectionHeader>
    {#if conditions.length === 0}
      <EmptyState message="No conditions" size="sm" />
    {:else}
      <DataTable
        columns={[{ label: 'Type' }, { label: 'Status', width: '5rem' }, { label: 'Reason', width: '8rem' }, { label: 'Message' }]}
        items={conditions}
      >
        {#snippet row(cond)}
          <td class="px-3 py-2 font-medium">{cond.type ?? ''}</td>
          <td class="px-3 py-2"><StatusBadge status={cond.status} mode="pill">{cond.status ?? ''}</StatusBadge></td>
          <td class="px-3 py-2 text-muted">{cond.reason ?? ''}</td>
          <td class="px-3 py-2 text-muted truncate max-w-xs" title={cond.message ?? ''}>{cond.message ?? ''}</td>
        {/snippet}
      </DataTable>
    {/if}
  </section>

  <section>
    <SectionHeader>Taints</SectionHeader>
    {#if taints.length === 0}
      <EmptyState message="No taints" size="sm" />
    {:else}
      <div class="border border-border rounded overflow-hidden">
        <table class="w-full text-xs">
          <thead class="bg-surface">
            <tr>
              <th class="px-3 py-2 text-left font-medium text-muted">Key</th>
              <th class="px-3 py-2 text-left font-medium text-muted">Value</th>
              <th class="px-3 py-2 text-left font-medium text-muted w-28">Effect</th>
            </tr>
          </thead>
          <tbody>
            {#each taints as taint}
              <tr class="border-t border-border">
                <td class="px-3 py-2 font-medium">{taint.key ?? ''}</td>
                <td class="px-3 py-2 text-muted">{taint.value ?? ''}</td>
                <td class="px-3 py-2">
                  <span class="px-1.5 py-0.5 rounded text-xs font-medium bg-surface border border-border text-muted">
                    {taint.effect ?? ''}
                  </span>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </section>
</div>
