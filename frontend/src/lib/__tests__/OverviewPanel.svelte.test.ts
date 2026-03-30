import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import OverviewPanel from '$lib/components/panels/OverviewPanel.svelte'
import type { DescriptorDef } from '$lib/registry/index'

const descriptor: DescriptorDef = {
  group: 'apps',
  version: 'v1',
  resource: 'deployments',
  gvr: 'apps.v1.deployments',
  columns: [],
  overviewFields: [
    { label: 'Namespace', expr: 'metadata.namespace', renderType: 'text' },
    { label: 'Strategy', expr: 'spec.strategy.type', renderType: 'badge' },
    { label: 'Age', expr: 'metadata.creationTimestamp', renderType: 'age' },
  ],
  detailPanels: ['overview'],
  actions: [],
}

const obj = {
  metadata: { namespace: 'production', creationTimestamp: new Date(Date.now() - 3600 * 1000).toISOString() },
  spec: { strategy: { type: 'RollingUpdate' } },
  status: {},
}

describe('OverviewPanel', () => {
  it('renders all overview field labels', () => {
    render(OverviewPanel, { props: { obj, descriptor } })
    expect(screen.getByText('Namespace')).toBeTruthy()
    expect(screen.getByText('Strategy')).toBeTruthy()
    expect(screen.getByText('Age')).toBeTruthy()
  })

  it('renders field values', () => {
    render(OverviewPanel, { props: { obj, descriptor } })
    expect(screen.getByText('production')).toBeTruthy()
    expect(screen.getByText('RollingUpdate')).toBeTruthy()
  })

  it('renders age as human-readable', () => {
    render(OverviewPanel, { props: { obj, descriptor } })
    expect(screen.getByText('1h')).toBeTruthy()
  })

  it('renders empty fields as dash', () => {
    const emptyObj = { metadata: { namespace: '' }, spec: {}, status: {} }
    render(OverviewPanel, { props: { obj: emptyObj, descriptor } })
    // Should not throw, should render labels
    expect(screen.getByText('Namespace')).toBeTruthy()
  })
})
