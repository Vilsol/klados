# YAML Schema Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone `codemirror-yaml-completion` package that provides schema-driven property completions on empty lines and fuzzy matching on typed input, supplementing `codemirror-json-schema`'s broken empty-line behavior.

**Architecture:** A `CompletionSource` registered on `yamlLanguage` that parses YAML via the `yaml` library's AST, maps cursor position to a JSON pointer, resolves the schema at that pointer via `json-schema-library`'s `Draft07`, and returns fuzzy-matched property completions. Coexists with the upstream `yamlCompletion` — CodeMirror merges and deduplicates results.

**Tech Stack:** TypeScript, `@codemirror/autocomplete`, `@codemirror/state`, `@codemirror/lang-yaml`, `json-schema-library` (Draft07), `yaml` (parseDocument AST), `vitest`

---

## File Structure

```
packages/codemirror-yaml-completion/
  package.json          — Package manifest with peer deps
  tsconfig.json         — TypeScript config (ESNext, bundler resolution)
  vitest.config.ts      — Test config (node environment)
  src/
    index.ts            — Public API: exports yamlSchemaCompletion()
    completion.ts       — CompletionSource: activation logic, orchestrates position/schema/matching
    position.ts         — AST position mapping: parseDocument, find enclosing map, derive pointer
    schema-resolve.ts   — Schema resolution: Draft07 getSchema, collect properties from subschema
    fuzzy.ts            — Fuzzy matching: prefix/substring/subsequence with boost scoring
    __tests__/
      completion.test.ts — Integration tests: full completion source with EditorState + schema
      position.test.ts   — Unit tests: AST position mapping, pointer derivation, edge cases
      fuzzy.test.ts      — Unit tests: fuzzy matching logic
```

Consumer modifications:
```
packages/ui/src/lib/YAMLEditor.svelte       — Add yamlSchemaCompletion alongside yamlSchema
frontend/src/lib/components/CreateResourceDialog.svelte — Add yamlSchemaCompletion, remove debug code
```

---

### Task 1: Package scaffold and fuzzy matching

Set up the package and implement the self-contained fuzzy matching module — it has no dependencies on the rest of the system so it's a clean starting point.

**Files:**
- Create: `packages/codemirror-yaml-completion/package.json`
- Create: `packages/codemirror-yaml-completion/tsconfig.json`
- Create: `packages/codemirror-yaml-completion/vitest.config.ts`
- Create: `packages/codemirror-yaml-completion/src/fuzzy.ts`
- Create: `packages/codemirror-yaml-completion/src/__tests__/fuzzy.test.ts`

- [ ] **Step 1: Create the package scaffold**

`packages/codemirror-yaml-completion/package.json`:
```json
{
  "name": "codemirror-yaml-completion",
  "version": "0.1.0",
  "private": false,
  "type": "module",
  "exports": {
    ".": {
      "types": "./src/index.ts",
      "default": "./src/index.ts"
    }
  },
  "scripts": {
    "test": "vitest run"
  },
  "peerDependencies": {
    "@codemirror/autocomplete": ">=6",
    "@codemirror/state": ">=6",
    "@codemirror/lang-yaml": ">=6",
    "json-schema-library": ">=9"
  },
  "dependencies": {
    "yaml": "^2.8.0"
  },
  "devDependencies": {
    "vitest": "^4",
    "@codemirror/autocomplete": "^6",
    "@codemirror/state": "^6",
    "@codemirror/language": "^6",
    "@codemirror/lang-yaml": "^6",
    "json-schema-library": "^9",
    "json-schema": "^0.4"
  }
}
```

`packages/codemirror-yaml-completion/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ESNext",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "allowJs": true,
    "isolatedModules": true,
    "strict": true,
    "declaration": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  },
  "include": ["src/**/*.ts"]
}
```

`packages/codemirror-yaml-completion/vitest.config.ts`:
```ts
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    include: ['src/**/*.test.ts'],
  },
})
```

Create a placeholder `src/index.ts`:
```ts
export { yamlSchemaCompletion } from './completion'
```

Run: `cd packages/codemirror-yaml-completion && pnpm install` (from repo root: `pnpm install`)

- [ ] **Step 2: Write fuzzy matching tests**

`packages/codemirror-yaml-completion/src/__tests__/fuzzy.test.ts`:
```ts
import { describe, it, expect } from 'vitest'
import { fuzzyMatch } from '../fuzzy'

describe('fuzzyMatch', () => {
  it('returns boost 2 for exact prefix match', () => {
    expect(fuzzyMatch('ann', 'annotations')).toBe(2)
  })

  it('returns boost 1 for case-insensitive prefix', () => {
    expect(fuzzyMatch('Ann', 'annotations')).toBe(1)
  })

  it('returns boost 0 for substring match', () => {
    expect(fuzzyMatch('age', 'managedFields')).toBe(0)
  })

  it('returns boost -1 for subsequence match', () => {
    expect(fuzzyMatch('mf', 'managedFields')).toBe(-1)
  })

  it('returns null for no match', () => {
    expect(fuzzyMatch('xyz', 'annotations')).toBeNull()
  })

  it('returns boost 0 for empty input (show all)', () => {
    expect(fuzzyMatch('', 'annotations')).toBe(0)
  })

  it('handles single character prefix match', () => {
    expect(fuzzyMatch('a', 'annotations')).toBe(2)
  })

  it('handles single character case-insensitive prefix', () => {
    expect(fuzzyMatch('A', 'annotations')).toBe(1)
  })

  it('handles single character substring (not prefix)', () => {
    expect(fuzzyMatch('t', 'metadata')).toBe(0)
  })

  it('returns null when subsequence fails', () => {
    expect(fuzzyMatch('zx', 'metadata')).toBeNull()
  })

  it('handles full exact match', () => {
    expect(fuzzyMatch('name', 'name')).toBe(2)
  })
})
```

Run: `cd packages/codemirror-yaml-completion && npx vitest run src/__tests__/fuzzy.test.ts`
Expected: FAIL (module not found)

- [ ] **Step 3: Implement fuzzy matching**

`packages/codemirror-yaml-completion/src/fuzzy.ts`:
```ts
/**
 * Fuzzy match an input string against a candidate.
 * Returns a boost score (higher = better match), or null for no match.
 *
 * - Exact prefix: 2
 * - Case-insensitive prefix: 1
 * - Substring (case-insensitive): 0
 * - Subsequence (case-insensitive): -1
 * - No match: null
 * - Empty input: 0 (show all)
 */
export function fuzzyMatch(input: string, candidate: string): number | null {
  if (input.length === 0) return 0

  if (candidate.startsWith(input)) return 2

  const lowerInput = input.toLowerCase()
  const lowerCandidate = candidate.toLowerCase()

  if (lowerCandidate.startsWith(lowerInput)) return 1

  if (lowerCandidate.includes(lowerInput)) return 0

  // Subsequence: each character of input appears in order in candidate
  let ci = 0
  for (let i = 0; i < lowerInput.length; i++) {
    const idx = lowerCandidate.indexOf(lowerInput[i], ci)
    if (idx === -1) return null
    ci = idx + 1
  }
  return -1
}
```

Run: `cd packages/codemirror-yaml-completion && npx vitest run src/__tests__/fuzzy.test.ts`
Expected: All PASS

- [ ] **Step 4: Commit**

```
jj new && jj desc -m "Add codemirror-yaml-completion package scaffold and fuzzy matching"
```

---

### Task 2: AST position mapping

Implement the core algorithm that takes an editor state + cursor position and returns the JSON pointer, typed prefix, and existing sibling keys.

**Files:**
- Create: `packages/codemirror-yaml-completion/src/position.ts`
- Create: `packages/codemirror-yaml-completion/src/__tests__/position.test.ts`

- [ ] **Step 1: Write position mapping tests**

`packages/codemirror-yaml-completion/src/__tests__/position.test.ts`:
```ts
import { describe, it, expect } from 'vitest'
import { EditorState } from '@codemirror/state'
import { yaml } from '@codemirror/lang-yaml'
import { resolvePosition } from '../position'

function stateAt(doc: string, cursorMarker = '|'): { state: EditorState; pos: number } {
  const pos = doc.indexOf(cursorMarker)
  const text = doc.slice(0, pos) + doc.slice(pos + 1)
  return { state: EditorState.create({ doc: text, extensions: [yaml()] }), pos }
}

describe('resolvePosition', () => {
  it('returns root pointer for empty document', () => {
    const { state, pos } = stateAt('|')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('')
    expect(result!.prefix).toBe('')
    expect(result!.existingKeys).toEqual([])
  })

  it('returns root pointer for empty line at root', () => {
    const { state, pos } = stateAt('apiVersion: v1\n|')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('')
    expect(result!.existingKeys).toContain('apiVersion')
  })

  it('returns nested pointer for empty line inside mapping', () => {
    const { state, pos } = stateAt('metadata:\n  name: test\n  |')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/metadata')
    expect(result!.existingKeys).toContain('name')
  })

  it('extracts typed prefix', () => {
    const { state, pos } = stateAt('metadata:\n  ann|')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/metadata')
    expect(result!.prefix).toBe('ann')
  })

  it('returns null for value position', () => {
    const { state, pos } = stateAt('apiVersion: |')
    const result = resolvePosition(state, pos)
    expect(result).toBeNull()
  })

  it('handles deeply nested path', () => {
    const { state, pos } = stateAt(
      'spec:\n  template:\n    spec:\n      |'
    )
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/spec/template/spec')
  })

  it('handles multiple existing keys at same level', () => {
    const { state, pos } = stateAt(
      'apiVersion: v1\nkind: Pod\nmetadata:\n  name: test\n|'
    )
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('')
    expect(result!.existingKeys).toEqual(
      expect.arrayContaining(['apiVersion', 'kind', 'metadata'])
    )
  })

  it('handles whitespace-only line with correct indentation', () => {
    const { state, pos } = stateAt('metadata:\n  name: test\n  |')
    const result = resolvePosition(state, pos)
    expect(result!.pointer).toBe('/metadata')
  })

  it('returns null for line with colon (value position)', () => {
    const { state, pos } = stateAt('metadata:\n  name: |')
    const result = resolvePosition(state, pos)
    expect(result).toBeNull()
  })
})
```

Run: `cd packages/codemirror-yaml-completion && npx vitest run src/__tests__/position.test.ts`
Expected: FAIL (module not found)

- [ ] **Step 2: Implement position mapping**

`packages/codemirror-yaml-completion/src/position.ts`:
```ts
import { EditorState } from '@codemirror/state'
import { parseDocument } from 'yaml'
import { isMap, isPair, isScalar, isSeq } from 'yaml'
import type { YAMLMap, Pair, Node as YAMLNode, ParsedNode } from 'yaml'

export interface PositionResult {
  /** JSON pointer to the enclosing map (e.g., "/metadata" or "" for root) */
  pointer: string
  /** Text the user has typed so far on this line (for matching) */
  prefix: string
  /** Property keys already present in the enclosing map */
  existingKeys: string[]
  /** Start offset of the prefix (for CompletionResult.from) */
  from: number
}

/**
 * Resolve the cursor position in a YAML document to a schema path context.
 * Returns null if the cursor is in a value position (after `key: `).
 */
export function resolvePosition(
  state: EditorState,
  pos: number
): PositionResult | null {
  // Extract prefix and detect value position from the current line
  const line = state.doc.lineAt(pos)
  const lineTextBeforeCursor = state.sliceDoc(line.from, pos)
  const trimmed = lineTextBeforeCursor.trimStart()

  // If the trimmed text contains ':', cursor is in a value position
  if (trimmed.includes(':')) return null

  const prefix = trimmed
  const from = pos - prefix.length

  // Parse the YAML document
  const doc = state.doc.toString()
  const yamlDoc = parseDocument(doc, { keepSourceTokens: true })

  // Find the enclosing map and derive the pointer
  const contents = yamlDoc.contents
  if (!contents) {
    // Empty or unparseable document — return root context
    return { pointer: '', prefix, existingKeys: [], from }
  }

  // Find the deepest YAMLMap that contains the cursor position
  const enclosingMap = findEnclosingMap(contents as ParsedNode, pos)

  if (!enclosingMap) {
    // Cursor is outside any map — treat as root if contents is a map
    if (isMap(contents)) {
      return {
        pointer: '',
        prefix,
        existingKeys: getMapKeys(contents),
        from,
      }
    }
    return { pointer: '', prefix, existingKeys: [], from }
  }

  const pointer = derivePointer(contents as ParsedNode, enclosingMap)
  const existingKeys = getMapKeys(enclosingMap)

  return { pointer, prefix, existingKeys, from }
}

/**
 * Recursively find the deepest YAMLMap node whose range contains the cursor.
 */
function findEnclosingMap(
  node: ParsedNode | Pair | null,
  pos: number
): YAMLMap.Parsed | null {
  if (!node) return null

  let best: YAMLMap.Parsed | null = null

  if (isMap(node)) {
    const range = (node as YAMLMap.Parsed).range
    if (range && pos >= range[0] && pos <= range[2]) {
      best = node as YAMLMap.Parsed
    }
  }

  // Recurse into children
  if (isMap(node)) {
    for (const pair of (node as YAMLMap).items) {
      const child = findEnclosingMap(pair as any, pos)
      if (child) best = child
    }
  } else if (isPair(node)) {
    const valResult = findEnclosingMap(node.value as ParsedNode | null, pos)
    if (valResult) best = valResult
  } else if (isSeq(node)) {
    for (const item of node.items) {
      const child = findEnclosingMap(item as ParsedNode, pos)
      if (child) best = child
    }
  }

  return best
}

/**
 * Derive the JSON pointer from root to a target YAMLMap node.
 */
function derivePointer(root: ParsedNode, target: YAMLMap.Parsed): string {
  const segments: string[] = []
  if (findPath(root, target, segments)) {
    return segments.length === 0 ? '' : '/' + segments.join('/')
  }
  return ''
}

function findPath(
  node: ParsedNode | Pair | null,
  target: YAMLMap.Parsed,
  segments: string[]
): boolean {
  if (!node) return false
  if (node === target) return true

  if (isMap(node)) {
    for (const pair of (node as YAMLMap).items) {
      if (isPair(pair) && isScalar(pair.key)) {
        segments.push(String(pair.key.value))
        if (findPath(pair.value as ParsedNode | null, target, segments)) {
          return true
        }
        segments.pop()
      }
    }
  } else if (isSeq(node)) {
    for (let i = 0; i < node.items.length; i++) {
      segments.push(String(i))
      if (findPath(node.items[i] as ParsedNode, target, segments)) {
        return true
      }
      segments.pop()
    }
  }

  return false
}

function getMapKeys(map: YAMLMap): string[] {
  const keys: string[] = []
  for (const pair of map.items) {
    if (isPair(pair) && isScalar(pair.key) && typeof pair.key.value === 'string') {
      keys.push(pair.key.value)
    }
  }
  return keys
}
```

Run: `cd packages/codemirror-yaml-completion && npx vitest run src/__tests__/position.test.ts`
Expected: All PASS

- [ ] **Step 3: Commit**

```
jj new && jj desc -m "Add AST position mapping for YAML cursor resolution"
```

---

### Task 3: Schema resolution

Implement the module that takes a JSON pointer + root schema and returns the list of available property names with their types and descriptions.

**Files:**
- Create: `packages/codemirror-yaml-completion/src/schema-resolve.ts`

No separate test file — this module is thin enough to be covered by the integration tests in Task 4. It's essentially a wrapper around `Draft07.getSchema()` plus collecting properties from `allOf`/`anyOf`/`oneOf` branches.

- [ ] **Step 1: Implement schema resolution**

`packages/codemirror-yaml-completion/src/schema-resolve.ts`:
```ts
import { Draft07, isJsonError } from 'json-schema-library'
import type { JSONSchema7, JSONSchema7Definition } from 'json-schema'

export interface SchemaProperty {
  name: string
  type: string
  description: string
}

/**
 * Resolve the schema at a JSON pointer and return available property names.
 * Returns null if the schema has no explicit properties at this path.
 */
export function resolveSchemaProperties(
  schema: JSONSchema7,
  pointer: string
): SchemaProperty[] | null {
  const draft = new Draft07(schema)

  let subSchema: any
  if (!pointer || pointer === '') {
    // Root level — use the schema directly
    subSchema = schema
  } else {
    subSchema = draft.getSchema({ pointer })
    if (isJsonError(subSchema)) {
      subSchema = subSchema.data?.schema
    }
  }

  if (!subSchema) return null

  const properties = collectProperties(subSchema, schema)
  if (properties.length === 0) return null

  return properties
}

function collectProperties(
  subSchema: any,
  rootSchema: JSONSchema7
): SchemaProperty[] {
  const result = new Map<string, SchemaProperty>()

  addProperties(subSchema, rootSchema, result)

  // Collect from allOf/anyOf/oneOf branches
  for (const key of ['allOf', 'anyOf', 'oneOf'] as const) {
    const branches = subSchema[key]
    if (Array.isArray(branches)) {
      for (const branch of branches) {
        const resolved = resolveRef(branch, rootSchema)
        if (resolved && typeof resolved === 'object') {
          addProperties(resolved, rootSchema, result)
        }
      }
    }
  }

  return Array.from(result.values())
}

function addProperties(
  schema: any,
  rootSchema: JSONSchema7,
  result: Map<string, SchemaProperty>
): void {
  const resolved = resolveRef(schema, rootSchema)
  if (!resolved || typeof resolved !== 'object' || !resolved.properties) return

  for (const [name, def] of Object.entries(resolved.properties)) {
    if (result.has(name)) continue
    if (typeof def === 'boolean') continue
    const prop = def as JSONSchema7
    const type = Array.isArray(prop.type) ? prop.type.join(' | ') : (prop.type ?? '')
    result.set(name, {
      name,
      type,
      description: prop.description ?? '',
    })
  }
}

function resolveRef(schema: any, root: JSONSchema7): any {
  if (!schema || typeof schema !== 'object') return schema
  if (!schema.$ref) return schema

  const refPath = schema.$ref.split('/')
  let current: any = root
  for (const segment of refPath) {
    if (segment === '#') { current = root; continue }
    current = current?.[segment]
  }
  return current
}
```

- [ ] **Step 2: Commit**

```
jj new && jj desc -m "Add schema resolution for property completions"
```

---

### Task 4: CompletionSource and integration tests

Wire everything together into the `CompletionSource` and write integration tests that exercise the full pipeline.

**Files:**
- Create: `packages/codemirror-yaml-completion/src/completion.ts`
- Update: `packages/codemirror-yaml-completion/src/index.ts`
- Create: `packages/codemirror-yaml-completion/src/__tests__/completion.test.ts`

- [ ] **Step 1: Write integration tests**

`packages/codemirror-yaml-completion/src/__tests__/completion.test.ts`:
```ts
import { describe, it, expect } from 'vitest'
import { EditorState } from '@codemirror/state'
import { yaml } from '@codemirror/lang-yaml'
import { CompletionContext } from '@codemirror/autocomplete'
import { yamlSchemaCompletion } from '../index'
import type { JSONSchema7 } from 'json-schema'

const testSchema: JSONSchema7 = {
  type: 'object',
  properties: {
    apiVersion: { type: 'string', description: 'API version' },
    kind: { type: 'string', description: 'Resource kind' },
    metadata: {
      type: 'object',
      description: 'Standard metadata',
      properties: {
        name: { type: 'string', description: 'Resource name' },
        namespace: { type: 'string', description: 'Resource namespace' },
        labels: { type: 'object', description: 'Labels', additionalProperties: { type: 'string' } },
        annotations: { type: 'object', description: 'Annotations', additionalProperties: { type: 'string' } },
        managedFields: { type: 'array', description: 'Managed fields' },
      },
    },
    spec: {
      type: 'object',
      description: 'Spec',
      properties: {
        replicas: { type: 'integer', description: 'Number of replicas' },
      },
    },
  },
}

function createContext(
  doc: string,
  cursorMarker = '|',
  explicit = true
): { ctx: CompletionContext; state: EditorState } {
  const pos = doc.indexOf(cursorMarker)
  const text = doc.slice(0, pos) + doc.slice(pos + 1)
  const state = EditorState.create({
    doc: text,
    extensions: [yaml(), yamlSchemaCompletion(testSchema)],
  })
  const ctx = new CompletionContext(state, pos, explicit)
  return { ctx, state }
}

// Helper to extract the completion source from the extension
async function complete(doc: string, explicit = true) {
  const { ctx } = createContext(doc, '|', explicit)
  // The completion source is registered on the yaml language data.
  // We need to find it and call it directly.
  const sources = ctx.state.languageDataAt<(ctx: CompletionContext) => any>(
    'autocomplete',
    ctx.pos
  )
  // Find our source (the one that isn't null for these cases)
  for (const source of sources) {
    const result = await source(ctx)
    if (result && result.options && result.options.length > 0) {
      return result
    }
  }
  return null
}

describe('yamlSchemaCompletion', () => {
  it('returns all root properties on empty document', async () => {
    const result = await complete('|')
    expect(result).not.toBeNull()
    const labels = result!.options.map((o: any) => o.label)
    expect(labels).toContain('apiVersion')
    expect(labels).toContain('kind')
    expect(labels).toContain('metadata')
    expect(labels).toContain('spec')
  })

  it('returns root properties on empty line at root', async () => {
    const result = await complete('apiVersion: v1\n|')
    expect(result).not.toBeNull()
    const labels = result!.options.map((o: any) => o.label)
    expect(labels).toContain('kind')
    expect(labels).toContain('metadata')
    expect(labels).not.toContain('apiVersion') // already present
  })

  it('returns metadata properties inside metadata block', async () => {
    const result = await complete('metadata:\n  name: test\n  |')
    expect(result).not.toBeNull()
    const labels = result!.options.map((o: any) => o.label)
    expect(labels).toContain('namespace')
    expect(labels).toContain('annotations')
    expect(labels).not.toContain('name') // already present
  })

  it('returns fuzzy matches when typing', async () => {
    const result = await complete('metadata:\n  a|', false)
    expect(result).not.toBeNull()
    const labels = result!.options.map((o: any) => o.label)
    // Should include all metadata properties via fuzzy, not just 'annotations'
    expect(labels).toContain('annotations')
    expect(labels).toContain('name')
    expect(labels).toContain('namespace')
    expect(labels).toContain('managedFields')
  })

  it('returns null for value position', async () => {
    const result = await complete('apiVersion: |')
    expect(result).toBeNull()
  })

  it('excludes existing properties', async () => {
    const result = await complete(
      'apiVersion: v1\nkind: Pod\nmetadata:\n  name: x\n|'
    )
    expect(result).not.toBeNull()
    const labels = result!.options.map((o: any) => o.label)
    expect(labels).not.toContain('apiVersion')
    expect(labels).not.toContain('kind')
    expect(labels).not.toContain('metadata')
    expect(labels).toContain('spec')
  })

  it('returns null when schema has no properties at path', async () => {
    // labels has additionalProperties but no explicit properties
    const result = await complete('metadata:\n  labels:\n    |')
    expect(result).toBeNull()
  })

  it('provides type and description in completions', async () => {
    const result = await complete('|')
    expect(result).not.toBeNull()
    const apiVersionOption = result!.options.find(
      (o: any) => o.label === 'apiVersion'
    )
    expect(apiVersionOption).toBeDefined()
    expect(apiVersionOption!.detail).toBe('string')
    expect(apiVersionOption!.type).toBe('property')
  })

  it('applies property name with colon suffix', async () => {
    const result = await complete('|')
    expect(result).not.toBeNull()
    const option = result!.options.find((o: any) => o.label === 'apiVersion')
    expect(option!.apply).toBe('apiVersion: ')
  })

  it('subsequence matching works', async () => {
    const result = await complete('metadata:\n  mf|', false)
    expect(result).not.toBeNull()
    const labels = result!.options.map((o: any) => o.label)
    expect(labels).toContain('managedFields')
  })
})
```

Run: `cd packages/codemirror-yaml-completion && npx vitest run src/__tests__/completion.test.ts`
Expected: FAIL (module not found)

- [ ] **Step 2: Implement the CompletionSource**

`packages/codemirror-yaml-completion/src/completion.ts`:
```ts
import type { CompletionContext, CompletionResult, Completion } from '@codemirror/autocomplete'
import { yamlLanguage } from '@codemirror/lang-yaml'
import type { Extension } from '@codemirror/state'
import type { JSONSchema7 } from 'json-schema'
import { resolvePosition } from './position'
import { resolveSchemaProperties } from './schema-resolve'
import { fuzzyMatch } from './fuzzy'

/**
 * Creates a CodeMirror extension that provides schema-driven YAML property
 * completions. Supplements codemirror-json-schema's yamlCompletion with:
 * - Completions on empty lines (where the upstream source fails)
 * - Fuzzy matching (substring + subsequence, not just prefix)
 *
 * Registers as a CompletionSource on the YAML language. Coexists with
 * yamlCompletion — CodeMirror merges and deduplicates results by label.
 */
export function yamlSchemaCompletion(schema: JSONSchema7): Extension {
  return yamlLanguage.data.of({
    autocomplete: (ctx: CompletionContext): CompletionResult | null => {
      return doComplete(ctx, schema)
    },
  })
}

function doComplete(
  ctx: CompletionContext,
  schema: JSONSchema7
): CompletionResult | null {
  // Resolve cursor position to schema context
  const position = resolvePosition(ctx.state, ctx.pos)
  if (!position) return null

  // Resolve schema at the JSON pointer
  const properties = resolveSchemaProperties(schema, position.pointer)
  if (!properties) return null

  // Filter out existing sibling keys
  const existing = new Set(position.existingKeys)
  const candidates = properties.filter((p) => !existing.has(p.name))

  // Build completion options with fuzzy matching
  const options: Completion[] = []
  for (const prop of candidates) {
    const boost = fuzzyMatch(position.prefix, prop.name)
    if (boost === null) continue

    options.push({
      label: prop.name,
      apply: `${prop.name}: `,
      type: 'property',
      detail: prop.type,
      info: prop.description || undefined,
      boost,
    })
  }

  if (options.length === 0) return null

  return {
    from: position.from,
    options,
    filter: false,
  }
}
```

Update `packages/codemirror-yaml-completion/src/index.ts`:
```ts
export { yamlSchemaCompletion } from './completion'
```

Run: `cd packages/codemirror-yaml-completion && npx vitest run`
Expected: All tests PASS (fuzzy, position, completion)

- [ ] **Step 3: Commit**

```
jj new && jj desc -m "Implement CompletionSource with integration tests"
```

---

### Task 5: Integrate into klados consumers and clean up

Wire the new package into both YAML editors and remove debug instrumentation.

**Files:**
- Modify: `packages/ui/package.json` — add peer dep
- Modify: `packages/ui/src/lib/YAMLEditor.svelte:101-107` — use `yamlSchemaCompletion`
- Modify: `frontend/package.json` — add dep
- Modify: `frontend/src/lib/components/CreateResourceDialog.svelte:67-95` — use `yamlSchemaCompletion`, remove debug code

- [ ] **Step 1: Add dependency to packages/ui**

In `packages/ui/package.json`, add to `peerDependencies`:
```json
"codemirror-yaml-completion": ">=0.1"
```

- [ ] **Step 2: Update YAMLEditor.svelte**

In `packages/ui/src/lib/YAMLEditor.svelte`, add import:
```ts
import { yamlSchemaCompletion } from 'codemirror-yaml-completion'
```

Change `safeSchemaExtensions` (line 101-107) from:
```ts
function safeSchemaExtensions(s: Record<string, any>) {
  try {
    return yamlSchema(s as any)
  } catch {
    return yamlLang()
  }
}
```
to:
```ts
function safeSchemaExtensions(s: Record<string, any>) {
  try {
    return [yamlSchema(s as any), yamlSchemaCompletion(s as any)]
  } catch {
    return yamlLang()
  }
}
```

- [ ] **Step 3: Update CreateResourceDialog.svelte**

In `frontend/package.json`, add to `dependencies`:
```json
"codemirror-yaml-completion": "workspace:*"
```

In `CreateResourceDialog.svelte`, add import:
```ts
import { yamlSchemaCompletion } from 'codemirror-yaml-completion'
```

Change the extensions array in `rebuildEditor` (around line 67):
```ts
extensions: [
  ...cmYamlExtensions({
    lang: schema ? [yamlSchema(schema), yamlSchemaCompletion(schema)] : undefined,
  }),
  EditorView.updateListener.of((update) => {
    if (update.docChanged) {
      editorDirty = true;
    }
  }),
],
```

This removes both debug blocks:
- The `EditorView.domEventHandlers` Ctrl+Space logger (lines 73-86)
- The `yamlLanguage.data.of` debug completion source (lines 87-95)

Remove the `yamlLanguage` import if no longer used:
```ts
// Remove this line if nothing else references yamlLanguage:
import {yamlLanguage} from "@codemirror/lang-yaml";
```

Run: `pnpm install` from repo root to link the workspace package.

- [ ] **Step 4: Verify the build and type-check**

Run: `cd frontend && pnpm check` — verify no type errors.
Run: `cd packages/codemirror-yaml-completion && npx vitest run` — verify all package tests still pass.

- [ ] **Step 5: Commit**

```
jj new && jj desc -m "Integrate codemirror-yaml-completion into YAMLEditor and CreateResourceDialog"
```

---

### Task 6: Manual verification

Start the dev server and verify the completion behavior in the browser.

- [ ] **Step 1: Start dev server and test**

Run: `task dev`

Test these scenarios in the Create Resource dialog and the resource detail YAML editor:

1. Open a resource YAML editor with a schema loaded (check for "Schema active" indicator)
2. Go to an empty line at root level → press Ctrl+Space → should show all root properties
3. Inside `metadata:`, go to an empty line → press Ctrl+Space → should show metadata properties (minus existing ones)
4. Inside `metadata:`, type `a` → should show all metadata properties (fuzzy), not just `annotations`
5. Type `mf` → should show `managedFields` (subsequence match)
6. After `apiVersion: ` (value position) → Ctrl+Space should NOT show property completions from our source (upstream handles values)

- [ ] **Step 2: Final commit if any fixes needed**

If manual testing reveals issues, fix them and commit.
