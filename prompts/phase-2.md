# Phase 2 — OCI Remote Layer

Implement push and pull against real OCI registries using `oras.land/oras-go/v2`, with a
unified auth resolution chain — the shared backend that both the CLI commands (Phase 3) and
the GUI service (Phase 4) call into.

## First Action

Read `internal/plugin/packaging.go` in full — specifically the `Unpack` function's internal
structure where it reads `entries map[string][]byte` and the `extractTarGz` helper. Your new
`unpackDir` helper replicates the same extraction logic but reads blobs from an OCI layout
directory on disk instead of from a tar archive map.

## Context

The plugin system can already pack and unpack local `.oci.tar.gz` archives, but has no way to
communicate with an OCI registry. This phase adds that capability as a pure library layer in
`internal/plugin/remote.go`, kept deliberately free of Cobra and Wails concerns so it can be
imported by both the CLI commands and the GUI service. The auth chain — token → explicit
credentials → Docker config file → anonymous — must be resolved in one place here rather than
duplicated across callers.

## Files to Read

- `internal/plugin/packaging.go` — study `Unpack`'s `entries map[string][]byte` loop and the
  `extractTarGz` / `addBlob` helpers; `unpackDir` shares the same extraction logic but reads
  blobs from `blobs/sha256/<hash>` files instead of a map
- `go.mod` — where to add the `oras.land/oras-go/v2` dependency; check existing dependency
  versions to avoid conflicts
- `internal/plugin/packaging_test.go` — understand the test patterns (temp dirs, round-trip
  assertions) you'll replicate in `remote_test.go`

## What Exists

- `internal/plugin/packaging.go` — `Pack()`, `Unpack()`, all media type constants, OCI struct
  types (`ociLayout`, `ociManifest`, `ociIndex`, `ociDescriptor`)
- `internal/plugin/packaging_test.go` — passing round-trip and edge-case tests
- No `oras-go` dependency yet; `go.mod` does not reference it

## Deliverables

1. `oras.land/oras-go/v2` (and its transitive deps) added to `go.mod` / `go.sum`

2. `internal/plugin/remote.go` containing:

   **Types**
   - `RemoteOpts` struct: `Username, Password, Token string; Insecure bool`
   - `ErrAuthRequired` — exported sentinel error; exact message string must be
     `"authentication required"` (the frontend matches on this string in Phase 5)

   **Unexported helpers**
   - `resolveCredStore(opts RemoteOpts) credentials.Store` — priority order:
     1. `opts.Token` set → `credentials.NewStaticCredentials("", "", token)` (empty username,
        token as access token)
     2. `opts.Username` + `opts.Password` both set → `credentials.NewStaticCredentials`
     3. Neither set → `credentials.NewFileStore(dockerConfigPath())` where `dockerConfigPath()`
        returns `filepath.Join(os.Getenv("HOME"), ".docker", "config.json")`; this single call
        covers plain stored creds AND all configured credential helpers automatically
     4. Fallback to `credentials.Anonymous` if the file doesn't exist
   - `newRepo(ref string, opts RemoteOpts) (*remote.Repository, error)` — strips `oci://`
     prefix, parses the remaining reference, constructs a `remote.Repository` with
     `PlainHTTP: opts.Insecure` and the resolved credential store attached
   - `isAuthError(err error) bool` — returns true if the error (or any wrapped cause) contains
     `"unauthorized"` or `"authentication required"` in its message; used to map to
     `ErrAuthRequired`
   - `unpackDir(layoutDir, pluginsDir string) error` — reads `index.json` from `layoutDir`,
     resolves the first manifest blob from `blobs/sha256/<hash>`, extracts config + layers
     exactly as `Unpack` does, writing the plugin into `pluginsDir`

   **Exported functions**
   - `PullFromRegistry(ref, pluginsDir string, opts RemoteOpts) error` — creates a temp dir,
     opens it as an ORAS `layout.New` store (pull target), calls `oras.Copy` from remote to
     local store, then calls `unpackDir(tempDir, pluginsDir)`; maps auth errors via
     `isAuthError` → `ErrAuthRequired`
   - `PushToRegistry(archivePath, ref string, opts RemoteOpts) error` — extracts the
     `.oci.tar.gz` to a temp dir using the existing tar-reading code (reuse `entries` map
     approach from `Unpack`), writes each blob to `tempDir/blobs/sha256/<hash>` and writes
     `index.json` + `oci-layout`, opens temp dir as `layout.New`, resolves descriptor from
     index, calls `oras.Copy` from local layout to remote; maps auth errors → `ErrAuthRequired`
   - `SaveDockerCredentials(host, username, password string) error` — reads (or creates)
     `~/.docker/config.json`, sets `auths.<host>.auth` to `base64(username + ":" + password)`,
     writes back; must not clobber existing entries for other hosts

3. `internal/plugin/remote_test.go` — see Tests section

## Tests

- **Go unit test** (`internal/plugin/remote_test.go`) using `net/http/httptest` to simulate an
  OCI registry:
  - Anonymous pull: test server returns blobs without auth; `PullFromRegistry` succeeds and
    produces expected `manifest.json` in plugins dir
  - Basic auth: test server requires `Authorization: Basic ...`; `RemoteOpts{Username, Password}`
    produces correct header
  - Token auth: test server requires `Authorization: Bearer <token>`; `RemoteOpts{Token}` works
  - Docker config pickup: write a temp `config.json` with an `auths` entry; override
    `dockerConfigPath()` via env; verify credentials are used automatically
  - 401 response → `errors.Is(err, ErrAuthRequired)` is true
  - Push round-trip: `PushToRegistry` on a test `.oci.tar.gz`, verify the test server received
    the blobs; then `PullFromRegistry` from the same test server, verify plugin dir contents
  - `Insecure: true` connects to a plain-HTTP test server without TLS errors
  - `SaveDockerCredentials`: creates file when absent, merges without clobbering other hosts,
    encodes correctly

## Acceptance Criteria

- [ ] `go test ./internal/plugin/ -run TestRemote` passes
- [ ] Push + pull round-trip test produces identical `manifest.json` to the source archive
- [ ] `errors.Is(err, ErrAuthRequired)` is true for 401/403 responses
- [ ] `SaveDockerCredentials` test verifies merge behaviour (existing hosts preserved)
- [ ] `ErrAuthRequired.Error()` returns exactly `"authentication required"`
- [ ] `go test ./internal/plugin/` passes (no regressions to existing packaging tests)

## Definition of Done

`PullFromRegistry("oci://ghcr.io/...", tmpDir, RemoteOpts{})` pulls a plugin and writes it to
disk. `PushToRegistry("plugin.oci.tar.gz", "oci://ghcr.io/...", RemoteOpts{Username, Password})`
pushes without error. Auth failures return `ErrAuthRequired`. All tests in
`internal/plugin/` pass.

## Known Gotchas

- **The trap**: passing `oci://ghcr.io/foo/bar:v1` directly to ORAS reference parsing
  **Why**: ORAS does not understand the `oci://` scheme prefix; it parses bare registry
  references like `ghcr.io/foo/bar:v1`.
  **What to do instead**: strip `oci://` with `strings.TrimPrefix(ref, "oci://")` before any
  ORAS call.

- **The trap**: using `credentials.NewFileStore` without an explicit path
  **Why**: the implicit default resolves to the actual `~/.docker/config.json` on the running
  machine, making tests non-deterministic and environment-dependent.
  **What to do instead**: always pass an explicit path via a `dockerConfigPath()` helper that
  reads `$HOME`; in tests, `t.Setenv("HOME", tmpDir)` to redirect it.

- **The trap**: detecting auth errors by type-asserting on ORAS error types
  **Why**: different registries (GHCR, Docker Hub, self-hosted) return auth errors in different
  formats — some follow the OCI error spec, others return plain HTTP 401s with non-standard
  bodies.
  **What to do instead**: use `isAuthError` that checks both the typed `errcode.ErrorResponse`
  code AND the error message string as a fallback. Match on `"unauthorized"` and
  `"authentication required"`.

- **The trap**: implementing `unpackDir` from scratch without reusing packaging.go logic
  **Why**: `Unpack` and `unpackDir` share the same extraction steps (parse index → resolve
  manifest → extract config + layers). Duplicating them creates drift.
  **What to do instead**: extract the blob-extraction and layer-dispatch logic from `Unpack`
  into an unexported `extractPlugin(blobs func(digest string) []byte, pluginsDir string) error`
  helper that both `Unpack` and `unpackDir` call.
