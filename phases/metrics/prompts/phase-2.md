# Phase 2 — Prometheus Provider & Auto-Detection

Add the Prometheus HTTP client, auto-detection of in-cluster Prometheus instances, manual endpoint configuration, and range query support — enabling historical time-series data for all metric views.

## First Action

Read `internal/metrics/provider.go` to understand the `MetricsProvider` interface you're implementing, then read `internal/metrics/metricsserver.go` as a reference for how the first provider was structured — your Prometheus provider follows the same pattern.

## Context

Phase 1 established the shared types, `MetricsProvider` interface, and the metrics-server provider. The frontend can already fetch live snapshots. This phase adds the second provider (Prometheus) which unlocks historical range queries, auto-detects Prometheus in the cluster, and adds manual endpoint configuration. Phase 3 (uPlot charts) runs in parallel and will consume the range data this phase produces.

## Files to Read

- `internal/metrics/provider.go` — **what to look for**: `MetricsProvider` interface signatures (`QueryRange`, `QueryInstant`, `Available`, `Name`) and `providerSet` struct — implement this interface for Prometheus
- `internal/metrics/metricsserver.go` — **what to look for**: provider struct pattern (fields, constructor, method implementations) — replicate this structure
- `internal/metrics/service.go` — **what to look for**: `GetResourceMetrics` current implementation — you'll add the Prometheus path (prefer Prometheus for `rangeMinutes > 0`, fall back to metrics-server)
- `internal/metrics/types.go` — **what to look for**: `MetricQuery` struct (has `Query`, `Vars` fields) — your PromQL template system produces these
- `internal/cluster/manager.go` — **what to look for**: `Connect()` method and `DiscoverResources()` — you'll add Prometheus detection alongside metrics-server detection
- `internal/config/config.go` — **what to look for**: `MetricsConfig` struct (added in Phase 1) — `SetPrometheusEndpoint` persists the URL here
- `METRICS_SPEC.md` — **what to look for**: Prometheus detection strategy (lines 79–83), HTTP client (lines 284–297), PromQL templates (lines 208–253), time range/step table (lines 299–307)

## What Exists

- `internal/metrics/types.go` — all shared types (`TimeSeriesPoint`, `TimeSeries`, `MetricResult`, `MetricsResponse`, `MetricQuery`, `MetricsCapability`)
- `internal/metrics/provider.go` — `MetricsProvider` interface and `providerSet`
- `internal/metrics/metricsserver.go` — working metrics-server provider
- `internal/metrics/service.go` — `MetricsService` with `GetCapabilities`, `GetResourceMetrics` (metrics-server path), `GetNamespaceMetrics`
- metrics-server detection in `cluster.Manager.Connect()` — sets `HasMetricsServer` on capability
- `config.Config.Metrics` map with per-context `MetricsConfig` struct (has `PrometheusURL` field)
- `metrics:{ctx}:capabilities` event emitted on connect

## Deliverables

1. `internal/metrics/prometheus.go` — `PrometheusClient` struct with:
   - `baseURL string` and `httpClient *http.Client`
   - Constructor that accepts a URL and optional `*rest.Config` (for in-cluster transport reuse; nil = plain HTTP for external endpoints)
   - `QueryRange(ctx, query, start, end, step)` → calls `/api/v1/query_range`, parses Prometheus JSON response into `[]TimeSeries`
   - `QueryInstant(ctx, query)` → calls `/api/v1/query`, parses into `[]TimeSeries`
   - `Available()` → pings `/-/ready` or `/api/v1/status/config`
   - `Name()` → `"prometheus"`
   - 10s connect timeout, 30s response timeout
2. `internal/metrics/queries.go` — `BuiltinQueries` map (as specified in METRICS_SPEC.md): pod CPU/memory/throttling, pod thresholds (KSM), node CPU/memory, deployment aggregation, namespace aggregation, sparkline batch queries. Template variable substitution function `substituteVars(query string, vars map[string]string) string` using simple `{{var}}` string replacement.
3. Prometheus auto-detection in `cluster.Manager.Connect()`:
   - Step 1: Scan well-known service names (`prometheus`, `prometheus-server`, `prometheus-kube-prometheus-prometheus`, `prometheus-operated`) in namespaces (`monitoring`, `prometheus`, `observability`, `default`)
   - Step 2: If `monitoring.coreos.com/v1` in discovered API groups, list `Prometheus` CRs and extract service endpoint
   - First match wins. Store URL on `MetricsCapability`. Manual config overrides auto-detection.
4. `MetricsService.SetPrometheusEndpoint(clusterCtx, url)` — persists URL to `config.json` via `Config.Metrics[ctx].PrometheusURL`, rebuilds Prometheus provider, re-emits `metrics:{ctx}:capabilities`
5. `MetricsService.RedetectSources(clusterCtx)` — re-runs auto-detection, updates capability, emits event
6. `MetricsService.GetResourceMetrics` updated: when `rangeMinutes > 0` and Prometheus is available, execute the matching `BuiltinQueries` entry with `QueryRange`, using the step size table (15m→15s, 1h→15s, 6h→1m, 24h→5m, 7d→30m). Fall back to metrics-server for instant when Prometheus is unavailable.

## Tests

- **Go unit test**
  - `PrometheusClient` parses a valid `/api/v1/query_range` JSON response (matrix result type) into `[]TimeSeries` with correct timestamps and values
  - `PrometheusClient` parses `/api/v1/query` (instant vector) response correctly
  - `PrometheusClient` returns structured error on HTTP 400/500 and on malformed JSON
  - `PrometheusClient.QueryRange` sends correct `query`, `start`, `end`, `step` URL parameters
  - `substituteVars` replaces `{{namespace}}` and `{{name}}` correctly; leaves `{{unknown}}` as-is
  - Step size calculation: 15m→15s, 1h→15s, 6h→1m, 24h→5m, 7d→30m
  - Auto-detection finds service named `prometheus-server` in `monitoring` namespace via mock discovery
  - Auto-detection finds Prometheus Operator CR and extracts correct endpoint
  - Manual endpoint override (`SetPrometheusEndpoint`) takes precedence over auto-detection
  - `GetResourceMetrics` with `rangeMinutes=60` uses Prometheus `QueryRange`; with `rangeMinutes=0` uses metrics-server `QueryInstant`
- **Go unit test (HTTP mock — httptest.Server)**
  - `QueryRange` sends correct URL params and parses matrix response
  - `QueryInstant` sends correct URL params and parses vector response
  - Returns error on non-200 status codes with Prometheus error body
- **Integration test (requires cluster with Prometheus)**
  - Auto-detection discovers kube-prometheus-stack installation
  - `QueryRange` returns multi-point time-series for a known running pod

## Acceptance Criteria

- [ ] `GetCapabilities` returns `HasPrometheus=true` and populated `PrometheusURL` on a cluster with kube-prometheus-stack
- [ ] `GetResourceMetrics` with `rangeMinutes=60` returns multi-point historical time-series (not just an instant snapshot)
- [ ] `SetPrometheusEndpoint` persists URL to `config.json` and subsequent `GetCapabilities` reflects it
- [ ] `RedetectSources` re-runs detection and updates capabilities, emitting the event
- [ ] Works with any Prometheus-compatible `/api/v1/query_range` endpoint (Thanos, Mimir, VictoriaMetrics)
- [ ] Falls back to metrics-server when Prometheus is unavailable and `rangeMinutes > 0`
- [ ] Built-in PromQL templates cover pods, nodes, deployments, namespace aggregation, and sparkline batch queries
- [ ] All Go unit tests pass

## Definition of Done

After connecting to a cluster with Prometheus, `GetCapabilities` reports `HasPrometheus=true`. Calling `GetResourceMetrics("ctx", "core.v1.pods", "default", "some-pod", 60)` returns a `MetricsResponse` with multi-point `TimeSeries` (one per container) for CPU and memory over the last hour at 15s resolution. Manually setting a Prometheus URL via `SetPrometheusEndpoint` persists across app restarts and overrides auto-detection.

## Known Gotchas

- **Prometheus JSON response format varies by result type.** `query_range` returns `"resultType": "matrix"` with `values: [[timestamp, "stringValue"], ...]`. `query` (instant) returns `"resultType": "vector"` with `value: [timestamp, "stringValue"]`. Values are strings, not numbers — parse with `strconv.ParseFloat`.
- **In-cluster Prometheus may require the cluster's auth transport.** When auto-detected, build the HTTP client from `rest.Config` via `rest.TransportFor(config)` to inherit ServiceAccount tokens and TLS settings. For manually configured external URLs, use a plain `http.Client`.
- **`sparkline:*` and `*:thresholds` query keys are consumed by later phases.** Phase 5 uses `sparkline:core.v1.pods` etc., Phase 4 uses `core.v1.pods:thresholds`. Define them now in `BuiltinQueries` even though nothing calls them yet — don't remove them.
- **Template substitution is simple string replacement, not Go templates.** Use `strings.ReplaceAll(query, "{{namespace}}", ns)` — no template parsing. This keeps it trivial for plugins to author queries in Phase 6.
- **Auto-detection is best-effort and runs once at connect time.** If Prometheus is deployed after Klados connects, it won't be detected until reconnect or `RedetectSources()` is called. Phase 3 adds a "re-detect" button in the UI.
