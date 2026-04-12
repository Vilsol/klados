export type PanelKind = "logs" | "terminal" | "aggregate-logs" | "yaml";

export interface PanelTab {
  id: string;
  kind: PanelKind;
  resourceKind: string;
  resourceName: string;
  ctxName: string;
  gvr: string;
  namespace: string;
  name: string;
  obj: Record<string, unknown>;
  poppedOut: boolean;
}

const MIN_HEIGHT = 120;
const MAX_HEIGHT_RATIO = 0.8;

class BottomPanelStore {
  tabs = $state<PanelTab[]>([]);
  activeTabId = $state<string | null>(null);
  collapsed = $state(false);
  height = $state(300);

  get visibleTabs(): PanelTab[] {
    return this.tabs.filter((t) => !t.poppedOut);
  }

  get hasVisibleTabs(): boolean {
    return this.visibleTabs.length > 0;
  }

  addTab(tab: Omit<PanelTab, "id" | "poppedOut">): string {
    const id = crypto.randomUUID();
    this.tabs = [...this.tabs, {...tab, id, poppedOut: false}];
    this.activeTabId = id;
    this.collapsed = false;
    return id;
  }

  closeTab(id: string) {
    this.tabs = this.tabs.filter((t) => t.id !== id);
    if (this.activeTabId === id) {
      const visible = this.visibleTabs;
      this.activeTabId = visible.length > 0 ? visible.at(-1).id : null;
    }
  }

  setActive(id: string) {
    if (this.tabs.some((t) => t.id === id)) {
      this.activeTabId = id;
      this.collapsed = false;
    }
  }

  toggleCollapsed() {
    this.collapsed = !this.collapsed;
  }

  setHeight(h: number) {
    const maxH = typeof window === "undefined" ? 600 : window.innerHeight * MAX_HEIGHT_RATIO;
    this.height = Math.max(MIN_HEIGHT, Math.min(h, maxH));
  }

  popOut(id: string) {
    const tab = this.tabs.find((t) => t.id === id);
    if (!tab) {
      return;
    }
    tab.poppedOut = true;
    this.tabs = [...this.tabs];
    if (this.activeTabId === id) {
      const visible = this.visibleTabs;
      this.activeTabId = visible.length > 0 ? visible[0].id : null;
    }
  }

  popIn(id: string) {
    const tab = this.tabs.find((t) => t.id === id);
    if (!tab) {
      return;
    }
    tab.poppedOut = false;
    this.tabs = [...this.tabs];
    this.activeTabId = id;
    this.collapsed = false;
  }
}

export const bottomPanelStore = new BottomPanelStore();
