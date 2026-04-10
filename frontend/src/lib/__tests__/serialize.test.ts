import { describe, it, expect } from 'vitest'
import { savedFilterToQuery, queryToSavedFilter } from '$lib/search/serialize'
import type { SavedFilter } from '$lib/stores/preferences.svelte'

describe('savedFilterToQuery', () => {
  it('converts labels to label: terms', () => {
    const filter: SavedFilter = { name: 'test', labels: { app: 'web', env: 'prod' } }
    const query = savedFilterToQuery(filter)
    expect(query).toContain('l:app=web')
    expect(query).toContain('l:env=prod')
  })

  it('converts annotations to ann: terms', () => {
    const filter: SavedFilter = { name: 'test', annotations: { owner: 'team-a' } }
    const query = savedFilterToQuery(filter)
    expect(query).toContain('ann:owner=team-a')
  })

  it('appends search text as-is', () => {
    const filter: SavedFilter = { name: 'test', search: 'nginx' }
    const query = savedFilterToQuery(filter)
    expect(query).toContain('nginx')
  })

  it('combines all fields', () => {
    const filter: SavedFilter = {
      name: 'test',
      labels: { app: 'web' },
      annotations: { owner: 'team-a' },
      search: 'nginx',
    }
    const query = savedFilterToQuery(filter)
    expect(query).toContain('l:app=web')
    expect(query).toContain('ann:owner=team-a')
    expect(query).toContain('nginx')
  })

  it('returns empty string for empty filter', () => {
    const filter: SavedFilter = { name: 'test' }
    expect(savedFilterToQuery(filter)).toBe('')
  })
})

describe('queryToSavedFilter', () => {
  it('extracts labels from label: terms', () => {
    const filter = queryToSavedFilter('l:app=web l:env=prod')
    expect(filter.labels).toEqual({ app: 'web', env: 'prod' })
  })

  it('extracts annotations from ann: terms', () => {
    const filter = queryToSavedFilter('ann:owner=team-a')
    expect(filter.annotations).toEqual({ owner: 'team-a' })
  })

  it('puts text and non-model terms into search', () => {
    const filter = queryToSavedFilter('l:app=web nginx name:proxy')
    expect(filter.labels).toEqual({ app: 'web' })
    expect(filter.search).toBe('nginx name:proxy')
  })

  it('returns empty fields for empty input', () => {
    const filter = queryToSavedFilter('')
    expect(filter.labels).toBeUndefined()
    expect(filter.annotations).toBeUndefined()
    expect(filter.search).toBeUndefined()
  })
})
