import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/svelte'
import Input from '../Input.svelte'

describe('Input', () => {
  it('renders an input element', () => {
    const { container } = render(Input)
    expect(container.querySelector('input')).toBeTruthy()
  })

  it('renders with label', () => {
    const { getByText } = render(Input, { props: { label: 'Username' } })
    expect(getByText('Username')).toBeTruthy()
  })

  it('renders with error message', () => {
    const { getByText, container } = render(Input, { props: { error: 'Required' } })
    expect(getByText('Required')).toBeTruthy()
    expect(container.querySelector('input')!.className).toContain('border-destructive')
  })

  it('is disabled when disabled prop is true', () => {
    const { container } = render(Input, { props: { disabled: true } })
    expect(container.querySelector('input')!.disabled).toBe(true)
  })

  it('renders with placeholder', () => {
    const { container } = render(Input, { props: { placeholder: 'Enter text' } })
    expect(container.querySelector('input')!.placeholder).toBe('Enter text')
  })
})
