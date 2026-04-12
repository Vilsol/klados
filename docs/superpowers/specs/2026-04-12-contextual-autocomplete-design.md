# Contextual Autocomplete

## Summary

When enabled (default: on), autocomplete suggestions in the resource list search bar reflect only items matching already-committed filter chips, rather than the entire unfiltered item list.

## Motivation

Currently, typing `label:foo=bar ` then starting a new token like `label:` shows autocomplete suggestions drawn from all items in the list. This is confusing when the user is building a compound filter — they see label keys/values that don't exist in the already-filtered subset, leading to zero-result filters.

## Behavior

- **Setting on (default):** `getSuggestions()` receives `filterItems(items, chips)` — only items matching completed chips. The in-progress token is excluded from filtering so the user can still see suggestions for what they're typing.
- **Setting off:** `getSuggestions()` receives the full `items` array (current behavior).

## Changes

### Go backend

#### `internal/config/config.go`

Add `ContextualAutocomplete *bool` to the `Config` struct. Using a pointer allows distinguishing "not set" (nil, defaults to enabled) from explicitly `false` (user disabled it). This follows the same `*bool` pattern used by `ClusterPrefs.ReadOnly` and `ClusterPrefs.CompactRows`.

#### `internal/config/resolve.go`

Add `ContextualAutocomplete bool` to `ResolvedPrefs`. In `ResolveForCluster`, default to `true` when `cfg.ContextualAutocomplete` is nil:

```go
if c.ContextualAutocomplete != nil {
    r.ContextualAutocomplete = *c.ContextualAutocomplete
} else {
    r.ContextualAutocomplete = true
}
```

#### `internal/services/config.go`

Add `SetContextualAutocomplete(enabled bool) error` — stores `&enabled` into `cfg.ContextualAutocomplete`.

### Frontend

#### `frontend/src/lib/stores/preferences.svelte.ts`

Add `contextualAutocomplete: boolean` to `ResolvedPrefs`, default `true`.

#### `frontend/src/lib/components/SmartSearch.svelte`

In `updateAutocomplete()`:

```ts
import { filterItems } from '$lib/search/filter'
import { preferencesStore } from '$lib/stores/preferences.svelte'

function updateAutocomplete() {
  if (!inputEl) return
  const pool = preferencesStore.prefs.contextualAutocomplete
    ? filterItems(items, chips)
    : items
  suggestions = getSuggestions(value, value.length, pool)
  selectedIndex = 0
  showAutocomplete = suggestions.length > 0
}
```

#### `frontend/src/routes/settings/AppearanceSettings.svelte`

Add a checkbox following the existing "Compact Rows" pattern:

- Label: "Contextual autocomplete"
- Description: "Autocomplete suggestions reflect active search filters"
- Calls `ConfigService.SetContextualAutocomplete(checked)`

### Tests

- Update `autocomplete.test.ts` to add a test verifying that when a pre-filtered subset is passed, suggestions only reflect that subset.
- No new test file needed — the logic change is in the caller (`SmartSearch`), and the `getSuggestions` function itself is unchanged.

## Non-goals

- No per-cluster override for this setting (unlike `readOnly`/`compactRows`).
- No inline toggle in the search bar.
- No changes to the filter logic itself.
