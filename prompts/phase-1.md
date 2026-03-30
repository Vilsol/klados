# Phase 1 — Cobra CLI Restructure

Replace the hand-rolled `flag`-based CLI in `main.go` and the standalone `cmd/pluginpack` binary
with a Cobra command tree so that `klados` works as both a GUI launcher and a proper CLI — the
structural foundation that all new subcommands in later phases depend on.

## First Action

Open `main.go` and read the `runPluginCLI` function and the `main()` body — these are the two
blocks you will split apart: the CLI logic moves into `cmd/`, the Wails startup becomes the root
command's `RunE`.

## Context

The project currently has two separate CLI entry points: a minimal `flag`-based `runPluginCLI`
in `main.go` (handles `pack` and `install`) and a standalone `cmd/pluginpack/main.go` binary.
Both are dead ends — neither can be extended cleanly. This phase replaces both with a single
Cobra command tree where `klados` with no arguments opens the GUI and every other invocation
is a proper subcommand with `--help`, flags, and clean error messages. All future CLI commands
(push, install from registry) are added as children of the `plugin` subgroup created here.

## Files to Read

- `main.go` — find the `runPluginCLI` function (everything before `cfg, err := config.Load()`)
  and the Wails startup block (everything after the CLI check); these become `cmd/plugin_pack.go`
  and `cmd/root.go` respectively
- `cmd/pluginpack/main.go` — understand what it does so you know what to delete; it's a thin
  wrapper around `plugin.Pack()` with no logic worth preserving

## What Exists

- `main.go` with hand-rolled `flag`-based `runPluginCLI` (pack + install) and full Wails startup
- `cmd/pluginpack/main.go` — standalone pack binary, calls `plugin.Pack(os.Args[1], true)`
- `internal/plugin/packaging.go` — `Pack()` and `Unpack()` functions the commands call into
- All existing services, stores, and tests passing

## Deliverables

1. `github.com/spf13/cobra` added to `go.mod` and `go.sum`
2. `main.go` reduced to exactly two things: the `//go:embed all:frontend/dist` declaration and
   a `cmd.Execute(assets)` call — nothing else
3. `cmd/root.go` — exports `Execute(assets embed.FS)`; root Cobra command whose `RunE` contains
   the Wails startup block moved verbatim from `main.go` (config load → session load → services
   construction → `application.New` → window creation → `app.Run()`)
4. `cmd/plugin.go` — `plugin` subcommand group with `Use: "plugin"`, no `RunE`; added to root
   in an `init()` or explicit registration
5. `cmd/plugin_pack.go` — `klados plugin pack <dir> [--no-compress]`; calls `plugin.Pack()`;
   prints the output path on success; exits non-zero with stderr message on failure
6. `cmd/pluginpack/` directory deleted entirely
7. Any `Taskfile.yml` or `mise.toml` entries that reference `pluginpack` updated to build/use
   `klados`

## Tests

- **Manual verification**
  - `klados` (no arguments) opens the Wails window
  - `klados plugin pack ./examples/plugin-node-annotator` produces `node-annotator-*.oci.tar.gz`
    in that directory
  - `klados plugin pack --no-compress ./examples/plugin-node-annotator` produces `*.oci.tar`
  - `klados plugin --help` lists `pack` as a subcommand
  - `klados --help` lists `plugin` as a subcommand
  - `klados plugin pack` (no dir arg) exits 1 with a usage message
- **Go unit test** — run `go test ./...` and verify no regressions; no new tests needed in
  this phase since `plugin.Pack()` is already tested in `internal/plugin/packaging_test.go`

## Acceptance Criteria

- [ ] `main.go` contains only the embed directive and `cmd.Execute(assets)`
- [ ] `cmd/pluginpack/` does not exist
- [ ] `klados plugin pack ./examples/plugin-node-annotator` produces the same archive as the
      old `pluginpack` binary did
- [ ] `klados --help` and `klados plugin --help` both render correctly
- [ ] `go test ./...` passes with no regressions

## Definition of Done

Running `klados` opens the Wails GUI. Running `klados plugin pack <dir>` prints the archive
path and exits 0. Running `klados plugin --help` shows a proper Cobra help block. The old
`cmd/pluginpack/` is gone. All existing Go tests pass.

## Known Gotchas

- **The trap**: moving `//go:embed all:frontend/dist` to `cmd/root.go`
  **Why**: Go's embed directive resolves paths relative to the file it's declared in; `cmd/` is
  a subdirectory, so `all:frontend/dist` would not resolve and `all:../frontend/dist` is
  explicitly forbidden.
  **What to do instead**: keep the embed declaration in `main.go`, pass `assets embed.FS` as
  a parameter to `cmd.Execute(assets)`, capture it in a closure inside `cmd/root.go`.

- **The trap**: Cobra printing its own usage on every non-zero return from `RunE`
  **Why**: Cobra's default `SilenceUsage` is false, so returning any error from `RunE` also
  prints the command usage — noisy for runtime errors.
  **What to do instead**: set `SilenceUsage: true` and `SilenceErrors: true` on the root
  command; handle error printing explicitly in `Execute()`.

- **The trap**: forgetting that `klados plugin install` already exists in the old `runPluginCLI`
  **Why**: the old hand-rolled CLI had both `pack` and `install`. The new `install` subcommand
  will be a proper implementation in Phase 3 — do NOT port the old `install` to Cobra in this
  phase. Leave `klados plugin install` absent for now; it ships in Phase 3.
