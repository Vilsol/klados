# Klados — MVP Implementation Phases

> The Pod is a Lie

Each phase produces a runnable, testable increment. Phases build on each other sequentially.

---

## Phase 1 — Skeleton & Cluster Connection

### Scope
- Wails 3 app shell with Svelte 5 frontend
- Fiber WebSocket server lifecycle (random port, `streaming:ready` event)
- Sidebar layout (collapsible, placeholder resource tree)
- Tab bar (static, wired up for future use)
- Header with cluster switcher and namespace picker
- Cluster manager: kubeconfig auto-detection, manual import, connect/disconnect
- Connection health monitoring and disconnect indicator (header color shift)
- Namespace listing and switching
- Logging setup (slog + tint + slox)
- Config/preferences storage (xdg + json)
- Session state persistence and restore
- Dark/light theme with system detection
- Tailwind v4 + Bits UI setup

### Test Targets

**Go unit tests:**
- Cluster manager: kubeconfig parsing, context enumeration, connect/disconnect lifecycle
- Config storage: read/write/update preferences, XDG path resolution
- Session state: serialize/deserialize, debounced writes
- Fiber server: startup on random port, auth token generation, shutdown

**Go integration tests (envtest):**
- Connect to real API server, verify client health check
- Namespace listing returns expected namespaces
- Connection drop detection and reconnect

**Frontend tests:**
- Header renders cluster name and namespace picker
- Sidebar collapses/expands
- Theme switching between dark/light/system
- Session restore populates correct cluster and namespace on mount

### Definition of Done
- App opens to cluster list view
- User can add a cluster via kubeconfig, connect to it, see connection status
- Namespace picker populates and switches active namespace
- Closing and reopening the app restores the last connected cluster and namespace
- Theme toggle works
- Fiber WebSocket server starts and frontend receives the `streaming:ready` event

---

## Phase 2 — Resource Engine & List Views

### Scope
- Resource engine: list, get, delete via dynamic client for any GVR
- API resource discovery on connect (core + CRDs)
- Sidebar populated from discovery results, grouped by category
- Watch manager: metadata-only informers, start/stop on navigation, grace period
- Wails event namespacing (`watch:{clusterID}:{gvr}:{namespace}`)
- Descriptor registry with CEL expressions (cel-go + cel-js)
- Built-in descriptors for core resources: Pods, Deployments, StatefulSets, DaemonSets, ReplicaSets, Jobs, CronJobs, Services, Ingresses, ConfigMaps, Secrets, PVs, PVCs
- ResourceList component: virtual scrolling (TanStack), column rendering from descriptors
- Filter by labels, sort by any column
- Multi-namespace view (all namespaces)
- Resource delete with confirmation dialog
- Notification system (toast/snackbar)
- Multi-resource grouped views (Workloads, Networking, Storage, Config sidebar groups)

### Test Targets

**Go unit tests:**
- Resource engine: list/get/delete against fake dynamic client
- Watch manager: informer start/stop lifecycle, grace period expiry
- Descriptor CEL evaluation: all built-in descriptors produce correct column values
- Discovery: parse API resource list into categorized GVR list

**Go integration tests (envtest):**
- List pods/deployments/services returns correct data
- Watch receives create/update/delete events
- Delete resource returns success, watch fires delete event
- Discovery returns expected resource types

**Frontend tests:**
- ResourceList renders correct columns from descriptor
- Virtual scrolling handles 1000+ rows without lag
- Label filter reduces visible rows
- Column sort toggles ascending/descending
- Sidebar tree matches discovered resources
- Delete confirmation dialog blocks until confirmed

### Definition of Done
- Sidebar shows all discovered resource types grouped by category
- Clicking a resource type shows a list view with correct columns
- List updates in real-time when resources change (watch-driven)
- User can filter by labels, sort by columns
- User can delete a resource with confirmation
- Navigating away and back does not cause duplicate watches
- All built-in resource types render meaningful columns via CEL descriptors

---

## Phase 3 — Resource Detail & YAML Editing

### Scope
- ResourceDetail view: generic shell with type-specific panels from descriptor
- Overview panel: key-value display of important fields
- Pod detail: containers, init containers, status breakdown, conditions, env vars, volume mounts
- Deployment detail: strategy, selectors, conditions, replica count
- Detail views for all other built-in resource types
- CodeMirror 6 integration: YAML view with syntax highlighting, line numbers, find/replace
- YAML edit and apply (full object update)
- Conflict handling (409): show error, surface conflicting fields, offer refresh
- Schema fetching from cluster `/openapi/v3`, version-keyed disk cache
- Schema validation in editor via `codemirror-json-schema`
- Events panel on detail view (resource-specific events)
- Labels and annotations view/edit
- Pod actions: delete, force delete
- Deployment actions: scale replicas, restart (rollout restart)
- Export resource as YAML, copy to clipboard
- Breadcrumb navigation

### Test Targets

**Go unit tests:**
- Apply resource: success case, 409 conflict case, validation error case
- Schema service: fetch, cache hit, cache miss, version mismatch invalidation
- Enrichers: pod status summary, deployment ready count, container restart count
- Scale/restart: correct patch request generated

**Go integration tests (envtest):**
- Get resource returns full object
- Apply modified resource succeeds
- Apply with stale resourceVersion returns 409
- Scale deployment changes replica count
- Events listed for specific resource match expected events

**Frontend tests:**
- ResourceDetail renders correct panels based on descriptor
- CodeMirror loads with YAML content, syntax highlighting active
- Edit YAML and apply triggers backend call
- Conflict error displays with refresh action
- Breadcrumb shows correct navigation path
- Copy YAML to clipboard works
- Scale slider/input updates replica count

### Definition of Done
- Clicking a resource in the list opens a detail view with overview, YAML, and events tabs
- YAML editor has syntax highlighting, schema validation, and find/replace
- User can edit YAML and apply changes
- Conflicts are surfaced clearly with option to refresh and retry
- Pod detail shows container breakdown, env vars, volume mounts
- Deployment can be scaled and restarted from the detail view
- Events tab shows resource-specific events

---

## Phase 4 — Logs & Terminal

### Scope
- LogStreamer: stream pod logs over Fiber WebSocket
- xterm.js integration with WebGL renderer + DOM fallback
- Log follow mode (real-time streaming)
- Historical log retrieval with line limits
- Multi-container log selection
- Init container logs, previous container logs
- Log search/filter (regex)
- Log level highlighting (error, warn, info)
- Timestamp toggle
- Log wrapping toggle
- Log download/export
- ExecManager: terminal exec sessions over Fiber WebSocket
- Shell selection (bash, sh, zsh)
- Terminal resize handling (ResizeObserver → fitAddon → SIGWINCH)
- Multiple concurrent terminal sessions with tabs
- Backpressure flow control (high/low water mark)
- Scrollback cap (10,000 lines)
- Copy/paste support
- Explicit `.dispose()` on teardown

### Test Targets

**Go unit tests:**
- LogStreamer: start/stop lifecycle, options (follow, previous, since, tail)
- ExecManager: session create/destroy, resize propagation
- Backpressure: high water mark triggers pause, low water mark triggers resume
- WebSocket auth: valid token accepted, invalid token rejected

**Go integration tests (envtest + real pod):**
- Stream logs from a running pod, verify lines received
- Exec into pod, send command, receive output
- Resize terminal, verify new dimensions propagated
- Multiple concurrent exec sessions to different pods

**Frontend tests:**
- Terminal component initializes xterm.js, connects to WebSocket
- WebGL fallback to DOM renderer on error
- Log viewer renders streamed lines, follows new output
- Log filter reduces visible lines by regex
- Multi-container dropdown switches log source
- Timestamp toggle shows/hides timestamps
- Terminal dispose cleans up all resources (no memory leak)

### Definition of Done
- User can tail logs from any pod with real-time streaming
- Multi-container pods show a container selector
- Log search highlights matches, filter hides non-matching lines
- Timestamps can be toggled, logs can be downloaded
- User can exec into a pod and get an interactive shell
- Multiple terminal tabs work concurrently
- Terminal handles resize correctly (no garbled output)
- Rapid log output does not crash the app (backpressure works)
- WebGL renderer used by default, DOM fallback works on degraded systems

---

## Phase 5 — Networking & Port Forwarding

### Scope
- PortForwardManager: start/stop, state tracking (active, failed, reconnecting)
- Port-forward UI: active forwards list, start/stop controls, status indicator
- Local port selection (auto or manual)
- Port-forward target intent storage (pod name vs service selector vs statefulset)
- Auto-reconnect for statefulset and service-based forwards
- Notify-only for generated pod name forwards
- Service detail: selectors, endpoints, endpoint resolution (show backing pods)
- Ingress detail: TLS, rules, annotations, open in browser link
- ConfigMap detail: syntax-highlighted values, edit individual keys
- Secret detail: base64 decoded toggle, show/hide values, copy decoded to clipboard

### Test Targets

**Go unit tests:**
- PortForwardManager: start/stop lifecycle, state transitions
- Target intent resolution: service selector → pod lookup, statefulset name stability
- Auto-reconnect: triggers on pod death for service/statefulset, does not trigger for generated pods
- Secret decode: base64 decode/encode roundtrip

**Go integration tests (envtest):**
- Port-forward to a pod, verify local port accepts connections
- Service endpoint resolution returns correct backing pods
- Port-forward reconnect after pod restart (statefulset)

**Frontend tests:**
- Port-forward list shows active forwards with status
- Start/stop controls trigger backend calls
- Service detail shows backing pods
- Secret values hidden by default, revealed on toggle
- Copy decoded secret value to clipboard

### Definition of Done
- User can port-forward to a pod or service, selecting local port
- Active port-forwards shown in sidebar with status indicators
- Service-based forwards auto-reconnect when backing pods change
- StatefulSet-based forwards auto-reconnect on pod restart
- Generated pod forwards notify user on disconnect without auto-reconnect
- Service detail shows which pods back the service
- Ingress detail links open in default browser
- ConfigMaps and Secrets can be viewed and edited with appropriate UX (syntax highlighting, show/hide)

---

## Phase 6 — Polish & UX Completion

### Scope
- Command palette (Ctrl+K): resource search, actions, navigation
- Global fuzzy search across all resource types
- Focus-aware keyboard shortcut system (normal, terminal-capture, editor-capture modes)
- Escape chord for terminal capture exit
- Central shortcut registry
- Multi-cluster simultaneous connections
- Tab management: open/close/reorder tabs, tabs remember scroll position
- Cluster-wide event stream view
- Event filtering (type, reason, source)
- Warning event highlighting
- Notification system polish (error details, action buttons)
- Confirmation dialogs for all destructive actions
- Import YAML from clipboard
- Create resource from YAML editor
- Virtual scrolling polish (edge cases, scroll position restoration)
- Session restore: full tab state, scroll positions, sidebar state, window geometry
- Performance audit: watch cleanup, memory profiling, dispose verification
- Accessibility baseline: keyboard navigation throughout non-terminal/editor areas

### Test Targets

**Go unit tests:**
- Shortcut registry: mode-aware dispatch, no conflicts in default keymap
- Multi-cluster: independent watch managers, event isolation, no cross-cluster leaks
- Create resource: valid YAML accepted, invalid YAML rejected with error

**Go integration tests (envtest):**
- Two clusters connected simultaneously, watches independent
- Create resource via YAML, verify it appears in list
- Event stream receives cluster events

**Frontend tests:**
- Command palette opens on Ctrl+K, fuzzy search returns correct results
- Shortcuts blocked in terminal capture mode, work in normal mode
- Escape chord exits terminal capture mode
- Tab management: open, close, reorder, restore on app restart
- Multi-cluster: switching tabs changes active cluster context in header
- Session restore: tabs, scroll positions, sidebar state all restored

### Definition of Done
- Command palette searches across all resource types and provides quick navigation
- Keyboard shortcuts work correctly and never interfere with terminal/editor input
- Multiple clusters connected simultaneously with independent state
- Full session restore on app restart (tabs, clusters, namespaces, scroll positions, window geometry)
- All destructive actions require confirmation
- Event stream view shows cluster events with filtering
- Resources can be created from YAML
- App feels responsive with no memory leaks after extended use
- All MVP features from FEATURES.md are implemented and tagged complete
