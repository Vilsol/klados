# Saved Filters — Implementation Reference

How saved filters are stored, resolved, and accessed. Use this when wiring them into `ResourceListPage` / `ResourceList`.

## Data Model

```go
// internal/config/config.go
type SavedFilter struct {
    Name        string            `json:"name"`
    Labels      map[string]string `json:"labels,omitempty"`
    Annotations map[string]string `json:"annotations,omitempty"`
    Search      string            `json:"search,omitempty"`
}
```

Each filter is a named preset combining any of: label selectors, annotation selectors, and a text search string. All fields are optional except `Name`.

## Storage

Saved filters live in `config.json` at two levels:

```jsonc
{
  // Global filters — apply to any cluster
  "savedFilters": {
    "core.v1.pods": [
      { "name": "erroring", "labels": { "app": "web" }, "search": "CrashLoop" }
    ]
  },
  // Per-cluster filters — scoped to a single cluster context
  "clusters": {
    "prod-ctx": {
      "savedFilters": {
        "core.v1.pods": [
          { "name": "prod-only", "labels": { "env": "prod" } }
        ]
      }
    }
  }
}
```

The map key is the GVR string (e.g. `core.v1.pods`, `apps.v1.deployments`).

## Cascade Resolution

`ResolveForCluster(ctxName)` merges both levels: global filters come first, then per-cluster filters are **appended** (not replaced). For the example above, resolving for `prod-ctx` yields 2 filters for `core.v1.pods`.

```go
// internal/config/resolve.go — relevant excerpt
for gvr, filters := range cluster.SavedFilters {
    r.SavedFilters[gvr] = append(r.SavedFilters[gvr], filters...)
}
```

## Backend RPCs

| Method | Purpose |
|---|---|
| `ConfigService.GetSavedFilters(gvr)` | Read global filters for a specific GVR |
| `ConfigService.SetSavedFilters(gvr, filters)` | Write global filters for a GVR (empty array deletes the key) |
| `ConfigService.SetClusterSavedFilters(ctx, gvr, filters)` | Write per-cluster filters for a GVR |
| `ConfigService.GetResolvedPrefs(ctx)` | Returns `ResolvedPrefs` with merged `savedFilters` map |

All write methods trigger `config:updated` → preferencesStore auto-refreshes.

## Frontend Access

```typescript
// Preferred: use preferencesStore (already cascade-resolved)
import { preferencesStore } from '$lib/stores/preferences.svelte'

// Get all filters for the current cluster + GVR:
const filters = preferencesStore.getSavedFilters('core.v1.pods')
// Returns SavedFilter[] — global + per-cluster merged

// The interface:
interface SavedFilter {
  name: string
  labels?: Record<string, string>
  annotations?: Record<string, string>
  search?: string
}
```

`preferencesStore.prefs.savedFilters` is the full `Record<string, SavedFilter[]>` map. It re-fetches automatically when:
- The active cluster context changes (via `$effect` in `App.svelte`)
- Any config write occurs (via `config:updated` Wails event subscription)

## Integration into ResourceList (TODO)

The intended UX is a dropdown in the resource list toolbar that lets the user pick a saved filter. When selected, it should apply the filter's fields to the existing filter state:

1. **Labels** → set as active label filter (same format as the existing label filter)
2. **Annotations** → set as active annotation filter
3. **Search** → populate the search input

The filter is a **preset** — selecting it populates the filter inputs, but the user can then modify them freely. It doesn't lock the UI to the preset.

```
┌─ Resource List Toolbar ────────────────────────────┐
│ [Search...] [Labels...] [Annotations...] [▾ Saved] │
└────────────────────────────────────────────────────┘
```

The `[▾ Saved]` dropdown should:
- List `preferencesStore.getSavedFilters(currentGVR)`
- Show filter name + a summary (e.g. "app=web, search: CrashLoop")
- On select, populate the filter inputs
- Optionally include a "Save current filters" action that opens the filter settings or saves inline

## Settings UI

Filters are managed in `/settings/filters` (`FilterSettings.svelte`). The settings page reads/writes global filters via `ConfigService.SetSavedFilters()`. Per-cluster filters can be managed via `ConfigService.SetClusterSavedFilters()` (the scope toggle in the settings UI).
