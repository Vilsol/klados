# Phase 4 — Thresholds, Annotations & Namespace Metrics

Add request/limit overlay lines and OOMKill/throttle/event annotation markers to the existing uPlot charts, plus the namespace-level aggregated metrics view.

## First Action

Read `frontend/src/lib/components/charts/MetricsChart.svelte` to understand the current uPlot setup — you'll add two uPlot hooks (`drawSeries` for threshold lines, `drawAxes` for annotation markers) to the existing chart options.

## Context

Phase 2 delivered the Prometheus provider with built-in PromQL templates (including `*:thresholds` queries for kube-state-metrics). Phase 3 delivered the uPlot chart wrapper and MetricsTab with `thresholds` and `annotations` props that accept data but render nothing. This phase connects both: the backend collects threshold/annotation data and the frontend renders them as visual overlays. It also adds the namespace-level metrics tab.

## Files to Read

- `frontend/src/lib/components/charts/MetricsChart.svelte` — **what to look for**: existing uPlot options object, the `$effect` lifecycle, and where to insert `drawSeries` and `drawAxes` hooks. Also check that `thresholds` and `annotations` props exist (added in Phase 3).
- `internal/metrics/queries.go` — **what to look for**: `"core.v1.pods:thresholds"` key in `BuiltinQueries` — these are the KSM queries for requests/limits that this phase executes
- `internal/metrics/service.go` — **what to look for**: `GetResourceMetrics` method — you'll add threshold and annotation collection to the response
- `internal/metrics/types.go` — **what to look for**: `ThresholdLine` and `Annotation` structs — verify the structure matches what the frontend expects
- `frontend/src/lib/components/charts/MetricsTab.svelte` — **what to look for**: how `thresholds` and `annotations` are passed to `MetricsChart` — ensure the data flow is wired up
- `frontend/src/routes/ClusterOverview.svelte` — **what to look for**: namespace overview page structure — you may need to add a metrics section or tab here
- `METRICS_SPEC.md` — **what to look for**: Annotation collection logic (lines 400–417), threshold fallback to pod spec (gotcha on line 57)

## What Exists

- `MetricsChart.svelte` — working uPlot wrapper with multi-series, cursor, zoom. Accepts `thresholds: ThresholdLine[]` and `annotations: Annotation[]` props but does not render them yet.
- `MetricsTab.svelte` — detail tab with time range selector, aggregate + per-container charts, polling
- `PrometheusClient` with `QueryRange` and `QueryInstant`
- `BuiltinQueries` including `"core.v1.pods:thresholds"` (KSM requests/limits queries) and `"namespace"` (aggregation queries)
- `MetricsService.GetResourceMetrics` returning `MetricsResponse` with `Thresholds: []` and `Annotations: []` (empty)
- `MetricsService.GetNamespaceMetrics` aggregating via metrics-server; Prometheus path may exist from Phase 2
- `ResourceEngine` and Events API accessible for annotation collection

## Deliverables

1. **Threshold rendering in `MetricsChart.svelte`** — uPlot `drawSeries` hook draws horizontal dashed lines for each `ThresholdLine`. Request lines = blue dashed, Limit lines = red dashed. Label text drawn on the Y axis side. Lines are per-series (one per container for requests/limits). Time-varying thresholds from Prometheus render as step-lines, not straight horizontals.
2. **Annotation rendering in `MetricsChart.svelte`** — uPlot `drawAxes` hook draws vertical lines at each `Annotation.Timestamp`. Color by severity: error=red (`--destructive`), warning=amber, info=blue (`--accent`). Hover on a marker shows tooltip with `label` and formatted timestamp. Pre-sort annotations on data change; binary-search visible range during draw for performance.
3. **`collectAnnotations` in `internal/metrics/service.go`** — gathers annotations from:
   - OOMKill: reads pod status via `ResourceEngine`, checks `containerStatuses[].lastState.terminated.reason == "OOMKilled"`, creates error-severity annotation at termination timestamp
   - CPU throttling (Prometheus only): queries `rate(container_cpu_cfs_throttled_periods_total{...}[1m]) > 0.5`, each timestamp with throttling becomes a warning annotation. Silently returns empty if metric doesn't exist.
   - Warning/Error events: queries Kubernetes events API filtered by `involvedObject.name`, maps each Warning event to a warning-severity annotation with `event.reason` as label
4. **Threshold data in `GetResourceMetrics`** — executes `*:thresholds` PromQL queries from `BuiltinQueries` when Prometheus+KSM available. Falls back to reading `resources.requests` and `resources.limits` from pod spec (constant values) when KSM unavailable. Returns populated `Thresholds` field.
5. **KSM availability detection** — on connect (or `RedetectSources`), probe for `kube_pod_container_resource_requests` metric via instant query. Cache result per cluster context. If absent, threshold queries fall back to pod spec.
6. **Namespace metrics tab** — `MetricsTab` reused or a variant integrated into namespace overview, calling `GetNamespaceMetrics` with Prometheus aggregation queries. Shows total CPU and memory for the namespace.

## Tests

- **Go unit test**
  - `collectAnnotations` extracts OOMKill from pod status with correct timestamp and severity
  - `collectAnnotations` maps Warning events to annotations with `event.reason` as label and warning severity
  - `collectAnnotations` skips throttling when Prometheus is unavailable (returns empty, no error)
  - Threshold query returns request/limit `ThresholdLine` series from Prometheus response
  - Threshold fallback reads `spec.containers[].resources.requests.cpu` and returns constant `ThresholdLine`
  - `GetNamespaceMetrics` with Prometheus returns aggregated time-series
- **Frontend test (vitest)**
  - `MetricsChart` with non-empty `thresholds` prop triggers `drawSeries` hook (verify canvas draw calls or mock uPlot hooks)
  - `MetricsChart` with non-empty `annotations` prop triggers `drawAxes` hook
  - Annotation tooltip content matches the annotation label
- **Manual verification**
  - Pod CPU chart shows blue dashed "request" line and red dashed "limit" line
  - OOMKill annotation appears as red vertical line after deliberately OOMKilling a container
  - Warning events (e.g. `BackOff`) appear as amber vertical markers
  - Hovering annotation marker shows tooltip with reason and time
  - Namespace overview shows aggregated CPU and memory charts

## Acceptance Criteria

- [ ] CPU and memory charts show request (blue dashed) and limit (red dashed) overlay lines when data is available
- [ ] Thresholds are time-varying from Prometheus/KSM; constant from pod spec fallback
- [ ] OOMKill annotations render as red vertical markers with "OOMKilled" label
- [ ] Warning events render as amber vertical markers with event reason as label
- [ ] CPU throttling annotations render when the metric exists, silently skipped when absent
- [ ] Hovering an annotation marker shows a tooltip with label and timestamp
- [ ] Namespace overview has a metrics view with aggregated CPU/memory charts
- [ ] KSM detection is cached per cluster context and re-checked on `RedetectSources`
- [ ] All unit tests pass

## Definition of Done

Open a pod detail view with Prometheus available. The CPU chart shows the data line plus dashed request and limit overlay lines. After OOMKilling a container, a red vertical marker appears at the kill timestamp. Warning events from the pod's event timeline show as amber markers. On a cluster without KSM, limits/requests still display as constant lines from the pod spec. The namespace overview shows aggregated CPU/memory charts.

## Known Gotchas

- **uPlot `drawAxes` hook fires on every frame during zoom/pan.** The annotation renderer must be fast. Pre-sort annotations by timestamp once when data changes (not in the hook). In the hook, binary-search for the visible time range to only draw annotations currently on screen.
- **KSM may not be installed.** The `kube_pod_container_resource_requests` probe will return empty results, not an error. Detect this by checking for zero results from the instant query, not by catching errors. Cache the boolean per cluster context.
- **CFS throttling metric may not exist.** `container_cpu_cfs_throttled_periods_total` requires cgroups v2 or specific cAdvisor config. The Prometheus query will return `"resultType": "vector"` with zero results if the metric doesn't exist. Treat zero results as "no throttling data" — don't log errors.
- **Threshold lines for multi-container pods.** Each container has its own request and limit. The `ThresholdLine.Series` needs a way to associate with a specific series on the chart. Use `ThresholdLine.Label` like `"request:nginx"` or include container name in the `Labels` map for matching.
- **Pod spec fallback returns constant values.** When falling back to pod spec, create a `ThresholdLine` with two points: `[{t: start, v: value}, {t: end, v: value}]` spanning the chart's visible range. This renders as a straight horizontal line.
