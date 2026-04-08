# Phase 1 — Go Backend

Establish the data model, config storage types, RPC surface, and Namespace columns that all subsequent frontend and enricher work depends on.

## First Action

Read `internal/resource/descriptor.go` to see the current `Column` struct — you'll add `Align AlignType` and `Hidden bool` fields to it, plus define the `AlignType` constants. Everything else in this phase extends from this struct change.

## Context

This is the first phase of the Resource List Column Improvements project. The resource list currently has fixed columns defined by `Descriptor` structs in Go. There's no way for users to configure column visibility, order, width, or alignment. This phase builds the Go-side foundation: the extended data model, the config storage shape for column preferences, and the RPC methods the frontend will call to read/write those preferences.

## Files to Read

- `internal/resource/descriptor.go` — **what to look for**: the `Column` struct definition and `RenderType` constants. You'll add `AlignType`, `Align`, and `Hidden` fields here.
- `internal/config/config.go` — **what to look for**: the `Config` struct and `Save`/`Update`/`Load` pattern. You'll add `ColumnPrefs` and `CompactRows` fields plus new supporting types.
- `internal/services/config.go` — **what to look for**: existing RPC method pattern (`GetTheme`/`SetTheme`). You'll replicate this for `GetColumnPrefs`/`SetColumnPrefs`/`GetCompactRows`/`SetCompactRows`.
- `internal/resource/builtin.go` — **what to look for**: the `builtinDescriptors` slice. Every non-`ClusterScoped` descriptor needs a `Namespace` column with `Hidden: true` added after `Name`.
- `frontend/src/lib/registry/index.ts` — **what to look for**: the `ColumnDef` interface and how descriptors are mapped in `load()`. You'll add `align?` and `hidden?` fields, plus a `defaultAlign()` helper.
- `internal/config/config_test.go` — **what to look for**: existing test patterns for config round-trip. You'll add tests for `ColumnPrefs` and `CompactRows`.

## Source Documents

- `RESOURCE_LIST_COLUMNS.md` — §Storage shape, §Go backend changes, §Column struct additions. The canonical reference for type definitions, field names, JSON tags, and RPC signatures.
- `PHASES.md` — Phase 1 section for deliverables, acceptance criteria, and handoff notes.

## What Exists

- `Column` struct with `Name`, `Expr`, `RenderType`, `Width` fields (no alignment or hidden support)
- `Config` struct with theme, kubeconfig, terminal, plugin, metrics fields (no column prefs)
- `ConfigService` with theme, WebGL, TLS skip, and config getter RPCs
- 22 builtin descriptors in `builtin.go` — none have a Namespace column
- `ColumnDef` TypeScript interface mirroring the Go `Column` struct (no `align`/`hidden`)
- Wails bindings already generated for current service methods

## Deliverables

1. `AlignType` string type with constants `AlignLeft`, `AlignRight`, `AlignCenter` in `descriptor.go`
2. `Align AlignType` and `Hidden bool` fields added to `Column` struct with JSON tags `"align,omitempty"` and `"hidden,omitempty"`
3. `ColumnSettings` struct (with `Width int`), `GVRColumnPrefs` struct (with `Columns`, `Order`, `Sort`), and `SortPrefs` struct (with `Column`, `Direction`) in `config/config.go`
4. `ColumnPrefs map[string]*GVRColumnPrefs` and `CompactRows bool` fields on `Config` struct
5. Four new RPC methods on `ConfigService`: `GetColumnPrefs(gvr string) *config.GVRColumnPrefs`, `SetColumnPrefs(gvr string, prefs *config.GVRColumnPrefs) error`, `GetCompactRows() bool`, `SetCompactRows(compact bool) error`
6. Namespace column `{Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150, Hidden: true}` added to every non-cluster-scoped builtin descriptor, positioned after the Name column
7. Regenerated Wails TypeScript bindings via `wails3 generate bindings`
8. `ColumnDef` interface updated with `align?: AlignType` and `hidden?: boolean` fields; `AlignType` type and `defaultAlign(renderType)` helper exported from `registry/index.ts`
9. `load()` in `DescriptorRegistry` updated to map `align` and `hidden` from Go descriptors

## Tests

- **Go unit test (config)**
  - `TestColumnPrefsRoundTrip` — save a Config with populated `ColumnPrefs`, reload from disk, assert deep equality
  - `TestMissingColumnPrefsDefaultsGracefully` — load a config JSON that has no `columnPrefs` key, verify the field is nil/empty (not an error)
  - `TestCompactRowsDefault` — load default config, verify `CompactRows` is `false`

- **Go unit test (descriptor)**
  - `TestColumnAlignDefault` — create a Column with no Align set, verify it marshals with `align` absent from JSON
  - `TestNamespaceColumnOnAllNamespacedDescriptors` — iterate `builtinDescriptors`, for each non-`ClusterScoped` descriptor assert it has a column named "Namespace" with `Hidden: true`
  - `TestClusterScopedDescriptorsHaveNoNamespaceColumn` — iterate `builtinDescriptors`, for each `ClusterScoped` descriptor assert no column named "Namespace" exists

- **Frontend type check**
  - `cd frontend && pnpm check` passes cleanly

## Acceptance Criteria

- [ ] `Column` struct has `Align AlignType` and `Hidden bool` fields with JSON tags `"align,omitempty"` and `"hidden,omitempty"`
- [ ] `Config` struct has `ColumnPrefs map[string]*GVRColumnPrefs` and `CompactRows bool` fields
- [ ] `ConfigService` has `GetColumnPrefs`, `SetColumnPrefs`, `GetCompactRows`, `SetCompactRows` methods
- [ ] Every non-cluster-scoped builtin descriptor has a `Namespace` column with `Hidden: true`
- [ ] No cluster-scoped descriptor has a `Namespace` column
- [ ] `wails3 generate bindings` succeeds without errors
- [ ] `go test ./internal/config/ ./internal/resource/ -v` passes
- [ ] `cd frontend && pnpm check` passes

## Definition of Done

Running `go test ./internal/config/ ./internal/resource/ -v` passes all existing and new tests. The generated Wails bindings include TypeScript signatures for `GetColumnPrefs`, `SetColumnPrefs`, `GetCompactRows`, `SetCompactRows`. The frontend type-checks cleanly with the new `align` and `hidden` fields on `ColumnDef`. Inspecting `builtin.go` shows every namespaced descriptor has a hidden Namespace column and no cluster-scoped descriptor does.

## Known Gotchas

- **`Align` must be optional (empty string = use frontend default).** If you set a default value on the Go side, every existing column definition would need to be updated. Leave it as empty string — the frontend applies `defaultAlign(renderType)` when `align` is falsy. The spec is explicit about this: age→right, everything else→left.

- **`GVRColumnPrefs.Order` semantics: presence = visible.** The `Order` array only contains names of visible columns. If `Order` is nil/empty, the frontend falls back to descriptor column order (excluding `Hidden: true` columns). Don't confuse this with "all columns in some order" — absence from Order means hidden.

- **Config migration: existing `config.json` files have no `columnPrefs` field.** The `Load()` function uses `json.Unmarshal` which leaves missing fields as zero values. The code must handle `ColumnPrefs == nil` gracefully (treat as "no preferences, use defaults"). Don't add migration logic — just nil-check.

- **Wails binding regeneration may add new model files.** After adding `GVRColumnPrefs` and related types, `wails3 generate bindings` will create new TypeScript model files. Make sure to check `frontend/bindings/` for the generated types and verify their shapes match what the frontend expects.
