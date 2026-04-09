# Bottom Panel for Interactive Views

## Context

Klados currently renders all resource detail panels (Logs, Terminal, YAML Editor, etc.) as exclusive tabs within `ResourceDetail.svelte`. This means only one panel is visible at a time, and only for the currently viewed resource. Users need to view multiple log streams, terminals, or editors simultaneously — across different resources or the same resource.

The solution is a persistent bottom panel in the main layout that hosts interactive panels as tabs, independent of navigation. Tabs can be popped out into separate OS windows and popped back in.

## Decisions

**Bottom panel lives in Layout.svelte, sharing space with the content area**
The sidebar remains full height. The bottom panel sits below the main content area (ResourceList, ClusterOverview, etc.) with a draggable resize handle. The detail drawer continues to overlay everything in the right area as it does today.

**Only four panel types are interactive / bottom-panel eligible**
Logs, Terminal, Aggregate Logs, and YAML Editor. All other panels (Overview, Events, Labels, Containers, etc.) remain as tabs in the ResourceDetail view.

**"Open in Panel" action creates a tab in the bottom panel**
Each interactive panel in ResourceDetail gets a button (e.g., an icon) that moves/creates it as a tab in the bottom panel. The panel tab persists across page navigation but not app restarts.

**Tab identity: `{icon} {kind}: {name}`**
Icon represents the panel type (terminal, log, editor). Kind is the resource kind. Name is the resource name. Kept narrow so many tabs fit. Tabs scroll horizontally when they overflow.

**Single active tab at a time in the bottom panel**
No side-by-side split within the panel. Tab switching only.

**Pop-out / pop-in via button, not drag**
Each tab gets a "pop out" icon button that opens it in a native OS window (Wails multi-window). The window gets a "pop back in" button to return to the bottom panel. Closing the OS window destroys the tab entirely (stops log stream / terminal session).

**Collapsible with toggle**
A `▼`/`▲` icon on the far right of the tab bar collapses/expands the bottom panel. Panel is hidden when no tabs exist.

**Resizable**
A drag handle between the content area and the bottom panel. Minimum height enforced. Height is ephemeral (not persisted across restarts).

## Rejected Alternatives

**Tiling / flexible drag-and-drop layout**
Too much interaction cost for "I want to see 2-3 things." Users shouldn't have to carefully arrange panels.

**Detach-to-window only (no bottom panel)**
Clutters the desktop with windows for single-monitor users. The bottom panel is the primary experience; pop-out is the escape hatch.

**Side-by-side split within bottom panel**
Adds complexity without enough benefit for v1. Can be added later if needed.

**Persistent across restarts**
Would require serializing/restoring active log streams and terminal sessions, which is complex and fragile. Navigation persistence is sufficient.

## Priorities & Tradeoffs

Optimizing for low interaction cost and simplicity. Users should go from "I want to see Pod A logs while I work on Pod B" to seeing both in one click. Sacrificing flexibility (no arbitrary tiling, no side-by-side) in exchange for a predictable, consistent layout.

## Potential Gotchas

- **Terminal/log lifecycle on pop-out**: When popping out to a native window, the xterm.js or log stream instance needs to be transferred (or recreated) without losing state. Svelte component destruction + recreation in a new window context could drop buffered content. May need to hold the backing data (log buffer, terminal state) in a store separate from the component.
- **Wails multi-window**: Wails v3 supports multi-window but each window is a separate webview. State sharing between windows needs to go through Go (Wails events or shared service calls), not JS globals. The bottom panel store must be accessible from both the main window and pop-out windows.
- **Resize handle vs. detail drawer**: When the detail drawer is open (overlaying the content area), the resize handle and bottom panel should still be functional underneath / alongside it. Need to clarify z-index layering.
- **xterm.js resize**: Terminal panels need to respond to bottom panel resize events (fit addon). Same for popping in/out — the terminal dimensions change.

## Implementation Details

### Bottom Panel Store

```typescript
// frontend/src/lib/stores/bottom-panel.svelte.ts

type PanelType = 'logs' | 'terminal' | 'aggregate-logs' | 'yaml'

interface PanelTab {
  id: string                    // crypto.randomUUID()
  type: PanelType
  icon: string                  // lucide icon name
  resourceKind: string          // e.g. "Pod"
  resourceName: string          // e.g. "nginx-abc123"
  ctxName: string
  gvr: string
  namespace: string
  name: string
  obj: Record<string, any>      // resource object snapshot
  poppedOut: boolean            // true if in a separate OS window
}

class BottomPanelStore {
  tabs = $state<PanelTab[]>([])
  activeTabId = $state<string | null>(null)
  collapsed = $state(false)
  height = $state(300)          // pixels, with min enforced in UI

  openTab(tab: Omit<PanelTab, 'id' | 'poppedOut'>): string
  closeTab(id: string): void
  setActiveTab(id: string): void
  toggleCollapsed(): void
  popOut(id: string): void      // marks poppedOut = true, triggers Wails window creation
  popIn(id: string): void       // marks poppedOut = false, returns to bottom panel
}

export const bottomPanelStore = new BottomPanelStore()
```

### Layout structure change

```svelte
<!-- Layout.svelte — new structure -->
<div class="flex flex-col h-full">
  <Header />
  <div class="flex flex-1 overflow-hidden relative">
    <Sidebar />
    <div class="flex flex-col flex-1 overflow-hidden">
      <main id="main-content" class="flex flex-col flex-1 overflow-hidden" tabindex="-1">
        <TabBar />
        <div class="flex-1 overflow-hidden">
          {@render children()}
        </div>
      </main>
      {#if bottomPanelStore.tabs.some(t => !t.poppedOut)}
        <ResizeHandle />
        <BottomPanel />
      {/if}
    </div>
  </div>
  <!-- status bar if plugins -->
</div>
```

### BottomPanel component

```svelte
<!-- frontend/src/lib/components/BottomPanel.svelte -->
<!-- Tab bar with: scrollable tabs, collapse toggle on far right -->
<!-- Each tab shows: {icon} {kind}: {name}, close button -->
<!-- Each tab has a "pop out" icon button -->
<!-- Active tab's content rendered below the tab bar -->
<!-- Only renders tabs where poppedOut === false -->
```

### ResourceDetail integration

For each of the four interactive panels, add an "Open in Bottom Panel" icon button in the panel tab bar area. Clicking it:
1. Calls `bottomPanelStore.openTab(...)` with the current resource context
2. Optionally switches the ResourceDetail to a different panel (e.g., back to Overview)

### Pop-out window

Each popped-out window:
- Receives the `PanelTab` data via Wails event or URL params
- Renders only the single panel component (no sidebar, no header, no tab bar)
- Has a "Pop back in" button in a minimal title bar
- On window close: calls `bottomPanelStore.closeTab(id)` via Wails event back to main window

### New files

```
frontend/src/lib/stores/bottom-panel.svelte.ts
frontend/src/lib/components/BottomPanel.svelte
frontend/src/lib/components/ResizeHandle.svelte
frontend/src/lib/components/PanelWindow.svelte  (pop-out window root)
```

### Modified files

```
frontend/src/lib/components/Layout.svelte       — add BottomPanel + ResizeHandle
frontend/src/lib/components/ResourceDetail.svelte — add "Open in Panel" buttons
frontend/src/lib/stores/session.svelte.ts        — no change (bottom panel is separate store)
```

## Definition of Done

- [ ] Bottom panel renders in Layout below content area, with tab bar and resize handle
- [ ] Clicking "Open in Panel" on Logs/Terminal/AggregateLogs/YAML in ResourceDetail creates a tab in the bottom panel
- [ ] Bottom panel tabs persist across page navigation (navigating to a different resource doesn't close them)
- [ ] Bottom panel tabs are destroyed on app restart (not serialized to session)
- [ ] Collapse/expand toggle works
- [ ] Resize handle allows dragging the panel height with a minimum enforced
- [ ] Tab bar scrolls horizontally when many tabs are open
- [ ] Pop-out button opens the panel in a separate OS window via Wails multi-window
- [ ] Pop-in button in the separate window returns the tab to the bottom panel
- [ ] Closing the OS window destroys the tab and cleans up resources (log stream, terminal session)
- [ ] Terminal panels respond to resize events (xterm.js fit)
- [ ] Log streams continue running in bottom panel tabs while user navigates elsewhere
