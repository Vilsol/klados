# Klados — OCI Plugin Registry & Cobra CLI Restructure

## Project Overview

Replace the hand-rolled `flag`-based CLI in `main.go` and the standalone `cmd/pluginpack` binary
with a single Cobra-driven `klados` binary. Launching with no arguments opens the Wails GUI;
subcommands (`klados plugin pack/push/install`) run as a pure CLI without starting the GUI.
Add OCI registry push and pull via `oras.land/oras-go/v2`, with auth sourced from the Docker
credential chain, CLI flags, or a bearer token — and expose registry install in the GUI plugin
management page.

---

## Phase Map

```
Phase 1 (Cobra restructure) ──────────────────┐
                                               ├── Phase 3 (CLI push/install commands)
Phase 2 (OCI remote layer) ───────────────────┘
         │
         └──────────── Phase 4 (PluginService extension + Wails bindings)
                                  │
                                  └── Phase 5 (Frontend registry UI)
```

Phase 1 and Phase 2 are fully independent and can be worked in parallel.
Phase 3 requires both Phase 1 and Phase 2.
Phase 4 requires only Phase 2.
Phase 5 requires Phase 4 (regenerated bindings).

---

## Phase 1 — Cobra CLI Restructure

> Replace the hand-rolled `flag`/`runPluginCLI` in `main.go` with a Cobra command tree so that
> `klados` works as both a GUI launcher and a proper CLI — the structural prerequisite for all
> new subcommands.

| | |
|---|---|
| **Depends on** | none |
| **Parallel with** | Phase 2 |

### Deliverables

- `github.com/spf13/cobra` added to `go.mod` / `go.sum`
- `main.go` reduced to embed declaration + single `cmd.Execute(assets)` call
  (embed must stay in `main.go` — Go disallows `..` in embed paths, so `assets embed.FS` is
  passed into the `cmd` package rather than declared there)
- `cmd/root.go` — `Execute(assets embed.FS)` entry point; root Cobra command whose `RunE`
  contains the full Wails app startup logic moved verbatim from `main.go`
- `cmd/plugin.go` — `plugin` subcommand group (no `RunE`; parent only)
- `cmd/plugin_pack.go` — `klados plugin pack <dir> [--no-compress]`; calls `plugin.Pack()`
- `cmd/pluginpack/` directory deleted
- `Taskfile.yml` / `mise.toml` build references updated (the binary is still named `klados`)

### Tests

- **Manual verification**
  - `klados` (no args) opens the Wails window
  - `klados plugin pack ./examples/plugin-node-annotator` produces `node-annotator-*.oci.tar.gz`
  - `klados plugin pack --no-compress ./examples/plugin-node-annotator` produces `*.oci.tar`
  - `klados plugin --help` prints the subcommand list
  - `klados --help` shows root command usage
- **Go unit test** — existing `go test ./...` suite still passes (no regressions)

### Out of Scope

- `push` and `install` subcommands — those ship in Phase 3
- Any OCI registry networking — Phase 2

### Acceptance Criteria

- [ ] `main.go` contains only the embed directive and `cmd.Execute(assets)`
- [ ] `cmd/pluginpack/` does not exist
- [ ] `klados plugin pack` produces the same archive as the old `pluginpack` binary
- [ ] All existing Go tests pass
- [ ] `klados --help` lists `plugin` as a subcommand

### Handoff Notes

- `assets embed.FS` must be declared in `main.go` (package main) and passed to `cmd.Execute`
  because `//go:embed all:frontend/dist` is relative to the file's directory; `cmd/` is a
  subdirectory and cannot reach `../frontend/dist` with an embed directive.
- The root command's `RunE` is essentially the block of code in `main()` after the `os.Args`
  CLI check — move it wholesale, do not refactor yet.
- Cobra's default behaviour when no subcommand is given is to run the root command's `RunE`,
  which is exactly what we want.

---

## Phase 2 — OCI Remote Layer

> Implement the push and pull primitives against OCI registries using `oras-go/v2`, with a
> unified auth resolution chain — the shared backend that both the CLI commands and the GUI
> service call into.

| | |
|---|---|
| **Depends on** | none |
| **Parallel with** | Phase 1 |

### Deliverables

- `oras.land/oras-go/v2` added to `go.mod` / `go.sum`
- `internal/plugin/remote.go` containing:
  - `RemoteOpts` struct: `Username, Password, Token string; Insecure bool`
  - `ErrAuthRequired` — typed sentinel error returned when a registry rejects with 401/403,
    used by GUI to trigger the inline credential prompt
  - `resolveCredStore(opts RemoteOpts) credentials.Store` (unexported) — priority:
    1. `opts.Token` → static credential with token as password, empty username
    2. `opts.Username` + `opts.Password` → static credential
    3. `~/.docker/config.json` loaded via `credentials.NewFileStore` (covers plain stored
       creds and all configured credential helpers automatically)
    4. Anonymous fallback (`credentials.Anonymous`)
  - `newRepo(ref string, opts RemoteOpts) (*remote.Repository, error)` (unexported) —
    constructs the ORAS `remote.Repository`, sets `PlainHTTP` from `opts.Insecure`, attaches
    resolved credential store; strips `oci://` prefix before parsing
  - `PullFromRegistry(ref, pluginsDir string, opts RemoteOpts) error` — pulls to a temp OCI
    layout directory via `oras.Copy`, then calls `unpackDir(tempDir, pluginsDir)`; maps auth
    errors to `ErrAuthRequired`
  - `PushToRegistry(archivePath, ref string, opts RemoteOpts) error` — extracts the
    `.oci.tar.gz` to a temp directory using the existing tar-reading logic, opens it as
    `layout.New(tempDir)`, resolves the descriptor from the layout's index, and uses
    `oras.Copy` to push to the remote; maps auth errors to `ErrAuthRequired`
  - `SaveDockerCredentials(host, username, password string) error` — reads (or creates)
    `~/.docker/config.json`, sets `auths.<host>.auth` to `base64(username:password)`, writes
    back atomically
  - `unpackDir(layoutDir, pluginsDir string) error` (unexported) — reads `index.json` and
    blobs from an OCI layout directory, extracts the plugin the same way `Unpack` does for
    tar archives (shares the extraction logic via a helper)

### Tests

- **Go unit test** (`internal/plugin/remote_test.go`) using an `httptest.Server` that speaks
  the OCI Distribution Spec (or using ORAS's built-in `registry/remote/test` helpers):
  - Anonymous pull succeeds against a public (no-auth) test server
  - `Username`+`Password` opts produce a correct `Authorization: Basic` header
  - `Token` opt produces the correct bearer token header
  - Docker config file credentials are picked up automatically (temp `~/.docker/config.json`)
  - 401 response from registry returns `ErrAuthRequired`
  - Push round-trip: `PushToRegistry` then `PullFromRegistry` into a temp plugins dir
    produces the same `manifest.json` as the original archive
  - `Insecure: true` uses plain HTTP (no TLS) to reach the test server
  - `SaveDockerCredentials` creates the file if absent, merges without clobbering other hosts

### Out of Scope

- Cobra command wiring — Phase 3
- GUI integration — Phases 4 and 5
- Cloud-provider-native auth (ECR, GCR, ACR) — covered automatically via Docker credential
  helpers if the user has them configured; no native SDK integration in this phase

### Acceptance Criteria

- [ ] `PushToRegistry` + `PullFromRegistry` round-trip test passes
- [ ] Auth resolution priority order is covered by tests
- [ ] `ErrAuthRequired` is returned on 401/403, not a generic error
- [ ] `SaveDockerCredentials` test verifies file creation and merge behaviour
- [ ] `go test ./internal/plugin/` passes

### Handoff Notes

- ORAS `credentials.NewFileStore(path)` accepts an explicit path — use
  `filepath.Join(os.Getenv("HOME"), ".docker", "config.json")` rather than relying on the
  Docker default so the path is testable via env override.
- The `oci://` prefix is stripped before passing to ORAS — ORAS reference parsing does not
  understand it.
- Auth error detection: check whether the wrapped error or its cause satisfies
  `*errcode.ErrorResponse` with code 401/403, or match on the string `"unauthorized"` /
  `"authentication required"` as a fallback for registries that don't follow the spec precisely.
- `PushToRegistry` extracts the tar to a temp dir and opens it with `layout.New` —
  it does NOT stream blobs directly from the tar. This is intentional: simpler code, and
  plugin archives are small enough that temp-dir overhead is negligible.

---

## Phase 3 — CLI Push & Install Commands

> Wire the OCI remote layer into two new Cobra subcommands so developers can push plugin
> archives to a registry and install plugins from a registry or local path from the shell.

| | |
|---|---|
| **Depends on** | Phase 1, Phase 2 |
| **Parallel with** | Phase 4 |

### Deliverables

- `cmd/plugin_push.go` — `klados plugin push <archive.oci.tar.gz> <oci://ref>`
  - Flags: `-u/--username`, `-p/--password`, `-t/--token`, `--insecure`
  - Validates the archive path exists and ends in `.oci.tar.gz` or `.oci.tar`
  - Calls `plugin.PushToRegistry`; prints success or error to stderr and exits non-zero on failure
- `cmd/plugin_install.go` — `klados plugin install <path-or-oci://ref>`
  - Same auth flags as push
  - If arg starts with `oci://`: calls `plugin.PullFromRegistry` writing to
    `$XDG_DATA_HOME/klados/plugins/`
  - Otherwise (local path): if it's a directory, copies contents into the plugins dir (reuses
    `copyDirContents` logic from `services/plugin.go`, extracted to `internal/plugin/` so both
    the service and the CLI can use it); if `.oci.tar` / `.oci.tar.gz`, calls `plugin.Unpack`
  - Prints the installed plugin name and destination directory on success

### Tests

- **Manual verification**
  - `klados plugin push ./node-annotator-0.1.0.oci.tar.gz oci://ghcr.io/<owner>/node-annotator:0.1.0`
    pushes successfully (requires a writable test registry or GHCR dev account)
  - `klados plugin install oci://ghcr.io/<owner>/node-annotator:0.1.0` installs to plugins dir
  - `klados plugin install ./examples/plugin-node-annotator` installs from local dir
  - `klados plugin install ./node-annotator-0.1.0.oci.tar.gz` installs from archive
  - `klados plugin push ... --insecure` works against a local plain-HTTP registry
  - Missing credentials → error message includes "authentication required" and suggests
    `--username`/`--password` or `--token`
- **Go unit test** — argument validation: missing args, invalid archive extension, unrecognised
  path type all return the correct usage error without making network calls

### Out of Scope

- GUI install — Phase 5
- Credential storage to `~/.docker/config.json` from CLI (CLI auth is flags-only; GUI handles
  persistent storage in Phase 5)

### Acceptance Criteria

- [ ] `klados plugin push --help` shows all four auth flags
- [ ] `klados plugin install oci://...` exits 0 and prints install path on success
- [ ] `klados plugin install` with a local dir/archive works identically to the old `pluginpack install`
- [ ] Non-zero exit code and stderr message on auth failure, bad path, or network error
- [ ] `go test ./cmd/` passes

### Handoff Notes

- Extract `copyDirContents` + `copyFile` from `internal/services/plugin.go` to
  `internal/plugin/install.go` (or similar) before Phase 3, so neither the CLI nor the service
  duplicates the logic. The service then calls the shared helper.
- `$XDG_DATA_HOME` via `github.com/adrg/xdg` — already a dependency, use `xdg.DataHome`.
- The CLI install command does NOT spin up `PluginService` — it writes files to disk only.
  The running GUI picks up the new plugin via fsnotify on next launch or live via the watcher.

---

## Phase 4 — PluginService Extension & Wails Bindings

> Extend `PluginService.InstallPlugin` to handle `oci://` registry references and add the Wails
> methods the frontend needs to save credentials and manage insecure registries.

| | |
|---|---|
| **Depends on** | Phase 2 |
| **Parallel with** | Phase 3 |

### Deliverables

- `internal/config/config.go` — `InsecureRegistries []string` field added to `Config` struct
- `PluginService.InstallPlugin(path string) error` extended:
  - If `path` starts with `oci://`, call `plugin.PullFromRegistry(ref, s.pluginsDir, opts)`
    where `opts.Insecure` is derived by checking whether the registry host is in
    `s.appService.Config().InsecureRegistries`
  - On success, load + register the plugin exactly as the existing local-path branch does
  - If `PullFromRegistry` returns `plugin.ErrAuthRequired`, return it unwrapped so the
    frontend can detect it by type
- `PluginService.SaveRegistryCredentials(host, username, password string) error` — new
  Wails-bound method; calls `plugin.SaveDockerCredentials`
- `PluginService.AddInsecureRegistry(host string) error` — new Wails-bound method; appends
  host to `Config.InsecureRegistries` via `Config.Update`
- Wails bindings regenerated: `wails3 generate bindings`
  (new methods appear in `frontend/bindings/.../pluginservice.ts` / `.js`)

### Tests

- **Go unit test** — extend `PluginService` tests (or add new file) to cover:
  - `InstallPlugin("oci://...")` with a mock that returns `ErrAuthRequired` → error propagated
  - `InstallPlugin("oci://...")` with a mock that returns `nil` → plugin loaded and emits
    `plugins:loaded` event
  - `SaveRegistryCredentials` calls through to `SaveDockerCredentials` (use a temp home dir)
  - `AddInsecureRegistry` persists to config and is reflected in subsequent `PullFromRegistry`
    opts

### Out of Scope

- Frontend rendering of the credential form — Phase 5
- Wails binding JS imports in the frontend — Phase 5

### Acceptance Criteria

- [ ] `InstallPlugin("oci://ghcr.io/foo/bar:v1")` routes to `PullFromRegistry`
- [ ] `ErrAuthRequired` propagates to the Wails caller unmodified
- [ ] `SaveRegistryCredentials` and `AddInsecureRegistry` appear in regenerated bindings
- [ ] `go test ./internal/services/` passes
- [ ] `go test ./internal/config/` passes (new field has zero-value default, no migration needed)

### Handoff Notes

- `ErrAuthRequired` must propagate as-is through the Wails service boundary — Wails serialises
  Go errors to `{"code": ..., "message": "..."}` JSON. The frontend detects auth failure by
  matching the error message string `"authentication required"` (or a constant exported from
  the bindings). Agree on this string before Phase 5 begins.
- `wails3 generate bindings` produces `.js` files (not `.ts`) per the Wails v3 alpha.74
  behaviour noted in cerebrum. Import with `.js` extension in frontend code.
- After regenerating bindings, update the Wails mock in `frontend/src/lib/__tests__/setup.ts`
  and `wails-mock.ts` if new methods appear that are called by components under test.

---

## Phase 5 — Frontend Registry UI

> Add the registry install field and inline credential prompt to the plugin management page,
> completing the end-to-end GUI flow for installing plugins from an OCI registry.

| | |
|---|---|
| **Depends on** | Phase 4 |
| **Parallel with** | nothing |

### Deliverables

- `frontend/src/routes/PluginManagement.svelte` updated with:
  - "Install from registry" input field accepting bare refs (`ghcr.io/foo/bar:v1`) or
    `oci://`-prefixed refs; the frontend always normalises to `oci://` before calling
    `InstallPlugin`
  - Loading state while install is in progress
  - On `ErrAuthRequired` response: inline credential form appears below the input showing
    the detected registry host, a username field, a password/token field (single field — token
    goes here), and an "Insecure (HTTP)" checkbox
  - On credential form submit: call `SaveRegistryCredentials(host, username, password)`, then
    if insecure is checked call `AddInsecureRegistry(host)`, then retry `InstallPlugin`
  - On success: dismiss form, show toast notification, refresh plugin list
  - On repeated auth failure after saving credentials: show persistent error with suggestion
    to verify the credentials
- Frontend bindings imports updated to the newly generated `.js` files for
  `SaveRegistryCredentials` and `AddInsecureRegistry`

### Tests

- **Frontend test (vitest + @testing-library/svelte)**
  - Happy path: `InstallPlugin` resolves → toast shown, plugin list refreshed
  - Auth failure path: `InstallPlugin` rejects with `"authentication required"` → credential
    form rendered with correct host pre-filled
  - Credential submit: `SaveRegistryCredentials` called with correct args, then `InstallPlugin`
    retried
  - Insecure checkbox: `AddInsecureRegistry` called before retry when checked
  - `oci://` prefix normalisation: bare ref `ghcr.io/foo/bar:v1` passed to `InstallPlugin`
    as `oci://ghcr.io/foo/bar:v1`
  - All new Wails methods mocked in `wails-mock.ts`

### Out of Scope

- A dedicated "registry credentials manager" settings screen — the inline prompt on first
  failure is sufficient for now
- Showing installed registry sources on the plugin card
- Plugin marketplace / search — entirely out of scope

### Acceptance Criteria

- [ ] Typing a registry ref and submitting installs the plugin (happy path, tested with mock)
- [ ] Auth failure shows the inline credential form without a full page error
- [ ] Credentials are saved to `~/.docker/config.json` and the install retries automatically
- [ ] Insecure checkbox triggers `AddInsecureRegistry` before retry
- [ ] `npx vitest run src/lib/__tests__/PluginManagement.svelte.test.ts` passes
- [ ] All existing frontend tests still pass (`npx vitest run`)

### Handoff Notes

- The auth error string check should use the exported constant / message agreed in Phase 4
  handoff — do not hardcode `"authentication required"` in multiple places.
- The registry host is parsed from the `oci://` ref by splitting on `/` and taking the first
  segment — this is sufficient for `ghcr.io/foo/bar:v1` but will be wrong for
  `localhost:5000/foo/bar:v1`. Use a proper URL/ref parser (ORAS ref parsing is Go-side;
  on the frontend, a simple regex `^oci://([^/]+)` is fine for now).
- After Phase 5 ships, update `PLUGIN_ARCHITECTURE.md` and `ARCHITECTURE.md` to reflect the
  OCI registry install flow, the auth chain, and the `InsecureRegistries` config field.
