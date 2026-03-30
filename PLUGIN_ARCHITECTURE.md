# Klados — Plugin Architecture

> Detailed specification for the Klados plugin system (v2).

---

## Overview

Klados plugins extend the application with custom resource views, enrichers, sidebar entries, commands, and detail panels. Plugins are composed of two optional halves — a **Wasm backend** (enrichers, data logic) and **Svelte UI bundles** (frontend components) — described by a **JSON Schema-validated manifest**.

The design prioritizes:
- **Language agnosticism** — any language that compiles to WASI preview 1 can produce a plugin Wasm module (Go is the first-class target for v2, with TinyGo as an optimization path)
- **Strict isolation** — plugins only access capabilities they declare in their manifest; undeclared capabilities are structurally absent, not just permission-checked
- **Hot reload** — file system changes trigger full plugin lifecycle teardown and re-initialization, enabling rapid plugin development
- **One packaging pipeline** — the same OCI artifact structure works for registry distribution, local sideloading (`.oci.tar.gz`), and development (directory)

---

## Manifest

Every plugin has a `manifest.json` at its root. The manifest is validated against a versioned JSON Schema (`manifest.v1.json`).

```jsonc
{
  "schemaVersion": 1,
  "name": "cert-manager",
  "version": "1.0.0",
  "displayName": "Cert-Manager",
  "description": "Certificate lifecycle management for Kubernetes",
  "minHostVersion": "2.0.0",

  "permissions": {
    "resources": [
      {
        "group": "cert-manager.io",
        "version": "v1",
        "resource": "certificates",
        "verbs": ["list", "get", "watch"]
      },
      {
        "group": "cert-manager.io",
        "version": "v1",
        "resource": "issuers",
        "verbs": ["list", "get", "watch"]
      }
    ],
    "logs": false,
    "exec": false,
    "storage": true,
    "events": true,
    "wasi": {
      "clock": true,
      "filesystem": false,
      "network": false,
      "env": false
    }
  },

  "extensions": {
    "descriptors": [
      "descriptors/certificate.yaml",
      "descriptors/issuer.yaml"
    ],
    "enrichers": {
      "wasm": "plugin.wasm",
      "gvrs": [
        "cert-manager.io.v1.certificates",
        "cert-manager.io.v1.issuers"
      ]
    },
    "sidebar": [
      {
        "category": "Security",
        "label": "Certificates",
        "gvr": "cert-manager.io.v1.certificates",
        "icon": "shield-check"
      }
    ],
    "detailTabs": [
      {
        "gvr": "cert-manager.io.v1.certificates",
        "id": "cert-status",
        "label": "Certificate Status",
        "component": "ui/CertStatus.js"
      }
    ],
    "commands": [
      {
        "id": "cert-manager.renew",
        "label": "Renew Certificate",
        "icon": "refresh-cw"
      }
    ]
  }
}
```

### Versioning

- **`schemaVersion`** — governs the shape of the manifest itself (fields, extension point types, permission model). Bumped when incompatible manifest changes are introduced. The host rejects manifests with an unsupported schema version and notifies the user.
- **`minHostVersion`** — semver. The host checks its own version against this field on load. If the host is older, the plugin is not loaded and the user is notified.
- **Plugin SDK versioning** — the SDK follows semver independently. Plugins are NOT required to match exact SDK versions — the host API schema version is the contract. Plugins built against an older compatible SDK version continue to work.

---

## Packaging & Distribution

All three distribution paths decompose to the same **directory format** at runtime:

```
plugins/
└── cert-manager/
    ├── manifest.json
    ├── plugin.wasm
    ├── descriptors/
    │   ├── certificate.yaml
    │   └── issuer.yaml
    └── ui/
        ├── CertStatus.js
        └── IssuerPanel.js
```

The plugin loader only ever reads directories. Install commands extract into this format.

### Directory (Development)

Edit files directly in `~/.config/klados/plugins/{name}/`. Hot reload watches for changes and triggers the full plugin lifecycle (disable → shutdown → replace → init → enable).

### OCI Tar (Sideloading)

A `.oci.tar.gz` file (default, gzip-compressed) or `.oci.tar` (uncompressed) containing an OCI Image Layout:

```
cert-manager-1.0.0.oci.tar.gz
└── OCI Image Layout
    ├── oci-layout                  {"imageLayoutVersion": "1.0.0"}
    ├── index.json                  manifest digest pointer
    └── blobs/sha256/
        ├── <manifest>              references layers below
        ├── <config>                plugin manifest.json
        ├── <layer-wasm>            plugin.wasm
        └── <layer-ui>              ui/ directory as tar
```

On import, the host detects compression by checking gzip magic bytes (`1f 8b`) and handles both formats transparently.

### OCI Registry (Production)

Same artifact structure, pushed to any OCI-compliant registry (GHCR, Docker Hub, ECR, self-hosted) via ORAS.

**Media types (versioned):**

| Layer | Media Type |
|---|---|
| Config | `application/vnd.klados.plugin.manifest.v1+json` |
| Wasm binary | `application/vnd.klados.plugin.wasm.v1` |
| UI bundle | `application/vnd.klados.plugin.ui.v1+tar+gzip` |
| Descriptors | `application/vnd.klados.plugin.descriptors.v1+tar+gzip` |

Media type versions are independent from manifest schema versions. Media type versions change when the packaging structure changes; manifest schema versions change when manifest content changes.

### CLI Commands

```
klados plugin pack ./cert-manager/                      → cert-manager-1.0.0.oci.tar.gz
klados plugin pack --no-compress ./cert-manager/         → cert-manager-1.0.0.oci.tar

klados plugin install ./cert-manager-1.0.0.oci.tar.gz   → extracts to plugins/cert-manager/
klados plugin install ./cert-manager-1.0.0.oci.tar       → also works
klados plugin install oci://ghcr.io/foo/cert-manager:v1  → pulls + extracts

klados plugin push ./cert-manager/ oci://ghcr.io/foo/cert-manager:v1
```

---

## Wasm Runtime (Backend)

### Host Environment

Plugins run in **wazero** (pure Go, no CGO) with WASI preview 1 support. Each plugin gets its own isolated Wasm module instance.

### WASI Capabilities

Default posture: **deny all**. The plugin manifest explicitly requests WASI capabilities under `permissions.wasi`:

| Capability | Default | Notes |
|---|---|---|
| `clock` | Denied | Wall clock and monotonic clock access |
| `filesystem` | Denied | Scoped to plugin data directory if granted |
| `network` | Denied | Raw socket access (k8s access goes through host API) |
| `env` | Denied | Environment variable access for plugin config injection |

**Stdout/stderr** are always captured and routed to the host's slox (slog) logger with a `plugin={name}` group label:

```
INFO plugin=cert-manager msg="enriching certificate" gvr="cert-manager.io.v1.certificates"
```

### Calling Convention

The host-plugin boundary uses a single dispatch function with JSON serialization. This keeps the Wasm import surface minimal, making it easy for any language to bind.

**Host functions (imported by plugin):**

```
host_call(method_ptr, method_len, req_ptr, req_len) → (resp_ptr, resp_len)
host_log(level, msg_ptr, msg_len)
host_alloc(size) → ptr
host_free(ptr, size)
```

**Plugin functions (exported by plugin):**

```
plugin_init() → i32                                        // 0 = success
plugin_enrich(gvr_ptr, gvr_len, obj_ptr, obj_len) → (ptr, len)
plugin_destroy()                                           // cleanup before unload
plugin_alloc(size) → ptr
plugin_free(ptr, size)
```

All data crossing the boundary is **JSON**. The host API request/response types are defined by JSON Schema (`host_api.v1.json`), enabling codegen of typed structures in any guest language.

### Host API Methods

Dispatched via `host_call` with a method name string:

| Method | Description | Permission Gate |
|---|---|---|
| `k8s.list` | List resources by GVR | `permissions.resources[].verbs` contains `list` for the requested GVR |
| `k8s.get` | Get a single resource | `permissions.resources[].verbs` contains `get` |
| `k8s.watch` | Subscribe to resource changes | `permissions.resources[].verbs` contains `watch` |
| `k8s.create` | Create a resource | `permissions.resources[].verbs` contains `create` |
| `k8s.update` | Update a resource | `permissions.resources[].verbs` contains `update` |
| `k8s.delete` | Delete a resource | `permissions.resources[].verbs` contains `delete` |
| `storage.get` | Read plugin-local storage | `permissions.storage` is `true` |
| `storage.set` | Write plugin-local storage | `permissions.storage` is `true` |
| `storage.delete` | Delete plugin-local storage key | `permissions.storage` is `true` |
| `logs.stream` | Start a log stream | `permissions.logs` is `true` |
| `exec.open` | Open an exec session | `permissions.exec` is `true` |
| `event.subscribe` | Subscribe to Wails events | `permissions.events` is `true` |

Methods not granted by the manifest are rejected with a structured error. The rejection is logged at WARN level with the `plugin={name}` group.

### Permission Enforcement

Permissions are enforced at two levels:

1. **Structural (capability level)** — if a capability is not declared (e.g., `permissions.logs: false`), the corresponding `host_call` methods are not registered for that module. Calling them returns "method not available."
2. **Fine-grained (verb + GVR level)** — for capabilities that are present (like `k8s`), each call checks the verb and GVR against the manifest's resource permission list.

### Enricher Integration

Plugin enrichers integrate into the existing `resource.EnricherRegistry`. When a plugin declares enricher GVRs, the host registers a `PluginEnricher` adapter that:

1. Serializes the `unstructured.Unstructured` object to JSON
2. Calls the plugin's `plugin_enrich` export
3. Deserializes the result back to `unstructured.Unstructured`
4. Returns the enriched object into the existing pipeline

Multiple enrichers for the same GVR (built-in + plugins, or multiple plugins) are **chained in load order**. Each enricher sees the output of the previous one. If two enrichers write the same field path, last writer wins — the host logs a warning identifying which plugins conflict.

### Descriptor Registration

Plugin descriptors (YAML files referenced in `extensions.descriptors`) are loaded and merged into the existing `resource.Registry`. They follow the same format as built-in descriptors (columns with CEL expressions, detail panels, actions). Plugin descriptors for a GVR that already has a built-in descriptor **extend** it (add columns, panels) rather than replace it.

---

## Frontend Plugin System

### Dependency Sharing

Plugin UI bundles are pre-built Svelte ES modules. To avoid version conflicts and bundle bloat, plugins mark shared dependencies as **external** in their Vite config:

```javascript
// Plugin's vite.config.js
export default {
  build: {
    rollupOptions: {
      external: ['svelte', 'svelte/internal', '@klados/plugin-ui'],
    }
  }
}
```

The host application provides these at runtime. `@klados/plugin-ui` is a **published npm package** containing:
- Tailwind CSS design tokens (colors, spacing, typography)
- Shared UI primitives (buttons, badges, tables, inputs) built on Bits UI
- TypeScript types for the plugin context API
- Lucide icon re-exports

This ensures visual consistency between host and plugin UI without plugins bundling their own copies.

### Component Mounting

Plugin UI components are loaded via dynamic `import()` and mounted using Svelte 5's `mount()` API:

```typescript
import { mount, unmount } from 'svelte';

// Load plugin component
const module = await import(`/plugins/cert-manager/ui/CertStatus.js`);

// Mount into a slot element
const instance = mount(module.default, {
    target: slotElement,
    props: {
        resource: obj,
        ctx: pluginContext
    }
});

// Update props reactively
instance.$set({ resource: newObj });

// Teardown on plugin unload or slot unmount
unmount(instance);
```

### Plugin Context (Frontend)

Plugin UI components receive a **`PluginContext`** object as a prop. This is the single interface between plugin UI and the host. The context is **dynamically constructed from the manifest** — capabilities not declared in the manifest are structurally absent from the context object.

```typescript
interface PluginContext {
    // Always present
    cluster: { name: string; version: string };
    namespace: string;

    // Only present if permissions.resources declared
    k8s?: {
        list(gvr: string, ns?: string, opts?: ListOpts): Promise<any[]>;
        get(gvr: string, ns: string, name: string): Promise<any>;
        watch(gvr: string, ns: string, callback: WatchCallback): Unsubscribe;
    };

    // Only present if permissions.logs: true
    logs?: {
        stream(pod: string, ns: string, container: string, opts?: LogOpts): ReadableStream<string>;
    };

    // Only present if permissions.exec: true
    exec?: {
        open(pod: string, ns: string, container: string): TerminalHandle;
    };

    // Only present if permissions.storage: true
    storage?: {
        get(key: string): Promise<string | null>;
        set(key: string, value: string): Promise<void>;
        delete(key: string): Promise<void>;
    };

    // Only present if permissions.events: true
    subscribe?(event: string, callback: (data: any) => void): Unsubscribe;
}
```

Context construction:

```typescript
function createPluginContext(manifest: PluginManifest, host: HostServices): PluginContext {
    const ctx: Partial<PluginContext> = {
        cluster: { name: host.clusterName, version: host.version },
        namespace: host.activeNamespace,
    };

    if (manifest.permissions.resources?.length) {
        ctx.k8s = {
            list: (gvr, ns, opts) => {
                assertGVRPermission(manifest, gvr, 'list');
                return host.k8s.list(gvr, ns, opts);
            },
            get: (gvr, ns, name) => {
                assertGVRPermission(manifest, gvr, 'get');
                return host.k8s.get(gvr, ns, name);
            },
            watch: (gvr, ns, cb) => {
                assertGVRPermission(manifest, gvr, 'watch');
                return host.k8s.watch(gvr, ns, cb);
            },
        };
    }

    if (manifest.permissions.logs)    ctx.logs = host.logs;
    if (manifest.permissions.exec)    ctx.exec = host.exec;
    if (manifest.permissions.storage) ctx.storage = scopedStorage(manifest.name, host.storage);
    if (manifest.permissions.events)  ctx.subscribe = host.subscribe;

    return Object.freeze(ctx as PluginContext);
}
```

`Object.freeze` prevents plugins from monkey-patching the context to bypass checks or attach undeclared capabilities.

### UI Slots

Plugin components are mounted into designated slots throughout the application:

| Slot | Location | Props Provided |
|---|---|---|
| **Sidebar entry** | Sidebar resource tree | `ctx` |
| **Detail tab** | Resource detail tab bar | `resource, ctx` |
| **Overview field** | Resource detail overview panel | `resource, ctx` |
| **Command action** | Command palette | `ctx` |
| **Status bar widget** | Bottom status bar | `ctx` |
| **Resource list column** | Custom columns in resource lists | `resource, ctx` |
| **Context menu item** | Resource right-click menu | `resource, ctx` |
| **Header widget** | Header bar area | `ctx` |

---

## Lifecycle

### Plugin Load Sequence

```
1. Scan plugins directory for manifest.json files
2. Validate manifest against JSON Schema (manifest.v1.json)
3. Check schemaVersion compatibility
4. Check minHostVersion compatibility
5. Load and instantiate Wasm module (if declared)
   a. Configure WASI capabilities from manifest
   b. Register host functions (scoped by permissions)
   c. Call plugin_init(), check return code
6. Load and validate descriptors (if declared)
7. Register enrichers into EnricherRegistry (if declared)
8. Register sidebar entries, commands, detail tabs
9. Notify frontend of new extension points
10. Plugin is now active
```

### Plugin Unload Sequence

```
1. Unmount all plugin UI components (unmount() each instance)
2. Unregister sidebar entries, commands, detail tabs
3. Unregister enrichers from EnricherRegistry
4. Call plugin_destroy() on Wasm module
5. Close Wasm module instance
6. Remove plugin from registry
```

### Hot Reload

File system changes in the plugin directory trigger a debounced (200ms) reload cycle:

```
fsnotify event → debounce 200ms → disable → shutdown → replace → init → enable
```

The loader uses `fsnotify/fsnotify` to watch the plugin root directory and each plugin's subdirectories recursively. When a new subdirectory is created, a watch is registered for it. The full lifecycle (unload → load) is executed — no partial updates.

### Lifecycle Events

Plugins receive lifecycle callbacks via exported Wasm functions:

| Export | When Called |
|---|---|
| `plugin_init()` | After Wasm instantiation, before any enricher calls |
| `plugin_destroy()` | Before Wasm module close, during unload |

Additionally, plugins that subscribe to events (`permissions.events: true`) can listen for cluster lifecycle events:

| Event | Payload |
|---|---|
| `cluster:connected` | `{ name, version, platform }` |
| `cluster:disconnected` | `{ name, reason }` |
| `namespace:changed` | `{ namespace }` |

---

## Error Handling

### Wasm Errors

wazero catches Wasm traps (panics) — a crashing plugin cannot take down the host. When a plugin Wasm module errors:

1. **Toast notification** shown to the user with the plugin name and error summary
2. **Plugin UI indicator** marks the plugin as errored in the plugin management UI
3. **Console/log output** with full error details under the `plugin={name}` slog group
4. **Plugin is auto-disabled** — no automatic retry. The user can manually reload from the plugin management UI.

### Frontend Errors

JavaScript errors in plugin UI components are caught by the host's existing error boundary. Errors bubble to the Go logger and browser console through the existing mechanism.

### Permission Violations

Calls to denied methods (structural absence or fine-grained verb/GVR mismatch) return a typed error to the plugin. The host logs the violation at WARN level. The plugin is NOT disabled for permission violations — it may be handling the error gracefully.

---

## Conflict Resolution

### Enricher Conflicts

Multiple enrichers for the same GVR are chained in load order. If two plugins write the same field path on an object, last writer wins. The host detects this by comparing the object before and after each enricher, and logs a warning:

```
WARN plugin=cert-manager msg="enricher field conflict" gvr="core.v1.pods" field="status.securityScore" other_plugin="pod-security"
```

### Sidebar Conflicts

If two plugins register sidebar entries with the same label, both are shown. No intervention needed — the plugin name disambiguates in tooltips.

### Command Conflicts

If two plugins register commands with conflicting keyboard shortcuts, the first-loaded plugin wins. A warning is shown to the user:

```
Plugin "cert-manager" shortcut Ctrl+Shift+R conflicts with "pod-security" — using cert-manager's binding.
```

---

## Plugin Storage

All plugin storage is Go-managed, persisted to disk following XDG conventions. No browser localStorage is used.

**Backend (Wasm):** `host_call("storage.get", ...)` / `host_call("storage.set", ...)` — routed to a file-backed key-value store at `$XDG_DATA_HOME/klados/plugins/{name}/storage.json`.

**Frontend (UI):** `ctx.storage.get(key)` / `ctx.storage.set(key, value)` — calls Wails bindings that delegate to the same Go-managed store.

Both paths access the same underlying store, ensuring consistency between Wasm and UI reads/writes for the same plugin.

---

## Plugin UI for Users

The plugin management UI displays:

- Installed plugins with name, version, status (active, disabled, errored)
- **Permission summary** — clearly lists what each plugin has access to (resource GVRs + verbs, logs, exec, storage, WASI capabilities)
- Enable/disable toggle
- Manual reload button (for errored plugins)
- Uninstall action

---

## Schema & Codegen

All types crossing the host-plugin boundary are defined in **JSON Schema** (draft 2020-12) and used as the single source of truth for codegen:

| Schema File | Purpose |
|---|---|
| `schemas/manifest.v1.json` | Plugin manifest structure |
| `schemas/host_api.v1.json` | Host API request/response types |
| `schemas/plugin_context.v1.json` | Frontend PluginContext interface |

**Codegen pipeline:**

| Target | Tool | Output |
|---|---|---|
| Go structs | `omissis/go-jsonschema` | `internal/plugin/types/` |
| TypeScript types | `json-schema-to-typescript` | `frontend/src/lib/plugins/types/` |
| Plugin SDK types | `omissis/go-jsonschema` | `klados-plugin-sdk/types/` |

Codegen runs via `mise run generate:plugin-types`.

**Validation at runtime:** `santhosh-tekuri/jsonschema/v6` validates plugin manifests on load and host API payloads at the boundary.

---

## Dependencies

### Host (Go Backend)

| Purpose | Library |
|---|---|
| Wasm runtime | `tetratelabs/wazero` |
| JSON Schema validation | `santhosh-tekuri/jsonschema/v6` |
| File watching | `fsnotify/fsnotify` |
| OCI distribution (future) | `oras.land/oras-go` |
| Go codegen | `omissis/go-jsonschema` |

### Host (Frontend)

| Purpose | Library |
|---|---|
| TS codegen | `json-schema-to-typescript` |
| Component mounting | Svelte 5 `mount()` / `unmount()` |
| Plugin UI library | `@klados/plugin-ui` (published npm package) |

### Plugin SDK (Guest)

| Purpose | Library |
|---|---|
| Go module | `github.com/Vilsol/klados-plugin-sdk` |
| npm package | `@klados/plugin-sdk` (TypeScript types + helpers for UI) |

### Plugin Compilation

| Toolchain | Target | Binary Size | Notes |
|---|---|---|---|
| Go std (`GOOS=wasip1 GOARCH=wasm`) | First-class | ~5-15 MB | Zero-friction, full language support |
| TinyGo (`--target=wasip1`) | Supported | ~100-500 KB | Smaller binaries, subset of Go stdlib |
| Any WASI-targeting compiler | Supported | Varies | Rust, C, AssemblyScript, etc. — host doesn't care |

---

## Directory Structure

```
klados/
├── internal/
│   └── plugin/
│       ├── loader.go            # Scan plugin dirs, validate manifest, load
│       ├── registry.go          # Track loaded plugins, extension points
│       ├── host_api.go          # wazero host functions (k8s, storage, events)
│       ├── permissions.go       # Enforce manifest permissions on every call
│       ├── wasm_runtime.go      # wazero module instantiation, lifecycle
│       ├── enricher_adapter.go  # PluginEnricher → resource.Enricher bridge
│       ├── storage.go           # File-backed per-plugin key-value store
│       ├── watcher.go           # fsnotify hot reload with recursive subdirs
│       └── types/               # Generated from JSON Schema
│           ├── manifest.go
│           ├── request.go
│           └── response.go
├── schemas/
│   ├── manifest.v1.json
│   ├── host_api.v1.json
│   └── plugin_context.v1.json
├── frontend/src/lib/plugins/
│   ├── loader.ts                # Dynamic import() of plugin UI bundles
│   ├── slots.ts                 # Slot registry (sidebar, tabs, commands, etc.)
│   ├── context.ts               # PluginContext construction from manifest
│   ├── permissions.ts           # Frontend GVR permission checking
│   └── types/                   # Generated from JSON Schema
│       ├── manifest.ts
│       └── context.ts
└── examples/
    └── plugin-node-annotator/
        ├── main.go
        ├── go.mod
        ├── manifest.json
        ├── mise.toml             # build:go, build:tinygo, test (both)
        └── ui/
            └── NodeAnnotation.svelte
```

---

## Example Plugin

An example plugin (`plugin-node-annotator`) lives in `examples/` and serves as the template for plugin developers. It:

- Compiles with both standard Go (`GOOS=wasip1`) and TinyGo (`--target=wasip1`) to prove dual support
- Includes a `mise.toml` with `build:go`, `build:tinygo`, and `test` tasks
- The `test` task loads both `.wasm` binaries into wazero and asserts identical enricher output
- Demonstrates: manifest structure, enricher implementation, UI component, storage usage, event subscription
