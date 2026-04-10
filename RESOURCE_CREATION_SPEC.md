# Resource Creation & Template System

## Context
Klados has a basic `CreateResourceDialog` with a hardcoded YAML skeleton and clipboard import. This work extends it into a full resource creation system with per-GVR templates (builtin curated, schema-generated fallback, plugin-contributed), and adds a separate "Apply Manifest" flow for multi-document YAML with server-side apply semantics.

## Decisions

**Hybrid template resolution: curated → plugin → schema-generated fallback**
Builtin resources get hand-curated templates that include the fields users actually need (not just schema-required fields). Plugins can register additional templates for any GVR. For GVRs with no curated or plugin templates, a skeleton is auto-generated from the cluster's OpenAPI schema. This gives good DX for common types and automatic coverage for everything else (including CRDs).

**Multiple templates per GVR, no override**
All template sources coexist in a flat list. A Deployment might have "basic web server" (builtin), "GPU workload" (builtin), and "with Istio sidecar" (plugin) — all shown in the picker. No precedence or conflict resolution needed.

**Progressive dropdown UX in existing create dialog**
Two combo boxes at the top of the existing `CreateResourceDialog`: GVR selector → Template selector. Selecting a template populates the YAML editor. When opened from a resource list page, GVR is pre-filled. This unifies the "generic create" and "create from list page" flows in one component.

**Separate "Apply Manifest" dialog for multi-document YAML**
Multi-document apply (`---` separated) is a distinct operation from template-driven creation. Gets its own dialog with file picker and paste support. Uses server-side apply with `fieldManager: "klados"`.

**Templates stored Go-side, served via RPC**
Templates live in Go (embedded YAML files or constants) and are served to the frontend via Wails bindings. This allows templates to be cluster-version-aware and keeps GVR knowledge centralized with the existing descriptor registry.

## Rejected Alternatives

**Form-based ConfigMap/Secret creation**
Dedicated key-value editor dialogs would be friendlier for these two types but create a one-off UI pattern to maintain. Templates provide equivalent scaffolding with zero per-type frontend code.

**Pure schema-generated templates**
K8s builtin OpenAPI schemas mark very little as `required` — a Deployment skeleton would be technically valid but useless (missing `containers`, `image`, etc.). CRD schema quality varies wildly. Schema-only gives correctness but not usefulness.

**Frontend-side template storage**
Would be simpler (no RPC) but can't adapt to cluster version, can't do schema-generated fallback (needs cluster connection), and duplicates GVR knowledge that already lives in Go.

## Priorities & Tradeoffs
Optimized for **coverage and low maintenance** — schema fallback means every resource type (including unknown CRDs) gets a template without hand-authoring. Curated templates for builtins optimize for **usefulness over minimalism** — they include commonly-needed fields, not just required ones. Plugin extensibility is prioritized to prove the plugin system's value for non-enricher use cases.

## Potential Gotchas
- **OpenAPI schema fetching**: The cluster's OpenAPI v3 endpoint can be slow on large clusters with many CRDs. Cache schemas after first fetch per GVR, invalidate on reconnect.
- **Schema-generated templates for schema-less CRDs**: Some CRDs have no structural schema at all. Fallback to a bare `apiVersion/kind/metadata/spec: {}` skeleton.
- **Server-side apply field conflicts**: If a resource was previously managed by `kubectl` (fieldManager `"kubectl"`), applying with fieldManager `"klados"` can produce field ownership conflicts. Pass `force: true` to match kubectl's default behavior, or surface the conflict to the user.
- **Template YAML must not hardcode namespace**: Templates should use a placeholder or omit namespace — the dialog's namespace context (from the active namespace or the GVR dropdown) fills it in.
- **Plugin template registration timing**: Plugin templates must be available after plugin init but before the user opens the create dialog. The existing plugin lifecycle (init → enable) handles this — register templates during `plugin_init`.

## Implementation Details

### Template data model

```go
// internal/resource/template.go

type Template struct {
    GVR         string `json:"gvr"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Content     string `json:"content"` // YAML string
    Source      string `json:"source"`  // "builtin" | "schema" | "plugin:{name}"
}
```

### Template registry

```go
// internal/resource/template_registry.go

type TemplateRegistry struct {
    mu       sync.RWMutex
    builtin  map[string][]Template   // GVR → curated templates
    plugin   map[string][]Template   // GVR → plugin-contributed templates
    schemas  map[string]*spec.Schema // cached OpenAPI schemas per GVR
}

func (r *TemplateRegistry) Register(t Template)
func (r *TemplateRegistry) RegisterPlugin(pluginName string, t Template)
func (r *TemplateRegistry) UnregisterPlugin(pluginName string) // on plugin disable/unload
func (r *TemplateRegistry) GetTemplates(gvr string) []Template // curated + plugin; schema fallback if empty
func (r *TemplateRegistry) GenerateFromSchema(gvr string) (Template, error)
```

### Schema-generated fallback logic

```go
func (r *TemplateRegistry) GenerateFromSchema(gvr string) (Template, error) {
    // 1. Fetch OpenAPI v3 schema from cluster discovery (cached)
    // 2. Walk schema tree:
    //    - Emit apiVersion, kind from GVR
    //    - Emit metadata.name: "" (always)
    //    - Emit all properties marked "required"
    //    - Use "default" values where present
    //    - Type placeholders: string→"", int→0, bool→false, object→{}, array→[]
    // 3. Serialize to YAML
    // 4. Return Template{Source: "schema", Name: "Default", Description: "Auto-generated from schema"}
}
```

### Builtin curated templates

```
internal/resource/templates/
├── core.v1.configmaps.yaml
├── core.v1.secrets.yaml
├── core.v1.secrets_dockerconfigjson.yaml
├── core.v1.secrets_tls.yaml
├── core.v1.services_clusterip.yaml
├── core.v1.services_loadbalancer.yaml
├── core.v1.services_nodeport.yaml
├── core.v1.pods.yaml
├── core.v1.persistentvolumeclaims.yaml
├── core.v1.persistentvolumes_nfs.yaml
├── core.v1.persistentvolumes_hostpath.yaml
├── core.v1.serviceaccounts.yaml
├── core.v1.resourcequotas.yaml
├── core.v1.limitranges.yaml
├── apps.v1.deployments.yaml
├── apps.v1.deployments_worker.yaml
├── apps.v1.statefulsets.yaml
├── apps.v1.daemonsets.yaml
├── batch.v1.jobs.yaml
├── batch.v1.cronjobs.yaml
├── networking.k8s.io.v1.ingresses.yaml
├── networking.k8s.io.v1.networkpolicies.yaml
├── rbac.authorization.k8s.io.v1.roles.yaml
├── rbac.authorization.k8s.io.v1.rolebindings.yaml
├── rbac.authorization.k8s.io.v1.clusterroles.yaml
├── rbac.authorization.k8s.io.v1.clusterrolebindings.yaml
├── policy.v1.poddisruptionbudgets.yaml
├── autoscaling.v2.horizontalpodautoscalers.yaml
├── storage.k8s.io.v1.storageclasses.yaml
```

Each file is a YAML document with frontmatter-style comments:

```yaml
# name: Basic Web Server
# description: Deployment with a single container, service port, and resource limits
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ""
  labels:
    app: ""
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ""
  template:
    metadata:
      labels:
        app: ""
    spec:
      containers:
        - name: ""
          image: ""
          ports:
            - containerPort: 80
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
```

Embedded via `//go:embed templates/*.yaml` and parsed at startup.

### Service layer

```go
// internal/services/resource.go — new methods

func (s *ResourceService) GetTemplates(contextName, gvr string) ([]resource.Template, error)
func (s *ResourceService) GetAllTemplateGVRs(contextName string) ([]string, error) // for GVR dropdown
func (s *ResourceService) ApplyManifest(contextName string, yaml string) ([]ApplyResult, error)
```

### Server-side apply

```go
// internal/resource/engine.go — new method

type ApplyResult struct {
    GVR       string `json:"gvr"`
    Namespace string `json:"namespace"`
    Name      string `json:"name"`
    Action    string `json:"action"` // "created" | "configured" | "unchanged"
    Error     string `json:"error,omitempty"`
}

func (e *ResourceEngine) Apply(ctx context.Context, contextName string, obj *unstructured.Unstructured) (*ApplyResult, error) {
    // Uses dynamic client Patch with types.ApplyPatchType
    // fieldManager: "klados"
    // force: true (match kubectl behavior)
}
```

### Plugin host API extension

```
method: "register_template"
params: {
    "gvr": "apps.v1.deployments",
    "name": "With Istio Sidecar",
    "description": "Deployment pre-configured for Istio service mesh",
    "content": "apiVersion: apps/v1\nkind: Deployment\n..."
}
```

Called during `plugin_init`. Templates removed automatically on plugin unload via `UnregisterPlugin`.

### Frontend: CreateResourceDialog changes

```
┌─────────────────────────────────────────────┐
│ Create Resource                          [X] │
│                                              │
│ Resource Type: [apps.v1.deployments     ▼]  │
│ Template:      [Basic Web Server        ▼]  │
│                                              │
│ ┌──────────────────────────────────────────┐ │
│ │ apiVersion: apps/v1                      │ │
│ │ kind: Deployment                         │ │
│ │ metadata:                                │ │
│ │   name: ""            ← YAML editor      │ │
│ │ ...                                      │ │
│ └──────────────────────────────────────────┘ │
│                                              │
│              [Cancel]  [Create]              │
└─────────────────────────────────────────────┘
```

- GVR dropdown: populated from `GetAllTemplateGVRs()` + full discovery list
- Template dropdown: populated from `GetTemplates(gvr)` on GVR selection
- Selecting a template replaces editor content (with confirmation if editor is dirty)
- GVR pre-filled when opened from a resource list page
- Namespace injected from active context (not hardcoded in template)

### Frontend: ApplyManifestDialog (new)

```
┌─────────────────────────────────────────────┐
│ Apply Manifest                           [X] │
│                                              │
│ [Open File...]  [Paste from Clipboard]      │
│                                              │
│ ┌──────────────────────────────────────────┐ │
│ │ apiVersion: v1                           │ │
│ │ kind: Service                            │ │
│ │ ...                                      │ │
│ │ ---                                      │ │
│ │ apiVersion: apps/v1    ← YAML editor     │ │
│ │ kind: Deployment                         │ │
│ │ ...                                      │ │
│ └──────────────────────────────────────────┘ │
│                                              │
│ [Cancel]  [Apply (2 resources)]             │
│                                              │
│ Results:                                     │
│ ✓ Service/my-svc — created                  │
│ ✓ Deployment/my-app — configured            │
└─────────────────────────────────────────────┘
```

- File picker via Wails native dialog
- Paste from clipboard button
- Document count shown on Apply button (parse on edit, split on `---`)
- Results displayed inline after apply completes
- Errors shown per-document, non-fatal (partial success allowed)
- Command palette entries for both "Create Resource" and "Apply Manifest"

## Definition of Done
- [ ] `TemplateRegistry` with Register/GetTemplates/GenerateFromSchema
- [ ] Curated YAML templates for all builtin resource types (~20 templates covering common variants)
- [ ] Schema-generated fallback working for CRDs and uncommon types
- [ ] Plugin `register_template` host API call working end-to-end
- [ ] `CreateResourceDialog` refactored with GVR + Template dropdowns
- [ ] Pre-fill GVR when opened from resource list page
- [ ] `ApplyManifest` service method with server-side apply (`fieldManager: "klados"`, `force: true`)
- [ ] `ApplyManifestDialog` with file picker and paste support
- [ ] Multi-document splitting and per-document result reporting
- [ ] Command palette entries for both "Create Resource" and "Apply Manifest"
