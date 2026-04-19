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

func TestResolveForCluster_VolumeBrowser_GlobalDefaults(t *testing.T) {
	cfg := DefaultConfig()
	r := cfg.ResolveForCluster("")
	testza.AssertEqual(t, "alpine:edge", r.VolumeBrowser.Image)
	testza.AssertEqual(t, "/mnt/volume", r.VolumeBrowser.MountPath)
	testza.AssertNotNil(t, r.VolumeBrowser.ActiveDeadlineSeconds)
	testza.AssertEqual(t, int64(3600), *r.VolumeBrowser.ActiveDeadlineSeconds)
	testza.AssertEqual(t, "prompt", r.VolumeBrowser.OrphanCleanupOnStartup)
}

func TestResolveForCluster_VolumeBrowser_NoOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VolumeBrowser.Image = "custom:v1"
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {FavoriteNS: []string{"default"}},
	}

	r := cfg.ResolveForCluster("prod")
	testza.AssertEqual(t, "custom:v1", r.VolumeBrowser.Image)
	testza.AssertEqual(t, "/mnt/volume", r.VolumeBrowser.MountPath)
}

func TestResolveForCluster_VolumeBrowser_PerClusterOverride(t *testing.T) {
	cfg := DefaultConfig()
	deadline := int64(600)
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {
			VolumeBrowser: &VolumeBrowserConfig{
				Image:                 "busybox",
				ActiveDeadlineSeconds: &deadline,
				Resources: &ResourceReqs{
					Requests: map[string]string{"cpu": "50m"},
				},
				NodeSelector: map[string]string{"zone": "us-east"},
			},
		},
	}

	r := cfg.ResolveForCluster("prod")
	testza.AssertEqual(t, "busybox", r.VolumeBrowser.Image)
	testza.AssertEqual(t, "/mnt/volume", r.VolumeBrowser.MountPath)
	testza.AssertNotNil(t, r.VolumeBrowser.ActiveDeadlineSeconds)
	testza.AssertEqual(t, int64(600), *r.VolumeBrowser.ActiveDeadlineSeconds)
	testza.AssertNotNil(t, r.VolumeBrowser.Resources)
	testza.AssertEqual(t, "50m", r.VolumeBrowser.Resources.Requests["cpu"])
	testza.AssertEqual(t, "us-east", r.VolumeBrowser.NodeSelector["zone"])
	testza.AssertEqual(t, "prompt", r.VolumeBrowser.OrphanCleanupOnStartup)
}

func TestResolveForCluster_VolumeBrowser_EmptyOverrideDoesNotClobber(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VolumeBrowser.Image = "global:v1"
	cfg.VolumeBrowser.MountPath = "/global"
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {
			VolumeBrowser: &VolumeBrowserConfig{},
		},
	}

	r := cfg.ResolveForCluster("prod")
	testza.AssertEqual(t, "global:v1", r.VolumeBrowser.Image)
	testza.AssertEqual(t, "/global", r.VolumeBrowser.MountPath)
	testza.AssertNotNil(t, r.VolumeBrowser.ActiveDeadlineSeconds)
	testza.AssertEqual(t, int64(3600), *r.VolumeBrowser.ActiveDeadlineSeconds)
}

func TestResolveForCluster_VolumeBrowser_OverrideReadOnlyTrueToFalse(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VolumeBrowser.ReadOnly = ptr(true)
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {
			VolumeBrowser: &VolumeBrowserConfig{
				ReadOnly: ptr(false),
			},
		},
	}

	r := cfg.ResolveForCluster("prod")
	testza.AssertNotNil(t, r.VolumeBrowser.ReadOnly)
	testza.AssertFalse(t, *r.VolumeBrowser.ReadOnly)
}

func TestResolveForCluster_VolumeBrowser_NilResourcesInherit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VolumeBrowser.Resources = &ResourceReqs{
		Requests: map[string]string{"cpu": "200m"},
	}
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {
			VolumeBrowser: &VolumeBrowserConfig{Image: "override"},
		},
	}

	r := cfg.ResolveForCluster("prod")
	testza.AssertEqual(t, "override", r.VolumeBrowser.Image)
	testza.AssertNotNil(t, r.VolumeBrowser.Resources)
	testza.AssertEqual(t, "200m", r.VolumeBrowser.Resources.Requests["cpu"])
}

func TestResolveForCluster_VolumeBrowser_NilDeadlinePreservedAtGlobal(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VolumeBrowser.ActiveDeadlineSeconds = nil

	r := cfg.ResolveForCluster("")
	testza.AssertNil(t, r.VolumeBrowser.ActiveDeadlineSeconds)
}

func TestResolveForCluster_VolumeBrowser_DeepCopy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.VolumeBrowser.NodeSelector = map[string]string{"a": "1"}

	r := cfg.ResolveForCluster("")
	r.VolumeBrowser.NodeSelector["a"] = "mutated"
	r.VolumeBrowser.Image = "mutated"

	testza.AssertEqual(t, "1", cfg.VolumeBrowser.NodeSelector["a"])
	testza.AssertEqual(t, "alpine:edge", cfg.VolumeBrowser.Image)
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
