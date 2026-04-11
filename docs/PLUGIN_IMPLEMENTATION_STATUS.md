# Plugin System — Implementation Status

> Cross-reference of `PLUGIN_ARCHITECTURE.md` spec against the current codebase.
> Covers gaps, stubs, deviations, and undocumented additions.
> Last updated: 2026-03-29

---

## What Is Fully Implemented

These items match the spec completely.

| Area | Detail |
|---|---|
| Manifest format | `manifest.json` validated against `schemas/manifest.v1.json` (JSON Schema draft 2020-12) |
| Schema/host version checks | `schemaVersion` and `minHostVersion` enforced in `loader.go` |
| WASI capability configuration | All 4 WASI caps (`clock`, `filesystem`, `network`, `env`) default-deny; manifest-gated |
| Wasm host functions | All 4 imported: `host_call`, `host_log`, `host_alloc`, `host_free` |
| Plugin exports | All 5 spec'd exports present: `plugin_init`, `plugin_enrich`, `plugin_destroy`, `plugin_alloc`, `plugin_free` |
| Storage host API | `storage.get`, `storage.set`, `storage.delete` fully wired to `PluginStorage` |
| Permission enforcement | Dual-layer: structural absence (undeclared methods not registered) + fine-grained verb/GVR checks |
| Plugin load sequence | manifest scan → validate → Wasm instantiate → `plugin_init` → descriptor/enricher registration |
| Plugin unload sequence | `plugin_destroy` → module close → deregister enrichers/descriptors/sidebar/commands |
| Hot reload | fsnotify recursive watch, 200ms debounce per plugin, full lifecycle teardown + re-init |
| OCI packaging (pack) | `packaging.go` implements OCI Image Layout with gzip detection; `cmd/pluginpack` CLI tool |
| OCI packaging (install) | `PluginService.InstallPlugin()` handles directory, `.oci.tar`, and `.oci.tar.gz` |
| Descriptor registration | Plugin descriptors loaded and merged into `resource.Registry` |
| Enricher adapter | `PluginEnricher` wraps `WasmRuntime`, chained via `EnricherRegistry` |
| Stdout/stderr capture | `pluginLogWriter` routes Wasm output to `slox` logger with `plugin={name}` group |
| PluginContext construction | Built dynamically from manifest; undeclared capabilities absent from object |
| PluginContext freeze | `Object.freeze` applied to prevent monkey-patching |
| Frontend detail tab slot | `slotRegistry.detailTabs` wired and rendered in `ResourceDetail.svelte` |
| Frontend command slot | `slotRegistry.commands` wired into `CommandPalette.svelte` |
| Plugin management UI | `PluginManagement.svelte` — list, status, enable/disable, reload, uninstall, permissions summary |
| Go SDK | `sdk/go/`: `plugin_init/enrich/destroy/alloc/free` exported for both std Go and TinyGo |
| JS/UI SDK | `sdk/js/` and `sdk/js-ui/`: `PluginContext` types, `defineKladosPlugin()` Vite helper |
| Dependency externalization | `svelte` and `@klados/plugin-ui` marked external in SDK's `rollupOptions` |
| Example plugin | `plugin-node-annotator`: manifest, enricher, descriptor, UI component, both `plugin.wasm` and `plugin-tiny.wasm` |
| Enricher conflict detection | Field collision logged at WARN with both plugin names; last writer wins |
| Error isolation | wazero catches Wasm traps; plugin auto-disabled with toast + log; host does not crash |
| Host API — k8s | `k8s.list`, `k8s.get`, `k8s.create`, `k8s.update`, `k8s.delete` dispatch to active cluster's `ResourceEngine` |
| Host API — k8s.watch | `k8s.watch` starts `WatchManager` watch, subscribes Wails event, forwards to plugin via event channel |
| Host API — logs | `logs.stream` calls `LogStreamer.StartStream`, returns `streamId` |
| Host API — exec | `exec.open` calls `ExecManager.OpenSession`, returns `sessionId` |
| Host API — event.subscribe | `event.subscribe` subscribes Wails event, forwards payload to plugin via event channel |
| Host API event delivery | 64-buffer `chan eventPayload` owned by `WasmRuntime`; draining goroutine calls `plugin_on_event` |
| Frontend PluginContext — `k8s.watch()` | Calls `ResourceService.StartWatch`, subscribes `Events.On`, returns unsubscribe closure |
| Frontend PluginContext — `logs` | `logs.stream` → `LogService.StartLogStream`; `logs.stop` → `LogService.StopLogStream` |
| Frontend PluginContext — `exec` | `exec.open` → `ExecService.OpenExecSession`; `exec.close` → `ExecService.CloseExecSession` |
| Frontend PluginContext — `storage` | `storage.get/set/delete` → `PluginService.GetPluginStorageKey/SetPluginStorageKey/DeletePluginStorageKey` |
| Frontend sidebar slot | `slotRegistry.sidebarEntries` fetched and rendered in `Sidebar.svelte` grouped by category |
| Frontend overview field slot | `slotRegistry.overviewFields` rendered in `OverviewPanel.svelte` per GVR |
| Frontend header widget slot | `slotRegistry.headerWidgets` rendered in `Header.svelte` |
| Frontend status bar slot | `slotRegistry.statusBarWidgets` rendered in `Layout.svelte` conditional strip |
| CLI `plugin pack` | `klados plugin pack [--no-compress] <dir>` — runs without GUI via early intercept in `main.go` |
| CLI `plugin install` | `klados plugin install <path>` — installs to `$XDG_DATA_HOME/klados/plugins` without GUI |
| Example plugin — storage | `plugin-node-annotator` calls `sdk.Storage.Set` in `init()` and `sdk.Storage.Get` in enricher |
| Example plugin — events | `plugin-node-annotator` calls `sdk.OnEvent("cluster:connected", ...)` in `init()` |

---

## Unimplemented Items

### Frontend UI Slots — List Column and Context Menu

**Spec defines 8 mounting slots. Status:**

| Slot | Implemented |
|---|---|
| Detail tab | ✅ |
| Command action | ✅ |
| Sidebar entry | ✅ |
| Overview field | ✅ |
| Status bar widget | ✅ |
| Header widget | ✅ |
| Resource list column | ✅ |
| Context menu item | ✅ |

All 8 slots are now fully implemented end-to-end.

---

### Plugin Settings Page

**Spec:** "Plugin settings page" — per-plugin configuration UI accessible from the management screen.

**Reality:** `PluginManagement.svelte` shows permissions, enable/disable toggle, reload, and uninstall. There is no mechanism for a plugin to declare or render its own settings panel.

---

### OCI Registry Install (`oci://` prefix)

**Spec:** `klados plugin install oci://ghcr.io/foo/cert-manager:v1` — pull from registry then extract.

**Reality:** `InstallPlugin()` and the CLI `install` command only handle local paths (directory and `.oci.tar*`). The `oci://` scheme is not parsed or routed to any registry pull logic. ORAS dependency not present in `go.mod`.

---

### CLI `plugin push`

**Spec:** `klados plugin push ./cert-manager/ oci://ghcr.io/foo/cert-manager:v1`

**Reality:** Not implemented. ORAS dependency not present in `go.mod`.

---

## Deviations From Spec (Implemented Differently)

These are cases where the implementation diverges from the spec description but a deliberate design decision was made.

### `plugin_enrich` Return Type — Packed `uint64` Instead of `(i32, i32)`

**Spec (`PLUGIN_ARCHITECTURE.md`):**
```
plugin_enrich(gvr_ptr, gvr_len, obj_ptr, obj_len) → (ptr, len)
```
Two separate i32 return values.

**Implementation:** Returns a single `uint64` where `ptr << 32 | len`. This was required for TinyGo compatibility — TinyGo's `//export` directive does not support multiple return values.

---

### Plugin Descriptor Columns — Replace vs Extend

**Spec:** "Plugin descriptors for a GVR that already has a built-in descriptor **extend** it (add columns, panels) rather than replace it."

**Implementation (`registry/index.ts`):** Plugin-provided columns **replace** the built-in column list entirely. Only `detailPanels` and `actions` are additive (deduped); `overviewFields` are appended. This was an intentional decision to give plugin authors full control over column display.

---

### Plugin Events — Generic Names With Data Payload

**Spec:** Implies per-plugin event names (e.g., `plugin:{name}:reloading`).

**Implementation:** Generic event names (`plugin:reloading`, `plugin:loaded`, `plugin:error`) with `{"name": "..."}` in the data payload. This avoids the need for dynamic `Events.On()` subscriptions per plugin (Wails does not support wildcard subscriptions).

---

### `EnricherRegistry` — Slice-Based, Not Single-Value

**Spec:** Implies one enricher per GVR (describes "chaining" as a conflict scenario).

**Implementation:** `map[string][]Enricher` — `Register` appends, multiple enrichers per GVR are the normal case, not an edge case.

---

### `PluginEnricher` Implements `NamedEnricher`

**Spec:** No mention of a `NamedEnricher` interface.

**Implementation:** `PluginEnricher` adds `GetPluginName() string` to satisfy a `NamedEnricher` interface, enabling `EnricherRegistry.UnregisterPlugin(name)` to filter by plugin without a circular import (`services → plugin`, avoiding `plugin → services`).

---

### Enricher Adapter — Delta Merge, Not Full Object

**Spec:** Enricher receives full object, returns enriched full object.

**Implementation:** Enrichers should return **only the delta** (new or changed fields). The adapter performs a deep merge, logging a warning only when a leaf value is overwritten. Returning the full object causes spurious warnings for unchanged fields (`apiVersion`, `kind`, etc.).

---

### `plugin_on_event` Export — Undocumented

**Spec:** No mention of a `plugin_on_event` Wasm export.

**Implementation:** `wasm_runtime.go` implements `CallOnEvent()` which calls a `plugin_on_event(event_ptr, event_len)` export on the plugin. This is the mechanism for delivering subscribed events to plugin Wasm code. It is functional but absent from the architecture spec and the SDK documentation.

---

### Plugin Registry — `Deactivate` vs `Remove`

**Spec:** Describes a single unload path.

**Implementation:** Two distinct operations in `registry.go`:
- `Deactivate(name)` — removes extension points (enrichers, descriptors, sidebar, tabs, commands) but retains the plugin entry for the management UI.
- `Remove(name)` — full deletion (used on uninstall).
- `SetStatus(name, status)` — updates display status without touching extensions.

---

## FEATURES.md Checkbox Status

Section 13 in `FEATURES.md` marks all plugin items as `[ ]` (unchecked). The following items have sufficient implementation to be checked off:

| Item | Status |
|---|---|
| Wasm plugin loading via wazero | ✅ Implemented |
| Plugin manifest format | ✅ Implemented |
| Plugin lifecycle management (install, enable, disable, uninstall) | ✅ Implemented |
| Plugin sandboxing — scoped API access | ✅ Implemented |
| Plugin-to-host API | ✅ Implemented (k8s, logs, exec, events, storage all wired) |
| Register new resource views / detail tabs | ✅ Implemented |
| Register command palette commands | ✅ Implemented |
| Plugin local storage | ✅ Implemented (Wasm side + frontend context) |
| Svelte component bundles loaded at runtime | ✅ Implemented |
| Plugin UI slots | ✅ All 8 slots implemented |
| Host-provided UI component library | ✅ Implemented (`@klados/plugin-ui`) |
| Local plugin loading (from filesystem) | ✅ Implemented |
| CLI plugin pack/install | ✅ Implemented (`klados plugin pack/install`) |
| Register new sidebar entries | ✅ Implemented |
| Register overview field widgets | ✅ Implemented |
| Register header widgets | ✅ Implemented |
| Register status bar widgets | ✅ Implemented |
| Register custom actions on resources | ✅ Implemented (context menu in ResourceList.svelte) |
| Register resource list columns | ✅ Implemented (plugin columns in ResourceList.svelte) |
| Plugin settings page | ❌ Not implemented |
| OCI registry install / push | ❌ Not implemented |
