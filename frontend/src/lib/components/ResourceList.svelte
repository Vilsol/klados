<script lang="ts">
  import {createVirtualizer} from "@tanstack/svelte-virtual";
  import {ArrowUpDown, ArrowUp, ArrowDown, Trash2, RefreshCw, Columns3, Check, Minus, Download} from "lucide-svelte";
  import {ConfirmDialog} from "@klados/ui";
  import {notificationStore} from "$lib/stores/notification.svelte";
  import {evalExpr, defaultAlign, type ColumnDef, type RenderType} from "$lib/registry/index";
  import {getControllerRef, type ControllerRef} from "$lib/utils/relationships";
  import * as ResourceService from "../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js";
  import {formatAge} from "$lib/utils/age";
  import {onMount} from "svelte";
  import {slotRegistry} from "$lib/plugins/slots.svelte.js";
  import {loadPluginComponent} from "$lib/plugins/loader.js";
  import {streamingStore} from "$lib/stores/streaming.svelte.js";
  import Sparkline from "./charts/Sparkline.svelte";
  import type {MetricResult} from "./charts/types";
  import {columnStore} from "$lib/stores/columns.svelte";
  import {clusterStore} from "$lib/stores/cluster.svelte";
  import {selectionStore} from "$lib/stores/selection.svelte";
  import ColumnMenu from "./ColumnMenu.svelte";
  import SmartSearch from "./SmartSearch.svelte";
  import SavedFilterDropdown from "./SavedFilterDropdown.svelte";
  import {filterItems} from "$lib/search/filter";
  import type {SearchTerm} from "$lib/search/parser";
  import {exportItems} from "$lib/utils/export";

  function itemKey(obj: Record<string, any>): string {
    const ns = obj.metadata?.namespace ?? "";
    const name = obj.metadata?.name ?? "";
    return ns ? `${ns}/${name}` : name;
  }

  let now = $state(Date.now());
  onMount(() => {
    const id = setInterval(() => {
      now = Date.now();
    }, 1_000);
    return () => clearInterval(id);
  });

  let {
    items,
    contextName,
    gvr,
    selectedNamespaces = [],
    loading = false,
    error = null,
    selectedName = null,
    scrollContainer = $bindable<HTMLDivElement | undefined>(undefined),
    onrefresh,
    onselect,
    onopenowner,
    sparklineGvrs = [],
    sparklineData = {},
    sparklineColumns = [],
    onSparklineToggle,
    rowActions,
  }: {
    items: Record<string, any>[];
    contextName: string;
    gvr: string;
    selectedNamespaces?: string[];
    loading?: boolean;
    error?: string | null;
    selectedName?: string | null;
    scrollContainer?: HTMLDivElement;
    onrefresh?: () => void;
    onselect?: (item: Record<string, any>) => void;
    onopenowner?: (ref: ControllerRef, namespace: string) => void;
    sparklineGvrs?: string[];
    sparklineData?: Record<string, MetricResult[]>;
    sparklineColumns?: string[];
    onSparklineToggle?: (columns: string[]) => void;
    rowActions?: (
      item: Record<string, any>,
    ) => Array<{label: string; icon?: any; onClick: () => void; variant?: "default" | "destructive"}>;
  } = $props();

  let searchTerms = $state<SearchTerm[]>([]);
  let searchQuery = $state("");
  let deleteTarget = $state<{namespace: string; name: string} | null>(null);
  let confirmOpen = $state(false);
  let ctxMenu = $state<{x: number; y: number; item: Record<string, any>} | null>(null);
  let ctxMenuEl = $state<HTMLDivElement | null>(null);
  let columnMenuOpen = $state(false);
  let exportMenuOpen = $state(false);
  let resizing = $state<{name: string; startX: number; startWidth: number} | null>(null);

  function getSparklinePoints(itemName: string, metricName: string): {t: number; v: number}[] {
    const metrics = sparklineData[itemName];
    if (!metrics) {
      return [];
    }
    const metric = metrics.find((m) => m.name === metricName);
    if (!metric?.series?.[0]?.points) {
      return [];
    }
    return metric.series[0].points;
  }

  const pluginColumns = $derived(slotRegistry.getListColumns(gvr));
  const pluginMenuItems = $derived(slotRegistry.getContextMenuItems(gvr));
  const basePluginURL = $derived(
    streamingStore.config ? `http://127.0.0.1:${streamingStore.config.port}/${streamingStore.config.token}/plugins` : null,
  );

  $effect(() => {
    if (!ctxMenu) {
      return;
    }
    const close = () => {
      ctxMenu = null;
    };
    window.addEventListener("click", close, {once: true});
    return () => window.removeEventListener("click", close);
  });

  $effect(() => {
    if (!ctxMenu || !ctxMenuEl) {
      return;
    }
    const rect = ctxMenuEl.getBoundingClientRect();
    const maxX = window.innerWidth - rect.width - 8;
    const maxY = window.innerHeight - rect.height - 8;
    if (ctxMenu.x > maxX || ctxMenu.y > maxY) {
      ctxMenu = {
        ...ctxMenu,
        x: Math.max(0, Math.min(ctxMenu.x, maxX)),
        y: Math.max(0, Math.min(ctxMenu.y, maxY)),
      };
    }
  });

  $effect(() => {
    if (!columnMenuOpen) {
      return;
    }
    const close = () => {
      columnMenuOpen = false;
    };
    const timer = setTimeout(() => window.addEventListener("click", close, {once: true}), 0);
    return () => {
      clearTimeout(timer);
      window.removeEventListener("click", close);
    };
  });

  $effect(() => {
    if (!exportMenuOpen) {
      return;
    }
    const close = () => {
      exportMenuOpen = false;
    };
    const timer = setTimeout(() => window.addEventListener("click", close, {once: true}), 0);
    return () => {
      clearTimeout(timer);
      window.removeEventListener("click", close);
    };
  });

  // Scroll to top when GVR changes
  $effect(() => {
    gvr;
    searchQuery = "";
    searchTerms = [];
    if (scrollContainer) {
      scrollContainer.scrollTop = 0;
    }
  });

  const filtered = $derived.by(() => {
    let result = items;
    if (selectedNamespaces.length > 1) {
      result = result.filter((item) => selectedNamespaces.includes(item.metadata?.namespace ?? ""));
    }
    result = filterItems(result, searchTerms);
    if (columnStore.sortState) {
      const {column, direction} = columnStore.sortState;
      const col = columnStore.visibleColumns.find((c) => c.name === column);
      if (col?.expr) {
        // Pre-compute sort keys to avoid repeated evalExpr in comparator
        const keyed = result.map((item) => ({item, key: String(evalExpr(col.expr, item) ?? "")}));
        const isAge = col.renderType === "age";
        keyed.sort((a, b) => {
          let cmp: number;
          if (isAge) {
            cmp = a.key.localeCompare(b.key);
          } else {
            const an = parseFloat(a.key);
            const bn = parseFloat(b.key);
            cmp = Number.isFinite(an) && Number.isFinite(bn) ? an - bn : a.key.localeCompare(b.key);
          }
          return direction === "asc" ? cmp : -cmp;
        });
        result = keyed.map((k) => k.item);
      }
    }
    return result;
  });

  const filteredKeys = $derived(filtered.map((item) => itemKey(item)));
  const filteredItemsByKey = $derived.by(() => {
    const map = new Map<string, Record<string, any>>();
    for (const item of filtered) {
      map.set(itemKey(item), item);
    }
    return map;
  });

  $effect(() => {
    selectionStore.setVisibleKeys(new Set(filteredKeys));
  });

  const allVisibleSelected = $derived(filtered.length > 0 && filteredKeys.every((k) => selectionStore.isSelected(k)));
  const someVisibleSelected = $derived(!allVisibleSelected && filteredKeys.some((k) => selectionStore.isSelected(k)));
  const canMutate = $derived(clusterStore.canMutate());

  const tooManyForSparklines = $derived(filtered.length > 200);

  const rowHeight = $derived(columnStore.compact ? 28 : 36);

  const virtualizer = $derived.by(() => {
    const rh = rowHeight;
    return createVirtualizer({
      count: filtered.length,
      getScrollElement: () => scrollContainer ?? null,
      estimateSize: () => rh,
      overscan: 10,
    });
  });

  function toggleSort(name: string) {
    const current = columnStore.sortState;
    if (current?.column === name) {
      columnStore.setSort(name, current.direction === "asc" ? "desc" : "asc");
    } else {
      columnStore.setSort(name, "asc");
    }
  }

  function renderCell(col: ColumnDef, item: Record<string, any>) {
    return evalExpr(col.expr, item);
  }

  function renderValue(value: any, renderType: RenderType): string {
    if (value == null) {
      return "";
    }
    if (renderType === "age") {
      return formatAge(String(value), now);
    }
    return String(value);
  }

  function badgeClass(value: any): string {
    const v = String(value ?? "").toLowerCase();
    if (["running", "active", "bound", "available", "true"].includes(v)) {
      return "bg-accent/20 text-accent border-accent/30";
    }
    if (["error", "crashloopbackoff", "failed", "oomkilled"].includes(v)) {
      return "bg-destructive/20 text-destructive border-destructive/30";
    }
    if (["pending", "terminating"].includes(v)) {
      return "bg-muted/20 text-muted border-muted/30";
    }
    return "bg-muted/10 text-fg border-border";
  }

  function alignClass(col: ColumnDef): string {
    const align = col.align ?? defaultAlign(col.renderType);
    return align === "right" ? "text-right" : align === "center" ? "text-center" : "text-left";
  }

  async function confirmDelete() {
    if (!deleteTarget) {
      return;
    }
    const {namespace, name} = deleteTarget;
    try {
      await ResourceService.DeleteResource(contextName, gvr, namespace, name);
      notificationStore.push(`Deleted ${name}`, "success");
    } catch (e: any) {
      notificationStore.push(`Failed to delete: ${e?.message ?? e}`, "error");
    }
    deleteTarget = null;
  }

  function requestDelete(item: Record<string, any>) {
    deleteTarget = {
      namespace: item.metadata?.namespace ?? "",
      name: item.metadata?.name ?? "",
    };
    confirmOpen = true;
  }

  const gridTemplateCols = $derived.by(() => {
    const parts: string[] = [];
    if (canMutate) {
      parts.push("36px");
    }
    for (const c of columnStore.visibleColumns) {
      parts.push(c.width ? `${c.width}px` : "minmax(20px, 1fr)");
    }
    for (const _ of pluginColumns) {
      parts.push("1fr");
    }
    for (const _ of sparklineColumns) {
      parts.push("80px");
    }
    parts.push("36px");
    return parts.join(" ");
  });

  function snapAllColumnsToPixels() {
    const headerCells = scrollContainer?.querySelectorAll<HTMLElement>("[data-header-col]");
    if (!headerCells) {
      return;
    }
    for (const cell of headerCells) {
      const name = cell.dataset.headerCol!;
      const col = columnStore.visibleColumns.find((c) => c.name === name);
      if (col && !col.width) {
        columnStore.resizeColumn(name, cell.getBoundingClientRect().width);
      }
    }
  }

  function startResize(e: MouseEvent, col: ColumnDef) {
    e.preventDefault();
    snapAllColumnsToPixels();
    const cell = (e.currentTarget as HTMLElement).parentElement;
    const measuredWidth = cell ? cell.getBoundingClientRect().width : (col.width ?? 100);
    resizing = {name: col.name, startX: e.clientX, startWidth: measuredWidth};
    window.addEventListener("mousemove", onResizeMove);
    window.addEventListener("mouseup", onResizeUp, {once: true});
  }

  function onResizeMove(e: MouseEvent) {
    if (!resizing) {
      return;
    }
    const delta = e.clientX - resizing.startX;
    const newWidth = Math.max(20, resizing.startWidth + delta);
    columnStore.resizeColumn(resizing.name, newWidth);
  }

  function onResizeUp() {
    window.removeEventListener("mousemove", onResizeMove);
    resizing = null;
  }

  function autoFit(name: string) {
    const cells = scrollContainer?.querySelectorAll(`[data-col="${name}"]`);
    if (!cells) {
      return;
    }
    let max = 60;
    for (const cell of cells) {
      max = Math.max(max, (cell as HTMLElement).scrollWidth);
    }
    columnStore.autoFitColumn(name, max);
  }
</script>

<div class="flex flex-col h-full overflow-hidden isolate">
  <div class="flex items-center gap-2 px-3 py-2 border-b border-border shrink-0">
    <SmartSearch {items} bind:value={searchQuery} ontermschange={(t) => { searchTerms = t }} />
    <SavedFilterDropdown {gvr} {contextName} currentQuery={searchQuery} onapply={(q) => { searchQuery = q }} />
    <span class="text-xs text-muted">{filtered.length} items</span>
    <div class="relative">
      <button
        onclick={() => exportMenuOpen = !exportMenuOpen}
        class="p-1 rounded hover:bg-surface-hover transition-colors"
        title="Export visible"
        aria-label="Export visible"
      >
        <Download size={14} />
      </button>
      {#if exportMenuOpen}
        <div class="absolute top-full mt-1 right-0 z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-24">
          <button
            class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
            onclick={() => { exportItems(filtered, gvr, 'yaml'); exportMenuOpen = false }}
          >
            YAML
          </button>
          <button
            class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover"
            onclick={() => { exportItems(filtered, gvr, 'json'); exportMenuOpen = false }}
          >
            JSON
          </button>
        </div>
      {/if}
    </div>
    <div class="relative">
      <button
        onclick={() => columnMenuOpen = !columnMenuOpen}
        class="p-1 rounded hover:bg-surface-hover transition-colors"
        title="Manage columns"
        aria-label="Manage columns"
      >
        <Columns3 size={14} />
      </button>
      {#if columnMenuOpen}
        <ColumnMenu {gvr} {sparklineGvrs} {sparklineColumns} {onSparklineToggle} />
      {/if}
    </div>
    {#if onrefresh}
      <button onclick={onrefresh} class="p-1 rounded hover:bg-surface-hover transition-colors" title="Refresh" aria-label="Refresh">
        <RefreshCw size={14} class={loading ? 'animate-spin' : ''} />
      </button>
    {/if}
  </div>

  {#if error}
    <div class="p-4 text-sm text-destructive">{error}</div>
  {:else}
    <div bind:this={scrollContainer} class="flex-1 overflow-auto">
      <div
        class="grid text-xs font-semibold uppercase tracking-wider text-muted border-b border-border sticky top-0 z-20 bg-bg px-2"
        style="grid-template-columns: {gridTemplateCols}"
      >
        {#if canMutate}
          <div class="flex items-center justify-center {columnStore.compact ? 'py-1' : 'py-2'}">
            <button
              onclick={() => {
                if (allVisibleSelected) {
                  selectionStore.deselectAll()
                } else {
                  selectionStore.selectAll(filteredKeys, filteredItemsByKey)
                }
              }}
              class="w-4 h-4 rounded border border-border flex items-center justify-center hover:border-accent transition-colors
                {allVisibleSelected || someVisibleSelected ? 'bg-accent border-accent' : ''}"
              aria-label={allVisibleSelected ? 'Deselect all' : 'Select all'}
            >
              {#if allVisibleSelected}
                <Check size={10} class="text-accent-fg" />
              {:else if someVisibleSelected}
                <Minus size={10} class="text-accent-fg" />
              {/if}
            </button>
          </div>
        {/if}
        {#each columnStore.visibleColumns as col, i}
          <div class="relative" data-header-col={col.name}>
            <button
              onclick={() => toggleSort(col.name)}
              class="flex items-center gap-1 px-1 hover:text-fg transition-colors text-left w-full {columnStore.compact ? 'py-1' : 'py-2'}"
            >
              {col.name}
              {#if columnStore.sortState?.column === col.name}
                {#if columnStore.sortState.direction === 'asc'}
                  <ArrowUp size={10} />
                {:else}
                  <ArrowDown size={10} />
                {/if}
              {:else}
                <ArrowUpDown size={10} class="opacity-30" />
              {/if}
            </button>
            {#if i < columnStore.visibleColumns.length - 1}
              <!-- svelte-ignore a11y_no_static_element_interactions -->
              <div
                class="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize bg-border/50 hover:bg-accent/70 z-20"
                onmousedown={(e) => startResize(e, col)}
                ondblclick={() => autoFit(col.name)}
              ></div>
            {/if}
          </div>
        {/each}
        {#each pluginColumns as pcol (pcol.id)}
          <div class="py-2 px-1">{pcol.label}</div>
        {/each}
        {#each sparklineColumns as scol}
          <div class="py-2 px-1">{scol}</div>
        {/each}
        <div></div>
      </div>
      {#if loading}
        <div class="flex items-center justify-center py-12 text-sm text-muted">Loading...</div>
      {:else if filtered.length === 0}
        <div class="flex items-center justify-center py-12 text-sm text-muted">No resources found</div>
      {:else}
        <div style="height: {$virtualizer.getTotalSize()}px; position: relative;">
          {#each $virtualizer.getVirtualItems() as row (row.index)}
            {@const item = filtered[row.index]}
            {@const isSelected = selectedName === `${item.metadata?.name ?? ''}/${item.metadata?.namespace ?? ''}`}
            <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
            <div
              class="absolute top-0 left-0 min-w-full flex items-center px-2 transition-colors group
                {isSelected ? 'bg-accent/10 border-l-2 border-accent' : 'hover:bg-surface-hover border-l-2 border-transparent'}
                {onselect ? 'cursor-pointer' : ''}"
              style="transform: translateY({row.start}px); height: {rowHeight}px;"
              tabindex={onselect ? 0 : undefined}
              onclick={() => onselect?.(item)}
              onkeydown={(e) => { if (e.key === 'Enter') onselect?.(item) }}
              oncontextmenu={(e) => { e.preventDefault(); e.stopPropagation(); ctxMenu = { x: e.clientX, y: e.clientY, item } }}
            >
              <div class="grid flex-1" style="grid-template-columns: {gridTemplateCols}">
                {#if canMutate}
                  {@const key = itemKey(item)}
                  <div class="flex items-center justify-center" onclick={(e) => e.stopPropagation()} role="none">
                    <button
                      onclick={(e) => {
                        e.stopPropagation()
                        if (e.shiftKey) {
                          selectionStore.selectRange(key, filteredKeys, filteredItemsByKey)
                        } else {
                          selectionStore.toggle(key, item)
                        }
                      }}
                      class="w-4 h-4 rounded border border-border flex items-center justify-center hover:border-accent transition-colors
                        {selectionStore.isSelected(key) ? 'bg-accent border-accent' : ''}"
                      aria-label={selectionStore.isSelected(key) ? 'Deselect' : 'Select'}
                    >
                      {#if selectionStore.isSelected(key)}
                        <Check size={10} class="text-accent-fg" />
                      {/if}
                    </button>
                  </div>
                {/if}
                {#each columnStore.visibleColumns as col, i}
                  {@const value = renderCell(col, item)}
                  <div
                    class="px-1 truncate text-sm {alignClass(col)}
                      {col.name === 'Namespace' ? 'cursor-pointer hover:text-accent' : ''}"
                    data-col={col.name}
                    onclick={col.name === 'Namespace' ? (e) => { e.stopPropagation(); clusterStore.setNamespaces(contextName, [String(value)]) } : undefined}
                  >
                    {#if col.renderType === 'controlledBy'}
                      {@const ref = getControllerRef(item)}
                      {#if ref}
                        {#if onopenowner && clusterStore.resolveOwnerGVR(ref.apiVersion, ref.kind)}
                          <button
                            class="text-accent hover:underline cursor-pointer"
                            title="{ref.kind}/{ref.name}"
                            onclick={(e) => { e.stopPropagation(); onopenowner!(ref, item.metadata?.namespace ?? '') }}
                          >
                            {ref.kind}
                          </button>
                        {:else}
                          <span title="{ref.kind}/{ref.name}">{ref.kind}</span>
                        {/if}
                      {/if}
                    {:else if col.renderType === 'badge'}
                      <span class="px-1.5 py-0.5 text-xs rounded border {badgeClass(value)}" title={renderValue(value, col.renderType)}>
                        {renderValue(value, col.renderType)}
                      </span>
                    {:else}
                      <span class={col.renderType === 'age' ? 'text-muted' : ''} title={renderValue(value, col.renderType)}>
                        {renderValue(value, col.renderType)}
                      </span>
                    {/if}
                  </div>
                {/each}
                {#each pluginColumns as pcol (pcol.id)}
                  <div class="px-1 flex items-center overflow-hidden text-sm">
                    {#if basePluginURL}
                      {#await loadPluginComponent(pcol.pluginName, pcol.component, basePluginURL) then Cmp}
                        {#if Cmp}
                          <Cmp resource={item} />
                        {/if}
                      {/await}
                    {/if}
                  </div>
                {/each}
                {#each sparklineColumns as scol}
                  <div class="px-1 flex items-center overflow-hidden">
                    {#if tooManyForSparklines}
                      <span class="text-xs text-muted" title="Sparklines disabled for >200 resources">Too many</span>
                    {:else}
                      {@const pts = getSparklinePoints(item.metadata?.name ?? '', scol)}
                      {#if pts.length > 0}
                        <Sparkline points={pts} height={20} />
                      {:else}
                        <div style="height: 20px;"></div>
                      {/if}
                    {/if}
                  </div>
                {/each}
                <div class="flex items-center justify-end gap-1">
                  {#if rowActions}
                    {#each rowActions(item) as action}
                      <button
                        onclick={(e) => { e.stopPropagation(); action.onClick() }}
                        class="p-1 rounded opacity-0 group-hover:opacity-60 hover:!opacity-100 transition-all {action.variant === 'destructive' ? 'hover:text-destructive' : 'hover:text-fg'}"
                        title={action.label}
                        aria-label={action.label}
                      >
                        {#if action.icon}
                          <action.icon size={13} />
                        {:else}
                          <span class="text-xs">{action.label}</span>
                        {/if}
                      </button>
                    {/each}
                  {:else if canMutate}
                    <button
                      onclick={(e) => { e.stopPropagation(); requestDelete(item) }}
                      class="p-1 rounded opacity-0 group-hover:opacity-60 hover:!opacity-100 hover:text-destructive transition-all"
                      title="Delete"
                      aria-label="Delete {item.metadata?.name}"
                    >
                      <Trash2 size={13} />
                    </button>
                  {/if}
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>

{#if ctxMenu}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    bind:this={ctxMenuEl}
    class="fixed z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-36"
    style="left:{ctxMenu.x}px; top:{ctxMenu.y}px"
    onclick={(e) => e.stopPropagation()}
    onkeydown={(e) => e.stopPropagation()}
  >
    {#each pluginMenuItems as mi (mi.id)}
      {#if basePluginURL}
        {#await loadPluginComponent(mi.pluginName, mi.component, basePluginURL) then Cmp}
          {#if Cmp}
            {@const menuItem = ctxMenu}
            <Cmp resource={menuItem.item} onclose={() => { ctxMenu = null }} />
          {/if}
        {/await}
      {/if}
    {/each}
    {#if canMutate}
      <button
        class="w-full text-left px-3 py-1.5 text-sm text-destructive hover:bg-surface-hover"
        onclick={() => { requestDelete(ctxMenu!.item); ctxMenu = null }}
      >
        Delete
      </button>
    {/if}
  </div>
{/if}

<ConfirmDialog
  bind:open={confirmOpen}
  title="Delete resource"
  message="Delete {deleteTarget?.name}? This action cannot be undone."
  confirmLabel="Delete"
  onconfirm={confirmDelete}
/>
