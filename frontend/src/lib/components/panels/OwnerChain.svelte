<script lang="ts">
  import {getOwnerReferences, gvrFromAPIVersion} from "$lib/kubernetes/owners";
  import {GetResource} from "../../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js";
  import {push} from "svelte-spa-router";

  interface Props {
    contextName: string;
    obj: Record<string, unknown>;
  }
  let {contextName, obj}: Props = $props();

  interface ChainNode {
    gvr: string;
    namespace: string;
    name: string;
    kind: string;
  }

  let chain = $state<ChainNode[]>([]);

  $effect(() => {
    const currentObj = obj;
    const ctx = contextName;
    (async () => {
      const result: ChainNode[] = [];
      let current: Record<string, unknown> | null = currentObj;
      for (let depth = 0; depth < 5; depth++) {
        const refs = getOwnerReferences(current);
        if (refs.length === 0) break;
        const controller = refs.find((r) => r.controller) ?? refs[0];
        const parentGvr = gvrFromAPIVersion(controller.apiVersion, controller.kind);
        const parentNs = (current as any)?.metadata?.namespace ?? "";
        result.push({gvr: parentGvr, namespace: parentNs, name: controller.name, kind: controller.kind});
        try {
          current = await GetResource(ctx, parentGvr, parentNs, controller.name) as Record<string, unknown>;
        } catch {
          break;
        }
      }
      chain = result;
    })();
  });

  function navigate(n: ChainNode) {
    push(`/c/${encodeURIComponent(contextName)}/${n.gvr}/${encodeURIComponent(n.namespace)}/${encodeURIComponent(n.name)}`);
  }
</script>

{#if chain.length > 0}
  <div class="text-xs text-muted flex flex-wrap items-center gap-1">
    <span>Owned by:</span>
    {#each chain as n, i}
      {#if i > 0}<span>→</span>{/if}
      <button class="underline text-accent hover:text-accent/80" onclick={() => navigate(n)}>
        {n.kind}/{n.name}
      </button>
    {/each}
  </div>
{/if}
