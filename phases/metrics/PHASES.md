# Metrics System — Phased Implementation Plan

## Project Overview

Klados needs resource metrics (CPU, memory) for nodes, pods, and containers, sourced from metrics-server and/or any Prometheus-compatible endpoint. The system must gracefully degrade based on available sources, display interactive time-series charts via uPlot, and support plugin-extensible metric queries. No metric data is stored locally — all persistence comes from the upstream source.

## Phase Map

```
Phase 1 — Types, Provider Interface, metrics-server Provider
  ├── Phase 2 — Prometheus Provider + Auto-Detection (parallel)
  │     └── Phase 4 — Thresholds, Annotations & Overlays
  └── Phase 3 — uPlot Wrapper + MetricsTab (parallel)
        └── Phase 4 — Thresholds, Annotations & Overlays
              └── Phase 5 — Sparklines in List Views
                    └── Phase 6 — Plugin Metric Templates
```

---

## Phase 1 — Core Types, Provider Interface & metrics-server

> Establishes the shared data types, the `MetricsProvider` abstraction, and the first concrete provider (metrics-server), giving the frontend a working RPC to fetch live CPU/memory for pods and nodes.

| | |
|---|---|
| **Depends on** | none |
| **Parallel with** | nothing |

### Deliverables

- `internal/metrics/types.go` — `TimeSeriesPoint`, `TimeSeries`, `MetricResult`, `ThresholdLine`, `Annotation`, `MetricsResponse`, `MetricQuery`, `MetricsCapability` structs as defined in METRICS_SPEC.md
- `internal/metrics/provider.go` — `MetricsProvider` interface (`QueryRange`, `QueryInstant`, `Available`, `Name`) and `providerSet` struct
- `internal/metrics/metricsserver.go` — metrics-server provider that calls the `metrics.k8s.io/v1beta1` API for `PodMetrics` and `NodeMetrics`, normalizes nanocores → cores and bytes, returns `MetricsResponse`
- `internal/metrics/service.go` — `MetricsService` (Wails-bound) with `GetCapabilities`, `GetResourceMetrics` (metrics-server path only), `GetNamespaceMetrics` (aggregates PodMetrics list)
- metrics-server detection integrated into `cluster.Manager.Connect()` — checks discovered API groups for `metrics.k8s.io/v1beta1`, populates `MetricsCapability.HasMetricsServer`, emits `metrics:{ctx}:capabilities` event
- `MetricsConfig` added to `config.Config` (per cluster context), initially just the struct with `PrometheusURL` field (unused until Phase 2)
- Wails bindings regenerated and `MetricsService` registered in `AppService`

### Tests

- **Go unit test**
  - `MetricsCapability` correctly reports `HasMetricsServer=true` when `metrics.k8s.io/v1beta1` is in discovery, `false` when absent
  - metrics-server provider normalizes nanocores to cores correctly (e.g. `500_000_000` nanocores → `0.5` cores)
  - metrics-server provider returns error when metrics-server API is unavailable
  - `GetResourceMetrics` for `core.v1.pods` returns CPU and memory `MetricResult` with correct units
  - `GetNamespaceMetrics` aggregates across all pods in namespace
- **Integration test (requires live cluster with metrics-server)**
  - `GetCapabilities` detects metrics-server on a real cluster
  - `GetResourceMetrics` returns non-zero values for a running pod

### Out of Scope

- Prometheus provider and detection (Phase 2)
- Frontend chart rendering (Phase 3)
- Threshold/annotation overlays (Phase 4)
- `GetListMetrics` / sparklines (Phase 5)
- Plugin metric queries (Phase 6)

### Acceptance Criteria

- [ ] `MetricsService.GetCapabilities` returns `HasMetricsServer=true` on a cluster with metrics-server installed
- [ ] `MetricsService.GetResourceMetrics` returns CPU and memory data for a pod, with `unit: "cores"` and `unit: "bytes"`
- [ ] `MetricsService.GetNamespaceMetrics` returns aggregated CPU/memory across all pods in a namespace
- [ ] `metrics:{ctx}:capabilities` event is emitted on cluster connect
- [ ] Unit values are normalized: CPU always in cores (float64), memory always in bytes (float64)
- [ ] All Go unit tests pass

### Handoff Notes

- The `MetricsProvider` interface has `QueryRange` which the metrics-server provider should return `ErrNotSupported` for (it only does instant queries). Phase 2 implements the Prometheus provider that supports range queries.
- `MetricsService.GetResourceMetrics` has a `rangeMinutes` parameter that is ignored when only metrics-server is available (returns instant snapshot). Phase 2 will add the Prometheus path that uses this parameter.
- The `MetricsCapability` struct has `HasPrometheus` and `PrometheusURL` fields — leave them unpopulated until Phase 2.
- Unit normalization contract: CPU = `"cores"` (float64, e.g. 0.5 = 500m), Memory = `"bytes"` (float64). Both providers must output in these units. The frontend will handle display formatting (millicores, MiB, GiB).

---

## Phase 2 — Prometheus Provider & Auto-Detection

> Adds the Prometheus HTTP client, auto-detection of in-cluster Prometheus instances, manual endpoint configuration, and range query support — enabling historical time-series data.

| | |
|---|---|
| **Depends on** | Phase 1 |
| **Parallel with** | Phase 3 |

### Deliverables

- `internal/metrics/prometheus.go` — `PrometheusClient` HTTP client (`QueryRange` via `/api/v1/query_range`, `QueryInstant` via `/api/v1/query`), reuses cluster `rest.Config` transport for in-cluster access, plain HTTP for external endpoints
- `internal/metrics/queries.go` — `BuiltinQueries` map with PromQL templates for pods, nodes, deployments, namespace aggregation, and sparkline batch queries as defined in METRICS_SPEC.md. Template variable substitution (`{{namespace}}`, `{{name}}`, `{{node}}`, `{{container}}`)
- Prometheus auto-detection in `cluster.Manager.Connect()`:
  1. Well-known service name scan (`prometheus`, `prometheus-server`, `prometheus-kube-prometheus-prometheus`, `prometheus-operated`) across namespaces (`monitoring`, `prometheus`, `observability`, `default`)
  2. Prometheus Operator CRD check (`monitoring.coreos.com/v1` in discovery → list `Prometheus` CRs → extract service endpoint)
- `MetricsService.SetPrometheusEndpoint` — persists manual endpoint to `config.json`, rebuilds provider, re-emits capabilities event
- `MetricsService.RedetectSources` — re-runs auto-detection, updates capability, emits event
- `MetricsService.GetResourceMetrics` updated: prefers Prometheus for range queries (uses `rangeMinutes` + step size table), falls back to metrics-server for instant
- Time range → step size mapping: 15m→15s, 1h→15s, 6h→1m, 24h→5m, 7d→30m

### Tests

- **Go unit test**
  - Prometheus client parses `/api/v1/query_range` JSON response correctly into `[]TimeSeries`
  - Prometheus client parses `/api/v1/query` (instant) response correctly
  - Template variable substitution replaces `{{namespace}}`, `{{name}}` etc. in PromQL strings
  - Step size calculation returns correct duration for each preset (15m→15s, 1h→15s, 6h→1m, 24h→5m, 7d→30m)
  - Auto-detection finds service named `prometheus-server` in `monitoring` namespace (mock discovery)
  - Auto-detection finds Prometheus Operator CR and extracts endpoint (mock discovery)
  - Manual endpoint override takes precedence over auto-detection
  - `GetResourceMetrics` with `rangeMinutes > 0` uses Prometheus; with `rangeMinutes == 0` uses metrics-server
- **Go unit test (HTTP mock)**
  - `PrometheusClient.QueryRange` sends correct `query`, `start`, `end`, `step` URL params
  - `PrometheusClient` returns structured error on non-200 or malformed JSON
- **Integration test (requires cluster with Prometheus)**
  - Auto-detection discovers kube-prometheus-stack
  - Range query returns time-series data for a known pod

### Out of Scope

- Frontend chart rendering (Phase 3 — runs in parallel)
- Threshold/annotation overlays (Phase 4)
- Sparkline batch queries (Phase 5 — queries defined here but `GetListMetrics` not implemented)
- Plugin metric templates (Phase 6)

### Acceptance Criteria

- [ ] `GetCapabilities` returns `HasPrometheus=true` and a populated `PrometheusURL` on a cluster with kube-prometheus-stack
- [ ] `GetResourceMetrics` returns historical time-series (multiple points) when Prometheus is available
- [ ] `SetPrometheusEndpoint` persists URL to config and subsequent `GetCapabilities` reflects it
- [ ] `RedetectSources` re-runs detection and updates capabilities
- [ ] Works with Thanos/Mimir/VictoriaMetrics query endpoints (they speak the same `/api/v1/query_range` API)
- [ ] `GetResourceMetrics` falls back to metrics-server when Prometheus is unavailable
- [ ] All Go unit tests pass

### Handoff Notes

- The built-in PromQL queries in `queries.go` include `sparkline:*` and `*:thresholds` keys — these are used by Phase 5 (sparklines) and Phase 4 (thresholds) respectively. Don't remove them even though nothing calls them yet.
- Prometheus detection is best-effort. If neither well-known services nor Prometheus Operator CRDs are found, `HasPrometheus` stays false. The user can always manually configure via `SetPrometheusEndpoint`.
- The HTTP client for in-cluster Prometheus reuses `rest.Config` transport — this means it inherits the cluster's auth (ServiceAccount token, client certs). For external endpoints the user provides, use a plain `http.Client` with reasonable timeouts (10s connect, 30s response).
- `queries.go` template vars use double-brace `{{var}}` syntax — simple string replacement, not Go templates. This keeps it trivial for plugins to author queries in Phase 6.

---

## Phase 3 — uPlot Wrapper & Metrics Tab

> Builds the frontend charting infrastructure: the reusable uPlot Svelte wrapper with hover/zoom/multi-series, the MetricsTab detail panel, time range selector, and the polling data flow — wired to the backend RPCs from Phase 1.

| | |
|---|---|
| **Depends on** | Phase 1 |
| **Parallel with** | Phase 2 |

### Deliverables

- `uplot` npm dependency added (`pnpm add uplot`)
- `frontend/src/lib/components/charts/types.ts` — Frontend TypeScript types mirroring the Go types: `TimeSeriesPoint`, `TimeSeries`, `MetricResult`, `ThresholdLine`, `Annotation`, `MetricsResponse`, `MetricsCapability`
- `frontend/src/lib/components/charts/MetricsChart.svelte` — Core uPlot wrapper:
  - Creates uPlot in `$effect`, destroys on cleanup
  - Multi-series support with color-coded lines and legend toggle
  - Cursor crosshair with tooltip showing all series values at hover point
  - Min/max display for visible range
  - Click-drag to zoom, double-click to reset; zoom callback to parent for re-fetching at higher resolution
  - ResizeObserver for responsive sizing
  - Unit formatting (millicores/cores, bytes/MiB/GiB) via axis format functions
  - Loading skeleton state
- `frontend/src/lib/components/charts/TimeRangeSelector.svelte` — Preset buttons (15m, 1h, 6h, 24h, 7d), hidden when only metrics-server is available
- `frontend/src/lib/components/charts/MetricsTab.svelte` — Detail drawer tab layout:
  - Time range selector + source indicator (metrics-server / prometheus)
  - Aggregate CPU chart (all containers overlaid)
  - Aggregate Memory chart (all containers overlaid)
  - Per-container section with individual CPU and memory charts
  - Polls at correct interval (15s for short range, 60s for 6h+), cleanup on unmount
  - Graceful degradation: when only metrics-server, hides range selector, shows "live only" indicator, accumulates rolling data in component state
  - When no source available, tab is not shown
- `MetricsTab` integrated into `ResourceDetailPage` for `core.v1.pods`, `core.v1.nodes`, `apps.v1.deployments`, `apps.v1.statefulsets`, `apps.v1.daemonsets`
- `frontend/src/lib/components/charts/units.ts` — Unit formatting utilities: `formatCPU(cores)` → "500m" / "2.5", `formatMemory(bytes)` → "128 MiB" / "4.2 GiB", `formatRatio(r)` → "45%"

### Tests

- **Frontend test (vitest)**
  - `MetricsChart` mounts and creates a uPlot instance with correct series count
  - `MetricsChart` calls `uPlot.setData()` on data prop change (not full recreate)
  - `MetricsChart` destroys uPlot instance on unmount
  - `TimeRangeSelector` emits correct range value on button click
  - `TimeRangeSelector` is hidden when capability has no Prometheus
  - `MetricsTab` calls `GetResourceMetrics` on mount and sets up polling interval
  - `MetricsTab` clears interval on unmount
  - `MetricsTab` is not rendered when `GetCapabilities` returns no sources
  - Unit formatting: `formatCPU(0.5)` → "500m", `formatCPU(2.5)` → "2.5", `formatMemory(1073741824)` → "1 GiB"
- **Manual verification**
  - Chart renders with visible data points for a pod with metrics-server
  - Hover shows crosshair and tooltip with values
  - Click-drag zoom narrows the visible range, double-click resets
  - Legend toggle hides/shows individual series

### Out of Scope

- Threshold overlay lines (Phase 4)
- Annotation markers (Phase 4)
- Sparkline component (Phase 5)
- Plugin metrics section in MetricsTab (Phase 6)
- Namespace-level metrics tab (deferred to Phase 4 alongside aggregation queries)

### Acceptance Criteria

- [ ] `MetricsChart` renders a visible time-series line for at least one data series
- [ ] Hover shows crosshair + tooltip with all series values and min/max for visible range
- [ ] Click-drag zoom works; double-click resets to full range
- [ ] Multi-series renders each container in a different color with toggleable legend
- [ ] Time range selector switches between presets, triggering data refetch
- [ ] Polling starts on mount at correct interval and stops on unmount (no leaked intervals)
- [ ] Graceful degradation: metrics-server only → no range selector, rolling live data; no sources → tab hidden
- [ ] Per-container section shows individual charts below the aggregate overlay
- [ ] All frontend unit tests pass

### Handoff Notes

- uPlot is imperative — the Svelte wrapper must use `$effect` for lifecycle, not reactive declarations. Key pattern: create in `$effect` with a container ref, update via `uPlot.setData()`, destroy via `uPlot.destroy()` in the effect cleanup. Never recreate the chart just because data changed.
- The `MetricsChart` component accepts `thresholds` and `annotations` props but ignores them in this phase. Phase 4 adds the rendering hooks for these. The props should exist now so Phase 4 doesn't change the component API.
- Polling uses `setInterval` inside `$effect`, gated on `metricsCapability` being non-null. Use `untrack()` around any state reads that shouldn't re-trigger the effect (see cerebrum.md Do-Not-Repeat about `$effect` + `$state` loops).
- `MetricsTab` is registered as a detail panel. Follow existing pattern in `ResourceDetailPage` for conditionally showing tabs based on capability.

---

## Phase 4 — Thresholds, Annotations & Namespace Metrics

> Adds request/limit overlay lines, OOMKill/throttle/event annotation markers to charts, and the namespace-level aggregated metrics view.

| | |
|---|---|
| **Depends on** | Phase 2, Phase 3 |
| **Parallel with** | nothing |

### Deliverables

- **Threshold rendering in `MetricsChart.svelte`** — Horizontal dashed lines for requests/limits drawn via uPlot `drawSeries` hook. Labeled on the Y axis. Color-coded: request = blue dashed, limit = red dashed.
- **Annotation rendering in `MetricsChart.svelte`** — Vertical markers drawn via uPlot `drawAxes` hook. Color-coded by severity: error = red, warning = amber, info = blue. Hover on marker shows label tooltip.
- **`MetricsService.collectAnnotations`** (Go) — Gathers annotations from three sources:
  1. OOMKill from pod status (`containerStatuses[].lastState.terminated.reason == "OOMKilled"`)
  2. CPU throttling from Prometheus (`container_cpu_cfs_throttled_periods_total` rate > 0.5) — skipped if metric unavailable
  3. Warning/Error events from Kubernetes events API (filtered by `involvedObject.name`)
- **Threshold queries** — `GetResourceMetrics` now also executes `*:thresholds` queries from `BuiltinQueries` (kube-state-metrics `kube_pod_container_resource_requests/limits`). Falls back to reading requests/limits from pod spec when KSM is unavailable.
- **`MetricsResponse` fully populated** — `Thresholds` and `Annotations` fields now returned by `GetResourceMetrics`
- **Namespace metrics tab** — `MetricsTab` integrated into namespace overview, using `GetNamespaceMetrics` with Prometheus aggregation queries (`sum by namespace`)
- **KSM availability detection** — probe for `kube_pod_container_resource_requests` metric on connect; if absent, set a flag so threshold queries fall back to pod spec

### Tests

- **Go unit test**
  - `collectAnnotations` extracts OOMKill events from pod status correctly
  - `collectAnnotations` maps warning events to annotations with correct severity
  - Throttling annotation is skipped when Prometheus is unavailable (no error, just empty)
  - Threshold query returns request/limit series from Prometheus response
  - Threshold fallback reads requests/limits from pod spec when KSM unavailable
  - `GetNamespaceMetrics` returns aggregated CPU and memory
- **Frontend test (vitest)**
  - `MetricsChart` renders threshold lines when `thresholds` prop is provided
  - `MetricsChart` renders annotation markers when `annotations` prop is provided
  - Annotation tooltip appears on marker hover
- **Manual verification**
  - Request/limit lines visible on CPU and memory charts for a pod with defined requests/limits
  - OOMKill marker visible on graph after a container OOMKills
  - Warning events (e.g. `BackOff`, `FailedScheduling`) appear as amber markers
  - Namespace metrics tab shows aggregated CPU/memory

### Out of Scope

- Sparklines (Phase 5)
- Plugin metrics (Phase 6)

### Acceptance Criteria

- [ ] CPU charts show request line (blue dashed) and limit line (red dashed) when data is available
- [ ] Memory charts show request and limit lines
- [ ] Thresholds are time-varying when sourced from Prometheus/KSM, constant when from pod spec
- [ ] OOMKill annotations render as red vertical markers with "OOMKilled" label
- [ ] Warning events render as amber vertical markers with event reason as label
- [ ] CPU throttling annotations render when the metric exists, silently skipped when it doesn't
- [ ] Hovering an annotation marker shows a tooltip with label and timestamp
- [ ] Namespace overview has a metrics tab with aggregated CPU/memory charts
- [ ] All unit tests pass

### Handoff Notes

- Threshold fallback logic: check if `kube_pod_container_resource_requests` returns data for any pod. If empty, assume KSM is not installed and fall back to pod spec for all threshold queries in that cluster. Cache this detection per cluster context (re-checked on `RedetectSources`).
- CFS throttling metric (`container_cpu_cfs_throttled_periods_total`) may not exist on all clusters. The query should fail gracefully (empty result, no error propagation).
- Annotation drawing in uPlot uses the `drawAxes` hook which fires on every frame during zoom/pan. Keep the annotation render path fast — pre-sort annotations by timestamp once on data change, binary search for visible range.

---

## Phase 5 — Sparklines in Resource Lists

> Adds opt-in sparkline columns to pod and node list views, backed by efficient batch queries, with a resource count cap for performance.

| | |
|---|---|
| **Depends on** | Phase 4 |
| **Parallel with** | nothing |

### Deliverables

- `frontend/src/lib/components/charts/Sparkline.svelte` — Minimal uPlot instance: no axes, no labels, no cursor. Fixed 20px height, width fills column. Area fill with stroke. Single series.
- `MetricsService.GetListMetrics` (Go) — Returns `map[string][]MetricResult` (resource name → CPU + memory sparkline data). Uses `sparkline:*` queries from `BuiltinQueries` for Prometheus, or lists all PodMetrics/NodeMetrics for metrics-server. Returns error if >200 resources (frontend handles this).
- **Sparkline columns in `ResourceListPage`** — Two new optional columns (`CPU Sparkline`, `Memory Sparkline`) for `core.v1.pods` and `core.v1.nodes`. Hidden by default, toggled via existing column visibility UI.
- **Batch query polling** — When sparkline columns are enabled, `ResourceListPage` starts a 15s poll via `GetListMetrics`. Poll stops when columns are hidden or component unmounts.
- **Resource count cap** — If namespace has >200 resources of the queried type, sparkline columns show a "Too many resources" tooltip instead of charts. `GetListMetrics` returns an error/flag for this case.
- **metrics-server sparkline path** — When only metrics-server is available, sparklines show a single current-value bar (not a line), since there's no history to graph. Accumulates points while the list view is open.

### Tests

- **Go unit test**
  - `GetListMetrics` returns data keyed by pod name for a namespace
  - `GetListMetrics` returns error/flag when resource count exceeds 200
  - `GetListMetrics` uses sparkline-specific PromQL (aggregated by pod/node)
  - `GetListMetrics` falls back to PodMetrics list for metrics-server
- **Frontend test (vitest)**
  - `Sparkline` renders a canvas element with correct dimensions
  - `Sparkline` destroys uPlot on unmount
  - Sparkline columns hidden by default, visible after toggle
  - Sparkline column shows tooltip when resource count exceeds cap
- **Manual verification**
  - Enable CPU sparkline column in pod list → tiny area charts appear per row
  - Sparklines update every 15s
  - Navigate away and back → polling restarts cleanly, no duplicates
  - List with >200 pods → sparkline columns show "Too many resources" message

### Out of Scope

- Plugin metric sparklines (not planned — plugins only extend detail tab charts)
- Sparklines for resource types other than pods and nodes (could be added later by extending `sparkline:*` queries)

### Acceptance Criteria

- [ ] Sparkline columns appear in column toggle for pod and node list views
- [ ] Sparklines render as tiny area charts (~20px height) with correct data per row
- [ ] Batch query fires once per poll cycle (not per row)
- [ ] Polling starts when sparkline column is enabled, stops when disabled or unmounted
- [ ] >200 resources shows "Too many resources" tooltip, no query fires
- [ ] metrics-server only: sparklines show single bar or accumulating points
- [ ] All unit tests pass

### Handoff Notes

- Sparkline data is keyed by resource name in the `GetListMetrics` response. The frontend matches these keys to the resource list items. If a resource appears in the list but not in the metrics response (e.g. just-created pod), show an empty sparkline, not an error.
- The 200-resource cap is checked server-side before executing the query. The frontend should also check client-side (via the list item count) to avoid even making the RPC when clearly over the cap.
- uPlot sparkline instances are lightweight but not free. With 200 rows, that's 400 canvas elements (CPU + memory). TanStack Virtual already virtualizes the rows, so only visible sparklines are mounted — this should be fine, but verify scroll performance with a full 200-row list.

---

## Phase 6 — Plugin Metric Templates

> Enables plugins to register additional PromQL queries per GVR that render as extra charts in the metrics tab, using the existing chart infrastructure.

| | |
|---|---|
| **Depends on** | Phase 4 |
| **Parallel with** | Phase 5 |

### Deliverables

- **Plugin descriptor `metrics` field** — Extend the plugin manifest schema to accept `metrics: [{ gvr, queries: [{ name, query, unit }] }]`. Schema validation via existing JSON Schema pipeline.
- **`MetricsService.GetPluginMetrics`** (Go) — Loads plugin-registered queries from `PluginRegistry`, performs template variable substitution, executes via Prometheus provider, returns `[]MetricResult` grouped by plugin name.
- **Plugin query loading in `PluginRegistry`** — On plugin load/reload, extract `metrics` from descriptor and register queries. On plugin deactivate/uninstall, unregister. Follow existing pattern for sidebar/commands/enrichers.
- **Plugin metrics section in `MetricsTab.svelte`** — Rendered below the built-in charts. Grouped by plugin name with a header. Each plugin query renders as a `MetricsChart`. Only shown when Prometheus is available (plugin queries are PromQL).
- **Template variable substitution** — `{{namespace}}`, `{{name}}`, `{{node}}`, `{{container}}` replaced from the current resource context before query execution. Unknown variables left as-is (query will likely return empty, not error).

### Tests

- **Go unit test**
  - Plugin descriptor with `metrics` field parses correctly
  - Plugin metric queries are registered in `PluginRegistry` on load
  - Plugin metric queries are unregistered on deactivate
  - Template variable substitution works for all four variables
  - `GetPluginMetrics` returns results grouped by plugin source name
  - `GetPluginMetrics` returns empty (not error) when Prometheus is unavailable
- **Frontend test (vitest)**
  - Plugin metrics section renders when `GetPluginMetrics` returns results
  - Plugin metrics section hidden when no plugin queries exist
  - Plugin metrics grouped under plugin name header
- **Manual verification**
  - Install a test plugin with `metrics` descriptor → extra charts appear in pod metrics tab
  - Reload plugin → charts update
  - Disable plugin → charts disappear

### Out of Scope

- PromQL editor / custom user queries (future feature, clean seam exists via the same template system)
- Plugin-provided sparklines (detail tab only)
- Plugin metric queries for resource types not currently showing a metrics tab

### Acceptance Criteria

- [ ] Plugin descriptor YAML with `metrics` field validates and loads correctly
- [ ] Plugin metric queries execute via Prometheus and render as charts in the metrics tab
- [ ] Charts appear grouped under a plugin name header, visually distinct from built-in charts
- [ ] Plugin reload updates metric queries without restart
- [ ] Plugin deactivation removes its metric charts
- [ ] Missing Prometheus → plugin metrics section hidden (not errored)
- [ ] Template variables substituted correctly in plugin PromQL
- [ ] All unit tests pass

### Handoff Notes

- The plugin metric template system uses the same `{{var}}` substitution as built-in queries in `queries.go`. If a future PromQL editor is added (Option B from the brainstorm), it should reuse this substitution logic.
- Plugin queries are Prometheus-only. There is no metrics-server equivalent for arbitrary custom metrics. This is by design — if Prometheus isn't available, plugin metrics simply don't render.
- The manifest schema addition (`metrics` field) should be optional — existing plugins without it continue to work unchanged. Follow the pattern of other optional descriptor fields (e.g., `commands`, `enrichers`).
