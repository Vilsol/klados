import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import IngressPanel from '$lib/components/panels/IngressPanel.svelte'

vi.mock('@wailsio/runtime', () => ({
  Browser: { OpenURL: vi.fn() },
  Events: { On: vi.fn(() => vi.fn()), Off: vi.fn(), Emit: vi.fn() },
}))

const obj = {
  metadata: { name: 'my-ingress', namespace: 'default' },
  spec: {
    rules: [
      {
        host: 'app.example.com',
        http: {
          paths: [
            { path: '/', pathType: 'Prefix', backend: { service: { name: 'my-svc', port: { number: 80 } } } },
            { path: '/api', pathType: 'Exact', backend: { service: { name: 'api-svc', port: { number: 8080 } } } },
          ],
        },
      },
    ],
    tls: [
      { secretName: 'tls-secret', hosts: ['app.example.com'] },
    ],
  },
}

describe('IngressPanel', () => {
  it('renders rule host', () => {
    render(IngressPanel, { props: { obj } })
    // Host appears in both the rule header and TLS hosts list
    expect(screen.getAllByText('app.example.com').length).toBeGreaterThan(0)
  })

  it('renders paths', () => {
    render(IngressPanel, { props: { obj } })
    expect(screen.getByText('/')).toBeTruthy()
    expect(screen.getByText('/api')).toBeTruthy()
  })

  it('renders backend service names', () => {
    render(IngressPanel, { props: { obj } })
    expect(screen.getByText('my-svc:80')).toBeTruthy()
    expect(screen.getByText('api-svc:8080')).toBeTruthy()
  })

  it('renders TLS section', () => {
    render(IngressPanel, { props: { obj } })
    expect(screen.getByText('tls-secret')).toBeTruthy()
  })

  it('shows Open link for named hosts', () => {
    render(IngressPanel, { props: { obj } })
    expect(screen.getByText('Open ↗')).toBeTruthy()
  })

  it('calls Browser.OpenURL on open click', async () => {
    const { Browser } = await import('@wailsio/runtime')
    render(IngressPanel, { props: { obj } })
    await fireEvent.click(screen.getByText('Open ↗'))
    expect(Browser.OpenURL).toHaveBeenCalledWith('https://app.example.com')
  })

  it('shows wildcard host for rules without host', () => {
    const noHostObj = {
      metadata: {},
      spec: {
        rules: [{ http: { paths: [{ path: '/', pathType: 'Prefix', backend: {} }] } }],
        tls: [],
      },
    }
    render(IngressPanel, { props: { obj: noHostObj } })
    expect(screen.getByText('*')).toBeTruthy()
  })

  it('renders empty state with no rules', () => {
    render(IngressPanel, { props: { obj: { metadata: {}, spec: { rules: [], tls: [] } } } })
    expect(screen.getByText('No rules')).toBeTruthy()
  })
})
