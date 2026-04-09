# Phase 2 — Log/Terminal Enhancements

Add multi-pod log aggregation with color-coded pod prefixes, shared font size control for logs and terminal panels, and proper terminal clear/reset via TERM environment variable handling.

## First Action

Read `frontend/src/lib/components/panels/LogsPanel.svelte` to understand the current single-pod log streaming pattern — how it opens a WebSocket, receives lines, and renders them. The aggregate log panel replicates this pattern N times and merges the output.

## Context

Phase 1 landed port-forward persistence and the management page, modifying config schema, sidebar navigation, and route structure. This phase adds three independent log/terminal improvements that touch the other side of the streaming subsystem. The LogStreamer backend already supports multiple simultaneous WebSocket connections (verified: unique stream IDs, independent buffers, deadlock-aware mutex), so multi-pod aggregation is a frontend-only fan-out. Font size and terminal clear are small targeted changes.

## Files to Read

- `frontend/src/lib/components/panels/LogsPanel.svelte` — **what to look for**: WebSocket connection lifecycle, how log lines are received and rendered, the panel header layout where font size controls will be added
- `frontend/src/lib/components/panels/TerminalPanel.svelte` — **what to look for**: xterm.js `Terminal` instantiation, `fitAddon` usage, the toolbar layout for adding Clear button and font size controls
- `internal/exec/manager.go` — **what to look for**: lines ~122-132 where `Param("command", session.shell)` builds the exec request — this is where `TERM=xterm-256color` wrapping goes
- `internal/logs/streamer.go` — **what to look for**: `StartStream` return value (stream ID), `HandleConn` WebSocket handler — confirms no changes needed here, just understand the interface
- `frontend/src/lib/stores/session.svelte.ts` — **what to look for**: `SessionStore` class pattern with `$state` fields, how values are persisted — add `terminalFontSize` here
- `frontend/src/app.css` — **what to look for**: existing CSS custom property conventions and dark mode token patterns — add `--log-color-{1-8}` here

## Source Documents

- `STREAMING_SPEC.md` — sections 3 (Multi-pod Log Aggregation), 4 (Font Size Control), and 5 (Terminal Clear/Reset) contain the full design: `AggregateLogStore` shape, color palette, font size range, and `env` wrapper approach
- `STREAMING_PHASES.md` — Phase 2 section for deliverables, tests, acceptance criteria, and handoff notes

## What Exists

- **From Phase 1**: config schema with `PortForwards`, management page at `/c/:ctx/port-forwards`, sidebar navigation changes, regenerated Wails bindings
- `LogStreamer` backend supporting multiple simultaneous WebSocket streams (unique IDs, independent buffers)
- `LogsPanel.svelte` for single-pod log streaming
- `TerminalPanel.svelte` with xterm.js + fitAddon
- `SessionStore` with debounced persistence to `session.json`
- Exec `Manager` building k8s exec requests with `Param("command", shell)`
- Detail pages for Deployment, ReplicaSet, StatefulSet, DaemonSet with panel rendering infrastructure

## Deliverables

1. `AggregateLogsPanel.svelte` — streams logs from N pods simultaneously via independent WebSocket connections, displays merged output in arrival order
2. `AggregateLogStore` in `aggregate-logs.svelte.ts` — manages per-pod streams (`Map<string, {streamId, ws, color}>`), merged line buffer (capped at 10,000 lines), color assignment from 8-color palette, and `showPodPrefix` toggle
3. "Aggregate Logs" button on Deployment / ReplicaSet / StatefulSet / DaemonSet detail pages, resolving owned pods via label selector
4. Color-coded `[pod-name]` prefix on each log line as a `<span>` with the pod's assigned color, toggleable via panel header button
5. 8 CSS custom properties (`--log-color-{1-8}`) in `app.css`, working in both light and dark themes with WCAG AA contrast
6. `terminalFontSize` field in `SessionStore` (default 13px), persisted to `session.json`
7. `+` / `-` font size controls in both `LogsPanel` and `TerminalPanel` headers (range 8-24px, step 1px)
8. Font size applied via CSS custom property `--log-font-size` on log container, and `Terminal.options.fontSize` + `fitAddon.fit()` on terminal
9. Exec command wrapping in `manager.go`: `Param("command", "env").Param("command", "TERM=xterm-256color").Param("command", session.shell)` — three separate `command` params forming argv
10. UI "Clear" button in `TerminalPanel` toolbar calling `terminal.clear()`

## Tests

- **Go unit test**
  - Exec session builds correct command argv: `["env", "TERM=xterm-256color", "/bin/sh"]` (or whichever shell)
  - Verify the Param chain produces three `command` params, not one concatenated string
- **Frontend test (vitest)**
  - `AggregateLogsPanel` renders lines from multiple mock streams with correct pod prefixes
  - Toggling pod prefix off removes `[pod-name]` prefix from rendered lines
  - Pod stream disconnection appends `[stream ended]` marker without affecting other streams
  - Font size `+` increments `sessionStore.terminalFontSize`, `-` decrements, clamped to 8-24
  - Font size change triggers xterm.js `options.fontSize` update (mock terminal object)
  - Clear button calls `terminal.clear()` on xterm instance
- **Manual verification**
  - Deployment detail page with 3+ pods → "Aggregate Logs" → all pods stream with distinct colored prefixes
  - Kill one pod mid-stream → `[stream ended]` marker, others continue
  - Toggle pod prefix off → lines render without `[pod-name]`
  - Font size `+`/`-` in LogsPanel → text resizes, persists after page navigation
  - Font size `+`/`-` in TerminalPanel → terminal resizes, `fitAddon.fit()` recalculates
  - Type `clear` in terminal → scrollback fully cleared
  - Type `reset` in terminal → full reset including scrollback
  - Click "Clear" button → same effect as `clear` command

## Acceptance Criteria

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

## Definition of Done

Opening a Deployment detail page with multiple running pods shows an "Aggregate Logs" button. Clicking it opens a panel streaming all pods' logs simultaneously, each line prefixed with `[pod-name]` in a distinct color. Toggling prefixes off removes them. Killing a pod shows `[stream ended]` while other streams continue. Font size `+`/`-` controls in both log and terminal panels resize text immediately, and the preference persists across sessions. Typing `clear` or `reset` in a terminal clears the full scrollback (not just the viewport), and a "Clear" button in the toolbar does the same.

## Known Gotchas

- **Line buffer must be capped.** `AggregateLogStore` merges N streams into a single `lines` array. Without a cap, long-running aggregate streams will consume unbounded memory. Implement a 10,000-line cap with shift-on-overflow or a ring buffer.
- **Pod colors must pass WCAG AA in both themes.** Colors chosen for dark backgrounds may lack contrast on light backgrounds. Test all 8 colors against both `--bg` and `--surface` tokens in light and dark modes before committing the palette.
- **`env` may not exist in distroless containers.** The `["env", "TERM=xterm-256color", "<shell>"]` argv assumes `/usr/bin/env` is present. If it's missing, the exec will fail visibly. A fallback (writing `export TERM=xterm-256color\n` as first stdin bytes) is deferred — don't implement it now, but don't be surprised if it comes up.
- **xterm.js `fitAddon.fit()` after font size change.** Changing `Terminal.options.fontSize` alone doesn't resize the terminal grid. You must call `fitAddon.fit()` after the font size update so xterm recalculates rows/columns. If `fit()` is called before the DOM reflects the new font size, it may calculate wrong dimensions — use a microtask or `requestAnimationFrame` if needed.
- **`$effect` tracking on font size.** Reading `sessionStore.terminalFontSize` inside an `$effect` that also reads other state (like the WebSocket connection) will re-run the entire effect on font size change. Isolate the font size effect from the connection effect — use separate `$effect` blocks (see cerebrum: `$state` in `$effect` caused infinite reconnect loop).
