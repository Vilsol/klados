# Phase 4 — PluginService Extension & Wails Bindings

Extend `PluginService.InstallPlugin` to handle `oci://` registry references and add the Wails
methods the frontend needs to save credentials and manage insecure registries — the Go-side
contract the GUI depends on.

## First Action

Open `internal/services/plugin.go` and read `InstallPlugin` (around line 531) in full — trace
exactly what happens after a successful local install (load plugin → register → init runtime →
watch → emit event) so you can replicate the same post-install sequence for the OCI path without
duplicating code.

## Context

Phase 2 built `PullFromRegistry` and `ErrAuthRequired` in `internal/plugin/remote.go`. The GUI
already has a plugin management page but has no way to trigger a registry pull — `InstallPlugin`
only handles local paths. This phase adds the `oci://` branch, the credential-save method, and
the insecure-registry config field. It also regenerates Wails bindings so Phase 5 has the correct
TypeScript signatures to import.

## Files to Read

- `internal/services/plugin.go` lines 531–583 — `InstallPlugin`: the branching structure (dir
  vs archive), the post-install sequence (LoadPlugin → Register → initPluginRuntime → Watch →
  Emit), and the `findNewPluginDir` fallback; the OCI branch must call the same post-install
  sequence
- `internal/plugin/remote.go` — `PullFromRegistry`, `RemoteOpts`, `ErrAuthRequired`,
  `SaveDockerCredentials` — the functions your new service methods delegate to
- `internal/config/config.go` — the `Config` struct where `InsecureRegistries []string` is added;
  study the `Update` method pattern used by other service methods to persist config changes
- `internal/services/plugin.go` lines 282–305 — `DisablePlugin` / the `Config().Update` pattern
  to follow for `AddInsecureRegistry`

## What Exists

**From Phase 2:**
- `internal/plugin/remote.go` — `PullFromRegistry`, `PushToRegistry`, `SaveDockerCredentials`,
  `RemoteOpts`, `ErrAuthRequired` (message: `"authentication required"`)

**Baseline:**
- `internal/services/plugin.go` — `InstallPlugin`, `PluginService` struct, all existing methods
- `internal/config/config.go` — `Config` struct with `Update` method
- Wails bindings at `frontend/bindings/github.com/Vilsol/klados/internal/services/pluginservice.ts`
  (or `.js`) — regeneration will add new methods here
- `frontend/src/lib/__tests__/wails-mock.ts` — mock to update after regeneration

## Deliverables

1. `internal/config/config.go` — add `InsecureRegistries []string \`json:"insecureRegistries,omitempty"\``
   to `Config`; zero value (nil slice) is the correct default, no migration needed

2. `PluginService.InstallPlugin(path string) error` extended — new branch before existing local
   path handling:
   ```
   if strings.HasPrefix(path, "oci://") {
       host := extractHost(path)  // ghcr.io/foo/bar:v1 → ghcr.io
       insecure := slices.Contains(s.appService.Config().InsecureRegistries, host)
       opts := plugin.RemoteOpts{Insecure: insecure}
       if err := plugin.PullFromRegistry(path, s.pluginsDir, opts); err != nil {
           return err  // ErrAuthRequired propagates unwrapped
       }
       // find and load the newly installed plugin dir (reuse findNewPluginDir logic)
       ...
   }
   ```
   The post-pull flow (LoadPlugin → Register → initPluginRuntime → Watch → Emit) must be
   identical to the existing archive branch — extract it into a shared `s.activatePlugin(destDir)`
   helper if the duplication is significant.

3. `PluginService.SaveRegistryCredentials(host, username, password string) error` — new
   Wails-bound method; calls `plugin.SaveDockerCredentials(host, username, password)`

4. `PluginService.AddInsecureRegistry(host string) error` — new Wails-bound method; appends
   `host` to `Config.InsecureRegistries` via `s.appService.Config().Update(func(c *config.Config)
   { ... })` using the same pattern as `DisablePlugin`

5. Wails bindings regenerated: run `wails3 generate bindings` — `SaveRegistryCredentials` and
   `AddInsecureRegistry` must appear in the generated plugin service bindings file

## Tests

- **Go unit test** (extend `internal/services/` test files or add new):
  - `InstallPlugin("oci://ghcr.io/test/plugin:v1")` with a mock `PullFromRegistry` that returns
    `nil` → plugin loaded, `plugins:loaded` event emitted
  - `InstallPlugin("oci://...")` with a mock that returns `plugin.ErrAuthRequired` → error
    returned is `plugin.ErrAuthRequired` (use `errors.Is`)
  - `SaveRegistryCredentials("ghcr.io", "user", "token")` writes to temp docker config (set
    `HOME` to a temp dir)
  - `AddInsecureRegistry("localhost:5000")` persists to config and is reflected in the
    `InsecureRegistries` slice on next `Config()` read
  - `go test ./internal/config/` — `InsecureRegistries` field round-trips through JSON save/load
    with nil and populated values

## Acceptance Criteria

- [ ] `InstallPlugin("oci://...")` calls `PullFromRegistry` (verified via mock)
- [ ] `errors.Is(err, plugin.ErrAuthRequired)` is true when pull returns auth failure
- [ ] `SaveRegistryCredentials` and `AddInsecureRegistry` appear in the regenerated bindings
- [ ] `wails3 generate bindings` runs without error
- [ ] `go test ./internal/services/` passes
- [ ] `go test ./internal/config/` passes (new field has correct zero-value default)

## Definition of Done

`PluginService.InstallPlugin("oci://ghcr.io/foo/bar:v1")` pulls from the registry and loads
the plugin exactly as a local install does. Auth failures return `ErrAuthRequired` unchanged.
`SaveRegistryCredentials` writes to `~/.docker/config.json`. The new methods appear in the
generated TypeScript/JS bindings file. All service tests pass.

## Known Gotchas

- **The trap**: wrapping `ErrAuthRequired` before returning from `InstallPlugin`
  **Why**: Wails serialises Go errors to JSON; if you wrap `ErrAuthRequired` with `fmt.Errorf`,
  `errors.Is` still works on the Go side, but the frontend matches on the error message string
  `"authentication required"`. If you wrap it with a different message, the frontend match fails.
  **What to do instead**: return `ErrAuthRequired` unwrapped from `InstallPlugin`. If you must
  wrap for context, use `fmt.Errorf("installing plugin: %w", err)` — `%w` preserves the message
  chain and the string `"authentication required"` still appears in `err.Error()`.

- **The trap**: forgetting to update `wails-mock.ts` after regenerating bindings
  **Why**: `wails3 generate bindings` adds new methods; any frontend test that imports
  `pluginservice` transitively will fail if the mock doesn't have matching stubs.
  **What to do instead**: after running `wails3 generate bindings`, open
  `frontend/src/lib/__tests__/wails-mock.ts` and add stub entries for `SaveRegistryCredentials`
  and `AddInsecureRegistry`. Also check `setup.ts` for any `$Create.Map` patterns the new
  bindings may introduce.

- **The trap**: calling `xdg.Reload()` in tests that modify `XDG_CONFIG_HOME`
  **Why**: `adrg/xdg` caches directory paths at startup; `t.Setenv` alone won't change what
  `xdg.ConfigHome` returns during the test.
  **What to do instead**: call `xdg.Reload()` after `t.Setenv("XDG_CONFIG_HOME", tmpDir)` and
  restore with `t.Cleanup(func() { xdg.Reload() })`. This is documented in `cerebrum.md`.

- **The trap**: `extractHost` parsing breaking on `localhost:5000/foo/bar:v1`
  **Why**: splitting `oci://localhost:5000/foo/bar:v1` on `/` and taking index [0] after
  stripping `oci://` gives `localhost:5000` — which is correct. But a naive `strings.Split`
  on `:` gives `localhost` — which is wrong.
  **What to do instead**: strip `oci://`, then split on `/` and take the first segment. The
  first `/`-delimited segment of an OCI reference is always the registry host (with optional port).
