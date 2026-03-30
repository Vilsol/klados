import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/svelte'

const { mockOpenExecSession, mockCloseExecSession } = vi.hoisted(() => ({
  mockOpenExecSession: vi.fn().mockResolvedValue('session-id-abc'),
  mockCloseExecSession: vi.fn().mockResolvedValue(undefined),
}))

vi.mock('../../../bindings/github.com/Vilsol/klados/internal/services/execservice.js', () => ({
  OpenExecSession: mockOpenExecSession,
  CloseExecSession: mockCloseExecSession,
}))

vi.mock('$lib/stores/streaming.svelte', () => ({
  streamingStore: { config: { port: 9999, token: 'test-token' } },
}))

vi.mock('$lib/components/Terminal.svelte', () => ({
  default: vi.fn(),
}))

import TerminalPanel from '$lib/components/panels/TerminalPanel.svelte'

const podObj = {
  spec: {
    containers: [
      { name: 'app' },
      { name: 'worker' },
    ],
    initContainers: [],
  },
}

describe('TerminalPanel', () => {
  beforeEach(() => {
    mockOpenExecSession.mockClear()
    mockCloseExecSession.mockClear()
    mockOpenExecSession.mockResolvedValue('session-id-abc')
  })

  it('renders container selector', async () => {
    render(TerminalPanel, {
      props: { obj: podObj, ctxName: 'ctx', namespace: 'default', name: 'mypod' },
    })
    // Open the dropdown to see all containers
    const dropdownBtn = screen.getByText('app')
    await fireEvent.click(dropdownBtn)
    expect(screen.getByText('worker')).toBeTruthy()
  })

  it('renders shell selector buttons', () => {
    render(TerminalPanel, {
      props: { obj: podObj, ctxName: 'ctx', namespace: 'default', name: 'mypod' },
    })
    expect(screen.getByText('bash')).toBeTruthy()
    expect(screen.getByText('sh')).toBeTruthy()
    expect(screen.getByText('zsh')).toBeTruthy()
  })

  it('renders Connect button', () => {
    render(TerminalPanel, {
      props: { obj: podObj, ctxName: 'ctx', namespace: 'default', name: 'mypod' },
    })
    expect(screen.getByText('Connect')).toBeTruthy()
  })

  async function getConnectBtn() {
    return waitFor(() => {
      const b = screen.getByText('Connect') as HTMLButtonElement
      expect(b.disabled).toBe(false)
      return b
    })
  }

  it('calls OpenExecSession with correct args on Connect click', async () => {
    render(TerminalPanel, {
      props: { obj: podObj, ctxName: 'ctx', namespace: 'default', name: 'mypod' },
    })
    await fireEvent.click(await getConnectBtn())
    await waitFor(() => expect(mockOpenExecSession).toHaveBeenCalledOnce())
    expect(mockOpenExecSession).toHaveBeenCalledWith('ctx', 'default', 'mypod', 'app', 'bash')
  })

  it('shows error when OpenExecSession rejects', async () => {
    mockOpenExecSession.mockRejectedValueOnce(new Error('pod not found'))
    render(TerminalPanel, {
      props: { obj: podObj, ctxName: 'ctx', namespace: 'default', name: 'mypod' },
    })
    await fireEvent.click(await getConnectBtn())
    await waitFor(() => expect(screen.getByText('pod not found')).toBeTruthy())
  })

  it('changes shell when clicking different shell button', async () => {
    render(TerminalPanel, {
      props: { obj: podObj, ctxName: 'ctx', namespace: 'default', name: 'mypod' },
    })
    await fireEvent.click(screen.getByText('zsh'))
    await fireEvent.click(await getConnectBtn())
    await waitFor(() => expect(mockOpenExecSession).toHaveBeenCalledOnce())
    expect(mockOpenExecSession).toHaveBeenCalledWith('ctx', 'default', 'mypod', 'app', 'zsh')
  })
})
