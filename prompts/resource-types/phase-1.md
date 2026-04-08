# Phase 1 — Backend & List Views

Add descriptors, enrichers, and sidebar entries for 12 new Kubernetes resource types so every type is browsable in list view with correct columns and generic detail panels.

## First Action

Read `internal/resource/enrichers/service.go` to see the full enricher pattern — struct, `Enrich` method signature, how `unstructured.NestedSlice`/`NestedField`/`SetNestedField` are used to read and write computed fields. Every enricher you write in this phase follows this exact pattern.

## Context

Klados has a well-established pipeline: a `Descriptor` in `builtin.go` defines columns and panels, an optional `Enricher` injects computed display fields into unstructured objects, and the frontend auto-renders list views from descriptors. 12 resource types are missing from the application: NetworkPolicies, IngressClasses, EndpointSlices, Resource Quotas, Limit Ranges, HPAs, PDBs, Leases, MutatingWebhookConfigurations, ValidatingWebhookConfigurations, PriorityClasses, and RuntimeClasses. This phase adds all backend infrastructure so they appear in the sidebar and render correct list views.

## Files to Read

- `internal/resource/enrichers/service.go` — **what to look for**: the `Enrich(_ string, obj *unstructured.Unstructured) error` method pattern, how `portsDisplay` and `externalIPDisplay` are computed and set via `unstructured.SetNestedField`
- `internal/resource/builtin.go` — **what to look for**: how existing descriptors are structured (Columns, DetailPanels, Actions, ClusterScoped), and the `RegisterBuiltin()` function where enrichers are registered
- `internal/resource/descriptor.go` — **what to look for**: `Descriptor`, `Column`, `Action`, `RenderType`, `AlignType` type definitions and available constants
- `internal/resource/enrichers/node.go` — **what to look for**: an enricher with multiple computed fields as a reference for the more complex enrichers (HPA, NetworkPolicy)
- `frontend/src/lib/components/Sidebar.svelte` — **what to look for**: the `gvrGroups` map structure to understand where to add new GVR entries
- `frontend/src/lib/components/ResourceDetail.svelte` — **what to look for**: the `panelComponents` Map and `panelLabels` Record to understand how panel names map to components

## Source Documents

- `RESOURCE_TYPES_SPEC.md` — complete spec with all 12 descriptor definitions (columns, detail panels, actions), enricher pseudocode, sidebar placement, and panel registration details
- `RESOURCE_TYPES_PHASES.md` — phase plan; Phase 1 section has the full deliverables list and enricher field specifications

## What Exists

- Established descriptor + enricher + panel pipeline handling ~20 existing resource types
- `RegisterBuiltin()` function in `builtin.go` that registers all descriptors and enrichers
- `EnricherRegistry` with `Register(gvr string, enricher Enricher)` API
- Sidebar with existing categories: Workloads, Networking, Config, Storage, Cluster, RBAC
- `ResourceDetail.svelte` with `panelComponents` map and conditional panel rendering
- Existing enricher test files as reference patterns

## Deliverables

1. **12 descriptors** added to `builtin.go` — each with Name column, relevant data columns, Age column, detail panel list, and delete action. Cluster-scoped types (IngressClasses, MutatingWebhookConfigs, ValidatingWebhookConfigs, PriorityClasses, RuntimeClasses) set `ClusterScoped: true`. Non-cluster-scoped types include a hidden Namespace column.

2. **`networkpolicy.go`** enricher — computes `status.podSelectorDisplay` (flatten matchLabels, empty selector → `<all pods>`), `status.policyTypesDisplay` (join policyTypes), `status.ingressRuleCount` (nil key → "-", empty array → "0"), `status.egressRuleCount` (same nil/empty distinction).

3. **`ingressclass.go`** enricher — computes `status.isDefault` from annotation `ingressclass.kubernetes.io/is-default-class` ("true" → "Yes", otherwise → "").

4. **`endpointslice.go`** enricher — computes `status.serviceDisplay` (from label `kubernetes.io/service-name`, fallback to ownerReferences), `status.portsDisplay` (format as `name:port/protocol`), `status.endpointCount`.

5. **`resourcequota.go`** enricher — computes `status.resourceCount` from `len(spec.hard)`.

6. **`limitrange.go`** enricher — computes `status.limitCount` from `len(spec.limits)`.

7. **`hpa.go`** enricher — computes `status.referenceDisplay` (format `scaleTargetRef` as `Kind/Name`), `status.targetsDisplay` (summarize `spec.metrics[]` + `status.currentMetrics[]` for all 4 metric source types: Resource, Pods, Object, External; cap at 3 with "..."; missing currentMetrics → "?").

8. **`pdb.go`** enricher — computes `status.podSelectorDisplay` from `spec.selector.matchLabels`.

9. **`lease.go`** enricher — computes `status.leaseDurationDisplay` (format `spec.leaseDurationSeconds` as "Xs" or "Xm").

10. **`webhook.go`** enricher (shared) — computes `status.webhookCount` from `len(webhooks)`. Registered for both `admissionregistration.k8s.io.v1.mutatingwebhookconfigurations` and `validatingwebhookconfigurations`.

11. **`priorityclass.go`** enricher — computes `status.globalDefaultDisplay` (true → "Yes", false/missing → "").

12. **All enrichers registered** in `RegisterBuiltin()`. RuntimeClasses need no enricher.

13. **Sidebar entries** in `Sidebar.svelte` — Workloads: HPAs, PDBs; Networking: NetworkPolicies, IngressClasses, EndpointSlices; Config: ResourceQuotas, LimitRanges, Leases; Cluster: MutatingWebhookConfigs, ValidatingWebhookConfigs, PriorityClasses, RuntimeClasses.

14. **Panel stubs** in `ResourceDetail.svelte` — register 7 placeholder components for panel names `netpol`, `endpointslice`, `resourcequota`, `limitrange`, `hpa`, `pdb`, `webhooks` with corresponding labels ("Rules", "Addresses", "Usage", "Limits", "Scaling", "Budget", "Webhooks"). Placeholders render minimal content so tabs appear without crashing.

## Tests

- **Go unit tests (`internal/resource/enrichers/`)**
  - `networkpolicy_test.go` — empty selector → `<all pods>`, matchLabels flattened to `key=val, ...`, nil ingress key → "-", empty ingress array → "0", policyTypes inferred when missing
  - `ingressclass_test.go` — annotation present and "true" → "Yes", annotation missing → "", annotation "false" → ""
  - `endpointslice_test.go` — service name from label, fallback to ownerRef when label missing, ports formatted as `name:port/protocol`, endpoint count as string
  - `resourcequota_test.go` — count matches number of keys in `spec.hard`
  - `limitrange_test.go` — count matches number of entries in `spec.limits`
  - `hpa_test.go` — referenceDisplay as "Deployment/nginx", targetsDisplay for Resource type ("cpu: 60%/80%"), Pods type, Object type, External type, missing currentMetrics → "?", >3 metrics → "..." suffix
  - `pdb_test.go` — matchLabels flattened, empty selector handled
  - `lease_test.go` — 30 seconds → "30s", 120 seconds → "2m", nil → ""
  - `webhook_test.go` — count of webhooks array, works for both mutating and validating object shapes
  - `priorityclass_test.go` — globalDefault true → "Yes", false → "", field missing → ""

## Acceptance Criteria

- [ ] All 12 resource types appear in the sidebar under correct categories
- [ ] Each type's list view renders with all specified columns
- [ ] Enricher-computed columns display correct values
- [ ] Generic detail panels (overview, labels, events, yaml) work for all 12 types
- [ ] Custom panel tabs appear in detail view with placeholder content
- [ ] `go test ./internal/resource/enrichers/... -v` passes with all new test files
- [ ] `pnpm check` passes
- [ ] Existing tests unaffected

## Definition of Done

All 12 resource types are visible in the sidebar, each renders a list view with the specified columns (including enricher-computed values), clicking any row opens a detail view with working overview/labels/events/yaml tabs plus stub tabs for custom panels. Running `go test ./internal/resource/enrichers/... -v` shows all new enricher tests passing.

## Known Gotchas

- **NetworkPolicy nil vs empty ingress/egress**: A nil `ingress` key (absent from JSON) means "policy doesn't affect ingress direction." An empty `ingress: []` means "deny all ingress." Use `unstructured.NestedSlice(obj.Object, "spec", "ingress")` — check the `found` bool (second return) to distinguish nil from empty. Store "-" for nil, "0" for empty. Getting this wrong breaks the panel visualization in Phase 3.

- **HPA has 4 metric source types with different spec paths**: `Resource` uses `spec.metrics[].resource`, `Pods` uses `spec.metrics[].pods`, `Object` uses `spec.metrics[].object`, `External` uses `spec.metrics[].external`. Each has different subfields for target and current values. The enricher must switch on the `type` field. Don't assume only `Resource` type exists.

- **IngressClass default is an annotation, not a spec field**: The `is-default` flag lives at `metadata.annotations["ingressclass.kubernetes.io/is-default-class"]`, not in `spec`. Read from annotations map, not nested spec fields.

- **WebhookConfig enricher is shared**: Register a single `WebhookConfigEnricher{}` instance for both `admissionregistration.k8s.io.v1.mutatingwebhookconfigurations` and `validatingwebhookconfigurations`. The `webhooks` field name is identical in both resource types.

- **Lease `renewTime` precision**: Kubernetes stores `renewTime` as MicroTime (RFC3339 with microseconds). The `age` render type should handle this, but verify with a test object that the CEL expression `spec.renewTime` evaluates correctly through the existing age formatter.
