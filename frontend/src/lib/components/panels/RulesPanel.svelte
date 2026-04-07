<script lang="ts">
  import { SectionHeader, EmptyState } from '@klados/ui'

  let { obj }: { obj: Record<string, any> } = $props()

  const rules = $derived<any[]>(obj.rules ?? [])
</script>

<div class="p-4 space-y-6">
  <section>
    <SectionHeader>Rules</SectionHeader>
    {#if rules.length === 0}
      <EmptyState message="No rules" />
    {:else}
      <div class="border border-border rounded overflow-hidden">
        <table class="w-full text-xs">
          <thead class="bg-surface">
            <tr>
              <th class="px-3 py-2 text-left font-medium text-muted">API Groups</th>
              <th class="px-3 py-2 text-left font-medium text-muted">Resources</th>
              <th class="px-3 py-2 text-left font-medium text-muted">Verbs</th>
              <th class="px-3 py-2 text-left font-medium text-muted">Resource Names</th>
            </tr>
          </thead>
          <tbody>
            {#each rules as rule}
              <tr class="border-t border-border">
                <td class="px-3 py-2 font-mono">{(rule.apiGroups ?? []).map((g: string) => g === '' ? '""' : g).join(', ')}</td>
                <td class="px-3 py-2">{(rule.resources ?? []).join(', ')}</td>
                <td class="px-3 py-2">{(rule.verbs ?? []).join(', ')}</td>
                <td class="px-3 py-2 text-muted">{rule.resourceNames?.length ? rule.resourceNames.join(', ') : '*'}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </section>
</div>
