import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/svelte'
import { tick } from 'svelte'

vi.mock('../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js', () => ({
  CreateResource: vi.fn().mockResolvedValue({}),
  GetTemplates: vi.fn().mockResolvedValue([
    { gvr: 'core.v1.pods', name: 'Basic Pod', description: 'A basic pod', content: 'apiVersion: v1\nkind: Pod\n', source: 'builtin' },
  ]),
  GetAllTemplateGVRs: vi.fn().mockResolvedValue(['core.v1.pods', 'apps.v1.deployments']),
}))

vi.mock('$lib/stores/notification.svelte', () => ({
  notificationStore: { push: vi.fn() },
}))

import CreateResourceDialog from '$lib/components/CreateResourceDialog.svelte'

describe('CreateResourceDialog', () => {
  it('renders Resource Type and Template labels when open', async () => {
    render(CreateResourceDialog, { props: { open: true, ctxName: 'test-ctx' } })
    await tick()
    await waitFor(() => {
      expect(screen.getByText('Resource Type')).toBeTruthy()
    })
  })

  it('pre-fills GVR select when gvr prop provided', async () => {
    render(CreateResourceDialog, {
      props: { open: true, ctxName: 'test-ctx', gvr: 'core.v1.pods' },
    })
    await tick()
    await waitFor(() => {
      const selects = document.querySelectorAll('select')
      const gvrSelect = selects[0] as HTMLSelectElement
      expect(gvrSelect).toBeTruthy()
      expect(gvrSelect.value).toBe('core.v1.pods')
    })
  })

  it('shows Template label when a GVR is selected', async () => {
    render(CreateResourceDialog, {
      props: { open: true, ctxName: 'test-ctx', gvr: 'core.v1.pods' },
    })
    await tick()
    await waitFor(() => {
      expect(screen.getByText('Template')).toBeTruthy()
    })
  })
})
