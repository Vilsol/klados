# Klados — Plugin System Phase Prompts

Prompts for starting each plugin implementation phase in a new session. The plugin system is the first v2 feature, building on the completed MVP (Phases 1-6).

---

## Phase P1 — Schema & Loader Foundation

### Context

I'm starting Phase P1 of the klados plugin system. The MVP is complete — the app has cluster management, resource browsing/editing, logs, terminal, port-forwarding, and all core UX. This is the first v2 phase.

Read these files first (in order):
- `PLUGIN_ARCHITECTURE.md` — full plugin system spec (manifest, Wasm runtime, host API, frontend, packaging, lifecycle, permissions, error handling)
- `ARCHITECTURE.md` — overall app architecture, especially the directory structure (look for `internal/plugin/` and `frontend/src/lib/plugins/`)
- `CLAUDE.md` — build/test commands, conventions, GVR format, three-stage rendering pipeline, existing package responsibilities

Skim these for existing patterns to follow:
- `internal/resource/builtin.go` — how built-in descriptors are registered (plugin descriptors follow the same format)
- `internal/resource/descriptor.go` — the `Descriptor` and `Column` types
- `internal/resource/enricher.go` — the `Enricher` interface (plugin enrichers will adapt to this)
- `internal/resource/engine.go` — `ResourceEngine` and `Registry` that plugin descriptors merge into

### What exists

- `internal/resource/` — `Registry` holds descriptors, `EnricherRegistry` holds enrichers, `ResourceEngine` does CRUD. All keyed by GVR string.
- `frontend/src/lib/registry/index.ts` — `DescriptorRegistry` loads descriptors from Go via `GetDescriptors()`, provides `get(gvr)` with fallback. CEL evaluation via `evalExpr()`.
- No plugin code exists yet. The `internal/plugin/` and `frontend/src/lib/plugins/` directories need to be created.

### Deliverables

1. **JSON Schema definitions** in `schemas/`:
   - `manifest.v1.json` — plugin manifest structure (see PLUGIN_ARCHITECTURE.md for the full manifest example). Include: `schemaVersion`, `name`, `version`, `displayName`, `description`, `minHostVersion`, `permissions` (resources, logs, exec, storage, events, wasi), `extensions` (descriptors, enrichers, sidebar, detailTabs, commands).
   - `host_api.v1.json` — host API request/response types for all `host_call` methods (k8s.list, k8s.get, storage.get, etc.)
   - `plugin_context.v1.json` — frontend PluginContext interface shape

2. **Codegen pipeline**:
   - `mise run generate:plugin-types` task that runs:
     - `omissis/go-jsonschema` → generates Go structs in `internal/plugin/types/`
     - `json-schema-to-typescript` → generates TS types in `frontend/src/lib/plugins/types/`
   - Generated types should have json tags, be exported, and include validation annotations where possible

3. **Plugin loader** (`internal/plugin/loader.go`):
   - Scan a configurable plugins directory (default: `$XDG_DATA_HOME/klados/plugins/`)
   - Find `manifest.json` in each subdirectory
   - Validate against `manifest.v1.json` using `santhosh-tekuri/jsonschema/v6`
   - Check `schemaVersion` compatibility (only v1 supported initially)
   - Check `minHostVersion` against the app's version
   - Load referenced descriptor YAML files (relative paths from manifest)
   - Return structured errors for validation failures (which field, why)

4. **Plugin registry** (`internal/plugin/registry.go`):
   - Track loaded plugins with metadata (name, version, status, manifest)
   - Register plugin descriptors into the existing `resource.Registry`
   - Register sidebar entries for the frontend to consume
   - Expose `GetPlugins()` and `GetPluginDescriptors()` for the frontend

5. **Wails service** (`internal/services/plugin.go`):
   - `PluginService` bound to Wails, exposes: `ListPlugins()`, `GetPluginDescriptors()`, `GetPluginSidebarEntries()`
   - Emit `plugins:loaded` Wails event after all plugins are scanned on startup

6. **Frontend integration**:
   - `frontend/src/lib/plugins/types/` — generated TS types
   - Extend the existing `DescriptorRegistry` to merge plugin descriptors (loaded via `GetPluginDescriptors()`)
   - Extend `Sidebar.svelte` to render plugin-registered sidebar entries under their declared categories

### Tests

- **Go unit tests** for the loader: valid manifest loads, invalid manifest rejected (missing required fields, bad schemaVersion, version mismatch), descriptor files loaded correctly, missing descriptor file errors gracefully
- **Go unit tests** for the registry: descriptors merge without clobbering built-ins, sidebar entries registered, duplicate plugin names rejected
- **Schema validation tests**: ensure the JSON Schemas themselves are valid and that example manifests pass validation
- **Frontend**: mock `PluginService` bindings, verify plugin descriptors merge into `DescriptorRegistry`, verify sidebar renders plugin entries

### Definition of done

Drop a directory into the plugins path with a `manifest.json` and `descriptors/*.yaml` → the app loads it on startup, the sidebar shows the plugin's entries, and resource lists render plugin-defined columns via CEL expressions. No Wasm module is needed — this phase proves the manifest pipeline and descriptor registration end-to-end.

### Known gotchas

- GVR format is dot-separated: `cert-manager.io.v1.certificates`. `ParseGVR()` splits from the right. See CLAUDE.md.
- Descriptor YAML format: check `internal/resource/builtin.go` `RegisterBuiltin()` for the exact column/panel/action shape. Plugin descriptors must match.
- Wails bindings: after adding `PluginService`, run `wails3 generate bindings`. Import generated TS with `.js` extension.
- The `$Create.Map` mock in `frontend/src/lib/__tests__/setup.ts` may need updating if the new bindings use map fields.

---

## Phase P2 — Wasm Runtime & Enrichers

### Context

I'm starting Phase P2 of the klados plugin system. Phase P1 is complete — JSON Schemas are defined, codegen pipeline works, the plugin loader validates manifests, and plugin descriptors/sidebar entries are registered and rendering in the app.

Read these files first (in order):
- `PLUGIN_ARCHITECTURE.md` — especially: Wasm Runtime section (calling convention, host API methods, permission enforcement, WASI capabilities, enricher integration)
- `ARCHITECTURE.md` — overall app architecture
- `CLAUDE.md` — conventions, especially logging (`slox.Info(ctx, ...)`) and the enricher pipeline

Review these for integration points:
- `internal/plugin/loader.go` — P1 loader (you'll extend it to instantiate Wasm modules)
- `internal/plugin/registry.go` — P1 registry (you'll register enricher adapters here)
- `internal/resource/enricher.go` — the `Enricher` interface your adapter must implement
- `internal/resource/enrichers/` — existing enricher implementations for reference

### What exists from P1

- `schemas/` — JSON Schema definitions for manifest, host API, plugin context
- `internal/plugin/types/` — generated Go types from schemas
- `internal/plugin/loader.go` — scans plugin dirs, validates manifests, loads descriptor YAMLs
- `internal/plugin/registry.go` — tracks loaded plugins, merges descriptors into `resource.Registry`
- `internal/services/plugin.go` — `PluginService` Wails service
- `frontend/src/lib/plugins/types/` — generated TS types
- Sidebar and descriptor registration working for static plugins (no Wasm)

### Deliverables

1. **Wasm runtime** (`internal/plugin/wasm_runtime.go`):
   - Use `tetratelabs/wazero` with WASI preview 1
   - Instantiate one module per plugin that declares `extensions.enrichers.wasm`
   - Configure WASI capabilities from `manifest.permissions.wasi` (default deny-all):
     - Clock: `wasi_snapshot_preview1.clock_time_get` etc.
     - Filesystem: scoped to plugin data dir if granted
     - Env: inject plugin config vars if granted
   - Capture stdout/stderr → route to slox logger with `plugin={name}` slog group
   - Call `plugin_init()` on instantiation, check return code (0 = success)
   - Call `plugin_destroy()` before module close
   - Handle Wasm traps: catch, log error, mark plugin as errored

2. **Host API dispatch** (`internal/plugin/host_api.go`):
   - Register `host_call(method_ptr, method_len, req_ptr, req_len) → (resp_ptr, resp_len)` as a wazero host function
   - Register `host_log(level, msg_ptr, msg_len)` — routes to slox with `plugin={name}` group
   - Register `host_alloc(size) → ptr` and `host_free(ptr, size)` — manage Wasm memory
   - Dispatch method names to handlers: `k8s.list`, `k8s.get`, `k8s.watch`, `k8s.create`, `k8s.update`, `k8s.delete`, `storage.get`, `storage.set`, `storage.delete`, `logs.stream`, `exec.open`, `event.subscribe`
   - Each handler deserializes JSON request, executes against the real backend services, serializes JSON response
   - Unknown methods return structured error

3. **Permission enforcement** (`internal/plugin/permissions.go`):
   - **Structural level**: only register host function handlers for capabilities declared in the manifest. If `permissions.logs: false`, the `logs.stream` handler is not registered — calling it returns "method not available."
   - **Fine-grained level**: for `k8s.*` methods, check the requested GVR and verb against `permissions.resources[]`. Denied calls return a structured error and log at WARN with `plugin={name}` group.
   - Export a `CheckPermission(manifest, method, request)` function for testability

4. **Enricher adapter** (`internal/plugin/enricher_adapter.go`):
   - Implement the `resource.Enricher` interface
   - Serializes `unstructured.Unstructured` to JSON, calls `plugin_enrich(gvr, obj)`, deserializes result
   - Handle errors gracefully (enricher failure should not break the resource list — log and return unenriched object)

5. **Enricher chaining** (modify `internal/resource/enricher.go` or `internal/resource/engine.go`):
   - `EnricherRegistry` supports multiple enrichers per GVR
   - Enrichers execute in registration order (built-in first, then plugins in load order)
   - Each enricher receives the output of the previous one
   - Detect field collisions: diff the object before/after each plugin enricher, log warning if a field was already set by a previous enricher

6. **Extend the loader** to:
   - After manifest validation, check for `extensions.enrichers.wasm` field
   - If present, instantiate the Wasm module via `wasm_runtime.go`
   - Register the `PluginEnricher` adapter in `EnricherRegistry` for each declared GVR

### Tests

- **Wasm runtime tests**: create a minimal test `.wasm` binary (compile a tiny Go program that exports `plugin_init` returning 0 and `plugin_enrich` that adds a field). Test: module loads, init succeeds, enrich returns modified object, destroy is called on close.
- **Host API tests**: mock the backend services, verify `host_call` dispatches correctly, verify permission denials return structured errors, verify unknown methods error.
- **Permission tests**: manifest with `resources: [{gvr: "apps.v1.deployments", verbs: ["list"]}]` → `k8s.list` for deployments succeeds, `k8s.delete` for deployments denied, `k8s.list` for pods denied.
- **Enricher adapter tests**: enricher modifies object correctly, enricher error returns unenriched object, enricher panic (Wasm trap) is caught.
- **Enricher chaining tests**: two enrichers for same GVR both execute in order, field collision logged.
- **WASI capability tests**: plugin without clock permission cannot call clock functions, stdout captured to logger.

### Definition of done

A plugin with a `.wasm` enricher module loads on startup, its `plugin_init()` is called, and when resources of its declared GVRs are listed, the enricher injects computed fields that appear in the resource list columns via CEL expressions. Permission violations are rejected and logged. Wasm traps are caught without crashing the app.

### Known gotchas

- Standard Go `GOOS=wasip1 GOARCH=wasm` produces large binaries (~5-15MB). For test fixtures, consider a minimal TinyGo binary or hand-written WAT.
- wazero's `host_alloc`/`host_free` pattern: the host allocates memory in the guest's address space by calling the guest's exported allocator. Make sure the test Wasm binary exports these.
- `slox.Info(ctx, ...)` — the context carries the logger. When logging from host functions, use the plugin's context (which should have the `plugin={name}` group already set up).
- The existing `EnricherRegistry` may be a simple map of GVR → single enricher. You'll need to change it to GVR → slice of enrichers for chaining.

---

## Phase P3 — Frontend Plugin System

### Context

I'm starting Phase P3 of the klados plugin system. Phases P1-P2 are complete — manifests validate, descriptors and sidebar entries load, Wasm enrichers run with permission enforcement, and the enricher pipeline chains correctly.

Read these files first (in order):
- `PLUGIN_ARCHITECTURE.md` — especially: Frontend Plugin System section (dependency sharing, component mounting, PluginContext, UI slots)
- `ARCHITECTURE.md` — frontend architecture, component structure, stores
- `CLAUDE.md` — Svelte 5 conventions (runes, `$state`, `$derived`, `$effect`), Tailwind v4 tokens, Wails event patterns

Review these for integration points:
- `frontend/src/lib/components/ResourceDetail.svelte` — where detail tab slots will render
- `frontend/src/lib/components/Sidebar.svelte` — where plugin sidebar entries render (partially done in P1)
- `frontend/src/lib/components/CommandPalette.svelte` — where plugin commands will register
- `frontend/src/lib/stores/cluster.svelte.ts` — clusterStore that plugin context reads from
- `frontend/src/lib/stores/resource.svelte.ts` — ResourceStore pattern for watch subscriptions

### What exists from P1-P2

- Plugin loader, manifest validation, descriptor/sidebar registration (P1)
- Wasm runtime, host API dispatch, permission enforcement, enricher adapter + chaining (P2)
- Frontend already renders plugin-defined sidebar entries and list columns from descriptors (P1)
- Generated TS types in `frontend/src/lib/plugins/types/` (P1)

### Deliverables

1. **Plugin context** (`frontend/src/lib/plugins/context.ts`):
   - `createPluginContext(manifest, hostServices)` — dynamically constructs a `PluginContext` object from the manifest
   - Capabilities not declared in the manifest are structurally absent (property doesn't exist)
   - `k8s` methods include fine-grained GVR/verb assertion before delegating to Wails bindings
   - `Object.freeze()` the returned context to prevent monkey-patching
   - `hostServices` wraps existing Wails service bindings (`ResourceService`, `LogService`, etc.)

2. **Frontend permission checking** (`frontend/src/lib/plugins/permissions.ts`):
   - `assertGVRPermission(manifest, gvr, verb)` — throws typed error if denied
   - Used by `PluginContext.k8s.*` methods
   - Matches the same logic as Go-side `CheckPermission` (consistent behavior)

3. **Plugin UI loader** (`frontend/src/lib/plugins/loader.ts`):
   - `loadPluginComponent(pluginName, componentPath)` → dynamic `import()` of the ES module
   - Returns the default export (Svelte component constructor)
   - Error handling: if import fails, return null and surface error via notification store
   - Cache loaded modules (don't re-import on every mount)

4. **Slot registry** (`frontend/src/lib/plugins/slots.ts`):
   - Central registry of all plugin UI extension points
   - Methods: `registerDetailTab(gvr, pluginName, tabDef)`, `registerCommand(pluginName, commandDef)`, `registerHeaderWidget(pluginName, componentPath)`, etc.
   - Query methods: `getDetailTabs(gvr)`, `getCommands()`, `getHeaderWidgets()`, etc.
   - Populated from plugin manifests on load (driven by `plugins:loaded` Wails event)
   - Reactive (Svelte 5 rune-based) so UI updates when plugins load/unload

5. **Detail tab slot rendering** (modify `ResourceDetail.svelte`):
   - After built-in tabs, render plugin-registered tabs for the current GVR
   - Each plugin tab: dynamically load component via `loader.ts`, mount with `mount()` from `svelte`, pass `{ resource, ctx }` props
   - On tab switch away: keep mounted (don't destroy/recreate)
   - On resource change: `instance.$set({ resource: newObj })`
   - On plugin unload: `unmount(instance)`, remove tab

6. **Command palette integration** (modify `CommandPalette.svelte`):
   - Merge plugin-registered commands into the command list
   - Plugin commands show plugin name as a badge/prefix for disambiguation
   - Command execution calls back into the plugin (via Wails binding or event)

7. **`@klados/plugin-ui` package** (initial version):
   - For this phase, this can be a local package in the monorepo (publish to npm later in P5)
   - Re-export Tailwind CSS design tokens, shared Bits UI primitives, Lucide icons
   - Export TypeScript types for `PluginContext`
   - Include a Vite config helper for plugin authors (externals pre-configured)

### Tests

- **PluginContext tests**: manifest with `resources` only → `ctx.k8s` exists, `ctx.logs` undefined, `ctx.storage` undefined. Manifest with all permissions → all properties exist. Frozen context cannot be modified.
- **Permission tests**: `ctx.k8s.list("apps.v1.deployments")` succeeds when permitted, `ctx.k8s.delete("apps.v1.deployments")` throws when verb not in manifest, `ctx.k8s.list("core.v1.pods")` throws when GVR not in manifest.
- **Loader tests**: mock `import()`, verify component loaded and cached. Verify import failure surfaces notification.
- **Slot registry tests**: register a detail tab, verify `getDetailTabs(gvr)` returns it. Unregister, verify removed.
- **Integration test**: mock a plugin with a detail tab component, verify it renders in `ResourceDetail` when viewing the correct GVR, verify it receives `resource` and `ctx` props.

### Definition of done

A plugin with a UI component (pre-built `.js` bundle) renders as a detail tab when viewing a resource of its declared GVR. The component receives a frozen `PluginContext` with only the capabilities the manifest declares. The component can call `ctx.k8s.list()` for permitted GVRs and gets real data back. Plugin commands appear in the command palette.

### Known gotchas

- Svelte 5 `mount()` returns an object with `$set()` and needs `unmount()` for cleanup — NOT `$destroy()` (that was Svelte 4). Verify the exact API against the installed Svelte version.
- Dynamic `import()` paths must be resolvable at runtime. Plugin UI bundles are served from the filesystem — make sure Vite/Wails serves the plugins directory or the loader constructs the correct URL.
- `Object.freeze()` is shallow — if `ctx.k8s` contains nested objects, freeze those too or use a deep freeze utility.
- Wails bindings mock in `setup.ts` will need the new `PluginService` methods mocked.
- The existing `ResourceDetail.svelte` uses a tab system — study its current implementation before adding plugin tab slots to avoid breaking the existing tab switching logic.

---

## Phase P4 — Developer Experience & Management

### Context

I'm starting Phase P4 of the klados plugin system. Phases P1-P3 are complete — manifests validate, Wasm enrichers run, plugin UI components render in slots with permission-scoped contexts, and commands appear in the palette.

Read these files first (in order):
- `PLUGIN_ARCHITECTURE.md` — especially: Lifecycle (load/unload sequences, hot reload), Error Handling, Plugin Storage, Plugin UI for Users, Conflict Resolution
- `ARCHITECTURE.md` — app lifecycle, session persistence, notification system
- `CLAUDE.md` — conventions

Review these for integration points:
- `internal/plugin/loader.go` — P1 loader (you'll add fsnotify watching)
- `internal/plugin/wasm_runtime.go` — P2 runtime (you'll add destroy/reload lifecycle)
- `internal/plugin/registry.go` — P1-P2 registry (you'll add unregister/re-register flow)
- `frontend/src/lib/stores/notification.svelte.ts` — toast system for plugin errors
- `frontend/src/lib/plugins/slots.ts` — P3 slot registry (you'll add unregister on plugin unload)

### What exists from P1-P3

- Full plugin loading pipeline: manifest → Wasm → enrichers → frontend UI slots
- Permission enforcement on both Go and frontend sides
- Plugin descriptors, sidebar entries, detail tabs, and commands all rendering
- No file watching, no storage, no management UI, no error recovery UX

### Deliverables

1. **Hot reload** (`internal/plugin/watcher.go`):
   - Use `fsnotify/fsnotify` to watch the plugins root directory
   - On plugin load, recursively watch all subdirectories (for UI component changes)
   - When a new subdirectory is created, register a watch for it
   - On any file change within a plugin directory, debounce 200ms, then execute full reload cycle:
     1. Frontend notification: "Reloading plugin {name}..."
     2. Unload: unmount all UI components, unregister sidebar/commands/tabs, unregister enrichers, call `plugin_destroy()`, close Wasm module
     3. Re-load: validate manifest, instantiate Wasm, register everything fresh
     4. Frontend notification: "Plugin {name} reloaded" (or error message)
   - Emit Wails events: `plugin:{name}:reloading`, `plugin:{name}:loaded`, `plugin:{name}:error`

2. **Plugin storage** (`internal/plugin/storage.go`):
   - File-backed key-value store at `$XDG_DATA_HOME/klados/plugins/{name}/storage.json`
   - Operations: `Get(key)`, `Set(key, value)`, `Delete(key)`, `List()` (list keys)
   - Thread-safe (sync.RWMutex)
   - Debounced write to disk (500ms, same pattern as session.go)
   - Accessible from Wasm via `host_call("storage.get/set/delete")`
   - Accessible from frontend via `PluginService.GetPluginStorage(pluginName, key)` etc. Wails binding

3. **Lifecycle events**:
   - When a cluster connects/disconnects, emit to all loaded plugins that have `permissions.events: true`:
     - Wasm: call a `plugin_on_event(event_ptr, event_len)` export if it exists (optional export — not all plugins need it)
     - Frontend: plugin components that subscribed via `ctx.subscribe("cluster:connected", cb)` receive the callback
   - Events: `cluster:connected`, `cluster:disconnected`, `namespace:changed`

4. **Error handling UX**:
   - Wasm trap → toast notification with plugin name and error summary → plugin auto-disabled
   - Plugin status tracked in registry: `active`, `disabled`, `errored` (with error message)
   - Frontend error boundary around plugin component mount — if `mount()` or dynamic `import()` throws, catch it, show toast, mark plugin as errored
   - Plugin errors logged to console via slox with `plugin={name}` group

5. **Plugin management UI** (new route or modal):
   - List installed plugins: name, version, status badge (active/disabled/errored)
   - Permission summary: expandable section showing what each plugin can access (GVRs + verbs, logs, exec, storage, WASI capabilities)
   - Enable/disable toggle (disabled = loaded but not active, no enrichers run, no UI rendered)
   - Manual reload button (for errored plugins or manual refresh)
   - Uninstall button (deletes plugin directory, with confirmation dialog)
   - Add to sidebar navigation under a "Plugins" or settings section

6. **Conflict warnings**:
   - Enricher field collisions: show in plugin management UI under the affected plugin
   - Command shortcut conflicts: show warning toast on load, note in management UI
   - Store conflicts in the plugin registry metadata for display

### Tests

- **Hot reload tests**: modify a file in a plugin dir → verify full unload/reload cycle executes (mock fsnotify events). Verify debouncing (rapid changes = single reload). Verify new subdirectory gets watched.
- **Storage tests**: set/get/delete operations, concurrent access (goroutine safety), debounced write to disk, separate storage per plugin (plugin A can't read plugin B's data).
- **Lifecycle event tests**: connect a cluster → plugins with event permission receive `cluster:connected`. Plugins without event permission don't.
- **Error handling tests**: Wasm trap during enrichment → plugin disabled, resource list still renders (without enrichment). UI component import failure → toast shown, tab slot empty. Manual reload after fix → plugin re-enabled.
- **Management UI tests**: render plugin list, toggle enable/disable, verify permission summary displays correctly.

### Definition of done

Edit a plugin's `.wasm` file or UI component → the app detects the change, reloads the plugin, and the updated behavior is visible within seconds. Plugin storage persists across reloads. A crashing plugin shows an error toast and is disabled — the rest of the app is unaffected. The management UI shows all plugins with their permissions, status, and controls.

### Known gotchas

- fsnotify on macOS uses kqueue which requires one file descriptor per watched file. With nested plugin UI dirs this can add up — but for a handful of plugins it's fine.
- Debounce must handle the case where multiple files change in rapid succession (e.g., a build tool writing `.wasm` + `manifest.json` + UI files). A single reload after all writes settle is correct.
- Storage debounced write: use the same pattern as `internal/session/session.go` (timer reset on each write, flush on shutdown).
- The unload sequence must clean up everything: Wasm module closed, enrichers unregistered, sidebar entries removed, detail tab components unmounted, commands removed from palette, fsnotify watches for subdirs removed. Missing any of these = memory leak or stale UI.
- Plugin disable vs unload: disabled plugins keep their manifest loaded (for the management UI) but don't run enrichers or render UI. This is different from unload (which removes everything).

---

## Phase P5 — Packaging, SDK & Example Plugin

### Context

I'm starting Phase P5 of the klados plugin system. Phases P1-P4 are complete — the full plugin runtime works: manifest validation, Wasm enrichers, frontend UI slots, hot reload, storage, error handling, and the management UI.

Read these files first (in order):
- `PLUGIN_ARCHITECTURE.md` — especially: Packaging & Distribution section (directory, OCI tar, OCI registry, media types, CLI commands), Example Plugin section, Dependencies (Plugin SDK)
- `ARCHITECTURE.md` — overall architecture
- `CLAUDE.md` — conventions, build commands

### What exists from P1-P4

- Complete plugin runtime: load → validate → Wasm → enrich → UI render → hot reload → storage → error handling → management UI
- All internal APIs are working and tested
- No packaging tools, no SDK, no example plugin, no OCI support

### Deliverables

1. **OCI tar packaging** (`internal/plugin/packaging.go` or a CLI subcommand):
   - `Pack(pluginDir) → .oci.tar.gz` (default compressed) or `.oci.tar` (with flag)
   - Produces a valid OCI Image Layout:
     - `oci-layout` with `imageLayoutVersion: "1.0.0"`
     - `index.json` pointing to the manifest digest
     - Blobs: config (`application/vnd.klados.plugin.manifest.v1+json`), wasm layer (`application/vnd.klados.plugin.wasm.v1`), UI layer (`application/vnd.klados.plugin.ui.v1+tar+gzip`), descriptors layer (`application/vnd.klados.plugin.descriptors.v1+tar+gzip`)
   - Compress with gzip by default
   - `Unpack(archivePath, pluginsDir)` — detect compression via gzip magic bytes (`1f 8b`), extract OCI layout, unpack layers into standard plugin directory structure

2. **Plugin CLI commands** (via Wails service or standalone):
   - `PluginService.InstallPlugin(path)` — accepts `.oci.tar.gz`, `.oci.tar`, or directory path. Extracts to plugins dir. Triggers load.
   - `PluginService.UninstallPlugin(name)` — unloads plugin, deletes directory.
   - `PluginService.PackPlugin(pluginDir)` — returns path to generated `.oci.tar.gz`.
   - Add install/uninstall UI to the management page (file picker for local install)

3. **Plugin SDK — Go module** (`github.com/Vilsol/klados-plugin-sdk`):
   - Separate Go module (can live in `sdk/go/` within the repo for now, extract to separate repo later)
   - Provides:
     - `sdk.RegisterEnricher(gvr, func(obj map[string]any) map[string]any)` — hides `plugin_enrich` export wiring
     - `sdk.K8s.List(gvr, ns)`, `sdk.K8s.Get(gvr, ns, name)` etc. — typed wrappers around `host_call`
     - `sdk.Storage.Get(key)`, `sdk.Storage.Set(key, value)` — typed wrappers
     - `sdk.Log.Info(msg)`, `sdk.Log.Warn(msg)` — wraps `host_log`
     - `sdk.OnEvent(eventType, callback)` — wraps event subscription
   - All `host_call` serialization/deserialization handled internally — plugin authors never touch pointers or memory
   - Types generated from the same JSON Schemas (import from `types/` subpackage)

4. **Plugin SDK — npm package** (`@klados/plugin-sdk`):
   - TypeScript types for `PluginContext`, all request/response types
   - Vite config helper: `createPluginViteConfig({ externals: true })` — pre-configures externals for `svelte`, `@klados/plugin-ui`
   - Can live in `sdk/js/` within the repo initially

5. **Publish `@klados/plugin-ui`** to npm (or finalize the local package from P3):
   - Re-exported Tailwind tokens, Bits UI primitives, Lucide icons, TS types
   - Versioned, so plugin authors can pin to a compatible version

6. **Example plugin** (`examples/plugin-node-annotator/`):
   - A complete, working plugin that demonstrates all capabilities:
     - `manifest.json` — requests node GVR (list, get, watch), storage, events
     - `main.go` — enricher that adds a computed field to nodes (e.g., counts taints, computes a readiness summary)
     - `ui/NodeAnnotation.svelte` — detail tab component that displays the enriched data plus a custom visualization, uses `ctx.k8s.list()` and `ctx.storage`
     - `descriptors/nodes.yaml` — adds a custom column to the node list
   - `mise.toml` with tasks:
     - `build:go` — `GOOS=wasip1 GOARCH=wasm go build -o dist/plugin.wasm .`
     - `build:tinygo` — `tinygo build -target=wasip1 -o dist/plugin-tiny.wasm .`
     - `build:ui` — `cd ui && npx vite build`
     - `build` — runs all three
     - `test` — loads both `.wasm` binaries into wazero, runs the enricher, asserts identical output
     - `pack` — builds + packs to `.oci.tar.gz`
   - Demonstrates dual Go/TinyGo compilation, proving both toolchains work

7. **OCI registry push/pull** (stretch goal — defer if time-constrained):
   - `PluginService.InstallFromRegistry(ociRef)` — uses `oras.land/oras-go` to pull
   - `PluginService.PushToRegistry(pluginDir, ociRef)` — uses ORAS to push
   - If deferred, document in PLUGIN_ARCHITECTURE.md as "Future: OCI registry support via ORAS"

### Tests

- **Packaging tests**: pack a plugin dir → verify valid OCI layout in tar. Unpack → verify directory matches original. Test both compressed and uncompressed. Test gzip detection.
- **Install/uninstall tests**: install from `.oci.tar.gz` → plugin loads and works. Uninstall → plugin removed, directory deleted.
- **SDK tests**: write a minimal plugin using the SDK, compile to Wasm, load in wazero, verify host_call wrappers serialize/deserialize correctly.
- **Example plugin tests**: `mise run build` produces valid `.wasm` for both Go and TinyGo. `mise run test` passes (identical enricher output). The plugin loads in Klados and renders correctly.

### Definition of done

Clone the example plugin, run `mise run build`, copy the directory to the plugins path → it loads, enriches nodes, shows a custom detail tab, and has a custom list column. Pack it to `.oci.tar.gz`, install on a clean setup via the management UI file picker → it works identically. Both Go and TinyGo builds produce working plugins with identical behavior.

### Known gotchas

- OCI Image Layout spec: `oci-layout` file must contain exactly `{"imageLayoutVersion":"1.0.0"}`. `index.json` must reference the manifest by digest. All blobs are content-addressed (filename = sha256 hash).
- Go Wasm binaries are large (~5-15MB). The `.oci.tar.gz` compression helps significantly. TinyGo binaries are much smaller (~100-500KB) but have stdlib limitations.
- The SDK's `host_call` wrapper must handle the alloc/free dance: allocate in guest memory for the request, call host, read response from the pointer the host returns. This is the trickiest part of the SDK — get it right with tests before building the rest on top.
- Plugin UI Vite build must output a single ES module (not code-split). Configure `rollupOptions.output.format: 'es'` and `inlineDynamicImports: true` in the plugin's Vite config.
- If the example plugin uses TinyGo, avoid: `reflect`, `encoding/json` (use a TinyGo-compatible JSON library or manual marshaling), and some `fmt` verbs. The test task should catch these issues.
