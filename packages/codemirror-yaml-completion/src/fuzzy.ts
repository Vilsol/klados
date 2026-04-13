/**
 * Fuzzy match an input string against a candidate.
 * Returns a boost score (higher = better match), or null for no match.
 *
 * - Exact prefix: 2
 * - Case-insensitive prefix: 1
 * - Substring (case-insensitive): 0
 * - Subsequence (case-insensitive): -1
 * - No match: null
 * - Empty input: 0 (show all)
 */
export function fuzzyMatch(input: string, candidate: string): number | null {
  if (input.length === 0) return 0

  if (candidate.startsWith(input)) return 2

  const lowerInput = input.toLowerCase()
  const lowerCandidate = candidate.toLowerCase()

  if (lowerCandidate.startsWith(lowerInput)) return 1

  if (lowerCandidate.includes(lowerInput)) return 0

  // Subsequence: each character of input appears in order in candidate
  let ci = 0
  for (let i = 0; i < lowerInput.length; i++) {
    const idx = lowerCandidate.indexOf(lowerInput[i], ci)
    if (idx === -1) return null
    ci = idx + 1
  }
  return -1
}
