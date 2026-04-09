# Streaming Enhancements

## Context

The streaming subsystem (port-forwarding, logs, exec) has several gaps: port-forwards are ephemeral and lost on restart, there's no centralized management view, log viewing is single-pod only, and terminal clear/reset doesn't flush scrollback. This spec covers five enhancements that share code proximity in the streaming layer.

Hard constraints:
- Wails v3 alpha.74, Svelte 5 runes, Tailwind v4
- LogStreamer already supports multiple simultaneous WebSocket connections (unique stream IDs, independent buffers, deadlock-aware mutex)
- Port-forward Manager emits per-forward (`portforward:{ctx}:{id}`) and aggregate (`portforward:{ctx}:updated`) events
- xterm.js supports CSI 3J (clear scrollback) and RIS (full reset) natively

## Decisions

**Port-forward persistence in config, not session**
Config (`config.json`) stores intentional user preferences; session stores ephemeral UI state. Saved port-forwards are user intent ("I want this forward available"), so they belong in config. Each saved forward has an `enabled` boolean — when `false`, the forward appears in the management page but does not auto-reconnect.

**Port-forward management page reuses ResourceList**
The management page at `/c/:ctx/port-forwards` reuses the existing `ResourceList` component with a virtual descriptor (not backed by a real k8s GVR). This gives column sorting, virtual scrolling, and consistent UX for free. New forwards are created via the existing port-forward dialog.

**Frontend fan-out for multi-pod log aggregation**
The frontend opens N WebSocket connections (one per pod/container) and merges them client-side. No backend multiplexing abstraction needed. Each line gets a color-coded pod-name prefix that can be toggled on/off. Ordering is arrival-order (not timestamp-sorted) — strict cross-pod ordering would require buffering and add latency with no practical benefit for tail-style streaming.

**TERM=xterm-256color via env wrapper**
The exec session wraps the shell command with `env TERM=xterm-256color <shell>` so that `clear` emits CSI 3J and `reset` emits RIS. xterm.js handles both natively — no frontend interception needed. A UI "Clear" button is added as a convenience shortcut.

**Shared font size preference**
One font size setting for both logs and terminal panels, stored in session store (user/screen preference). Applied via CSS custom property and xterm.js `Terminal.options.fontSize`.

## Rejected Alternatives

**Cross-session terminal command history**
Shell readline already handles in-session history. Persisting across sessions would require intercepting keystrokes and maintaining a parallel history buffer, conflicting with the shell's own readline. Not worth the complexity — the shell's `.bash_history`/`.zsh_history` on the remote pod already serves this purpose.

**Backend log multiplexing**
Considered merging multiple pod log streams in the Go backend into a single WebSocket. Rejected because the frontend already handles per-stream WebSocket connections, the LogStreamer is designed for 1:1 stream-to-connection mapping, and client-side merge is simpler with no new backend abstraction.

**Timestamp-sorted cross-pod log ordering**
Would require buffering incoming lines and sorting by timestamp before display. Adds latency, complexity, and breaks real-time streaming feel. Arrival-order with pod labels is sufficient — users primarily care about "which pod said this" not strict global ordering.

**Per-forward auto-reconnect toggle**
Considered letting users mark individual forwards as "auto-reconnect" vs "manual reconnect." Simplified to a single `enabled` boolean — enabled forwards auto-reconnect, disabled forwards don't. Users who want a forward to not reconnect simply disable it.

## Priorities & Tradeoffs

- **Simplicity over configurability**: one font size for both logs and terminal, all-or-nothing auto-reconnect per forward, arrival-order log merging.
- **Reuse over new abstractions**: ResourceList for port-forward page, existing dialog for creating forwards, frontend fan-out instead of backend multiplexer.
- **Correctness via standards**: TERM=xterm-256color leverages existing terminal escape sequence support rather than custom parsing.

## Potential Gotchas

- **TERM in minimal containers**: Some containers (distroless, scratch) may not have `/usr/bin/env`. Fallback: write `export TERM=xterm-256color\n` as the first stdin byte after connection. Check if `env` exists before using the wrapper approach.
- **Port conflicts on auto-reconnect**: A saved forward using local port 8080 may conflict with another process on startup. The management page must surface the error clearly per-forward rather than failing silently.
- **Pod churn in log aggregation**: Pods may terminate while streaming. The frontend must handle individual stream disconnections gracefully — mark that pod's stream as ended, keep other streams running, and pick up new pods if the set changes (e.g., rolling deployment).
- **Color palette exhaustion**: If streaming logs from >N pods (where N is the palette size), colors will cycle and repeat. Use a palette of 8-10 distinguishable colors — sufficient for most use cases.
- **ResourceList with virtual data**: The port-forward "descriptor" has no GVR and no watch. The ResourceList component currently assumes a GVR-backed data source. The management page will need to provide items directly rather than through the watch/store mechanism.
- **Config file writes during reconnect storm**: On startup with many saved forwards, rapid status changes could trigger many config writes. The existing 500ms debounce on config saves should handle this, but verify it applies to port-forward state updates.

## Implementation Details

### 1. Port-forward Persistence

**Config schema addition** (`internal/config/`):

```go
type Config struct {
    // ... existing fields
    PortForwards map[string][]SavedPortForward `json:"portForwards,omitempty"` // keyed by context name
}

type SavedPortForward struct {
    ID         string `json:"id"`
    Namespace  string `json:"namespace"`
    Resource   string `json:"resource"`   // e.g. "pods/my-pod", "services/my-svc"
    LocalPort  int    `json:"localPort"`
    RemotePort int    `json:"remotePort"`
    Enabled    bool   `json:"enabled"`
}
```

**Manager changes** (`internal/portforward/`):

```go
// Add to Manager
func (m *Manager) SaveForward(ctxName string, fwd SavedPortForward) error
func (m *Manager) RemoveSavedForward(ctxName, id string) error
func (m *Manager) SetForwardEnabled(ctxName, id string, enabled bool) error
func (m *Manager) ListSavedForwards(ctxName string) []SavedPortForward
func (m *Manager) ReconnectSaved(ctxName string) // called on cluster connect
```

**Service layer** (`internal/services/`):

```go
// Add to AppService or a new PortForwardService
func (s *AppService) SavePortForward(ctxName string, fwd config.SavedPortForward) error
func (s *AppService) RemoveSavedPortForward(ctxName, id string) error
func (s *AppService) SetPortForwardEnabled(ctxName, id string, enabled bool) error
func (s *AppService) ListSavedPortForwards(ctxName string) []config.SavedPortForward
```

**Auto-reconnect trigger**: In `AppService` or `cluster.Manager`, after successful cluster connection, call `ReconnectSaved(ctxName)`. Each forward that fails gets status `error` with a message, emitted via the existing `portforward:{ctx}:{id}` event.

### 2. Port-forward Management Page

**Route**: `/c/:ctx/port-forwards`

Add to `routes/routes.ts`:
```typescript
{ path: '/c/:ctx/port-forwards', component: PortForwardPage }
```

**Sidebar**: Replace the `+` button behavior. Instead of opening the dialog, navigate to `/c/:ctx/port-forwards`.

**Page component** (`routes/portforwards/PortForwardPage.svelte`):

Uses `ResourceList` with a virtual descriptor:

```typescript
const descriptor: Descriptor = {
    group: '_internal',
    version: 'v1',
    resource: 'portforwards',
    columns: [
        { name: 'Resource',    expr: 'resource',          renderType: 'text' },
        { name: 'Namespace',   expr: 'namespace',         renderType: 'text' },
        { name: 'Local Port',  expr: 'localPort',         renderType: 'text' },
        { name: 'Remote Port', expr: 'remotePort',        renderType: 'text' },
        { name: 'Status',      expr: 'status',            renderType: 'badge' },
        { name: 'Enabled',     expr: 'enabled',           renderType: 'text' },
    ],
    // no detailPanels, no watch
};
```

Data source: call `ListSavedPortForwards(ctx)` + merge with active forward statuses from `portforward:{ctx}:updated` events. Items are plain objects matching the column expressions, not unstructured k8s resources.

**Actions per row**:
- Connect / Disconnect (toggle)
- Enable / Disable (toggle auto-reconnect)
- Remove (delete saved forward)
- Copy local URL (`http://localhost:{localPort}`)

**Header button**: "New Port Forward" opens the existing port-forward dialog.

### 3. Multi-pod Log Aggregation

**Entry point**: Deployment / ReplicaSet / StatefulSet / DaemonSet detail page — "Aggregate Logs" button.

**Component** (`lib/components/panels/AggregateLogsPanel.svelte`):

```typescript
// Props
interface AggregateLogsPanelProps {
    ctxName: string;
    pods: Array<{ name: string; namespace: string; containers: string[] }>;
}
```

**Pod discovery**: The parent detail page resolves owned pods via label selector (existing `ResourceEngine.List` with label filter). Passes the pod list to the panel.

**Stream management**:

```typescript
class AggregateLogStore {
    streams: Map<string, { streamId: string; ws: WebSocket; color: string }>;
    lines: Array<{ pod: string; container: string; text: string; color: string; timestamp: number }>;
    showPodPrefix: boolean; // toggleable, default true
    maxLines: number;       // buffer cap, e.g. 10000

    addPod(pod: string, ns: string, container: string): void;
    removePod(pod: string): void;
    clear(): void;
}
```

**Color assignment**: Fixed palette of 8-10 visually distinct colors (works on both light/dark themes). Assigned by pod index, cycling on overflow.

```typescript
const POD_COLORS = [
    'var(--log-color-1)', // blue
    'var(--log-color-2)', // green
    'var(--log-color-3)', // orange
    'var(--log-color-4)', // purple
    'var(--log-color-5)', // cyan
    'var(--log-color-6)', // pink
    'var(--log-color-7)', // yellow
    'var(--log-color-8)', // red
];
```

**Pod prefix display** (when enabled):
```
[my-pod-abc12] 2024-01-15T10:30:00Z INFO Starting server...
[my-pod-def34] 2024-01-15T10:30:01Z INFO Starting server...
```

Prefix is rendered as a `<span>` with the pod's assigned color. Toggle button in the panel header.

**Disconnection handling**: When a pod's WebSocket closes (pod terminated), mark its stream as ended, append a `[stream ended]` line in that pod's color, keep other streams running. Do not auto-reconnect individual pod streams — pod churn means the pod is gone.

### 4. Font Size Control

**Session store addition** (`lib/stores/session.svelte.ts`):

```typescript
class SessionStore {
    // ... existing fields
    terminalFontSize: number = $state(13); // default 13px
}
```

**UI control**: `+` / `-` buttons in both LogsPanel and TerminalPanel headers. Step size: 1px. Range: 8-24px.

**Application**:
- LogsPanel: CSS custom property `--log-font-size` on the log container, read from session store
- TerminalPanel: `terminal.options.fontSize = sessionStore.terminalFontSize` + call `terminal.resize()` / `fitAddon.fit()` on change

```typescript
$effect(() => {
    const size = sessionStore.terminalFontSize;
    if (terminal) {
        terminal.options.fontSize = size;
        fitAddon.fit();
    }
});
```

### 5. Terminal Clear/Reset

**Exec session command wrapping** (`internal/exec/manager.go`):

Change the exec request to wrap the shell with `TERM=xterm-256color`:

```go
// Before:
Param("command", session.shell)

// After:
Param("command", "env").
Param("command", "TERM=xterm-256color").
Param("command", session.shell)
```

Note: `kubectl exec` accepts multiple `command` params which become `argv`. `env TERM=xterm-256color /bin/sh` as argv = `["env", "TERM=xterm-256color", "/bin/sh"]`.

If the container lacks `/usr/bin/env` (distroless), fall back to writing `export TERM=xterm-256color\n` as the first stdin bytes after WebSocket connection in `HandleConn`.

**UI Clear button** (`lib/components/panels/TerminalPanel.svelte`):

Add a "Clear" button to the terminal toolbar that calls:

```typescript
function clearTerminal() {
    terminal.clear(); // clears scrollback + viewport
}
```

This is a convenience shortcut independent of the TERM fix — both work together.

### File Change Summary

| File | Change |
|------|--------|
| `internal/config/config.go` | Add `PortForwards` field and `SavedPortForward` type |
| `internal/portforward/manager.go` | Add save/remove/enable/list/reconnect methods, read config on init |
| `internal/services/app_service.go` | Add port-forward CRUD RPC methods, trigger reconnect on cluster connect |
| `internal/exec/manager.go` | Wrap shell command with `env TERM=xterm-256color` |
| `frontend/src/routes/routes.ts` | Add `/c/:ctx/port-forwards` route |
| `frontend/src/routes/portforwards/PortForwardPage.svelte` | New — management page using ResourceList |
| `frontend/src/lib/components/sidebar/` | Change `+` button to navigate instead of opening dialog |
| `frontend/src/lib/components/panels/AggregateLogsPanel.svelte` | New — multi-pod log viewer |
| `frontend/src/lib/stores/aggregate-logs.svelte.ts` | New — multi-stream merge store |
| `frontend/src/lib/stores/session.svelte.ts` | Add `terminalFontSize` field |
| `frontend/src/lib/components/panels/LogsPanel.svelte` | Add font size controls, read from session store |
| `frontend/src/lib/components/panels/TerminalPanel.svelte` | Add font size controls, Clear button |
| `frontend/src/app.css` | Add `--log-color-{1-8}` CSS custom properties |

### Wails Bindings

After adding the port-forward RPC methods to `AppService`, regenerate bindings:

```bash
wails3 generate bindings
```

## Definition of Done

- [ ] Port-forwards persist across app restarts in `config.json`, keyed by context
- [ ] Saved forwards auto-reconnect on cluster connect (if enabled)
- [ ] Disabled forwards appear in management page but do not attempt connection
- [ ] `/c/:ctx/port-forwards` page lists all saved + active forwards with status
- [ ] Sidebar `+` button navigates to port-forward management page
- [ ] New forwards can be created from the management page via existing dialog
- [ ] Individual forwards can be connected/disconnected/enabled/disabled/removed from management page
- [ ] Aggregate log panel streams from multiple pods simultaneously
- [ ] Each pod's log lines have a color-coded prefix (toggleable)
- [ ] Individual pod stream disconnections are handled gracefully (stream ended marker, others continue)
- [ ] Font size control (`+`/`-`) works in both LogsPanel and TerminalPanel
- [ ] Font size preference persists across sessions
- [ ] Typing `clear` in terminal clears scrollback (TERM=xterm-256color set)
- [ ] Typing `reset` in terminal performs full reset including scrollback
- [ ] UI "Clear" button in terminal toolbar clears scrollback
- [ ] Port conflicts on auto-reconnect show error status per-forward, not a toast storm
