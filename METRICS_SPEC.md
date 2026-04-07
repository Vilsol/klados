# Metrics System

## Context

Klados needs resource metrics (CPU, memory) for nodes, pods, and containers — displayed as interactive graphs in detail views and optional sparklines in list views. Two data sources are supported: the Kubernetes metrics-server API (live snapshots only) and any Prometheus-compatible endpoint (historical range queries). Klados stores no metric data itself; all persistence comes from the upstream source. The system must gracefully degrade: metrics-server only → live gauges and rolling sparklines; Prometheus available → full historical graphs with time range selection; neither → metrics tab hidden entirely.

Plugin authors can register additional metric queries per GVR via descriptor templates, without Klados needing a general-purpose PromQL editor.

## Decisions

**Dual-source architecture with graceful degradation**
metrics-server provides ubiquitous baseline coverage (most clusters have it). Prometheus provides depth. Rather than picking one, the system layers them: metrics-server for current values and real-time accumulation, Prometheus for historical range queries. The UI adapts based on what's available — never shows empty graphs, just hides or simplifies panels that lack a data source.

**No local metric storage**
Klados does not persist metric samples. Component-local state accumulates a rolling window while mounted (lost on navigation). This keeps the architecture simple and avoids stale data, cache invalidation, and disk usage concerns. Historical depth is exclusively a Prometheus feature.

**Plugin-extensible metric queries via descriptor templates**
Plugins declare PromQL templates in their descriptors. The metrics tab renders them alongside built-in queries using the same chart infrastructure. This gives plugin authors full Prometheus expressiveness without Klados needing a query editor UI. Variable substitution covers resource identity (pod, namespace, node, container).

**Requests/limits overlays queried from Prometheus**
Rather than reading current pod spec values (which are point-in-time), requests and limits are queried from Prometheus (`kube_state_metrics`) so the overlay reflects historical changes. Falls back to pod spec values when only metrics-server is available.

**Event annotations on graphs**
OOMKill, CPU throttling, and warning/error Kubernetes events are rendered as vertical markers on time-series graphs. OOMKill comes from pod status, throttling from `container_cpu_cfs_throttled_periods_total`, and events from the existing event timeline. Gives immediate visual correlation between metric anomalies and cluster events.

## Rejected Alternatives

**Local metric storage (SQLite / in-memory ring buffer)**
Would enable historical graphs without Prometheus, but adds significant complexity: storage management, retention policies, data migration, disk usage. The value proposition is weak — if users need history, they already have or should set up Prometheus.

**Grafana embedding / iframe**
Would reuse existing dashboards but breaks the native desktop experience, requires Grafana to be running and accessible, and offers no offline capability. Klados should own its rendering.

**ECharts / Chart.js for graphing**
ECharts is too heavy (~300KB+ tree-shaken). Chart.js is adequate but slower with high point counts and less suited to time-series-specific interactions (crosshair cursor, select-to-zoom). uPlot is purpose-built for this exact use case.

## Library Selections

| Library | Purpose | Why chosen | Alternatives considered |
|---------|---------|------------|------------------------|
| uPlot | Time-series chart rendering | ~35KB, canvas-based, built-in zoom/cursor, handles 10k+ points, used by Grafana internally. Imperative API pairs well with Svelte 5 `$effect`. | Chart.js (heavier, less time-series-focused), ECharts (too large), Lightweight Charts (finance-oriented API) |

## Priorities & Tradeoffs

**Optimized for:** Debugging speed — a developer looking at a misbehaving pod should see CPU/memory, limits, OOMKills, and related events on one screen within seconds of opening the detail view.

**Optimized for:** Minimal configuration — auto-detection means most users get metrics without configuring anything.

**Sacrificed:** Historical depth without Prometheus — accepted tradeoff since local storage adds more complexity than value.

**Sacrificed:** Custom PromQL editor — deferred to a future phase. The plugin template system provides an escape hatch for power users who need custom queries.

## Potential Gotchas

- **metrics-server returns cumulative CPU in nanocores, Prometheus returns rate of seconds.** The metrics service must normalize both to a common unit (cores) before handing data to the UI. Document the unit contract clearly.
- **Prometheus label variance.** `container_cpu_usage_seconds_total` uses `pod` label in standard setups, but some custom relabeling configs rename it. The auto-detected queries should work with standard kube-prometheus-stack labels. Custom setups may need the manual endpoint config.
- **kube-state-metrics dependency.** Requests/limits overlay and some annotations require `kube-state-metrics` to be running alongside Prometheus. If only raw Prometheus exists without KSM, fall back to pod spec values for limits/requests.
- **CFS throttling metrics require cgroups v2 or specific cAdvisor config.** Not all clusters expose `container_cpu_cfs_throttled_periods_total`. Treat as optional — if the metric doesn't exist, skip the throttling annotation.
- **Sparkline batch queries can be expensive on large namespaces.** A namespace with 500 pods means a Prometheus query returning 500 series. Cap the sparkline query at a reasonable limit (e.g., skip sparklines if >200 pods in view) and show a "too many pods for sparklines" indicator.
- **uPlot is imperative** — no declarative Svelte bindings exist. The wrapper must carefully manage create/update/destroy lifecycle via `$effect`. Avoid recreating the chart on every data update; use `uPlot.setData()` for efficient redraws.
- **Prometheus auto-detection runs at connect time.** If Prometheus is deployed after Klados connects, the user must reconnect or manually configure the endpoint. Consider a "re-detect" button in metrics settings.

## Implementation Details

### Source Detection

Detection runs during `cluster.Manager.Connect()`, after `DiscoverResources()`:

```go
type MetricsCapability struct {
    HasMetricsServer bool
    HasPrometheus    bool
    PrometheusURL    string // empty if not detected or not configured
}
```

**metrics-server detection:** Check if `metrics.k8s.io/v1beta1` exists in the discovered API groups (already available from `DiscoverResources()`).

**Prometheus detection** (in order, first match wins):
1. Well-known service names: `prometheus`, `prometheus-server`, `prometheus-kube-prometheus-prometheus`, `prometheus-operated` — scanned in namespaces `monitoring`, `prometheus`, `observability`, `default`.
2. Prometheus Operator CRD: if `monitoring.coreos.com/v1` is in discovered API groups, list `Prometheus` custom resources and extract the service endpoint from the instance spec.

Result stored on `Connection`. Emit `metrics:{ctx}:capabilities` event with the `MetricsCapability` struct. The manual configuration endpoint (stored in Klados config per cluster context) overrides auto-detection when set.

### Backend: `internal/metrics/`

```
internal/metrics/
  service.go       -- MetricsService (Wails-bound)
  provider.go      -- MetricsProvider interface + registry
  metricsserver.go -- metrics-server provider (k8s PodMetrics/NodeMetrics API)
  prometheus.go    -- Prometheus provider (HTTP client, PromQL)
  queries.go       -- built-in PromQL templates
  types.go         -- shared types
```

#### Core Types

```go
// TimeSeriesPoint is a single (timestamp, value) sample.
type TimeSeriesPoint struct {
    Timestamp int64   `json:"t"` // unix seconds
    Value     float64 `json:"v"`
}

// TimeSeries is a labeled series of points.
type TimeSeries struct {
    Labels map[string]string `json:"labels"` // e.g. {"container": "nginx"}
    Points []TimeSeriesPoint `json:"points"`
}

// MetricResult is the response for a single metric query.
type MetricResult struct {
    Name   string       `json:"name"`   // e.g. "CPU Usage"
    Unit   string       `json:"unit"`   // e.g. "cores", "bytes", "req/s"
    Series []TimeSeries `json:"series"`
}

// ThresholdLine is a horizontal overlay (requests/limits).
type ThresholdLine struct {
    Label  string            `json:"label"`  // e.g. "request", "limit"
    Series []TimeSeriesPoint `json:"series"` // time-varying from Prometheus, or constant from pod spec
}

// Annotation is a vertical event marker on the graph.
type Annotation struct {
    Timestamp int64  `json:"t"`
    Label     string `json:"label"`    // e.g. "OOMKilled", "BackOff"
    Severity  string `json:"severity"` // "error", "warning", "info"
}

// MetricsResponse is the full response for a resource's metrics tab.
type MetricsResponse struct {
    Metrics     []MetricResult  `json:"metrics"`
    Thresholds  []ThresholdLine `json:"thresholds"`
    Annotations []Annotation    `json:"annotations"`
}

// MetricQuery is a PromQL template registered by built-in code or plugins.
type MetricQuery struct {
    Name     string            `json:"name"`
    Query    string            `json:"query"`    // PromQL with {{var}} placeholders
    Unit     string            `json:"unit"`
    Vars     map[string]string `json:"vars"`     // available substitution variables
    Source   string            `json:"source"`   // "builtin" or plugin name
}
```

#### MetricsProvider Interface

```go
type MetricsProvider interface {
    // QueryRange returns time-series data for a time range.
    // Only implemented by Prometheus provider.
    QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]TimeSeries, error)

    // QueryInstant returns current metric values.
    // Implemented by both providers.
    QueryInstant(ctx context.Context, resourceType string, namespace string, name string) (*MetricsResponse, error)

    // Available returns true if this provider is operational.
    Available() bool

    // Name returns "metrics-server" or "prometheus".
    Name() string
}
```

#### MetricsService (Wails-bound)

```go
type MetricsService struct {
    clusterMgr *cluster.Manager
    providers  map[string]*providerSet // keyed by cluster context name
}

type providerSet struct {
    metricsServer MetricsProvider // may be nil
    prometheus    MetricsProvider // may be nil
}

// GetCapabilities returns what metric sources are available for a cluster.
func (s *MetricsService) GetCapabilities(clusterCtx string) MetricsCapability

// GetResourceMetrics returns metrics for a specific resource.
// Prefers Prometheus for range queries, falls back to metrics-server for instant.
func (s *MetricsService) GetResourceMetrics(clusterCtx, gvr, namespace, name string, rangeMinutes int) (*MetricsResponse, error)

// GetNamespaceMetrics returns aggregated metrics for a namespace.
func (s *MetricsService) GetNamespaceMetrics(clusterCtx, namespace string, rangeMinutes int) (*MetricsResponse, error)

// GetListMetrics returns sparkline data for all resources of a type in a namespace.
// Returns map of resource name → mini MetricResult (CPU + memory only, sparse points).
func (s *MetricsService) GetListMetrics(clusterCtx, gvr, namespace string) (map[string][]MetricResult, error)

// SetPrometheusEndpoint manually configures a Prometheus-compatible endpoint.
func (s *MetricsService) SetPrometheusEndpoint(clusterCtx, url string) error

// RedetectSources re-runs auto-detection for the given cluster.
func (s *MetricsService) RedetectSources(clusterCtx string) (*MetricsCapability, error)

// GetPluginMetrics returns results for plugin-registered metric queries.
func (s *MetricsService) GetPluginMetrics(clusterCtx, gvr, namespace, name string, rangeMinutes int) ([]MetricResult, error)
```

### Built-in PromQL Queries

```go
var BuiltinQueries = map[string][]MetricQuery{
    // Pod CPU & Memory
    "core.v1.pods": {
        {Name: "CPU Usage", Query: `sum(rate(container_cpu_usage_seconds_total{namespace="{{namespace}}", pod="{{name}}"}[5m])) by (container)`, Unit: "cores"},
        {Name: "Memory Usage", Query: `sum(container_memory_working_set_bytes{namespace="{{namespace}}", pod="{{name}}"}) by (container)`, Unit: "bytes"},
        {Name: "CPU Throttling", Query: `rate(container_cpu_cfs_throttled_periods_total{namespace="{{namespace}}", pod="{{name}}"}[5m]) / rate(container_cpu_cfs_periods_total{namespace="{{namespace}}", pod="{{name}}"}[5m])`, Unit: "ratio"},
    },

    // Pod thresholds (from kube-state-metrics)
    "core.v1.pods:thresholds": {
        {Name: "CPU Request", Query: `kube_pod_container_resource_requests{namespace="{{namespace}}", pod="{{name}}", resource="cpu"}`, Unit: "cores"},
        {Name: "CPU Limit", Query: `kube_pod_container_resource_limits{namespace="{{namespace}}", pod="{{name}}", resource="cpu"}`, Unit: "cores"},
        {Name: "Memory Request", Query: `kube_pod_container_resource_requests{namespace="{{namespace}}", pod="{{name}}", resource="memory"}`, Unit: "bytes"},
        {Name: "Memory Limit", Query: `kube_pod_container_resource_limits{namespace="{{namespace}}", pod="{{name}}", resource="memory"}`, Unit: "bytes"},
    },

    // Node CPU & Memory
    "core.v1.nodes": {
        {Name: "CPU Usage", Query: `sum(rate(node_cpu_seconds_total{mode!="idle", node="{{name}}"}[5m]))`, Unit: "cores"},
        {Name: "Memory Usage", Query: `node_memory_MemTotal_bytes{node="{{name}}"} - node_memory_MemAvailable_bytes{node="{{name}}"}`, Unit: "bytes"},
    },

    // Deployments, StatefulSets, DaemonSets — aggregate across owned pods
    "apps.v1.deployments": {
        {Name: "CPU Usage", Query: `sum(rate(container_cpu_usage_seconds_total{namespace="{{namespace}}", pod=~"{{name}}-[a-z0-9]+-[a-z0-9]+"}[5m])) by (pod)`, Unit: "cores"},
        {Name: "Memory Usage", Query: `sum(container_memory_working_set_bytes{namespace="{{namespace}}", pod=~"{{name}}-[a-z0-9]+-[a-z0-9]+"}) by (pod)`, Unit: "bytes"},
    },

    // Namespace aggregation
    "namespace": {
        {Name: "CPU Usage", Query: `sum(rate(container_cpu_usage_seconds_total{namespace="{{namespace}}"}[5m]))`, Unit: "cores"},
        {Name: "Memory Usage", Query: `sum(container_memory_working_set_bytes{namespace="{{namespace}}"})`, Unit: "bytes"},
    },

    // Sparkline batch queries (used by list views)
    "sparkline:core.v1.pods": {
        {Name: "CPU", Query: `sum(rate(container_cpu_usage_seconds_total{namespace="{{namespace}}"}[5m])) by (pod)`, Unit: "cores"},
        {Name: "Memory", Query: `sum(container_memory_working_set_bytes{namespace="{{namespace}}"}) by (pod)`, Unit: "bytes"},
    },
    "sparkline:core.v1.nodes": {
        {Name: "CPU", Query: `sum(rate(node_cpu_seconds_total{mode!="idle"}[5m])) by (node)`, Unit: "cores"},
        {Name: "Memory", Query: `(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes)`, Unit: "bytes"},
    },
}
```

### Plugin Metric Templates

Plugins register metric queries in their descriptor YAML:

```yaml
metrics:
  - gvr: core.v1.pods
    queries:
      - name: "HTTP Request Rate"
        query: 'sum(rate(http_requests_total{namespace="{{namespace}}", pod="{{name}}"}[5m])) by (code)'
        unit: "req/s"
      - name: "HTTP Error Rate"
        query: 'sum(rate(http_requests_total{namespace="{{namespace}}", pod="{{name}}", code=~"5.."}[5m]))'
        unit: "req/s"
```

Template variables available for substitution:

| Variable | Description | Available for |
|----------|-------------|---------------|
| `{{namespace}}` | Resource namespace | All namespaced resources |
| `{{name}}` | Resource name | All resources |
| `{{node}}` | Node name | Nodes, Pods (from pod spec) |
| `{{container}}` | Container name | Container-level queries |

Plugin queries are loaded via `PluginRegistry` and passed to `MetricsService` alongside built-in queries. They render as additional charts in the metrics tab, visually grouped under the plugin name.

### Prometheus HTTP Client

```go
type PrometheusClient struct {
    baseURL    string
    httpClient *http.Client // uses cluster's rest.Config transport for in-cluster access
}

// QueryRange calls /api/v1/query_range.
func (c *PrometheusClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]TimeSeries, error)

// QueryInstant calls /api/v1/query.
func (c *PrometheusClient) QueryInstant(ctx context.Context, query string) ([]TimeSeries, error)
```

For in-cluster Prometheus, the HTTP client reuses the cluster's `rest.Config` transport (handles auth, TLS). For manually configured external endpoints, use a plain HTTP client (the user is responsible for accessibility).

### Time Range Presets & Step Sizes

| Preset | Duration | Step | ~Points |
|--------|----------|------|---------|
| 15m | 15 min | 15s | 60 |
| 1h | 1 hour | 15s | 240 |
| 6h | 6 hours | 1m | 360 |
| 24h | 24 hours | 5m | 288 |
| 7d | 7 days | 30m | 336 |

### Poll Intervals

| Context | Interval | Source |
|---------|----------|--------|
| Detail tab (short range: 15m, 1h) | 15s | Either |
| Detail tab (long range: 6h+) | 60s | Prometheus only |
| Sparklines in list views | 15s | Either (batch query) |

### Frontend: uPlot Wrapper

```
frontend/src/lib/components/charts/
  MetricsChart.svelte     -- main interactive chart (uPlot wrapper)
  Sparkline.svelte        -- tiny inline chart for list columns
  MetricsTab.svelte       -- detail tab layout (multiple charts + controls)
  TimeRangeSelector.svelte -- preset buttons
  types.ts                -- frontend metric types
```

#### MetricsChart.svelte

Core uPlot wrapper responsibilities:
- Create uPlot instance in `$effect`, destroy on cleanup
- Multi-series support: one series per container/pod, color-coded with legend toggle
- Threshold lines: horizontal dashed lines for requests/limits (drawn via uPlot hooks)
- Annotations: vertical markers for OOMKill/throttle/events (drawn via uPlot `drawAxes` hook)
- Cursor plugin: crosshair with tooltip showing all series values at hover point, plus min/max for visible range
- Zoom: click-drag to select range, double-click to reset. Zoom fires callback to parent for fetching higher-resolution data within the zoomed range.
- Responsive: resize observer updates chart dimensions
- Unit formatting: auto-scale (millicores, cores; bytes, MiB, GiB) via format function passed to uPlot axis

```typescript
interface MetricsChartProps {
  title: string;
  unit: string;          // "cores", "bytes", "ratio", "req/s"
  series: TimeSeries[];
  thresholds?: ThresholdLine[];
  annotations?: Annotation[];
  loading?: boolean;
  height?: number;       // default 200
}
```

#### Sparkline.svelte

Minimal uPlot instance: no axes, no labels, no cursor. Fixed height (20px), width fills column. Area fill with stroke. Single series, last 5-15 minutes of data.

```typescript
interface SparklineProps {
  points: TimeSeriesPoint[];
  color?: string;   // default: accent
  width?: number;
  height?: number;  // default: 20
}
```

Sparkline columns are opt-in via column toggle in list views. Hidden by default. When enabled, `ResourceListPage` starts a 15s poll via `GetListMetrics()`. If the namespace contains >200 resources, sparklines are disabled with a tooltip explaining why.

#### MetricsTab.svelte

Layout for the detail drawer metrics tab:

```
[TimeRangeSelector: 15m | 1h | 6h | 24h | 7d]   [Source indicator: metrics-server | prometheus]

[CPU Usage chart - all containers overlaid]
[Memory Usage chart - all containers overlaid]

[Per-container section]
  [Container: nginx]
    [CPU chart] [Memory chart]
  [Container: sidecar]
    [CPU chart] [Memory chart]

[Plugin metrics section - if any]
  [Plugin: istio-metrics]
    [HTTP Request Rate chart]
    [HTTP Error Rate chart]
```

When only metrics-server is available:
- Time range selector hidden
- Charts show rolling real-time data (accumulated while tab is open)
- Source indicator shows "metrics-server (live only)"
- Thresholds show current pod spec values (constant lines)
- No throttling annotation (requires Prometheus)

### Annotation Collection

Annotations are gathered from multiple sources and merged by timestamp:

```go
func (s *MetricsService) collectAnnotations(ctx context.Context, clusterCtx, namespace, name string, start, end time.Time) ([]Annotation, error) {
    var annotations []Annotation

    // 1. OOMKill — from pod status (already available via ResourceEngine)
    // Check containerStatuses[].lastState.terminated.reason == "OOMKilled"

    // 2. CPU throttling — from Prometheus (if available)
    // Query: rate(container_cpu_cfs_throttled_periods_total{...}[1m]) > 0.5
    // Each result timestamp becomes an annotation

    // 3. Warning/Error events — from Kubernetes events API
    // Filter: involvedObject.name == name, type in ("Warning")
    // Map each event to an annotation with event.reason as label

    return annotations, nil
}
```

### Wails Events

| Event | Payload | When |
|-------|---------|------|
| `metrics:{ctx}:capabilities` | `MetricsCapability` | On connect, on re-detect, on manual config |

Metric data is fetched via RPC (not events) since it's request-response, not streaming. The frontend polls on the intervals defined above using `setInterval` inside `$effect`, cleaned up on unmount.

### Configuration

Stored in Klados config (`config.json`) per cluster context:

```go
type MetricsConfig struct {
    PrometheusURL string `json:"prometheusUrl,omitempty"` // manual override, empty = auto-detect
}
```

Accessible via a "Metrics" section in cluster settings (or a configure button on the metrics tab when auto-detection fails).

## Definition of Done

- [ ] metrics-server detection works automatically on connect; `PodMetrics` and `NodeMetrics` queries return data
- [ ] Prometheus auto-detection finds kube-prometheus-stack and bare Prometheus installs
- [ ] Manual Prometheus endpoint configuration works and persists across sessions
- [ ] Any Prometheus-compatible endpoint (Thanos, Mimir, VictoriaMetrics) works via manual config
- [ ] Node detail view shows CPU and memory graphs
- [ ] Pod detail view shows per-container and aggregate CPU/memory graphs
- [ ] Deployment/StatefulSet/DaemonSet detail views show aggregate metrics across owned pods
- [ ] Namespace overview shows aggregated CPU/memory
- [ ] Graphs show requests/limits as horizontal overlays (from Prometheus/KSM or pod spec fallback)
- [ ] OOMKill, throttling, and warning events appear as annotations on graphs
- [ ] uPlot charts support hover with tooltip (all series values + min/max), click-drag zoom, double-click reset
- [ ] Time range presets (15m, 1h, 6h, 24h, 7d) work with correct step sizes
- [ ] Graceful degradation: metrics-server only shows live rolling data; no source hides the tab entirely
- [ ] Sparkline columns available in pod and node list views (opt-in toggle, batch query, capped at 200 resources)
- [ ] Plugin metric queries render in the metrics tab alongside built-in charts
- [ ] Polls at correct intervals (15s for short range, 60s for long range) and cleans up on unmount
- [ ] Re-detect button available when auto-detection misses a Prometheus instance
