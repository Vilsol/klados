# Preferences & Settings System — Design Spec

> Date: 2026-04-10

## Overview

A comprehensive preferences and settings system for Klados with a dedicated settings UI, two-level config cascade (global → per-cluster), plugin settings via JSON Schema, rebindable keybindings, saved filters, and related customization features.

### Features Covered

- Per-cluster settings with global fallthrough
- UI preferences (theme, accent color, font size, compact rows, startup behavior)
- Cluster-specific color coding and display names
- Namespace favorites / pinning
- Rebindable keybindings
- Saved filter presets (per-GVR, scoped global or per-cluster)
- Custom column definitions (existing `columnPrefs`, surfaced in settings UI)
- Plugin settings pages (schema-driven from manifest)
- Configurable font size

## Architecture

### Approach: Extend Existing Config

The current `config.json` is extended with new fields. No new persistence layer — the existing `Config` struct with its mutex-protected `Update`/`Read`/`Save` pattern handles everything. New fields use `omitempty` so existing config files load without migration.

### Config Schema

```go
type Config struct {
    // --- Existing fields (become global defaults) ---
    Theme                 string                        `json:"theme"`
    KubeconfigPaths       []string                      `json:"kubeconfigPaths"`
    TerminalWebGL         bool                          `json:"terminalWebGL"`
    DisabledPlugins       []string                      `json:"disabledPlugins,omitempty"`
    InsecureRegistries    []string                      `json:"insecureRegistries,omitempty"`
    InsecureSkipTLSVerify bool                          `json:"insecureSkipTLSVerify,omitempty"`
    Metrics               map[string]*MetricsConfig     `json:"metrics,omitempty"`
    ColumnPrefs           map[string]*GVRColumnPrefs    `json:"columnPrefs,omitempty"`
    CompactRows           bool                          `json:"compactRows,omitempty"`
    ReadOnly              bool                          `json:"readOnly,omitempty"`
    PortForwards          map[string][]SavedPortForward `json:"portForwards,omitempty"`

    // --- New fields ---
    Clusters        map[string]*ClusterPrefs  `json:"clusters,omitempty"`
    Keybindings     map[string]string         `json:"keybindings,omitempty"`
    SavedFilters    map[string][]SavedFilter  `json:"savedFilters,omitempty"`
    StartupBehavior string                    `json:"startupBehavior,omitempty"`
    StartupCluster  string                    `json:"startupCluster,omitempty"`
    AccentColor     string                    `json:"accentColor,omitempty"`     // hex color, e.g. "#ff6b6b"
    FontSize        int                       `json:"fontSize,omitempty"`        // base font size in px (0 = default 14)
}
```

### ClusterPrefs (Per-Cluster Overrides)

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
```

Pointer types for override fields: `nil` = inherit from global.

### SavedFilter

```go
type SavedFilter struct {
    Name        string            `json:"name"`
    Labels      map[string]string `json:"labels,omitempty"`
    Annotations map[string]string `json:"annotations,omitempty"`
    Search      string            `json:"search,omitempty"`
}
```

Per-GVR filter presets. The map key in `SavedFilters` is the GVR string (e.g. `"core.v1.pods"`). Each filter has a name and any combination of label selectors, annotation filters, and text search.

### Cascade Resolution

```go
type ResolvedPrefs struct {
    Theme       string
    AccentColor string
    FontSize    int
    CompactRows bool
    ReadOnly     bool
    TerminalWebGL bool
    Metrics      *MetricsConfig
    ColumnPrefs map[string]*GVRColumnPrefs
    FavoriteNS    []string
    Keybindings   map[string]string
    SavedFilters  map[string][]SavedFilter
}

func (c *Config) ResolveForCluster(ctxName string) ResolvedPrefs {
    // 1. Start with hardcoded defaults
    // 2. Overlay global config fields
    // 3. If clusters[ctxName] exists, overlay non-nil pointer fields
}
```

Pure computation, no caching. Called on demand via `ConfigService.GetResolvedPrefs(ctxName)`.

## Plugin Settings

### Manifest Declaration

Plugins declare settings in their manifest under `extensions.settings`:

```yaml
extensions:
  settings:
    schema:
      type: object
      properties:
        refreshInterval:
          type: number
          title: "Refresh Interval (seconds)"
          default: 30
          minimum: 5
        showNotifications:
          type: boolean
          title: "Show Notifications"
          default: true
        endpoint:
          type: string
          title: "API Endpoint"
          description: "Custom endpoint URL"
```

### Backend

- `Extensions` struct gets a new `Settings *SettingsSchema` field
- `SettingsSchema` holds the raw JSON Schema as `json.RawMessage`
- Writes validated against the schema using `santhosh-tekuri/jsonschema/v6`
- Values stored in existing plugin storage (`$XDG_DATA_HOME/klados/plugins/{name}/storage.json`) under a `"settings"` key
- New host API methods: `settings.get`, `settings.set` for Wasm plugin access

### Frontend Rendering

The `SchemaForm.svelte` component auto-renders form fields from JSON Schema:

| JSON Schema type | Rendered as |
|---|---|
| `string` | Text input |
| `string` + `enum` | Select dropdown |
| `string` + `format: "color"` | Color picker |
| `number` / `integer` | Number input (respects `minimum`/`maximum`) |
| `boolean` | Toggle switch |

`title` → label, `description` → help text, `default` → placeholder/initial value.

## Settings UI

### Route Structure

```
/settings                    → redirect to /settings/general
/settings/general            → theme, font size, startup behavior, terminal WebGL
/settings/appearance         → accent color, compact rows
/settings/clusters           → list of known clusters, click to edit overrides
/settings/clusters/:ctx      → per-cluster: color, read-only, display name, favorites, metrics
/settings/keybindings        → table of action → key combo, rebindable
/settings/filters            → saved filter management, grouped by GVR
/settings/columns            → column visibility/order, grouped by GVR
/settings/plugins            → lists plugins with settings schemas
/settings/plugins/:name      → auto-rendered form from plugin JSON Schema
```

### Settings Sidebar

```
General
Appearance
Clusters
Keybindings
Filters
Columns
─────────
Plugins
  +-- plugin-a
  +-- plugin-b
```

Only plugins with a `settings.schema` in their manifest appear. Divider separates core from plugin settings.

### Entry Point

Gear icon in the main sidebar footer. Navigates to `/settings`. Back button or sidebar cluster list returns to the previous view.

### Keybinding Editor

- Table: Action name | Current binding | Default binding
- Click row → "listening" mode → next keypress captured as new combo
- "Reset to default" per row and "Reset all" global
- Conflict highlighting when two actions share the same combo

### Cluster Settings

- Left column: list of all known clusters (from kubeconfigs)
- Click → right panel shows overridable fields
- Each field has a "Use global default" toggle (sets pointer to `nil`)
- Accent color shown as colored dot in main sidebar when set

### Saved Filters

- Grouped by GVR (e.g. "Pods", "Deployments")
- Each shows name, label selectors, annotation filters, search text
- Add/edit via inline form, delete with confirmation
- Scope toggle: Global or per-cluster (stored at the appropriate cascade level)

## Reactivity

### Config Update Event

`Config.Save()` emits a `config:updated` Wails event after writing to disk. All stores that depend on config re-fetch. Single reactivity mechanism — no polling.

### Frontend `preferencesStore`

New store (`preferences.svelte.ts`):
- Calls `GetResolvedPrefs(activeContext)` on init and on `activeContext` change
- Exposes resolved values: theme, accentColor, compactRows, fontSize, readOnly, keybindings, etc.
- Listens for `config:updated` event to re-fetch
- Replaces scattered `GetConfig()` calls (e.g. `clusterStore.isReadOnly`, theme logic)

### Keybindings Resolution

`ShortcutStore` reads `keybindings` from preferences store on init. For each registered shortcut, user-defined bindings override the default `keys`. The settings page shows all registered actions with current (possibly overridden) key combos.

## Migration & Backwards Compatibility

**Zero migration.** Existing config fields remain in place as global defaults. New fields are `omitempty` — an old config file loads cleanly. The `startupBehavior` field defaults to `"last"` (current behavior).

**Startup behavior options:**
- `"last"` — reconnect to previous session's clusters (current behavior)
- `"chooser"` — always show cluster list
- `"specific"` — auto-connect to cluster named in `startupCluster`

**Scattered settings consolidation:** Existing one-off controls (theme toggle, compact rows, font size) continue to work — they write to config via `ConfigService.UpdateConfig()`. The settings page is an additional centralized surface for the same values.

**Plugin settings:** The `"settings"` key in plugin storage is separate from existing plugin storage keys. Plugins without a `settings` schema are unaffected.
