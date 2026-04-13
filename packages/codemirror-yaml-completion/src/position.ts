import { EditorState } from '@codemirror/state'
import { parseDocument } from 'yaml'
import { isMap, isPair, isScalar, isSeq } from 'yaml'
import type { YAMLMap, Pair, ParsedNode } from 'yaml'

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
  const line = state.doc.lineAt(pos)
  const lineTextBeforeCursor = state.sliceDoc(line.from, pos)
  const trimmed = lineTextBeforeCursor.trimStart()

  if (trimmed.includes(':')) return null

  const prefix = trimmed
  const from = pos - prefix.length
  const cursorIndent = lineTextBeforeCursor.length - trimmed.length

  const doc = state.doc.toString()
  const yamlDoc = parseDocument(doc, { keepSourceTokens: true })

  const contents = yamlDoc.contents
  if (!contents) {
    return { pointer: '', prefix, existingKeys: [], from }
  }

  if (!isMap(contents)) {
    return { pointer: '', prefix, existingKeys: [], from }
  }

  // Collect all maps with their indentation levels and pointers
  const maps = collectMaps(contents as YAMLMap.Parsed, state, [])

  // First try: find the deepest map whose range contains the cursor
  let bestMap: MapInfo | null = null
  for (const m of maps) {
    if (pos >= m.range[0] && pos <= m.range[2]) {
      if (!bestMap || m.depth > bestMap.depth) {
        bestMap = m
      }
    }
  }

  // Fallback: cursor is outside all ranges (empty/trailing lines) or
  // inside a map but the typed prefix was parsed as a scalar value.
  // Use indentation to find the right map — the map whose content indent
  // (indent of child keys) matches the cursor's indent.
  if (!bestMap || needsIndentFallback(bestMap, cursorIndent)) {
    // Find the map whose contentIndent matches cursor indent.
    // If multiple match, pick the deepest one that precedes the cursor.
    const candidates = maps
      .filter((m) => m.range[0] <= pos && m.contentIndent === cursorIndent)
      .sort((a, b) => b.depth - a.depth)

    if (candidates.length > 0) {
      bestMap = candidates[0]
    } else {
      // No exact contentIndent match — the cursor is at an indent level
      // where no map's children live. This means we're typing inside a
      // key whose value hasn't become a map yet (e.g., `metadata:\n  ann`
      // where `ann` is parsed as scalar value, not a key).
      //
      // Find the pair in the nearest parent map whose key is at
      // (cursorIndent - indentStep) and resolve to that pair's path.
      const parentMap = findParentMapForNewBlock(maps, pos, cursorIndent)
      if (parentMap) {
        bestMap = parentMap
      } else {
        bestMap = maps.find((m) => m.depth === 0) ?? null
      }
    }
  }

  if (!bestMap) {
    return {
      pointer: '',
      prefix,
      existingKeys: getMapKeys(contents),
      from,
    }
  }

  return {
    pointer: bestMap.pointer,
    prefix,
    existingKeys: getMapKeys(bestMap.map),
    from,
  }
}

/**
 * Check if the initial range-based match is wrong and we need indent fallback.
 */
function needsIndentFallback(map: MapInfo, cursorIndent: number): boolean {
  return map.contentIndent !== cursorIndent
}

/**
 * When no map has contentIndent matching cursorIndent, the cursor is likely
 * inside a key:value pair where the value is not yet a map (e.g., typing a
 * new key under `metadata:` when the parser sees the typed text as a scalar).
 *
 * Walk all maps and find a pair whose key indent + indentStep matches
 * cursorIndent. Return a synthetic MapInfo pointing to that pair's path
 * (the pointer the map *would* have if it existed).
 */
function findParentMapForNewBlock(
  maps: MapInfo[],
  pos: number,
  cursorIndent: number
): MapInfo | null {
  // Walk maps from deepest to shallowest, looking for one that contains
  // a pair whose value would be at cursorIndent
  const sorted = [...maps].filter((m) => m.range[0] <= pos).sort((a, b) => b.depth - a.depth)

  for (const m of sorted) {
    for (const pair of m.map.items) {
      if (!isPair(pair) || !isScalar(pair.key)) continue
      const keyRange = (pair.key as any).range as [number, number, number] | undefined
      if (!keyRange) continue

      // The key's indent
      const keyIndent = m.contentIndent

      // If cursor indent is deeper than this key's indent, the cursor could
      // be inside this pair's value block
      if (cursorIndent > keyIndent) {
        const key = String(pair.key.value)
        const pairPointer = m.pointer ? m.pointer + '/' + key : '/' + key
        // Return a synthetic MapInfo — we use the parent map but with
        // the pointer pointing to this pair's key
        return {
          map: { items: [] } as any, // no existing keys — it's a new block
          range: m.range,
          pointer: pairPointer,
          depth: m.depth + 1,
          indent: m.contentIndent,
          contentIndent: cursorIndent,
        }
      }
    }
  }

  return null
}

interface MapInfo {
  map: YAMLMap.Parsed
  range: [number, number, number]
  pointer: string
  depth: number
  /** The indentation of the map node's key (or 0 for root) */
  indent: number
  /** The indentation of the map's child keys */
  contentIndent: number
}

function collectMaps(
  node: ParsedNode | null,
  state: EditorState,
  pathSegments: string[],
  depth = 0
): MapInfo[] {
  if (!node) return []
  const results: MapInfo[] = []

  if (isMap(node)) {
    const parsed = node as YAMLMap.Parsed
    const range = parsed.range
    if (range) {
      const indent = state.doc.lineAt(range[0]).from
      const mapIndent = range[0] - indent

      // Content indent: the indent of the first child key
      let contentIndent = mapIndent
      for (const pair of parsed.items) {
        if (isPair(pair) && isScalar(pair.key)) {
          const keyRange = (pair.key as any).range as [number, number, number] | undefined
          if (keyRange) {
            const keyLineFrom = state.doc.lineAt(keyRange[0]).from
            contentIndent = keyRange[0] - keyLineFrom
            break
          }
        }
      }

      results.push({
        map: parsed,
        range,
        pointer: pathSegments.length === 0 ? '' : '/' + pathSegments.join('/'),
        depth,
        indent: mapIndent,
        contentIndent,
      })
    }

    for (const pair of parsed.items) {
      if (isPair(pair) && isScalar(pair.key)) {
        const key = String(pair.key.value)
        if (pair.value) {
          results.push(
            ...collectMaps(
              pair.value as ParsedNode,
              state,
              [...pathSegments, key],
              depth + 1
            )
          )
        }
      }
    }
  } else if (isSeq(node)) {
    for (let i = 0; i < node.items.length; i++) {
      results.push(
        ...collectMaps(
          node.items[i] as ParsedNode,
          state,
          [...pathSegments, String(i)],
          depth + 1
        )
      )
    }
  }

  return results
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
