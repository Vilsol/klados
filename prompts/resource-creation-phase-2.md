# Phase 2 — Apply Manifest

Add server-side apply to the resource engine and build a new ApplyManifestDialog for multi-document YAML with file picker and clipboard paste support.

## First Action

Read `internal/resource/engine.go` and find the existing `Create` method — your new `Apply` method sits alongside it but uses `Patch` with `types.ApplyPatchType` instead of `Create`. Understanding the dynamic client usage pattern here is the starting point.

## Context

Phase 1 built the template system and refactored the CreateResourceDialog. This phase adds a complementary flow: applying multi-document YAML manifests (like `kubectl apply -f`). This is a separate dialog because the mental model is different — users are applying a complete manifest (possibly from a file), not creating a single resource from a template. Server-side apply with `fieldManager: "klados"` ensures proper field ownership tracking distinct from kubectl.

## Files to Read

- `internal/resource/engine.go` — **what to look for**: existing `Create` method's dynamic client usage pattern, how it resolves GVR to a namespaced/cluster-scoped client — your `Apply` method follows the same pattern but uses `Patch`
- `internal/services/resource.go` — **what to look for**: how `CreateResource` wraps the engine call, error handling, and context passing — `ApplyManifest` follows the same service-layer pattern but iterates over multiple documents
- `frontend/src/lib/components/CreateResourceDialog.svelte` — **what to look for**: dialog structure, CodeMirror integration, and how it calls backend methods — reference for building `ApplyManifestDialog` with similar patterns
- `packages/ui/src/lib/YAMLEditor.svelte` — **what to look for**: the CodeMirror YAML editor component's props and events — reuse this in the ApplyManifestDialog

## Source Documents

- `RESOURCE_CREATION_SPEC.md` — ApplyResult struct definition, server-side apply details (fieldManager, force flag), ApplyManifestDialog wireframe with file picker + results display
- `RESOURCE_CREATION_PHASES.md` — Phase 2 deliverables, test requirements, and handoff notes about YAML splitting edge cases
- `CLAUDE.md` — project conventions: Wails binding generation, Svelte 5 runes, slox logging

## What Exists

- **From Phase 1**: `Template` struct, `TemplateRegistry` with Register/GetTemplates/GenerateFromSchema/Plugin API, ~29 curated template files, refactored `CreateResourceDialog` with GVR → Template dropdowns, plugin `register_template` host API
- `ResourceEngine.Create` method using dynamic client
- `ResourceService.CreateResource` Wails-bound RPC
- `YAMLEditor.svelte` CodeMirror component with syntax highlighting, validation, diff view
- Wails native dialog API for file picking
- Command palette registration pattern (used by Create Resource in Phase 1)

## Deliverables

1. `ApplyResult` struct in `internal/resource/engine.go` with fields: GVR, Namespace, Name, Action (`"created"` | `"configured"` | `"unchanged"`), Error (string, empty on success)
2. `ResourceEngine.Apply(ctx, contextName, obj *unstructured.Unstructured) (*ApplyResult, error)` — uses dynamic client `Patch` with `types.ApplyPatchType`, `fieldManager: "klados"`, `force: true`. Determines action by comparing before/after or from API response.
3. `ResourceService.ApplyManifest(contextName, yaml string) ([]ApplyResult, error)` — splits multi-document YAML using `yaml.NewDecoder` with sequential `Decode` calls (not naive string split), parses each into `unstructured.Unstructured`, calls `Apply` per document, collects results. Partial failures do not abort remaining documents.
4. `ApplyManifestDialog.svelte` — new dialog component with: "Open File..." button (Wails native file picker), "Paste from Clipboard" button, YAML editor (reuses CodeMirror setup), dynamic "Apply (N resources)" button showing document count parsed from editor content, inline results section after apply showing per-document success/error
5. Command palette entry for "Apply Manifest"

## Tests

- **Go unit test (engine)**
  - `Apply` calls dynamic client `Patch` with `types.ApplyPatchType`, `metav1.PatchOptions{FieldManager: "klados", Force: boolPtr(true)}`
  - `Apply` returns `ApplyResult` with action `"created"` for new resources
  - `Apply` returns `ApplyResult` with action `"configured"` for updated resources
  - `Apply` populates `ApplyResult.Error` for invalid resources without returning a Go error (non-fatal)

- **Go unit test (service layer)**
  - `ApplyManifest` splits multi-document YAML on `---` correctly via sequential decode
  - Handles: leading `---`, trailing `---`, empty documents between separators, single document without `---`
  - Returns one `ApplyResult` per non-empty document
  - Partial failures: one bad document does not prevent remaining documents from applying
  - Empty/comment-only documents between `---` separators are skipped (no empty ApplyResult)

- **Frontend test (vitest)**
  - `ApplyManifestDialog` renders "Open File..." and "Paste from Clipboard" buttons
  - Pasting multi-document YAML into editor updates document count on Apply button
  - Apply button disabled when editor is empty
  - Results section appears after apply, showing per-document status with action and error

- **Manual verification**
  - Open Apply Manifest from command palette → paste multi-document YAML → apply → see results
  - Use file picker to load a `.yaml` file → content appears in editor
  - Apply manifest with one valid and one invalid document → partial success shown
  - Apply manifest updating existing resource → "configured" action displayed

## Acceptance Criteria

- [ ] `ResourceEngine.Apply` uses server-side apply with `fieldManager: "klados"` and `force: true`
- [ ] `ApplyManifest` correctly splits multi-document YAML and applies each independently
- [ ] Partial failures: one bad document does not prevent others from applying
- [ ] `ApplyManifestDialog` loads YAML from native file picker (Wails dialog)
- [ ] `ApplyManifestDialog` loads YAML from clipboard paste
- [ ] Apply button shows accurate document count parsed from editor content
- [ ] Results displayed inline with per-document action/error status
- [ ] Command palette "Apply Manifest" entry opens the dialog
- [ ] All Go unit tests and frontend vitest tests pass

## Definition of Done

A developer can open Apply Manifest from the command palette, load a multi-document YAML file via the native file picker (or paste from clipboard), see the document count on the Apply button, click Apply, and see per-document results inline showing whether each resource was created, configured, or failed. A manifest with a mix of valid and invalid documents applies the valid ones and reports errors for the invalid ones. All Go and frontend tests pass.

## Known Gotchas

- **Don't split YAML with string splitting on `---`**: Use `yaml.NewDecoder` with sequential `Decode()` calls. Naive string splitting breaks on `---` appearing inside YAML values (multiline strings, comments). The decoder handles all edge cases: leading/trailing separators, empty documents, documents that are only comments.

- **`force: true` takes over field ownership**: When applying with `force: true`, Klados will claim ownership of all fields in the manifest, even if they were previously managed by `kubectl` or another tool. This matches kubectl's behavior but can surprise users. This is the intended design — document it but don't try to prevent it.

- **Determining the `action` field in `ApplyResult`**: Server-side apply's PATCH response doesn't explicitly say "created" vs "configured". Check if the resource existed before the patch (GET first, or inspect the response's `creationTimestamp` vs `resourceVersion`). A simpler approach: attempt a GET before Apply — 404 means "created", 200 means "configured" or "unchanged".

- **Unstructured objects need `apiVersion` and `kind`**: After YAML decode into `map[string]any`, you must extract `apiVersion` and `kind` to resolve the GVR for the dynamic client. Use `runtime.DefaultUnstructuredConverter` or construct `unstructured.Unstructured` directly. Missing `apiVersion`/`kind` should produce a clear error in `ApplyResult.Error`, not a panic.
