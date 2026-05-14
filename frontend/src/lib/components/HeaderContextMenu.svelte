<script lang="ts">
  let {
    x,
    y,
    columnName,
    isPinned = false,
    canHide = true,
    onSort,
    onAutoFit,
    onTogglePin,
    onHide,
    onClose,
  }: {
    x: number;
    y: number;
    columnName: string;
    isPinned?: boolean;
    canHide?: boolean;
    onSort: (direction: "asc" | "desc") => void;
    onAutoFit: () => void;
    onTogglePin: () => void;
    onHide: () => void;
    onClose: () => void;
  } = $props();

  let menuEl = $state<HTMLDivElement | null>(null);

  $effect(() => {
    if (!menuEl) return;
    const rect = menuEl.getBoundingClientRect();
    const maxX = window.innerWidth - rect.width - 8;
    const maxY = window.innerHeight - rect.height - 8;
    if (x > maxX) menuEl.style.left = `${Math.max(0, maxX)}px`;
    if (y > maxY) menuEl.style.top = `${Math.max(0, maxY)}px`;
  });

  function clickAndClose(fn: () => void) {
    fn();
    onClose();
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  bind:this={menuEl}
  class="fixed z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-44"
  style="left:{x}px; top:{y}px"
  onclick={(e) => e.stopPropagation()}
  onkeydown={(e) => e.stopPropagation()}
>
  <div class="px-3 py-1 text-xs font-semibold text-muted truncate">{columnName}</div>
  <div class="border-t border-border my-1"></div>
  <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover" onclick={() => clickAndClose(() => onSort("asc"))}>Sort ascending</button>
  <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover" onclick={() => clickAndClose(() => onSort("desc"))}>Sort descending</button>
  <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover" onclick={() => clickAndClose(onAutoFit)}>Auto-fit width</button>
  <div class="border-t border-border my-1"></div>
  <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover" onclick={() => clickAndClose(onTogglePin)}>
    {isPinned ? "Unpin" : "Pin to left"}
  </button>
  {#if canHide}
    <button type="button" class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover" onclick={() => clickAndClose(onHide)}>Hide column</button>
  {/if}
</div>
