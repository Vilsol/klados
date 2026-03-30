<script lang="ts">
  let { obj }: { obj: Record<string, any> } = $props()

  const conditions = $derived<any[]>(obj.status?.conditions ?? [])
  const taints = $derived<any[]>(obj.spec?.taints ?? [])
</script>

<div class="p-4 space-y-6">
  <section>
    <h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Conditions</h3>
    {#if conditions.length === 0}
      <p class="text-sm text-muted">No conditions</p>
    {:else}
      <div class="border border-border rounded overflow-hidden">
        <table class="w-full text-xs">
          <thead class="bg-surface">
            <tr>
              <th class="px-3 py-2 text-left font-medium text-muted">Type</th>
              <th class="px-3 py-2 text-left font-medium text-muted w-20">Status</th>
              <th class="px-3 py-2 text-left font-medium text-muted w-32">Reason</th>
              <th class="px-3 py-2 text-left font-medium text-muted">Message</th>
            </tr>
          </thead>
          <tbody>
            {#each conditions as cond}
              <tr class="border-t border-border">
                <td class="px-3 py-2 font-medium">{cond.type ?? ''}</td>
                <td class="px-3 py-2">
                  <span class="px-1.5 py-0.5 rounded text-xs font-medium
                    {cond.status === 'True' ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400' :
                     cond.status === 'False' ? 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400' :
                     'bg-surface text-muted border border-border'}">
                    {cond.status ?? ''}
                  </span>
                </td>
                <td class="px-3 py-2 text-muted">{cond.reason ?? ''}</td>
                <td class="px-3 py-2 text-muted truncate max-w-xs" title={cond.message ?? ''}>{cond.message ?? ''}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </section>

  <section>
    <h3 class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Taints</h3>
    {#if taints.length === 0}
      <p class="text-sm text-muted">No taints</p>
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
