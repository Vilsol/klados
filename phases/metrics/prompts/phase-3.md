# Phase 3 — uPlot Wrapper & Metrics Tab

Build the frontend charting infrastructure: the reusable uPlot Svelte 5 wrapper with hover/zoom/multi-series, the MetricsTab detail panel, time range selector, and the polling data flow — wired to the backend RPCs from Phase 1.

## First Action

Run `cd frontend && pnpm add uplot` to install the dependency, then read `frontend/src/lib/components/ResourceDetail.svelte` lines 36–110 to understand how detail panels are registered and rendered — your `MetricsTab` plugs into this system via the `panelComponents` map and `detailPanels` descriptor arrays.

## Context

Phase 1 created the backend `MetricsService` with `GetCapabilities` and `GetResourceMetrics` RPCs. This phase builds the entire frontend visualization layer. Phase 2 (Prometheus provider) runs in parallel — the charts you build here initially display metrics-server data (instant snapshots accumulated in component state). When Phase 2 completes, the same charts will seamlessly render historical range data without UI changes.

## Files to Read

- `frontend/src/lib/components/ResourceDetail.svelte` — **what to look for**: `panelComponents` Map (line 36), `panelLabels` Record (line 60), `visiblePanels` derived (line 102), and the `{#if panel === ...}` rendering block (line 213) — you'll add a `'metrics'` entry to the map and a corresponding rendering case
- `internal/resource/builtin.go` — **what to look for**: `DetailPanels` arrays (line 28 for pods, etc.) — you'll add `"metrics"` to the panel list for pods, nodes, deployments, statefulsets, daemonsets
- `frontend/src/lib/stores/cluster.svelte.ts` — **what to look for**: `clusterStore` singleton pattern and how it exposes reactive state — your metrics tab reads `activeContext` from here
- `frontend/bindings/` — **what to look for**: binding import pattern (`.js` extension, ESM). After `wails3 generate bindings`, you'll import `MetricsService` from the bindings
- `frontend/src/lib/components/panels/EventsPanel.svelte` — **what to look for**: a reference panel component showing props pattern (`ctxName`, `namespace`, `obj`) and how it's wired in ResourceDetail — your MetricsTab follows the same convention
- `METRICS_SPEC.md` — **what to look for**: MetricsChart props (lines 340–349), Sparkline props (lines 356–363), MetricsTab layout (lines 371–394), poll intervals (lines 309–315)

## What Exists

- `MetricsService` Wails-bound with `GetCapabilities(clusterCtx)`, `GetResourceMetrics(clusterCtx, gvr, ns, name, rangeMinutes)`, `GetNamespaceMetrics(clusterCtx, ns, rangeMinutes)` — returns `MetricsResponse` with `metrics[]`, `thresholds[]` (empty for now), `annotations[]` (empty for now)
- `MetricsCapability` with `HasMetricsServer` boolean (and `HasPrometheus`/`PrometheusURL` fields that may or may not be populated depending on whether Phase 2 is complete)
- `metrics:{ctx}:capabilities` Wails event emitted on cluster connect
- `panelComponents` Map in `ResourceDetail.svelte` — existing panels registered with string keys
- `DetailPanels` arrays in `internal/resource/builtin.go` — per-GVR panel lists
- Tailwind v4 custom tokens: `bg`, `fg`, `muted`, `border`, `accent`, `surface`, `surface-hover`

## Deliverables

1. `uplot` npm dependency installed via `pnpm add uplot`
2. `frontend/src/lib/components/charts/types.ts` — TypeScript interfaces mirroring Go types: `TimeSeriesPoint`, `TimeSeries`, `MetricResult`, `ThresholdLine`, `Annotation`, `MetricsResponse`, `MetricsCapability`
3. `frontend/src/lib/components/charts/units.ts` — Unit formatting: `formatCPU(cores)` → "500m" / "2.5 cores", `formatMemory(bytes)` → "128 MiB" / "4.2 GiB", `formatRatio(r)` → "45%"
4. `frontend/src/lib/components/charts/MetricsChart.svelte` — Core uPlot wrapper:
   - Creates uPlot in `$effect`, destroys on cleanup
   - Multi-series: one series per container/pod, color-coded with toggleable legend
   - Cursor crosshair with tooltip showing all series values at hover point + min/max for visible range
   - Click-drag to zoom, double-click to reset; zoom fires `onzoom` callback
   - ResizeObserver for responsive dimensions
   - Unit formatting via axis format functions (uses `units.ts`)
   - Loading skeleton state
   - Accepts `thresholds` and `annotations` props (renders nothing for now — Phase 4 adds drawing hooks)
5. `frontend/src/lib/components/charts/TimeRangeSelector.svelte` — Preset buttons (15m, 1h, 6h, 24h, 7d), emits selected value. Hidden when `hasPrometheus` is false.
6. `frontend/src/lib/components/charts/MetricsTab.svelte` — Detail drawer tab:
   - Fetches capabilities on mount, conditionally renders
   - Time range selector + source indicator ("metrics-server (live)" or "prometheus")
   - Aggregate CPU chart (all containers overlaid) + Aggregate Memory chart
   - Per-container section with individual CPU and Memory charts
   - Polls at 15s (short range) or 60s (long range), cleans up on unmount
   - Graceful degradation: metrics-server only → hides range selector, accumulates rolling data in component `$state`; no sources → tab not shown
7. `MetricsTab` registered in `ResourceDetail.svelte` `panelComponents` map as `'metrics'`
8. `"metrics"` added to `DetailPanels` in `internal/resource/builtin.go` for: `core.v1.pods`, `core.v1.nodes`, `apps.v1.deployments`, `apps.v1.statefulsets`, `apps.v1.daemonsets`
9. Wails bindings regenerated after Phase 1's Go changes (if not already done)

## Tests

- **Frontend test (vitest)**
  - `MetricsChart` mounts without error and creates a canvas element (uPlot renders to canvas)
  - `MetricsChart` calls `uPlot.setData()` on data prop change, does not recreate the instance
  - `MetricsChart` calls `uPlot.destroy()` on unmount
  - `TimeRangeSelector` emits correct value (`15`, `60`, `360`, `1440`, `10080`) on button click
  - `TimeRangeSelector` is hidden when `hasPrometheus` is `false`
  - `MetricsTab` calls `GetCapabilities` and `GetResourceMetrics` on mount
  - `MetricsTab` sets up `setInterval` polling and clears it on unmount
  - `MetricsTab` is not rendered / returns null when capabilities show no sources
  - `formatCPU(0.5)` → `"500m"`, `formatCPU(2.5)` → `"2.5"`, `formatMemory(1073741824)` → `"1 GiB"`
- **Manual verification**
  - Open pod detail → "Metrics" tab appears
  - Chart renders visible data points (at least one line)
  - Hover shows crosshair + tooltip with values
  - Click-drag zoom narrows visible range, double-click resets
  - Multiple containers show as separate colored lines with legend toggle
  - Per-container section shows individual charts below the aggregate

## Acceptance Criteria

- [ ] `MetricsChart` renders a visible time-series line for at least one data series
- [ ] Hover shows crosshair + tooltip with all series values and min/max for visible range
- [ ] Click-drag zoom works; double-click resets to full range
- [ ] Multi-series renders each container in a different color with toggleable legend
- [ ] Time range selector switches between presets, triggering data refetch
- [ ] Polling starts on mount at correct interval (15s/60s) and stops on unmount (no leaked intervals)
- [ ] Graceful degradation: metrics-server only → no range selector, rolling live data; no sources → tab hidden
- [ ] Per-container section shows individual CPU and Memory charts per container
- [ ] "Metrics" tab visible in detail view for pods, nodes, deployments, statefulsets, daemonsets
- [ ] All frontend unit tests pass

## Definition of Done

Open a pod detail view on a cluster with metrics-server. The "Metrics" tab shows aggregate CPU and Memory charts with data lines, plus per-container breakdowns. Hovering shows a crosshair with exact values. If Prometheus is also available (Phase 2 complete), switching the time range to "1h" fetches historical data and the charts show a full hour of data points. On a cluster with no metric sources, the "Metrics" tab does not appear.

## Known Gotchas

- **uPlot is imperative — no reactive bindings.** Create in `$effect` with a container `div` ref. Update data via `uPlot.setData(newData)`, NOT by destroying and recreating. Destroy in the effect cleanup function. If you recreate on every data update, you'll see flickering and memory leaks.
- **`$effect` + `$state` infinite loop risk.** If the polling `$effect` reads a `$state` variable (like the accumulated data array), every push will re-trigger the effect, causing infinite polling restarts. Wrap data reads inside `untrack()` — see cerebrum.md Do-Not-Repeat entry from 2026-03-23.
- **uPlot data format is columnar, not row-based.** uPlot expects `[timestamps[], series1Values[], series2Values[], ...]` — NOT an array of point objects. You'll need to transform `TimeSeries[]` → columnar arrays before passing to `uPlot.setData()`.
- **`panelComponents` rendering block uses `{#if panel === ...}` chains.** Add a case for `'metrics'` that passes the appropriate props: `obj`, `ctxName`, `gvr`, `namespace`, `name`. Follow the existing pattern — each panel type gets slightly different props.
- **Wails bindings use `.js` extension.** Import as `from '../../bindings/.../metricsservice.js'` — NOT `.ts`. See cerebrum.md Key Learnings.
- **`thresholds` and `annotations` props on `MetricsChart`** should exist in the interface now but render nothing. Phase 4 adds uPlot hooks (`drawSeries`, `drawAxes`) to render them. Defining the props now avoids changing the component API later.
