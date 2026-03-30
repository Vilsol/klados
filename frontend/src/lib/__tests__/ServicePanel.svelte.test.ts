import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/svelte'

const { mockGetResource } = vi.hoisted(() => ({
  mockGetResource: vi.fn(),
}))

vi.mock(
  '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js',
  () => ({ GetResource: mockGetResource }),
)

import ServicePanel from '$lib/components/panels/ServicePanel.svelte'

const obj = {
  metadata: { name: 'my-service', namespace: 'default' },
  spec: {
    type: 'ClusterIP',
    selector: { app: 'myapp', tier: 'frontend' },
    ports: [
      { name: 'http', port: 80, protocol: 'TCP', targetPort: 8080 },
    ],
  },
}

const endpointsObj = {
  subsets: [
    {
      addresses: [
        { ip: '10.0.0.1', targetRef: { name: 'myapp-abc' } },
        { ip: '10.0.0.2', targetRef: { name: 'myapp-def' } },
      ],
    },
  ],
}

describe('ServicePanel', () => {
  beforeEach(() => {
    mockGetResource.mockResolvedValue(endpointsObj)
  })

  it('renders selector labels', () => {
    render(ServicePanel, { props: { obj, ctxName: 'ctx1' } })
    expect(screen.getByText('app=myapp')).toBeTruthy()
    expect(screen.getByText('tier=frontend')).toBeTruthy()
  })

  it('renders port table', () => {
    render(ServicePanel, { props: { obj, ctxName: 'ctx1' } })
    expect(screen.getByText('http')).toBeTruthy()
    expect(screen.getByText('80')).toBeTruthy()
  })

  it('shows loading state while fetching endpoints', () => {
    mockGetResource.mockReturnValue(new Promise(() => {})) // never resolves
    render(ServicePanel, { props: { obj, ctxName: 'ctx1' } })
    expect(screen.getByText('Loading…')).toBeTruthy()
  })

  it('shows backing pods after endpoints load', async () => {
    render(ServicePanel, { props: { obj, ctxName: 'ctx1' } })
    await waitFor(() => {
      expect(screen.getByText('myapp-abc')).toBeTruthy()
      expect(screen.getByText('myapp-def')).toBeTruthy()
    })
  })

  it('shows IP addresses for endpoints', async () => {
    render(ServicePanel, { props: { obj, ctxName: 'ctx1' } })
    await waitFor(() => {
      expect(screen.getByText('10.0.0.1')).toBeTruthy()
    })
  })

  it('calls GetResource with correct endpoint GVR', async () => {
    render(ServicePanel, { props: { obj, ctxName: 'ctx1' } })
    await waitFor(() => {
      expect(mockGetResource).toHaveBeenCalledWith('ctx1', 'core.v1.endpoints', 'default', 'my-service')
    })
  })

  it('shows no endpoints message when none found', async () => {
    mockGetResource.mockResolvedValue({ subsets: [] })
    render(ServicePanel, { props: { obj, ctxName: 'ctx1' } })
    await waitFor(() => {
      expect(screen.getByText('No endpoints')).toBeTruthy()
    })
  })

  it('shows no selector message when empty', () => {
    const noSelectorObj = { metadata: { name: 'svc', namespace: 'ns' }, spec: { selector: {} } }
    render(ServicePanel, { props: { obj: noSelectorObj, ctxName: 'ctx1' } })
    expect(screen.getByText('No selector')).toBeTruthy()
  })
})
