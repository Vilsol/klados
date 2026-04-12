<script lang="ts">
  import {onMount} from "svelte";
  import * as ConfigService from "../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js";

  interface ColumnPrefsEntry {
    order: string[];
    sort?: {column: string; direction: string} | null;
    columns: Record<string, {width?: number}>;
  }

  let columnPrefs = $state<Record<string, ColumnPrefsEntry>>({});

  onMount(() => {
    (async () => {
      const config = await ConfigService.GetConfig();
      if (config && (config as any).columnPrefs) {
        columnPrefs = (config as any).columnPrefs;
      }
    })();
  });

  async function resetGVR(gvr: string) {
    await ConfigService.DeleteColumnPrefs(gvr);
    const {[gvr]: _, ...rest} = columnPrefs;
    columnPrefs = rest;
  }

  let gvrKeys = $derived(Object.keys(columnPrefs).sort());
</script>

<div class="max-w-3xl space-y-6">
  <h2 class="text-base font-medium text-fg">Column Preferences</h2>
  <p class="text-sm text-muted-foreground">Column order and widths are saved automatically as you adjust them in resource lists.</p>

  {#if gvrKeys.length === 0}
    <p class="text-sm text-muted-foreground">No column preferences saved yet. Adjust columns in resource lists to save preferences.</p>
  {:else}
    {#each gvrKeys as gvr}
      {@const prefs = columnPrefs[gvr]}
      <div class="border border-border rounded">
        <div class="flex items-center justify-between px-4 py-2 bg-surface border-b border-border">
          <span class="text-sm font-mono text-fg">{gvr}</span>
          <button class="text-xs text-destructive hover:underline" onclick={() => resetGVR(gvr)}>Reset to default</button>
        </div>
        <div class="px-4 py-3 space-y-2">
          {#if prefs?.order?.length > 0}
            <div>
              <span class="text-xs text-muted-foreground">Column order:</span>
              <div class="flex flex-wrap gap-1 mt-1">
                {#each prefs.order as col}
                  <span class="px-2 py-0.5 rounded bg-surface border border-border text-xs text-fg font-mono">{col}</span>
                {/each}
              </div>
            </div>
          {/if}
          {#if prefs?.sort}
            <div class="text-xs text-muted-foreground">
              Sorted by: <span class="text-fg font-mono">{prefs.sort.column}</span>
              ({prefs.sort.direction})
            </div>
          {/if}
        </div>
      </div>
    {/each}
  {/if}
</div>
