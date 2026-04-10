import { describe, it, expect } from 'vitest'
import { getSuggestions, type Suggestion } from '$lib/search/autocomplete'

function makeItem(name: string, namespace: string, labels: Record<string, string> = {}, annotations: Record<string, string> = {}) {
  return {
    metadata: { name, namespace, labels, annotations },
  }
}

const items = [
  makeItem('nginx-proxy', 'default', { app: 'web', env: 'prod' }, { owner: 'team-a' }),
  makeItem('nginx-ingress', 'kube-system', { app: 'web', env: 'dev' }, { owner: 'team-b' }),
  makeItem('redis-master', 'default', { app: 'cache', env: 'prod' }, {}),
]

describe('getSuggestions', () => {
  it('suggests qualifiers when input is empty', () => {
    const result = getSuggestions('', 0, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['label:', 'annotation:', 'name:', 'namespace:'])
    )
  })

  it('suggests qualifiers matching partial text', () => {
    const result = getSuggestions('lab', 3, items)
    expect(result).toHaveLength(1)
    expect(result[0].value).toBe('label:')
  })

  it('suggests label keys after label:', () => {
    const result = getSuggestions('label:', 6, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['app', 'env'])
    )
  })

  it('suggests label keys after alias l:', () => {
    const result = getSuggestions('l:', 2, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['app', 'env'])
    )
  })

  it('filters label key suggestions by partial input', () => {
    const result = getSuggestions('label:ap', 8, items)
    expect(result).toHaveLength(1)
    expect(result[0].value).toBe('app')
  })

  it('suggests label values after key=', () => {
    const result = getSuggestions('label:app=', 10, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['web', 'cache'])
    )
  })

  it('filters label value suggestions by partial input', () => {
    const result = getSuggestions('label:app=w', 11, items)
    expect(result).toHaveLength(1)
    expect(result[0].value).toBe('web')
  })

  it('suggests annotation keys after annotation:', () => {
    const result = getSuggestions('annotation:', 11, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(['owner'])
  })

  it('suggests annotation keys after ann:', () => {
    const result = getSuggestions('ann:', 4, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(['owner'])
  })

  it('suggests namespace values after namespace:', () => {
    const result = getSuggestions('namespace:', 10, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['default', 'kube-system'])
    )
  })

  it('suggests namespace values after ns:', () => {
    const result = getSuggestions('ns:', 3, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['default', 'kube-system'])
    )
  })

  it('includes count in suggestions', () => {
    const result = getSuggestions('label:', 6, items)
    const appSuggestion = result.find((s: Suggestion) => s.value === 'app')
    expect(appSuggestion?.count).toBe(3)
  })

  it('returns no suggestions for bare text mid-word', () => {
    const result = getSuggestions('ngi', 3, items)
    expect(result).toHaveLength(0)
  })

  it('handles cursor in the middle of multi-term input', () => {
    const result = getSuggestions('label:app=web ns:', 17, items)
    expect(result.map((s: Suggestion) => s.value)).toEqual(
      expect.arrayContaining(['default', 'kube-system'])
    )
  })

  it('suggests qualifiers after negation prefix', () => {
    const result = getSuggestions('-lab', 4, items)
    expect(result).toHaveLength(1)
    expect(result[0].value).toBe('label:')
  })
})
