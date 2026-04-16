<script lang="ts">
  import { findWarnings, getConditions } from "../kubernetes/conditions";

  interface Props { obj: Record<string, unknown>; }
  let { obj }: Props = $props();

  let warnings = $derived(findWarnings(getConditions(obj)));
</script>

{#if warnings.length > 0}
  <div class="m-4 rounded border border-amber-500/40 bg-amber-500/5 p-3 text-sm">
    <div class="font-semibold text-amber-500 mb-1">
      {warnings.length === 1 ? "Validation warning" : `${warnings.length} validation warnings`}
    </div>
    <ul class="space-y-1">
      {#each warnings as w}
        <li>
          <span class="font-mono text-xs text-amber-500 mr-2">{w.type}</span>
          {#if w.reason}<span class="text-muted mr-2">{w.reason}:</span>{/if}
          <span>{w.message}</span>
        </li>
      {/each}
    </ul>
  </div>
{/if}
