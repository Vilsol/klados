<script lang="ts" module>
  export type DataTableColumn = {
    name: string;
    width?: number;
    align?: "left" | "right" | "center";
    hidden?: boolean;
  };
</script>

<script lang="ts" generics="T">
  import {createVirtualizer} from "@tanstack/svelte-virtual";
  import {ArrowUpDown, ArrowUp, ArrowDown} from "lucide-svelte";
  import {untrack, type Snippet} from "svelte";
  import {dndzone, type DndEvent} from "svelte-dnd-action";
  import HeaderContextMenu from "./HeaderContextMenu.svelte";

  let {
    items,
    visibleColumns,
    pinnedNames = [],
    sortState = null,
    compact = false,
    loading = false,
    error = null,
    emptyMessage = "No items found",
    prefixGridCols = [],
    suffixGridCols = [],
    toolbar,
    cell,
    emptyAction,
    headerPrefix,
    headerSuffix,
    rowPrefix,
    rowSuffix,
    selectedRow,
    onsort,
    onresize,
    onreorder,
    onTogglePin,
    onHideColumn,
    onrowclick,
    oncontextmenu,
    scrollContainer = $bindable<HTMLDivElement | undefined>(undefined),
  }: {
    items: T[];
    visibleColumns: DataTableColumn[];
    pinnedNames?: string[];
    sortState?: {column: string; direction: "asc" | "desc"} | null;
    compact?: boolean;
    loading?: boolean;
    error?: string | null;
    emptyMessage?: string;
    prefixGridCols?: string[];
    suffixGridCols?: string[];
    toolbar?: Snippet;
    cell: Snippet<[{item: T; column: DataTableColumn}]>;
    emptyAction?: Snippet;
    headerPrefix?: Snippet;
    headerSuffix?: Snippet;
    rowPrefix?: Snippet<[{item: T}]>;
    rowSuffix?: Snippet<[{item: T}]>;
    selectedRow?: (item: T) => boolean;
    onsort?: (column: string, direction: "asc" | "desc") => void;
    onresize?: (column: string, width: number) => void;
    onreorder?: (names: string[]) => void;
    onTogglePin?: (name: string) => void;
    onHideColumn?: (name: string) => void;
    onrowclick?: (item: T) => void;
    oncontextmenu?: (event: MouseEvent, item: T) => void;
    scrollContainer?: HTMLDivElement;
  } = $props();

  let resizing = $state<{name: string; startX: number; startWidth: number} | null>(null);
  let headerCtxMenu = $state<{x: number; y: number; columnName: string} | null>(null);

  $effect(() => {
    if (!headerCtxMenu) return;
    const close = () => {
      headerCtxMenu = null;
    };
    const t = setTimeout(() => window.addEventListener("click", close, {once: true}), 0);
    return () => {
      clearTimeout(t);
      window.removeEventListener("click", close);
    };
  });

  const rowHeight = $derived(compact ? 28 : 36);

  const pinnedSet = $derived(new Set(pinnedNames));
  const pinnedColumns = $derived(visibleColumns.filter((c) => pinnedSet.has(c.name)));
  const mainColumns = $derived(visibleColumns.filter((c) => !pinnedSet.has(c.name)));

  // svelte-dnd-action requires items with an `id` field; the library mutates the items array
  // via consider/finalize events, so we maintain a local $state mirror synced from the derived
  // `mainColumns`. This keeps the upstream store as source of truth between drags.
  type DnDColumn = DataTableColumn & {id: string};
  // Initialize synchronously from props (not from the $derived mainColumns) so the first render
  // has populated body cells before the $effect below has fired.
  const _initialPinnedSet = new Set(pinnedNames);
  let liveMainColumns = $state<DnDColumn[]>(
    visibleColumns.filter((c) => !_initialPinnedSet.has(c.name)).map((c) => ({...c, id: c.name})),
  );
  $effect(() => {
    liveMainColumns = mainColumns.map((c) => ({...c, id: c.name}));
  });

  function handleDndConsider(e: CustomEvent<DndEvent<DnDColumn>>) {
    liveMainColumns = e.detail.items;
  }

  function handleDndFinalize(e: CustomEvent<DndEvent<DnDColumn>>) {
    liveMainColumns = e.detail.items;
    onreorder?.(e.detail.items.map((c) => c.name));
  }

  const virtualizer = createVirtualizer({
    count: 0,
    getScrollElement: () => scrollContainer ?? null,
    estimateSize: () => rowHeight,
    overscan: 10,
  });

  $effect(() => {
    const count = items.length;
    const rh = rowHeight;
    void scrollContainer;
    untrack(() => {
      $virtualizer.setOptions({
        count,
        getScrollElement: () => scrollContainer ?? null,
        estimateSize: () => rh,
        overscan: 10,
      });
    });
  });

  const pinnedGridCols = $derived.by(() => {
    const parts: string[] = [...prefixGridCols];
    for (const c of pinnedColumns) {
      parts.push(c.width ? `${c.width}px` : "minmax(20px, max-content)");
    }
    return parts.join(" ");
  });

  const mainGridCols = $derived.by(() => {
    const parts: string[] = [];
    for (const c of liveMainColumns) {
      parts.push(c.width ? `${c.width}px` : "minmax(20px, 1fr)");
    }
    parts.push(...suffixGridCols);
    return parts.join(" ");
  });

  function alignClass(col: DataTableColumn): string {
    const align = col.align ?? "left";
    if (align === "right") return "text-right";
    if (align === "center") return "text-center";
    return "text-left";
  }

  function toggleSort(name: string) {
    if (!onsort) return;
    if (sortState?.column === name) {
      onsort(name, sortState.direction === "asc" ? "desc" : "asc");
    } else {
      onsort(name, "asc");
    }
  }

  function snapAllColumnsToPixels() {
    const headerCells = scrollContainer?.querySelectorAll<HTMLElement>("[data-header-col]");
    if (!headerCells) return;
    for (const cell of headerCells) {
      const name = cell.dataset.headerCol ?? "";
      const col = visibleColumns.find((c) => c.name === name);
      if (col && !col.width) {
        onresize?.(name, cell.getBoundingClientRect().width);
      }
    }
  }

  function startResize(e: MouseEvent, col: DataTableColumn) {
    e.preventDefault();
    snapAllColumnsToPixels();
    const cell = (e.currentTarget as HTMLElement).parentElement;
    const measuredWidth = cell ? cell.getBoundingClientRect().width : (col.width ?? 100);
    resizing = {name: col.name, startX: e.clientX, startWidth: measuredWidth};
    window.addEventListener("mousemove", onResizeMove);
    window.addEventListener("mouseup", onResizeUp, {once: true});
  }

  function onResizeMove(e: MouseEvent) {
    if (!resizing) return;
    const delta = e.clientX - resizing.startX;
    const newWidth = Math.max(20, resizing.startWidth + delta);
    onresize?.(resizing.name, newWidth);
  }

  function onResizeUp() {
    window.removeEventListener("mousemove", onResizeMove);
    resizing = null;
  }

  function autoFit(name: string) {
    const cells = scrollContainer?.querySelectorAll(`[data-col="${name}"]`);
    if (!cells) return;
    let max = 60;
    for (const cell of cells) {
      max = Math.max(max, (cell as HTMLElement).scrollWidth);
    }
    onresize?.(name, max);
  }
</script>

{#snippet headerCell(col: DataTableColumn, isLast: boolean)}
  <div class="relative" data-header-col={col.name}>
    <button
      type="button"
      data-no-dnd="true"
      onclick={() => toggleSort(col.name)}
      oncontextmenu={(e) => { e.preventDefault(); headerCtxMenu = {x: e.clientX, y: e.clientY, columnName: col.name}; }}
      class="flex items-center gap-1 px-1 hover:text-fg transition-colors text-left w-full {compact ? 'py-1' : 'py-2'}"
    >
      {col.name}
      {#if sortState?.column === col.name}
        {#if sortState.direction === 'asc'}
          <ArrowUp size={10} />
        {:else}
          <ArrowDown size={10} />
        {/if}
      {:else}
        <ArrowUpDown size={10} class="opacity-30" />
      {/if}
    </button>
    {#if !isLast}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        data-no-dnd="true"
        class="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize bg-border/50 hover:bg-accent/70 z-20"
        onmousedown={(e) => startResize(e, col)}
        ondblclick={() => autoFit(col.name)}
      ></div>
    {/if}
  </div>
{/snippet}

<div class="flex flex-col h-full overflow-hidden isolate">
  {#if toolbar}
    <div class="flex items-center gap-2 px-3 py-2 border-b border-border shrink-0">
      {@render toolbar()}
    </div>
  {/if}

  {#if error}
    <div class="p-4 text-sm text-destructive">{error}</div>
  {:else}
    <div bind:this={scrollContainer} class="flex-1 overflow-auto">
      <div
        class="flex sticky top-0 z-20 bg-bg text-xs font-semibold uppercase tracking-wider text-muted border-b border-border"
      >
        <div
          class="grid sticky left-0 z-30 bg-bg pl-2"
          style="grid-template-columns: {pinnedGridCols}"
        >
          {#if headerPrefix}
            {@render headerPrefix()}
          {/if}
          {#each pinnedColumns as col, i (col.name)}
            {@render headerCell(col, mainColumns.length === 0 && i === pinnedColumns.length - 1)}
          {/each}
        </div>
        <div
          class="grid flex-1 pr-2"
          style="grid-template-columns: {mainGridCols}"
          use:dndzone={{
            items: liveMainColumns,
            type: "table-columns",
            flipDurationMs: 150,
            dropTargetStyle: {outline: "2px dashed currentColor"},
          }}
          onconsider={handleDndConsider}
          onfinalize={handleDndFinalize}
        >
          {#each liveMainColumns as col, i (col.id)}
            {@render headerCell(col, i === liveMainColumns.length - 1)}
          {/each}
          {#if headerSuffix}
            {@render headerSuffix()}
          {/if}
        </div>
      </div>
      {#if loading && items.length === 0}
        <div>
          {#each Array(8) as _, i}
            <div
              class="flex items-center border-b border-border/40"
              style="height: {rowHeight}px;"
            >
              <div class="grid sticky left-0 z-10 pl-2 h-full items-center bg-bg" style="grid-template-columns: {pinnedGridCols}">
                {#each Array(prefixGridCols.length + pinnedColumns.length) as _2, j}
                  <div class="px-1"><div class="h-3 rounded bg-surface-hover animate-pulse" style="width: {50 + ((i + j) % 4) * 10}%"></div></div>
                {/each}
              </div>
              <div class="grid flex-1 pr-2 h-full items-center" style="grid-template-columns: {mainGridCols}">
                {#each Array(liveMainColumns.length + suffixGridCols.length) as _2, j}
                  <div class="px-1"><div class="h-3 rounded bg-surface-hover animate-pulse" style="width: {40 + ((i + j) % 5) * 10}%"></div></div>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      {:else if items.length === 0}
        <div class="flex flex-col items-center justify-center py-12 gap-3">
          <div class="text-sm text-muted">{emptyMessage}</div>
          {#if emptyAction}
            {@render emptyAction()}
          {/if}
        </div>
      {:else}
        <div style="height: {$virtualizer.getTotalSize()}px; position: relative;">
          {#each $virtualizer.getVirtualItems() as row (row.index)}
            {@const item = items[row.index]}
            {#if item}
              <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div
                class="absolute top-0 left-0 min-w-full flex items-center transition-colors group
                  {selectedRow?.(item) ? 'bg-accent/10 border-l-2 border-accent' : 'hover:bg-surface-hover border-l-2 border-transparent'}
                  {onrowclick ? 'cursor-pointer' : ''}"
                style="transform: translateY({row.start}px); height: {rowHeight}px;"
                tabindex={onrowclick ? 0 : undefined}
                onclick={() => onrowclick?.(item)}
                onkeydown={(e) => { if (e.key === 'Enter') onrowclick?.(item) }}
                oncontextmenu={oncontextmenu ? (e) => { e.preventDefault(); e.stopPropagation(); oncontextmenu?.(e, item) } : undefined}
              >
                <div
                  class="grid sticky left-0 z-10 pl-2 h-full items-center
                    {selectedRow?.(item) ? 'bg-accent/10' : 'bg-bg group-hover:bg-surface-hover'}"
                  style="grid-template-columns: {pinnedGridCols}"
                >
                  {#if rowPrefix}
                    {@render rowPrefix({item})}
                  {/if}
                  {#each pinnedColumns as column (column.name)}
                    <div class="px-1 truncate text-sm {alignClass(column)}" data-col={column.name}>
                      {@render cell({item, column})}
                    </div>
                  {/each}
                </div>
                <div
                  class="grid flex-1 pr-2 h-full items-center"
                  style="grid-template-columns: {mainGridCols}"
                >
                  {#each liveMainColumns as column (column.id)}
                    <div class="px-1 truncate text-sm {alignClass(column)}" data-col={column.name}>
                      {@render cell({item, column})}
                    </div>
                  {/each}
                  {#if rowSuffix}
                    {@render rowSuffix({item})}
                  {/if}
                </div>
              </div>
            {/if}
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>

{#if headerCtxMenu}
  {@const menu = headerCtxMenu}
  <HeaderContextMenu
    x={menu.x}
    y={menu.y}
    columnName={menu.columnName}
    isPinned={pinnedSet.has(menu.columnName)}
    canHide={menu.columnName !== "Name" && !pinnedSet.has(menu.columnName)}
    onSort={(dir) => onsort?.(menu.columnName, dir)}
    onAutoFit={() => autoFit(menu.columnName)}
    onTogglePin={() => onTogglePin?.(menu.columnName)}
    onHide={() => onHideColumn?.(menu.columnName)}
    onClose={() => { headerCtxMenu = null; }}
  />
{/if}
