# Resource Creation & Template System — Phased Plan

## Project Overview

Extend Klados's basic `CreateResourceDialog` into a full template-driven resource creation system with curated builtin templates, OpenAPI schema-generated fallback for CRDs, and plugin-contributed templates. Add a separate "Apply Manifest" flow for multi-document YAML with server-side apply semantics. Two phases: template infrastructure + dialog refactor first, then multi-document apply.

## Phase Map

```
Phase 1 — Template System & Create Dialog
  └── Phase 2 — Apply Manifest
```

---

## Phase 1 — Template System & Create Dialog

> Builds the template registry, curated builtin templates, schema-generated fallback, plugin template registration, and refactors the CreateResourceDialog with GVR + Template dropdowns.

| | |
|---|---|
| **Depends on** | none |
| **Parallel with** | nothing |

### Deliverables

- `Template` struct and `TemplateRegistry` in `internal/resource/` with `Register`, `RegisterPlugin`, `UnregisterPlugin`, `GetTemplates`, `GenerateFromSchema` methods
- ~29 curated YAML template files in `internal/resource/templates/`, embedded via `//go:embed`, covering all builtin resource types with common variants (e.g. Service has ClusterIP/LoadBalancer/NodePort, Secret has Opaque/dockerconfigjson/TLS)
- Schema-generated fallback: `GenerateFromSchema` walks OpenAPI v3 schema from cluster discovery, emits required fields with type-appropriate placeholders, caches schemas per GVR
- `GetTemplates(contextName, gvr)` and `GetAllTemplateGVRs(contextName)` methods on `ResourceService`
- Plugin host API `register_template` method — plugins call during `plugin_init`, templates removed on plugin unload
- Refactored `CreateResourceDialog.svelte` with two progressive combo boxes (Resource Type → Template), YAML editor below
- GVR pre-fill when dialog opened from a resource list page
- Namespace injected from active context (not hardcoded in templates)
- Command palette entry for "Create Resource"

### Tests

- **Go unit test**
  - `TemplateRegistry.Register` adds templates, `GetTemplates` returns them grouped by GVR
  - `RegisterPlugin` / `UnregisterPlugin` correctly adds and removes plugin templates
  - `GetTemplates` returns curated + plugin templates combined; falls back to schema-generated when no curated/plugin templates exist
  - `GenerateFromSchema` produces valid YAML with apiVersion/kind/metadata.name for a given schema; handles schema-less CRDs (bare skeleton fallback)
  - Embedded template files parse correctly (name/description extracted from comment frontmatter, YAML content is valid)
- **Go unit test (service layer)**
  - `ResourceService.GetTemplates` delegates to registry, returns templates for known GVR
  - `ResourceService.GetAllTemplateGVRs` returns union of builtin + plugin + discovered GVRs
- **Go unit test (plugin host API)**
  - `register_template` host call adds template to registry; template appears in `GetTemplates` response
  - Plugin unload removes all templates registered by that plugin
- **Frontend test (vitest)**
  - `CreateResourceDialog` renders GVR and Template dropdowns
  - Selecting a GVR populates the template dropdown with matching templates
  - Selecting a template populates the YAML editor with template content
  - Opening dialog with pre-filled GVR skips to template selection
  - Dirty editor shows confirmation before template switch
- **Manual verification**
  - Open Create Resource from command palette, select a GVR, pick a template, verify YAML appears
  - Open Create Resource from a resource list page, verify GVR is pre-filled
  - Connect to a cluster with CRDs, verify schema-generated templates appear for CRDs without curated templates

### Out of Scope

- Multi-document YAML apply (`---` splitting, server-side apply) — Phase 2
- `ApplyManifestDialog` — Phase 2
- File picker for loading YAML from disk — Phase 2
- User-defined templates (saved to config) — not planned, may be future work

### Acceptance Criteria

- [ ] `TemplateRegistry` with full Register/Get/Plugin/Schema API, backed by Go unit tests
- [ ] All ~29 curated template files embedded and parseable at startup
- [ ] Schema fallback generates a usable skeleton for a CRD with a schema and a bare skeleton for a CRD without one
- [ ] Plugin `register_template` host API call works end-to-end (register during init, appears in GetTemplates, removed on unload)
- [ ] `CreateResourceDialog` shows GVR dropdown → Template dropdown → YAML editor flow
- [ ] GVR pre-filled when opened from resource list page
- [ ] Templates do not contain hardcoded namespaces; namespace injected by dialog
- [ ] Command palette "Create Resource" entry opens the dialog
- [ ] All Go unit tests and frontend vitest tests pass

### Source Documents

- `RESOURCE_CREATION_SPEC.md` — full spec with data model, registry API, template format, frontend wireframe
- `internal/resource/engine.go` — existing ResourceEngine, new methods will neighbor this
- `internal/resource/builtin.go` — existing resource descriptors, template GVR naming must match
- `internal/services/resource.go` — existing ResourceService, new GetTemplates/GetAllTemplateGVRs methods added here
- `internal/plugin/host_api.go` — existing host API dispatch, add `register_template` case
- `frontend/src/lib/components/CreateResourceDialog.svelte` — existing dialog to refactor
- `frontend/src/lib/registry/index.ts` — DescriptorRegistry, GVR list source for dropdown
- `CLAUDE.md` — project conventions (GVR format, Wails bindings, Svelte 5 runes, slox logging)

### Handoff Notes

- The `Template` struct and `TemplateRegistry` are the foundation Phase 2 builds on — don't change the `Template` type without considering Apply Manifest needs.
- `GetAllTemplateGVRs` returns the union of all GVRs that have at least one template (builtin or plugin) plus all discovered GVRs from the cluster. The frontend dropdown uses this to show every possible resource type.
- Template YAML files use `# name:` and `# description:` comment lines as metadata — this is a simple convention that avoids YAML frontmatter (which would conflict with the resource YAML itself). The parser must strip these before serving the content.
- Schema caching is per-GVR and invalidated on cluster reconnect. The cache lives on `TemplateRegistry` and is populated lazily on first `GetTemplates` call for a GVR with no curated/plugin templates.

---

## Phase 2 — Apply Manifest

> Adds server-side apply to the resource engine and a new ApplyManifestDialog for multi-document YAML with file picker and clipboard paste support.

| | |
|---|---|
| **Depends on** | Phase 1 |
| **Parallel with** | nothing |

### Deliverables

- `ResourceEngine.Apply` method using dynamic client `Patch` with `types.ApplyPatchType`, `fieldManager: "klados"`, `force: true`
- `ApplyResult` struct with GVR, Namespace, Name, Action (`created`/`configured`/`unchanged`), Error fields
- `ResourceService.ApplyManifest(contextName, yaml)` method that splits on `---`, parses each document, calls `Apply` per document, returns `[]ApplyResult`
- `ApplyManifestDialog.svelte` — new dialog with:
  - "Open File..." button using Wails native file picker
  - "Paste from Clipboard" button
  - YAML editor (reuses existing CodeMirror setup)
  - Dynamic "Apply (N resources)" button with document count parsed from editor content
  - Inline results display after apply (per-document success/error)
- Command palette entry for "Apply Manifest"

### Tests

- **Go unit test**
  - `ResourceEngine.Apply` calls dynamic client Patch with correct patch type, field manager, and force flag
  - `ResourceEngine.Apply` returns `ApplyResult` with action "created" for new resources, "configured" for existing
  - `ResourceEngine.Apply` returns error in `ApplyResult.Error` for invalid resources (does not abort batch)
- **Go unit test (service layer)**
  - `ResourceService.ApplyManifest` splits multi-document YAML on `---` correctly
  - Handles edge cases: leading `---`, trailing `---`, empty documents between separators, single document (no `---`)
  - Returns one `ApplyResult` per document; partial failures don't abort remaining documents
- **Frontend test (vitest)**
  - `ApplyManifestDialog` renders file picker and paste buttons
  - Pasting YAML into editor updates document count on Apply button
  - Apply button disabled when editor is empty
  - Results section appears after apply with per-document status
- **Manual verification**
  - Open Apply Manifest from command palette, paste a multi-document YAML, apply, verify results
  - Open a YAML file via file picker, verify content loads into editor
  - Apply a manifest with one valid and one invalid document, verify partial success reporting
  - Apply a manifest that updates an existing resource, verify "configured" action

### Out of Scope

- Dry-run / diff preview before apply — could be future enhancement
- Applying from a URL — not planned
- Watch/track applied resources after apply — existing watch infrastructure handles this automatically
- Directory watching (`Dir view` feature from FEATURES.md) — separate feature entirely

### Acceptance Criteria

- [ ] `ResourceEngine.Apply` uses server-side apply with `fieldManager: "klados"` and `force: true`
- [ ] `ApplyManifest` correctly splits multi-document YAML and applies each independently
- [ ] Partial failures: one bad document does not prevent others from applying
- [ ] `ApplyManifestDialog` loads YAML from native file picker (Wails dialog)
- [ ] `ApplyManifestDialog` loads YAML from clipboard paste
- [ ] Apply button shows accurate document count
- [ ] Results displayed inline with per-document action/error status
- [ ] Command palette "Apply Manifest" entry opens the dialog
- [ ] All Go unit tests and frontend vitest tests pass

### Source Documents

- `RESOURCE_CREATION_SPEC.md` — server-side apply details, ApplyResult struct, ApplyManifestDialog wireframe
- `internal/resource/engine.go` — existing ResourceEngine, add `Apply` method here
- `internal/services/resource.go` — existing ResourceService, add `ApplyManifest` method here (Phase 1 already adds template methods)
- `frontend/src/lib/components/CreateResourceDialog.svelte` — reference for dialog patterns (refactored in Phase 1)
- `packages/ui/src/lib/YAMLEditor.svelte` — existing YAML editor component, reuse in ApplyManifestDialog
- `CLAUDE.md` — project conventions

### Handoff Notes

- `fieldManager: "klados"` is a permanent choice — all resources applied through Klados will carry this field manager. This is intentional and desirable (distinguishes Klados-managed fields from kubectl-managed ones).
- `force: true` matches kubectl's default behavior but can cause field ownership takeover. If users report unexpected behavior with resources managed by other tools, this is the knob to revisit.
- Multi-document splitting must handle YAML edge cases: `---` at start/end of file, empty documents, documents that are just comments. Use `yaml.NewDecoder` with sequential `Decode` calls rather than naive string splitting.
