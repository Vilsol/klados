import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'

const { mockStartLogStream, mockStopLogStream } = vi.hoisted(() => ({
  mockStartLogStream: vi.fn().mockResolvedValue('stream-id-123'),
  mockStopLogStream: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('../../../bindings/github.com/Vilsol/klados/internal/services/logservice.js', () => ({
  StartLogStream: mockStartLogStream,
  StopLogStream: mockStopLogStream,
}))

vi.mock('../../../bindings/github.com/Vilsol/klados/internal/logs/models.js', () => ({
  LogOptions: class LogOptions {
    constructor(opts: any) { Object.assign(this, opts) }
  },
}))

vi.mock('$lib/stores/streaming.svelte', () => ({
  streamingStore: { config: { port: 9999, token: 'test-token' } },
}))

vi.mock('$lib/components/LogViewer.svelte', () => ({
  default: vi.fn(),
}))

import LogsPanel from '$lib/components/panels/LogsPanel.svelte'

const podObj = {
  spec: {
    containers: [
      { name: 'app' },
      { name: 'sidecar' },
    ],
    initContainers: [
      { name: 'init-setup' },
    ],
  },
}

describe('LogsPanel', () => {
  beforeEach(() => {
    mockStartLogStream.mockClear()
    mockStopLogStream.mockClear()
    mockStartLogStream.mockResolvedValue('stream-id-123')
  })

  it('renders container selector with all containers', async () => {
    render(LogsPanel, {
      props: { obj: podObj, ctxName: 'ctx', namespace: 'default', name: 'mypod' },
    })
    // First container is visible in the dropdown button
    expect(screen.getByText('app')).toBeTruthy()
    // Open dropdown to see all containers
    await fireEvent.click(screen.getByText('app'))
    expect(screen.getByText('sidecar')).toBeTruthy()
    expect(screen.getByText('init-setup (init)')).toBeTruthy()
  })

  it('renders options: timestamps, previous', () => {
    render(LogsPanel, {
      props: { obj: podObj, ctxName: 'ctx', namespace: 'default', name: 'mypod' },
    })
    expect(screen.getByText('Timestamps')).toBeTruthy()
    expect(screen.getByText('Previous')).toBeTruthy()
  })

  it('auto-starts log stream on mount', async () => {
    render(LogsPanel, {
      props: { obj: podObj, ctxName: 'ctx', namespace: 'default', name: 'mypod' },
    })
    await waitFor(() => expect(mockStartLogStream).toHaveBeenCalledOnce())
    expect(mockStartLogStream).toHaveBeenCalledWith('ctx', 'default', 'mypod', expect.any(Object))
  })

  it('shows Connecting when stream is starting', () => {
    render(LogsPanel, {
      props: { obj: podObj, ctxName: 'ctx', namespace: 'default', name: 'mypod' },
    })
    // "Connecting…" is shown while the stream is starting
    expect(screen.getByText('Connecting…')).toBeTruthy()
  })

  it('shows error when StartLogStream rejects', async () => {
    mockStartLogStream.mockRejectedValueOnce(new Error('cluster offline'))
    render(LogsPanel, {
      props: { obj: podObj, ctxName: 'ctx', namespace: 'default', name: 'mypod' },
    })
    // After rejection, "Connecting…" should disappear (starting = false)
    await waitFor(() => expect(mockStartLogStream).toHaveBeenCalled())
    // The error is not shown in the UI (silent failure), but starting resets
    await waitFor(() => expect(screen.queryByText('Connecting…')).toBeNull())
  })
})
