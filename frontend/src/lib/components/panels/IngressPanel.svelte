<script lang="ts">
  import {Browser} from "@wailsio/runtime";
  import {SectionHeader, EmptyState} from "@klados/ui";

  let {obj}: {obj: Record<string, any>} = $props();

  const rules = $derived<any[]>(obj.spec?.rules ?? []);
  const tls = $derived<any[]>(obj.spec?.tls ?? []);

  function openURL(host: string) {
    Browser.OpenURL(`https://${host}`);
  }
</script>

<div class="flex flex-col gap-4 p-4 overflow-auto h-full">
  <!-- Rules -->
  <section>
    <SectionHeader>Rules</SectionHeader>
    {#if rules.length === 0}
      <EmptyState message="No rules" size="sm" />
    {:else}
      <div class="flex flex-col gap-3">
        {#each rules as rule}
          {@const host = rule.host ?? '*'}
          <div class="bg-surface border border-border rounded-lg overflow-hidden">
            <div class="flex items-center justify-between px-3 py-2 border-b border-border bg-surface-hover">
              <span class="text-sm font-medium font-mono">{host}</span>
              {#if rule.host}
                <button onclick={() => openURL(rule.host)} class="text-xs text-accent hover:underline">Open ↗</button>
              {/if}
            </div>
            {#if rule.http?.paths?.length > 0}
              <table class="w-full text-sm">
                <thead>
                  <tr class="text-left text-muted text-xs border-b border-border">
                    <th class="px-3 py-1 font-medium">Path</th>
                    <th class="px-3 py-1 font-medium">Type</th>
                    <th class="px-3 py-1 font-medium">Backend</th>
                  </tr>
                </thead>
                <tbody>
                  {#each rule.http.paths as p}
                    <tr class="border-b border-border/50 last:border-0">
                      <td class="px-3 py-1.5 font-mono text-xs">{p.path ?? '/'}</td>
                      <td class="px-3 py-1.5 text-muted text-xs">{p.pathType ?? '—'}</td>
                      <td class="px-3 py-1.5 text-xs">
                        {#if p.backend?.service}
                          {p.backend.service.name}:{p.backend.service.port?.number ?? p.backend.service.port?.name ?? '?'}
                        {:else}
                          —
                        {/if}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </section>

  <!-- TLS -->
  {#if tls.length > 0}
    <section>
      <SectionHeader>TLS</SectionHeader>
      <div class="flex flex-col gap-2">
        {#each tls as entry}
          <div class="bg-surface border border-border rounded p-3 text-sm">
            <div class="flex gap-2 text-muted text-xs mb-1">
              <span class="font-medium text-fg">Secret:</span>
              <span class="font-mono">{entry.secretName ?? '—'}</span>
            </div>
            {#if entry.hosts?.length > 0}
              <div class="flex flex-wrap gap-1">
                {#each entry.hosts as h}
                  <span class="text-xs bg-surface-hover border border-border rounded px-2 py-0.5 font-mono">{h}</span>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </section>
  {/if}
</div>
