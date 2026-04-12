<script lang="ts">
  import {push} from "svelte-spa-router";
  import {SectionHeader, StatusBadge} from "@klados/ui";
  import type {KubernetesResource} from "$lib/types";

  const {obj, ctxName}: {obj: KubernetesResource; ctxName: string} = $props();

  const storageGVR = $derived(obj.status?.storageGVR as string | undefined);
  const versions = $derived((obj.spec?.versions ?? []) as KubernetesResource[]);
</script>

<div class="p-4 flex flex-col gap-6">
  {#if storageGVR}
    <div>
      <button
        type="button"
        class="px-3 py-1.5 text-sm rounded border bg-accent/20 text-accent border-accent/30 hover:bg-accent/30 transition-colors"
        onclick={() => push(`/c/${ctxName}/${storageGVR}`)}
      >
        View Instances →
      </button>
    </div>
  {/if}

  <div>
    <SectionHeader>Versions</SectionHeader>
    <table class="w-full text-sm">
      <thead>
        <tr class="border-b border-border">
          <th class="text-left py-1.5 pr-4 font-medium text-muted">Name</th>
          <th class="text-left py-1.5 pr-4 font-medium text-muted">Served</th>
          <th class="text-left py-1.5 pr-4 font-medium text-muted">Storage</th>
          <th class="text-left py-1.5 font-medium text-muted">Deprecated</th>
        </tr>
      </thead>
      <tbody>
        {#each versions as v}
          <tr class="border-b border-border/50">
            <td class="py-1.5 pr-4 font-mono">{v.name}</td>
            <td class="py-1.5 pr-4">
              <StatusBadge status={v.served !== false} mode="pill">{v.served === false ? 'False' : 'True'}</StatusBadge>
            </td>
            <td class="py-1.5 pr-4">
              <StatusBadge status={v.storage === true} mode="pill">{v.storage === true ? 'True' : 'False'}</StatusBadge>
            </td>
            <td class="py-1.5">
              {#if v.deprecated}
                <StatusBadge status="False" mode="pill">True</StatusBadge>
              {:else}
                <span class="text-muted">—</span>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
