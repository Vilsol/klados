# YAML Schema Completion — Supplementary CompletionSource

**Date**: 2026-04-13
**Status**: Design approved

## Problem

`codemirror-json-schema` v0.8.1's `yamlCompletion()` fails to provide property name
completions on empty lines or whitespace-only lines. On empty lines,
`syntaxTree(state).resolveInner(pos, -1)` resolves to the document root node, causing
the library's property-vs-value routing to fall into the value branch, which returns
nothing useful.

Additionally, the library only does prefix matching — typing `a` inside `metadata:`
only shows `annotations`, not `managedFields`, `name`, `namespace`, etc.

GitHub issue #121 was filed and closed "Not Planned". v0.8.1 is the latest release
(April 2025) with no fix in sight and no configuration hooks to work around it.

## Solution

A standalone package (`codemirror-yaml-completion`) that registers a supplementary
`CompletionSource` on the YAML language. It coexists with `yamlCompletion` — CodeMirror
merges results from all registered sources and deduplicates by label.

## Package

**Name**: `codemirror-yaml-completion` (publishable standalone for the community)

**Location**: `packages/codemirror-yaml-completion/` in the klados monorepo

**Dependencies**:
- `@codemirror/autocomplete` — peer
- `@codemirror/state` — peer
- `@codemirror/lang-yaml` — peer
- `json-schema-library` — peer (consumer already has it via `codemirror-json-schema`)
- `yaml` — direct dependency (for `parseDocument` AST parsing)

## API

Single exported function:

```ts
import type { Extension } from '@codemirror/state'
import type { JSONSchema7 } from 'json-schema'

export function yamlSchemaCompletion(schema: JSONSchema7): Extension
```

Returns a `yamlLanguage.data.of({ autocomplete: ... })` extension. Consumers add it
alongside `yamlSchema()`:

```ts
cmYamlExtensions({
  lang: [yamlSchema(schema), yamlSchemaCompletion(schema)]
})
```

## Activation Rules

The CompletionSource activates in these cases:

1. **`ctx.explicit === true`** (Ctrl+Space) — always activate, return all valid
   properties at the cursor's schema path.
2. **Cursor line has no property key yet** — empty line or whitespace-only, or cursor
   is at start of a word that hasn't formed a `key:` pair. This is where the upstream
   source fails.
3. **On typing (non-explicit)** — activate and return fuzzy-matched results. The
   upstream returns prefix-matched results; ours returns fuzzy-matched results.
   CodeMirror merges both, so duplicates are naturally deduplicated by label.

The source returns `null` (delegates to upstream) when:
- The cursor is in a **value position** (after `key: ` on the same line)
- The schema at the resolved path has no explicit `properties`
  (e.g., `additionalProperties: true`)

Returns `CompletionResult` with `filter: false` — we handle our own fuzzy filtering.

## Core Algorithm: AST Position Mapping

### Step 1: Parse the document

```ts
const doc = state.doc.toString()
const yamlDoc = parseDocument(doc, { keepSourceTokens: true })
```

`parseDocument` handles incomplete/invalid YAML gracefully — it produces error nodes
but still builds a tree. The document is typically small (a single Kubernetes resource).

### Step 2: Find the enclosing map node

Walk the YAML AST to find the deepest `YAMLMap` whose `range` contains the cursor offset.

- Traverse recursively
- For each `YAMLMap`, check if `cursor >= range[0] && cursor <= range[2]`
- Pick the deepest (most specific) match
- For empty lines between properties, the cursor falls in a gap — the enclosing map's
  range still encompasses it

### Step 3: Derive the JSON pointer

Walk from the enclosing `YAMLMap` up to the root, collecting path segments:

- Parent is a `Pair` — use the pair's key as a path segment
- Parent is a `YAMLSeq` — use the item index as a path segment

Produces pointers like `/metadata` or `/spec/template/spec/containers/0`.

### Step 4: Extract the typed prefix

Read the current line from start-of-content (after indentation) to cursor position.
If it contains `:`, the cursor is in a value position — return `null`.
Otherwise the text is the property name prefix for fuzzy matching.

### Step 5: Determine existing sibling keys

From the enclosing `YAMLMap`'s `items` array, collect all existing key names.
These are excluded from completion results.

### Edge cases

- **Empty document**: no AST nodes. Pointer is `""` (root), show root-level properties.
- **Cursor after last property in a map**: offset is past the last item's range but
  within the map's range. The enclosing map is still correct.
- **Invalid YAML mid-edit**: `parseDocument` produces errors but builds a partial tree.
  Use whatever structure it managed to parse.
- **Flow-style mappings** (`{a: 1}`): the `yaml` library parses these into the same
  `YAMLMap` nodes, so position mapping works identically.

## Schema Resolution

Use `json-schema-library`'s `Draft07`:

```ts
const draft = new Draft07(schema)
const subSchema = draft.getSchema({ pointer })
```

Handles `$ref` resolution, `allOf`/`anyOf`/`oneOf` merging. If it returns a `JsonError`
or no schema, return `null`.

From the resolved sub-schema, read `properties` for valid property names. For
`allOf`/`anyOf`/`oneOf` at the target level, collect properties from all branches
(union of all possible keys).

## Fuzzy Matching

When the user has typed a prefix:

| Match type | Example | Boost |
|---|---|---|
| Exact prefix | `ann` → `annotations` | 2 |
| Case-insensitive prefix | `Ann` → `annotations` | 1 |
| Substring | `age` → `managedFields` | 0 |
| Subsequence | `mf` → `managedFields` | -1 |
| No match | excluded | — |

On explicit trigger (Ctrl+Space) with no typed text, all properties get `boost: 0`.

## Completion Items

Each completion:

- `label`: property name (e.g., `metadata`)
- `type`: `"property"`
- `detail`: schema type (e.g., `object`, `string`)
- `info`: schema `description` if present
- `apply`: property name + `: ` (simple — no snippets or default values)
- `boost`: per fuzzy match ranking above

## Integration into klados

### Consumer changes

`YAMLEditor.svelte`:
```ts
import { yamlSchemaCompletion } from '@klados/codemirror-yaml-completion'

function safeSchemaExtensions(s) {
  return [yamlSchema(s), yamlSchemaCompletion(s)]
}
```

`CreateResourceDialog.svelte`:
```ts
import { yamlSchemaCompletion } from '@klados/codemirror-yaml-completion'

cmYamlExtensions({
  lang: schema ? [yamlSchema(schema), yamlSchemaCompletion(schema)] : undefined
})
```

### Cleanup

Remove debug instrumentation from `CreateResourceDialog.svelte`:
- `EditorView.domEventHandlers` Ctrl+Space logging
- `yamlLanguage.data.of` debug completion source

## Testing

Unit tests within the package using `EditorState.create()` with YAML document + schema:

- Empty line at root → all root properties
- Empty line inside `metadata:` → metadata properties minus existing siblings
- Typing `a` inside `metadata:` → fuzzy matches including annotations, name, namespace, etc.
- Ctrl+Space after `key: ` (value position) → null (delegate to upstream)
- Empty document → root properties
- Invalid/partial YAML → graceful degradation, no errors
- Existing properties excluded from results
- Subsequence matching (`mf` → `managedFields`)
