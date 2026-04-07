package metrics

import (
	"strings"
	"time"
)

// BuiltinQueries maps resource type keys to their PromQL query templates.
// Keys follow the pattern "group.version.resource" for primary queries,
// "group.version.resource:thresholds" for limit/request overlays,
// and "sparkline:group.version.resource" for list-view batch queries.
var BuiltinQueries = map[string][]MetricQuery{
	"core.v1.pods": {
		{Name: "CPU Usage", Query: `sum(rate(container_cpu_usage_seconds_total{namespace="{{namespace}}", pod="{{name}}"}[5m])) by (container)`, Unit: "cores", Source: "builtin"},
		{Name: "Memory Usage", Query: `sum(container_memory_working_set_bytes{namespace="{{namespace}}", pod="{{name}}"}) by (container)`, Unit: "bytes", Source: "builtin"},
		{Name: "CPU Throttling", Query: `rate(container_cpu_cfs_throttled_periods_total{namespace="{{namespace}}", pod="{{name}}"}[5m]) / rate(container_cpu_cfs_periods_total{namespace="{{namespace}}", pod="{{name}}"}[5m])`, Unit: "ratio", Source: "builtin"},
	},

	"core.v1.pods:thresholds": {
		{Name: "CPU Request", Query: `kube_pod_container_resource_requests{namespace="{{namespace}}", pod="{{name}}", resource="cpu"}`, Unit: "cores", Source: "builtin"},
		{Name: "CPU Limit", Query: `kube_pod_container_resource_limits{namespace="{{namespace}}", pod="{{name}}", resource="cpu"}`, Unit: "cores", Source: "builtin"},
		{Name: "Memory Request", Query: `kube_pod_container_resource_requests{namespace="{{namespace}}", pod="{{name}}", resource="memory"}`, Unit: "bytes", Source: "builtin"},
		{Name: "Memory Limit", Query: `kube_pod_container_resource_limits{namespace="{{namespace}}", pod="{{name}}", resource="memory"}`, Unit: "bytes", Source: "builtin"},
	},

	"core.v1.nodes": {
		{Name: "CPU Usage", Query: `sum(rate(node_cpu_seconds_total{mode!="idle", node="{{name}}"}[5m]))`, Unit: "cores", Source: "builtin"},
		{Name: "Memory Usage", Query: `node_memory_MemTotal_bytes{node="{{name}}"} - node_memory_MemAvailable_bytes{node="{{name}}"}`, Unit: "bytes", Source: "builtin"},
	},

	"apps.v1.deployments": {
		{Name: "CPU Usage", Query: `sum(rate(container_cpu_usage_seconds_total{namespace="{{namespace}}", pod=~"{{name}}-[a-z0-9]+-[a-z0-9]+"}[5m])) by (pod)`, Unit: "cores", Source: "builtin"},
		{Name: "Memory Usage", Query: `sum(container_memory_working_set_bytes{namespace="{{namespace}}", pod=~"{{name}}-[a-z0-9]+-[a-z0-9]+"}) by (pod)`, Unit: "bytes", Source: "builtin"},
	},

	"namespace": {
		{Name: "CPU Usage", Query: `sum(rate(container_cpu_usage_seconds_total{namespace="{{namespace}}"}[5m]))`, Unit: "cores", Source: "builtin"},
		{Name: "Memory Usage", Query: `sum(container_memory_working_set_bytes{namespace="{{namespace}}"})`, Unit: "bytes", Source: "builtin"},
	},

	"sparkline:core.v1.pods": {
		{Name: "CPU", Query: `sum(rate(container_cpu_usage_seconds_total{namespace="{{namespace}}"}[5m])) by (pod)`, Unit: "cores", Source: "builtin"},
		{Name: "Memory", Query: `sum(container_memory_working_set_bytes{namespace="{{namespace}}"}) by (pod)`, Unit: "bytes", Source: "builtin"},
	},

	"sparkline:core.v1.nodes": {
		{Name: "CPU", Query: `sum(rate(node_cpu_seconds_total{mode!="idle"}[5m])) by (node)`, Unit: "cores", Source: "builtin"},
		{Name: "Memory", Query: `(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes)`, Unit: "bytes", Source: "builtin"},
	},
}

// SubstituteVars replaces {{key}} placeholders in a PromQL template with values.
// Unknown variables are left as-is.
func SubstituteVars(query string, vars map[string]string) string {
	for k, v := range vars {
		query = strings.ReplaceAll(query, "{{"+k+"}}", v)
	}
	return query
}

// StepForRange returns the appropriate Prometheus query step duration for a given time range.
func StepForRange(rangeMinutes int) time.Duration {
	switch {
	case rangeMinutes <= 60:
		return 15 * time.Second
	case rangeMinutes <= 360:
		return 1 * time.Minute
	case rangeMinutes <= 1440:
		return 5 * time.Minute
	default:
		return 30 * time.Minute
	}
}
