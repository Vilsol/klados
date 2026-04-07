import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/svelte'
import { TabBar, sessionStore } from '@klados/ui'

describe('TabBar', () => {
  beforeEach(() => {
    sessionStore.tabs = []
    sessionStore.activeTabIndex = 0
  })

  it('renders nothing when no tabs', () => {
    const { container } = render(TabBar)
    expect(container.querySelector('[role="tab"]')).toBeNull()
  })

  it('renders tabs', () => {
    sessionStore.tabs = [
      { clusterContext: 'ctx1', gvr: 'v1/pods', namespace: 'default', name: 'my-pod' },
      { clusterContext: 'ctx1', gvr: 'v1/services', namespace: 'default', name: 'my-svc' },
    ]
    sessionStore.activeTabIndex = 0

    render(TabBar)

    expect(screen.getByText('my-pod')).toBeTruthy()
    expect(screen.getByText('my-svc')).toBeTruthy()
  })

  it('clicking tab switches active index', async () => {
    sessionStore.tabs = [
      { clusterContext: 'ctx1', gvr: 'v1/pods', namespace: 'default', name: 'tab-a' },
      { clusterContext: 'ctx1', gvr: 'v1/pods', namespace: 'default', name: 'tab-b' },
    ]
    sessionStore.activeTabIndex = 0

    render(TabBar)

    const tabB = screen.getByText('tab-b').closest('[role="tab"]')!
    await fireEvent.click(tabB)

    expect(sessionStore.activeTabIndex).toBe(1)
  })

  it('close button removes tab', async () => {
    sessionStore.tabs = [
      { clusterContext: 'ctx1', gvr: 'v1/pods', namespace: 'default', name: 'only-tab' },
    ]

    render(TabBar)

    const closeBtn = screen.getByText('only-tab').parentElement!.querySelector('button')!
    await fireEvent.click(closeBtn)

    expect(sessionStore.tabs).toHaveLength(0)
  })
})
