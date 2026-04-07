import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/svelte'
import Badge from '../Badge.svelte'

describe('Badge', () => {
  it('renders default variant', () => {
    const { container } = render(Badge)
    const span = container.querySelector('span')!
    expect(span.className).toContain('bg-surface')
  })

  it('renders success variant', () => {
    const { container } = render(Badge, { props: { variant: 'success' } })
    expect(container.querySelector('span')!.className).toContain('bg-accent')
  })

  it('renders warning variant', () => {
    const { container } = render(Badge, { props: { variant: 'warning' } })
    expect(container.querySelector('span')!.className).toContain('bg-surface-hover')
  })

  it('renders destructive variant', () => {
    const { container } = render(Badge, { props: { variant: 'destructive' } })
    expect(container.querySelector('span')!.className).toContain('bg-destructive')
  })
})
