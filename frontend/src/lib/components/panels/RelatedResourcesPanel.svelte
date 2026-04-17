<script lang="ts">
  import {resourceCache} from "$lib/stores/resourceCache.svelte";
  import {push} from "svelte-spa-router";

  interface Props {
    contextName: string;
    obj: Record<string, unknown>;
    onopenresource?: (gvr: string, namespace: string, name: string) => void;
  }
  let {contextName, obj, onopenresource}: Props = $props();

  let uid = $derived(((obj as any)?.metadata?.uid as string) ?? "");
  let groups = $derived(uid ? resourceCache.findByOwnerUID(contextName, uid) : []);

  function nav(gvr: string, item: Record<string, unknown>) {
    const ns = (item as any)?.metadata?.namespace ?? "";
    const name = (item as any)?.metadata?.name ?? "";
    if (onopenresource) {
      onopenresource(gvr, ns, name);
    } else {
      push(`/c/${encodeURIComponent(contextName)}/${gvr}/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`);
    }
  }
</script>

{#if groups.length === 0}
  <div class="p-4 text-muted text-sm">
    No related resources found in active watches.
    <div class="text-xs mt-1">Klados only shows related resources that are currently being watched.</div>
  </div>
{:else}
  <div class="p-4 space-y-4">
    {#each groups as g (g.gvr)}
      <section>
        <h3 class="text-xs font-semibold uppercase text-muted mb-2">
          {g.gvr} ({g.items.length})
        </h3>
        <ul class="space-y-1">
          {#each g.items as it}
            {@const name = (it as any)?.metadata?.name ?? ""}
            {@const ns = (it as any)?.metadata?.namespace ?? ""}
            <li>
              <button class="text-accent underline hover:text-accent/80 text-sm" onclick={() => nav(g.gvr, it)}>
                {ns ? `${ns}/` : ""}{name}
              </button>
            </li>
          {/each}
        </ul>
      </section>
    {/each}
  </div>
{/if}
