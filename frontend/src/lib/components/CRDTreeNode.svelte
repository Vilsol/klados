<script lang="ts">
  import {ChevronRight} from "lucide-svelte";
  import type {CRDTreeNode as CRDTreeNodeType} from "$lib/utils/crdTree";
  import CRDTreeNode from "./CRDTreeNode.svelte";

  interface Props {
    node: CRDTreeNodeType;
    expanded: Set<string>;
    onToggle: (fullSuffix: string) => void;
    ctxName: string;
    activePath?: string;
  }

  const {node, expanded, onToggle, ctxName, activePath = ""}: Props = $props();
</script>

<div>
  <button
    type="button"
    onclick={() => onToggle(node.fullSuffix)}
    class="w-full flex items-center gap-1 px-3 py-1 text-sm hover:bg-surface-hover transition-colors rounded-sm text-left"
  >
    <ChevronRight size={12} class="transition-transform shrink-0 {expanded.has(node.fullSuffix) ? 'rotate-90' : ''}" />
    {node.label}
  </button>

  {#if expanded.has(node.fullSuffix)}
    <div class="ml-4">
      {#each node.children as child}
        <CRDTreeNode node={child} {expanded} {onToggle} {ctxName} {activePath} />
      {/each}
      {#each node.directGvrs as { gvr, kind }}
        <a
          href="#/c/{ctxName}/{gvr}"
          class="block px-3 py-1 text-sm transition-colors rounded-sm {activePath === `/c/${ctxName}/${gvr}` ? 'bg-surface-hover text-accent font-medium' : 'hover:bg-surface-hover'}"
        >
          {kind}
        </a>
      {/each}
    </div>
  {/if}
</div>
