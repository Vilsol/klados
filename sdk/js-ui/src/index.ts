/**
 * @klados/plugin-ui — shared UI primitives for klados plugins.
 *
 * This package is EXTERNALIZED by the klados vite build helper; the host
 * injects the actual runtime at load time. Do not bundle it into your plugin.
 *
 * Re-exports: Bits UI components, Lucide icons, and klados CSS token helpers.
 */

// Bits UI primitives
export * from 'bits-ui'

// Lucide icon components
export * from 'lucide-svelte'

// CSS custom-property tokens used by the klados theme.
// Consume these in Svelte style blocks: color: var(--color-fg)
export const tokens = {
  // Semantic colours
  bg: 'var(--color-bg)',
  fg: 'var(--color-fg)',
  muted: 'var(--color-muted)',
  border: 'var(--color-border)',
  accent: 'var(--color-accent)',
  surface: 'var(--color-surface)',
  surfaceHover: 'var(--color-surface-hover)',
  destructive: 'var(--color-destructive)',
} as const

export type TokenKey = keyof typeof tokens
