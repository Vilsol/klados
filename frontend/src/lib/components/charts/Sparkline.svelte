<script lang="ts">
  import {untrack} from "svelte";
  import uPlot from "uplot";
  import type {TimeSeriesPoint} from "./types";

  interface Props {
    points: TimeSeriesPoint[];
    color?: string;
    width?: number;
    height?: number;
  }

  let {points, color = "#3b82f6", width, height = 20}: Props = $props();

  let container: HTMLDivElement = $state(null as unknown as HTMLDivElement);
  let chart: uPlot | null = null;

  function toColumnar(pts: TimeSeriesPoint[]): uPlot.AlignedData {
    const xs = new Float64Array(pts.length);
    const ys = new Float64Array(pts.length);
    for (let i = 0; i < pts.length; i++) {
      xs[i] = pts[i].t;
      ys[i] = pts[i].v;
    }
    return [xs, ys];
  }

  // Create / destroy uPlot instance
  $effect(() => {
    const el = container;
    if (!el) {
      return;
    }

    const w = width ?? el.clientWidth;
    const opts: uPlot.Options = {
      width: w,
      height,
      cursor: {show: false},
      legend: {show: false},
      select: {show: false, left: 0, top: 0, width: 0, height: 0},
      axes: [{show: false}, {show: false}],
      scales: {
        x: {time: false},
        y: {
          range: {
            min: {pad: 0.1, soft: 0, mode: 2},
            max: {pad: 0.1},
          },
        },
      },
      series: [
        {},
        {
          stroke: color,
          fill: `${color}33`,
          width: 1,
        },
      ],
    };

    const data = untrack(() => toColumnar(points));
    chart = new uPlot(opts, data, el);

    const ro = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (!(entry && chart)) {
        return;
      }
      const newW = Math.round(entry.contentRect.width);
      if (newW > 0) {
        chart.setSize({width: newW, height});
      }
    });
    ro.observe(el);

    return () => {
      ro.disconnect();
      chart?.destroy();
      chart = null;
    };
  });

  // Reactive data update
  $effect(() => {
    const data = toColumnar(points);
    untrack(() => {
      if (!chart) {
        return;
      }
      chart.setData(data);
    });
  });
</script>

<div bind:this={container} class="w-full" style="height: {height}px;"></div>
