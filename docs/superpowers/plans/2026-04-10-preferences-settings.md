# Preferences & Settings System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a comprehensive preferences and settings system with a dedicated `/settings` UI, two-level config cascade (global → per-cluster), rebindable keybindings, saved filters, and plugin settings via JSON Schema.

**Architecture:** Extend the existing `config.json` with new fields (`Clusters`, `Keybindings`, `SavedFilters`, `AccentColor`, `FontSize`, `StartupBehavior`). Add a `ResolveForCluster()` method that cascades global → per-cluster. Emit `config:updated` Wails events on save. Frontend gets a new `preferencesStore` that subscribes to config changes and a new `/settings` route tree with category pages. Plugin settings are declared via JSON Schema in the manifest and rendered by a reusable `SchemaForm.svelte` component.

**Tech Stack:** Go 1.25, Wails v3 alpha.74, Svelte 5 (runes), Tailwind v4, testza, vitest, santhosh-tekuri/jsonschema/v6

---

## File Structure

### Go Backend — New/Modified Files

| File | Action | Responsibility |
|---|---|---|
| `internal/config/config.go` | Modify | Add new struct types and fields |
| `internal/config/resolve.go` | Create | `ResolvedPrefs` struct and `ResolveForCluster()` |
| `internal/config/resolve_test.go` | Create | Cascade resolution tests |
| `internal/config/config_test.go` | Modify | Add round-trip tests for new fields |
| `internal/services/config.go` | Modify | Add `GetResolvedPrefs`, `SetKeybinding`, `SaveFilter`, `SetClusterPrefs`, etc. |
| `internal/services/config.go` | Modify | Emit `config:updated` Wails event on saves |
| `internal/plugin/types/manifest.go` | Modify (regenerated) | Add `Settings` field to `Extensions` |
| `schemas/manifest.v1.json` | Modify | Add `settings` to Extensions schema |
| `internal/services/plugin.go` | Modify | Add `GetPluginSettings`, `SetPluginSettings` methods |
| `internal/plugin/host_api.go` | Modify | Add `settings.get`, `settings.set` dispatch methods |

### Frontend — New Files

| File | Responsibility |
|---|---|
| `frontend/src/lib/stores/preferences.svelte.ts` | Reactive preference store, cascade-aware |
| `frontend/src/routes/settings/SettingsLayout.svelte` | Settings page layout with sidebar + content area |
| `frontend/src/routes/settings/SettingsSidebar.svelte` | Settings category navigation |
| `frontend/src/routes/settings/GeneralSettings.svelte` | Theme, font size, startup behavior, terminal WebGL |
| `frontend/src/routes/settings/AppearanceSettings.svelte` | Accent color, compact rows |
| `frontend/src/routes/settings/ClusterListSettings.svelte` | List of clusters with overrides |
| `frontend/src/routes/settings/ClusterSettings.svelte` | Per-cluster override form |
| `frontend/src/routes/settings/KeybindingSettings.svelte` | Keybinding editor table |
| `frontend/src/routes/settings/FilterSettings.svelte` | Saved filter management |
| `frontend/src/routes/settings/ColumnSettings.svelte` | Column visibility/order (surfaces existing columnPrefs) |
| `frontend/src/routes/settings/PluginListSettings.svelte` | Plugin list (those with settings schemas) |
| `frontend/src/routes/settings/PluginSettings.svelte` | Schema-driven plugin settings form |
| `frontend/src/lib/components/SchemaForm.svelte` | JSON Schema → form fields renderer |

### Frontend — Modified Files

| File | Change |
|---|---|
| `frontend/src/routes/routes.ts` | Add `/settings/*` routes |
| `frontend/src/lib/components/Sidebar.svelte` | Add gear icon entry point |
| `frontend/src/App.svelte` | Subscribe to `config:updated`, initialize preferences store |
| `frontend/src/lib/stores/shortcuts.svelte.ts` | Support user-defined keybinding overrides |
| `frontend/src/lib/theme.svelte.ts` | Read from preferences store instead of direct ConfigService call |
| `frontend/src/lib/stores/cluster.svelte.ts` | Read `isReadOnly` from preferences store |
| `frontend/src/lib/__tests__/wails-mock.ts` | Add mock methods for new ConfigService RPCs |

---

## Task 1: Config Schema Extension (Go)

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for new types**

Add to `internal/config/config_test.go`:

```go
func TestClusterPrefsRoundTrip(t *testing.T) {
	cfg := tempConfig(t)
	ro := true
	accent := "#ff6b6b"
	display := "Production"
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod-ctx": {
			ReadOnly:    &ro,
			AccentColor: &accent,
			DisplayName: &display,
			FavoriteNS:  []string{"default", "kube-system"},
		},
	}

	testza.AssertNoError(t, cfg.Save())

	data, err := os.ReadFile(cfg.path)
	testza.AssertNoError(t, err)

	loaded := &Config{}
	testza.AssertNoError(t, json.Unmarshal(data, loaded))
	testza.AssertEqual(t, cfg.Clusters, loaded.Clusters)
}

func TestSavedFiltersRoundTrip(t *testing.T) {
	cfg := tempConfig(t)
	cfg.SavedFilters = map[string][]SavedFilter{
		"core.v1.pods": {
			{
				Name:   "erroring",
				Labels: map[string]string{"app": "web"},
				Search: "CrashLoop",
			},
		},
	}

	testza.AssertNoError(t, cfg.Save())

	data, err := os.ReadFile(cfg.path)
	testza.AssertNoError(t, err)

	loaded := &Config{}
	testza.AssertNoError(t, json.Unmarshal(data, loaded))
	testza.AssertEqual(t, cfg.SavedFilters, loaded.SavedFilters)
}

func TestKeybindingsRoundTrip(t *testing.T) {
	cfg := tempConfig(t)
	cfg.Keybindings = map[string]string{
		"command-palette": "Control+p",
	}

	testza.AssertNoError(t, cfg.Save())

	data, err := os.ReadFile(cfg.path)
	testza.AssertNoError(t, err)

	loaded := &Config{}
	testza.AssertNoError(t, json.Unmarshal(data, loaded))
	testza.AssertEqual(t, cfg.Keybindings, loaded.Keybindings)
}

func TestNewFieldsOmittedWhenEmpty(t *testing.T) {
	cfg := tempConfig(t)
	testza.AssertNoError(t, cfg.Save())

	data, err := os.ReadFile(cfg.path)
	testza.AssertNoError(t, err)

	s := string(data)
	testza.AssertFalse(t, strings.Contains(s, "clusters"))
	testza.AssertFalse(t, strings.Contains(s, "keybindings"))
	testza.AssertFalse(t, strings.Contains(s, "savedFilters"))
	testza.AssertFalse(t, strings.Contains(s, "startupBehavior"))
	testza.AssertFalse(t, strings.Contains(s, "accentColor"))
	testza.AssertFalse(t, strings.Contains(s, "fontSize"))
}

func TestStartupBehaviorRoundTrip(t *testing.T) {
	cfg := tempConfig(t)
	cfg.StartupBehavior = "specific"
	cfg.StartupCluster = "my-cluster"

	testza.AssertNoError(t, cfg.Save())

	data, err := os.ReadFile(cfg.path)
	testza.AssertNoError(t, err)

	loaded := &Config{}
	testza.AssertNoError(t, json.Unmarshal(data, loaded))
	testza.AssertEqual(t, "specific", loaded.StartupBehavior)
	testza.AssertEqual(t, "my-cluster", loaded.StartupCluster)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run "TestClusterPrefsRoundTrip|TestSavedFiltersRoundTrip|TestKeybindingsRoundTrip|TestNewFieldsOmittedWhenEmpty|TestStartupBehaviorRoundTrip" -v`

Expected: compilation errors — `ClusterPrefs`, `SavedFilter`, `Keybindings` etc. not defined.

- [ ] **Step 3: Add new types and fields to config.go**

Add these types before the `Config` struct in `internal/config/config.go`:

```go
type ClusterPrefs struct {
	ReadOnly     *bool                      `json:"readOnly,omitempty"`
	CompactRows  *bool                      `json:"compactRows,omitempty"`
	AccentColor  *string                    `json:"accentColor,omitempty"`
	DisplayName  *string                    `json:"displayName,omitempty"`
	Metrics      *MetricsConfig             `json:"metrics,omitempty"`
	ColumnPrefs  map[string]*GVRColumnPrefs `json:"columnPrefs,omitempty"`
	FavoriteNS   []string                   `json:"favoriteNamespaces,omitempty"`
	SavedFilters map[string][]SavedFilter   `json:"savedFilters,omitempty"`
}

type SavedFilter struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Search      string            `json:"search,omitempty"`
}
```

Add these fields to the `Config` struct (after `PortForwards`):

```go
	Clusters        map[string]*ClusterPrefs `json:"clusters,omitempty"`
	Keybindings     map[string]string        `json:"keybindings,omitempty"`
	SavedFilters    map[string][]SavedFilter `json:"savedFilters,omitempty"`
	StartupBehavior string                   `json:"startupBehavior,omitempty"`
	StartupCluster  string                   `json:"startupCluster,omitempty"`
	AccentColor     string                   `json:"accentColor,omitempty"`
	FontSize        int                      `json:"fontSize,omitempty"`
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`

Expected: all tests PASS including the new ones.

- [ ] **Step 5: Commit**

```
feat(config): add ClusterPrefs, SavedFilter, keybindings, and appearance fields
```

---

## Task 2: Cascade Resolution (Go)

**Files:**
- Create: `internal/config/resolve.go`
- Create: `internal/config/resolve_test.go`

- [ ] **Step 1: Write failing tests for cascade resolution**

Create `internal/config/resolve_test.go`:

```go
package config

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestResolveForCluster_GlobalDefaults(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Theme = "dark"
	cfg.CompactRows = true
	cfg.AccentColor = "#ff0000"
	cfg.FontSize = 16

	resolved := cfg.ResolveForCluster("")
	testza.AssertEqual(t, "dark", resolved.Theme)
	testza.AssertTrue(t, resolved.CompactRows)
	testza.AssertEqual(t, "#ff0000", resolved.AccentColor)
	testza.AssertEqual(t, 16, resolved.FontSize)
	testza.AssertFalse(t, resolved.ReadOnly)
}

func TestResolveForCluster_HardcodedDefaults(t *testing.T) {
	cfg := DefaultConfig()
	resolved := cfg.ResolveForCluster("")
	testza.AssertEqual(t, "system", resolved.Theme)
	testza.AssertFalse(t, resolved.CompactRows)
	testza.AssertEqual(t, "", resolved.AccentColor)
	testza.AssertEqual(t, 0, resolved.FontSize)
	testza.AssertFalse(t, resolved.ReadOnly)
}

func TestResolveForCluster_PerClusterOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Theme = "dark"
	cfg.CompactRows = true
	cfg.AccentColor = "#ff0000"

	ro := true
	compact := false
	accent := "#00ff00"
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {
			ReadOnly:    &ro,
			CompactRows: &compact,
			AccentColor: &accent,
			FavoriteNS:  []string{"default"},
		},
	}

	resolved := cfg.ResolveForCluster("prod")
	testza.AssertEqual(t, "dark", resolved.Theme) // not overridden, inherits
	testza.AssertTrue(t, resolved.ReadOnly)        // overridden
	testza.AssertFalse(t, resolved.CompactRows)    // overridden to false
	testza.AssertEqual(t, "#00ff00", resolved.AccentColor) // overridden
	testza.AssertEqual(t, []string{"default"}, resolved.FavoriteNS)
}

func TestResolveForCluster_UnknownCluster(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Theme = "light"

	resolved := cfg.ResolveForCluster("nonexistent")
	testza.AssertEqual(t, "light", resolved.Theme) // global value
}

func TestResolveForCluster_NilFieldsInherit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ReadOnly = true
	cfg.CompactRows = true

	cfg.Clusters = map[string]*ClusterPrefs{
		"dev": {
			// ReadOnly and CompactRows are nil → inherit from global
			FavoriteNS: []string{"testing"},
		},
	}

	resolved := cfg.ResolveForCluster("dev")
	testza.AssertTrue(t, resolved.ReadOnly)     // inherited
	testza.AssertTrue(t, resolved.CompactRows)  // inherited
	testza.AssertEqual(t, []string{"testing"}, resolved.FavoriteNS)
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

	resolved := cfg.ResolveForCluster("prod")
	testza.AssertEqual(t, "http://cluster:9090", resolved.Metrics.PrometheusURL)
}

func TestResolveForCluster_ColumnPrefsOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ColumnPrefs = map[string]*GVRColumnPrefs{
		"core.v1.pods": {
			Order: []string{"Name", "Status"},
		},
	}
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {
			ColumnPrefs: map[string]*GVRColumnPrefs{
				"core.v1.pods": {
					Order: []string{"Name", "Node", "Status"},
				},
			},
		},
	}

	resolved := cfg.ResolveForCluster("prod")
	testza.AssertEqual(t, []string{"Name", "Node", "Status"}, resolved.ColumnPrefs["core.v1.pods"].Order)
}

func TestResolveForCluster_SavedFilters_Merged(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SavedFilters = map[string][]SavedFilter{
		"core.v1.pods": {
			{Name: "global-filter", Search: "error"},
		},
	}
	cfg.Clusters = map[string]*ClusterPrefs{
		"prod": {
			SavedFilters: map[string][]SavedFilter{
				"core.v1.pods": {
					{Name: "prod-filter", Labels: map[string]string{"env": "prod"}},
				},
			},
		},
	}

	resolved := cfg.ResolveForCluster("prod")
	// Both global and cluster filters should be present
	testza.AssertEqual(t, 2, len(resolved.SavedFilters["core.v1.pods"]))
}

func TestResolveForCluster_Keybindings(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Keybindings = map[string]string{
		"command-palette": "Control+p",
		"close-tab":      "Control+w",
	}

	resolved := cfg.ResolveForCluster("any")
	testza.AssertEqual(t, "Control+p", resolved.Keybindings["command-palette"])
	testza.AssertEqual(t, "Control+w", resolved.Keybindings["close-tab"])
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run "TestResolveForCluster" -v`

Expected: compilation errors — `ResolvedPrefs` and `ResolveForCluster` not defined.

- [ ] **Step 3: Implement ResolvedPrefs and ResolveForCluster**

Create `internal/config/resolve.go`:

```go
package config

// ResolvedPrefs contains all preferences with cascade resolution applied.
// All fields are non-pointer — nil/empty cluster overrides fall through to global.
type ResolvedPrefs struct {
	Theme        string                     `json:"theme"`
	AccentColor  string                     `json:"accentColor"`
	FontSize     int                        `json:"fontSize"`
	CompactRows  bool                       `json:"compactRows"`
	ReadOnly     bool                       `json:"readOnly"`
	Metrics      *MetricsConfig             `json:"metrics,omitempty"`
	ColumnPrefs  map[string]*GVRColumnPrefs `json:"columnPrefs,omitempty"`
	FavoriteNS   []string                   `json:"favoriteNamespaces,omitempty"`
	Keybindings  map[string]string          `json:"keybindings,omitempty"`
	SavedFilters map[string][]SavedFilter   `json:"savedFilters,omitempty"`
}

// ResolveForCluster returns a flat ResolvedPrefs with global → per-cluster cascade applied.
// If ctxName is empty or has no cluster overrides, global values are returned.
func (c *Config) ResolveForCluster(ctxName string) ResolvedPrefs {
	c.mu.Lock()
	defer c.mu.Unlock()

	r := ResolvedPrefs{
		Theme:        c.Theme,
		AccentColor:  c.AccentColor,
		FontSize:     c.FontSize,
		CompactRows:  c.CompactRows,
		ReadOnly:     c.ReadOnly,
		Keybindings:  copyStringMap(c.Keybindings),
		SavedFilters: copySavedFilters(c.SavedFilters),
		ColumnPrefs:  copyColumnPrefs(c.ColumnPrefs),
	}

	if c.Metrics != nil {
		m := *c.Metrics[ctxName]
		_ = m // handled below
	}
	// Global metrics is a map keyed by context — extract the relevant entry
	if c.Metrics != nil {
		if m, ok := c.Metrics[ctxName]; ok {
			r.Metrics = m
		}
	}

	if ctxName == "" || c.Clusters == nil {
		return r
	}

	cp, ok := c.Clusters[ctxName]
	if !ok {
		return r
	}

	if cp.ReadOnly != nil {
		r.ReadOnly = *cp.ReadOnly
	}
	if cp.CompactRows != nil {
		r.CompactRows = *cp.CompactRows
	}
	if cp.AccentColor != nil {
		r.AccentColor = *cp.AccentColor
	}
	if cp.Metrics != nil {
		r.Metrics = cp.Metrics
	}
	if len(cp.FavoriteNS) > 0 {
		r.FavoriteNS = cp.FavoriteNS
	}

	// Column prefs: cluster entries override global per-GVR
	if len(cp.ColumnPrefs) > 0 {
		for gvr, prefs := range cp.ColumnPrefs {
			r.ColumnPrefs[gvr] = prefs
		}
	}

	// Saved filters: concatenate global + cluster (cluster appended)
	if len(cp.SavedFilters) > 0 {
		for gvr, filters := range cp.SavedFilters {
			r.SavedFilters[gvr] = append(r.SavedFilters[gvr], filters...)
		}
	}

	return r
}

func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func copySavedFilters(m map[string][]SavedFilter) map[string][]SavedFilter {
	if m == nil {
		return nil
	}
	out := make(map[string][]SavedFilter, len(m))
	for k, v := range m {
		cp := make([]SavedFilter, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

func copyColumnPrefs(m map[string]*GVRColumnPrefs) map[string]*GVRColumnPrefs {
	if m == nil {
		return nil
	}
	out := make(map[string]*GVRColumnPrefs, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
```

- [ ] **Step 4: Fix the incorrect Metrics handling**

The initial implementation has a bug — it tried to dereference `c.Metrics[ctxName]` incorrectly. Remove the broken lines (the `if c.Metrics != nil { m := *c.Metrics[ctxName]; _ = m }` block). The correct `Metrics` handling is already present below it.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```
feat(config): add ResolveForCluster cascade resolution
```

---

## Task 3: Config Updated Event (Go)

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/services/config.go`

- [ ] **Step 1: Add event emission to Config.Save()**

In `internal/config/config.go`, add an `emit` callback field to `Config` and call it at the end of `Save()`:

```go
// Add to Config struct fields (after `path string`):
	emit func(string, any)

// Add setter method:
func (c *Config) SetEmit(fn func(string, any)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.emit = fn
}
```

At the end of `Save()`, after the `WriteFile` call succeeds, emit the event:

```go
	if c.emit != nil {
		c.emit("config:updated", nil)
	}
	return nil
```

Note: `Save()` is called while the lock is held, but `emit` dispatches asynchronously via Wails, so this is safe.

- [ ] **Step 2: Wire up emit in ConfigService**

In `internal/services/config.go`, set the emit function during service construction. Add to `NewConfigService`:

```go
func NewConfigService(ctx context.Context, cfg *config.Config) *ConfigService {
	s := &ConfigService{ctx: ctx, config: cfg}
	return s
}
```

Add a `ServiceStartup` method to wire up the Wails event emitter:

```go
func (c *ConfigService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	app := application.Get()
	if app != nil {
		c.config.SetEmit(func(name string, data any) {
			app.Event.Emit(name, data)
		})
	}
	return nil
}
```

Add the import for `github.com/wailsapp/wails/v3/pkg/application`.

- [ ] **Step 3: Run existing tests to verify nothing broke**

Run: `go test ./internal/config/ -v`

Expected: all tests PASS (emit is nil in tests, so the nil check prevents any panic).

- [ ] **Step 4: Commit**

```
feat(config): emit config:updated Wails event on save
```

---

## Task 4: ConfigService New RPCs (Go)

**Files:**
- Modify: `internal/services/config.go`

- [ ] **Step 1: Add GetResolvedPrefs method**

```go
func (c *ConfigService) GetResolvedPrefs(ctxName string) config.ResolvedPrefs {
	return c.config.ResolveForCluster(ctxName)
}
```

- [ ] **Step 2: Add SetAccentColor and SetFontSize methods**

```go
func (c *ConfigService) SetAccentColor(color string) error {
	return c.config.Update(func(cfg *config.Config) {
		cfg.AccentColor = color
	})
}

func (c *ConfigService) SetFontSize(size int) error {
	return c.config.Update(func(cfg *config.Config) {
		cfg.FontSize = size
	})
}
```

- [ ] **Step 3: Add StartupBehavior methods**

```go
func (c *ConfigService) SetStartupBehavior(behavior string, cluster string) error {
	switch behavior {
	case "last", "chooser", "specific":
	default:
		return fmt.Errorf("invalid startup behavior: %q", behavior)
	}
	return c.config.Update(func(cfg *config.Config) {
		cfg.StartupBehavior = behavior
		cfg.StartupCluster = cluster
	})
}
```

- [ ] **Step 4: Add Keybinding methods**

```go
func (c *ConfigService) SetKeybinding(actionID string, keys string) error {
	return c.config.Update(func(cfg *config.Config) {
		if cfg.Keybindings == nil {
			cfg.Keybindings = make(map[string]string)
		}
		if keys == "" {
			delete(cfg.Keybindings, actionID)
		} else {
			cfg.Keybindings[actionID] = keys
		}
	})
}

func (c *ConfigService) ResetKeybindings() error {
	return c.config.Update(func(cfg *config.Config) {
		cfg.Keybindings = nil
	})
}
```

- [ ] **Step 5: Add ClusterPrefs methods**

```go
func (c *ConfigService) GetClusterPrefs(ctxName string) *config.ClusterPrefs {
	if c.config.Clusters == nil {
		return nil
	}
	var result *config.ClusterPrefs
	c.config.Read(func(cfg *config.Config) {
		result = cfg.Clusters[ctxName]
	})
	return result
}

func (c *ConfigService) SetClusterPrefs(ctxName string, prefs *config.ClusterPrefs) error {
	return c.config.Update(func(cfg *config.Config) {
		if cfg.Clusters == nil {
			cfg.Clusters = make(map[string]*config.ClusterPrefs)
		}
		cfg.Clusters[ctxName] = prefs
	})
}

func (c *ConfigService) DeleteClusterPrefs(ctxName string) error {
	return c.config.Update(func(cfg *config.Config) {
		delete(cfg.Clusters, ctxName)
	})
}
```

- [ ] **Step 6: Add SavedFilter methods**

```go
func (c *ConfigService) GetSavedFilters(gvr string) []config.SavedFilter {
	var result []config.SavedFilter
	c.config.Read(func(cfg *config.Config) {
		if cfg.SavedFilters != nil {
			result = cfg.SavedFilters[gvr]
		}
	})
	return result
}

func (c *ConfigService) SetSavedFilters(gvr string, filters []config.SavedFilter) error {
	return c.config.Update(func(cfg *config.Config) {
		if cfg.SavedFilters == nil {
			cfg.SavedFilters = make(map[string][]config.SavedFilter)
		}
		if len(filters) == 0 {
			delete(cfg.SavedFilters, gvr)
		} else {
			cfg.SavedFilters[gvr] = filters
		}
	})
}

func (c *ConfigService) SetClusterSavedFilters(ctxName string, gvr string, filters []config.SavedFilter) error {
	return c.config.Update(func(cfg *config.Config) {
		if cfg.Clusters == nil {
			cfg.Clusters = make(map[string]*config.ClusterPrefs)
		}
		cp := cfg.Clusters[ctxName]
		if cp == nil {
			cp = &config.ClusterPrefs{}
			cfg.Clusters[ctxName] = cp
		}
		if cp.SavedFilters == nil {
			cp.SavedFilters = make(map[string][]config.SavedFilter)
		}
		if len(filters) == 0 {
			delete(cp.SavedFilters, gvr)
		} else {
			cp.SavedFilters[gvr] = filters
		}
	})
}
```

- [ ] **Step 7: Regenerate Wails bindings**

Run: `wails3 generate bindings`

- [ ] **Step 8: Verify build**

Run: `go build ./...`

Expected: successful build.

- [ ] **Step 9: Commit**

```
feat(services): add ConfigService RPCs for preferences, keybindings, filters, and cluster prefs
```

---

## Task 5: Plugin Settings Schema (Go)

**Files:**
- Modify: `schemas/manifest.v1.json`
- Regenerate: `internal/plugin/types/manifest.go`
- Modify: `internal/services/plugin.go`
- Modify: `internal/plugin/host_api.go`

- [ ] **Step 1: Add settings to manifest JSON Schema**

In `schemas/manifest.v1.json`, add to the `Extensions` properties (after `"metrics"`):

```json
        "settings": {
          "$ref": "#/$defs/SettingsDeclaration"
        }
```

Add to `$defs`:

```json
    "SettingsDeclaration": {
      "type": "object",
      "required": ["schema"],
      "additionalProperties": false,
      "properties": {
        "schema": {
          "type": "object",
          "description": "JSON Schema (draft 2020-12) defining the plugin's settings. Must be type: object with properties."
        }
      }
    }
```

- [ ] **Step 2: Copy schema to internal and regenerate Go types**

Run: `cp schemas/manifest.v1.json internal/plugin/schema/manifest.v1.json && mise run generate:plugin-types`

Verify `internal/plugin/types/manifest.go` now has a `Settings` field in `Extensions`.

- [ ] **Step 3: Add GetPluginSettings and SetPluginSettings to PluginService**

In `internal/services/plugin.go`, add methods that read/write plugin settings using existing `PluginStorage`:

```go
func (s *PluginService) GetPluginSettings(name string) (string, error) {
	storage := s.registry.GetPluginStorage(name)
	if storage == nil {
		return "{}", nil
	}
	val, ok := storage.Get("settings")
	if !ok {
		return "{}", nil
	}
	return val, nil
}

func (s *PluginService) SetPluginSettings(name string, settingsJSON string) error {
	storage := s.registry.GetPluginStorage(name)
	if storage == nil {
		return fmt.Errorf("plugin %q not found", name)
	}
	// TODO: validate against manifest schema using jsonschema/v6
	storage.Set("settings", settingsJSON)
	return nil
}

func (s *PluginService) GetPluginSettingsSchema(name string) (string, error) {
	plugin := s.registry.GetPlugin(name)
	if plugin == nil {
		return "", fmt.Errorf("plugin %q not found", name)
	}
	manifest := plugin.Manifest
	if manifest.Extensions == nil || manifest.Extensions.Settings == nil {
		return "", nil
	}
	data, err := json.Marshal(manifest.Extensions.Settings.Schema)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
```

Note: Check what methods `registry` exposes for getting plugin storage. If `GetPluginStorage` doesn't exist, you'll need to add it — look at the registry's plugin map and storage field.

- [ ] **Step 4: Add settings.get and settings.set to host API**

In `internal/plugin/host_api.go`, add to the dispatch switch:

```go
	case "settings.get":
		val, ok := h.storage.Get("settings")
		if !ok {
			return "{}", nil
		}
		return val, nil

	case "settings.set":
		h.storage.Set("settings", request.Payload)
		return "{}", nil
```

- [ ] **Step 5: Regenerate Wails bindings**

Run: `wails3 generate bindings`

- [ ] **Step 6: Verify build**

Run: `go build ./...`

Expected: successful build.

- [ ] **Step 7: Commit**

```
feat(plugin): add settings schema declaration and storage host API
```

---

## Task 6: Frontend Preferences Store

**Files:**
- Create: `frontend/src/lib/stores/preferences.svelte.ts`
- Modify: `frontend/src/lib/__tests__/wails-mock.ts`

- [ ] **Step 1: Update wails mock with new ConfigService methods**

Add to `mockConfigService` in `frontend/src/lib/__tests__/wails-mock.ts`:

```typescript
  GetResolvedPrefs: vi.fn().mockResolvedValue({
    theme: 'system',
    accentColor: '',
    fontSize: 0,
    compactRows: false,
    readOnly: false,
    terminalWebGL: false,
    metrics: null,
    columnPrefs: null,
    favoriteNamespaces: null,
    keybindings: null,
    savedFilters: null,
  }),
  SetAccentColor: vi.fn().mockResolvedValue(undefined),
  SetFontSize: vi.fn().mockResolvedValue(undefined),
  SetStartupBehavior: vi.fn().mockResolvedValue(undefined),
  SetKeybinding: vi.fn().mockResolvedValue(undefined),
  ResetKeybindings: vi.fn().mockResolvedValue(undefined),
  GetClusterPrefs: vi.fn().mockResolvedValue(null),
  SetClusterPrefs: vi.fn().mockResolvedValue(undefined),
  DeleteClusterPrefs: vi.fn().mockResolvedValue(undefined),
  GetSavedFilters: vi.fn().mockResolvedValue([]),
  SetSavedFilters: vi.fn().mockResolvedValue(undefined),
  SetClusterSavedFilters: vi.fn().mockResolvedValue(undefined),
```

- [ ] **Step 2: Create preferences store**

Create `frontend/src/lib/stores/preferences.svelte.ts`:

```typescript
import { Events } from '@wailsio/runtime'
import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'

export interface ResolvedPrefs {
  theme: string
  accentColor: string
  fontSize: number
  compactRows: boolean
  readOnly: boolean
  terminalWebGL: boolean
  keybindings: Record<string, string> | null
  savedFilters: Record<string, SavedFilter[]> | null
  favoriteNamespaces: string[] | null
}

export interface SavedFilter {
  name: string
  labels?: Record<string, string>
  annotations?: Record<string, string>
  search?: string
}

class PreferencesStore {
  prefs = $state<ResolvedPrefs>({
    theme: 'system',
    accentColor: '',
    fontSize: 0,
    compactRows: false,
    readOnly: false,
    terminalWebGL: false,
    keybindings: null,
    savedFilters: null,
    favoriteNamespaces: null,
  })

  private activeContext = ''
  private unsub: (() => void) | undefined

  async load(ctxName: string) {
    this.activeContext = ctxName
    try {
      const resolved = await ConfigService.GetResolvedPrefs(ctxName)
      if (resolved) {
        this.prefs = resolved as ResolvedPrefs
      }
    } catch (e) {
      console.error('Failed to load preferences:', e)
    }
  }

  subscribe() {
    this.unsub = Events.On('config:updated', () => {
      this.load(this.activeContext)
    })
  }

  destroy() {
    this.unsub?.()
  }

  getKeybinding(actionID: string): string | undefined {
    return this.prefs.keybindings?.[actionID]
  }

  getSavedFilters(gvr: string): SavedFilter[] {
    return this.prefs.savedFilters?.[gvr] ?? []
  }
}

export const preferencesStore = new PreferencesStore()
```

- [ ] **Step 3: Commit**

```
feat(frontend): add preferencesStore with cascade-aware config loading
```

---

## Task 7: Wire Preferences Store into App

**Files:**
- Modify: `frontend/src/App.svelte`
- Modify: `frontend/src/lib/theme.svelte.ts`
- Modify: `frontend/src/lib/stores/cluster.svelte.ts`
- Modify: `frontend/src/lib/stores/shortcuts.svelte.ts`

- [ ] **Step 1: Initialize preferences store in App.svelte**

In `frontend/src/App.svelte`, import the preferences store and initialize it during mount:

```typescript
import { preferencesStore } from '$lib/stores/preferences.svelte'
```

In the `onMount` async block, after `await clusterStore.loadContexts()`, add:

```typescript
      preferencesStore.subscribe()
      await preferencesStore.load(clusterStore.activeContext ?? '')
```

In the cleanup function returned from `onMount`, add:

```typescript
      preferencesStore.destroy()
```

- [ ] **Step 2: Update theme.svelte.ts to read from preferences**

Replace the direct `ConfigService.GetTheme()` call in `App.svelte` with reading from `preferencesStore.prefs.theme`. In `App.svelte`, replace the theme loading block:

```typescript
      // Replace this:
      // const theme = await ConfigService.GetTheme()
      // if (theme) setTheme(theme as 'light' | 'dark' | 'system')
      
      // With (after preferencesStore.load):
      setTheme(preferencesStore.prefs.theme as 'light' | 'dark' | 'system')
```

Add a `$effect` in `App.svelte` to keep theme in sync when preferences change:

```typescript
  $effect(() => {
    const theme = preferencesStore.prefs.theme
    if (theme) setTheme(theme as 'light' | 'dark' | 'system')
  })
```

- [ ] **Step 3: Update cluster store to read readOnly from preferences**

In `frontend/src/lib/stores/cluster.svelte.ts`, in the `loadContexts()` method, replace the block that reads `isReadOnly` from `ConfigService.GetConfig()`:

```typescript
      // Replace:
      // const cfg = await ConfigService.GetConfig()
      // this.isReadOnly = cfg?.readOnly ?? false
      
      // With:
      // isReadOnly is now managed by preferencesStore — read from there
      // (This block can be removed; readOnly is accessed via preferencesStore.prefs.readOnly)
```

Update `canMutate()` to read from preferencesStore:

```typescript
import { preferencesStore } from './preferences.svelte'

  canMutate(): boolean {
    if (preferencesStore.prefs.readOnly) return false
    // ... rest unchanged
  }
```

Remove the `isReadOnly` state field and the `setReadOnly` method from `ClusterStore` — these now live in preferencesStore. The `setReadOnly` call should go through `ConfigService.SetConfig` which triggers `config:updated` → preferencesStore auto-refreshes.

Note: Check all usages of `clusterStore.isReadOnly` and `clusterStore.canMutate()` in the codebase and ensure they still work. `canMutate()` stays on clusterStore since it also checks RBAC.

- [ ] **Step 4: Add keybinding override support to ShortcutStore**

In `frontend/src/lib/stores/shortcuts.svelte.ts`, add a method to apply user overrides:

```typescript
import { preferencesStore } from './preferences.svelte'

export class ShortcutStore {
  // ... existing fields ...

  // Returns the effective key combo for a shortcut, considering user overrides
  getEffectiveKeys(def: ShortcutDef): string {
    const override = preferencesStore.getKeybinding(def.id)
    return override ?? def.keys
  }

  dispatch(e: KeyboardEvent) {
    const combo = buildKeyCombo(e)

    if (this.focusMode === 'terminal') {
      if (combo === 'Control+Shift+Escape') {
        this.focusMode = 'normal'
        e.preventDefault()
      }
      return
    }

    for (const def of this._shortcuts) {
      const effectiveKeys = this.getEffectiveKeys(def)
      if (combo !== effectiveKeys) continue
      const modes = def.modes ?? ['normal']
      if (!modes.includes(this.focusMode)) continue
      e.preventDefault()
      def.action()
      return
    }
  }

  // Expose registered shortcuts for the keybinding settings page
  getAll(): ShortcutDef[] {
    return [...this._shortcuts]
  }
}
```

- [ ] **Step 5: Regenerate Wails bindings**

Run: `wails3 generate bindings`

- [ ] **Step 6: Verify frontend type-checks**

Run: `cd frontend && pnpm check`

Expected: no errors.

- [ ] **Step 7: Commit**

```
feat(frontend): wire preferencesStore into theme, readOnly, and keybindings
```

---

## Task 8: Settings Route Structure

**Files:**
- Create: `frontend/src/routes/settings/SettingsLayout.svelte`
- Create: `frontend/src/routes/settings/SettingsSidebar.svelte`
- Modify: `frontend/src/routes/routes.ts`
- Modify: `frontend/src/lib/components/Sidebar.svelte`

- [ ] **Step 1: Create SettingsSidebar component**

Create `frontend/src/routes/settings/SettingsSidebar.svelte`:

```svelte
<script lang="ts">
  import { push } from 'svelte-spa-router'

  interface Props {
    activeSection: string
    pluginNames?: string[]
  }

  let { activeSection, pluginNames = [] }: Props = $props()

  const sections = [
    { id: 'general', label: 'General', route: '/settings/general' },
    { id: 'appearance', label: 'Appearance', route: '/settings/appearance' },
    { id: 'clusters', label: 'Clusters', route: '/settings/clusters' },
    { id: 'keybindings', label: 'Keybindings', route: '/settings/keybindings' },
    { id: 'filters', label: 'Filters', route: '/settings/filters' },
    { id: 'columns', label: 'Columns', route: '/settings/columns' },
  ]
</script>

<nav class="w-56 shrink-0 border-r border-border overflow-y-auto py-4">
  <ul class="space-y-0.5 px-2">
    {#each sections as section}
      <li>
        <button
          class="w-full text-left px-3 py-1.5 rounded text-sm transition-colors
            {activeSection === section.id ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-surface-hover hover:text-fg'}"
          onclick={() => push(section.route)}
        >
          {section.label}
        </button>
      </li>
    {/each}
  </ul>

  {#if pluginNames.length > 0}
    <div class="border-t border-border mt-4 pt-4 px-2">
      <p class="text-xs font-medium text-muted-foreground px-3 mb-2 uppercase tracking-wider">Plugins</p>
      <ul class="space-y-0.5">
        {#each pluginNames as name}
          <li>
            <button
              class="w-full text-left px-3 py-1.5 rounded text-sm transition-colors
                {activeSection === `plugin-${name}` ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-surface-hover hover:text-fg'}"
              onclick={() => push(`/settings/plugins/${name}`)}
            >
              {name}
            </button>
          </li>
        {/each}
      </ul>
    </div>
  {/if}
</nav>
```

- [ ] **Step 2: Create SettingsLayout component**

Create `frontend/src/routes/settings/SettingsLayout.svelte`:

```svelte
<script lang="ts">
  import { push } from 'svelte-spa-router'
  import SettingsSidebar from './SettingsSidebar.svelte'
  import type { Snippet } from 'svelte'

  interface Props {
    activeSection: string
    pluginNames?: string[]
    children: Snippet
  }

  let { activeSection, pluginNames = [], children }: Props = $props()
</script>

<div class="flex flex-col h-full">
  <header class="flex items-center gap-3 px-6 py-4 border-b border-border shrink-0">
    <button
      class="text-muted-foreground hover:text-fg transition-colors"
      onclick={() => push('/')}
      title="Back"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
        <path fill-rule="evenodd" d="M9.707 16.707a1 1 0 01-1.414 0l-6-6a1 1 0 010-1.414l6-6a1 1 0 011.414 1.414L5.414 9H17a1 1 0 110 2H5.414l4.293 4.293a1 1 0 010 1.414z" clip-rule="evenodd" />
      </svg>
    </button>
    <h1 class="text-lg font-semibold text-fg">Settings</h1>
  </header>

  <div class="flex flex-1 overflow-hidden">
    <SettingsSidebar {activeSection} {pluginNames} />
    <main class="flex-1 overflow-y-auto p-6">
      {@render children()}
    </main>
  </div>
</div>
```

- [ ] **Step 3: Create placeholder settings page components**

Create `frontend/src/routes/settings/GeneralSettings.svelte`:

```svelte
<script lang="ts">
  import { preferencesStore } from '$lib/stores/preferences.svelte'
  import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'

  const themes = ['system', 'light', 'dark'] as const
  const startupOptions = [
    { value: 'last', label: 'Reconnect to last session' },
    { value: 'chooser', label: 'Show cluster chooser' },
    { value: 'specific', label: 'Connect to specific cluster' },
  ]
</script>

<div class="max-w-2xl space-y-8">
  <section>
    <h2 class="text-base font-medium text-fg mb-4">General</h2>
    <p class="text-sm text-muted-foreground">General settings will be implemented in subsequent tasks.</p>
  </section>
</div>
```

Create similar placeholder files for `AppearanceSettings.svelte`, `ClusterListSettings.svelte`, `ClusterSettings.svelte`, `KeybindingSettings.svelte`, `FilterSettings.svelte`, `ColumnSettings.svelte`, `PluginListSettings.svelte`, and `PluginSettings.svelte` — each with a `<div>` and placeholder text.

- [ ] **Step 4: Create SettingsPage router component**

Create `frontend/src/routes/settings/SettingsPage.svelte`:

```svelte
<script lang="ts">
  import { querystring } from 'svelte-spa-router'
  import SettingsLayout from './SettingsLayout.svelte'
  import GeneralSettings from './GeneralSettings.svelte'
  import AppearanceSettings from './AppearanceSettings.svelte'
  import ClusterListSettings from './ClusterListSettings.svelte'
  import ClusterSettings from './ClusterSettings.svelte'
  import KeybindingSettings from './KeybindingSettings.svelte'
  import FilterSettings from './FilterSettings.svelte'
  import ColumnSettings from './ColumnSettings.svelte'
  import PluginListSettings from './PluginListSettings.svelte'
  import PluginSettings from './PluginSettings.svelte'
  import { onMount } from 'svelte'
  import * as PluginService from '../../../bindings/github.com/Vilsol/klados/internal/services/pluginservice.js'

  interface Props {
    params?: { wild?: string }
  }

  let { params }: Props = $props()

  let pluginNames = $state<string[]>([])

  const section = $derived(params?.wild || 'general')

  // Determine which sub-section and extract any parameters
  const sectionParts = $derived(section.split('/'))
  const mainSection = $derived(sectionParts[0])
  const subParam = $derived(sectionParts[1] ?? '')

  const activeSection = $derived(
    mainSection === 'plugins' && subParam ? `plugin-${subParam}` : mainSection
  )

  onMount(async () => {
    try {
      const plugins = await PluginService.ListPlugins()
      // Filter to only plugins that have settings schemas
      const withSettings: string[] = []
      for (const p of plugins ?? []) {
        try {
          const schema = await PluginService.GetPluginSettingsSchema(p.name)
          if (schema) withSettings.push(p.name)
        } catch {}
      }
      pluginNames = withSettings
    } catch {}
  })
</script>

<SettingsLayout {activeSection} {pluginNames}>
  {#if mainSection === 'general'}
    <GeneralSettings />
  {:else if mainSection === 'appearance'}
    <AppearanceSettings />
  {:else if mainSection === 'clusters' && subParam}
    <ClusterSettings ctxName={decodeURIComponent(subParam)} />
  {:else if mainSection === 'clusters'}
    <ClusterListSettings />
  {:else if mainSection === 'keybindings'}
    <KeybindingSettings />
  {:else if mainSection === 'filters'}
    <FilterSettings />
  {:else if mainSection === 'columns'}
    <ColumnSettings />
  {:else if mainSection === 'plugins' && subParam}
    <PluginSettings pluginName={subParam} />
  {:else if mainSection === 'plugins'}
    <PluginListSettings />
  {:else}
    <GeneralSettings />
  {/if}
</SettingsLayout>
```

- [ ] **Step 5: Add settings routes**

In `frontend/src/routes/routes.ts`, add the settings route:

```typescript
import SettingsPage from './settings/SettingsPage.svelte'

export const routes = {
  '/': ClusterList,
  '/clusters': ClusterList,
  '/plugins': PluginManagement,
  '/settings': SettingsPage,
  '/settings/*wild': SettingsPage,
  // ... rest unchanged
}
```

Note: Place `/settings` routes BEFORE the `/c/:ctx` routes to avoid matching conflicts.

- [ ] **Step 6: Add gear icon to Sidebar**

In `frontend/src/lib/components/Sidebar.svelte`, add a gear icon button at the bottom of the sidebar (in the footer area, near the collapse toggle):

```svelte
<button
  class="p-2 text-muted-foreground hover:text-fg transition-colors"
  onclick={() => push('/settings')}
  title="Settings"
>
  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
    <path fill-rule="evenodd" d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z" clip-rule="evenodd" />
  </svg>
</button>
```

- [ ] **Step 7: Verify frontend type-checks**

Run: `cd frontend && pnpm check`

Expected: no errors.

- [ ] **Step 8: Commit**

```
feat(frontend): add /settings route with layout, sidebar, and placeholder pages
```

---

## Task 9: General Settings Page

**Files:**
- Modify: `frontend/src/routes/settings/GeneralSettings.svelte`

- [ ] **Step 1: Implement General Settings page**

Replace the placeholder with the full implementation:

```svelte
<script lang="ts">
  import { preferencesStore } from '$lib/stores/preferences.svelte'
  import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'

  const themes = [
    { value: 'system', label: 'System' },
    { value: 'light', label: 'Light' },
    { value: 'dark', label: 'Dark' },
  ] as const

  const startupOptions = [
    { value: 'last', label: 'Reconnect to last session' },
    { value: 'chooser', label: 'Show cluster chooser' },
    { value: 'specific', label: 'Connect to specific cluster' },
  ] as const

  let startupBehavior = $state('last')
  let startupCluster = $state('')

  async function setTheme(theme: string) {
    try {
      await ConfigService.SetTheme(theme)
    } catch (e) {
      console.error('Failed to set theme:', e)
    }
  }

  async function setFontSize(size: number) {
    try {
      await ConfigService.SetFontSize(size)
    } catch (e) {
      console.error('Failed to set font size:', e)
    }
  }

  async function setTerminalWebGL(enabled: boolean) {
    try {
      await ConfigService.SetTerminalWebGL(enabled)
    } catch (e) {
      console.error('Failed to set terminal WebGL:', e)
    }
  }

  async function saveStartup() {
    try {
      await ConfigService.SetStartupBehavior(startupBehavior, startupCluster)
    } catch (e) {
      console.error('Failed to set startup behavior:', e)
    }
  }
</script>

<div class="max-w-2xl space-y-8">
  <section class="space-y-4">
    <h2 class="text-base font-medium text-fg">Theme</h2>
    <div class="flex gap-2">
      {#each themes as t}
        <button
          class="px-4 py-2 rounded border text-sm transition-colors
            {preferencesStore.prefs.theme === t.value
              ? 'border-accent bg-accent text-accent-foreground'
              : 'border-border text-muted-foreground hover:bg-surface-hover'}"
          onclick={() => setTheme(t.value)}
        >
          {t.label}
        </button>
      {/each}
    </div>
  </section>

  <section class="space-y-4">
    <h2 class="text-base font-medium text-fg">Font Size</h2>
    <div class="flex items-center gap-3">
      <input
        type="range"
        min="10"
        max="24"
        step="1"
        value={preferencesStore.prefs.fontSize || 14}
        oninput={(e) => setFontSize(parseInt(e.currentTarget.value))}
        class="flex-1"
      />
      <span class="text-sm text-muted-foreground w-8 text-right">
        {preferencesStore.prefs.fontSize || 14}px
      </span>
    </div>
  </section>

  <section class="space-y-4">
    <h2 class="text-base font-medium text-fg">Startup Behavior</h2>
    <div class="space-y-2">
      {#each startupOptions as opt}
        <label class="flex items-center gap-3 cursor-pointer">
          <input
            type="radio"
            name="startup"
            value={opt.value}
            checked={startupBehavior === opt.value}
            onchange={() => { startupBehavior = opt.value; saveStartup() }}
            class="accent-accent"
          />
          <span class="text-sm text-fg">{opt.label}</span>
        </label>
      {/each}
      {#if startupBehavior === 'specific'}
        <input
          type="text"
          placeholder="Cluster context name"
          bind:value={startupCluster}
          onblur={saveStartup}
          class="mt-2 w-full max-w-xs px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
        />
      {/if}
    </div>
  </section>

  <section class="space-y-4">
    <h2 class="text-base font-medium text-fg">Terminal</h2>
    <label class="flex items-center gap-3 cursor-pointer">
      <input
        type="checkbox"
        checked={preferencesStore.prefs.terminalWebGL ?? false}
        onchange={(e) => setTerminalWebGL(e.currentTarget.checked)}
        class="accent-accent"
      />
      <span class="text-sm text-fg">Use WebGL renderer</span>
    </label>
  </section>
</div>
```

Note: `terminalWebGL` needs to be added to `ResolvedPrefs` in the Go backend and the frontend `ResolvedPrefs` interface. Add `TerminalWebGL bool` to `ResolvedPrefs` in `resolve.go` and populate it from `c.TerminalWebGL`. Add `terminalWebGL?: boolean` to the frontend interface.

- [ ] **Step 2: Verify frontend type-checks**

Run: `cd frontend && pnpm check`

- [ ] **Step 3: Commit**

```
feat(frontend): implement General Settings page
```

---

## Task 10: Appearance Settings Page

**Files:**
- Modify: `frontend/src/routes/settings/AppearanceSettings.svelte`

- [ ] **Step 1: Implement Appearance Settings**

```svelte
<script lang="ts">
  import { preferencesStore } from '$lib/stores/preferences.svelte'
  import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'

  const presetColors = [
    '#6366f1', '#8b5cf6', '#ec4899', '#ef4444',
    '#f97316', '#eab308', '#22c55e', '#06b6d4',
  ]

  async function setAccentColor(color: string) {
    try {
      await ConfigService.SetAccentColor(color)
    } catch (e) {
      console.error('Failed to set accent color:', e)
    }
  }

  async function setCompactRows(compact: boolean) {
    try {
      await ConfigService.SetCompactRows(compact)
    } catch (e) {
      console.error('Failed to set compact rows:', e)
    }
  }
</script>

<div class="max-w-2xl space-y-8">
  <section class="space-y-4">
    <h2 class="text-base font-medium text-fg">Accent Color</h2>
    <div class="flex items-center gap-3 flex-wrap">
      {#each presetColors as color}
        <button
          class="w-8 h-8 rounded-full border-2 transition-all
            {preferencesStore.prefs.accentColor === color ? 'border-fg scale-110' : 'border-transparent hover:scale-105'}"
          style="background-color: {color}"
          onclick={() => setAccentColor(color)}
          title={color}
        />
      {/each}
      <div class="flex items-center gap-2">
        <input
          type="color"
          value={preferencesStore.prefs.accentColor || '#6366f1'}
          onchange={(e) => setAccentColor(e.currentTarget.value)}
          class="w-8 h-8 rounded cursor-pointer border-0 p-0"
        />
        <span class="text-xs text-muted-foreground">Custom</span>
      </div>
      {#if preferencesStore.prefs.accentColor}
        <button
          class="text-xs text-muted-foreground hover:text-fg"
          onclick={() => setAccentColor('')}
        >
          Reset to default
        </button>
      {/if}
    </div>
  </section>

  <section class="space-y-4">
    <h2 class="text-base font-medium text-fg">List Display</h2>
    <label class="flex items-center gap-3 cursor-pointer">
      <input
        type="checkbox"
        checked={preferencesStore.prefs.compactRows}
        onchange={(e) => setCompactRows(e.currentTarget.checked)}
        class="accent-accent"
      />
      <span class="text-sm text-fg">Compact rows</span>
    </label>
    <p class="text-xs text-muted-foreground">Reduce row height in resource lists</p>
  </section>
</div>
```

- [ ] **Step 2: Commit**

```
feat(frontend): implement Appearance Settings page with accent color and compact rows
```

---

## Task 11: Keybinding Settings Page

**Files:**
- Modify: `frontend/src/routes/settings/KeybindingSettings.svelte`

- [ ] **Step 1: Implement Keybinding Settings**

```svelte
<script lang="ts">
  import { shortcutStore, type ShortcutDef } from '$lib/stores/shortcuts.svelte'
  import { preferencesStore } from '$lib/stores/preferences.svelte'
  import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'

  let listeningFor = $state<string | null>(null)
  let conflicts = $derived(findConflicts())

  const shortcuts = $derived(shortcutStore.getAll())

  function getEffectiveKeys(def: ShortcutDef): string {
    return preferencesStore.getKeybinding(def.id) ?? def.keys
  }

  function findConflicts(): Map<string, string[]> {
    const byKey = new Map<string, string[]>()
    for (const def of shortcutStore.getAll()) {
      const keys = getEffectiveKeys(def)
      const list = byKey.get(keys) ?? []
      list.push(def.id)
      byKey.set(keys, list)
    }
    const result = new Map<string, string[]>()
    for (const [keys, ids] of byKey) {
      if (ids.length > 1) result.set(keys, ids)
    }
    return result
  }

  function isConflict(id: string): boolean {
    for (const ids of conflicts.values()) {
      if (ids.includes(id)) return true
    }
    return false
  }

  function startListening(id: string) {
    listeningFor = id
  }

  function handleKeyCapture(e: KeyboardEvent) {
    if (!listeningFor) return
    e.preventDefault()
    e.stopPropagation()

    if (e.key === 'Escape') {
      listeningFor = null
      return
    }

    // Build combo string matching ShortcutStore format
    const parts: string[] = []
    if (e.ctrlKey) parts.push('Control')
    if (e.altKey) parts.push('Alt')
    if (e.shiftKey) parts.push('Shift')
    if (e.metaKey) parts.push('Meta')
    if (!['Control', 'Alt', 'Shift', 'Meta'].includes(e.key)) {
      parts.push(e.key)
    }

    // Require at least one modifier + a non-modifier key
    if (parts.length < 2) return

    const combo = parts.join('+')
    const actionId = listeningFor
    listeningFor = null

    ConfigService.SetKeybinding(actionId, combo).catch((e) =>
      console.error('Failed to set keybinding:', e)
    )
  }

  async function resetBinding(id: string) {
    try {
      await ConfigService.SetKeybinding(id, '')
    } catch (e) {
      console.error('Failed to reset keybinding:', e)
    }
  }

  async function resetAll() {
    try {
      await ConfigService.ResetKeybindings()
    } catch (e) {
      console.error('Failed to reset keybindings:', e)
    }
  }
</script>

<svelte:window onkeydown={listeningFor ? handleKeyCapture : undefined} />

<div class="max-w-3xl space-y-6">
  <div class="flex items-center justify-between">
    <h2 class="text-base font-medium text-fg">Keyboard Shortcuts</h2>
    <button
      class="text-xs text-muted-foreground hover:text-fg px-3 py-1 rounded border border-border hover:bg-surface-hover"
      onclick={resetAll}
    >
      Reset all to defaults
    </button>
  </div>

  <table class="w-full text-sm">
    <thead>
      <tr class="text-left text-muted-foreground border-b border-border">
        <th class="py-2 font-medium">Action</th>
        <th class="py-2 font-medium">Current</th>
        <th class="py-2 font-medium">Default</th>
        <th class="py-2 font-medium w-20"></th>
      </tr>
    </thead>
    <tbody>
      {#each shortcuts as def}
        {@const effective = getEffectiveKeys(def)}
        {@const isOverridden = preferencesStore.getKeybinding(def.id) !== undefined}
        {@const hasConflict = isConflict(def.id)}
        <tr class="border-b border-border/50">
          <td class="py-2 text-fg">{def.description}</td>
          <td class="py-2">
            {#if listeningFor === def.id}
              <span class="px-2 py-0.5 rounded bg-accent text-accent-foreground text-xs animate-pulse">
                Press keys...
              </span>
            {:else}
              <button
                class="px-2 py-0.5 rounded text-xs font-mono transition-colors
                  {hasConflict ? 'bg-destructive/20 text-destructive' : isOverridden ? 'bg-accent/20 text-accent' : 'bg-surface text-fg'}"
                onclick={() => startListening(def.id)}
                title="Click to rebind"
              >
                {effective}
              </button>
            {/if}
          </td>
          <td class="py-2 text-muted-foreground font-mono text-xs">{def.keys}</td>
          <td class="py-2">
            {#if isOverridden}
              <button
                class="text-xs text-muted-foreground hover:text-fg"
                onclick={() => resetBinding(def.id)}
              >
                Reset
              </button>
            {/if}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
```

- [ ] **Step 2: Commit**

```
feat(frontend): implement Keybinding Settings page with capture and conflict detection
```

---

## Task 12: Cluster Settings Pages

**Files:**
- Modify: `frontend/src/routes/settings/ClusterListSettings.svelte`
- Modify: `frontend/src/routes/settings/ClusterSettings.svelte`

- [ ] **Step 1: Implement ClusterListSettings**

```svelte
<script lang="ts">
  import { push } from 'svelte-spa-router'
  import { clusterStore } from '$lib/stores/cluster.svelte'

  const contexts = $derived(clusterStore.contexts)
</script>

<div class="max-w-3xl space-y-6">
  <h2 class="text-base font-medium text-fg">Cluster Settings</h2>
  <p class="text-sm text-muted-foreground">Configure per-cluster overrides. Settings not specified here inherit from global defaults.</p>

  <div class="space-y-1">
    {#each contexts as ctx}
      <button
        class="w-full flex items-center justify-between px-4 py-3 rounded border border-border hover:bg-surface-hover transition-colors text-left"
        onclick={() => push(`/settings/clusters/${encodeURIComponent(ctx.name)}`)}
      >
        <div class="flex items-center gap-3">
          <span class="text-sm font-medium text-fg">{ctx.displayName || ctx.name}</span>
          {#if ctx.name !== (ctx.displayName || ctx.name)}
            <span class="text-xs text-muted-foreground">{ctx.name}</span>
          {/if}
        </div>
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clip-rule="evenodd" />
        </svg>
      </button>
    {/each}
  </div>
</div>
```

- [ ] **Step 2: Implement ClusterSettings**

```svelte
<script lang="ts">
  import { onMount } from 'svelte'
  import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'

  interface Props {
    ctxName: string
  }

  let { ctxName }: Props = $props()

  let readOnly = $state<boolean | null>(null)
  let compactRows = $state<boolean | null>(null)
  let accentColor = $state<string | null>(null)
  let displayName = $state<string | null>(null)
  let favoriteNS = $state<string>('')
  let favoriteList = $state<string[]>([])

  onMount(async () => {
    try {
      const prefs = await ConfigService.GetClusterPrefs(ctxName)
      if (prefs) {
        readOnly = prefs.readOnly ?? null
        compactRows = prefs.compactRows ?? null
        accentColor = prefs.accentColor ?? null
        displayName = prefs.displayName ?? null
        favoriteList = prefs.favoriteNamespaces ?? []
      }
    } catch {}
  })

  async function save() {
    const prefs: any = {}
    if (readOnly !== null) prefs.readOnly = readOnly
    if (compactRows !== null) prefs.compactRows = compactRows
    if (accentColor !== null) prefs.accentColor = accentColor
    if (displayName !== null) prefs.displayName = displayName
    if (favoriteList.length > 0) prefs.favoriteNamespaces = favoriteList

    try {
      await ConfigService.SetClusterPrefs(ctxName, prefs)
    } catch (e) {
      console.error('Failed to save cluster prefs:', e)
    }
  }

  function addFavorite() {
    const ns = favoriteNS.trim()
    if (ns && !favoriteList.includes(ns)) {
      favoriteList = [...favoriteList, ns]
      favoriteNS = ''
      save()
    }
  }

  function removeFavorite(ns: string) {
    favoriteList = favoriteList.filter((n) => n !== ns)
    save()
  }

  function toggleOverride(field: 'readOnly' | 'compactRows', current: boolean | null) {
    if (current === null) {
      if (field === 'readOnly') readOnly = false
      else compactRows = false
    } else {
      if (field === 'readOnly') readOnly = null
      else compactRows = null
    }
    save()
  }
</script>

<div class="max-w-2xl space-y-8">
  <h2 class="text-base font-medium text-fg">Cluster: {ctxName}</h2>

  <section class="space-y-4">
    <h3 class="text-sm font-medium text-fg">Display Name</h3>
    <div class="flex items-center gap-2">
      <input
        type="text"
        placeholder={ctxName}
        value={displayName ?? ''}
        oninput={(e) => { displayName = e.currentTarget.value || null }}
        onblur={save}
        class="w-full max-w-xs px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
      />
      {#if displayName !== null}
        <button class="text-xs text-muted-foreground hover:text-fg" onclick={() => { displayName = null; save() }}>
          Use default
        </button>
      {/if}
    </div>
  </section>

  <section class="space-y-4">
    <h3 class="text-sm font-medium text-fg">Accent Color</h3>
    <div class="flex items-center gap-3">
      <input
        type="color"
        value={accentColor ?? '#6366f1'}
        onchange={(e) => { accentColor = e.currentTarget.value; save() }}
        class="w-8 h-8 rounded cursor-pointer border-0 p-0"
      />
      {#if accentColor !== null}
        <button class="text-xs text-muted-foreground hover:text-fg" onclick={() => { accentColor = null; save() }}>
          Use global default
        </button>
      {/if}
    </div>
  </section>

  <section class="space-y-4">
    <h3 class="text-sm font-medium text-fg">Read-Only Mode</h3>
    <label class="flex items-center gap-3 cursor-pointer">
      <input
        type="checkbox"
        checked={readOnly ?? false}
        disabled={readOnly === null}
        onchange={(e) => { readOnly = e.currentTarget.checked; save() }}
        class="accent-accent"
      />
      <span class="text-sm text-fg">
        {readOnly === null ? 'Using global default' : readOnly ? 'Enabled' : 'Disabled'}
      </span>
    </label>
    <button
      class="text-xs text-muted-foreground hover:text-fg"
      onclick={() => toggleOverride('readOnly', readOnly)}
    >
      {readOnly === null ? 'Override' : 'Use global default'}
    </button>
  </section>

  <section class="space-y-4">
    <h3 class="text-sm font-medium text-fg">Favorite Namespaces</h3>
    <div class="flex gap-2">
      <input
        type="text"
        placeholder="Namespace name"
        bind:value={favoriteNS}
        onkeydown={(e) => { if (e.key === 'Enter') addFavorite() }}
        class="flex-1 max-w-xs px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
      />
      <button
        class="px-3 py-1.5 rounded bg-accent text-accent-foreground text-sm hover:opacity-90"
        onclick={addFavorite}
      >
        Add
      </button>
    </div>
    {#if favoriteList.length > 0}
      <div class="flex flex-wrap gap-2">
        {#each favoriteList as ns}
          <span class="flex items-center gap-1 px-2 py-1 rounded bg-surface text-sm text-fg border border-border">
            {ns}
            <button class="text-muted-foreground hover:text-destructive" onclick={() => removeFavorite(ns)}>
              &times;
            </button>
          </span>
        {/each}
      </div>
    {/if}
  </section>
</div>
```

- [ ] **Step 3: Commit**

```
feat(frontend): implement Cluster list and per-cluster Settings pages
```

---

## Task 13: Saved Filters Settings Page

**Files:**
- Modify: `frontend/src/routes/settings/FilterSettings.svelte`

- [ ] **Step 1: Implement FilterSettings**

```svelte
<script lang="ts">
  import { onMount } from 'svelte'
  import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'
  import { descriptorRegistry } from '$lib/registry/index'
  import type { SavedFilter } from '$lib/stores/preferences.svelte'

  let filtersByGVR = $state<Record<string, SavedFilter[]>>({})
  let editingGVR = $state<string | null>(null)
  let editingIndex = $state<number>(-1)
  let editName = $state('')
  let editLabels = $state('')
  let editAnnotations = $state('')
  let editSearch = $state('')

  const gvrNames = $derived(() => {
    const names: Record<string, string> = {}
    for (const gvr of Object.keys(filtersByGVR)) {
      const desc = descriptorRegistry.get(gvr)
      names[gvr] = desc?.resource ?? gvr
    }
    return names
  })

  onMount(async () => {
    try {
      const cfg = await ConfigService.GetConfig()
      if (cfg?.savedFilters) {
        filtersByGVR = cfg.savedFilters
      }
    } catch {}
  })

  function startAdd(gvr: string) {
    editingGVR = gvr
    editingIndex = -1
    editName = ''
    editLabels = ''
    editAnnotations = ''
    editSearch = ''
  }

  function startEdit(gvr: string, index: number) {
    const filter = filtersByGVR[gvr][index]
    editingGVR = gvr
    editingIndex = index
    editName = filter.name
    editLabels = filter.labels ? Object.entries(filter.labels).map(([k, v]) => `${k}=${v}`).join(', ') : ''
    editAnnotations = filter.annotations ? Object.entries(filter.annotations).map(([k, v]) => `${k}=${v}`).join(', ') : ''
    editSearch = filter.search ?? ''
  }

  function parseKV(str: string): Record<string, string> | undefined {
    if (!str.trim()) return undefined
    const result: Record<string, string> = {}
    for (const pair of str.split(',')) {
      const [k, ...rest] = pair.split('=')
      if (k.trim()) result[k.trim()] = rest.join('=').trim()
    }
    return result
  }

  async function saveFilter() {
    if (!editingGVR || !editName.trim()) return

    const filter: SavedFilter = {
      name: editName.trim(),
      labels: parseKV(editLabels),
      annotations: parseKV(editAnnotations),
      search: editSearch.trim() || undefined,
    }

    const gvr = editingGVR
    const filters = [...(filtersByGVR[gvr] ?? [])]
    if (editingIndex >= 0) {
      filters[editingIndex] = filter
    } else {
      filters.push(filter)
    }

    try {
      await ConfigService.SetSavedFilters(gvr, filters)
      filtersByGVR = { ...filtersByGVR, [gvr]: filters }
    } catch (e) {
      console.error('Failed to save filter:', e)
    }

    editingGVR = null
  }

  async function deleteFilter(gvr: string, index: number) {
    const filters = filtersByGVR[gvr].filter((_, i) => i !== index)
    try {
      await ConfigService.SetSavedFilters(gvr, filters)
      if (filters.length === 0) {
        const { [gvr]: _, ...rest } = filtersByGVR
        filtersByGVR = rest
      } else {
        filtersByGVR = { ...filtersByGVR, [gvr]: filters }
      }
    } catch (e) {
      console.error('Failed to delete filter:', e)
    }
  }

  function cancel() {
    editingGVR = null
  }

  let newGVR = $state('')

  function addGVR() {
    const gvr = newGVR.trim()
    if (gvr && !filtersByGVR[gvr]) {
      filtersByGVR = { ...filtersByGVR, [gvr]: [] }
      newGVR = ''
      startAdd(gvr)
    }
  }
</script>

<div class="max-w-3xl space-y-6">
  <h2 class="text-base font-medium text-fg">Saved Filters</h2>
  <p class="text-sm text-muted-foreground">Create reusable filter presets for resource lists.</p>

  {#each Object.entries(filtersByGVR) as [gvr, filters]}
    <section class="space-y-2">
      <h3 class="text-sm font-medium text-fg">{gvr}</h3>
      {#each filters as filter, i}
        <div class="flex items-center justify-between px-3 py-2 rounded border border-border">
          <div>
            <span class="text-sm text-fg font-medium">{filter.name}</span>
            {#if filter.search}
              <span class="ml-2 text-xs text-muted-foreground">search: {filter.search}</span>
            {/if}
          </div>
          <div class="flex gap-2">
            <button class="text-xs text-muted-foreground hover:text-fg" onclick={() => startEdit(gvr, i)}>Edit</button>
            <button class="text-xs text-muted-foreground hover:text-destructive" onclick={() => deleteFilter(gvr, i)}>Delete</button>
          </div>
        </div>
      {/each}
      <button
        class="text-xs text-accent hover:underline"
        onclick={() => startAdd(gvr)}
      >
        + Add filter
      </button>
    </section>
  {/each}

  <div class="flex gap-2 items-center pt-4 border-t border-border">
    <input
      type="text"
      placeholder="GVR (e.g. core.v1.pods)"
      bind:value={newGVR}
      onkeydown={(e) => { if (e.key === 'Enter') addGVR() }}
      class="flex-1 max-w-xs px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
    />
    <button
      class="px-3 py-1.5 rounded bg-accent text-accent-foreground text-sm hover:opacity-90"
      onclick={addGVR}
    >
      Add resource type
    </button>
  </div>

  {#if editingGVR}
    <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={cancel}>
      <div class="bg-bg border border-border rounded-lg p-6 w-full max-w-md space-y-4" onclick|stopPropagation>
        <h3 class="text-base font-medium text-fg">{editingIndex >= 0 ? 'Edit' : 'New'} Filter</h3>
        <div class="space-y-3">
          <input
            type="text"
            placeholder="Filter name"
            bind:value={editName}
            class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
          />
          <input
            type="text"
            placeholder="Labels (key=value, key2=value2)"
            bind:value={editLabels}
            class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
          />
          <input
            type="text"
            placeholder="Annotations (key=value, key2=value2)"
            bind:value={editAnnotations}
            class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
          />
          <input
            type="text"
            placeholder="Search text"
            bind:value={editSearch}
            class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
          />
        </div>
        <div class="flex justify-end gap-2">
          <button class="px-3 py-1.5 rounded border border-border text-sm text-fg hover:bg-surface-hover" onclick={cancel}>Cancel</button>
          <button class="px-3 py-1.5 rounded bg-accent text-accent-foreground text-sm hover:opacity-90" onclick={saveFilter}>Save</button>
        </div>
      </div>
    </div>
  {/if}
</div>
```

Note: The `onclick|stopPropagation` syntax is Svelte 5 — verify this compiles. If not, use `onclick={(e) => e.stopPropagation()}`.

- [ ] **Step 2: Commit**

```
feat(frontend): implement Saved Filters Settings page
```

---

## Task 14: SchemaForm Component + Plugin Settings Page

**Files:**
- Create: `frontend/src/lib/components/SchemaForm.svelte`
- Modify: `frontend/src/routes/settings/PluginListSettings.svelte`
- Modify: `frontend/src/routes/settings/PluginSettings.svelte`

- [ ] **Step 1: Create SchemaForm component**

Create `frontend/src/lib/components/SchemaForm.svelte`:

```svelte
<script lang="ts">
  interface SchemaProperty {
    type?: string
    title?: string
    description?: string
    default?: any
    enum?: string[]
    format?: string
    minimum?: number
    maximum?: number
  }

  interface JSONSchema {
    type: string
    properties?: Record<string, SchemaProperty>
    required?: string[]
  }

  interface Props {
    schema: JSONSchema
    values: Record<string, any>
    onchange: (key: string, value: any) => void
  }

  let { schema, values, onchange }: Props = $props()

  const properties = $derived(
    Object.entries(schema.properties ?? {}).map(([key, prop]) => ({
      key,
      ...prop,
    }))
  )

  function getValue(key: string, prop: SchemaProperty): any {
    return values[key] ?? prop.default ?? (prop.type === 'boolean' ? false : prop.type === 'number' || prop.type === 'integer' ? 0 : '')
  }
</script>

<div class="space-y-6">
  {#each properties as prop}
    <div class="space-y-1">
      <label class="block text-sm font-medium text-fg" for="schema-{prop.key}">
        {prop.title ?? prop.key}
      </label>
      {#if prop.description}
        <p class="text-xs text-muted-foreground">{prop.description}</p>
      {/if}

      {#if prop.type === 'boolean'}
        <label class="flex items-center gap-3 cursor-pointer mt-1">
          <input
            type="checkbox"
            id="schema-{prop.key}"
            checked={getValue(prop.key, prop)}
            onchange={(e) => onchange(prop.key, e.currentTarget.checked)}
            class="accent-accent"
          />
          <span class="text-sm text-muted-foreground">{getValue(prop.key, prop) ? 'Enabled' : 'Disabled'}</span>
        </label>
      {:else if prop.type === 'string' && prop.enum}
        <select
          id="schema-{prop.key}"
          class="w-full max-w-xs px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
          value={getValue(prop.key, prop)}
          onchange={(e) => onchange(prop.key, e.currentTarget.value)}
        >
          {#each prop.enum as option}
            <option value={option}>{option}</option>
          {/each}
        </select>
      {:else if prop.type === 'string' && prop.format === 'color'}
        <input
          type="color"
          id="schema-{prop.key}"
          value={getValue(prop.key, prop) || '#000000'}
          onchange={(e) => onchange(prop.key, e.currentTarget.value)}
          class="w-8 h-8 rounded cursor-pointer border-0 p-0"
        />
      {:else if prop.type === 'number' || prop.type === 'integer'}
        <input
          type="number"
          id="schema-{prop.key}"
          value={getValue(prop.key, prop)}
          min={prop.minimum}
          max={prop.maximum}
          step={prop.type === 'integer' ? 1 : undefined}
          oninput={(e) => onchange(prop.key, Number(e.currentTarget.value))}
          class="w-full max-w-xs px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
        />
      {:else}
        <input
          type="text"
          id="schema-{prop.key}"
          value={getValue(prop.key, prop)}
          oninput={(e) => onchange(prop.key, e.currentTarget.value)}
          class="w-full max-w-xs px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
        />
      {/if}
    </div>
  {/each}
</div>
```

- [ ] **Step 2: Implement PluginListSettings**

```svelte
<script lang="ts">
  import { push } from 'svelte-spa-router'
  import { onMount } from 'svelte'
  import * as PluginService from '../../../bindings/github.com/Vilsol/klados/internal/services/pluginservice.js'

  interface PluginInfo {
    name: string
    displayName: string
  }

  let plugins = $state<PluginInfo[]>([])

  onMount(async () => {
    try {
      const all = await PluginService.ListPlugins()
      const withSettings: PluginInfo[] = []
      for (const p of all ?? []) {
        try {
          const schema = await PluginService.GetPluginSettingsSchema(p.name)
          if (schema) withSettings.push({ name: p.name, displayName: p.displayName || p.name })
        } catch {}
      }
      plugins = withSettings
    } catch {}
  })
</script>

<div class="max-w-3xl space-y-6">
  <h2 class="text-base font-medium text-fg">Plugin Settings</h2>

  {#if plugins.length === 0}
    <p class="text-sm text-muted-foreground">No plugins with configurable settings installed.</p>
  {:else}
    <div class="space-y-1">
      {#each plugins as plugin}
        <button
          class="w-full flex items-center justify-between px-4 py-3 rounded border border-border hover:bg-surface-hover transition-colors text-left"
          onclick={() => push(`/settings/plugins/${plugin.name}`)}
        >
          <span class="text-sm font-medium text-fg">{plugin.displayName}</span>
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-muted-foreground" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clip-rule="evenodd" />
          </svg>
        </button>
      {/each}
    </div>
  {/if}
</div>
```

- [ ] **Step 3: Implement PluginSettings**

```svelte
<script lang="ts">
  import { onMount } from 'svelte'
  import SchemaForm from '$lib/components/SchemaForm.svelte'
  import * as PluginService from '../../../bindings/github.com/Vilsol/klados/internal/services/pluginservice.js'

  interface Props {
    pluginName: string
  }

  let { pluginName }: Props = $props()

  let schema = $state<any>(null)
  let values = $state<Record<string, any>>({})
  let loading = $state(true)

  onMount(async () => {
    try {
      const [schemaStr, settingsStr] = await Promise.all([
        PluginService.GetPluginSettingsSchema(pluginName),
        PluginService.GetPluginSettings(pluginName),
      ])
      if (schemaStr) schema = JSON.parse(schemaStr)
      if (settingsStr) values = JSON.parse(settingsStr)
    } catch (e) {
      console.error('Failed to load plugin settings:', e)
    } finally {
      loading = false
    }
  })

  async function handleChange(key: string, value: any) {
    values = { ...values, [key]: value }
    try {
      await PluginService.SetPluginSettings(pluginName, JSON.stringify(values))
    } catch (e) {
      console.error('Failed to save plugin setting:', e)
    }
  }
</script>

<div class="max-w-2xl space-y-6">
  <h2 class="text-base font-medium text-fg">{pluginName}</h2>

  {#if loading}
    <p class="text-sm text-muted-foreground">Loading...</p>
  {:else if !schema}
    <p class="text-sm text-muted-foreground">This plugin has no configurable settings.</p>
  {:else}
    <SchemaForm {schema} {values} onchange={handleChange} />
  {/if}
</div>
```

- [ ] **Step 4: Verify frontend type-checks**

Run: `cd frontend && pnpm check`

- [ ] **Step 5: Commit**

```
feat(frontend): add SchemaForm component and Plugin Settings pages
```

---

## Task 15: Column Settings Page

**Files:**
- Modify: `frontend/src/routes/settings/ColumnSettings.svelte`

- [ ] **Step 1: Implement ColumnSettings**

This page surfaces the existing `columnPrefs` from config — column order and visibility per GVR. It reads from `ConfigService.GetConfig()` and writes via `ConfigService.SetColumnPrefs()`.

```svelte
<script lang="ts">
  import { onMount } from 'svelte'
  import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'
  import { descriptorRegistry } from '$lib/registry/index'

  interface ColumnPref {
    columns: Record<string, { width?: number }>
    order: string[]
    sort?: { column: string; direction: string }
  }

  let columnPrefs = $state<Record<string, ColumnPref>>({})

  onMount(async () => {
    try {
      const cfg = await ConfigService.GetConfig()
      if (cfg?.columnPrefs) columnPrefs = cfg.columnPrefs
    } catch {}
  })

  async function resetGVR(gvr: string) {
    try {
      await ConfigService.DeleteColumnPrefs(gvr)
      const { [gvr]: _, ...rest } = columnPrefs
      columnPrefs = rest
    } catch (e) {
      console.error('Failed to reset column prefs:', e)
    }
  }
</script>

<div class="max-w-3xl space-y-6">
  <h2 class="text-base font-medium text-fg">Column Customization</h2>
  <p class="text-sm text-muted-foreground">Column order and widths are saved automatically as you adjust them in resource lists. Use this page to review or reset customizations.</p>

  {#if Object.keys(columnPrefs).length === 0}
    <p class="text-sm text-muted-foreground">No column customizations saved yet.</p>
  {:else}
    {#each Object.entries(columnPrefs) as [gvr, prefs]}
      <section class="space-y-2">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-medium text-fg">{gvr}</h3>
          <button
            class="text-xs text-muted-foreground hover:text-destructive"
            onclick={() => resetGVR(gvr)}
          >
            Reset to default
          </button>
        </div>
        <div class="px-3 py-2 rounded border border-border text-xs text-muted-foreground">
          <span>Columns: {prefs.order?.join(', ') || 'default order'}</span>
          {#if prefs.sort}
            <span class="ml-4">Sort: {prefs.sort.column} {prefs.sort.direction}</span>
          {/if}
        </div>
      </section>
    {/each}
  {/if}
</div>
```

- [ ] **Step 2: Commit**

```
feat(frontend): implement Column Settings page
```

---

## Task 16: Accent Color Application

**Files:**
- Modify: `frontend/src/App.svelte` (or relevant CSS/theme location)

- [ ] **Step 1: Apply accent color as CSS custom property**

In `frontend/src/App.svelte`, add an `$effect` that sets the accent color CSS variable when preferences change:

```typescript
  $effect(() => {
    const color = preferencesStore.prefs.accentColor
    if (color) {
      document.documentElement.style.setProperty('--color-accent', color)
    } else {
      document.documentElement.style.removeProperty('--color-accent')
    }
  })
```

This works because Tailwind v4 custom tokens (`accent`, `accent-foreground`) are defined via CSS custom properties. Setting `--color-accent` overrides the default.

Note: Check what the actual CSS custom property name is in the project's Tailwind config. It may be `--accent` or `--color-accent` depending on the Tailwind v4 setup. Grep for the accent color definition in the CSS files.

- [ ] **Step 2: Apply font size as CSS custom property**

```typescript
  $effect(() => {
    const size = preferencesStore.prefs.fontSize
    if (size > 0) {
      document.documentElement.style.setProperty('font-size', `${size}px`)
    } else {
      document.documentElement.style.removeProperty('font-size')
    }
  })
```

- [ ] **Step 3: Add cluster accent color dot in sidebar**

In `frontend/src/lib/components/Sidebar.svelte`, when rendering the cluster context in the sidebar, check if the active cluster has an accent color set:

```svelte
{#if clusterAccentColor}
  <span class="w-2 h-2 rounded-full inline-block" style="background-color: {clusterAccentColor}"></span>
{/if}
```

Read the accent color from `preferencesStore.prefs.accentColor`.

- [ ] **Step 4: Commit**

```
feat(frontend): apply accent color and font size as CSS custom properties
```

---

## Task 17: Frontend Tests

**Files:**
- Create: `frontend/src/lib/__tests__/preferences.svelte.test.ts`

- [ ] **Step 1: Write preferences store tests**

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mockConfigService, mockEvents, resetMocks } from './wails-mock'

vi.mock('../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js', () => mockConfigService)
vi.mock('@wailsio/runtime', () => ({ Events: mockEvents }))

describe('PreferencesStore', () => {
  beforeEach(() => {
    resetMocks()
  })

  it('loads resolved prefs on init', async () => {
    mockConfigService.GetResolvedPrefs.mockResolvedValue({
      theme: 'dark',
      accentColor: '#ff0000',
      fontSize: 16,
      compactRows: true,
      readOnly: false,
      keybindings: { 'command-palette': 'Control+p' },
      savedFilters: null,
      favoriteNamespaces: ['default'],
    })

    const { preferencesStore } = await import('$lib/stores/preferences.svelte')
    await preferencesStore.load('test-ctx')

    expect(mockConfigService.GetResolvedPrefs).toHaveBeenCalledWith('test-ctx')
    expect(preferencesStore.prefs.theme).toBe('dark')
    expect(preferencesStore.prefs.accentColor).toBe('#ff0000')
  })

  it('returns keybinding override', async () => {
    mockConfigService.GetResolvedPrefs.mockResolvedValue({
      theme: 'system',
      accentColor: '',
      fontSize: 0,
      compactRows: false,
      readOnly: false,
      keybindings: { 'command-palette': 'Control+p' },
      savedFilters: null,
      favoriteNamespaces: null,
    })

    const { preferencesStore } = await import('$lib/stores/preferences.svelte')
    await preferencesStore.load('')

    expect(preferencesStore.getKeybinding('command-palette')).toBe('Control+p')
    expect(preferencesStore.getKeybinding('nonexistent')).toBeUndefined()
  })
})
```

- [ ] **Step 2: Run tests**

Run: `cd frontend && npx vitest run src/lib/__tests__/preferences.svelte.test.ts`

Expected: PASS

- [ ] **Step 3: Commit**

```
test(frontend): add preferences store tests
```

---

## Task 18: Go Cascade Resolution Tests (Extended)

**Files:**
- Existing: `internal/config/resolve_test.go` (already created in Task 2)

- [ ] **Step 1: Run full test suite to verify everything integrates**

Run: `go test ./internal/config/ ./internal/services/ -v`

Expected: all tests PASS.

- [ ] **Step 2: Run frontend test suite**

Run: `cd frontend && pnpm test`

Expected: all tests PASS.

- [ ] **Step 3: Run frontend type-check**

Run: `cd frontend && pnpm check`

Expected: no errors.

- [ ] **Step 4: Commit any test fixes needed**

If any tests needed fixing due to the new mock methods or changed store interfaces, commit those fixes.

```
fix: update test mocks for new ConfigService preferences methods
```

---

## Task 19: Final Integration Verification

- [ ] **Step 1: Regenerate Wails bindings one final time**

Run: `wails3 generate bindings`

- [ ] **Step 2: Verify Go build**

Run: `go build ./...`

- [ ] **Step 3: Verify frontend build**

Run: `cd frontend && pnpm build`

- [ ] **Step 4: Run all Go tests**

Run: `go test ./internal/config/ -v`

- [ ] **Step 5: Run all frontend tests**

Run: `cd frontend && pnpm test`

- [ ] **Step 6: Final commit if any adjustments were needed**

```
chore: final integration fixes for preferences system
```
