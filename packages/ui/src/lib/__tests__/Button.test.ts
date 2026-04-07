import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import Button from '../Button.svelte'

describe('Button', () => {
  it('renders with default variant', () => {
    render(Button, { props: { children: () => 'Click me' } })
    expect(document.querySelector('button')).toBeTruthy()
  })

  it('renders primary variant', () => {
    render(Button, { props: { variant: 'primary' } })
    const btn = document.querySelector('button')!
    expect(btn.className).toContain('bg-accent')
  })

  it('renders destructive variant', () => {
    render(Button, { props: { variant: 'destructive' } })
    const btn = document.querySelector('button')!
    expect(btn.className).toContain('bg-destructive')
  })

  it('renders ghost variant', () => {
    render(Button, { props: { variant: 'ghost' } })
    const btn = document.querySelector('button')!
    expect(btn.className).toContain('text-fg')
  })

  it('renders outline variant', () => {
    render(Button, { props: { variant: 'outline' } })
    const btn = document.querySelector('button')!
    expect(btn.className).toContain('border-border')
  })

  it('is disabled when disabled prop is true', () => {
    render(Button, { props: { disabled: true } })
    expect(document.querySelector('button')!.disabled).toBe(true)
  })
})
