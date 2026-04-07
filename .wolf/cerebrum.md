# Cerebrum

> OpenWolf's learning memory. Updated automatically as the AI learns from interactions.
> Do not edit manually unless correcting an error.
> Last updated: 2026-03-22

## User Preferences

<!-- How the user likes things done. Code style, tools, patterns, communication. -->

- **Package manager is pnpm**: Use `pnpm install` (not `npm install`) for this project.

## Key Learnings

- **PortForward tunnelFunc is injectable**: `portforward.Manager` stores a `tunnel tunnelFunc` field. In tests, replace with `blockingTunnel` (blocks on ctx.Done) or `failingTunnel` (returns error immediately) to test manager state machine without real k8s connections.
- **portforward aggregate event**: Manager emits both `portforward:{ctx}:{id}` (per-forward) and `portforward:{ctx}:updated` (list-level). Sidebar subscribes to the latter and calls ListForwards.
- **Service endpoint resolution**: Use `ResourceService.GetResource(ctx, ns, 'core.v1.endpoints', serviceName)` — endpoints resource name matches service name. No new RPC needed.

- **Wails bindings are now `.js` files** (not `.ts`): `wails3 generate bindings` generates `.js` files in Wails v3 alpha.74. Import with `.js` extension in frontend code.
- **Connection.Clientset should be `kubernetes.Interface`** (not `*kubernetes.Clientset`): Using the interface allows `fake.NewSimpleClientset()` in tests. Changed in Phase 4.
- **Fiber v2 WebSocket**: Use `github.com/gofiber/websocket/v2` for Fiber v2 WebSocket support. Import as `fiberws "github.com/gofiber/websocket/v2"`.
- **Streaming server routing**: The `Use("/:token", ...)` middleware covers nested paths. WebSocket routes registered as `Get("/:token/ws/logs/:streamID", fiberws.New(...))` are covered by this middleware.
- **Svelte 5 component mocking in Vitest**: Mock component as `{ default: vi.fn() }` — NOT `{ default: { render: () => {} } }` (the latter is Svelte 4 SSR format). `vi.fn()` is callable and won't crash Svelte 5's render path.
- **vi.mock factory hoisting**: `vi.mock()` factories cannot reference top-level variables. Use `vi.hoisted()` to define shared mock functions that are accessible both in the factory and in test assertions.
- **Mock paths are relative to the test file**: In `vi.mock('../../../bindings/...')` from `src/lib/__tests__/`, use 3 levels `../` — NOT 4 (which would be relative to `src/lib/components/panels/`). Both resolve to the same absolute path, but vi.mock hoisting uses the test file's location.
- **LogStreamer backpressure**: 1024-item buffered channel. k8s reader goroutine blocks when full (natural backpressure). On StopStream(), cancel() is called → goroutine exits → channel closed → HandleConn() drain loop exits. StopStream() also removes the stream from the map before cancel to avoid double-delete.
- **ExecManager session flow**: OpenSession() validates connection + stores config. Actual k8s exec starts in HandleConn() when WebSocket connects. Resize sent as text JSON `{"type":"resize","cols":N,"rows":N}`.
- **go-deadlock is a drop-in replacement for sync.Mutex/RWMutex**: `github.com/sasha-s/go-deadlock` — just change the field type, all Lock/Unlock/RLock/RUnlock calls work identically. Prints goroutine stacks with lock acquisition sites after 30s timeout. Applied project-wide across all 11 mutex sites.
- **ResourceEngine.ListRaw skips enrichers**: Added to avoid re-entrant WasmRuntime mutex deadlock from host API callbacks. Use `ListRaw` in plugin host API; use `List` (with enrichers) for frontend-facing resource reads.
- **Svelte 5 async in mount()-ed components**: Use `async IIFE` inside `$effect` (not `.then()` chains on Promise.resolve()) to reliably flush `$state` updates from async callbacks in dynamically mounted components. Pattern: `$effect(() => { ;(async () => { const data = await fetch(); stateVar = data })() })`.

## Do-Not-Repeat

<!-- Mistakes made and corrected. Each entry prevents the same mistake recurring. -->

- **[2026-04-06]** Do NOT pass objects to `fakemetrics.NewSimpleClientset(objects...)` — the object tracker fails to find them via Get/List. Use `NewSimpleClientset()` with no args, then `PrependReactor("get"/"list", resource, func)` to return objects. `Tracker.Add` works in standalone programs but fails in `_test` packages.

- **[2026-03-22]** Do NOT use `*kubernetes.Clientset` in `cluster.Connection` — use `kubernetes.Interface` to allow test doubles (`fake.NewSimpleClientset()`).
- **[2026-03-22]** Do NOT reference top-level variables inside `vi.mock()` factory — they're hoisted before the variables are initialized. Use `vi.hoisted()` instead.
- **[2026-03-22]** Do NOT use `{ render: () => {} }` to mock Svelte 5 components — Svelte 5 expects components to be callable functions. Use `vi.fn()` instead.
- **[2026-03-22]** Do NOT use 4 `../` in vi.mock paths from `src/lib/__tests__/` — only 3 are needed. The mock path must be relative to the TEST FILE, not the component file.
- **[2026-03-23]** Do NOT read `$state` variables inside a `$effect` that should only track specific signals — wrap with `untrack()`. Reading `lines.length` in a WS `$effect` caused an infinite reconnect loop (every push re-ran the effect).
- **[2026-03-23]** Do NOT use `BrowserOpenURL` from `@wailsio/runtime` — it's not exported. Use `Browser.OpenURL` (import `{ Browser }` from `@wailsio/runtime`).
- **[2026-03-23]** Do NOT subscribe to wildcard Wails events (e.g. `portforward:{ctx}:*`) — Events.On() takes exact names. Emit an aggregate `portforward:{ctx}:updated` event from the manager for list-level subscriptions.
- **[2026-03-24]** Do NOT call `t.Setenv("XDG_STATE_HOME", ...)` alone in Go tests — `adrg/xdg` caches directory paths. Must also call `xdg.Reload()` after the setenv and restore in `t.Cleanup`. Same applies to `XDG_CONFIG_HOME`.
- **[2026-03-24]** Do NOT forget to add `Create.Map` to the Wails mock in `setup.ts` after running `wails3 generate bindings` — the bindings may use `$Create.Map` for map fields. Current mock: `(kfn, vfn) => (obj) => maps values over object keys`.
- **[2026-03-24]** Do NOT test container names in LogsPanel/TerminalPanel by looking for them in the DOM without opening the dropdown first
- **[2026-03-24]** Do NOT use `term.onFocus`/`term.onBlur` on xterm.js Terminal — those methods don't exist. Use `term.textarea?.addEventListener('focus'/'blur', ...)` instead. — the container dropdown is closed by default, so names are not visible until clicked.
- **[2026-03-27]** TinyGo `//go:wasmexport` does NOT support multiple return values — use packed uint64 (`ptr<<32 | len`) instead. Also has a `runtime.wasmExportCheckRun()` guard that panics without `_start`. Use `//export` (CGO style) instead: no guard, works without `_start`, supports multiple returns as packed uint64.
- **[2026-03-27]** TinyGo WASM generates `_start` (command model), NOT `_initialize` (reactor model). After `_start`, wazero closes the module (proc_exit). Use `WithStartFunctions()` (empty) to skip `_start` and call exported functions directly. Then call `plugin_init()` explicitly to initialize plugin state.
- **[2026-03-27]** testza import path is `github.com/MarvinJWendt/testza` — NOT `github.com/testza/testza`.
- **[2026-03-29]** Do NOT use `uint64`/i64 as a return type for `//go:wasmimport` functions in TinyGo — asyncify corrupts i64 return values when the goroutine stack is rewound. Use i32-only ABI for all host imports.
- **[2026-03-28]** Standard Go WASM command modules call `proc_exit(0)` after `_start`, which closes the wazero module. Do NOT call `_start` then expect module to remain usable. TinyGo exports are callable WITHOUT calling `_start` first — use `WithStartFunctions()` (empty) on the wazero `ModuleConfig`. Standard Go reactor mode (`_initialize`) is not confirmed working in Go 1.25.
- **[2026-03-28]** TinyGo `//export` functions cannot be passed as values (function pointers). Extract the implementation to a non-exported helper function and delegate from the export wrapper. Example: `tinygoAlloc` helper called by both the `//export plugin_alloc` wrapper and `//export plugin_enrich`.
- **[2026-04-03]** Do NOT call `ResourceEngine.List` from within the plugin host API (`host_api.go`) — it runs enrichers which call `WasmRuntime.CallEnrich` which tries to acquire `r.mu`, deadlocking if `CallCommand` already holds it. Use `ResourceEngine.ListRaw` instead (skips enrichers). Same risk applies to `Get` if enrichment is ever added there.
- **[2026-04-03]** When importing `go-deadlock` in a file that also uses `sync.WaitGroup` or other `sync` primitives, keep both imports — do NOT replace `"sync"` entirely. Only change the struct field types to `deadlock.Mutex`/`deadlock.RWMutex`.

## Decision Log

- **[2026-03-22] Fiber v2 for WebSocket**: Used `gofiber/websocket/v2` (Fiber v2 compatible) rather than contrib/websocket. go.mod already uses fiber/v2.
- **[2026-03-22] LogService/ExecService as separate Wails services**: Matches ARCHITECTURE.md which lists LogService and ExecService as distinct services. Keeps concerns separated.
- **[2026-03-22] Session management pattern**: Follows watcher/manager.go (sync.Mutex + context.CancelFunc per session). Consistent across codebase.
- **[2026-03-22] Resize via WebSocket (not Wails)**: Terminal resize is high-frequency and must be synchronous with data flow. Kept entirely in WebSocket protocol as text JSON frames, not a separate Wails RPC call.
- **[2026-03-25] Plugin Wasm runtime: wazero**: Pure Go, no CGO, WASI preview 1. Supports any language compiled to Wasm. Go std (`GOOS=wasip1`) is first-class, TinyGo as optimization path.
- **[2026-03-27] plugin_enrich returns packed uint64**: `plugin_enrich(gvr_ptr, gvr_len, obj_ptr, obj_len) → uint64` where return = `ptr<<32 | len`. Changed from `(i32, i32)` multi-return to support TinyGo's `//export` limitation.
- **[2026-03-27] EnricherRegistry is now slice-based**: `map[string][]Enricher` — `Register` appends, `GetAll` returns slice. Built-in enrichers registered first, plugin enrichers appended in load order.
- **[2026-03-27] EnrichRuntime interface**: `PluginEnricher` uses `EnrichRuntime` interface (not `*WasmRuntime`) to enable unit testing with `fakeRuntime`. Always pass `Ctx` field when constructing `PluginEnricher` in services.
- **[2026-03-25] Plugin host API: JSON over single `host_call` dispatch**: One import function with method name string, JSON serialization. Minimal Wasm import surface, easy for any guest language to bind.
- **[2026-03-25] Plugin UI: dynamic import() + Svelte 5 mount()**: Plugin UI bundles are pre-built ES modules with shared deps marked external. Host provides Svelte, Bits UI, Tailwind tokens at runtime via `@klados/plugin-ui` npm package.
- **[2026-03-25] Plugin permissions: structural absence + fine-grained checks**: Undeclared capabilities are not present on the PluginContext object (Object.freeze). Declared capabilities still get verb/GVR checks per call.
- **[2026-03-25] Plugin packaging: directory (dev) / .oci.tar.gz (sideload) / OCI registry (prod)**: All decompose to same directory format at runtime. OCI Image Layout spec for single-file transport. Compression detected via gzip magic bytes.
- **[2026-03-25] Plugin storage: Go-managed only**: No localStorage. All plugin storage via Go backend at `$XDG_DATA_HOME/klados/plugins/{name}/storage.json`. Both Wasm and UI access same store.
- **[2026-03-25] Plugin error handling: auto-disable, no retry**: Wasm traps → toast + plugin UI indicator + console log. Plugin disabled until user manually reloads.
- **[2026-03-25] Plugin hot reload: full lifecycle**: fsnotify → debounce 200ms → disable → shutdown → replace → init → enable. No partial updates.
- **[2026-03-25] Plugin enricher conflicts: chain in load order**: Multiple enrichers for same GVR are chained. Field collision = last writer wins + warning logged.
- **[2026-03-25] Schema as source of truth**: JSON Schema (draft 2020-12) for manifest, host API, plugin context. Codegen: `omissis/go-jsonschema` (Go), `json-schema-to-typescript` (TS). Validation: `santhosh-tekuri/jsonschema/v6`.
- **[2026-03-25] Plugin stdout/stderr → slox logger**: Captured and wrapped in slog with `plugin={name}` group label.
- **[2026-03-25] mise for task running**: Use mise.toml, not Taskfile.yml.
- **[2026-03-28] Plugin event naming: generic events with name in data**: Use `plugin:reloading`, `plugin:loaded`, `plugin:error` (generic) with `{"name":"..."}` in event data — NOT `plugin:{name}:reloading` per-plugin. Frontend can subscribe once per event type. Wails Events.On() requires exact names (no wildcards).
- **[2026-03-28] PluginWatcher uses callback not back-reference**: `plugin.NewPluginWatcher(ctx, func(name string))` takes a callback to avoid circular import (services → plugin, not plugin → services).
- **[2026-03-28] Registry.Deactivate vs Remove**: `Deactivate` removes extension points (sidebar/tabs/commands/enrichers) but keeps plugin entry for management UI. `Remove` deletes everything (for uninstall). `SetStatus` updates status without touching extensions.
- **[2026-03-28] Plugin enricher adapter uses deep merge**: `deepMerge` in `enricher_adapter.go` recurses into nested maps rather than doing a top-level overwrite. Only warns when a leaf (string/number/array) value is overwritten. Plugin enrichers should return ONLY the delta (new fields), not the full object — returning the full object causes spurious leaf-overwrite warnings for `apiVersion`, `kind`, etc.
- **[2026-03-28] Plugin descriptor columns REPLACE built-in columns**: In `registry/index.ts`, when a plugin provides a descriptor for an existing GVR, its columns replace the built-in column list. `detailPanels` and `actions` are additive (deduped). `overviewFields` are appended. Always create a NEW object in `descriptors.set()` — never mutate the value from `builtins`, or `reloadPlugins()` will see corrupted builtins on next call.
- **[2026-03-28] Plugin descriptor YAML uses Go JSON struct tags**: `group`/`version`/`resource` (NOT `gvr`), `columns` (NOT `defaultColumns`), CEL expressions without `object.` prefix (e.g. `metadata.name` not `object.metadata.name`).
- **[2026-03-28] PluginEnricher implements NamedEnricher**: `GetPluginName()` method added so `EnricherRegistry.UnregisterPlugin(name)` can filter by plugin via `NamedEnricher` interface (no circular import needed).
- **[2026-03-28] PluginStorage path**: `$XDG_DATA_HOME/klados/plugins/{name}/storage.json` (data home, not config home). Created via `os.MkdirAll`.
- **[2026-03-28] NewWasmRuntime now takes *PluginStorage**: Passed to hostAPI for real storage dispatch. Pass nil for no-storage plugins.
- **[2026-03-28] loader.ts clears moduleCache on plugin:reloading**: Essential for hot reload — otherwise Svelte re-uses the old cached component. Cache keys contain `/{name}/` substring for matching.
- **[2026-03-29] TinyGo asyncify: no i64 in wasmimport ABI**: TinyGo asyncify does NOT correctly restore `uint64`/i64 return values from `wasmimport` functions when replaying the goroutine stack. Always use i32-only host import ABI. Pattern: `host_call` returns response length (i32 only), then `host_read_response(buf_ptr, buf_len)` copies the response into a guest-`make`d buffer. No re-entrant `plugin_alloc` from host.
- **[2026-04-03] go-deadlock for deadlock monitoring**: Applied `github.com/sasha-s/go-deadlock` project-wide as a drop-in for all `sync.Mutex`/`sync.RWMutex` fields. Zero behavioral change; adds automatic goroutine dump on 30s lock timeout. Preferred over manual pprof or SIGQUIT for Wails desktop apps where terminal signals aren't easily accessible.
- **[2026-04-03] Plugin command invocation — two dispatch paths**: Commands with `component` field → dynamic `import()` + Svelte `mount()` on `document.body`. Commands without `component` → `PluginService.InvokeCommand` → `WasmRuntime.CallCommand` → `plugin_command` Wasm export. Both paths defined in `frontend/src/lib/plugins/slots.svelte.ts`.

- **[2026-04-07] `clusterStore.selectedNamespaces` is `Record<string, string[]>`**: Indexing with `[0]` returns `string[] | undefined`, not `string`. Always use `clusterStore.getSelectedNamespaces(ctxName)[0] ?? ''`.
- **[2026-04-07] Svelte 5 `onMount(async)` with cleanup**: `onMount` doesn't support returning cleanup from an async function (`Promise<() => void>` is invalid). Pattern: `onMount(() => { let cleanup; ;(async () => { ...; cleanup = () => { ... } })(); return () => cleanup?.() })`. If no awaits inside, just remove `async`.
- **[2026-04-07] `context.d.ts` K8SContext methods typed as objects**: Auto-generated from JSON schema, `list`/`get`/`watch` were `{ [k: string]: unknown }` (not callable). Manually fixed to proper function signatures. Re-running `json-schema-to-typescript` will revert — keep this in mind.
- **[2026-04-07] `services/index.ts` duplicate `PluginService`**: Generated binding re-exports `PluginService` as both a service namespace (from pluginservice.js) and a data model (from models.js). Fixed by aliasing model as `PluginServiceModel`.
- **[2026-04-07] `svelte/internal/client` has no type declarations**: Added `svelte-internal-client.d.ts` ambient declaration in `src/plugin-shared/`.
- **[2026-04-07] Plugin descriptor `actions` must be objects, not strings**: `actions: [- delete]` (bare string) fails with `cannot unmarshal string into Go struct field Descriptor.actions of type resource.Action`. Correct form: `- name: delete\n  label: Delete`.
