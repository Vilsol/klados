import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, waitFor } from '@testing-library/svelte'

const mockSetData = vi.hoisted(() => vi.fn())
const mockDestroy = vi.hoisted(() => vi.fn())
const mockSetSeries = vi.hoisted(() => vi.fn())
const mockSetSize = vi.hoisted(() => vi.fn())
const mockSetScale = vi.hoisted(() => vi.fn())
const mockInstances = vi.hoisted(() => [] as any[])

vi.mock('uplot', () => {
  class UPlot {
    setData = mockSetData
    destroy = mockDestroy
    setSeries = mockSetSeries
    setSize = mockSetSize
    setScale = mockSetScale
    data: any
    series: any[]
    over = { addEventListener: vi.fn(), clientWidth: 400 }
    root = document.createElement('div')
    cursor = { idx: null as number | null, left: 0, top: 0 }
    scales = { x: { min: 0, max: 0 } }

    constructor(_opts: any, _data: any, el: HTMLElement) {
      this.data = _data
      this.series = _opts?.series ?? []
      const canvas = document.createElement('canvas')
      el?.appendChild(canvas)
      mockInstances.push(this)
    }
  }
  return { default: UPlot }
})

vi.mock('uplot/dist/uPlot.min.css', () => ({}))

import MetricsChart from '$lib/components/charts/MetricsChart.svelte'
import type { TimeSeries } from '$lib/components/charts/types'

function makeSeries(n = 2): TimeSeries[] {
  return Array.from({ length: n }, (_, i) => ({
    labels: { container: `c${i}` },
    points: [
      { t: 1000, v: 0.1 * (i + 1) },
      { t: 1015, v: 0.2 * (i + 1) },
    ],
  }))
}

beforeEach(() => {
  mockSetData.mockClear()
  mockDestroy.mockClear()
  mockInstances.length = 0
  vi.clearAllMocks()
})

describe('MetricsChart', () => {
  it('creates a canvas element on mount', async () => {
    const { container } = render(MetricsChart, {
      props: { title: 'CPU', unit: 'cores', series: makeSeries() },
    })
    await waitFor(() => expect(container.querySelector('canvas')).toBeTruthy())
  })

  it('calls setData on data change and does not call destroy', async () => {
    // Uses a Svelte 5 wrapper component to test reactive data updates
    // without @testing-library/svelte rerender's prop-object-replacement side-effects
    const { default: DataWrapper } = await import('./MetricsChartDataWrapper.svelte')
    const { container } = render(DataWrapper)

    await waitFor(() => expect(container.querySelector('canvas')).toBeTruthy())

    // Trigger the state update (wrapper's button updates series data)
    const btn = container.querySelector('[data-testid="update-data"]') as HTMLButtonElement
    btn?.click()

    await waitFor(() => expect(mockSetData).toHaveBeenCalled())
    // destroy should NOT have been called from a data update
    expect(mockDestroy).not.toHaveBeenCalled()
  })

  it('calls destroy on unmount', async () => {
    const { unmount } = render(MetricsChart, {
      props: { title: 'CPU', unit: 'cores', series: makeSeries() },
    })
    await waitFor(() => expect(mockInstances.length).toBeGreaterThan(0))
    unmount()
    expect(mockDestroy).toHaveBeenCalled()
  })

  it('shows loading skeleton when loading is true', () => {
    const { container } = render(MetricsChart, {
      props: { title: 'CPU', unit: 'cores', series: [], loading: true },
    })
    expect(container.querySelector('.animate-pulse')).toBeTruthy()
  })
})
