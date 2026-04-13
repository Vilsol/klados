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
  // === Empty document / root level ===

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

  it('returns root with all existing keys', () => {
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

  // === Nested maps ===

  it('returns nested pointer for empty line inside mapping', () => {
    const { state, pos } = stateAt('metadata:\n  name: test\n  |')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/metadata')
    expect(result!.existingKeys).toContain('name')
  })

  it('handles whitespace-only line with correct indentation', () => {
    const { state, pos } = stateAt('metadata:\n  name: test\n  |')
    const result = resolvePosition(state, pos)
    expect(result!.pointer).toBe('/metadata')
  })

  it('handles deeply nested path', () => {
    const { state, pos } = stateAt(
      'spec:\n  template:\n    spec:\n      |'
    )
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/spec/template/spec')
  })

  // === Typed prefix ===

  it('extracts typed prefix', () => {
    const { state, pos } = stateAt('metadata:\n  ann|')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/metadata')
    expect(result!.prefix).toBe('ann')
  })

  it('extracts single-character prefix', () => {
    const { state, pos } = stateAt('metadata:\n  a|')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/metadata')
    expect(result!.prefix).toBe('a')
  })

  it('returns empty prefix on whitespace-only line', () => {
    const { state, pos } = stateAt('metadata:\n  |')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.prefix).toBe('')
  })

  // === Value position detection ===

  it('returns null for value position after colon', () => {
    const { state, pos } = stateAt('apiVersion: |')
    const result = resolvePosition(state, pos)
    expect(result).toBeNull()
  })

  it('returns null for value position in nested key', () => {
    const { state, pos } = stateAt('metadata:\n  name: |')
    const result = resolvePosition(state, pos)
    expect(result).toBeNull()
  })

  it('returns null when cursor is after colon with value', () => {
    const { state, pos } = stateAt('apiVersion: v|')
    const result = resolvePosition(state, pos)
    expect(result).toBeNull()
  })

  // === Sibling maps at same depth — the critical bug ===

  it('resolves to spec, not metadata, when cursor is inside spec after metadata', () => {
    const { state, pos } = stateAt(
      'metadata:\n  name: test\nspec:\n  containers:\n  - name: nginx\n  |'
    )
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/spec')
    expect(result!.existingKeys).toContain('containers')
  })

  it('resolves to correct sibling map when multiple exist at same depth', () => {
    const { state, pos } = stateAt(
      'apiVersion: v1\nmetadata:\n  name: test\nspec:\n  |'
    )
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/spec')
  })

  it('resolves to first map when cursor is inside first sibling', () => {
    const { state, pos } = stateAt(
      'metadata:\n  name: test\n  |\nspec:\n  replicas: 1'
    )
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/metadata')
  })

  it('resolves correctly with three sibling maps', () => {
    const { state, pos } = stateAt(
      'metadata:\n  name: a\nspec:\n  replicas: 1\nstatus:\n  |'
    )
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/status')
  })

  // === Empty/new map values (key exists but no map children yet) ===

  it('resolves to new map block under key with no value', () => {
    const { state, pos } = stateAt('spec:\n  |')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/spec')
  })

  it('resolves to new map block under deeply nested key', () => {
    const { state, pos } = stateAt('spec:\n  template:\n    |')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/spec/template')
  })

  // === Gap between key and value map (cursor on blank line before first child) ===

  it('resolves to spec when cursor is on empty line between spec: and containers:', () => {
    const doc = [
      'apiVersion: v1',
      'kind: Pod',
      'metadata:',
      '  name: ""',
      '  namespace: default',
      'spec:',
      '  |',
      '  containers:',
      '    - name: ""',
      '      image: ""',
      '      resources:',
      '        requests:',
      '          cpu: 100m',
      '          memory: 128Mi',
    ].join('\n')
    const { state, pos } = stateAt(doc)
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/spec')
    expect(result!.existingKeys).toContain('containers')
  })

  it('resolves to metadata when cursor is on empty line between metadata: and name:', () => {
    const doc = [
      'apiVersion: v1',
      'metadata:',
      '  |',
      '  name: test',
      '  namespace: default',
    ].join('\n')
    const { state, pos } = stateAt(doc)
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/metadata')
    expect(result!.existingKeys).toContain('name')
    expect(result!.existingKeys).toContain('namespace')
  })

  // === Realistic Kubernetes YAML ===

  it('resolves inside a Pod spec with multiple sections', () => {
    const doc = [
      'apiVersion: v1',
      'kind: Pod',
      'metadata:',
      '  name: nginx',
      '  namespace: default',
      '  labels:',
      '    app: nginx',
      'spec:',
      '  |',
    ].join('\n')
    const { state, pos } = stateAt(doc)
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/spec')
  })

  it('resolves inside metadata labels', () => {
    const doc = [
      'apiVersion: v1',
      'kind: Pod',
      'metadata:',
      '  name: nginx',
      '  labels:',
      '    app: nginx',
      '    |',
    ].join('\n')
    const { state, pos } = stateAt(doc)
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('/metadata/labels')
    expect(result!.existingKeys).toContain('app')
  })

  it('resolves at root after full document', () => {
    const doc = [
      'apiVersion: v1',
      'kind: Pod',
      'metadata:',
      '  name: nginx',
      'spec:',
      '  containers:',
      '  - name: nginx',
      '|',
    ].join('\n')
    const { state, pos } = stateAt(doc)
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('')
    expect(result!.existingKeys).toEqual(
      expect.arrayContaining(['apiVersion', 'kind', 'metadata', 'spec'])
    )
  })

  // === Edge cases ===

  it('handles document with only a key and no value', () => {
    const { state, pos } = stateAt('apiVersion:\n|')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('')
    expect(result!.existingKeys).toContain('apiVersion')
  })

  it('handles cursor at very start of document', () => {
    const { state, pos } = stateAt('|apiVersion: v1')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('')
  })

  it('handles prefix at root level', () => {
    const { state, pos } = stateAt('apiVersion: v1\nki|')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.pointer).toBe('')
    expect(result!.prefix).toBe('ki')
  })

  it('handles from offset correctly for prefix', () => {
    const { state, pos } = stateAt('metadata:\n  ann|')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    // from should be at the start of 'ann', not at cursor
    expect(result!.from).toBe(pos - 3)
  })

  it('from equals pos when prefix is empty', () => {
    const { state, pos } = stateAt('metadata:\n  |')
    const result = resolvePosition(state, pos)
    expect(result).not.toBeNull()
    expect(result!.from).toBe(pos)
  })
})
