<script lang="ts">
  import {SectionHeader, StatusBadge} from "@klados/ui";
  import type {KubernetesResource} from "$lib/types";

  const reDecimalOrInt = /^\d+(\.\d+)?$/;
  const reMilliCores = /^(\d+(?:\.\d+)?)m$/;
  const reBinarySuffix = /^(\d+(?:\.\d+)?)(Ki|Mi|Gi|Ti)$/;

  let {obj}: {obj: Record<string, KubernetesResource>} = $props();

  const hard = $derived<Record<string, string>>(obj.status?.hard ?? {});
  const used = $derived<Record<string, string>>(obj.status?.used ?? {});
  const scopes = $derived<string[]>(obj.spec?.scopes ?? []);
  const scopeSelector = $derived(obj.spec?.scopeSelector);

  function parseQuantity(q: string): number | null {
    const s = String(q).trim();
    if (reDecimalOrInt.test(s)) {
      return Number.parseFloat(s);
    }
    const mMatch = s.match(reMilliCores);
    if (mMatch) {
      return Number.parseFloat(mMatch[1]) / 1000;
    }
    const binMatch = s.match(reBinarySuffix);
    if (binMatch) {
      const mult: Record<string, number> = {Ki: 1024, Mi: 1024 ** 2, Gi: 1024 ** 3, Ti: 1024 ** 4};
      return Number.parseFloat(binMatch[1]) * mult[binMatch[2]];
    }
    return null;
  }

  function barColor(pct: number): string {
    if (pct >= 90) {
      return "bg-red-500";
    }
    if (pct >= 70) {
      return "bg-yellow-500";
    }
    return "bg-green-500";
  }
</script>

<div class="flex flex-col gap-6 p-4 overflow-auto">
  {#if scopes.length > 0 || scopeSelector}
    <section>
      <SectionHeader>Scopes</SectionHeader>
      <div class="flex flex-wrap gap-1.5">
        {#each scopes as scope}
          <span class="text-xs px-2 py-0.5 rounded-full bg-surface border border-border font-mono">{scope}</span>
        {/each}
        {#if scopeSelector?.matchExpressions}
          {#each scopeSelector.matchExpressions as expr}
            <span class="text-xs px-2 py-0.5 rounded-full bg-surface border border-border font-mono"
              >{expr.scopeName ?? expr.operator}</span
            >
          {/each}
        {/if}
      </div>
    </section>
  {/if}

  {#if Object.keys(hard).length > 0}
    <section>
      <SectionHeader>Usage</SectionHeader>
      <table class="w-full text-xs">
        <thead class="bg-surface">
          <tr>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Resource</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Used</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Hard</th>
            <th class="px-2 py-1.5 text-muted font-medium w-36">Usage</th>
          </tr>
        </thead>
        <tbody>
          {#each Object.entries(hard) as [ resource, hardVal ]}
            {@const usedVal = used[resource]}
            {@const usedNum = parseQuantity(usedVal ?? '0')}
            {@const hardNum = parseQuantity(hardVal)}
            {@const pct = (usedNum !== null && hardNum !== null && hardNum > 0) ? Math.min(100, (usedNum / hardNum) * 100) : null}
            <tr class="border-t border-border">
              <td class="px-2 py-1.5 font-mono">{resource}</td>
              <td class="px-2 py-1.5 font-mono">{usedVal ?? '0'}</td>
              <td class="px-2 py-1.5 font-mono">{hardVal}</td>
              <td class="px-2 py-1.5">
                <div class="flex items-center gap-2">
                  <div class="flex-1 h-1.5 bg-border rounded-full overflow-hidden">
                    {#if pct !== null}
                      <div class="{barColor(pct)} h-full rounded-full transition-all" style="width: {pct}%"></div>
                    {:else}
                      <div class="bg-muted/40 h-full rounded-full" style="width: 0%"></div>
                    {/if}
                  </div>
                  <span class="text-muted w-10 text-right">{pct !== null ? `${Math.round(pct)}%` : '—'}</span>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </section>
  {/if}
</div>
