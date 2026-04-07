# Phase 1 — Core Types, Provider Interface & metrics-server

Establish the shared metric data types, the `MetricsProvider` abstraction, and the first concrete provider (metrics-server) so the frontend has a working Wails RPC to fetch live CPU/memory snapshots for pods and nodes.

## First Action

Read `internal/cluster/manager.go` lines 74–93 and 160–200 — understand the `Connection` struct and `Connect()` flow, because you'll add metrics-server detection at the end of `Connect()` after `DiscoverResources()` completes.

## Context

This is the first phase of the metrics system. No metrics code exists yet — this phase creates `internal/metrics/` from scratch. The cluster manager already discovers API groups on connect and emits events; you'll hook into that to detect whether `metrics.k8s.io/v1beta1` is available. The `MetricsService` you build here becomes the Wails-bound RPC layer that all frontend metric views will call.

## Files to Read

- `internal/cluster/manager.go` — **what to look for**: `Connection` struct (line 74), `Connect()` method (line 160), `DiscoverResources()` (line 342), and `emitEvent` pattern — you'll add detection logic after discovery and emit `metrics:{ctx}:capabilities`
- `internal/services/app.go` — **what to look for**: `AppService` struct (line 19) and `ServiceStartup()` (line 40) — this is where services are wired up; you'll register `MetricsService` following the same pattern as existing services
- `internal/services/resource.go` — **what to look for**: how an existing Wails-bound service struct is organized (fields, constructor, method signatures) — replicate this pattern for `MetricsService`
- `internal/config/config.go` — **what to look for**: `Config` struct (line 12) — you'll add a `MetricsConfig` field here (per-cluster-context map, initially just `PrometheusURL` for Phase 2, but define the struct now)
- `internal/resource/descriptor.go` — **what to look for**: `DetailPanels` field (line 45) — you'll eventually add `"metrics"` to panel lists, but not yet (Phase 3)
- `METRICS_SPEC.md` — **what to look for**: Core Types section (lines 99–146) for exact struct definitions, MetricsProvider interface (lines 150–167), MetricsService signatures (lines 169–204)

## What Exists

- `cluster.Manager` with `Connect()`, `GetConnection()`, `DiscoverResources()` — manages cluster connections and API group discovery
- `cluster.Connection` with `Clientset` (`kubernetes.Interface`), `Dynamic`, `Discovery` interfaces
- `config.Config` at `$XDG_CONFIG_HOME/klados/config.json` with JSON serialization and mutex-protected save
- `services.AppService` as the Wails service orchestrator — owns cluster manager, registers services
- `services.ResourceService` as a reference for how Wails-bound services are structured
- Wails event emission via `emitEvent func(string, any)` passed to managers
- `slox` context-based logging throughout

## Deliverables

1. `internal/metrics/types.go` — `TimeSeriesPoint`, `TimeSeries`, `MetricResult`, `ThresholdLine`, `Annotation`, `MetricsResponse`, `MetricQuery`, `MetricsCapability` structs with JSON tags exactly as specified in METRICS_SPEC.md
2. `internal/metrics/provider.go` — `MetricsProvider` interface with `QueryRange`, `QueryInstant`, `Available`, `Name` methods; `providerSet` struct holding optional `metricsServer` and `prometheus` providers
3. `internal/metrics/metricsserver.go` — Concrete provider that calls `metrics.k8s.io/v1beta1` API via the cluster's `Clientset`. Fetches `PodMetrics` and `NodeMetrics`. Normalizes nanocores → cores (divide by 1e9) and bytes pass-through. `QueryRange` returns `ErrNotSupported`. `Available()` checks API group presence.
4. `internal/metrics/service.go` — `MetricsService` Wails-bound service with:
   - `GetCapabilities(clusterCtx string) MetricsCapability`
   - `GetResourceMetrics(clusterCtx, gvr, namespace, name string, rangeMinutes int) (*MetricsResponse, error)` — metrics-server path only; `rangeMinutes` ignored (returns instant snapshot)
   - `GetNamespaceMetrics(clusterCtx, namespace string, rangeMinutes int) (*MetricsResponse, error)` — aggregates PodMetrics across namespace
5. metrics-server detection in `cluster.Manager.Connect()` — after `DiscoverResources()`, check for `metrics.k8s.io/v1beta1` in API groups, store result on `Connection` (add `MetricsCapability` field), emit `metrics:{ctx}:capabilities` event
6. `MetricsConfig` struct added to `config.Config` as `Metrics map[string]MetricsConfig` (keyed by cluster context name) with `PrometheusURL` field (unused until Phase 2, but define now)
7. `MetricsService` registered in `AppService.ServiceStartup()` and Wails bindings regenerated

## Tests

- **Go unit test**
  - `MetricsCapability` correctly reports `HasMetricsServer=true` when `metrics.k8s.io/v1beta1` is in discovered API groups, `false` when absent
  - metrics-server provider normalizes nanocores to cores: `500_000_000` nanocores → `0.5` cores
  - metrics-server provider returns `MetricsResponse` with two `MetricResult` entries (CPU in `"cores"`, memory in `"bytes"`) for a pod
  - metrics-server `QueryRange` returns `ErrNotSupported`
  - `GetResourceMetrics` for `core.v1.pods` returns CPU and memory results with correct units
  - `GetNamespaceMetrics` sums CPU and memory across all pods returned by PodMetrics list
  - `GetCapabilities` returns `HasPrometheus=false` and empty `PrometheusURL` (no Prometheus provider yet)
- **Integration test (requires live cluster with metrics-server)**
  - `GetCapabilities` detects metrics-server on a real cluster
  - `GetResourceMetrics` returns non-zero CPU/memory values for a running pod

## Acceptance Criteria

- [ ] `MetricsService.GetCapabilities` returns `HasMetricsServer=true` on a cluster with metrics-server
- [ ] `MetricsService.GetResourceMetrics` returns CPU (`unit: "cores"`) and memory (`unit: "bytes"`) for a pod
- [ ] `MetricsService.GetNamespaceMetrics` returns aggregated CPU/memory across all pods in a namespace
- [ ] `metrics:{ctx}:capabilities` event is emitted during `Connect()` after discovery
- [ ] CPU values are always in cores (float64), memory always in bytes (float64)
- [ ] `config.json` schema includes `metrics` map with per-context `MetricsConfig`
- [ ] Wails bindings generated and `MetricsService` accessible from frontend TypeScript
- [ ] All Go unit tests pass

## Definition of Done

After connecting to a cluster with metrics-server installed, calling `MetricsService.GetResourceMetrics("ctx", "core.v1.pods", "default", "some-pod", 0)` from the frontend bindings returns a `MetricsResponse` with two `MetricResult` entries: CPU usage in cores and memory usage in bytes. `GetCapabilities` reports `HasMetricsServer=true`. No UI exists yet — this is verified via Go tests and manual binding calls.

## Known Gotchas

- **metrics-server returns CPU in nanocores (int64), not cores.** Divide by `1e9` to get cores as float64. Memory is already in bytes. If you forget the normalization, CPU values will look like `500000000` instead of `0.5`.
- **`kubernetes.Interface` for testability.** The `Connection.Clientset` is `kubernetes.Interface` (not `*kubernetes.Clientset`) — this is a deliberate decision (see cerebrum.md). Use `fake.NewSimpleClientset()` in unit tests. The metrics-server API (`PodMetrics`, `NodeMetrics`) is accessed via the metrics client (`k8s.io/metrics/pkg/client/clientset/versioned`), which also has a fake.
- **Event name format follows existing conventions.** Use `metrics:{ctx}:capabilities` (colon-separated, context name interpolated). Check how `discovery:{ctx}:resources` and `status:{ctx}:connection` are emitted in `manager.go` for the pattern.
- **`MetricsConfig` must be a map, not a single struct.** Each cluster context can have its own Prometheus URL. Key the map by context name (`map[string]MetricsConfig` on `Config`).
