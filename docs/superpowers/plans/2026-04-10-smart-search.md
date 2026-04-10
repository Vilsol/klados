# Smart Search & Saved Filters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the separate search input and AnnotationFilter in ResourceList with a unified tokenized smart search bar that supports `label:`, `annotation:`, `name:`, `namespace:` qualifiers with autocomplete, negation, and saved filter integration.

**Architecture:** A new `SmartSearch.svelte` component wraps `@muhgholy/search-query-parser` to parse a single text input into typed filter terms rendered as chips. A `filter.ts` module applies parsed terms to resource items. Autocomplete suggestions are derived from the current ResourceStore items. Saved filters are loaded/saved via the existing `preferencesStore` and `ConfigService` RPCs.

**Tech Stack:** Svelte 5, @muhgholy/search-query-parser v3, Vitest, @testing-library/svelte

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `frontend/src/lib/search/parser.ts` | Create | Wraps @muhgholy/search-query-parser with Klados operator config |
| `frontend/src/lib/search/filter.ts` | Create | Applies TParsedTerm[] to items, returns filtered array |
| `frontend/src/lib/search/autocomplete.ts` | Create | Extracts context-aware suggestions from items based on cursor position |
| `frontend/src/lib/search/serialize.ts` | Create | Converts SavedFilter ↔ query string |
| `frontend/src/lib/components/SmartSearch.svelte` | Create | Unified chip-based search input with autocomplete |
| `frontend/src/lib/components/SmartSearchAutocomplete.svelte` | Create | Floating autocomplete popup |
| `frontend/src/lib/components/SavedFilterDropdown.svelte` | Create | Saved filter list + save popover |
| `frontend/src/lib/components/ResourceList.svelte` | Modify | Replace search input + AnnotationFilter with SmartSearch |
| `frontend/src/lib/components/AnnotationFilter.svelte` | Remove | Absorbed into SmartSearch |
| `frontend/src/lib/__tests__/parser.test.ts` | Create | Tests for parser wrapper |
| `frontend/src/lib/__tests__/filter.test.ts` | Create | Tests for filter logic |
| `frontend/src/lib/__tests__/autocomplete.test.ts` | Create | Tests for autocomplete suggestions |
| `frontend/src/lib/__tests__/serialize.test.ts` | Create | Tests for SavedFilter ↔ query string |
| `frontend/src/lib/__tests__/SmartSearch.svelte.test.ts` | Create | Tests for SmartSearch component |

---

### Task 1: Install @muhgholy/search-query-parser

**Files:**
- Modify: `frontend/package.json`

- [ ] **Step 1: Install the dependency**

Run:
```bash
cd frontend && pnpm add @muhgholy/search-query-parser
```

- [ ] **Step 2: Verify installation**

Run:
```bash
cd frontend && pnpm ls @muhgholy/search-query-parser
```

Expected: Shows `@muhgholy/search-query-parser 3.0.0`

- [ ] **Step 3: Verify types are available**

Run:
```bash
cd frontend && npx tsc --noEmit --moduleResolution bundler --module esnext -e "import { parse } from '@muhgholy/search-query-parser'"
```

If tsc doesn't support `-e`, just proceed — types will be verified in Task 2.

- [ ] **Step 4: Commit**

```bash
jj new && jj desc -m "feat(search): add @muhgholy/search-query-parser dependency"
```

---

### Task 2: Create parser.ts — Klados operator config wrapper

**Files:**
- Create: `frontend/src/lib/search/parser.ts`
- Create: `frontend/src/lib/__tests__/parser.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `frontend/src/lib/__tests__/parser.test.ts`:

```typescript
import { describe, it, expect } from 'vitest'
import { parseSearch, type SearchTerm } from '$lib/search/parser'

describe('parseSearch', () => {
  it('parses bare text as name filter', () => {
    const terms = parseSearch('nginx')
    expect(terms).toEqual([{ type: 'text', value: 'nginx', negated: false }])
  })

  it('parses label qualifier', () => {
    const terms = parseSearch('label:app=web')
    expect(terms).toEqual([{ type: 'label', value: 'app=web', negated: false }])
  })

  it('parses label alias l:', () => {
    const terms = parseSearch('l:app=web')
    expect(terms).toEqual([{ type: 'label', value: 'app=web', negated: false }])
  })

  it('parses annotation qualifier', () => {
    const terms = parseSearch('annotation:helm.sh/chart=myapp')
    expect(terms).toEqual([{ type: 'annotation', value: 'helm.sh/chart=myapp', negated: false }])
  })

  it('parses annotation alias ann:', () => {
    const terms = parseSearch('ann:owner=team-a')
    expect(terms).toEqual([{ type: 'annotation', value: 'owner=team-a', negated: false }])
  })

  it('parses name qualifier', () => {
    const terms = parseSearch('name:nginx')
    expect(terms).toEqual([{ type: 'name', value: 'nginx', negated: false }])
  })

  it('parses name alias n:', () => {
    const terms = parseSearch('n:nginx')
    expect(terms).toEqual([{ type: 'name', value: 'nginx', negated: false }])
  })

  it('parses namespace qualifier', () => {
    const terms = parseSearch('namespace:kube-system')
    expect(terms).toEqual([{ type: 'namespace', value: 'kube-system', negated: false }])
  })

  it('parses namespace alias ns:', () => {
    const terms = parseSearch('ns:default')
    expect(terms).toEqual([{ type: 'namespace', value: 'default', negated: false }])
  })

  it('parses negation', () => {
    const terms = parseSearch('-label:env=dev')
    expect(terms).toEqual([{ type: 'label', value: 'env=dev', negated: true }])
  })

  it('parses negated bare text', () => {
    const terms = parseSearch('-test')
    expect(terms).toEqual([{ type: 'text', value: 'test', negated: true }])
  })

  it('parses multiple terms', () => {
    const terms = parseSearch('l:app=web -ns:kube-system nginx')
    expect(terms).toHaveLength(3)
    expect(terms[0]).toEqual({ type: 'label', value: 'app=web', negated: false })
    expect(terms[1]).toEqual({ type: 'namespace', value: 'kube-system', negated: true })
    expect(terms[2]).toEqual({ type: 'text', value: 'nginx', negated: false })
  })

  it('parses quoted phrases', () => {
    const terms = parseSearch('"crash loop"')
    expect(terms).toEqual([{ type: 'phrase', value: 'crash loop', negated: false }])
  })

  it('returns empty array for empty string', () => {
    const terms = parseSearch('')
    expect(terms).toEqual([])
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
cd frontend && npx vitest run src/lib/__tests__/parser.test.ts
```

Expected: FAIL — module `$lib/search/parser` not found.

- [ ] **Step 3: Implement parser.ts**

Create `frontend/src/lib/search/parser.ts`:

```typescript
import { parse, type TParsedTerm } from '@muhgholy/search-query-parser'

export interface SearchTerm {
  type: string
  value: string
  negated: boolean
}

const KLADOS_OPTIONS = {
  operators: [
    { name: 'label', aliases: ['l'], type: 'string' as const, allowNegation: true },
    { name: 'annotation', aliases: ['ann'], type: 'string' as const, allowNegation: true },
    { name: 'name', aliases: ['n'], type: 'string' as const, allowNegation: true },
    { name: 'namespace', aliases: ['ns'], type: 'string' as const, allowNegation: true },
  ],
  operatorsAllowed: ['label', 'annotation', 'name', 'namespace'],
}

export function parseSearch(input: string): SearchTerm[] {
  if (!input.trim()) return []

  const parsed = parse(input, KLADOS_OPTIONS)

  return parsed.map((term: TParsedTerm) => ({
    type: term.type,
    value: term.value,
    negated: term.negated,
  }))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:
```bash
cd frontend && npx vitest run src/lib/__tests__/parser.test.ts
```

Expected: All tests PASS. If the library's output structure differs from expected (e.g. different field names or alias resolution behavior), adjust the mapping in `parseSearch` and update test expectations to match.

- [ ] **Step 5: Commit**

```bash
jj new && jj desc -m "feat(search): add parser wrapper with Klados operator config"
```

---

### Task 3: Create filter.ts — filter items by parsed terms

**Files:**
- Create: `frontend/src/lib/search/filter.ts`
- Create: `frontend/src/lib/__tests__/filter.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `frontend/src/lib/__tests__/filter.test.ts`:

```typescript
import { describe, it, expect } from 'vitest'
import { filterItems } from '$lib/search/filter'
import type { SearchTerm } from '$lib/search/parser'

function makeItem(name: string, namespace: string, labels: Record<string, string> = {}, annotations: Record<string, string> = {}) {
  return {
    metadata: { name, namespace, labels, annotations },
  }
}

const items = [
  makeItem('nginx-proxy', 'default', { app: 'web', env: 'prod' }, { owner: 'team-a' }),
  makeItem('nginx-ingress', 'kube-system', { app: 'web', env: 'dev' }, { owner: 'team-b' }),
  makeItem('redis-master', 'default', { app: 'cache', env: 'prod' }, {}),
  makeItem('test-pod', 'testing', { app: 'test' }, { 'helm.sh/chart': 'myapp' }),
]

describe('filterItems', () => {
  it('returns all items when no terms', () => {
    expect(filterItems(items, [])).toHaveLength(4)
  })

  it('filters by bare text on name', () => {
    const terms: SearchTerm[] = [{ type: 'text', value: 'nginx', negated: false }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(2)
    expect(result.map((r: any) => r.metadata.name)).toEqual(['nginx-proxy', 'nginx-ingress'])
  })

  it('filters by phrase on name', () => {
    const terms: SearchTerm[] = [{ type: 'phrase', value: 'redis-master', negated: false }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(1)
  })

  it('filters by name qualifier', () => {
    const terms: SearchTerm[] = [{ type: 'name', value: 'proxy', negated: false }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(1)
    expect(result[0].metadata.name).toBe('nginx-proxy')
  })

  it('filters by namespace qualifier', () => {
    const terms: SearchTerm[] = [{ type: 'namespace', value: 'default', negated: false }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(2)
  })

  it('filters by label key=value', () => {
    const terms: SearchTerm[] = [{ type: 'label', value: 'app=web', negated: false }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(2)
  })

  it('filters by label key exists', () => {
    const terms: SearchTerm[] = [{ type: 'label', value: 'env', negated: false }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(3)
  })

  it('filters by annotation key=value', () => {
    const terms: SearchTerm[] = [{ type: 'annotation', value: 'owner=team-a', negated: false }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(1)
    expect(result[0].metadata.name).toBe('nginx-proxy')
  })

  it('filters by annotation key exists', () => {
    const terms: SearchTerm[] = [{ type: 'annotation', value: 'owner', negated: false }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(2)
  })

  it('negates text filter', () => {
    const terms: SearchTerm[] = [{ type: 'text', value: 'nginx', negated: true }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(2)
    expect(result.map((r: any) => r.metadata.name)).toEqual(['redis-master', 'test-pod'])
  })

  it('negates label filter', () => {
    const terms: SearchTerm[] = [{ type: 'label', value: 'env=dev', negated: true }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(3)
  })

  it('negates namespace filter', () => {
    const terms: SearchTerm[] = [{ type: 'namespace', value: 'kube-system', negated: true }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(3)
  })

  it('ANDs multiple terms', () => {
    const terms: SearchTerm[] = [
      { type: 'label', value: 'app=web', negated: false },
      { type: 'namespace', value: 'default', negated: false },
    ]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(1)
    expect(result[0].metadata.name).toBe('nginx-proxy')
  })

  it('handles annotation with dots and slashes in key', () => {
    const terms: SearchTerm[] = [{ type: 'annotation', value: 'helm.sh/chart=myapp', negated: false }]
    const result = filterItems(items, terms)
    expect(result).toHaveLength(1)
    expect(result[0].metadata.name).toBe('test-pod')
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
cd frontend && npx vitest run src/lib/__tests__/filter.test.ts
```

Expected: FAIL — module `$lib/search/filter` not found.

- [ ] **Step 3: Implement filter.ts**

Create `frontend/src/lib/search/filter.ts`:

```typescript
import type { SearchTerm } from './parser'

function matchKeyValue(map: Record<string, string> | undefined, filter: string): boolean {
  if (!map) return false
  const eqIdx = filter.indexOf('=')
  if (eqIdx === -1) {
    return filter in map
  }
  const key = filter.substring(0, eqIdx)
  const val = filter.substring(eqIdx + 1)
  return map[key] === val
}

function matchesTerm(item: Record<string, any>, term: SearchTerm): boolean {
  const meta = item.metadata ?? {}
  const name: string = (meta.name ?? '').toLowerCase()

  let matches: boolean
  switch (term.type) {
    case 'text':
    case 'phrase':
      matches = name.includes(term.value.toLowerCase())
      break
    case 'name':
      matches = name.includes(term.value.toLowerCase())
      break
    case 'namespace':
      matches = (meta.namespace ?? '') === term.value
      break
    case 'label':
      matches = matchKeyValue(meta.labels, term.value)
      break
    case 'annotation':
      matches = matchKeyValue(meta.annotations, term.value)
      break
    default:
      matches = true
  }

  return term.negated ? !matches : matches
}

export function filterItems(items: Record<string, any>[], terms: SearchTerm[]): Record<string, any>[] {
  if (terms.length === 0) return items
  return items.filter((item) => terms.every((term) => matchesTerm(item, term)))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:
```bash
cd frontend && npx vitest run src/lib/__tests__/filter.test.ts
```

Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
jj new && jj desc -m "feat(search): add filter logic for parsed search terms"
```

---

### Task 4: Create autocomplete.ts — context-aware suggestions

**Files:**
- Create: `frontend/src/lib/search/autocomplete.ts`
- Create: `frontend/src/lib/__tests__/autocomplete.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `frontend/src/lib/__tests__/autocomplete.test.ts`:

```typescript
import { describe, it, expect } from 'vitest'
import { getSuggestions, type Suggestion } from '$lib/search/autocomplete'

function makeItem(name: string, namespace: string, labels: Record<string, string> = {}, annotations: Record<string, string> = {}) {
  return {
    metadata: { name, namespace, labels, annotations },
  }
}

const items = [
  makeItem('nginx-proxy', 'default', { app: 'web', env: 'prod' }, { owner: 'team-a' }),
  makeItem('nginx-ingress', 'kube-system', { app: 'web', env: 'dev' }, { owner: 'team-b' }),
  makeItem('redis-master', 'default', { app: 'cache', env: 'prod' }, {}),
]

describe('getSuggestions', () => {
  it('suggests qualifiers when input is empty', () => {
    const result = getSuggestions('', 0, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['label:', 'annotation:', 'name:', 'namespace:'])
    )
  })

  it('suggests qualifiers matching partial text', () => {
    const result = getSuggestions('lab', 3, items)
    expect(result).toHaveLength(1)
    expect(result[0].value).toBe('label:')
  })

  it('suggests label keys after label:', () => {
    const result = getSuggestions('label:', 6, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['app', 'env'])
    )
  })

  it('suggests label keys after alias l:', () => {
    const result = getSuggestions('l:', 2, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['app', 'env'])
    )
  })

  it('filters label key suggestions by partial input', () => {
    const result = getSuggestions('label:ap', 8, items)
    expect(result).toHaveLength(1)
    expect(result[0].value).toBe('app')
  })

  it('suggests label values after key=', () => {
    const result = getSuggestions('label:app=', 10, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['web', 'cache'])
    )
  })

  it('filters label value suggestions by partial input', () => {
    const result = getSuggestions('label:app=w', 11, items)
    expect(result).toHaveLength(1)
    expect(result[0].value).toBe('web')
  })

  it('suggests annotation keys after annotation:', () => {
    const result = getSuggestions('annotation:', 11, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(['owner'])
  })

  it('suggests annotation keys after ann:', () => {
    const result = getSuggestions('ann:', 4, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(['owner'])
  })

  it('suggests namespace values after namespace:', () => {
    const result = getSuggestions('namespace:', 10, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['default', 'kube-system'])
    )
  })

  it('suggests namespace values after ns:', () => {
    const result = getSuggestions('ns:', 3, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['default', 'kube-system'])
    )
  })

  it('includes count in suggestions', () => {
    const result = getSuggestions('label:', 6, items)
    const appSuggestion = result.find((s: Suggestion) => s.value === 'app')
    expect(appSuggestion?.count).toBe(3)
  })

  it('returns no suggestions for bare text mid-word', () => {
    const result = getSuggestions('ngi', 3, items)
    expect(result).toHaveLength(0)
  })

  it('handles cursor in the middle of multi-term input', () => {
    const result = getSuggestions('label:app=web ns:', 17, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['default', 'kube-system'])
    )
  })

  it('suggests qualifiers after negation prefix', () => {
    const result = getSuggestions('-lab', 4, items)
    expect(result).toHaveLength(1)
    expect(result[0].value).toBe('label:')
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
cd frontend && npx vitest run src/lib/__tests__/autocomplete.test.ts
```

Expected: FAIL — module `$lib/search/autocomplete` not found.

- [ ] **Step 3: Implement autocomplete.ts**

Create `frontend/src/lib/search/autocomplete.ts`:

```typescript
export interface Suggestion {
  value: string
  count?: number
  description?: string
}

const QUALIFIERS = [
  { value: 'label:', aliases: ['l:'], description: 'Filter by label' },
  { value: 'annotation:', aliases: ['ann:'], description: 'Filter by annotation' },
  { value: 'name:', aliases: ['n:'], description: 'Filter by name' },
  { value: 'namespace:', aliases: ['ns:'], description: 'Filter by namespace' },
]

const QUALIFIER_ALIASES: Record<string, string> = {
  'l:': 'label:',
  'ann:': 'annotation:',
  'n:': 'name:',
  'ns:': 'namespace:',
}

function extractCurrentToken(input: string, cursor: number): string {
  const before = input.substring(0, cursor)
  const lastSpace = before.lastIndexOf(' ')
  return before.substring(lastSpace + 1)
}

function collectDistinct(items: Record<string, any>[], extractor: (item: Record<string, any>) => Record<string, string> | undefined): Map<string, number> {
  const counts = new Map<string, number>()
  for (const item of items) {
    const map = extractor(item)
    if (map) {
      for (const key of Object.keys(map)) {
        counts.set(key, (counts.get(key) ?? 0) + 1)
      }
    }
  }
  return counts
}

function collectValues(items: Record<string, any>[], extractor: (item: Record<string, any>) => Record<string, string> | undefined, key: string): Map<string, number> {
  const counts = new Map<string, number>()
  for (const item of items) {
    const map = extractor(item)
    if (map && key in map) {
      const val = map[key]
      counts.set(val, (counts.get(val) ?? 0) + 1)
    }
  }
  return counts
}

function collectNamespaces(items: Record<string, any>[]): Map<string, number> {
  const counts = new Map<string, number>()
  for (const item of items) {
    const ns = item.metadata?.namespace ?? ''
    if (ns) counts.set(ns, (counts.get(ns) ?? 0) + 1)
  }
  return counts
}

function mapToSuggestions(counts: Map<string, number>, prefix: string): Suggestion[] {
  return Array.from(counts.entries())
    .filter(([key]) => !prefix || key.toLowerCase().startsWith(prefix.toLowerCase()))
    .map(([value, count]) => ({ value, count }))
    .sort((a, b) => (b.count ?? 0) - (a.count ?? 0))
}

export function getSuggestions(input: string, cursor: number, items: Record<string, any>[]): Suggestion[] {
  const token = extractCurrentToken(input, cursor)

  // Strip leading - for negation
  const stripped = token.startsWith('-') ? token.substring(1) : token

  // Check if token contains a qualifier
  const colonIdx = stripped.indexOf(':')

  if (colonIdx === -1) {
    // No colon — suggest qualifiers if prefix matches
    if (stripped === '') {
      return QUALIFIERS.map((q) => ({ value: q.value, description: q.description }))
    }
    // Check if it looks like start of a qualifier
    const lower = stripped.toLowerCase()
    const matches = QUALIFIERS.filter(
      (q) => q.value.startsWith(lower) || q.aliases.some((a) => a.startsWith(lower))
    )
    if (matches.length > 0) {
      return matches.map((q) => ({ value: q.value, description: q.description }))
    }
    // Bare text mid-word — no suggestions
    return []
  }

  // Has a colon — resolve the qualifier
  let qualifier = stripped.substring(0, colonIdx + 1)
  qualifier = QUALIFIER_ALIASES[qualifier] ?? qualifier
  const afterColon = stripped.substring(colonIdx + 1)

  // Check for key=value pattern
  const eqIdx = afterColon.indexOf('=')

  if (qualifier === 'label:') {
    const extractor = (item: Record<string, any>) => item.metadata?.labels
    if (eqIdx === -1) {
      return mapToSuggestions(collectDistinct(items, extractor), afterColon)
    }
    const key = afterColon.substring(0, eqIdx)
    const valPrefix = afterColon.substring(eqIdx + 1)
    return mapToSuggestions(collectValues(items, extractor, key), valPrefix)
  }

  if (qualifier === 'annotation:') {
    const extractor = (item: Record<string, any>) => item.metadata?.annotations
    if (eqIdx === -1) {
      return mapToSuggestions(collectDistinct(items, extractor), afterColon)
    }
    const key = afterColon.substring(0, eqIdx)
    const valPrefix = afterColon.substring(eqIdx + 1)
    return mapToSuggestions(collectValues(items, extractor, key), valPrefix)
  }

  if (qualifier === 'namespace:') {
    return mapToSuggestions(collectNamespaces(items), afterColon)
  }

  // name: — no autocomplete for names (free text)
  return []
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:
```bash
cd frontend && npx vitest run src/lib/__tests__/autocomplete.test.ts
```

Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
jj new && jj desc -m "feat(search): add context-aware autocomplete suggestions"
```

---

### Task 5: Create serialize.ts — SavedFilter ↔ query string

**Files:**
- Create: `frontend/src/lib/search/serialize.ts`
- Create: `frontend/src/lib/__tests__/serialize.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `frontend/src/lib/__tests__/serialize.test.ts`:

```typescript
import { describe, it, expect } from 'vitest'
import { savedFilterToQuery, queryToSavedFilter } from '$lib/search/serialize'
import type { SavedFilter } from '$lib/stores/preferences.svelte'

describe('savedFilterToQuery', () => {
  it('converts labels to label: terms', () => {
    const filter: SavedFilter = { name: 'test', labels: { app: 'web', env: 'prod' } }
    const query = savedFilterToQuery(filter)
    expect(query).toContain('l:app=web')
    expect(query).toContain('l:env=prod')
  })

  it('converts annotations to ann: terms', () => {
    const filter: SavedFilter = { name: 'test', annotations: { owner: 'team-a' } }
    const query = savedFilterToQuery(filter)
    expect(query).toContain('ann:owner=team-a')
  })

  it('appends search text as-is', () => {
    const filter: SavedFilter = { name: 'test', search: 'nginx' }
    const query = savedFilterToQuery(filter)
    expect(query).toContain('nginx')
  })

  it('combines all fields', () => {
    const filter: SavedFilter = {
      name: 'test',
      labels: { app: 'web' },
      annotations: { owner: 'team-a' },
      search: 'nginx',
    }
    const query = savedFilterToQuery(filter)
    expect(query).toContain('l:app=web')
    expect(query).toContain('ann:owner=team-a')
    expect(query).toContain('nginx')
  })

  it('returns empty string for empty filter', () => {
    const filter: SavedFilter = { name: 'test' }
    expect(savedFilterToQuery(filter)).toBe('')
  })
})

describe('queryToSavedFilter', () => {
  it('extracts labels from label: terms', () => {
    const filter = queryToSavedFilter('l:app=web l:env=prod')
    expect(filter.labels).toEqual({ app: 'web', env: 'prod' })
  })

  it('extracts annotations from ann: terms', () => {
    const filter = queryToSavedFilter('ann:owner=team-a')
    expect(filter.annotations).toEqual({ owner: 'team-a' })
  })

  it('puts text and non-model terms into search', () => {
    const filter = queryToSavedFilter('l:app=web nginx name:proxy')
    expect(filter.labels).toEqual({ app: 'web' })
    expect(filter.search).toBe('nginx name:proxy')
  })

  it('returns empty fields for empty input', () => {
    const filter = queryToSavedFilter('')
    expect(filter.labels).toBeUndefined()
    expect(filter.annotations).toBeUndefined()
    expect(filter.search).toBeUndefined()
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
cd frontend && npx vitest run src/lib/__tests__/serialize.test.ts
```

Expected: FAIL — module `$lib/search/serialize` not found.

- [ ] **Step 3: Implement serialize.ts**

Create `frontend/src/lib/search/serialize.ts`:

```typescript
import type { SavedFilter } from '$lib/stores/preferences.svelte'
import { parseSearch } from './parser'

export function savedFilterToQuery(filter: SavedFilter): string {
  const parts: string[] = []

  if (filter.labels) {
    for (const [key, value] of Object.entries(filter.labels)) {
      parts.push(`l:${key}=${value}`)
    }
  }

  if (filter.annotations) {
    for (const [key, value] of Object.entries(filter.annotations)) {
      parts.push(`ann:${key}=${value}`)
    }
  }

  if (filter.search) {
    parts.push(filter.search)
  }

  return parts.join(' ')
}

export function queryToSavedFilter(query: string): Omit<SavedFilter, 'name'> {
  if (!query.trim()) return {}

  const terms = parseSearch(query)
  const labels: Record<string, string> = {}
  const annotations: Record<string, string> = {}
  const searchParts: string[] = []

  for (const term of terms) {
    if (term.type === 'label' && term.value.includes('=') && !term.negated) {
      const [key, ...rest] = term.value.split('=')
      labels[key] = rest.join('=')
    } else if (term.type === 'annotation' && term.value.includes('=') && !term.negated) {
      const [key, ...rest] = term.value.split('=')
      annotations[key] = rest.join('=')
    } else if (term.type === 'text' || term.type === 'phrase') {
      searchParts.push(term.negated ? `-${term.value}` : term.value)
    } else {
      // name:, namespace:, negated labels/annotations — preserve as search text
      const prefix = term.negated ? '-' : ''
      if (term.type === 'text' || term.type === 'phrase') {
        searchParts.push(`${prefix}${term.value}`)
      } else {
        searchParts.push(`${prefix}${term.type}:${term.value}`)
      }
    }
  }

  const result: Omit<SavedFilter, 'name'> = {}
  if (Object.keys(labels).length > 0) result.labels = labels
  if (Object.keys(annotations).length > 0) result.annotations = annotations
  if (searchParts.length > 0) result.search = searchParts.join(' ')
  return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:
```bash
cd frontend && npx vitest run src/lib/__tests__/serialize.test.ts
```

Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
jj new && jj desc -m "feat(search): add SavedFilter to query string serialization"
```

---

### Task 6: Create SmartSearchAutocomplete.svelte

**Files:**
- Create: `frontend/src/lib/components/SmartSearchAutocomplete.svelte`

- [ ] **Step 1: Create the autocomplete popup component**

Create `frontend/src/lib/components/SmartSearchAutocomplete.svelte`:

```svelte
<script lang="ts">
  import type { Suggestion } from '$lib/search/autocomplete'

  let {
    suggestions = [],
    visible = false,
    selectedIndex = 0,
    onselect,
  }: {
    suggestions: Suggestion[]
    visible: boolean
    selectedIndex: number
    onselect?: (suggestion: Suggestion) => void
  } = $props()
</script>

{#if visible && suggestions.length > 0}
  <div
    class="absolute left-0 top-full mt-1 z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-48 max-h-64 overflow-y-auto"
    role="listbox"
  >
    {#each suggestions as suggestion, i}
      <button
        class="w-full flex items-center justify-between gap-4 px-3 py-1.5 text-sm text-left hover:bg-surface-hover {i === selectedIndex ? 'bg-surface-hover' : ''}"
        role="option"
        aria-selected={i === selectedIndex}
        onmousedown|preventDefault={() => onselect?.(suggestion)}
      >
        <span class="text-fg">{suggestion.value}</span>
        <span class="flex items-center gap-2">
          {#if suggestion.description}
            <span class="text-muted text-xs">{suggestion.description}</span>
          {/if}
          {#if suggestion.count !== undefined}
            <span class="text-muted text-xs tabular-nums">{suggestion.count}</span>
          {/if}
        </span>
      </button>
    {/each}
  </div>
{/if}
```

- [ ] **Step 2: Verify no type errors**

Run:
```bash
cd frontend && pnpm check
```

Expected: No errors related to SmartSearchAutocomplete.

- [ ] **Step 3: Commit**

```bash
jj new && jj desc -m "feat(search): add autocomplete popup component"
```

---

### Task 7: Create SmartSearch.svelte — the main search bar

**Files:**
- Create: `frontend/src/lib/components/SmartSearch.svelte`

This is the largest component. It manages the text input, chip rendering, autocomplete integration, and parsed term output.

- [ ] **Step 1: Create the component**

Create `frontend/src/lib/components/SmartSearch.svelte`:

```svelte
<script lang="ts">
  import { parseSearch, type SearchTerm } from '$lib/search/parser'
  import { getSuggestions, type Suggestion } from '$lib/search/autocomplete'
  import SmartSearchAutocomplete from './SmartSearchAutocomplete.svelte'

  let {
    items = [],
    value = $bindable(''),
    ontermschange,
  }: {
    items: Record<string, any>[]
    value?: string
    ontermschange?: (terms: SearchTerm[]) => void
  } = $props()
  let inputEl: HTMLInputElement | undefined = $state()
  let suggestions = $state<Suggestion[]>([])
  let selectedIndex = $state(0)
  let showAutocomplete = $state(false)

  let terms = $derived.by(() => {
    return parseSearch(value)
  })

  // Chips are completed terms (all except the trailing incomplete token)
  let chips = $derived.by(() => {
    const raw = value
    if (!raw.trim()) return []
    // If input ends with a space, all tokens are complete
    if (raw.endsWith(' ')) return terms
    // Otherwise, last token is still being typed — exclude it from chips
    return terms.slice(0, -1)
  })

  let trailingText = $derived.by(() => {
    const raw = value
    if (!raw.trim()) return ''
    const lastSpace = raw.lastIndexOf(' ')
    if (raw.endsWith(' ')) return ''
    return raw.substring(lastSpace + 1)
  })

  $effect(() => {
    ontermschange?.(terms)
  })

  function updateAutocomplete() {
    if (!inputEl) return
    const cursor = inputEl.selectionStart ?? value.length
    suggestions = getSuggestions(value, cursor, items)
    selectedIndex = 0
    showAutocomplete = suggestions.length > 0
  }

  function handleInput() {
    updateAutocomplete()
  }

  function handleFocus() {
    updateAutocomplete()
  }

  function handleBlur() {
    showAutocomplete = false
  }

  function applySuggestion(suggestion: Suggestion) {
    const cursor = inputEl?.selectionStart ?? value.length
    const before = value.substring(0, cursor)
    const after = value.substring(cursor)
    const lastSpace = before.lastIndexOf(' ')
    const tokenStart = before.substring(lastSpace + 1)

    // Strip negation prefix to find the qualifier part
    const stripped = tokenStart.startsWith('-') ? tokenStart.substring(1) : tokenStart
    const negPrefix = tokenStart.startsWith('-') ? '-' : ''
    const colonIdx = stripped.indexOf(':')

    let replacement: string

    if (colonIdx === -1) {
      // Suggesting a qualifier — replace the partial text with the qualifier
      replacement = negPrefix + suggestion.value
    } else {
      // Suggesting a key or value after qualifier
      const qualifier = stripped.substring(0, colonIdx + 1)
      const afterColon = stripped.substring(colonIdx + 1)
      const eqIdx = afterColon.indexOf('=')

      if (eqIdx === -1) {
        // Suggesting a key — append = to invite value completion
        replacement = negPrefix + qualifier + suggestion.value + '='
      } else {
        // Suggesting a value — complete the value and add space
        const key = afterColon.substring(0, eqIdx)
        replacement = negPrefix + qualifier + key + '=' + suggestion.value + ' '
      }
    }

    value = before.substring(0, lastSpace + 1) + replacement + after
    showAutocomplete = false

    // Re-focus and move cursor to end of replacement
    requestAnimationFrame(() => {
      inputEl?.focus()
      updateAutocomplete()
    })
  }

  function handleKeydown(e: KeyboardEvent) {
    if (showAutocomplete && suggestions.length > 0) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        selectedIndex = (selectedIndex + 1) % suggestions.length
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        selectedIndex = (selectedIndex - 1 + suggestions.length) % suggestions.length
        return
      }
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault()
        applySuggestion(suggestions[selectedIndex])
        return
      }
      if (e.key === 'Escape') {
        e.preventDefault()
        showAutocomplete = false
        return
      }
    }
  }

  function removeChip(index: number) {
    const allTerms = [...terms]
    allTerms.splice(index, 1)
    // Rebuild input from remaining terms + trailing text
    const parts = allTerms.map((t) => {
      const neg = t.negated ? '-' : ''
      if (t.type === 'text' || t.type === 'phrase') {
        return t.type === 'phrase' ? `${neg}"${t.value}"` : `${neg}${t.value}`
      }
      return `${neg}${t.type}:${t.value}`
    })
    value = parts.join(' ') + (parts.length > 0 ? ' ' : '')
    requestAnimationFrame(() => inputEl?.focus())
  }

  function chipColor(type: string): string {
    switch (type) {
      case 'label': return 'bg-blue-500/15 text-blue-400 border-blue-500/30'
      case 'annotation': return 'bg-purple-500/15 text-purple-400 border-purple-500/30'
      case 'namespace': return 'bg-green-500/15 text-green-400 border-green-500/30'
      case 'name': return 'bg-orange-500/15 text-orange-400 border-orange-500/30'
      default: return 'bg-muted/15 text-fg border-border'
    }
  }

  function chipLabel(term: SearchTerm): string {
    const neg = term.negated ? '-' : ''
    if (term.type === 'text' || term.type === 'phrase') {
      return `${neg}${term.value}`
    }
    const short: Record<string, string> = { label: 'l', annotation: 'ann', namespace: 'ns', name: 'n' }
    return `${neg}${short[term.type] ?? term.type}:${term.value}`
  }
</script>

<div class="relative flex items-center gap-1 flex-1 min-w-0">
  <div class="flex flex-wrap items-center gap-1 flex-1 min-w-0 px-2 py-1 bg-surface border border-border rounded text-sm focus-within:ring-1 focus-within:ring-accent">
    {#each chips as chip, i}
      <span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded border text-xs font-mono {chipColor(chip.type)} {chip.negated ? 'line-through opacity-75' : ''}">
        {chipLabel(chip)}
        <button
          class="ml-0.5 hover:text-fg"
          onclick={() => removeChip(i)}
          tabindex={-1}
        >&times;</button>
      </span>
    {/each}
    <input
      bind:this={inputEl}
      bind:value={value}
      oninput={handleInput}
      onfocus={handleFocus}
      onblur={handleBlur}
      onkeydown={handleKeydown}
      class="flex-1 min-w-24 bg-transparent outline-none text-fg placeholder:text-muted"
      placeholder={chips.length === 0 ? 'Filter resources... (label:key=value, name:..., ns:...)' : ''}
    />
  </div>

  <SmartSearchAutocomplete
    {suggestions}
    visible={showAutocomplete}
    {selectedIndex}
    onselect={applySuggestion}
  />
</div>
```

- [ ] **Step 2: Verify no type errors**

Run:
```bash
cd frontend && pnpm check
```

Expected: No errors related to SmartSearch.

- [ ] **Step 3: Commit**

```bash
jj new && jj desc -m "feat(search): add SmartSearch component with chips and autocomplete"
```

---

### Task 8: Create SavedFilterDropdown.svelte

**Files:**
- Create: `frontend/src/lib/components/SavedFilterDropdown.svelte`

- [ ] **Step 1: Create the component**

Create `frontend/src/lib/components/SavedFilterDropdown.svelte`:

```svelte
<script lang="ts">
  import { preferencesStore, type SavedFilter } from '$lib/stores/preferences.svelte'
  import { savedFilterToQuery, queryToSavedFilter } from '$lib/search/serialize'
  import { Bookmark } from 'lucide-svelte'
  import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'

  let {
    gvr,
    contextName,
    currentQuery = '',
    onapply,
  }: {
    gvr: string
    contextName: string
    currentQuery: string
    onapply?: (query: string) => void
  } = $props()

  let dropdownOpen = $state(false)
  let showSaveForm = $state(false)
  let saveName = $state('')
  let saveScope = $state<'cluster' | 'global'>('cluster')

  let savedFilters = $derived(preferencesStore.getSavedFilters(gvr))

  function toggleDropdown() {
    dropdownOpen = !dropdownOpen
    if (!dropdownOpen) showSaveForm = false
  }

  function applyFilter(filter: SavedFilter) {
    onapply?.(savedFilterToQuery(filter))
    dropdownOpen = false
  }

  function openSaveForm() {
    showSaveForm = true
    saveName = ''
    saveScope = 'cluster'
  }

  async function saveFilter() {
    if (!saveName.trim()) return

    const filterData = queryToSavedFilter(currentQuery)
    const newFilter: SavedFilter = { name: saveName.trim(), ...filterData }

    const existing = [...savedFilters]
    const dupeIdx = existing.findIndex((f) => f.name === newFilter.name)
    if (dupeIdx >= 0) {
      existing[dupeIdx] = newFilter
    } else {
      existing.push(newFilter)
    }

    if (saveScope === 'cluster') {
      await ConfigService.SetClusterSavedFilters(contextName, gvr, existing)
    } else {
      await ConfigService.SetSavedFilters(gvr, existing)
    }

    showSaveForm = false
    dropdownOpen = false
  }

  function filterPreview(filter: SavedFilter): string {
    return savedFilterToQuery(filter) || '(empty)'
  }

  function handleClickOutside(e: MouseEvent) {
    const target = e.target as HTMLElement
    if (!target.closest('.saved-filter-dropdown')) {
      dropdownOpen = false
    }
  }
</script>

<svelte:window onclick={handleClickOutside} />

<div class="relative saved-filter-dropdown">
  <button
    class="p-1.5 rounded text-muted hover:text-fg hover:bg-surface-hover"
    title="Saved filters"
    onclick={toggleDropdown}
  >
    <Bookmark size={16} />
  </button>

  {#if dropdownOpen}
    <div class="absolute right-0 top-full mt-1 z-50 bg-surface border border-border rounded shadow-lg py-1 min-w-64">
      {#if !showSaveForm}
        <div class="px-3 py-1.5 text-xs font-medium text-muted uppercase tracking-wide">Saved Filters</div>

        {#if savedFilters.length === 0}
          <div class="px-3 py-2 text-sm text-muted">No saved filters for this resource</div>
        {/if}

        {#each savedFilters as filter}
          <button
            class="w-full text-left px-3 py-1.5 hover:bg-surface-hover"
            onclick={() => applyFilter(filter)}
          >
            <div class="text-sm text-fg font-medium">{filter.name}</div>
            <div class="text-xs text-muted font-mono truncate">{filterPreview(filter)}</div>
          </button>
        {/each}

        <div class="border-t border-border mt-1 pt-1">
          <button
            class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover disabled:opacity-50 disabled:cursor-not-allowed text-accent"
            disabled={!currentQuery.trim()}
            onclick={openSaveForm}
          >
            + Save current filter
          </button>
        </div>
      {:else}
        <div class="px-3 py-2">
          <div class="text-xs font-medium text-muted uppercase tracking-wide mb-2">Save Filter</div>

          <input
            bind:value={saveName}
            class="w-full px-2 py-1 text-sm bg-surface border border-border rounded text-fg placeholder:text-muted mb-2"
            placeholder="Filter name"
            onkeydown={(e) => e.key === 'Enter' && saveFilter()}
          />

          <div class="flex gap-3 mb-3 text-sm">
            <label class="flex items-center gap-1.5 text-fg">
              <input type="radio" bind:group={saveScope} value="cluster" class="accent-accent" />
              This cluster
            </label>
            <label class="flex items-center gap-1.5 text-fg">
              <input type="radio" bind:group={saveScope} value="global" class="accent-accent" />
              Global
            </label>
          </div>

          <div class="flex justify-end gap-2">
            <button
              class="px-2 py-1 text-sm rounded text-muted hover:text-fg"
              onclick={() => (showSaveForm = false)}
            >Cancel</button>
            <button
              class="px-2 py-1 text-sm rounded bg-accent text-white hover:opacity-90 disabled:opacity-50"
              disabled={!saveName.trim()}
              onclick={saveFilter}
            >Save</button>
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>
```

- [ ] **Step 2: Verify the ConfigService bindings exist**

Run:
```bash
cd frontend && grep -l 'SetSavedFilters\|SetClusterSavedFilters' ../bindings/github.com/Vilsol/klados/internal/services/configservice.js 2>/dev/null || echo "Check binding path"
```

If the path is wrong, find the correct one with:
```bash
grep -rl 'SetSavedFilters' frontend/bindings/
```

Update the import in SavedFilterDropdown.svelte to match the actual path.

- [ ] **Step 3: Verify no type errors**

Run:
```bash
cd frontend && pnpm check
```

Expected: No errors related to SavedFilterDropdown.

- [ ] **Step 4: Commit**

```bash
jj new && jj desc -m "feat(search): add SavedFilterDropdown component"
```

---

### Task 9: Wire SmartSearch into ResourceList.svelte

**Files:**
- Modify: `frontend/src/lib/components/ResourceList.svelte:35-77` (props and state)
- Modify: `frontend/src/lib/components/ResourceList.svelte:124-159` (filter logic)
- Modify: `frontend/src/lib/components/ResourceList.svelte:292-300` (toolbar)

This task replaces the existing search input + AnnotationFilter with SmartSearch and rewires the filter logic.

- [ ] **Step 1: Read the current ResourceList.svelte**

Read the full file to understand current structure before making changes.

- [ ] **Step 2: Add SmartSearch imports and remove old imports**

At the top of ResourceList.svelte, add:

```typescript
import SmartSearch from './SmartSearch.svelte'
import SavedFilterDropdown from './SavedFilterDropdown.svelte'
import { filterItems } from '$lib/search/filter'
import type { SearchTerm } from '$lib/search/parser'
```

Remove the AnnotationFilter import:

```typescript
// REMOVE: import AnnotationFilter from './AnnotationFilter.svelte'
```

- [ ] **Step 3: Replace filter state variables**

Remove the old state variables (around lines 71, 77):

```typescript
// REMOVE: let filterText = $state('')
// REMOVE: let annotationFilters: { key: string; value: string }[] = $state([])
```

Add the new state:

```typescript
let searchTerms = $state<SearchTerm[]>([])
let searchQuery = $state('')
```

- [ ] **Step 4: Replace the filter $derived block**

Replace the existing `filtered` derived block (lines ~124-159) with one that uses `filterItems` for the search terms. Keep namespace filtering and sorting intact:

```typescript
let filtered = $derived.by(() => {
  let result = items

  // Namespace filter (when multiple namespaces are selected)
  if (selectedNamespaces.length > 1) {
    const nsSet = new Set(selectedNamespaces)
    result = result.filter((item: Record<string, any>) => nsSet.has(item.metadata?.namespace))
  }

  // Smart search filter
  result = filterItems(result, searchTerms)

  // Sorting (keep existing sort logic)
  // ... existing sort code ...

  return result
})
```

- [ ] **Step 5: Replace the toolbar UI**

Replace the search input and AnnotationFilter in the toolbar (around lines 292-300) with:

```svelte
<SmartSearch
  {items}
  ontermschange={(t) => { searchTerms = t }}
  bind:value={searchQuery}
/>
<SavedFilterDropdown
  {gvr}
  {contextName}
  currentQuery={searchQuery}
  onapply={(q) => { searchQuery = q }}
/>
```

Note: SmartSearch already exposes `value` as a bindable prop (defined in Task 7), so SavedFilterDropdown can set it via the `onapply` callback.

- [ ] **Step 6: Verify no type errors**

Run:
```bash
cd frontend && pnpm check
```

Expected: No errors. If there are errors about missing props (e.g. `contextName` or `gvr` not available in ResourceList), check that ResourceList already receives these as props (it does — `contextName` and `gvr` are existing props).

- [ ] **Step 7: Commit**

```bash
jj new && jj desc -m "feat(search): wire SmartSearch into ResourceList, remove AnnotationFilter"
```

---

### Task 10: Remove AnnotationFilter.svelte

**Files:**
- Remove: `frontend/src/lib/components/AnnotationFilter.svelte`

- [ ] **Step 1: Verify no remaining imports of AnnotationFilter**

Run:
```bash
cd frontend && grep -r 'AnnotationFilter' src/
```

Expected: No results (it was already removed from ResourceList in Task 9).

- [ ] **Step 2: Delete the file**

```bash
rm frontend/src/lib/components/AnnotationFilter.svelte
```

- [ ] **Step 3: Run type check**

Run:
```bash
cd frontend && pnpm check
```

Expected: No errors.

- [ ] **Step 4: Commit**

```bash
jj new && jj desc -m "refactor(search): remove AnnotationFilter component"
```

---

### Task 11: Write SmartSearch component tests

**Files:**
- Create: `frontend/src/lib/__tests__/SmartSearch.svelte.test.ts`

- [ ] **Step 1: Write the test file**

Create `frontend/src/lib/__tests__/SmartSearch.svelte.test.ts`:

```typescript
import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent } from '@testing-library/svelte'
import SmartSearch from '$lib/components/SmartSearch.svelte'

const items = [
  { metadata: { name: 'nginx-proxy', namespace: 'default', labels: { app: 'web' }, annotations: {} } },
  { metadata: { name: 'redis-master', namespace: 'kube-system', labels: { app: 'cache' }, annotations: {} } },
]

describe('SmartSearch', () => {
  it('renders the search input', () => {
    const { container } = render(SmartSearch, { props: { items } })
    const input = container.querySelector('input')
    expect(input).toBeTruthy()
  })

  it('calls ontermschange when input changes', async () => {
    const ontermschange = vi.fn()
    const { container } = render(SmartSearch, { props: { items, ontermschange } })
    const input = container.querySelector('input')!

    await fireEvent.input(input, { target: { value: 'nginx' } })

    expect(ontermschange).toHaveBeenCalled()
    const lastCall = ontermschange.mock.calls[ontermschange.mock.calls.length - 1]
    expect(lastCall[0]).toEqual([{ type: 'text', value: 'nginx', negated: false }])
  })

  it('shows autocomplete when typing a qualifier prefix', async () => {
    const { container } = render(SmartSearch, { props: { items } })
    const input = container.querySelector('input')!

    await fireEvent.focus(input)
    await fireEvent.input(input, { target: { value: 'lab' } })

    const popup = container.querySelector('[role="listbox"]')
    expect(popup).toBeTruthy()
  })

  it('shows placeholder when input is empty', () => {
    const { container } = render(SmartSearch, { props: { items } })
    const input = container.querySelector('input')!
    expect(input.getAttribute('placeholder')).toContain('Filter resources')
  })
})
```

- [ ] **Step 2: Run the tests**

Run:
```bash
cd frontend && npx vitest run src/lib/__tests__/SmartSearch.svelte.test.ts
```

Expected: All tests PASS. If there are mock issues (e.g. missing Wails runtime mocks), the existing `setup.ts` should cover it since SmartSearch doesn't directly import Wails bindings.

- [ ] **Step 3: Commit**

```bash
jj new && jj desc -m "test(search): add SmartSearch component tests"
```

---

### Task 12: End-to-end verification

**Files:** None (verification only)

- [ ] **Step 1: Run all frontend tests**

Run:
```bash
cd frontend && pnpm test
```

Expected: All tests pass, including existing ResourceList tests. If ResourceList tests fail due to the removed AnnotationFilter, update them to work with SmartSearch instead.

- [ ] **Step 2: Run type check**

Run:
```bash
cd frontend && pnpm check
```

Expected: No type errors.

- [ ] **Step 3: Run dev mode and verify visually**

Run:
```bash
task dev
```

Verify:
- The smart search bar appears in the resource list toolbar
- Typing `label:` shows autocomplete with label keys from current items
- Selecting a suggestion completes the token
- Completed tokens render as colored chips
- The `☆` button opens the saved filters dropdown
- Saving a filter works (name + scope selection)
- Applying a saved filter populates the search bar
- Negation works (`-label:env=dev` excludes matching items)
- Backspace removes chips

- [ ] **Step 4: Final commit if any fixes needed**

```bash
jj new && jj desc -m "fix(search): address integration issues from end-to-end testing"
```
