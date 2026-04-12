# Contextual Autocomplete Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When a settings toggle is enabled (default: on), autocomplete suggestions in the resource list search bar only reflect items matching already-committed filter chips.

**Architecture:** Add a `*bool` config field with nil-defaults-to-true semantics. The frontend reads the resolved preference and conditionally pre-filters items before passing them to `getSuggestions()`. A new checkbox in AppearanceSettings controls the toggle.

**Tech Stack:** Go (config layer), Svelte 5 (SmartSearch + settings UI), Vitest (frontend tests)

---

### Task 1: Go backend — config field, resolver, and service setter

**Files:**
- Modify: `internal/config/config.go:63-86` (Config struct)
- Modify: `internal/config/resolve.go:1-15` (ResolvedPrefs struct), `resolve.go:17-31` (ResolveForCluster)
- Modify: `internal/services/config.go:85-89` (add setter after SetCompactRows)

- [ ] **Step 1: Add `ContextualAutocomplete *bool` to Config struct**

In `internal/config/config.go`, add the field to the `Config` struct after `FontSize`:

```go
ContextualAutocomplete *bool `json:"contextualAutocomplete,omitempty"`
```

This is the first `*bool` at the top-level Config (others use plain `bool`). The pointer is necessary because the feature defaults to *on*, so we must distinguish "user hasn't set it" (nil → true) from "user explicitly disabled it" (false).

- [ ] **Step 2: Add field to ResolvedPrefs and resolve it**

In `internal/config/resolve.go`, add to the `ResolvedPrefs` struct:

```go
ContextualAutocomplete bool `json:"contextualAutocomplete"`
```

In `ResolveForCluster`, add this block after the `r := ResolvedPrefs{...}` literal (before the Metrics block). Since this is a `*bool`, it can't be set in the struct literal — set it right after:

```go
if c.ContextualAutocomplete != nil {
    r.ContextualAutocomplete = *c.ContextualAutocomplete
} else {
    r.ContextualAutocomplete = true
}
```

- [ ] **Step 3: Add SetContextualAutocomplete to ConfigService**

In `internal/services/config.go`, add after `SetCompactRows`:

```go
func (c *ConfigService) SetContextualAutocomplete(enabled bool) error {
	return c.config.Update(func(cfg *config.Config) {
		cfg.ContextualAutocomplete = &enabled
	})
}
```

- [ ] **Step 4: Regenerate Wails bindings**

Run: `wails3 generate bindings`

This generates the TypeScript binding for `SetContextualAutocomplete` in `frontend/bindings/`.

- [ ] **Step 5: Verify Go compiles**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 6: Commit**

```
feat: add contextualAutocomplete config field with true-by-default semantics
```

---

### Task 2: Frontend — preferences store, SmartSearch behavior, settings UI, and test

**Files:**
- Modify: `frontend/src/lib/stores/preferences.svelte.ts` (ResolvedPrefs interface + default)
- Modify: `frontend/src/lib/components/SmartSearch.svelte` (updateAutocomplete)
- Modify: `frontend/src/routes/settings/AppearanceSettings.svelte` (add checkbox)
- Modify: `frontend/src/lib/__tests__/autocomplete.test.ts` (add filtered-subset test)

- [ ] **Step 1: Add `contextualAutocomplete` to frontend ResolvedPrefs**

In `frontend/src/lib/stores/preferences.svelte.ts`, add to the `ResolvedPrefs` interface:

```ts
contextualAutocomplete: boolean
```

And in the default value of `prefs`:

```ts
contextualAutocomplete: true,
```

- [ ] **Step 2: Update SmartSearch to use contextual filtering**

In `frontend/src/lib/components/SmartSearch.svelte`, add two imports:

```ts
import { filterItems } from '$lib/search/filter'
import { preferencesStore } from '$lib/stores/preferences.svelte'
```

Replace the `updateAutocomplete` function:

```ts
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

The key detail: `chips` (not `terms`) is used for filtering. `chips` excludes the in-progress token the user is actively typing, so the autocomplete won't try to filter against an incomplete term.

- [ ] **Step 3: Add checkbox to AppearanceSettings**

In `frontend/src/routes/settings/AppearanceSettings.svelte`:

Add state and init in `<script>`:

```ts
let contextualAutocomplete = $state<boolean>(true)
```

In `onMount`, add:

```ts
contextualAutocomplete = preferencesStore.prefs.contextualAutocomplete
```

Add setter function:

```ts
function setContextualAutocomplete(checked: boolean) {
  contextualAutocomplete = checked
  ConfigService.SetContextualAutocomplete(checked)
}
```

Add the checkbox in the template after the Compact Rows `<div>`:

```svelte
<div>
  <h2 class="text-base font-medium text-fg mb-4">Contextual Autocomplete</h2>
  <label class="flex items-center gap-2 cursor-pointer">
    <input
      type="checkbox"
      checked={contextualAutocomplete}
      onchange={(e) => setContextualAutocomplete((e.target as HTMLInputElement).checked)}
      class="accent-accent"
    />
    <span class="text-sm text-fg">Autocomplete suggestions reflect active search filters</span>
  </label>
</div>
```

- [ ] **Step 4: Add test for contextual autocomplete behavior**

In `frontend/src/lib/__tests__/autocomplete.test.ts`, add a new `describe` block at the end (after the existing `describe('getSuggestions', ...)`):

```ts
describe('contextual autocomplete (pre-filtered items)', () => {
  const allItems = [
    makeItem('nginx-proxy', 'default', { app: 'web', env: 'prod' }),
    makeItem('nginx-ingress', 'kube-system', { app: 'web', env: 'dev' }),
    makeItem('redis-master', 'default', { app: 'cache', env: 'prod' }),
  ]

  // Simulate pre-filtering: only items with env=prod
  const filtered = allItems.filter((i) => i.metadata.labels.env === 'prod')

  it('suggests only label values present in filtered subset', () => {
    // With full list: app values would be "web" (2) and "cache" (1)
    // With filtered (env=prod only): app values are "web" (1) and "cache" (1)
    const result = getSuggestions('label:app=', 10, filtered)
    const values = result.map((s: Suggestion) => s.value)
    expect(values).toContain('web')
    expect(values).toContain('cache')
    // "dev" env items are excluded, but both prod items have different app values
    // so the count should reflect the filtered set
    expect(result.find((s: Suggestion) => s.value === 'web')?.count).toBe(1)
  })

  it('does not suggest label keys absent from filtered subset', () => {
    // Filter to only redis-master (no annotations)
    const redisOnly = allItems.filter((i) => i.metadata.name === 'redis-master')
    const result = getSuggestions('label:', 6, redisOnly)
    const keys = result.map((s: Suggestion) => s.value)
    expect(keys).toContain('app')
    expect(keys).toContain('env')
    expect(result.find((s: Suggestion) => s.value === 'app')?.count).toBe(1)
  })

  it('suggests only namespaces present in filtered subset', () => {
    const defaultOnly = allItems.filter((i) => i.metadata.namespace === 'default')
    const result = getSuggestions('ns:', 3, defaultOnly)
    const ns = result.map((s: Suggestion) => s.value)
    expect(ns).toContain('default')
    expect(ns).not.toContain('kube-system')
  })
})
```

- [ ] **Step 5: Run tests and type-check**

Run: `cd frontend && npx vitest run src/lib/__tests__/autocomplete.test.ts`
Expected: all tests pass.

Run: `cd frontend && pnpm check`
Expected: no type errors.

- [ ] **Step 6: Commit**

```
feat: contextual autocomplete — filter suggestions by active chips
```
