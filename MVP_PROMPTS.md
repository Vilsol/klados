# Klados — Phase Prompts

Prompts for starting each implementation phase in a new session.

---

## Phase 1 — Skeleton & Cluster Connection

I'm starting Phase 1 of the klados project — a Kubernetes desktop IDE built on Go + Wails 3 + Svelte 5. All architectural decisions have been made and documented.

Read these files first:
- `ARCHITECTURE.md` — full technical architecture, dependencies, and design decisions
- `PHASES.md` — Phase 1 scope, test targets, and definition of done
- `FEATURES.md` — full feature list with MVP tags

The project currently has only the Wails 3 scaffold (default greetservice). Phase 1 delivers:

1. Wails 3 app shell with Svelte 5 + Tailwind v4 + Bits UI setup
2. Fiber WebSocket server lifecycle (random port, `streaming:ready` event, auth token)
3. Sidebar layout (collapsible, placeholder resource tree)
4. Tab bar (wired up for future use)
5. Header with cluster switcher, namespace picker, connection indicator
6. Cluster manager: kubeconfig auto-detection + manual import, connect/disconnect, health monitoring
7. Namespace listing and switching
8. Logging setup (slog + tint + slox)
9. Config/preferences storage (xdg + json)
10. Session state persistence and restore
11. Dark/light theme with system detection

Start by examining the existing scaffold, then plan the implementation order for Phase 1. Propose the order before writing code. Aim for 80%+ Go test coverage using testza + client-go/dynamic/fake + envtest.

---

## Phase 2 — Resource Engine & List Views

I'm starting Phase 2 of the klados project. Phase 1 is complete — the app shell, cluster connection, sidebar, theming, and session persistence are all working.

Read these files first:
- `ARCHITECTURE.md` — full technical architecture, especially: Resource Engine (Layer 2), Watch Manager (Layer 3), Descriptor Registry, CEL expressions, and the Dependencies section
- `PHASES.md` — Phase 2 scope, test targets, and definition of done
- `FEATURES.md` — full feature list with MVP tags

Phase 2 delivers:

1. Resource engine: list, get, delete via dynamic client for any GVR
2. API resource discovery on connect — sidebar populated from discovered resources, grouped by category
3. Watch manager: metadata-only informers, start/stop on navigation with grace period
4. Wails event namespacing (`watch:{clusterID}:{gvr}:{namespace}`)
5. Descriptor registry with CEL expressions (cel-go + cel-js) — three-stage pipeline: Go enricher → CEL extraction → frontend renderer
6. Built-in descriptors for all core MVP resources (Pods, Deployments, StatefulSets, DaemonSets, ReplicaSets, Jobs, CronJobs, Services, Ingresses, ConfigMaps, Secrets, PVs, PVCs)
7. ResourceList component with TanStack Virtual scrolling, column rendering from descriptors
8. Filter by labels, sort by any column
9. Multi-namespace view (all namespaces)
10. Resource delete with confirmation dialog
11. Notification system (toast/snackbar)
12. Multi-resource grouped views in sidebar (Workloads, Networking, Storage, Config)

Review the Phase 1 code to understand what's already built, then plan the implementation order. Propose the order before writing code. Aim for 80%+ Go test coverage.

---

## Phase 3 — Resource Detail & YAML Editing

I'm starting Phase 3 of the klados project. Phases 1-2 are complete — app shell, cluster connection, resource engine, list views with virtual scrolling, watches, and the descriptor registry are all working.

Read these files first:
- `ARCHITECTURE.md` — especially: YAML Editor Integration (CodeMirror 6), Schema Management, Descriptor Registry (detailPanels, actions), and Error Handling (YAML Apply Conflicts)
- `PHASES.md` — Phase 3 scope, test targets, and definition of done
- `FEATURES.md` — full feature list with MVP tags

Phase 3 delivers:

1. ResourceDetail view: generic shell with type-specific panels driven by descriptors
2. Overview panel: key-value display of important fields per resource type
3. Pod detail: containers, init containers, status breakdown, conditions, env vars, volume mounts
4. Deployment detail: strategy, selectors, conditions, replica count
5. Detail views for all other built-in resource types
6. CodeMirror 6 integration: YAML view with syntax highlighting, line numbers, find/replace, undo/redo
7. YAML edit and apply (full object update)
8. Conflict handling (409): show error, surface conflicting fields, offer refresh
9. Schema fetching from cluster `/openapi/v3` endpoint, version-keyed disk cache (`{clusterUID}:{serverVersion}`)
10. Schema validation in editor via codemirror-json-schema
11. Events panel on detail view (resource-specific events)
12. Labels and annotations view/edit
13. Pod actions: delete, force delete
14. Deployment actions: scale replicas, restart (rollout restart)
15. Export resource as YAML, copy to clipboard
16. Breadcrumb navigation

Review the Phase 1-2 code to understand what's already built, then plan the implementation order. Propose the order before writing code. Aim for 80%+ Go test coverage.

---

## Phase 4 — Logs & Terminal

I'm starting Phase 4 of the klados project. Phases 1-3 are complete — app shell, cluster connection, resource engine, list views, detail views, YAML editing with schema validation, and event panels are all working.

Read these files first:
- `ARCHITECTURE.md` — especially: Fiber WebSocket Server (Layer 5), Session Services (Layer 4: LogStreamer, ExecManager), Terminal Integration (xterm.js), and the streaming:ready event flow
- `PHASES.md` — Phase 4 scope, test targets, and definition of done
- `FEATURES.md` — full feature list with MVP tags

Phase 4 delivers:

1. LogStreamer: stream pod logs over Fiber WebSocket with unique channel IDs
2. xterm.js integration with WebGL renderer + DOM fallback (try/catch for WebKitGTK)
3. Log follow mode (real-time streaming)
4. Historical log retrieval with line limits
5. Multi-container log selection
6. Init container logs and previous container logs
7. Log search/filter (regex) and log level highlighting (error, warn, info)
8. Timestamp toggle and log wrapping toggle
9. Log download/export
10. ExecManager: terminal exec sessions over Fiber WebSocket
11. Shell selection (bash, sh, zsh)
12. Terminal resize handling (ResizeObserver → fitAddon.fit() → SIGWINCH)
13. Multiple concurrent terminal sessions with tabs
14. Backpressure flow control (high/low water mark) to prevent OOM from runaway output
15. Scrollback cap (10,000 lines)
16. Copy/paste support
17. Explicit .dispose() on teardown to prevent memory leaks

The Fiber WebSocket server is already running from Phase 1 (streaming:ready event, auth token). This phase adds the /ws/exec/:sessionID and /ws/logs/:streamID endpoints.

Review the Phase 1-3 code to understand what's already built, then plan the implementation order. Propose the order before writing code. Aim for 80%+ Go test coverage.

---

## Phase 5 — Networking & Port Forwarding

I'm starting Phase 5 of the klados project. Phases 1-4 are complete — app shell, cluster connection, resource browsing, detail views, YAML editing, logs, and terminal are all working.

Read these files first:
- `ARCHITECTURE.md` — especially: Port-Forward Resilience (two-layer architecture: discovery + tunnel, inspired by Tilt), the target intent behavior matrix, and best-pod selection
- `PHASES.md` — Phase 5 scope, test targets, and definition of done
- `FEATURES.md` — full feature list with MVP tags
- `tilt-port-forward-resiliency.md` — reference doc on Tilt's port-forward architecture

Phase 5 delivers:

1. PortForwardManager with two-layer architecture: discovery layer (pod tracking via owner refs / label selectors) and tunnel layer (SPDY connection lifecycle)
2. Best-pod selection: Running > Ready > creation timestamp
3. Port-forward UI: active forwards list in sidebar, start/stop controls, status indicator (Active, Reconnecting, Failed)
4. Local port selection (auto or manual)
5. Target intent storage and behavior matrix:
   - Raw pod name (generated): transport auto-reconnect only, notify on pod death
   - StatefulSet pod: auto-reconnect to same name
   - Service/Deployment: auto-select new best pod via owner refs or selectors
6. Transport-level auto-reconnect for ALL forward types (network blips, API server restarts)
7. Service detail: selectors, endpoints, endpoint resolution (show backing pods)
8. Ingress detail: TLS, rules, annotations, open in browser link
9. ConfigMap detail: syntax-highlighted values, edit individual keys
10. Secret detail: base64 decoded toggle, show/hide values, copy decoded to clipboard

Review the Phase 1-4 code to understand what's already built, then plan the implementation order. Propose the order before writing code. Aim for 80%+ Go test coverage.

---

## Phase 6 — Polish & UX Completion

I'm starting Phase 6 of the klados project. Phases 1-5 are complete — the full feature set is functional. This phase is about polish, UX completion, and making the app a reliable daily driver.

Read these files first:
- `ARCHITECTURE.md` — especially: Keyboard Shortcuts (focus-aware system), Multi-Cluster Connectivity (simultaneous connections), App Lifecycle & State Persistence
- `PHASES.md` — Phase 6 scope, test targets, and definition of done
- `FEATURES.md` — full feature list with MVP tags (verify all MVP items are implemented)

Phase 6 delivers:

1. Command palette (Ctrl+K): fuzzy resource search, actions, navigation via Bits UI combobox
2. Global fuzzy search across all resource types
3. Focus-aware keyboard shortcut system:
   - Normal mode: global shortcuts active
   - Terminal capture mode: all input passes to terminal, escape chord (Ctrl+Shift+Escape) to exit
   - Editor capture mode: editor shortcuts take priority, non-conflicting globals still work
   - Central shortcut registry for future keybinding customization
4. Multi-cluster simultaneous connections: independent watches, events, namespaces per cluster
5. Tab management: open/close/reorder, tabs remember scroll position, per-cluster tab context
6. Cluster-wide event stream view with filtering (type, reason, source) and warning highlighting
7. Notification system polish (error details, action buttons)
8. Confirmation dialogs for all destructive actions
9. Import YAML from clipboard
10. Create resource from YAML editor
11. Virtual scrolling polish (edge cases, scroll position restoration)
12. Session restore: full tab state, scroll positions, sidebar state, window geometry, connected clusters
13. Performance audit: watch cleanup on inactive clusters, memory profiling, dispose verification
14. Accessibility baseline: keyboard navigation throughout non-terminal/editor areas
15. Final pass: verify all MVP features from FEATURES.md are implemented

Review all existing code, then plan the implementation order. Propose the order before writing code. Aim for 80%+ Go test coverage. Run a final check against FEATURES.md MVP items.
