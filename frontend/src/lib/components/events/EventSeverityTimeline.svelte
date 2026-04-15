<script lang="ts">
  import type {EventItem} from "$lib/event/event-types";
  import {bucketize, pickBucketSize, type TimelineBucket} from "$lib/event/event-timeline";
  import {X} from "lucide-svelte";

  let {
    filteredItems,
    allItems,
    rangeMs,
    now,
    selectedWindow,
    onBrush,
  }: {
    filteredItems: EventItem[];
    allItems: EventItem[];
    rangeMs: number;
    now: number;
    selectedWindow: {from: number; to: number} | null;
    onBrush?: (window: {from: number; to: number} | null) => void;
  } = $props();

  const from = $derived(now - rangeMs);
  const to = $derived(now);
  const bucketSize = $derived(pickBucketSize(rangeMs));
  const filteredBuckets = $derived(bucketize(filteredItems, from, to, bucketSize));
  const totalBuckets = $derived(bucketize(allItems, from, to, bucketSize));
  const maxHeight = $derived.by(() => {
    let m = 1;
    for (const b of totalBuckets) m = Math.max(m, b.warn + b.normal);
    for (const b of filteredBuckets) m = Math.max(m, b.warn + b.normal);
    return m;
  });

  let brushStart = $state<number | null>(null);
  let brushEnd = $state<number | null>(null);
  let hoverIdx = $state<number | null>(null);
  let hoverX = $state(0);
  let containerEl = $state<HTMLDivElement | undefined>(undefined);
  let svgEl = $state<SVGSVGElement | undefined>(undefined);

  const tooltipLeft = $derived.by(() => {
    const w = containerEl?.clientWidth ?? 0;
    const desired = hoverX + 8;
    const maxLeft = Math.max(0, w - 160);
    return Math.min(desired, maxLeft);
  });

  const H = 40;
  const BAR_W = 4;

  function barH(count: number): number {
    if (count === 0) return 0;
    return Math.max(1, (count / maxHeight) * (H - 4));
  }

  function formatBucketLabel(b: TimelineBucket): string {
    const t0 = new Date(b.t0);
    const t1 = new Date(b.t1);
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${pad(t0.getHours())}:${pad(t0.getMinutes())}–${pad(t1.getHours())}:${pad(t1.getMinutes())}`;
  }

  function bucketIndexFromEvent(e: MouseEvent): number | null {
    if (!svgEl) return null;
    const rect = svgEl.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const svgX = (x / rect.width) * svgWidth;
    const idx = Math.floor(svgX / BAR_W);
    if (idx < 0 || idx >= filteredBuckets.length) return null;
    return idx;
  }

  function onSvgMouseDown(e: MouseEvent) {
    const i = bucketIndexFromEvent(e);
    if (i === null) return;
    brushStart = i;
    brushEnd = i;
  }

  function onSvgMouseMove(e: MouseEvent) {
    const i = bucketIndexFromEvent(e);
    if (i === null) return;
    hoverIdx = i;
    hoverX = i * BAR_W;
    if (brushStart !== null) brushEnd = i;
  }

  function onSvgMouseUp() {
    if (brushStart !== null && brushEnd !== null && onBrush) {
      const a = Math.min(brushStart, brushEnd);
      const b = Math.max(brushStart, brushEnd);
      onBrush({from: filteredBuckets[a].t0, to: filteredBuckets[b].t1});
    }
    brushStart = null;
    brushEnd = null;
  }

  const svgWidth = $derived(filteredBuckets.length * BAR_W);

  const selectedRange = $derived.by(() => {
    if (!selectedWindow) return null;
    // Buckets overlap [selectedWindow.from, selectedWindow.to) when bucket.t1 > from AND bucket.t0 < to
    let startIdx = -1;
    let endIdx = -1;
    for (let i = 0; i < filteredBuckets.length; i++) {
      const b = filteredBuckets[i];
      if (b.t1 <= selectedWindow.from) continue;
      if (b.t0 >= selectedWindow.to) break;
      if (startIdx === -1) startIdx = i;
      endIdx = i;
    }
    if (startIdx === -1) return null;
    return {startIdx, endIdx};
  });
</script>

<div
  bind:this={containerEl}
  class="relative border-b border-border"
  style="height: {H}px;"
  onmouseleave={() => { hoverIdx = null; }}
  role="presentation"
>
  <svg
    bind:this={svgEl}
    class="w-full h-full select-none"
    viewBox="0 0 {svgWidth} {H}"
    preserveAspectRatio="none"
    aria-label="Severity timeline"
    role="img"
    onmousedown={onSvgMouseDown}
    onmousemove={onSvgMouseMove}
    onmouseup={onSvgMouseUp}
  >
    <!-- Transparent hit-target across the full width -->
    <rect x="0" y="0" width={svgWidth} height={H} fill="transparent" />

    {#each filteredBuckets as b, i (i)}
      {@const total = totalBuckets[i]}
      {@const overlayH = barH(total.warn + total.normal)}
      {@const warnH = barH(b.warn)}
      {@const normalH = barH(b.normal)}
      <g data-bucket={i} pointer-events="none">
        <rect x={i * BAR_W} y={H - overlayH} width={BAR_W - 1} height={overlayH} class="fill-border/40" />
        <rect x={i * BAR_W} y={H - warnH} width={BAR_W - 1} height={warnH} class="fill-destructive" />
        <rect x={i * BAR_W} y={H - warnH - normalH} width={BAR_W - 1} height={normalH} class="fill-muted" opacity="0.6" />
      </g>
    {/each}

    {#if selectedRange}
      <rect
        x={selectedRange.startIdx * BAR_W}
        y="0"
        width={(selectedRange.endIdx - selectedRange.startIdx + 1) * BAR_W}
        height={H}
        class="fill-accent"
        opacity="0.2"
        pointer-events="none"
      />
    {/if}
  </svg>

  {#if selectedWindow}
    <button
      type="button"
      class="absolute top-0.5 right-0.5 p-0.5 rounded hover:bg-surface-hover text-muted"
      onclick={() => onBrush?.(null)}
      aria-label="Clear time window"
      data-testid="clear-window"
    >
      <X size={10} />
    </button>
  {/if}

  {#if hoverIdx !== null}
    {@const b = filteredBuckets[hoverIdx]}
    <div
      class="absolute pointer-events-none bg-surface border border-border rounded text-xs px-1.5 py-0.5 shadow z-10"
      style="left: {tooltipLeft}px; bottom: {H + 4}px"
    >
      {formatBucketLabel(b)} · {b.warn} warnings, {b.normal} normal
    </div>
  {/if}
</div>
