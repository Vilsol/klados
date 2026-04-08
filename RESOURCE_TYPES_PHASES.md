# Resource Types — Implementation Phases

## Project Overview

Add 12 missing v2 Kubernetes resource types to Klados: NetworkPolicies, IngressClasses, EndpointSlices, Resource Quotas, Limit Ranges, HPAs, PDBs, Leases, MutatingWebhookConfigurations, ValidatingWebhookConfigurations, PriorityClasses, and RuntimeClasses. Each type needs a descriptor, optional enricher, sidebar entry, and — for 7 of the 12 — a custom detail panel. All work follows established patterns (Descriptor + Enricher + panel registration).

## Phase Map

```
Phase 1 — Backend + Sidebar (all 12 types)
  ├── Phase 2 — Simple panels (4 panels, parallel with Phase 3)
  └── Phase 3 — Complex panels (3 panels, parallel with Phase 2)
```

---

## Phase 1 — Backend & List Views

> Adds all 12 resource type descriptors, enrichers, sidebar entries, and panel registration stubs so every type is browsable in list view with correct columns.

| | |
|---|---|
| **Depends on** | none |
| **Parallel with** | nothing |

### Deliverables

- 12 new descriptors in `internal/resource/builtin.go` with columns, detail panels, and actions
- 10 new enricher files in `internal/resource/enrichers/` (RuntimeClasses needs no enricher, WebhookConfig enricher is shared for both Mutating and Validating)
  - `networkpolicy.go` — `podSelectorDisplay`, `policyTypesDisplay`, `ingressRuleCount`, `egressRuleCount`
  - `ingressclass.go` — `isDefault` (from annotation)
  - `endpointslice.go` — `serviceDisplay`, `portsDisplay`, `endpointCount`
  - `resourcequota.go` — `resourceCount`
  - `limitrange.go` — `limitCount`
  - `hpa.go` — `referenceDisplay`, `targetsDisplay` (handles all 4 metric source types)
  - `pdb.go` — `podSelectorDisplay`
  - `lease.go` — `leaseDurationDisplay`
  - `webhook.go` — `webhookCount` (shared for Mutating and Validating)
  - `priorityclass.go` — `globalDefaultDisplay`
- All enrichers registered in `RegisterBuiltin()`
- 12 new GVR entries in `Sidebar.svelte` under correct categories (Workloads: HPAs, PDBs; Networking: NetworkPolicies, IngressClasses, EndpointSlices; Config: ResourceQuotas, LimitRanges, Leases; Cluster: MutatingWebhookConfigs, ValidatingWebhookConfigs, PriorityClasses, RuntimeClasses)
- 7 new panel name entries in `ResourceDetail.svelte` `panelComponents` map and `panelLabels` — pointing to placeholder components that render "Coming soon" or similar, so the tabs exist but don't crash
- Regenerated Wails bindings (if any service signatures changed — unlikely since descriptors are loaded via existing `GetDescriptors()`)

### Tests

- **Go unit tests (`internal/resource/enrichers/`)**
  - `networkpolicy_test.go` — empty selector → `<all pods>`, matchLabels flattened, nil ingress key → "-", empty ingress array → "0", policyTypes inference from rules
  - `ingressclass_test.go` — annotation present → "Yes", missing annotation → ""
  - `endpointslice_test.go` — service from label, fallback to ownerRef, ports formatted as `name:port/protocol`, endpoint count
  - `resourcequota_test.go` — count of `spec.hard` keys
  - `limitrange_test.go` — count of `spec.limits` entries
  - `hpa_test.go` — referenceDisplay format, targetsDisplay for Resource/Pods/Object/External metric types, missing currentMetrics → "?", cap at 3 metrics with "..."
  - `pdb_test.go` — podSelectorDisplay from matchLabels
  - `lease_test.go` — leaseDurationSeconds formatted as "Xs" / "Xm"
  - `webhook_test.go` — webhookCount for both mutating and validating shapes
  - `priorityclass_test.go` — globalDefault true → "Yes", false/missing → ""

### Out of Scope

- Custom detail panel implementations (Phases 2 and 3)
- Edit actions on new resource types (edit via YAML tab is already available through the generic panel)
- HPA edit min/max replicas action (listed in FEATURES.md but excluded from this spec — read-only panels only)

### Acceptance Criteria

- [ ] All 12 resource types appear in the sidebar under correct categories
- [ ] Each type's list view renders with all specified columns
- [ ] Enricher-computed columns display correct values (verified via Go unit tests)
- [ ] Generic detail panels (overview, labels, events, yaml) work for all 12 types
- [ ] Custom panel tabs appear in detail view (placeholder content acceptable)
- [ ] `go test ./internal/resource/enrichers/... -v` passes
- [ ] `pnpm check` passes
- [ ] Existing tests unaffected

### Source Documents

- `RESOURCE_TYPES_SPEC.md` — full spec with all descriptor definitions, enricher pseudocode, and sidebar placement
- `internal/resource/builtin.go` — add descriptors and enricher registrations here
- `internal/resource/descriptor.go` — `Descriptor`, `Column`, `Action` type definitions
- `internal/resource/enrichers/service.go` — reference enricher implementation pattern
- `internal/resource/enrichers/node.go` — reference for enricher with multiple computed fields
- `frontend/src/lib/components/Sidebar.svelte` — `gvrGroups` map to update
- `frontend/src/lib/components/ResourceDetail.svelte` — `panelComponents` and `panelLabels` maps

### Handoff Notes

- The HPA enricher's `targetsDisplay` is the most complex enricher in this batch. It must handle 4 metric source types (`Resource`, `Pods`, `Object`, `External`), each with different spec paths. The test file should cover all 4. See the `autoscaling/v2` API reference for the exact field paths.
- The NetworkPolicy enricher must distinguish between a nil `ingress`/`egress` key (policy doesn't affect that direction) and an empty array (deny all). The enricher stores "-" for nil and "0" for empty.
- The WebhookConfig enricher is a single struct registered for both GVRs. The `webhooks` field name is the same in both `MutatingWebhookConfiguration` and `ValidatingWebhookConfiguration`.
- Panel placeholder components should be minimal — a `<div>` with a message is fine. They exist so the tab structure is correct and the `visiblePanels` filter in `ResourceDetail.svelte` doesn't hide the tabs.

---

## Phase 2 — Simple Custom Panels

> Implements 4 custom detail panels that render structured data as tables, bars, and badges: ResourceQuota usage bars, LimitRange matrix, PDB disruption budget, and EndpointSlice address table.

| | |
|---|---|
| **Depends on** | Phase 1 |
| **Parallel with** | Phase 3 |

### Deliverables

- **`ResourceQuotaPanel.svelte`** — Usage bars
  - Scopes section (if `spec.scopeSelector` or `spec.scopes` present) with scope badges
  - One row per resource in `status.hard`: resource name, used value, hard value, percentage bar
  - Bar color: green (<70%), yellow (70-90%), red (>90%)
  - Missing `status.used` value → "0" with gray bar

- **`LimitRangePanel.svelte`** — Matrix table
  - One section per `spec.limits[]` entry with Type badge header (Container / Pod / PersistentVolumeClaim)
  - Table columns: Resource, Default, Default Request, Min, Max, Max Limit/Request Ratio
  - Rows: cpu, memory, storage — only those present in the entry
  - Sparse cells handled (PVC type has no cpu/memory)

- **`PDBPanel.svelte`** — Disruption budget
  - Selector section with label badges
  - Budget config: "Min Available: X" or "Max Unavailable: X"
  - Status bar: `currentHealthy / expectedPods` with proportional fill, green if `disruptionsAllowed > 0`, red if 0
  - Status fields grid: expectedPods, currentHealthy, desiredHealthy, disruptionsAllowed
  - Conditions section with status badges and messages

- **`EndpointSlicePanel.svelte`** — Address table
  - Ports summary row (name, port, protocol)
  - Address type indicator badge (IPv4/IPv6/FQDN)
  - Addresses table: Address, Node Name, Ready, Serving, Terminating, Target Ref
  - Ready/Serving/Terminating as green/yellow/red badges
  - Target Ref as clickable link navigating to pod detail (when `targetRef.kind == "Pod"`)

- Replace Phase 1 placeholder components with real implementations in `ResourceDetail.svelte` panel map

### Tests

- **Frontend type-check (`pnpm check`)**
  - All 4 new panel components type-check cleanly with their props interfaces

- **Manual verification**
  - ResourceQuotaPanel: bar colors change at 70% and 90% thresholds, missing used values show gray bar
  - LimitRangePanel: PVC-type entries show only storage rows, Container-type entries show cpu/memory
  - PDBPanel: status bar fill is proportional, red when 0 disruptions allowed
  - EndpointSlicePanel: target ref links navigate correctly, condition badges show correct colors

### Out of Scope

- NetworkPolicyPanel, HPAPanel, WebhookConfigPanel (Phase 3)
- Editable fields in any panel (all panels are read-only; edit via YAML tab)
- Cross-resource navigation from ResourceQuota/LimitRange panels (no links to affected pods)

### Acceptance Criteria

- [ ] ResourceQuotaPanel renders usage bars with correct colors for green/yellow/red thresholds
- [ ] ResourceQuotaPanel displays scopes when present, omits section when absent
- [ ] ResourceQuotaPanel handles missing `status.used` gracefully (shows "0")
- [ ] LimitRangePanel renders matrix table with per-type sections
- [ ] LimitRangePanel handles sparse cells (PVC without cpu/memory)
- [ ] PDBPanel status bar fill is proportional to currentHealthy/expectedPods
- [ ] PDBPanel shows conditions with status badges
- [ ] EndpointSlicePanel renders address table with Ready/Serving/Terminating badges
- [ ] EndpointSlicePanel target ref links navigate to pod detail view
- [ ] `pnpm check` passes
- [ ] Existing tests unaffected

### Source Documents

- `RESOURCE_TYPES_SPEC.md` — panel layouts and props interfaces for all 4 panels
- `frontend/src/lib/components/panels/ServicePanel.svelte` — reference panel pattern (props, layout, SectionHeader usage)
- `frontend/src/lib/components/panels/DeploymentPanel.svelte` — reference for conditions rendering
- `frontend/src/lib/components/ResourceDetail.svelte` — panel registration and props routing

### Handoff Notes

- The ResourceQuota `status.hard` and `status.used` fields are maps of resource name → quantity string. Quantity strings can be plain numbers ("10"), SI suffixes ("1Gi"), or decimal suffixes ("500m"). For the percentage bar, parse both values to a common numeric form. Kubernetes resource quantities follow the same format as `resource.Quantity` — consider a simple parser for `m` (milli), `Ki`/`Mi`/`Gi` (binary), and plain integers.
- EndpointSlice `targetRef` navigation: construct the URL as `/c/${ctxName}/${gvr}/${namespace}/${name}` where gvr is derived from `targetRef.kind` (e.g., Pod → `core.v1.pods`). The `ctxName` prop is needed for this.
- PDB `spec.minAvailable` and `spec.maxUnavailable` can be either an integer or a percentage string (e.g., "50%"). Display the raw value — don't try to resolve percentages to absolute numbers (that requires knowing the replica count of the target workload).

---

## Phase 3 — Complex Custom Panels

> Implements 3 custom detail panels with rich visualization: NetworkPolicy rule cards, HPA scaling detail with gauge and metrics, and WebhookConfig rule summary with collapsible sections.

| | |
|---|---|
| **Depends on** | Phase 1 |
| **Parallel with** | Phase 2 |

### Deliverables

- **`NetworkPolicyPanel.svelte`** — Rule visualization
  - "Applies to" section: pod selector labels as badges, empty selector → "All pods in namespace" warning badge
  - Policy types indicator: "Ingress", "Egress", or "Ingress + Egress"
  - Ingress rules section (if `policyTypes` includes Ingress):
    - Each rule as a card: FROM column (podSelector badges, namespaceSelector badges, ipBlock with CIDR + except) → arrow → PORTS column (port/protocol pairs)
    - Empty `ingress: []` → "Deny all ingress" red badge
    - Missing `ingress` key → section not shown
  - Egress rules section (same card pattern with TO)
  - Implicit deny footer: "All ingress/egress not explicitly allowed is denied"

- **`HPAPanel.svelte`** — Scaling detail
  - Scale target header: linked to referenced Deployment/StatefulSet/etc.
  - Replica gauge: `minReplicas ◄──[currentReplicas]──► maxReplicas` with proportional positioning, `desiredReplicas` marker if different from current
  - Metrics table: Type (badge), Name, Target, Current — one row per `spec.metrics[]` entry, all 4 source types handled (Resource, Pods, Object, External), missing currentMetrics → `<unknown>`
  - Scaling behavior section (if `spec.behavior` present): scaleUp/scaleDown policies with stabilization window, type, value, period, and selectPolicy indicator
  - Conditions section: AbleToScale, ScalingActive, ScalingLimited as status badges with messages

- **`WebhookConfigPanel.svelte`** — Webhook rule summary (shared for Mutating and Validating)
  - One collapsible section per webhook in `webhooks[]`
  - Header: webhook name + failure policy badge (Fail = red, Ignore = yellow warning)
  - Client config: Service as `namespace/name:port` + path, or raw URL; CA Bundle as "Present" / "Not set"
  - Match rules table: Operations (badges), API Groups, Resources, Scope — one row per `rules[]` entry, `*` highlighted
  - Selectors: namespace selector and object selector as label badges
  - Side effects badge (None / NoneOnDryRun / Unknown)
  - Timeout value
  - Match conditions (if present): CEL expressions listed

- Replace Phase 1 placeholder components with real implementations in `ResourceDetail.svelte` panel map

### Tests

- **Frontend type-check (`pnpm check`)**
  - All 3 new panel components type-check cleanly with their props interfaces

- **Manual verification**
  - NetworkPolicyPanel: empty `ingress: []` shows "Deny all ingress" red badge, nil ingress key hides section entirely, mixed FROM sources (podSelector + ipBlock) render in same card, empty podSelector shows warning badge
  - HPAPanel: gauge positions current replicas proportionally between min and max, all 4 metric types render with correct labels, missing behavior section doesn't break layout, scale target link navigates correctly
  - WebhookConfigPanel: collapsible sections expand/collapse, Ignore failure policy shows yellow warning, service and URL client configs render differently, `*` resources highlighted, works identically for both Mutating and Validating resources

### Out of Scope

- Cross-policy NetworkPolicy traffic matrix (future "Resource Relationships" feature)
- HPA edit min/max replicas action (read-only panels only)
- Webhook edit/create (edit via YAML tab)
- NetworkPolicy editor (read-only visualization only)

### Acceptance Criteria

- [ ] NetworkPolicyPanel distinguishes empty `ingress: []` (deny all) from missing `ingress` key (no ingress rules)
- [ ] NetworkPolicyPanel renders mixed FROM sources (podSelector + namespaceSelector + ipBlock) in same rule card
- [ ] NetworkPolicyPanel shows "All pods in namespace" warning for empty podSelector
- [ ] HPAPanel replica gauge positions current replicas proportionally between min/max
- [ ] HPAPanel handles all 4 metric source types (Resource, Pods, Object, External)
- [ ] HPAPanel shows scaling behavior section only when `spec.behavior` is present
- [ ] HPAPanel scale target link navigates to the referenced workload
- [ ] WebhookConfigPanel renders for both MutatingWebhookConfiguration and ValidatingWebhookConfiguration
- [ ] WebhookConfigPanel collapsible sections work (expand/collapse)
- [ ] WebhookConfigPanel shows failure policy badges with correct colors (Fail=red, Ignore=yellow)
- [ ] WebhookConfigPanel renders both service and URL client config variants
- [ ] `pnpm check` passes
- [ ] Existing tests unaffected

### Source Documents

- `RESOURCE_TYPES_SPEC.md` — panel layouts, props interfaces, and rendering details for all 3 panels
- `frontend/src/lib/components/panels/ServicePanel.svelte` — reference panel pattern
- `frontend/src/lib/components/panels/DeploymentPanel.svelte` — reference for conditions rendering and grid layout
- `frontend/src/lib/components/panels/RulesPanel.svelte` — reference for rendering RBAC rules (similar table pattern to webhook rules)
- `frontend/src/lib/components/ResourceDetail.svelte` — panel registration and props routing

### Handoff Notes

- The NetworkPolicy panel's most critical correctness requirement is the nil vs empty distinction for `ingress`/`egress`. In the unstructured object, a nil key means `spec.ingress` is absent from the JSON entirely (check with `_, found, _ := unstructured.NestedSlice(...)`), while an empty array means `spec.ingress` exists but is `[]`. The enricher in Phase 1 already encodes this as "-" vs "0" in the count columns, but the panel must also check the raw object for rendering decisions.
- The HPA panel needs `ctxName` for the scale target link. Construct the URL as `/c/${ctxName}/${gvr}/${namespace}/${name}` where gvr is derived from `scaleTargetRef.kind` + `scaleTargetRef.apiVersion` (e.g., `apps/v1` + `Deployment` → `apps.v1.deployments`). Use the GVR format helper from the registry if one exists.
- The WebhookConfig panel is a single component registered under the `webhooks` panel key. It works for both Mutating and Validating because the `webhooks[]` array has identical structure in both resource types. No conditional logic needed based on the resource kind.
- After Phase 2 and Phase 3 are both complete, all placeholder panel components from Phase 1 should be removed. Verify no placeholder imports remain in `ResourceDetail.svelte`.
