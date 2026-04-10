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
