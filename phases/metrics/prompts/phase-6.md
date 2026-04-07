# Phase 6 — Plugin Metric Templates

Enable plugins to register additional PromQL queries per GVR that render as extra charts in the metrics tab, using the existing chart infrastructure.

## First Action

Read `internal/plugin/manifest.go` (or wherever the plugin manifest schema is defined) to understand how plugin descriptors are structured and validated — you'll add a `metrics` field to the manifest schema following the same pattern as existing fields like `enrichers`, `commands`, `sidebarEntries`.

## Context

Phases 1–4 built the full metrics pipeline: backend providers, frontend charts, thresholds, annotations. The chart infrastructure in `MetricsTab.svelte` already renders `MetricResult[]` arrays as charts. This phase adds a plugin extension point: plugins declare PromQL templates in their descriptors, the backend executes them via the Prometheus provider, and the frontend renders them as additional charts grouped under the plugin name. This runs in parallel with Phase 5 (sparklines).

## Files to Read

- Plugin manifest definition file (likely `internal/plugin/manifest.go` or a JSON Schema file) — **what to look for**: existing descriptor fields (`enrichers`, `commands`, `sidebarEntries`) and how they're parsed/validated — add `metrics` following the same pattern
- `internal/plugin/registry.go` — **what to look for**: `Register`, `Deactivate`, `Remove` methods — you'll add metric query registration/deregistration alongside enrichers and sidebar entries
- `internal/metrics/service.go` — **what to look for**: `GetPluginMetrics` method signature (declared in Phase 1 but not implemented) — implement it here
- `internal/metrics/queries.go` — **what to look for**: `BuiltinQueries` map structure and `substituteVars` function — plugin queries use the same template variable substitution
- `frontend/src/lib/components/charts/MetricsTab.svelte` — **what to look for**: where to add a "Plugin metrics" section below the built-in charts — conditionally rendered when plugin queries exist for the current GVR
- `frontend/src/lib/plugins/slots.svelte.ts` — **what to look for**: how plugin UI extension points are registered/queried (e.g., `getDetailTabs`) — follow the same pattern for metric queries
- `METRICS_SPEC.md` — **what to look for**: Plugin metric template format (lines 255–280), template variables (lines 271–278)

## What Exists

- `MetricsService` with all RPC methods; `GetPluginMetrics` declared but returns empty/not-implemented
- `PrometheusClient` with `QueryRange` and `QueryInstant`
- `substituteVars(query, vars)` in `queries.go` — simple `{{var}}` replacement
- `MetricsChart.svelte` — full interactive chart component
- `MetricsTab.svelte` — detail tab rendering built-in charts with thresholds and annotations
- Plugin system: `PluginRegistry` with `Register`/`Deactivate`/`Remove`, manifest validation, hot reload via fsnotify
- Plugin descriptor schema with existing optional fields (`enrichers`, `commands`, `sidebarEntries`, `detailTabs`)
- `PluginRegistry.UnregisterPlugin(name)` pattern used by enrichers — follow the same for metric queries

## Deliverables

1. **Plugin manifest `metrics` field** — Extend schema to accept:
   ```yaml
   metrics:
     - gvr: core.v1.pods
       queries:
         - name: "HTTP Request Rate"
           query: 'sum(rate(http_requests_total{namespace="{{namespace}}", pod="{{name}}"}[5m])) by (code)'
           unit: "req/s"
   ```
   Optional field — existing plugins without it are unaffected. Validated via JSON Schema.
2. **Plugin metric query registration in `PluginRegistry`** — On plugin load, extract `metrics` from descriptor and store queries keyed by GVR + plugin name. On `Deactivate`, remove plugin's queries. On `Remove`, delete entirely. Emit `plugin:metrics-changed` event (or reuse existing `plugin:loaded`/`plugin:reloading` events).
3. **`MetricsService.GetPluginMetrics(clusterCtx, gvr, namespace, name, rangeMinutes)`** — Loads plugin queries for the given GVR from `PluginRegistry`, performs `substituteVars` with resource context, executes each via `PrometheusClient.QueryRange`, returns `[]MetricResult` with `Source` field set to plugin name. Returns empty (not error) when Prometheus is unavailable.
4. **Plugin metrics section in `MetricsTab.svelte`** — Rendered below built-in charts. Groups charts by plugin name with a header (e.g., "istio-metrics"). Each query renders as a `MetricsChart`. Section hidden when no plugin queries exist for the current GVR or when Prometheus is unavailable. Reloads when plugin events fire.
5. **Template variable context** — `{{namespace}}`, `{{name}}`, `{{node}}`, `{{container}}` populated from the current resource being viewed. Unknown variables left as-is in the query string (Prometheus will return empty, not error).

## Tests

- **Go unit test**
  - Plugin descriptor with `metrics` field parses correctly; missing `metrics` field produces no error
  - Plugin metric queries registered in `PluginRegistry` under correct GVR key
  - Plugin metric queries deregistered on `Deactivate` — subsequent `GetPluginMetrics` returns empty for that plugin
  - `GetPluginMetrics` executes queries with correct variable substitution (namespace, name)
  - `GetPluginMetrics` returns results with `Source` set to the plugin name
  - `GetPluginMetrics` returns empty slice (not error) when Prometheus is unavailable
  - Multiple plugins registering queries for the same GVR both appear in results
- **Frontend test (vitest)**
  - Plugin metrics section renders when `GetPluginMetrics` returns non-empty results
  - Plugin metrics section hidden when no plugin queries exist for the GVR
  - Plugin metrics section hidden when Prometheus is unavailable
  - Charts grouped under plugin name headers
  - Plugin reload event triggers re-fetch of plugin metrics
- **Manual verification**
  - Install a test plugin with `metrics` descriptor → extra charts appear in pod metrics tab
  - Charts show correct data from Prometheus
  - Hot reload plugin → charts update with new/changed queries
  - Disable plugin → charts disappear immediately
  - Re-enable plugin → charts reappear

## Acceptance Criteria

- [ ] Plugin descriptor YAML with `metrics` field validates and loads without error
- [ ] Plugin metric queries execute via Prometheus and render as charts in the metrics tab
- [ ] Charts grouped under plugin name header, visually separated from built-in charts
- [ ] Plugin hot reload updates metric queries without app restart
- [ ] Plugin deactivation removes its metric charts from the tab
- [ ] No Prometheus → plugin metrics section hidden entirely (no error state)
- [ ] Template variables (`{{namespace}}`, `{{name}}`, `{{node}}`, `{{container}}`) substituted correctly
- [ ] Multiple plugins contributing queries to the same GVR both render
- [ ] Existing plugins without `metrics` field continue to work unchanged
- [ ] All unit tests pass

## Definition of Done

Install a test plugin that declares a PromQL query for pods (e.g., HTTP request rate). Open a pod detail view — the metrics tab shows the built-in CPU/memory charts plus a new section labeled with the plugin name containing the custom chart with live Prometheus data. Hot-reload the plugin with a modified query — the chart updates. Disable the plugin — the section disappears. On a cluster without Prometheus, the plugin metrics section is not shown.

## Known Gotchas

- **Plugin queries are Prometheus-only.** There is no metrics-server equivalent for arbitrary PromQL. When Prometheus is unavailable, `GetPluginMetrics` returns empty — the frontend hides the section. Don't try to fall back to metrics-server for plugin queries.
- **Plugin descriptor `metrics` is optional.** Follow the same pattern as other optional fields (`commands`, `enrichers`). Nil/missing → no-op, not validation error. Use `omitempty` in Go struct tags.
- **Plugin reload cycle: deregister then register.** On hot reload (`plugin:reloading` → `plugin:loaded`), the old queries are deregistered via `Deactivate` before the new ones are registered. The frontend should handle the brief gap (empty plugin metrics between events) gracefully — don't show an error flash.
- **`substituteVars` leaves unknown variables as-is.** A plugin query with `{{custom_var}}` won't be substituted — it'll be sent to Prometheus literally. Prometheus will likely return empty results. This is by design — don't validate variable names, just document the available ones.
- **Multiple plugins, same GVR.** Both should render. Store as `map[string]map[string][]MetricQuery` (GVR → plugin name → queries) in the registry, not `map[string][]MetricQuery` which would overwrite. Follow the enricher pattern where multiple enrichers chain for the same GVR.
