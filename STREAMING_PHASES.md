# Streaming Enhancements — Phased Implementation

## Project Overview

Five enhancements to the streaming subsystem: port-forward persistence with auto-reconnect and a dedicated management page, multi-pod log aggregation with color-coded prefixes, font size control for logs/terminal, and terminal clear/reset via proper TERM handling. Split into two phases: port-forward infrastructure first (config + management page), then log/terminal UX improvements.

## Phase Map

```
Phase 1 — Port-forward Persistence & Management
  └── Phase 2 — Log/Terminal Enhancements
```

Phase 2 has no hard data dependency on Phase 1, but Phase 1 is sequenced first because it modifies the config schema, sidebar navigation, and route structure — changes that Phase 2's frontend work should build on top of rather than conflict with.

---

## Phase 1 — Port-forward Persistence & Management Page

> Adds persistent, auto-reconnecting port-forwards stored in config and a dedicated management page at `/c/:ctx/port-forwards`, replacing the sidebar `+` button.

| | |
|---|---|
| **Depends on** | none |
| **Parallel with** | nothing |

### Deliverables

- `SavedPortForward` type and `PortForwards` map field in `config.Config`, persisted to `config.json` keyed by cluster context name
- `portforward.Manager` methods for save/remove/enable/list/reconnect of saved forwards
- Wails RPC methods on `AppService` (or a new `PortForwardService`) exposing CRUD operations to the frontend
- Auto-reconnect logic triggered on cluster connect — iterates saved forwards for the context, starts enabled ones, emits error status per-forward on failure
- `/c/:ctx/port-forwards` route and `PortForwardPage.svelte` component using `ResourceList` with a virtual descriptor (no GVR, no watch)
- Row actions: connect/disconnect, enable/disable, remove, copy local URL
- "New Port Forward" header button that opens the existing port-forward dialog
- Sidebar `+` button changed from opening dialog to navigating to the management page
- Regenerated Wails bindings

### Tests

- **Go unit test**
  - `SaveForward` persists to config, `ListSavedForwards` returns it, `RemoveSavedForward` deletes it
  - `SetForwardEnabled(false)` marks forward as disabled; `ReconnectSaved` skips disabled forwards
  - `ReconnectSaved` starts enabled forwards and emits error event (not panic) when a forward fails (e.g. port conflict, missing pod)
  - Config round-trip: save forwards, reload config from disk, verify forwards survive
- **Frontend test (vitest)**
  - `PortForwardPage` renders saved forwards in ResourceList with correct columns
  - Row action "Enable/Disable" calls `SetPortForwardEnabled` binding
  - Row action "Remove" calls `RemoveSavedPortForward` binding
  - "New Port Forward" button opens the existing dialog component
- **Manual verification**
  - Create a port-forward, close the app, reopen — forward reconnects automatically
  - Disable a forward, restart — it appears in the list but does not connect
  - Start a forward on a port already in use — error status shown on that row, other forwards unaffected

### Out of Scope

- Multi-pod log aggregation, font size, terminal clear — Phase 2
- Port-forward history/favorites beyond save/remove — not in spec
- Editing saved forward parameters (change local port, etc.) — user can remove and recreate

### Acceptance Criteria

- [ ] `SavedPortForward` type exists in `internal/config/` with ID, Namespace, Resource, LocalPort, RemotePort, Enabled fields
- [ ] `config.json` persists `portForwards` map keyed by context name, survives app restart
- [ ] `portforward.Manager` exposes `SaveForward`, `RemoveSavedForward`, `SetForwardEnabled`, `ListSavedForwards`, `ReconnectSaved`
- [ ] Wails RPC methods for all CRUD operations, bindings regenerated
- [ ] On cluster connect, all enabled saved forwards for that context auto-reconnect
- [ ] Failed reconnects emit per-forward error status via `portforward:{ctx}:{id}` event (no toast storm)
- [ ] `/c/:ctx/port-forwards` page renders all saved + active forwards with status badges
- [ ] ResourceList virtual descriptor works without a GVR or watch — items provided directly
- [ ] Row actions (connect/disconnect, enable/disable, remove, copy URL) functional
- [ ] Sidebar `+` button navigates to `/c/:ctx/port-forwards` instead of opening dialog
- [ ] "New Port Forward" button on management page opens existing dialog
- [ ] Go unit tests pass for save/remove/enable/reconnect logic
- [ ] Frontend tests pass for page rendering and action bindings

### Source Documents

- `STREAMING_SPEC.md` — full spec with decisions, rejected alternatives, and implementation details (sections 1 and 2)
- `internal/config/config.go` — config schema to extend with `PortForwards` field
- `internal/portforward/manager.go` — port-forward Manager to extend with persistence methods
- `internal/services/app_service.go` — Wails service layer for new RPC methods
- `frontend/src/routes/routes.ts` — route definitions, add `/c/:ctx/port-forwards`
- `frontend/src/lib/components/sidebar/` — sidebar component where `+` button behavior changes
- `frontend/src/lib/components/ResourceList.svelte` — reused for virtual port-forward list
- `frontend/src/lib/registry/index.ts` — `DescriptorRegistry` and `Descriptor` type for virtual descriptor reference

### Handoff Notes

- The virtual descriptor pattern (no GVR, no watch, items provided directly) is new for ResourceList. If ResourceList assumes a watch-backed data source internally, the management page will need to bypass that assumption — likely by passing items as a prop rather than through ResourceStore. Document whatever adapter pattern is used so Phase 2 or future virtual pages can reuse it.
- The config debounce (500ms) should be verified to cover port-forward state writes during reconnect storms. If it doesn't, add explicit debouncing in the Manager's save path.
- Wails bindings must be regenerated after adding RPC methods. The `SavedPortForward` struct will generate a model class — verify no naming collision in `frontend/bindings/` index.

---

## Phase 2 — Log/Terminal Enhancements

> Adds multi-pod log aggregation with color-coded pod prefixes, shared font size control for logs and terminal panels, and proper terminal clear/reset via TERM environment variable handling.

| | |
|---|---|
| **Depends on** | Phase 1 (sidebar/route changes landed, avoids merge conflicts) |
| **Parallel with** | nothing |

### Deliverables

- `AggregateLogsPanel.svelte` component that streams logs from multiple pods simultaneously via N independent WebSocket connections
- `AggregateLogStore` (in `aggregate-logs.svelte.ts`) managing per-pod streams, merged line buffer, color assignment, and pod prefix toggle
- "Aggregate Logs" button on Deployment / ReplicaSet / StatefulSet / DaemonSet detail pages, resolving owned pods via label selector
- Color-coded pod-name prefix (`[pod-name]`) on each log line, toggleable via panel header button
- 8 CSS custom properties (`--log-color-{1-8}`) for pod colors, working in both light and dark themes
- `terminalFontSize` field in session store, persisted across sessions
- `+` / `-` font size controls in both LogsPanel and TerminalPanel headers (range 8-24px, step 1px)
- Font size applied via CSS custom property to LogsPanel and `Terminal.options.fontSize` + `fitAddon.fit()` to TerminalPanel
- Exec session command wrapping: `["env", "TERM=xterm-256color", "<shell>"]` in `manager.go`
- UI "Clear" button in TerminalPanel toolbar calling `terminal.clear()`

### Tests

- **Go unit test**
  - Exec session builds correct command argv: `["env", "TERM=xterm-256color", "/bin/sh"]` (or whichever shell)
  - Verify the Param chain produces three `command` params, not one concatenated string
- **Frontend test (vitest)**
  - `AggregateLogsPanel` renders lines from multiple mock streams with correct pod prefixes
  - Toggling pod prefix off removes `[pod-name]` prefix from rendered lines
  - Pod stream disconnection appends `[stream ended]` marker without affecting other streams
  - Font size `+` button increments `sessionStore.terminalFontSize`, `-` button decrements, clamped to 8-24
  - Font size change triggers xterm.js `options.fontSize` update (mock terminal object)
  - Clear button calls `terminal.clear()` on the xterm instance
- **Manual verification**
  - Open a Deployment detail page with 3+ pods, click "Aggregate Logs" — all pod logs stream with distinct colored prefixes
  - Kill one pod mid-stream — its stream shows `[stream ended]`, others continue
  - Toggle pod prefix off — lines render without `[pod-name]` prefix
  - Adjust font size in LogsPanel — text resizes, setting persists after page navigation
  - Adjust font size in TerminalPanel — terminal resizes correctly, `fitAddon.fit()` recalculates dimensions
  - Type `clear` in terminal — scrollback is fully cleared (not just viewport)
  - Type `reset` in terminal — full terminal reset including scrollback
  - Click "Clear" button — same effect as typing `clear`

### Out of Scope

- Timestamp-sorted cross-pod log ordering — rejected in spec, arrival-order is intentional
- Cross-session terminal command history — rejected in spec
- Backend log multiplexing — frontend fan-out is the chosen approach
- `env` fallback for distroless containers — noted as a gotcha in the spec but deferred; if `env` is missing the exec will fail visibly and can be addressed as a follow-up

### Acceptance Criteria

- [ ] `AggregateLogsPanel` opens N WebSocket streams (one per pod) and displays merged output
- [ ] Each pod's lines are prefixed with `[pod-name]` in a distinct color from the 8-color palette
- [ ] Pod prefix toggle in panel header shows/hides prefixes
- [ ] Individual stream disconnection shows `[stream ended]` marker, other streams continue
- [ ] "Aggregate Logs" button appears on Deployment, ReplicaSet, StatefulSet, DaemonSet detail pages
- [ ] Pod list resolved from owner resource via label selector
- [ ] `--log-color-{1-8}` CSS custom properties defined, working in light and dark themes
- [ ] `sessionStore.terminalFontSize` persists across sessions (written to session.json)
- [ ] `+`/`-` font size controls visible in both LogsPanel and TerminalPanel headers
- [ ] Font size changes applied immediately (CSS variable for logs, `Terminal.options.fontSize` + `fitAddon.fit()` for terminal)
- [ ] Font size clamped to 8-24px range
- [ ] Exec sessions use `["env", "TERM=xterm-256color", "<shell>"]` command argv
- [ ] `clear` command clears terminal scrollback (not just viewport)
- [ ] `reset` command performs full terminal reset including scrollback
- [ ] "Clear" button in terminal toolbar calls `terminal.clear()`
- [ ] Go unit tests pass for exec command wrapping
- [ ] Frontend tests pass for aggregate logs, font size, and clear button

### Source Documents

- `STREAMING_SPEC.md` — full spec with decisions and implementation details (sections 3, 4, and 5)
- `internal/exec/manager.go` — exec session setup, line ~122-132 where command params are built
- `internal/logs/streamer.go` — LogStreamer for understanding stream lifecycle (no changes needed)
- `frontend/src/lib/components/panels/LogsPanel.svelte` — add font size controls, reference for log rendering patterns
- `frontend/src/lib/components/panels/TerminalPanel.svelte` — add font size controls and Clear button
- `frontend/src/lib/stores/session.svelte.ts` — add `terminalFontSize` field
- `frontend/src/app.css` — add `--log-color-{1-8}` CSS custom properties
- `frontend/src/routes/` — detail page components where "Aggregate Logs" button is added

### Handoff Notes

- The `env` wrapper approach for TERM assumes `/usr/bin/env` exists in the container. The spec notes a fallback (writing `export TERM=xterm-256color\n` as first stdin bytes) but this phase does not implement it. If users report issues with distroless containers, that fallback can be added as a targeted fix.
- The 8-color palette needs to be tested in both light and dark themes. Colors that work on dark backgrounds may not have sufficient contrast on light backgrounds — pick colors that clear WCAG AA contrast in both modes.
- `AggregateLogStore` should cap its line buffer (spec suggests 10,000 lines). Without a cap, long-running aggregate streams will consume unbounded memory. Implement as a ring buffer or shift-on-overflow.
