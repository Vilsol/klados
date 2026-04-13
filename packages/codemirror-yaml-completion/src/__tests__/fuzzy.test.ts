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
