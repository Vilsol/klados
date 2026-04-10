# Smart Search & Saved Filters — Design Spec

Unified tokenized search bar for ResourceList that replaces the separate search input and AnnotationFilter with a single chip-based smart search, integrated with the existing saved filters system.

## Parser

**Library:** `@muhgholy/search-query-parser` v3 — TypeScript-first, zero deps, Gmail-like syntax with custom operators.

**Operator config:**

| Operator | Aliases | Match target |
|---|---|---|
| `label` | `l` | Label key=value or key exists |
| `annotation` | `ann` | Annotation key=value or key exists |
| `name` | `n` | Resource name substring |
| `namespace` | `ns` | Namespace exact match |

**Syntax examples:**

```
label:app=web                   → label with key=value
l:app=web                       → alias
-name:test                      → negated name match
"crash loop"                    → quoted phrase on name
l:app=web -ns:kube-system nginx → multiple terms, ANDed
```

Bare text (`text`/`phrase` term types) matches resource name only (case-insensitive substring).

## Smart Search Component

### SmartSearch.svelte

Single text input that parses tokens via the parser on each keystroke. Completed tokens (followed by space) render as colored chips inline.

```
┌──────────────────────────────────────────────────────┬───┐
│ l:app=web  -name:test  nginx▊                        │ ☆ │
└──────────────────────────────────────────────────────┴───┘
  ╭─chip──╮  ╭──chip───╮  ╭─trailing text input──╮
```

**Chip behavior:**
- Color-coded by qualifier type (label=blue, annotation=purple, name=default, namespace=green)
- Negated chips have visual distinction (red tint or strikethrough)
- Backspace into a chip selects it; another backspace deletes it
- Click a chip to expand it back to editable text

**Props:**
- `items: Unstructured[]` — for autocomplete data extraction
- `gvr: string` — for saved filter lookup
- `contextName: string` — for per-cluster saved filter saves
- `onTermsChange: (terms: TParsedTerm[]) => void` — or bindable `terms`

### SmartSearchAutocomplete.svelte

Floating popup that appears contextually based on cursor position.

| Cursor position | Shows |
|---|---|
| Empty input or bare text `lab▊` | Qualifier suggestions: `label:`, `name:`, `namespace:`, `annotation:` |
| After qualifier `label:▊` | Distinct label keys from current items (with count) |
| After key= `label:app=▊` | Distinct values for that key from current items |
| After `-` then text `-lab▊` | Qualifier suggestions (negated context) |
| Mid-word bare text `ngi▊` | No popup — free-form name filter |

**Data source:** Derived from unfiltered `ResourceStore.items`. Memoized to avoid rescanning on every keystroke.

**Interaction:**
- Arrow keys navigate, Enter/Tab selects, Escape dismisses
- Typing narrows the suggestion list (prefix match)
- Selecting a label key appends `=` and immediately shows value suggestions
- Selecting a value appends a space and returns to free typing

### SavedFilterDropdown.svelte

Dropdown triggered by the `☆` button at the right edge of the search bar.

**Listing:**
- Reads `preferencesStore.getSavedFilters(currentGVR)`
- Each entry shows: name, preview of serialized query, scope indicator `(g)` global / `(c)` per-cluster
- Selecting a filter replaces search bar contents with serialized terms

**Save popover:**
- Triggered by "+ Save current filter" action (disabled when search bar is empty)
- Fields: name (text input), scope (radio: "This cluster" default / "Global")
- Parses current search terms back into a `SavedFilter` object:
  - `text`/`phrase` terms → `search` field
  - `label` terms → `labels` map
  - `annotation` terms → `annotations` map
  - `name`/`namespace` terms are not part of the SavedFilter model — they serialize into the `search` field preserving qualifier syntax (e.g. `name:nginx ns:prod` stays as-is in `search`)
- Calls `ConfigService.SetClusterSavedFilters` or `ConfigService.SetSavedFilters` based on scope
- `config:updated` event auto-refreshes the preferences store

**Edge cases:**
- Duplicate name → overwrite with confirmation
- No saved filters for this GVR → dropdown shows only the save action

## Filter Logic

All parsed terms are ANDed together to produce the filter predicate.

| Term type | Match logic |
|---|---|
| `text` / `phrase` | Case-insensitive substring on resource name |
| `name` | Case-insensitive substring on resource name |
| `namespace` | Exact match on resource namespace |
| `label` | `key=value` → exact match. `key` alone → key exists |
| `annotation` | Same as label, on annotations |
| Negated | Inverts the match |

Filtering runs against unfiltered items. Autocomplete also uses unfiltered items so suggestions don't disappear as results narrow.

Namespace filtering from `clusterStore.selectedNamespaces` remains a separate concern, applied independently.

## File Changes

### New files

```
frontend/src/lib/components/SmartSearch.svelte
frontend/src/lib/components/SmartSearchAutocomplete.svelte
frontend/src/lib/components/SavedFilterDropdown.svelte
frontend/src/lib/search/parser.ts          — wraps parser with Klados operator config
frontend/src/lib/search/filter.ts          — TParsedTerm[] + items → filtered items
frontend/src/lib/search/autocomplete.ts    — extracts suggestions from items by cursor context
frontend/src/lib/search/serialize.ts       — SavedFilter ↔ query string conversion
```

### Modified files

```
frontend/src/lib/components/ResourceList.svelte
  — Replace search input + AnnotationFilter with SmartSearch
  — Remove filterText and annotationFilters state
  — Wire filtered items through new filter.ts
```

### Removed files

```
frontend/src/lib/components/AnnotationFilter.svelte
  — Functionality absorbed into SmartSearch
```

## Data Flow

```
                    ResourceStore.items (unfiltered)
                           │
              ┌────────────┼────────────────┐
              ▼            ▼                ▼
        SmartSearch    autocomplete.ts   filter.ts
        (raw input)    (suggestions)    (predicate)
              │                             │
              ▼                             ▼
         parser.ts                   filtered items[]
      (TParsedTerm[])                       │
              │                             ▼
              ├──────────────────→   ResourceList table
              │
              ▼
        SavedFilterDropdown
         (serialize ↔ parse)
              │
              ▼
        preferencesStore / ConfigService RPCs
```

## Dependencies

- `@muhgholy/search-query-parser` v3 — added to frontend/package.json
