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
