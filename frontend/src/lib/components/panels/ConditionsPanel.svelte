<script lang="ts">
  import { getConditions } from "../../kubernetes/conditions";

  interface Props { obj: Record<string, unknown>; }
  let { obj }: Props = $props();

  let conditions = $derived(getConditions(obj));

  function badgeClass(status: string): string {
    switch (status) {
      case "True": return "bg-emerald-500/15 text-emerald-500";
      case "False": return "bg-destructive/15 text-destructive";
      default: return "bg-muted/30 text-muted";
    }
  }
</script>

{#if conditions.length === 0}
  <div class="p-4 text-muted text-sm">No conditions reported on this resource.</div>
{:else}
  <table class="w-full text-sm">
    <thead class="text-muted text-left">
      <tr>
        <th class="p-2 w-40">Type</th>
        <th class="p-2 w-28">Status</th>
        <th class="p-2 w-40">Reason</th>
        <th class="p-2">Message</th>
        <th class="p-2 w-48">Last Transition</th>
      </tr>
    </thead>
    <tbody>
      {#each conditions as c (c.type)}
        <tr class="border-t border-border">
          <td class="p-2 font-mono text-xs">{c.type}</td>
          <td class="p-2">
            <span class="px-2 py-0.5 rounded text-xs {badgeClass(c.status)}">{c.status}</span>
          </td>
          <td class="p-2">{c.reason ?? ""}</td>
          <td class="p-2">{c.message ?? ""}</td>
          <td class="p-2 text-muted">{c.lastTransitionTime ?? ""}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
