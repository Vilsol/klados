# Plugin Command Invocation

## Context

Plugin manifests can declare commands that appear in the command palette. Currently the `action` in `slots.svelte.ts` is a no-op `() => {}` — selecting a command does nothing. This spec covers wiring the full invocation path for both a Wasm backend handler and a frontend UI component, at the plugin author's choice.

Hard constraints:
- Wazero module instances are not safe for concurrent calls — all Wasm dispatch must be serialized
- The invocation must be async and non-blocking from the host's perspective
- Plugin UI components must be fully autonomous (no host-managed chrome or lifecycle)

---

## Decisions

**Two dispatch paths discriminated by optional `component` field**
If `component` is absent from the command manifest entry, the host dispatches to the plugin's Wasm module. If `component` is present, the host dynamically imports and mounts the Svelte component. This keeps the manifest the single source of truth and requires no runtime type switching beyond a field presence check.

**Mutex serializes all Wasm calls**
A single `sync.Mutex` on `WasmRuntime` protects `CallEnrich`, `CallOnEvent`, and `CallCommand`. The existing event goroutine and enricher pipeline already race — this mutex fixes all three at once. A long-running command may briefly queue enricher calls, which is acceptable since commands are rare user-initiated actions.

**Frontend component is fully autonomous**
The component is mounted into `document.body` with `ctx` as a prop. The plugin is responsible for unmounting itself. The host tracks no component instances. This gives plugin authors maximum flexibility (dialogs, drawers, toasts, navigation — anything).

**`initPluginRuntime` restructured to cover command-only plugins**
Currently exits early when `Extensions.Enrichers == nil`. A plugin with Wasm command handlers but no enricher would never get a runtime. The condition is widened to: create a runtime if enrichers OR any commands without a `component` are declared.

**`CommandEntry` carries full `PermsSummary`**
The frontend needs permissions to build `PluginContext` for component-path commands. Rather than a new RPC, `CommandEntry` embeds the full `PermsSummary` (Resources, Logs, Exec, Storage, Events) — consistent with how `DetailTabEntry` embeds `ResourcePerms`, but complete. The frontend synthesizes a partial manifest from this data, matching the existing `makePluginCtx` pattern in `ResourceDetail.svelte`.

**Fix latent `makePluginCtx` bug in `ResourceDetail.svelte`**
`DetailTabEntry` only embeds `ResourcePerms`, so `makePluginCtx` silently drops storage/events/logs/exec from detail tab contexts. Fix `DetailTabEntry` to also carry full `PermsSummary`, and update `makePluginCtx` accordingly, as part of this change.

---

## Rejected Alternatives

**Channel-based serialization for Wasm calls**
Route all Wasm calls through a single channel with reply channels for results. Uniform pattern but enricher calls require allocating a reply channel per call and blocking on it — more overhead and complexity than a mutex for no practical gain.

**Refactor `createPluginContext` signature to accept permissions directly**
`ResourceDetail.svelte` already synthesizes a fake manifest from entry data and passes it to `createPluginContext`. Changing the signature would move complexity without eliminating it. The fake-manifest pattern is established and sufficient.

**Lazy `GetPluginManifest(name)` fetch for frontend context**
Async call on every command invocation, requires caching. Unnecessary given that embedding permissions in `CommandEntry` follows the existing pattern at zero extra RPC cost.

---

## Library Selections

No new dependencies.

---

## Priorities & Tradeoffs

- **Simplicity over throughput**: mutex serialization is simpler than a call queue; acceptable because commands are infrequent
- **Consistency over minimalism**: embedding full `PermsSummary` in `CommandEntry` is slightly more data than strictly needed for most commands, but consistency with the existing pattern matters more
- **Plugin autonomy over host control**: autonomous component mounting gives plugin authors maximum flexibility at the cost of host having no lifecycle visibility

---

## Potential Gotchas

- **`types/manifest.go` is generated** — after changing `manifest.v1.json`, run `mise run generate:plugin-types` before touching any Go code that references `types.Command`. Editing the generated file directly will be overwritten.
- **TinyGo export syntax differs** — `exports_go.go` uses `//go:wasmexport`, `exports_tinygo.go` uses `//export`. Both files need the `plugin_command` export added with their respective syntax.
- **`plugin_command` export is optional** — not all plugins will export it (e.g. component-path-only plugins). `WasmRuntime.CallCommand` must check `mod.ExportedFunction("plugin_command") == nil` and return nil gracefully, same as `CallOnEvent` does.
- **Mutex must wrap the event goroutine too** — `CallOnEvent` is currently called from a goroutine without holding any lock. The mutex acquisition must move into `CallOnEvent` itself, not just into the new `CallCommand`.
- **`initPluginRuntime` storage init is independent of enrichers** — storage is set up before the early-return check. The restructure must preserve that ordering.

---

## Implementation Details

### Schema change — `manifest.v1.json`

```json
"Command": {
  "type": "object",
  "required": ["id", "label"],
  "additionalProperties": false,
  "properties": {
    "id":        { "type": "string" },
    "label":     { "type": "string" },
    "icon":      { "type": "string" },
    "component": { "type": "string", "description": "Relative path to Svelte component JS bundle. If set, mounts UI component instead of calling Wasm." }
  }
}
```

Run `mise run generate:plugin-types` after — this updates `internal/plugin/types/manifest.go`.

### Go — `internal/plugin/registry.go`

```go
type CommandEntry struct {
    PluginName string       `json:"pluginName"`
    ID         string       `json:"id"`
    Label      string       `json:"label"`
    Icon       *string      `json:"icon,omitempty"`
    Component  *string      `json:"component,omitempty"`
    Perms      PermsSummary `json:"perms"`
}

// DetailTabEntry — extend with full Perms (replaces ResourcePerms-only)
type DetailTabEntry struct {
    PluginName string       `json:"pluginName"`
    GVR        string       `json:"gvr"`
    ID         string       `json:"id"`
    Label      string       `json:"label"`
    Component  string       `json:"component"`
    Perms      PermsSummary `json:"perms"`
}
```

`Registry.Register()` — update command and detailTab population to set `Perms: buildPermsSummary(p.Manifest.Permissions)` (dereference the pointer; `buildPermsSummary` already exists in `registry.go`). Remove the old `ResourcePerms` field from `DetailTabEntry` and update `toResourcePerms` usage accordingly.

### Go — `internal/plugin/wasm_runtime.go`

```go
type WasmRuntime struct {
    rt         wazero.Runtime
    mod        api.Module
    pluginName string
    ctx        context.Context
    eventCh    chan eventPayload
    hapi       *hostAPI
    mu         sync.Mutex  // serializes all mod calls
}

// CallCommand calls plugin_command(id_ptr, id_len).
// Returns nil if the export does not exist (component-path plugins won't export it).
func (r *WasmRuntime) CallCommand(commandID string) error {
    fn := r.mod.ExportedFunction("plugin_command")
    if fn == nil {
        return nil
    }

    idBytes := []byte(commandID)
    allocFn := r.mod.ExportedFunction("plugin_alloc")
    freeFn := r.mod.ExportedFunction("plugin_free")
    if allocFn == nil {
        return fmt.Errorf("plugin %s missing plugin_alloc", r.pluginName)
    }

    r.mu.Lock()
    defer r.mu.Unlock()

    results, err := allocFn.Call(r.ctx, uint64(len(idBytes)))
    if err != nil || len(results) == 0 {
        return fmt.Errorf("plugin_alloc failed: %w", err)
    }
    ptr := uint32(results[0])

    if !r.mod.Memory().Write(ptr, idBytes) {
        return fmt.Errorf("writing command id to guest memory failed")
    }

    _, err = fn.Call(r.ctx, uint64(ptr), uint64(len(idBytes)))

    if freeFn != nil {
        _, _ = freeFn.Call(r.ctx, uint64(ptr), uint64(len(idBytes)))
    }
    return err
}
```

Also add `r.mu.Lock()/Unlock()` around the `mod` calls in `CallEnrich` and `CallOnEvent`.

### Go — `internal/services/plugin.go`

```go
// InvokeCommand dispatches a plugin command asynchronously.
// Returns immediately; errors are emitted as plugin:error events.
func (s *PluginService) InvokeCommand(pluginName, commandID string) error {
    rt, ok := s.runtimes[pluginName]
    if !ok {
        return fmt.Errorf("no runtime for plugin %q", pluginName)
    }
    go func() {
        if err := rt.CallCommand(commandID); err != nil {
            slox.Warn(s.ctx, "plugin command failed", "plugin", pluginName, "command", commandID, "error", err)
            app := application.Get()
            if app != nil {
                app.Event.Emit("plugin:error", map[string]string{"name": pluginName, "error": err.Error()})
            }
        }
    }()
    return nil
}
```

Also restructure `initPluginRuntime` early-return condition:

```go
// Before: exits if no enricher. After: also stays if there are Wasm commands.
hasEnricher := p.Manifest.Extensions != nil && p.Manifest.Extensions.Enrichers != nil
hasWasmCommands := false
if p.Manifest.Extensions != nil {
    for _, cmd := range p.Manifest.Extensions.Commands {
        if cmd.Component == nil {
            hasWasmCommands = true
            break
        }
    }
}
if !hasEnricher && !hasWasmCommands {
    return
}
```

### Go SDK — `sdk/go/sdk.go`

```go
var commandHandlers = map[string]func(){}

// OnCommand registers a handler for the given command ID.
func OnCommand(id string, fn func()) {
    commandHandlers[id] = fn
}

// DispatchCommand delivers a command invocation to the registered handler.
func DispatchCommand(idPtr, idLen uint32) {
    id := string(ReadGuestBytes(idPtr, idLen))
    if fn, ok := commandHandlers[id]; ok {
        fn()
    }
}
```

### Go SDK — `sdk/go/exports_go.go`

```go
//go:wasmexport plugin_command
func PluginCommand(idPtr, idLen uint32) {
    DispatchCommand(idPtr, idLen)
}
```

### Go SDK — `sdk/go/exports_tinygo.go`

```go
//export plugin_command
func pluginCommand(idPtr, idLen uint32) {
    DispatchCommand(idPtr, idLen)
}
```

### Frontend — `frontend/src/lib/plugins/slots.svelte.ts`

In `initFromBackend`, change command mapping to preserve `component` and `perms`:

```typescript
this.commands = (cmds ?? []).map((c) => ({
    pluginName: c.pluginName ?? '',
    id:         c.id ?? '',
    label:      c.label ?? '',
    icon:        c.icon ?? undefined,
    component:  c.component ?? undefined,
    perms:      c.perms,
    action:     () => {},  // wired below
}))
```

Wire `action` based on path:

```typescript
action: c.component
    ? () => invokeComponentCommand(c.pluginName, c.component!, c.perms, basePluginURL)
    : () => { PluginService.InvokeCommand(c.pluginName, c.id).catch(() => {}) }
```

`invokeComponentCommand`:

```typescript
async function invokeComponentCommand(
    pluginName: string,
    component: string,
    perms: PermsSummary,
    basePluginURL: string | null,
) {
    if (!basePluginURL) return
    const url = `${basePluginURL}/${pluginName}/${component}`
    const mod = await import(/* @vite-ignore */ url)
    if (!mod?.default) return
    const ctx = buildCommandContext(pluginName, perms)
    mount(mod.default, { target: document.body, props: { ctx } })
}
```

`buildCommandContext` — synthesizes a partial manifest (same pattern as `makePluginCtx` in `ResourceDetail.svelte`, but with full permissions):

```typescript
function buildCommandContext(pluginName: string, perms: PermsSummary): Readonly<PluginContext> {
    const manifest = {
        schemaVersion: 1 as const,
        name: pluginName,
        version: '', displayName: '', minHostVersion: '',
        permissions: {
            resources: perms.resources?.map((p) => ({ ...p, verbs: p.verbs as any })),
            logs:    perms.logs    || undefined,
            exec:    perms.exec    || undefined,
            storage: perms.storage || undefined,
            events:  perms.events  || undefined,
        },
    }
    const ctx = clusterStore.activeContext ?? ''
    return createPluginContext(manifest, {
        clusterName:    ctx,
        clusterVersion: '',
        namespace:      clusterStore.selectedNamespaces[0] ?? '',
        listResources:  (g, n) => ResourceService.ListResources(ctx, g, n ?? ''),
        getResource:    (g, n, name) => ResourceService.GetResource(ctx, g, n, name),
    })
}
```

### Frontend — `frontend/src/lib/components/ResourceDetail.svelte`

Update `makePluginCtx` to map full permissions from `tab.perms` (once `DetailTabEntry` carries `PermsSummary` instead of `ResourcePerms`):

```typescript
function makePluginCtx(tab: RegisteredDetailTab) {
    const ns = clusterStore.selectedNamespaces[0] ?? namespace
    const manifest = {
        schemaVersion: 1 as const,
        name: tab.pluginName,
        version: '', displayName: '', minHostVersion: '',
        permissions: {
            resources: tab.perms.resources?.map((p) => ({ ...p, verbs: p.verbs as any })),
            logs:    tab.perms.logs    || undefined,
            exec:    tab.perms.exec    || undefined,
            storage: tab.perms.storage || undefined,
            events:  tab.perms.events  || undefined,
        },
    }
    return createPluginContext(manifest, { ... })
}
```

Update `RegisteredDetailTab` in `slots.svelte.ts` to carry `perms: PermsSummary` instead of `resourcePerms: ResourcePerm[]`.

### Data flow (Wasm path)

```
User selects command in palette
  → cmd.action() in slots.svelte.ts
  → PluginService.InvokeCommand(pluginName, commandID)   [Wails RPC, returns immediately]
  → goroutine: rt.CallCommand(commandID)
  → mu.Lock()
  → plugin_command(id_ptr, id_len) Wasm export
  → DispatchCommand in sdk.go
  → commandHandlers[id]()                                [arbitrary plugin code]
  → mu.Unlock()
  → errors → plugin:error Wails event → notificationStore
```

### Data flow (component path)

```
User selects command in palette
  → cmd.action() in slots.svelte.ts
  → invokeComponentCommand(pluginName, component, perms, basePluginURL)
  → dynamic import(componentURL)
  → buildCommandContext(pluginName, perms)
  → mount(mod.default, { target: document.body, props: { ctx } })
  → component owns its own lifecycle from here
```

---

## Definition of Done

- [ ] `manifest.v1.json` has optional `component` on `Command`; `mise run generate:plugin-types` run; generated `types/manifest.go` reflects the change
- [ ] `CommandEntry` carries `Component *string` and `Perms PermsSummary`; `DetailTabEntry` carries `Perms PermsSummary` (replacing `ResourcePerms`)
- [ ] `WasmRuntime` has `sync.Mutex`; mutex is held in `CallEnrich`, `CallOnEvent`, and `CallCommand`
- [ ] `CallCommand` returns nil gracefully when `plugin_command` export is absent
- [ ] `initPluginRuntime` creates a runtime for command-only (no enricher) Wasm plugins
- [ ] `PluginService.InvokeCommand` is Wails-bound, goroutine-dispatched, errors emitted as `plugin:error`
- [ ] `sdk.OnCommand` and `DispatchCommand` exist in `sdk.go`; `plugin_command` export added to both `exports_go.go` and `exports_tinygo.go`
- [ ] `slots.svelte.ts` wires `action` to Wasm path or component path based on `component` presence
- [ ] `makePluginCtx` in `ResourceDetail.svelte` maps full permissions (logs, exec, storage, events) — latent bug fixed
- [ ] `node-annotator` example updated: `sdk.OnCommand("node-annotator-taint-report", fn)` registered in `main.go` demonstrating the Wasm path
- [ ] Wails bindings regenerated (`wails3 generate bindings`) after adding `InvokeCommand`
