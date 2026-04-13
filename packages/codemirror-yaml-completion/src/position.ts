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
  if (!bestMap || bestMap.contentIndent !== cursorIndent) {
    bestMap = resolveByIndent(maps, pos, cursorIndent)
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
 * Resolve the correct map using indentation when range-based lookup fails.
 *
 * Strategy: walk all maps in the document and find pairs (key: value) where
 * the cursor could belong. The cursor belongs to the pair whose key is the
 * closest *before* the cursor at the indent level one step above cursorIndent.
 * If that pair has a map value, use it. If not, create a synthetic entry.
 */
function resolveByIndent(
  maps: MapInfo[],
  pos: number,
  cursorIndent: number
): MapInfo | null {
  // Strategy 1: Find the pair closest before cursor whose map value's
  // contentIndent matches cursorIndent.
  // Strategy 2: If no such map exists, find the pair closest before cursor
  // at one indent level up, whose value block would contain cursorIndent.

  // Collect all candidate contexts: real maps and potential synthetic parents
  interface Candidate {
    mapInfo: MapInfo
    /** Position of the key that "owns" this context, for proximity sorting */
    keyPos: number
  }

  const candidates: Candidate[] = []

  // Build a map from pointer → parent key position for gap detection
  const parentKeyPos = new Map<string, number>()
  for (const m of maps) {
    for (const pair of m.map.items) {
      if (!isPair(pair) || !isScalar(pair.key)) continue
      const keyRange = (pair.key as any).range as [number, number, number] | undefined
      if (!keyRange) continue
      const keyName = String((pair.key as any).value)
      const childPointer = m.pointer ? m.pointer + '/' + keyName : '/' + keyName
      parentKeyPos.set(childPointer, keyRange[0])
    }
  }

  for (const m of maps) {
    // Allow maps whose range starts after cursor IF the parent key is before cursor
    // (cursor is in the gap between `key:` and its map value on the next line)
    const keyPos = parentKeyPos.get(m.pointer)
    const effectiveKeyPos = keyPos ?? m.range[0]
    if (effectiveKeyPos > pos) continue

    if (m.contentIndent === cursorIndent) {
      candidates.push({ mapInfo: m, keyPos: effectiveKeyPos })
    }

    // Also check pairs in this map that might own the cursor's indent level
    // but don't have a map value yet
    if (cursorIndent > m.contentIndent) {
      let closestKey: string | null = null
      let closestKeyPos = -1

      for (const pair of m.map.items) {
        if (!isPair(pair) || !isScalar(pair.key)) continue
        const keyRange = (pair.key as any).range as [number, number, number] | undefined
        if (!keyRange || keyRange[0] > pos) continue

        const keyName = String((pair.key as any).value)

        const hasChildMap = maps.some(
          (child) =>
            child.pointer === (m.pointer ? m.pointer + '/' + keyName : '/' + keyName)
        )

        if (!hasChildMap && keyRange[0] > closestKeyPos) {
          closestKey = keyName
          closestKeyPos = keyRange[0]
        }
      }

      if (closestKey !== null) {
        const pointer = m.pointer ? m.pointer + '/' + closestKey : '/' + closestKey
        candidates.push({
          mapInfo: {
            map: { items: [] } as any,
            range: m.range,
            pointer,
            depth: m.depth + 1,
            indent: m.contentIndent,
            contentIndent: cursorIndent,
          },
          keyPos: closestKeyPos,
        })
      }
    }
  }

  if (candidates.length === 0) {
    return maps.find((m) => m.depth === 0) ?? null
  }

  // Sort by key position descending — closest to cursor wins
  candidates.sort((a, b) => b.keyPos - a.keyPos)
  return candidates[0].mapInfo
}

interface MapInfo {
  map: YAMLMap.Parsed
  range: [number, number, number]
  pointer: string
  depth: number
  indent: number
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
