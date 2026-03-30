<script lang="ts">
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
    <h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Replicas</h3>
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
    <h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Update Strategy</h3>
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
      <h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Selector</h3>
      <div class="flex flex-wrap gap-1.5">
        {#each Object.entries(selector) as [k, v]}
          <span class="text-xs font-mono bg-surface border border-border rounded px-2 py-0.5">
            <span class="text-accent">{k}</span><span class="text-muted">=</span>{v}
          </span>
        {/each}
      </div>
    </section>
  {/if}

  <!-- Conditions -->
  {#if conditions.length > 0}
    <section>
      <h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Conditions</h3>
      <table class="w-full text-xs">
        <thead class="bg-surface">
          <tr>
            <th class="text-left px-2 py-1.5 font-medium text-muted">Type</th>
            <th class="text-left px-2 py-1.5 font-medium text-muted">Status</th>
            <th class="text-left px-2 py-1.5 font-medium text-muted">Reason</th>
            <th class="text-left px-2 py-1.5 font-medium text-muted">Message</th>
          </tr>
        </thead>
        <tbody>
          {#each conditions as cond}
            <tr class="border-t border-border">
              <td class="px-2 py-1.5 font-mono">{cond.type}</td>
              <td class="px-2 py-1.5">
                <span class="{cond.status === 'True' ? 'text-green-600 dark:text-green-400' : 'text-muted'}">{cond.status}</span>
              </td>
              <td class="px-2 py-1.5 text-muted">{cond.reason ?? '—'}</td>
              <td class="px-2 py-1.5 text-muted max-w-xs truncate">{cond.message ?? '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </section>
  {/if}
</div>
