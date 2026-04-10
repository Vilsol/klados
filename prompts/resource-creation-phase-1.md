# Phase 1 — Template System & Create Dialog

Build the template registry, curated builtin templates, OpenAPI schema-generated fallback, plugin template registration, and refactor the CreateResourceDialog with progressive GVR → Template dropdowns.

## First Action

Read `internal/resource/engine.go` to understand the `ResourceEngine` struct and how it interacts with the cluster's dynamic client — your `TemplateRegistry` will live alongside it and need access to the same cluster connection for OpenAPI schema fetching.

## Context

Klados currently has a basic `CreateResourceDialog` with a hardcoded YAML skeleton (`apiVersion/kind/metadata/spec: {}`). Users must know the full resource spec from memory. This phase replaces that with a template-driven system: hand-curated templates for ~29 common builtin resource variants, auto-generated skeletons from OpenAPI schemas for CRDs, and a plugin API for third-party templates. The dialog gets two progressive dropdowns (Resource Type → Template) so users can pick what they're creating before touching YAML.

## Files to Read

- `internal/resource/engine.go` — **what to look for**: `ResourceEngine` struct, how it gets the dynamic client from `cluster.Manager`, and the existing `Create` method your templates will feed into
- `internal/resource/builtin.go` — **what to look for**: existing GVR string format (e.g. `apps.v1.deployments`) and `Descriptor` registration pattern — template GVR keys must match these exactly
- `internal/services/resource.go` — **what to look for**: `ResourceService` struct and how it wraps `ResourceEngine` — you'll add `GetTemplates` and `GetAllTemplateGVRs` methods here
- `internal/plugin/host_api.go` — **what to look for**: the `host_call` dispatch switch and JSON parameter pattern — add a `register_template` case following the same structure
- `frontend/src/lib/components/CreateResourceDialog.svelte` — **what to look for**: current dialog structure, how it calls `CreateResource`, and the hardcoded template string you'll replace
- `frontend/src/lib/registry/index.ts` — **what to look for**: `DescriptorRegistry` and `get(gvr)` — the GVR dropdown will pull from this plus template-specific GVR lists

## Source Documents

- `RESOURCE_CREATION_SPEC.md` — full spec with Template struct, TemplateRegistry API signatures, template file format (comment frontmatter), frontend wireframe, and plugin host API extension
- `RESOURCE_CREATION_PHASES.md` — phase plan with deliverables, test requirements, and handoff notes
- `CLAUDE.md` — project conventions: GVR dot-separated format, Wails binding generation, Svelte 5 runes, slox logging, `//go:embed` usage

## What Exists

- `CreateResourceDialog.svelte` with a hardcoded YAML skeleton and clipboard import
- `ResourceEngine.Create` method that takes an unstructured object and creates it via dynamic client
- `ResourceService.CreateResource` Wails-bound RPC wrapping the engine
- `DescriptorRegistry` on the frontend with GVR → Descriptor mappings loaded from Go
- Plugin host API dispatch in `host_api.go` with JSON-based method routing
- Plugin lifecycle (init → enable → disable → unload) with `UnregisterPlugin`-style cleanup patterns in the enricher registry

## Deliverables

1. `Template` struct in `internal/resource/template.go` with fields: GVR, Name, Description, Content (YAML string), Source (`"builtin"` | `"schema"` | `"plugin:{name}"`)
2. `TemplateRegistry` in `internal/resource/template_registry.go` with methods: `Register(t)`, `RegisterPlugin(pluginName, t)`, `UnregisterPlugin(pluginName)`, `GetTemplates(gvr) []Template`, `GenerateFromSchema(gvr) (Template, error)`
3. ~29 curated YAML template files in `internal/resource/templates/`, embedded via `//go:embed templates/*.yaml` — each file uses `# name:` and `# description:` comment lines as metadata, followed by the resource YAML (no namespace hardcoded)
4. Schema-generated fallback in `GenerateFromSchema`: fetch OpenAPI v3 schema from cluster discovery (cached per GVR, invalidated on reconnect), walk required fields, emit type-appropriate placeholders, fall back to bare `apiVersion/kind/metadata/spec: {}` for schema-less CRDs
5. `ResourceService.GetTemplates(contextName, gvr) ([]Template, error)` and `ResourceService.GetAllTemplateGVRs(contextName) ([]string, error)` — Wails-bound RPC methods
6. Plugin host API `register_template` method dispatched in `host_api.go` — called during `plugin_init`, templates auto-removed on plugin unload via `UnregisterPlugin`
7. Refactored `CreateResourceDialog.svelte` with: GVR dropdown (populated from `GetAllTemplateGVRs` + discovery), Template dropdown (populated from `GetTemplates` on GVR selection), YAML editor below. Template selection replaces editor content (confirm if dirty). GVR pre-filled when opened from resource list page. Namespace injected from active context.
8. Command palette entry for "Create Resource" opening the dialog

## Tests

- **Go unit test (template registry)**
  - `Register` adds templates; `GetTemplates` returns them for the correct GVR
  - `RegisterPlugin` adds plugin templates; `UnregisterPlugin` removes only that plugin's templates
  - `GetTemplates` returns curated + plugin templates combined for a GVR
  - `GetTemplates` falls back to `GenerateFromSchema` when no curated/plugin templates exist
  - `GenerateFromSchema` produces valid YAML with apiVersion/kind/metadata.name from a schema
  - `GenerateFromSchema` returns bare skeleton for schema-less CRDs
  - Embedded template files all parse correctly (name/description extracted, YAML content valid)

- **Go unit test (service layer)**
  - `GetTemplates` delegates to registry and returns results
  - `GetAllTemplateGVRs` returns union of builtin + plugin + discovered GVRs

- **Go unit test (plugin host API)**
  - `register_template` host call adds template to registry; appears in `GetTemplates`
  - Plugin unload removes all that plugin's templates

- **Frontend test (vitest)**
  - `CreateResourceDialog` renders GVR and Template dropdowns
  - Selecting a GVR populates Template dropdown with matching templates
  - Selecting a template populates YAML editor with template content
  - Opening with pre-filled GVR shows templates immediately
  - Dirty editor triggers confirmation before template switch

- **Manual verification**
  - Open Create Resource from command palette → select GVR → pick template → YAML appears
  - Open Create Resource from resource list page → GVR pre-filled
  - Connect to cluster with CRDs → schema-generated templates appear for CRDs without curated templates

## Acceptance Criteria

- [ ] `TemplateRegistry` with full Register/Get/Plugin/Schema API, backed by Go unit tests
- [ ] All ~29 curated template files embedded and parseable at startup
- [ ] Schema fallback generates usable skeleton for CRD with schema, bare skeleton for CRD without
- [ ] Plugin `register_template` host API works end-to-end (register → appears in GetTemplates → removed on unload)
- [ ] `CreateResourceDialog` shows GVR dropdown → Template dropdown → YAML editor flow
- [ ] GVR pre-filled when opened from resource list page
- [ ] Templates contain no hardcoded namespaces; namespace injected by dialog
- [ ] Command palette "Create Resource" entry opens the dialog
- [ ] All Go unit tests and frontend vitest tests pass

## Definition of Done

A developer can open the Create Resource dialog (from command palette or resource list), pick any resource type from the GVR dropdown, choose from one or more curated templates (or a schema-generated one for CRDs), see the YAML populated in the editor with the active namespace injected, edit it, and create the resource. A plugin can register additional templates via `register_template` during init and they appear in the picker. All Go and frontend tests pass.

## Known Gotchas

- **Template metadata conflicts with resource YAML**: Templates use `# name:` and `# description:` comment lines rather than YAML frontmatter. If you use `---` frontmatter, it'll be parsed as a YAML document separator and break the template content. The parser must strip comment-based metadata lines before serving content to the frontend.

- **OpenAPI schema caching must be per-connection**: Schema cache lives on `TemplateRegistry` and is populated lazily. If you cache globally without invalidation, reconnecting to a different cluster (or the same cluster after an upgrade) will serve stale schemas. Invalidate the schema cache whenever the cluster connection changes.

- **`GetAllTemplateGVRs` must merge multiple sources**: The GVR dropdown shows the union of GVRs from builtin templates, plugin templates, and cluster discovery. Don't just return template GVRs — include all discovered GVRs so users can create resources that only have a schema-generated fallback.

- **Template `Source` field format for plugins**: Use `"plugin:{name}"` (e.g. `"plugin:istio"`) not just `"plugin"`. This is needed for `UnregisterPlugin` to identify which templates belong to which plugin without scanning content.
