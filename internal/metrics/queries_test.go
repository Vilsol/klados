package metrics_test

import (
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/Vilsol/klados/internal/metrics"
)

func TestSubstituteVars_ReplacesKnownVars(t *testing.T) {
	query := `container_cpu_usage_seconds_total{namespace="{{namespace}}", pod="{{name}}"}`
	result := metrics.SubstituteVars(query, map[string]string{
		"namespace": "default",
		"name":      "my-pod",
	})
	testza.AssertEqual(t, `container_cpu_usage_seconds_total{namespace="default", pod="my-pod"}`, result)
}

func TestSubstituteVars_LeavesUnknownVarsAsIs(t *testing.T) {
	query := `metric{ns="{{namespace}}", other="{{unknown}}"}`
	result := metrics.SubstituteVars(query, map[string]string{
		"namespace": "test-ns",
	})
	testza.AssertContains(t, result, `ns="test-ns"`)
	testza.AssertContains(t, result, `{{unknown}}`)
}

func TestSubstituteVars_EmptyVars(t *testing.T) {
	query := `some_metric{pod="{{name}}"}`
	result := metrics.SubstituteVars(query, nil)
	testza.AssertEqual(t, query, result)
}

func TestStepForRange(t *testing.T) {
	tests := []struct {
		rangeMinutes int
		expected     time.Duration
	}{
		{15, 15 * time.Second},
		{60, 15 * time.Second},
		{360, 1 * time.Minute},
		{1440, 5 * time.Minute},
		{10080, 30 * time.Minute},
	}

	for _, tt := range tests {
		result := metrics.StepForRange(tt.rangeMinutes)
		testza.AssertEqual(t, tt.expected, result, "rangeMinutes=%d", tt.rangeMinutes)
	}
}

func TestBuiltinQueries_AllKeysHaveNonEmptyQueries(t *testing.T) {
	for key, queries := range metrics.BuiltinQueries {
		testza.AssertNotEqual(t, 0, len(queries), "key %q has no queries", key)
		for _, q := range queries {
			testza.AssertNotEqual(t, "", q.Name, "key %q has query with empty name", key)
			testza.AssertNotEqual(t, "", q.Query, "key %q has query with empty query", key)
			testza.AssertNotEqual(t, "", q.Unit, "key %q has query with empty unit", key)
			testza.AssertEqual(t, "builtin", q.Source, "key %q query %q source", key, q.Name)
		}
	}
}

func TestBuiltinQueries_ExpectedKeys(t *testing.T) {
	expectedKeys := []string{
		"core.v1.pods",
		"core.v1.pods:thresholds",
		"core.v1.nodes",
		"apps.v1.deployments",
		"namespace",
		"sparkline:core.v1.pods",
		"sparkline:core.v1.nodes",
	}
	for _, key := range expectedKeys {
		_, ok := metrics.BuiltinQueries[key]
		testza.AssertTrue(t, ok, "missing expected key %q", key)
	}
}
