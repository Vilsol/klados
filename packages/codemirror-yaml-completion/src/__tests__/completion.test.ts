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

async function complete(doc: string, explicit = true) {
  const pos = doc.indexOf('|')
  const text = doc.slice(0, pos) + doc.slice(pos + 1)
  const state = EditorState.create({
    doc: text,
    extensions: [yaml(), yamlSchemaCompletion(testSchema)],
  })
  const ctx = new CompletionContext(state, pos, explicit)
  const sources = ctx.state.languageDataAt<(ctx: CompletionContext) => any>(
    'autocomplete',
    ctx.pos
  )
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
    expect(labels).not.toContain('apiVersion')
  })

  it('returns metadata properties inside metadata block', async () => {
    const result = await complete('metadata:\n  name: test\n  |')
    expect(result).not.toBeNull()
    const labels = result!.options.map((o: any) => o.label)
    expect(labels).toContain('namespace')
    expect(labels).toContain('annotations')
    expect(labels).not.toContain('name')
  })

  it('returns fuzzy matches when typing', async () => {
    const result = await complete('metadata:\n  a|', false)
    expect(result).not.toBeNull()
    const labels = result!.options.map((o: any) => o.label)
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
