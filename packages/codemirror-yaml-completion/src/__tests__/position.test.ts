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
