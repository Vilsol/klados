export const SIDEBAR_MIN_WIDTH = 180;
export const SIDEBAR_MAX_WIDTH = 480;
export const SIDEBAR_DEFAULT_WIDTH = 240;

function clampSidebarWidth(n: number): number {
  if (!Number.isFinite(n)) return SIDEBAR_DEFAULT_WIDTH;
  return Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, Math.round(n)));
}

export interface TabState {
  clusterContext: string;
  gvr: string;
  namespace: string;
  name: string;
  scrollPosition?: number;
}

class SessionStore {
  tabs = $state<TabState[]>([]);
  activeTabIndex = $state(0);
  sidebarCollapsed = $state(false);
  terminalFontSize = $state(13);
  sidebarWidth = $state(SIDEBAR_DEFAULT_WIDTH);

  openTab(tab: TabState) {
    const existing = this.tabs.findIndex(
      (t) => t.clusterContext === tab.clusterContext && t.gvr === tab.gvr && t.name === tab.name && t.namespace === tab.namespace,
    );
    if (existing >= 0) {
      this.activeTabIndex = existing;
      return;
    }
    this.tabs = [...this.tabs, tab];
    this.activeTabIndex = this.tabs.length - 1;
  }

  closeTab(index: number) {
    this.tabs = this.tabs.filter((_, i) => i !== index);
    if (this.activeTabIndex >= this.tabs.length) {
      this.activeTabIndex = Math.max(0, this.tabs.length - 1);
    }
  }

  setActiveTab(index: number) {
    if (index >= 0 && index < this.tabs.length) {
      this.activeTabIndex = index;
    }
  }

  reorderTabs(from: number, to: number) {
    if (from === to) {
      return;
    }
    if (from < 0 || from >= this.tabs.length || to < 0 || to >= this.tabs.length) {
      return;
    }
    const next = [...this.tabs];
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    // Keep active tab pointing to the same tab after reorder
    if (this.activeTabIndex === from) {
      this.activeTabIndex = to;
    } else if (from < to && this.activeTabIndex > from && this.activeTabIndex <= to) {
      this.activeTabIndex--;
    } else if (from > to && this.activeTabIndex >= to && this.activeTabIndex < from) {
      this.activeTabIndex++;
    }
    this.tabs = next;
  }

  saveScrollPosition(tabIndex: number, position: number) {
    if (tabIndex < 0 || tabIndex >= this.tabs.length) {
      return;
    }
    this.tabs[tabIndex] = {...this.tabs[tabIndex], scrollPosition: position};
  }

  toggleSidebar() {
    this.sidebarCollapsed = !this.sidebarCollapsed;
  }

  setSidebarWidth(width: number) {
    this.sidebarWidth = clampSidebarWidth(width);
  }

  resetSidebarWidth() {
    this.sidebarWidth = SIDEBAR_DEFAULT_WIDTH;
  }

  restore(
    tabs: TabState[],
    activeTab: number,
    sidebarCollapsed: boolean,
    terminalFontSize?: number,
    sidebarWidth?: number,
  ) {
    this.tabs = tabs;
    this.activeTabIndex = activeTab < tabs.length ? activeTab : 0;
    this.sidebarCollapsed = sidebarCollapsed;
    this.terminalFontSize = terminalFontSize ?? 13;
    this.sidebarWidth = clampSidebarWidth(sidebarWidth ?? SIDEBAR_DEFAULT_WIDTH);
  }
}

export const sessionStore = new SessionStore();
