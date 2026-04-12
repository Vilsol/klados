<script lang="ts">
  import {SectionHeader} from "@klados/ui";
  import type {KubernetesResource} from "$lib/types";

  let {obj}: {obj: Record<string, KubernetesResource>} = $props();

  const limits = $derived<KubernetesResource[]>(obj.spec?.limits ?? []);

  function getResources(entry: KubernetesResource): string[] {
    const all = new Set<string>();
    for (const field of ["default", "defaultRequest", "min", "max", "maxLimitRequestRatio"]) {
      if (entry[field]) {
        for (const k of Object.keys(entry[field])) {
          all.add(k);
        }
      }
    }
    if (entry.type === "PersistentVolumeClaim") {
      return [...all].filter((r) => r === "storage");
    }
    return [...all];
  }
</script>

<div class="flex flex-col gap-6 p-4 overflow-auto">
  {#each limits as entry, i}
    <section>
      <SectionHeader>
        <span>Limit {i + 1}</span>
        <span class="ml-2 text-xs px-2 py-0.5 rounded-full bg-surface border border-border font-mono normal-case tracking-normal"
          >{entry.type ?? 'Unknown'}</span
        >
      </SectionHeader>
      <table class="w-full text-xs">
        <thead class="bg-surface">
          <tr>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Resource</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Default</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Default Request</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Min</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Max</th>
            <th class="text-left px-2 py-1.5 text-muted font-medium">Max L/R Ratio</th>
          </tr>
        </thead>
        <tbody>
          {#each getResources(entry) as resource}
            <tr class="border-t border-border">
              <td class="px-2 py-1.5 font-mono">{resource}</td>
              <td class="px-2 py-1.5 font-mono">{entry.default?.[resource] ?? '—'}</td>
              <td class="px-2 py-1.5 font-mono">{entry.defaultRequest?.[resource] ?? '—'}</td>
              <td class="px-2 py-1.5 font-mono">{entry.min?.[resource] ?? '—'}</td>
              <td class="px-2 py-1.5 font-mono">{entry.max?.[resource] ?? '—'}</td>
              <td class="px-2 py-1.5 font-mono">{entry.maxLimitRequestRatio?.[resource] ?? '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </section>
  {/each}
</div>
