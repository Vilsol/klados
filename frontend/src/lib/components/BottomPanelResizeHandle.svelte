<script lang="ts">
  import {onMount, onDestroy} from "svelte";
  import {bottomPanelStore} from "$lib/stores/bottom-panel.svelte";

  let dragging = $state(false);
  let dragStartY = 0;
  let dragStartHeight = 0;

  function onResizeStart(e: MouseEvent) {
    dragging = true;
    dragStartY = e.clientY;
    dragStartHeight = bottomPanelStore.height;
    document.body.style.cursor = "ns-resize";
    document.body.style.userSelect = "none";
    e.preventDefault();
  }

  function onResizeMove(e: MouseEvent) {
    if (!dragging) {
      return;
    }
    const delta = dragStartY - e.clientY;
    bottomPanelStore.setHeight(dragStartHeight + delta);
  }

  function onResizeEnd() {
    if (!dragging) {
      return;
    }
    dragging = false;
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
  }

  onMount(() => {
    document.addEventListener("mousemove", onResizeMove);
    document.addEventListener("mouseup", onResizeEnd);
  });

  onDestroy(() => {
    document.removeEventListener("mousemove", onResizeMove);
    document.removeEventListener("mouseup", onResizeEnd);
  });
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
  class="h-1 shrink-0 cursor-ns-resize transition-colors group
    {dragging ? 'bg-accent/40' : 'hover:bg-accent/30'}"
  role="separator"
  aria-orientation="horizontal"
  onmousedown={onResizeStart}
>
  <div class="h-px mx-auto bg-border group-hover:bg-accent/60 {dragging ? 'bg-accent/60' : ''}"></div>
</div>
