# Phase 5 — Sparklines in Resource Lists

Add opt-in sparkline columns to pod and node list views, backed by efficient batch queries with a resource count cap for performance.

## First Action

Read `frontend/src/routes/ResourceListPage.svelte` to understand how columns are defined and rendered, and how the list view interacts with `ResourceStore` — your sparkline columns integrate here as new column entries with a custom renderer.

## Context

Phases 1–4 built the full metrics detail experience: backend providers, charts, thresholds, annotations. This phase extends metrics to list views via tiny inline sparklines. The backend `BuiltinQueries` already include `sparkline:*` entries (defined in Phase 2). This phase implements `GetListMetrics` on the backend and the `Sparkline.svelte` component + column integration on the frontend.

## Files to Read

- `frontend/src/routes/ResourceListPage.svelte` — **what to look for**: column definition structure, how columns are passed to the list component, any existing column toggle or visibility mechanism. This is where sparkline columns get added.
- `frontend/src/lib/components/ResourceList.svelte` (if it exists, or wherever the actual list rendering happens) — **what to look for**: how custom cell renderers work, whether columns support component-based rendering, and how TanStack Virtual handles row virtualization — sparklines must only mount for visible rows
- `internal/metrics/service.go` — **what to look for**: existing `GetResourceMetrics` pattern — `GetListMetrics` follows the same style but returns `map[string][]MetricResult`
- `internal/metrics/queries.go` — **what to look for**: `sparkline:core.v1.pods` and `sparkline:core.v1.nodes` entries — these are the batch PromQL queries `GetListMetrics` will execute
- `internal/metrics/metricsserver.go` — **what to look for**: PodMetrics/NodeMetrics list calls — `GetListMetrics` uses these when only metrics-server is available
- `METRICS_SPEC.md` — **what to look for**: Sparkline component spec (lines 354–365), list metrics RPC (lines 193–194), 200-resource cap (line 59)

## What Exists

- `MetricsService` with `GetCapabilities`, `GetResourceMetrics`, `GetNamespaceMetrics`
- `BuiltinQueries` with `sparkline:core.v1.pods` and `sparkline:core.v1.nodes` entries
- metrics-server provider can list all PodMetrics/NodeMetrics in a namespace
- Prometheus provider can execute batch PromQL returning series `by (pod)` or `by (node)`
- `MetricsChart.svelte` — full interactive chart (overkill for sparklines — build a minimal `Sparkline.svelte`)
- `ResourceListPage.svelte` with column definitions and TanStack Virtual scroll

## Deliverables

1. `frontend/src/lib/components/charts/Sparkline.svelte` — Minimal uPlot instance: no axes, no labels, no cursor, no interactivity. Fixed height (~20px), width fills column. Area fill with stroke in accent color. Single series. Creates uPlot in `$effect`, destroys on cleanup. Accepts `SparklineProps { points: TimeSeriesPoint[], color?: string, width?: number, height?: number }`.
2. `MetricsService.GetListMetrics(clusterCtx, gvr, namespace string) (map[string][]MetricResult, error)` — Returns sparkline data keyed by resource name:
   - Prometheus path: executes `sparkline:{gvr}` query from `BuiltinQueries`, fans out result series by pod/node label into the map
   - metrics-server path: lists all PodMetrics or NodeMetrics, returns current instant per resource
   - Returns error if resource count > 200 (checked before executing query)
3. **Sparkline columns in `ResourceListPage`** — Two new optional columns ("CPU" sparkline, "Memory" sparkline) for `core.v1.pods` and `core.v1.nodes`. Hidden by default. Column toggle mechanism to show/hide (use existing column visibility if available, or add a simple toggle).
4. **Batch query polling** — When sparkline columns are enabled, `ResourceListPage` starts a 15s `setInterval` poll via `GetListMetrics`. Poll stops when columns are hidden or component unmounts. Uses `untrack()` to avoid re-triggering effects.
5. **Resource count cap** — If the list has >200 items, sparkline columns show a "Too many resources" tooltip/placeholder instead of charts. `GetListMetrics` returns a specific error for this case. Frontend also checks client-side to avoid unnecessary RPC.

## Tests

- **Go unit test**
  - `GetListMetrics` returns data keyed by pod name matching pods in namespace
  - `GetListMetrics` returns error when resource count > 200
  - `GetListMetrics` uses `sparkline:core.v1.pods` PromQL with correct namespace substitution
  - `GetListMetrics` falls back to PodMetrics list when only metrics-server available
  - `GetListMetrics` returns empty map (not error) for GVRs without sparkline queries defined
- **Frontend test (vitest)**
  - `Sparkline` renders a canvas element with expected dimensions (height ~20px)
  - `Sparkline` destroys uPlot on unmount
  - Sparkline columns are hidden by default in pod list
  - Sparkline columns show "Too many resources" when list exceeds 200 items
  - Enabling sparkline column triggers `GetListMetrics` call
  - Disabling sparkline column stops the poll interval
- **Manual verification**
  - Enable CPU sparkline column in pod list → tiny area charts appear per row
  - Sparklines update every 15s (visible data change)
  - Scroll through a long list → sparklines only mount for visible rows (virtual scroll)
  - Navigate away and back → polling restarts cleanly without duplicates
  - List with >200 pods → sparkline columns show "Too many resources" placeholder

## Acceptance Criteria

- [ ] Sparkline columns appear in column toggle for pod and node list views
- [ ] Sparklines render as tiny area charts (~20px height) per row with correct data
- [ ] Single batch query fires per poll cycle, not per row
- [ ] Polling starts when sparkline column is enabled, stops when disabled or unmounted
- [ ] >200 resources shows "Too many resources" placeholder, no query fires
- [ ] metrics-server only: sparklines show single current-value point (or accumulate while list is open)
- [ ] Missing resource in metrics response (newly created pod) shows empty sparkline, not error
- [ ] Virtual scroll: sparklines only mount for visible rows
- [ ] All unit tests pass

## Definition of Done

Open the pod list view, toggle on the "CPU" sparkline column. Each row shows a tiny area chart showing recent CPU usage. The data updates every 15 seconds. Scrolling through 100 pods is smooth (no jank from canvas elements). Switching to a namespace with >200 pods shows a "Too many resources" message in the sparkline cells instead of charts. Disabling the column stops all metric polling.

## Known Gotchas

- **Sparkline data is keyed by resource name.** If a pod name doesn't appear in the `GetListMetrics` response (e.g., just-created pod before metrics are available), show an empty sparkline — not an error or missing cell.
- **200 canvas elements per column is the practical limit.** TanStack Virtual only renders visible rows (typically 20-30), so actual mounted sparklines are well under this. But verify scroll performance with a full list — if uPlot instances aren't destroyed on row unmount, they accumulate.
- **uPlot sparkline mode is just uPlot with everything turned off.** Set `axes: [{show: false}, {show: false}]`, `legend: {show: false}`, `cursor: {show: false}`, `select: {show: false}`, and `scales: { x: {time: false} }` (or true, but no formatting needed). The area fill comes from `series[1].fill` + `series[1].stroke`.
- **`untrack()` in polling effect.** The sparkline polling `$effect` should track only the "is sparkline column enabled" signal, not the data array it writes to. Wrap `setInterval` callback data writes in `untrack()`. See cerebrum.md Do-Not-Repeat for the same pattern.
- **metrics-server returns one instant per resource.** There's no 5-minute history — each poll adds one point. After 20 polls (5 minutes at 15s interval), you'll have 20 points to draw. This is fine for a sparkline but the first render will show a single dot until points accumulate.
