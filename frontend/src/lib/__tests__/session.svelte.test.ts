import { describe, it, expect } from 'vitest'
import { sessionStore } from '$lib/stores/session.svelte'

describe('sessionStore', () => {
  it('opens a tab and sets it active', () => {
    sessionStore.tabs = []
    sessionStore.activeTabIndex = 0

    sessionStore.openTab({
      clusterContext: 'ctx1',
      gvr: 'v1/pods',
      namespace: 'default',
      name: 'my-pod',
    })

    expect(sessionStore.tabs).toHaveLength(1)
    expect(sessionStore.activeTabIndex).toBe(0)
    expect(sessionStore.tabs[0].name).toBe('my-pod')
  })

  it('does not duplicate tabs', () => {
    sessionStore.tabs = []

    const tab = {
      clusterContext: 'ctx1',
      gvr: 'v1/pods',
      namespace: 'default',
      name: 'my-pod',
    }
    sessionStore.openTab(tab)
    sessionStore.openTab(tab)

    expect(sessionStore.tabs).toHaveLength(1)
  })

  it('closes a tab', () => {
    sessionStore.tabs = []
    sessionStore.openTab({ clusterContext: 'a', gvr: 'v1/pods', namespace: 'default', name: 'p1' })
    sessionStore.openTab({ clusterContext: 'a', gvr: 'v1/pods', namespace: 'default', name: 'p2' })

    expect(sessionStore.tabs).toHaveLength(2)

    sessionStore.closeTab(0)
    expect(sessionStore.tabs).toHaveLength(1)
    expect(sessionStore.tabs[0].name).toBe('p2')
  })

  it('switches active tab', () => {
    sessionStore.tabs = []
    sessionStore.openTab({ clusterContext: 'a', gvr: 'v1/pods', namespace: 'default', name: 'p1' })
    sessionStore.openTab({ clusterContext: 'a', gvr: 'v1/pods', namespace: 'default', name: 'p2' })

    sessionStore.setActiveTab(0)
    expect(sessionStore.activeTabIndex).toBe(0)
  })

  it('toggles sidebar', () => {
    sessionStore.sidebarCollapsed = false
    sessionStore.toggleSidebar()
    expect(sessionStore.sidebarCollapsed).toBe(true)
    sessionStore.toggleSidebar()
    expect(sessionStore.sidebarCollapsed).toBe(false)
  })

  it('reorderTabs moves a tab and updates activeTabIndex', () => {
    sessionStore.tabs = []
    sessionStore.openTab({ clusterContext: 'a', gvr: 'v1/pods', namespace: 'default', name: 'p1' })
    sessionStore.openTab({ clusterContext: 'a', gvr: 'v1/pods', namespace: 'default', name: 'p2' })
    sessionStore.openTab({ clusterContext: 'a', gvr: 'v1/pods', namespace: 'default', name: 'p3' })
    sessionStore.activeTabIndex = 0

    // Move p1 (index 0) to index 2
    sessionStore.reorderTabs(0, 2)
    expect(sessionStore.tabs[0].name).toBe('p2')
    expect(sessionStore.tabs[1].name).toBe('p3')
    expect(sessionStore.tabs[2].name).toBe('p1')
    expect(sessionStore.activeTabIndex).toBe(2)
  })

  it('reorderTabs is a no-op for same index', () => {
    sessionStore.tabs = []
    sessionStore.openTab({ clusterContext: 'a', gvr: 'v1/pods', namespace: 'default', name: 'p1' })
    const before = sessionStore.tabs.slice()
    sessionStore.reorderTabs(0, 0)
    expect(sessionStore.tabs[0].name).toBe(before[0].name)
  })

  it('saveScrollPosition stores scroll position on tab', () => {
    sessionStore.tabs = []
    sessionStore.openTab({ clusterContext: 'a', gvr: 'v1/pods', namespace: 'default', name: 'p1' })
    sessionStore.saveScrollPosition(0, 250)
    expect(sessionStore.tabs[0].scrollPosition).toBe(250)
  })
})
