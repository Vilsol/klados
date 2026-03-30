# Phase 3 — CLI Push & Install Commands

Wire the OCI remote layer into two new Cobra subcommands so developers can push plugin archives
to a registry and install plugins from a registry or local path directly from the shell.

## First Action

Read `cmd/plugin_pack.go` to understand the exact subcommand pattern you'll replicate — how
flags are declared, how errors are surfaced, and how `plugin.*` functions are called. Then open
`internal/services/plugin.go` and find `copyDirContents` and `copyFile` at the bottom of the
file — extract these to `internal/plugin/` before writing any command code, since both the
CLI and the service need them.

## Context

Phase 1 built the Cobra skeleton with a `plugin` subgroup and a working `pack` command. Phase 2
built `PullFromRegistry`, `PushToRegistry`, and `ErrAuthRequired` in `internal/plugin/remote.go`.
This phase connects them: `klados plugin push` and `klados plugin install` become working CLI
commands. The `install` command also handles local paths (directory copy and `.oci.tar.gz`
extraction), replacing the removed `runPluginCLI` install case from the old `main.go`.

## Files to Read

- `cmd/plugin_pack.go` — the subcommand pattern to replicate: flag declaration, arg validation,
  `plugin.*` call, error handling
- `cmd/plugin.go` — where to register the new subcommands
- `internal/plugin/remote.go` — `PushToRegistry`, `PullFromRegistry`, `RemoteOpts`,
  `ErrAuthRequired` signatures
- `internal/services/plugin.go` lines ~592–660 — `copyDirContents` and `copyFile` to extract
  into `internal/plugin/install.go`; also `InstallPlugin` (lines ~531–583) to understand the
  full local install logic the CLI must replicate without `PluginService`
- `internal/plugin/packaging.go` — `Unpack()` signature (used by CLI install for archive paths)

## What Exists

**From Phase 1:**
- `cmd/root.go`, `cmd/plugin.go`, `cmd/plugin_pack.go` — working Cobra structure
- `cmd/pluginpack/` deleted

**From Phase 2:**
- `internal/plugin/remote.go` — `PullFromRegistry`, `PushToRegistry`, `SaveDockerCredentials`,
  `RemoteOpts`, `ErrAuthRequired`

**Baseline:**
- `internal/plugin/packaging.go` — `Pack()`, `Unpack()`
- `internal/services/plugin.go` — `copyDirContents`, `copyFile` (to be extracted)
- `github.com/adrg/xdg` available for `xdg.DataHome`

## Deliverables

1. `internal/plugin/install.go` — extracted `CopyPluginDir(srcDir, pluginsDir string) (string, error)`
   and `copyFile(src, dest string) error` helpers (exported `CopyPluginDir`, unexported `copyFile`);
   update `internal/services/plugin.go` to call `plugin.CopyPluginDir` instead of the local copies

2. `cmd/plugin_push.go` — `klados plugin push <archive> <oci://ref>`
   - Positional args: archive path (must end in `.oci.tar.gz` or `.oci.tar`), OCI ref
   - Flags: `-u/--username`, `-p/--password`, `-t/--token`, `--insecure`
   - Validates archive path exists and has correct extension before making any network call
   - Calls `plugin.PushToRegistry(archive, ref, opts)`
   - On `ErrAuthRequired`: prints `"authentication required — use --username/--password or --token"`
   - Exits 0 on success (prints `"pushed <ref>"`), non-zero on any error

3. `cmd/plugin_install.go` — `klados plugin install <path-or-oci://ref>`
   - Flags: same four auth flags as push
   - Routes by argument:
     - Starts with `oci://` → `plugin.PullFromRegistry(ref, pluginsDir, opts)` where
       `pluginsDir = filepath.Join(xdg.DataHome, "klados", "plugins")`
     - Is a directory → `plugin.CopyPluginDir(path, pluginsDir)`
     - Ends in `.oci.tar.gz` or `.oci.tar` → `plugin.Unpack(path, pluginsDir)`
     - Anything else → usage error
   - On `ErrAuthRequired`: prints `"authentication required — use --username/--password or --token"`
   - On success: prints `"installed plugin to <pluginsDir>/<name>"`
   - Exits non-zero with stderr message on any error

## Tests

- **Go unit test** (`cmd/plugin_install_test.go`, `cmd/plugin_push_test.go`) — argument
  validation only (no network calls):
  - `klados plugin push` with no args → usage error
  - `klados plugin push archive.zip oci://ref` (wrong extension) → error mentioning extension
  - `klados plugin install /nonexistent/path` → error (path does not exist)
  - `klados plugin install somefile.zip` → error (unrecognised format)
- **Manual verification**
  - `klados plugin push ./examples/plugin-node-annotator/node-annotator-0.1.0.oci.tar.gz oci://ghcr.io/<owner>/node-annotator:test --username <u> --password <p>` pushes successfully
  - `klados plugin install oci://ghcr.io/<owner>/node-annotator:test` installs to plugins dir
  - `klados plugin install ./examples/plugin-node-annotator` copies the directory
  - `klados plugin install ./examples/plugin-node-annotator/node-annotator-0.1.0.oci.tar.gz` unpacks

## Acceptance Criteria

- [ ] `klados plugin push --help` shows `--username`, `--password`, `--token`, `--insecure` flags
- [ ] `klados plugin install --help` shows the same four flags
- [ ] `klados plugin install oci://...` with no credentials attempts anonymous pull and returns
      clear error if registry requires auth
- [ ] `klados plugin install ./examples/plugin-node-annotator` exits 0 and prints install path
- [ ] Argument validation tests pass: `go test ./cmd/`
- [ ] `internal/services/plugin.go` no longer contains `copyDirContents` or `copyFile`
- [ ] `go test ./internal/services/` still passes after the extraction

## Definition of Done

`klados plugin push ./plugin.oci.tar.gz oci://ghcr.io/owner/plugin:v1 -u user -p token` pushes
and exits 0. `klados plugin install oci://ghcr.io/owner/plugin:v1` pulls and writes the plugin
to `$XDG_DATA_HOME/klados/plugins/`. `klados plugin install ./local-dir` copies it.
`klados plugin install ./plugin.oci.tar.gz` unpacks it. All four auth flags work on both commands.

## Known Gotchas

- **The trap**: the CLI install command calling into `PluginService`
  **Why**: `PluginService` is a Wails service with a full app lifecycle; spinning it up from a
  CLI command would drag in all of Wails, CGO, and the GUI.
  **What to do instead**: the CLI writes files to disk only. The running GUI picks up new plugins
  via the fsnotify watcher; on next launch the `Loader` finds them automatically.

- **The trap**: forgetting to extract `copyDirContents`/`copyFile` from `services/plugin.go`
  before writing `cmd/plugin_install.go`
  **Why**: if you write the install command first and duplicate the copy logic, you'll have three
  copies (services, cmd, and whatever you wrote). Extract first, then reference the shared helper.
  **What to do instead**: create `internal/plugin/install.go` with `CopyPluginDir` as the first
  step of this phase, update `services/plugin.go` to call it, run `go test ./internal/services/`
  to confirm no regression, then write the commands.

- **The trap**: accepting `oci://` in `plugin push` for the OCI ref argument but not stripping it
  **Why**: `PushToRegistry` in `remote.go` already strips the prefix; if you also strip it in the
  command, the ref arrives double-stripped. If you don't strip it in the command, it's consistent.
  **What to do instead**: do not strip in the command — let `PushToRegistry` handle it, as it does
  for `PullFromRegistry`. The convention is that callers always pass the `oci://` prefix; the
  remote layer always strips it internally.
