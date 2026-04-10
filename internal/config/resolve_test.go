package config

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func ptr[T any](v T) *T { return &v }

func TestResolveForCluster_GlobalDefaults(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Theme = "dark"
	cfg.CompactRows = true
	cfg.AccentColor = "#ff0000"
	cfg.FontSize = 16

	r := cfg.ResolveForCluster("")
	testza.AssertEqual(t, "dark", r.Theme)
	testza.AssertTrue(t, r.CompactRows)
	testza.AssertEqual(t, "#ff0000", r.AccentColor)
	testza.AssertEqual(t, 16, r.FontSize)
}

func TestResolveForCluster_HardcodedDefaults(t *testing.T) {
	cfg := DefaultConfig()
	r := cfg.ResolveForCluster("")
	testza.AssertEqual(t, "system", r.Theme)
	testza.AssertFalse(t, r.CompactRows)
	testza.AssertFalse(t, r.ReadOnly)
	testza.AssertFalse(t, r.TerminalWebGL)
	testza.AssertEqual(t, "", r.AccentColor)
	testza.AssertEqual(t, 0, r.FontSize)
}

func TestResolveForCluster_PerClusterOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Theme = "dark"
	cfg.CompactRows = true
	cfg.AccentColor = "#ff0000"
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {
			ReadOnly:    ptr(true),
			CompactRows: ptr(false),
			AccentColor: ptr("#00ff00"),
			FavoriteNS:  []string{"default"},
		},
	}

	r := cfg.ResolveForCluster("prod")
	testza.AssertEqual(t, "dark", r.Theme)
	testza.AssertTrue(t, r.ReadOnly)
	testza.AssertFalse(t, r.CompactRows)
	testza.AssertEqual(t, "#00ff00", r.AccentColor)
	testza.AssertEqual(t, []string{"default"}, r.FavoriteNS)
}

func TestResolveForCluster_UnknownCluster(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Theme = "light"

	r := cfg.ResolveForCluster("nonexistent")
	testza.AssertEqual(t, "light", r.Theme)
}

func TestResolveForCluster_NilFieldsInherit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ReadOnly = true
	cfg.CompactRows = true
	cfg.Clusters = map[string]*ClusterPrefs{
		"dev": {
			FavoriteNS: []string{"kube-system"},
		},
	}

	r := cfg.ResolveForCluster("dev")
	testza.AssertTrue(t, r.ReadOnly)
	testza.AssertTrue(t, r.CompactRows)
	testza.AssertEqual(t, []string{"kube-system"}, r.FavoriteNS)
}

func TestResolveForCluster_MetricsOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Metrics = map[string]*MetricsConfig{
		"prod": {PrometheusURL: "http://global:9090"},
	}
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {
			Metrics: &MetricsConfig{PrometheusURL: "http://cluster:9090"},
		},
	}

	r := cfg.ResolveForCluster("prod")
	testza.AssertNotNil(t, r.Metrics)
	testza.AssertEqual(t, "http://cluster:9090", r.Metrics.PrometheusURL)
}

func TestResolveForCluster_ColumnPrefsOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ColumnPrefs = map[string]*GVRColumnPrefs{
		"core.v1.pods": {Order: []string{"name", "status", "age"}},
	}
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {
			ColumnPrefs: map[string]*GVRColumnPrefs{
				"core.v1.pods": {Order: []string{"name", "node", "age"}},
			},
		},
	}

	r := cfg.ResolveForCluster("prod")
	testza.AssertNotNil(t, r.ColumnPrefs)
	testza.AssertEqual(t, []string{"name", "node", "age"}, r.ColumnPrefs["core.v1.pods"].Order)
}

func TestResolveForCluster_SavedFilters_Merged(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SavedFilters = map[string][]SavedFilter{
		"core.v1.pods": {{Name: "global-filter"}},
	}
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {
			SavedFilters: map[string][]SavedFilter{
				"core.v1.pods": {{Name: "cluster-filter"}},
			},
		},
	}

	r := cfg.ResolveForCluster("prod")
	testza.AssertNotNil(t, r.SavedFilters)
	testza.AssertLen(t, r.SavedFilters["core.v1.pods"], 2)
}

func TestResolveForCluster_Keybindings(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Keybindings = map[string]string{
		"delete": "d",
		"scale":  "s",
	}

	r := cfg.ResolveForCluster("")
	testza.AssertNotNil(t, r.Keybindings)
	testza.AssertEqual(t, "d", r.Keybindings["delete"])
	testza.AssertEqual(t, "s", r.Keybindings["scale"])
}
