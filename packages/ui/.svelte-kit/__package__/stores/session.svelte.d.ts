export interface TabState {
    clusterContext: string;
    gvr: string;
    namespace: string;
    name: string;
    scrollPosition?: number;
}
declare class SessionStore {
    tabs: TabState[];
    activeTabIndex: number;
    sidebarCollapsed: boolean;
    openTab(tab: TabState): void;
    closeTab(index: number): void;
    setActiveTab(index: number): void;
    reorderTabs(from: number, to: number): void;
    saveScrollPosition(tabIndex: number, position: number): void;
    toggleSidebar(): void;
    restore(tabs: TabState[], activeTab: number, sidebarCollapsed: boolean): void;
}
export declare const sessionStore: SessionStore;
export {};
