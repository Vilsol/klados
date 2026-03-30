# Klados — Architecture

> The Pod is a Lie

Technical architecture for a Kubernetes desktop IDE built on Go + Wails 3 + Svelte 5.

---

## Technology Decisions

| Concern | Choice | Rationale |
|---|---|---|
| Desktop framework | Wails 3 | Lightweight, no Electron overhead, uses OS webview |
| Backend language | Go | Native client-go access, strong k8s ecosystem |
| Frontend framework | Svelte 5 (runes) | Fast, minimal runtime, reactive |
| Terminal emulator | xterm.js | Industry standard, WebGL renderer, proven in VS Code |
| Code/YAML editor | CodeMirror 6 | No web workers (WebKitGTK safe), modular, immutable state, per-instance schema isolation |
| Streaming transport | Go Fiber (WebSocket) | localhost WebSocket server for terminal I/O and log streaming — Wails events are too slow for binary/high-throughput data |
| Request/response transport | Wails bindings | Wails bound methods for synchronous operations (list, get, apply, scale, etc.) |
| Descriptor expressions | CEL (Common Expression Language) | Same language used in k8s itself, runs in both Go (cel-go) and JS (cel-js), eliminates logic duplication |
| K8s client approach | Dynamic client (unstructured) | One codepath for core resources and CRDs, extensible via descriptor registry |
| Plugin runtime | Wasm (wazero) | Cross-platform, sandboxed, no CGO, any source language |
| Plugin UI | Svelte components | Dynamically loaded bundles in defined UI slots |

---

## Dependencies

### Go Backend

| Purpose | Library |
|---|---|
| Desktop framework | `github.com/wailsapp/wails/v3` |
| K8s client | `k8s.io/client-go` (dynamic, discovery, informers, remotecommand, portforward) |
| WebSocket server | `github.com/gofiber/fiber/v3` |
| CEL expressions | `github.com/google/cel-go` |
| YAML parsing | `gopkg.in/yaml.v3` |
| Logging | `log/slog` + `github.com/lmittmann/tint` (colored output) + `github.com/Vilsol/slox` (context-propagated logging) |
| Config/preferences | `github.com/adrg/xdg` (XDG paths, already transitive dep) + `encoding/json` (stdlib) |
| Wasm runtime (v2) | `github.com/tetragonolobus/wazero` |
| OCI artifacts (future) | `oras.land/oras-go` |
| Testing | `github.com/MarvinJWendt/testza` + `k8s.io/client-go/dynamic/fake` + `sigs.k8s.io/controller-runtime/pkg/envtest` |

Preferences stored at `$XDG_CONFIG_HOME/klados/config.json`. Schema cache stored at `$XDG_CACHE_HOME/klados/schemas/`.

### Frontend

| Purpose | Library |
|---|---|
| UI framework | Svelte 5 (runes) |
| Build tool | Vite |
| Router | `svelte-spa-router` |
| CSS | Tailwind CSS v4 |
| Component primitives | Bits UI (headless, Svelte 5 native, WAI-ARIA compliant — dropdowns, modals, combobox, context menus, tooltips, tabs, dialogs) |
| Icons | `lucide-svelte` |
| Terminal emulator | `xterm.js` + `@xterm/addon-webgl`, `@xterm/addon-fit`, `@xterm/addon-search`, `@xterm/addon-web-links`, `@xterm/addon-clipboard` |
| Code/YAML editor | CodeMirror 6 + `@codemirror/lang-yaml`, `@codemirror/search`, `@codemirror/autocomplete`, `@codemirror/merge` |
| Schema validation | `codemirror-json-schema` |
| CEL expressions | `cel-js` |
| Virtual scrolling | `@tanstack/svelte-virtual` |
| Testing | Vitest + `@testing-library/svelte` |

---

## Backend Architecture (Go)

### Layer 1 — Cluster Manager

Manages kubeconfig loading, multiple cluster connections, and client lifecycle.

Each cluster connection holds:
- `*rest.Config`
- `*kubernetes.Clientset` (for discovery and auth)
- `dynamic.Interface` (primary client for all resource operations)
- `discovery.DiscoveryClient`

Responsibilities:
- Kubeconfig auto-detection and manual import
- Connection health monitoring and reconnection
- Auth token refresh (exec-based, OIDC, etc.)
- Context/namespace switching
- Expose connection state changes as events

### Layer 2 — Resource Engine

Generic resource operations built on the dynamic client. All resource interactions go through a single codepath that takes a GVR (GroupVersionResource).

Operations: List, Get, Create, Update, Patch, Delete.

Returns `unstructured.Unstructured` objects. Type-specific logic lives in the **descriptor registry** (see below), not in this layer.

### Layer 3 — Watch Manager

Manages client-go informers for real-time resource updates.

- **Metadata-only informers** for list views — reduced memory footprint, no full object caching
- **Full object fetch** on demand for detail views
- Informer lifecycle tied to frontend navigation — start on view mount, stop on unmount (with grace period for back-navigation)
- Diffs pushed to frontend via Wails events, namespaced as `{clusterID}:{gvr}:{namespace}`
- Reconnection state exposed to frontend for disconnect indicator

### Layer 4 — Session Services

Stateful, long-lived operations managed on the backend:

**ExecManager**
- Spawns k8s exec sessions (remotecommand)
- Each session gets a unique ID
- I/O routed over Fiber WebSocket (not Wails events)
- Handles SIGWINCH propagation on resize
- Survives frontend re-renders

**LogStreamer**
- Streams pod logs via k8s API
- Each stream gets a unique channel ID
- Data routed over Fiber WebSocket
- Supports follow mode, previous container, since/tail options
- Implements backpressure: high/low water mark flow control to prevent OOM from runaway output

**PortForwardManager**
- Manages active port-forward sessions
- Tracks state (active, failed, reconnecting)
- Exposes start/stop via Wails bindings
- State changes pushed via Wails events

### Layer 5 — Fiber WebSocket Server

A Go Fiber HTTP server running on localhost (random port, bound to 127.0.0.1 only).

Purpose: high-throughput bidirectional streaming for terminal and log data, where Wails events are too slow.

Lifecycle:
- Fiber server starts during Wails `OnStartup`, binds to a random available port
- Once listening, emits a Wails event `streaming:ready` with `{ port, token }`
- Frontend components that need WebSocket (Terminal, LogViewer) wait for this event before rendering
- Fiber server shuts down during Wails `OnShutdown`

Security:
- Bind to localhost only
- Generate a random auth token on startup
- Token included in WebSocket URL path
- Validate Origin header to only allow Wails webview origins

Endpoints:
- `GET /ws/exec/:sessionID` — terminal I/O
- `GET /ws/logs/:streamID` — log streaming

### Layer 6 — Wails Services (API Surface)

Thin service structs bound to Wails. These are the only layer the frontend calls directly.

| Service | Responsibilities |
|---|---|
| `ClusterService` | Connect, disconnect, list contexts, switch namespace, connection health |
| `ResourceService` | List, get, create, update, delete for any GVR; start/stop watches |
| `LogService` | Start/stop log streams (returns WebSocket stream ID) |
| `ExecService` | Open/close terminal sessions (returns WebSocket session ID) |
| `PortForwardService` | Start/stop port-forwards, list active |
| `EditorService` | Validate YAML, apply resource, diff |
| `SchemaService` | Fetch and cache OpenAPI schemas from connected clusters |

---

## Frontend Architecture (Svelte 5)

### Routing

Lightweight client-side router (no SvelteKit — Wails handles the app shell).

Routes map to resource views:
- `/clusters` — cluster chooser
- `/c/:ctx` — cluster overview
- `/c/:ctx/:gvr` — resource list (e.g., `/c/prod/apps.v1.deployments`)
- `/c/:ctx/:gvr/:ns/:name` — resource detail

### State Management

Svelte 5 runes for reactivity. Key stores:

**ClusterStore** — active cluster, connection status, available contexts, active namespace, disconnect state.

**ResourceStore** — generic reactive store bound to a watch channel. Components request a watch on mount; the store receives Wails events and updates reactively. Watch released on unmount.

**SessionStore** — active terminals (WebSocket connections), active log streams, active port-forwards.

### Descriptor Registry

The central abstraction for type-specific rendering. A registry mapping GVRs to descriptor definitions.

Descriptors use **CEL (Common Expression Language)** for data extraction — the same expression language used in Kubernetes itself (ValidatingAdmissionPolicy, CRD validation rules). This eliminates logic duplication: both Go and JS evaluate the same expressions via `cel-go` and `cel-js` respectively.

The rendering pipeline has three stages:

1. **Go enricher** — injects precomputed/cross-resource fields into the unstructured object (e.g., `readyDisplay: "2/3"`, `restartCount: 5`)
2. **CEL expression** — extracts a display value from the enriched object
3. **Frontend renderer** — decides how to present the value (plain text, badge, progress bar, colored cell)

Descriptor definition (YAML/JSON, shared by both sides):

```yaml
gvr: apps/v1/deployments
columns:
  - name: Name
    expr: "metadata.name"
    render: text
  - name: Ready
    expr: "status.readyDisplay"
    render: badge
  - name: Up-to-date
    expr: "string(status.updatedReplicas)"
    render: text
  - name: Age
    expr: "metadata.creationTimestamp"
    render: age
detailPanels: [overview, pods, events, yaml]
actions: [scale, restart, delete]
```

CEL handles: field access, string formatting, conditionals, list operations, null coalescing.
CEL does NOT handle: cross-resource joins, async data, API calls, rendering — those are the enricher and renderer's jobs.

Core resources ship with built-in descriptors + enrichers. Plugins register additional descriptors for custom GVRs (CEL expressions for columns, Wasm enrichers for computed fields). Users can override column visibility and ordering.

### Component Structure

```
App
├── Header
│   ├── ClusterSwitcher
│   ├── NamespacePicker
│   ├── ConnectionIndicator (header color shift on disconnect)
│   └── CommandPalette (overlay, Ctrl+K)
├── Sidebar
│   ├── ResourceTree (grouped: Workloads, Networking, Config, Storage, RBAC)
│   └── PortForwardList
├── TabBar (open resource tabs)
└── ContentArea
    ├── ResourceList (generic — takes GVR + descriptor, renders columns)
    │   └── ResourceRow
    ├── ResourceDetail (generic shell, descriptor defines panels)
    │   ├── OverviewPanel (type-specific computed fields)
    │   ├── YAMLEditor (CodeMirror 6)
    │   ├── EventsPanel
    │   ├── LogsPanel (xterm.js via WebSocket)
    │   └── TerminalPanel (xterm.js via WebSocket)
    └── MultiResourceView (workloads/networking/storage/config grouped)
```

### Terminal Integration (xterm.js)

- WebGL renderer with automatic DOM fallback (try/catch for WebKitGTK)
- WebSocket connection to Fiber backend for I/O
- Backpressure via `term.write(chunk, callback)` with high/low water mark flow control
- `ResizeObserver` triggers `fitAddon.fit()`, sends new dimensions to backend, backend sends SIGWINCH
- Scrollback capped at 10,000 lines; deep history handled by backend if needed
- Explicit `.dispose()` on teardown to prevent memory leaks
- Addons: `addon-webgl`, `addon-fit`, `addon-search`, `addon-web-links`, `addon-clipboard`

### YAML Editor Integration (CodeMirror 6)

- No web worker dependency — works cleanly in Wails webview
- Per-instance schema validation via `codemirror-json-schema` with Kubernetes schemas
- Diff view via `@codemirror/merge` (lightweight, uses `google-diff-match-patch`)
- Incremental Lezer parser handles 10k+ line files without freezing
- Extensions: `@codemirror/lang-yaml`, `@codemirror/search`, `@codemirror/autocomplete`, `@codemirror/merge`

### Schema Management

Schemas are always fetched from the connected cluster's `/openapi/v3` endpoint — never bundled. This ensures correctness across different k8s versions and CRDs.

Caching strategy:
- Schemas cached to disk, keyed by `{clusterUID}:{serverVersion}`
- On connect: compare cluster server version to cached version
- Cache hit: use cached schemas instantly
- Cache miss: fetch from cluster, cache for next time
- CRD schemas fetched lazily on first edit of that resource type, also cached
- Cache invalidated when cluster version changes (upgrade/downgrade)

### Disconnect Indicator

When a cluster connection is lost:
- The app header shifts to a pulsing red glow/shadow
- The cluster name badge turns red
- Tooltip shows details ("Connection lost to cluster X — reconnecting...")
- Clears automatically on reconnection

No toasts or banners — the header is always visible and non-intrusive.

---

## Event Namespacing

All Wails events use a strict namespacing scheme to prevent cross-cluster leaks:

```
watch:{clusterID}:{gvr}:{namespace}     — resource watch updates
status:{clusterID}:connection            — connection state changes
portforward:{clusterID}:{forwardID}      — port-forward state changes
```

Terminal and log streams do NOT use Wails events — they go over Fiber WebSocket with session/stream IDs embedded in the URL path.

---

## Plugin Architecture (v2)

> Full specification in [PLUGIN_ARCHITECTURE.md](PLUGIN_ARCHITECTURE.md).

**Summary:** Plugins extend Klados with custom resource views, enrichers, sidebar entries, commands, and detail panels. Two optional halves — a Wasm backend (enrichers, data logic via wazero) and Svelte UI bundles (frontend components) — described by a JSON Schema-validated manifest.

Key design decisions:
- **Language agnostic** — any WASI preview 1 language (Go first-class, TinyGo optimization path, Rust/C/etc. supported)
- **Strict isolation** — undeclared capabilities are structurally absent from the plugin context, not just permission-checked
- **One packaging pipeline** — directory (dev), `.oci.tar.gz` (sideload), OCI registry (production) all decompose to the same runtime format
- **Hot reload** — fsnotify watches trigger full lifecycle teardown and re-initialization
- **JSON Schema as source of truth** — manifest, host API, and plugin context types all codegen'd for Go and TypeScript

Dependencies: `tetratelabs/wazero` (Wasm), `santhosh-tekuri/jsonschema/v6` (validation), `fsnotify/fsnotify` (hot reload), `omissis/go-jsonschema` + `json-schema-to-typescript` (codegen), `oras.land/oras-go` (OCI distribution, future).

---

## Directory Structure

```
klados/
├── main.go                          # Wails app entry point
├── internal/
│   ├── cluster/                     # Cluster manager, connection lifecycle
│   │   ├── manager.go
│   │   ├── connection.go
│   │   └── auth.go
│   ├── resource/                    # Generic resource engine
│   │   ├── engine.go
│   │   └── descriptors.go           # Built-in resource descriptors
│   ├── watcher/                     # Watch/informer lifecycle
│   │   ├── manager.go
│   │   └── metadata.go              # Metadata-only informer helpers
│   ├── logs/                        # Log streaming over WebSocket
│   │   └── streamer.go
│   ├── exec/                        # Terminal exec sessions over WebSocket
│   │   └── manager.go
│   ├── portforward/                 # Port-forward manager (two-layer: discovery + tunnel)
│   │   ├── manager.go
│   │   ├── discovery.go             # Pod tracking via owner refs / label selectors
│   │   └── tunnel.go                # SPDY tunnel lifecycle and auto-reconnect
│   ├── streaming/                   # Fiber WebSocket server
│   │   ├── server.go
│   │   └── auth.go
│   ├── plugin/                      # Plugin system (v2)
│   │   ├── loader.go               # Scan dirs, validate manifest, load
│   │   ├── registry.go             # Track loaded plugins, extension points
│   │   ├── host_api.go             # wazero host functions
│   │   ├── permissions.go          # Manifest permission enforcement
│   │   ├── wasm_runtime.go         # wazero module lifecycle
│   │   ├── enricher_adapter.go     # Plugin → resource.Enricher bridge
│   │   ├── storage.go              # Per-plugin key-value store
│   │   └── watcher.go              # fsnotify hot reload
│   └── services/                    # Wails-bound service structs
│       ├── cluster.go
│       ├── resource.go
│       ├── log.go
│       ├── exec.go
│       ├── portforward.go
│       ├── editor.go
│       └── schema.go
├── frontend/
│   ├── src/
│   │   ├── lib/
│   │   │   ├── components/          # Reusable UI components
│   │   │   │   ├── Header.svelte
│   │   │   │   ├── Sidebar.svelte
│   │   │   │   ├── TabBar.svelte
│   │   │   │   ├── ResourceList.svelte
│   │   │   │   ├── ResourceDetail.svelte
│   │   │   │   ├── Terminal.svelte
│   │   │   │   ├── LogViewer.svelte
│   │   │   │   ├── YAMLEditor.svelte
│   │   │   │   ├── CommandPalette.svelte
│   │   │   │   └── ...
│   │   │   ├── stores/              # Svelte stores
│   │   │   │   ├── cluster.svelte.ts
│   │   │   │   ├── resource.svelte.ts
│   │   │   │   └── session.svelte.ts
│   │   │   ├── registry/            # Resource descriptor registry
│   │   │   │   ├── index.ts
│   │   │   │   ├── pods.ts
│   │   │   │   ├── deployments.ts
│   │   │   │   └── ...
│   │   │   ├── plugins/             # Plugin frontend system (v2)
│   │   │   │   ├── loader.ts       # Dynamic import() of UI bundles
│   │   │   │   ├── slots.ts        # Slot registry
│   │   │   │   ├── context.ts      # PluginContext construction
│   │   │   │   └── permissions.ts  # Frontend permission checking
│   │   │   └── wails/               # Wails bridge helpers
│   │   │       ├── bindings.ts
│   │   │       └── events.ts
│   │   ├── routes/                  # Page components
│   │   │   ├── ClusterChooser.svelte
│   │   │   ├── ResourceListPage.svelte
│   │   │   ├── ResourceDetailPage.svelte
│   │   │   └── ...
│   │   └── App.svelte
│   ├── package.json
│   └── vite.config.ts
├── schemas/                         # JSON Schema definitions (plugin API)
│   ├── manifest.v1.json
│   ├── host_api.v1.json
│   └── plugin_context.v1.json
├── examples/
│   └── plugin-node-annotator/       # Example plugin (Go + TinyGo)
├── FEATURES.md
├── ARCHITECTURE.md
├── PLUGIN_ARCHITECTURE.md
└── CLAUDE.md
```

---

## Keyboard Shortcuts

### Focus-Aware Shortcut System

Global shortcuts (Ctrl+K for command palette, Ctrl+Shift+L for logs, etc.) must not interfere with terminal or editor input. The system uses a focus mode model:

- **Normal mode** (default) — global shortcuts are active
- **Terminal capture mode** — all keyboard input passes through to the terminal process, except a designated escape chord (e.g., Ctrl+Shift+Escape) to return to normal mode
- **Editor capture mode** — editor shortcuts take priority (Ctrl+F for find in editor, not global search), global shortcuts still work for non-conflicting chords

Focus mode is tracked in a global store. Components set the mode on focus/blur:
- Terminal panel: sets capture mode on focus, clears on blur
- YAML editor: sets editor mode on focus, clears on blur
- Everything else: normal mode

The shortcut handler checks the current mode before dispatching. Shortcuts are defined in a central registry (enables future keybinding customization).

---

## Multi-Cluster Connectivity

Clusters are connected **simultaneously**, not one-at-a-time. Each cluster maintains its own independent:
- Connection state and health monitoring
- Watch manager and informer set
- Auth token lifecycle
- Namespace selection

The UI supports viewing resources from different clusters in different tabs. The header shows the active cluster for the current tab, with a cluster switcher for navigation.

Memory consideration: each connected cluster consumes resources for its watches. Clusters that have no active tabs viewing their resources should have their watches suspended (with a grace period) to reduce memory pressure.

---

## API Resource Discovery

On cluster connect, the backend queries the API server's discovery endpoint to enumerate all available resource types (core + CRDs).

Flow:
1. Backend calls discovery API → gets list of APIResources (GVR, namespaced, verbs, short names)
2. Caches the result per cluster (refreshed on reconnect or manually)
3. Pushes the resource list to the frontend via Wails event `discovery:{clusterID}:resources`
4. Frontend builds the sidebar tree and command palette resource list from this data
5. CRDs appearing/disappearing are detected via a watch on the CRD resource type itself

The sidebar groups resources by category (Workloads, Networking, Config, Storage, RBAC, Custom Resources). Category assignment is determined by the descriptor registry for known types; unknown CRDs go under "Custom Resources."

---

## App Lifecycle & State Persistence

### Initial Launch

The app always opens to the **cluster list view**, regardless of how many clusters are detected. This gives the user a clear starting point and an overview of all available clusters with their connection status.

Auto-detection sources (scanned on startup):
- `~/.kube/config`
- `$KUBECONFIG` env var (supports colon-separated multiple files)

### Returning Users — Session Restore

On subsequent launches, the app restores the previous session state:
- Which clusters were connected
- Active namespace per cluster
- Open tabs (resource type, namespace, resource name)
- Active tab selection
- Sidebar collapsed state
- Window dimensions and position
- Column widths per resource type

Session state is persisted to `$XDG_STATE_HOME/klados/session.json` (separate from config preferences). Written on every meaningful state change (debounced).

Config preferences (`$XDG_CONFIG_HOME/klados/config.json`) store user settings: theme, keybindings, default namespace preferences, port-forward favorites.

Schema cache remains at `$XDG_CACHE_HOME/klados/schemas/`.

---

## Error Handling & Edge Cases

### YAML Apply Conflicts

When a user edits a resource and applies, the k8s API may return 409 Conflict if the resource was modified since it was fetched. Handling:
- Display a clear error indicating the conflict
- Where possible, surface what field(s) conflicted
- Offer the user a "Refresh" action to re-fetch the latest version and re-apply their changes
- Do NOT silently force-apply or auto-merge

### Exec Session Expiry

Auth tokens (exec-based, OIDC) can expire during long sessions. If an active terminal session drops:
- Surface a "Reconnect" button in the terminal panel
- Do NOT auto-reconnect — the session may still be alive server-side, and the user reconnects only if needed
- Log streams can auto-reconnect since they are stateless reads

### Port-Forward Resilience

Inspired by Tilt's two-layer architecture, port-forwarding is separated into **discovery** (which pod?) and **tunneling** (maintain the connection).

#### Discovery Layer

The discovery layer continuously tracks the best pod for a given forward target. It uses two strategies:

**Owner-reference tracking (default):** Follow the UID chain from the workload object (Deployment → ReplicaSet → Pod, StatefulSet → Pod). Handles rolling updates, scaling, and crashes automatically.

**Label selector tracking (fallback):** For CRDs or custom workloads without standard owner references, fall back to label selector matching against the service or workload selectors.

**Best-pod selection** when multiple candidates exist:
1. Prefer pods in **Running** phase
2. Among running pods, prefer those that are **Ready** (readiness probes passing)
3. Tiebreak by creation timestamp (newest)

This ensures that during rolling updates, the forward stays on the old pod until the new one is actually ready and serving traffic.

#### Tunnel Layer

The tunnel layer manages the actual SPDY connection via `client-go/tools/portforward`. It handles:

- **Transport-level auto-reconnect:** Network blips, API server restarts, or SPDY drops are detected and reconnected automatically for ALL forward types — the pod is still alive, only the tunnel broke.
- **Pod-level reconnect:** When the discovery layer selects a new best pod, the old tunnel is torn down and a new one is opened.

#### Target Intent & Behavior Matrix

The port-forward data model stores the **target intent** to determine discovery strategy:

| Target Intent | Discovery | Pod Death | Transport Drop |
|---|---|---|---|
| Raw pod name (generated, e.g. from Deployment) | None — direct pod | Notify user, no reconnect | Auto-reconnect tunnel |
| StatefulSet pod (stable name) | Owner-reference tracking | Auto-reconnect to same pod name | Auto-reconnect tunnel |
| Service | Label selector from service spec | Auto-select new best pod | Auto-reconnect tunnel |
| Deployment/workload | Owner-reference chain | Auto-select new best pod | Auto-reconnect tunnel |

UI shows forward status: **Active**, **Reconnecting...** (transport drop or pod swap), **Failed** (no viable pod found).

---

## Testing Strategy

### Go (target: 80%+ coverage)

**Unit tests** — fast, no external dependencies:
- Resource engine, enrichers, descriptor evaluation
- Watch manager lifecycle logic
- Port-forward manager state machine
- Auth token refresh logic
- Use `k8s.io/client-go/dynamic/fake` for fake dynamic client

**Integration tests** — real API server behavior:
- Use `sigs.k8s.io/controller-runtime/pkg/envtest` to spin up a real kube-apiserver + etcd locally
- Tests for: watch semantics, apply conflicts (409), RBAC, resource validation, schema fetching
- Run in CI, skippable locally with build tag

### Frontend (Vitest + Svelte testing library)

- Component tests for ResourceList, ResourceDetail, Terminal, YAMLEditor
- Store tests for ClusterStore, ResourceStore, SessionStore
- CEL expression evaluation tests (via cel-js)
- Mock Wails bindings and events

### No E2E initially

Too brittle for MVP. Rely on unit + integration coverage and manual testing.
