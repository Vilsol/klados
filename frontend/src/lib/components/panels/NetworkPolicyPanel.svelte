<script lang="ts">
  import {SectionHeader, KeyValueBadge, EmptyState} from "@klados/ui";

  let {obj}: {obj: Record<string, any>} = $props();

  const podSelector = $derived<Record<string, string>>(obj.spec?.podSelector?.matchLabels ?? {});
  const policyTypes = $derived<string[]>(obj.spec?.policyTypes ?? []);
  const ingressRules = $derived<any[] | undefined>(obj.spec?.ingress);
  const egressRules = $derived<any[] | undefined>(obj.spec?.egress);

  function formatPorts(ports: any[]): string {
    if (!ports || ports.length === 0) {
      return "All ports";
    }
    return ports.map((p: any) => `${p.port ?? "*"}/${p.protocol ?? "TCP"}`).join(", ");
  }
</script>

<div class="p-4 space-y-6">
  <!-- Applies to -->
  <section>
    <SectionHeader>Applies To</SectionHeader>
    {#if Object.keys(podSelector).length === 0}
      <span
        class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-500/20 text-yellow-400 border border-yellow-500/30"
      >
        All pods in namespace
      </span>
    {:else}
      <KeyValueBadge entries={Object.entries(podSelector)} />
    {/if}
  </section>

  <!-- Policy types -->
  {#if policyTypes.length > 0}
    <section>
      <SectionHeader>Policy Types</SectionHeader>
      <span class="text-xs font-medium">{policyTypes.join(' + ')}</span>
    </section>
  {/if}

  <!-- Ingress rules -->
  {#if policyTypes.includes('Ingress')}
    <section>
      <SectionHeader>Ingress Rules</SectionHeader>
      {#if ingressRules === undefined}
        <!-- nil ingress key — section intentionally empty, allow all -->
        <EmptyState message="No ingress restrictions (allow all)" />
      {:else if ingressRules.length === 0}
        <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-500/20 text-red-400 border border-red-500/30">
          Deny all ingress
        </span>
      {:else}
        <div class="space-y-2">
          {#each ingressRules as rule}
            <div class="bg-surface border border-border rounded-lg p-3 flex gap-4 items-start text-xs">
              <div class="flex-1 min-w-0">
                <div class="text-muted font-medium mb-1">FROM</div>
                {#if !rule.from || rule.from.length === 0}
                  <span class="text-muted italic">Any source</span>
                {:else}
                  <div class="space-y-1.5">
                    {#each rule.from as src}
                      <div class="space-y-1">
                        {#if src.podSelector}
                          <div class="flex items-center gap-1 flex-wrap">
                            <span class="text-muted">Pod:</span>
                            {#if Object.keys(src.podSelector.matchLabels ?? {}).length === 0}
                              <span class="text-muted italic">any</span>
                            {:else}
                              <KeyValueBadge entries={Object.entries(src.podSelector.matchLabels ?? {})} />
                            {/if}
                          </div>
                        {/if}
                        {#if src.namespaceSelector}
                          <div class="flex items-center gap-1 flex-wrap">
                            <span class="text-muted">NS:</span>
                            {#if Object.keys(src.namespaceSelector.matchLabels ?? {}).length === 0}
                              <span class="text-muted italic">any</span>
                            {:else}
                              <KeyValueBadge entries={Object.entries(src.namespaceSelector.matchLabels ?? {})} />
                            {/if}
                          </div>
                        {/if}
                        {#if src.ipBlock}
                          <div>
                            <span class="font-mono">{src.ipBlock.cidr}</span>
                            {#if src.ipBlock.except?.length}
                              <span class="text-muted ml-1">except {src.ipBlock.except.join(', ')}</span>
                            {/if}
                          </div>
                        {/if}
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>
              <div class="text-muted shrink-0 self-center">→</div>
              <div class="flex-1 min-w-0">
                <div class="text-muted font-medium mb-1">PORTS</div>
                <span class="font-mono">{formatPorts(rule.ports)}</span>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </section>
  {/if}

  <!-- Egress rules -->
  {#if policyTypes.includes('Egress')}
    <section>
      <SectionHeader>Egress Rules</SectionHeader>
      {#if egressRules === undefined}
        <EmptyState message="No egress restrictions (allow all)" />
      {:else if egressRules.length === 0}
        <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-500/20 text-red-400 border border-red-500/30">
          Deny all egress
        </span>
      {:else}
        <div class="space-y-2">
          {#each egressRules as rule}
            <div class="bg-surface border border-border rounded-lg p-3 flex gap-4 items-start text-xs">
              <div class="flex-1 min-w-0">
                <div class="text-muted font-medium mb-1">TO</div>
                {#if !rule.to || rule.to.length === 0}
                  <span class="text-muted italic">Any destination</span>
                {:else}
                  <div class="space-y-1.5">
                    {#each rule.to as dst}
                      <div class="space-y-1">
                        {#if dst.podSelector}
                          <div class="flex items-center gap-1 flex-wrap">
                            <span class="text-muted">Pod:</span>
                            {#if Object.keys(dst.podSelector.matchLabels ?? {}).length === 0}
                              <span class="text-muted italic">any</span>
                            {:else}
                              <KeyValueBadge entries={Object.entries(dst.podSelector.matchLabels ?? {})} />
                            {/if}
                          </div>
                        {/if}
                        {#if dst.namespaceSelector}
                          <div class="flex items-center gap-1 flex-wrap">
                            <span class="text-muted">NS:</span>
                            {#if Object.keys(dst.namespaceSelector.matchLabels ?? {}).length === 0}
                              <span class="text-muted italic">any</span>
                            {:else}
                              <KeyValueBadge entries={Object.entries(dst.namespaceSelector.matchLabels ?? {})} />
                            {/if}
                          </div>
                        {/if}
                        {#if dst.ipBlock}
                          <div>
                            <span class="font-mono">{dst.ipBlock.cidr}</span>
                            {#if dst.ipBlock.except?.length}
                              <span class="text-muted ml-1">except {dst.ipBlock.except.join(', ')}</span>
                            {/if}
                          </div>
                        {/if}
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>
              <div class="text-muted shrink-0 self-center">→</div>
              <div class="flex-1 min-w-0">
                <div class="text-muted font-medium mb-1">PORTS</div>
                <span class="font-mono">{formatPorts(rule.ports)}</span>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </section>
  {/if}

  <!-- Implicit deny footer -->
  <p class="text-xs text-muted italic">All traffic not explicitly allowed is denied.</p>
</div>
