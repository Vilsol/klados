<script lang="ts">
  import {SectionHeader, KeyValueBadge, DataTable} from "@klados/ui";

  let {obj}: {obj: Record<string, any>} = $props();

  const webhooks = $derived<any[]>(obj.webhooks ?? []);
  let expanded = $state<boolean[]>([]);

  $effect(() => {
    if (expanded.length !== webhooks.length) {
      expanded = webhooks.map(() => true);
    }
  });

  function failurePolicyClass(policy: string): string {
    if (policy === "Fail") {
      return "bg-red-500/20 text-red-400 border border-red-500/30";
    }
    if (policy === "Ignore") {
      return "bg-yellow-500/20 text-yellow-400 border border-yellow-500/30";
    }
    return "bg-surface border border-border text-muted";
  }
</script>

<div class="p-4 space-y-4">
  {#if webhooks.length === 0}
    <p class="text-xs text-muted">No webhooks defined.</p>
  {:else}
    {#each webhooks as wh, i}
      <div class="border border-border rounded-lg overflow-hidden">
        <!-- Header -->
        <button
          onclick={() => (expanded[i] = !expanded[i])}
          class="w-full flex items-center gap-3 px-4 py-2.5 bg-surface hover:bg-surface-hover transition-colors text-left"
        >
          <span class="text-xs font-medium flex-1 font-mono">{wh.name ?? `webhook-${i}`}</span>
          {#if wh.failurePolicy}
            <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium {failurePolicyClass(wh.failurePolicy)}">
              {wh.failurePolicy}
            </span>
          {/if}
          <span class="text-muted text-xs">{expanded[i] ? '▲' : '▼'}</span>
        </button>

        <!-- Body -->
        {#if expanded[i]}
          <div class="p-4 space-y-4 border-t border-border">
            <!-- Client config -->
            <section>
              <SectionHeader>Client Config</SectionHeader>
              <div class="grid grid-cols-[auto_1fr] gap-x-6 gap-y-1.5 text-xs">
                {#if wh.clientConfig?.service}
                  {@const svc = wh.clientConfig.service}
                  <span class="text-muted">Service</span>
                  <span class="font-mono"
                    >{svc.namespace}/{svc.name}{svc.port != null ? `:${svc.port}` : ''}{svc.path ? svc.path : ''}</span
                  >
                {:else if wh.clientConfig?.url}
                  <span class="text-muted">URL</span>
                  <span class="font-mono break-all">{wh.clientConfig.url}</span>
                {/if}
                <span class="text-muted">CA Bundle</span>
                <span class={wh.clientConfig?.caBundle ? 'text-fg' : 'text-muted italic'}>
                  {wh.clientConfig?.caBundle ? 'Present' : 'Not set'}
                </span>
              </div>
            </section>

            <!-- Match rules -->
            {#if wh.rules?.length}
              <section>
                <SectionHeader>Match Rules</SectionHeader>
                <DataTable
                  columns={[{ label: 'Operations' }, { label: 'API Groups' }, { label: 'Resources' }, { label: 'Scope' }]}
                  items={wh.rules}
                >
                  {#snippet row(rule)}
                    <td class="px-2 py-1.5">
                      <div class="flex flex-wrap gap-1">
                        {#each (rule.operations ?? []) as op}
                          <span class="px-1 py-0.5 rounded text-xs font-medium bg-surface border border-border font-mono">{op}</span>
                        {/each}
                      </div>
                    </td>
                    <td class="px-2 py-1.5 font-mono text-xs">
                      {(rule.apiGroups ?? []).map((g: string) => g === '' ? '""' : g).join(', ')}
                    </td>
                    <td class="px-2 py-1.5 font-mono text-xs">
                      {#each (rule.resources ?? []) as res, ri}
                        {#if ri > 0}
                          <span class="text-muted">, </span>
                        {/if}
                        {#if res === '*'}
                          <span class="text-accent font-bold">*</span>
                        {:else}
                          {res}
                        {/if}
                      {/each}
                    </td>
                    <td class="px-2 py-1.5 text-muted text-xs">{rule.scope ?? '*'}</td>
                  {/snippet}
                </DataTable>
              </section>
            {/if}

            <!-- Selectors -->
            {#if wh.namespaceSelector?.matchLabels && Object.keys(wh.namespaceSelector.matchLabels).length > 0}
              <section>
                <SectionHeader>Namespace Selector</SectionHeader>
                <KeyValueBadge entries={Object.entries(wh.namespaceSelector.matchLabels)} />
              </section>
            {/if}
            {#if wh.objectSelector?.matchLabels && Object.keys(wh.objectSelector.matchLabels).length > 0}
              <section>
                <SectionHeader>Object Selector</SectionHeader>
                <KeyValueBadge entries={Object.entries(wh.objectSelector.matchLabels)} />
              </section>
            {/if}

            <!-- Side effects + timeout -->
            <section>
              <div class="flex items-center gap-6 text-xs">
                {#if wh.sideEffects}
                  <div class="flex items-center gap-2">
                    <span class="text-muted">Side Effects</span>
                    <span class="px-1.5 py-0.5 rounded font-medium bg-surface border border-border">{wh.sideEffects}</span>
                  </div>
                {/if}
                {#if wh.timeoutSeconds != null}
                  <div class="flex items-center gap-2">
                    <span class="text-muted">Timeout</span>
                    <span class="font-mono">{wh.timeoutSeconds}s</span>
                  </div>
                {/if}
              </div>
            </section>

            <!-- Match conditions (CEL) -->
            {#if wh.matchConditions?.length}
              <section>
                <SectionHeader>Match Conditions</SectionHeader>
                <div class="space-y-1">
                  {#each wh.matchConditions as mc}
                    <div class="text-xs font-mono bg-surface border border-border rounded px-2 py-1">
                      <span class="text-muted">{mc.name}: </span>{mc.expression}
                    </div>
                  {/each}
                </div>
              </section>
            {/if}
          </div>
        {/if}
      </div>
    {/each}
  {/if}
</div>
