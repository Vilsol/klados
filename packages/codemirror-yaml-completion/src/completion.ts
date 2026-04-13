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
      try {
        return doComplete(ctx, schema)
      } catch {
        return null
      }
    },
  })
}

function doComplete(
  ctx: CompletionContext,
  schema: JSONSchema7
): CompletionResult | null {
  const position = resolvePosition(ctx.state, ctx.pos)
  if (!position) return null

  const properties = resolveSchemaProperties(schema, position.pointer)
  if (!properties) return null

  const existing = new Set(position.existingKeys)
  const candidates = properties.filter((p) => !existing.has(p.name))

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
