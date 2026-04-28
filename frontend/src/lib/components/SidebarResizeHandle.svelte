<script lang="ts">
  import {onDestroy} from "svelte";
  import {SIDEBAR_MIN_WIDTH, SIDEBAR_MAX_WIDTH, sessionStore} from "$lib/stores/session.svelte";

  let dragging = $state(false);
  let dragStartX = 0;
  let dragStartWidth = 0;

  function onPointerDown(e: PointerEvent) {
    if (e.button !== 0) return;
    dragging = true;
    dragStartX = e.clientX;
    dragStartWidth = sessionStore.sidebarWidth;
    document.body.dataset.resizing = "true";
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    e.preventDefault();
  }

  function onPointerMove(e: PointerEvent) {
    if (!dragging) return;
    const delta = e.clientX - dragStartX;
    sessionStore.setSidebarWidth(dragStartWidth + delta);
  }

  function endDrag() {
    if (!dragging) return;
    dragging = false;
    delete document.body.dataset.resizing;
  }

  function onDoubleClick() {
    sessionStore.resetSidebarWidth();
  }

  onDestroy(() => {
    if (dragging) {
      delete document.body.dataset.resizing;
    }
  });
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div
  class="absolute top-0 right-0 h-full w-1 cursor-col-resize transition-colors group z-10
    {dragging ? 'bg-accent/40' : 'hover:bg-accent/30'}"
  role="separator"
  aria-orientation="vertical"
  aria-valuenow={sessionStore.sidebarWidth}
  aria-valuemin={SIDEBAR_MIN_WIDTH}
  aria-valuemax={SIDEBAR_MAX_WIDTH}
  onpointerdown={onPointerDown}
  onpointermove={onPointerMove}
  onpointerup={endDrag}
  onpointercancel={endDrag}
  onlostpointercapture={endDrag}
  ondblclick={onDoubleClick}
  title="Drag to resize · double-click to reset"
>
  <div class="w-px h-full mx-auto bg-border group-hover:bg-accent/60 {dragging ? 'bg-accent/60' : ''}"></div>
</div>
