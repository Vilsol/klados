# Missing v2 Resource Types

## Context

Klados has well-established patterns for adding resource types: a `Descriptor` in `builtin.go` (columns, detail panels, actions), optional enrichers for computed display fields, and frontend detail panel components registered in `ResourceDetail.svelte`. The existing pipeline auto-renders list views from descriptors and routes detail views through the panel map.

This spec adds 12 new resource types with list columns, enrichers, and custom detail panels where warranted. All follow existing patterns. The scope includes:

- **Networking**: NetworkPolicies, IngressClasses, EndpointSlices
- **Config**: Resource Quotas, Limit Ranges, Leases
- **Workloads**: HPAs, PDBs
- **Cluster**: MutatingWebhookConfigurations, ValidatingWebhookConfigurations, PriorityClasses, RuntimeClasses

## Decisions

**EndpointSlices only, no legacy Endpoints**
Endpoints are legacy since Kubernetes 1.21. EndpointSlices are the modern replacement and the only type worth adding.

**Custom detail panels for 7 of 12 types**
NetworkPolicies (rule visualization), EndpointSlices (address table), Resource Quotas (usage bars), Limit Ranges (matrix table), HPAs (gauge + metrics + scaling behavior), PDBs (disruption budget visual), and Webhook Configurations (rule summary table) all benefit from structured rendering beyond the generic overview. The remaining 5 (IngressClasses, Leases, PriorityClasses, RuntimeClasses) use generic panels only.

**Shared webhook panel component**
MutatingWebhookConfigurations and ValidatingWebhookConfigurations share identical structure. A single `WebhookConfigPanel.svelte` handles both, receiving the object and rendering the webhook array.

**HPAs target autoscaling/v2, not v2beta2**
`autoscaling/v2` is GA since Kubernetes 1.23. No need to support the beta versions.

**NetworkPolicy visualization is per-policy, not cross-policy**
The detail panel shows ingress/egress rule cards for a single policy. A cross-policy traffic matrix (showing all policies in a namespace and how they interact) is deferred to a future "RBAC Visualization"-style feature.

## Rejected Alternatives

**Graphical network policy editor**
Too complex for this pass. The rule card visualization is read-only and covers the debugging use case (understanding what a policy allows/blocks).

**Shared "admission" category in sidebar**
Webhook configurations could get their own sidebar category, but they fit naturally under Cluster (cluster-scoped infrastructure). Adding a category for just two resource types creates unnecessary navigation depth.

## Priorities & Tradeoffs

- **Breadth over depth**: All 12 types ship with list views. Custom panels prioritized by debugging value (HPAs, NetworkPolicies, Quotas) over completeness.
- **Pattern reuse over novelty**: Every enricher and panel follows existing conventions exactly. No new infrastructure.
- **Read-only detail panels**: No edit actions in the custom panels (edit via YAML tab). Keeps scope bounded.

## Potential Gotchas

- **NetworkPolicy empty rules semantics**: An empty `ingress: []` array means "deny all ingress," but an omitted `ingress` key means "no ingress rules, don't affect ingress." The panel must distinguish these cases visually — both render as "no rules listed" if not handled carefully. Check `spec.policyTypes` to determine intent.
- **HPA metric types**: `autoscaling/v2` supports 4 metric source types (Resource, Pods, Object, External), each with different spec shapes. The enricher and panel must handle all four, not just Resource.
- **Resource Quota scopes**: Quotas can have `scopeSelector` (e.g., only count PriorityClass=high pods). The usage bar panel should display scope when present, not just resource names.
- **Limit Range types**: The `type` field is one of `Container`, `Pod`, `PersistentVolumeClaim` — each has different valid resource fields. The matrix table must handle sparse cells (e.g., PVC type only has `min`/`max` for storage, no CPU/memory).
- **EndpointSlice address types**: Can be `IPv4`, `IPv6`, or `FQDN`. The address table should display the type and handle mixed slices.
- **Webhook `clientConfig` variants**: Either `service` (in-cluster reference) or `url` (external). The panel must render both forms. `caBundle` should be shown as "present" (not the raw base64).
- **Lease `renewTime` is a MicroTime**: Kubernetes stores it as an RFC3339 timestamp with microseconds. The `age` render type should handle this, but verify the CEL expression works with the precision.
- **IngressClass default annotation**: The `is-default` flag is an annotation (`ingressclass.kubernetes.io/is-default-class: "true"`), not a spec field. Enricher must read from `metadata.annotations`, not `spec`.

## Implementation Details

### Sidebar Registration

In `frontend/src/lib/components/Sidebar.svelte`, add to existing `gvrGroups`:

```typescript
const gvrGroups: Record<string, string[]> = {
  Workloads: [
    // ... existing ...
    'autoscaling.v2.horizontalpodautoscalers',
    'policy.v1.poddisruptionbudgets',
  ],
  Networking: [
    // ... existing ...
    'networking.k8s.io.v1.networkpolicies',
    'networking.k8s.io.v1.ingressclasses',
    'discovery.k8s.io.v1.endpointslices',
  ],
  Config: [
    // ... existing ...
    'core.v1.resourcequotas',
    'core.v1.limitranges',
    'coordination.k8s.io.v1.leases',
  ],
  Cluster: [
    // ... existing ...
    'admissionregistration.k8s.io.v1.mutatingwebhookconfigurations',
    'admissionregistration.k8s.io.v1.validatingwebhookconfigurations',
    'scheduling.k8s.io.v1.priorityclasses',
    'node.k8s.io.v1.runtimeclasses',
  ],
}
```

### Descriptors (`internal/resource/builtin.go`)

#### NetworkPolicies

```go
{
    Group: "networking.k8s.io", Version: "v1", Resource: "networkpolicies",
    Kind: "NetworkPolicy",
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 250},
        {Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150, Hidden: true},
        {Name: "Pod Selector", Expr: "status.podSelectorDisplay", RenderType: RenderText, Width: 200},
        {Name: "Policy Types", Expr: "status.policyTypesDisplay", RenderType: RenderText, Width: 150},
        {Name: "Ingress Rules", Expr: "status.ingressRuleCount", RenderType: RenderText, Width: 100},
        {Name: "Egress Rules", Expr: "status.egressRuleCount", RenderType: RenderText, Width: 100},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "netpol", "labels", "events", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

#### IngressClasses

```go
{
    Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses",
    Kind: "IngressClass",
    ClusterScoped: true,
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 250},
        {Name: "Controller", Expr: "spec.controller", RenderType: RenderText, Width: 300},
        {Name: "Default", Expr: "status.isDefault", RenderType: RenderBadge, Width: 80},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "labels", "events", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

#### EndpointSlices

```go
{
    Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices",
    Kind: "EndpointSlice",
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 250},
        {Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150, Hidden: true},
        {Name: "Address Type", Expr: "addressType", RenderType: RenderBadge, Width: 100},
        {Name: "Service", Expr: "status.serviceDisplay", RenderType: RenderText, Width: 200},
        {Name: "Ports", Expr: "status.portsDisplay", RenderType: RenderText, Width: 150},
        {Name: "Endpoints", Expr: "status.endpointCount", RenderType: RenderText, Width: 80},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "endpointslice", "labels", "events", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

#### Resource Quotas

```go
{
    Group: "", Version: "v1", Resource: "resourcequotas",
    Kind: "ResourceQuota",
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 250},
        {Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150, Hidden: true},
        {Name: "Resources", Expr: "status.resourceCount", RenderType: RenderText, Width: 80},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "resourcequota", "labels", "events", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

#### Limit Ranges

```go
{
    Group: "", Version: "v1", Resource: "limitranges",
    Kind: "LimitRange",
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 250},
        {Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150, Hidden: true},
        {Name: "Limits", Expr: "status.limitCount", RenderType: RenderText, Width: 80},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "limitrange", "labels", "events", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

#### HPAs

```go
{
    Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers",
    Kind: "HorizontalPodAutoscaler",
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 250},
        {Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150, Hidden: true},
        {Name: "Reference", Expr: "status.referenceDisplay", RenderType: RenderText, Width: 200},
        {Name: "Targets", Expr: "status.targetsDisplay", RenderType: RenderText, Width: 200},
        {Name: "Min", Expr: "spec.minReplicas", RenderType: RenderText, Width: 60},
        {Name: "Max", Expr: "spec.maxReplicas", RenderType: RenderText, Width: 60},
        {Name: "Current", Expr: "status.currentReplicas", RenderType: RenderText, Width: 70},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "hpa", "labels", "events", "metrics", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

#### PDBs

```go
{
    Group: "policy", Version: "v1", Resource: "poddisruptionbudgets",
    Kind: "PodDisruptionBudget",
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 250},
        {Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150, Hidden: true},
        {Name: "Min Available", Expr: "spec.minAvailable", RenderType: RenderText, Width: 110},
        {Name: "Max Unavailable", Expr: "spec.maxUnavailable", RenderType: RenderText, Width: 120},
        {Name: "Allowed Disruptions", Expr: "status.disruptionsAllowed", RenderType: RenderText, Width: 140},
        {Name: "Current Healthy", Expr: "status.currentHealthy", RenderType: RenderText, Width: 120},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "pdb", "labels", "events", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

#### Leases

```go
{
    Group: "coordination.k8s.io", Version: "v1", Resource: "leases",
    Kind: "Lease",
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 300},
        {Name: "Namespace", Expr: "metadata.namespace", RenderType: RenderText, Width: 150, Hidden: true},
        {Name: "Holder", Expr: "spec.holderIdentity", RenderType: RenderText, Width: 300},
        {Name: "Duration", Expr: "status.leaseDurationDisplay", RenderType: RenderText, Width: 100},
        {Name: "Renew", Expr: "spec.renewTime", RenderType: RenderAge, Width: 100, Align: AlignRight},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "labels", "events", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

#### MutatingWebhookConfigurations

```go
{
    Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations",
    Kind: "MutatingWebhookConfiguration",
    ClusterScoped: true,
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 350},
        {Name: "Webhooks", Expr: "status.webhookCount", RenderType: RenderText, Width: 80},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "webhooks", "labels", "events", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

#### ValidatingWebhookConfigurations

```go
{
    Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations",
    Kind: "ValidatingWebhookConfiguration",
    ClusterScoped: true,
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 350},
        {Name: "Webhooks", Expr: "status.webhookCount", RenderType: RenderText, Width: 80},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "webhooks", "labels", "events", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

#### PriorityClasses

```go
{
    Group: "scheduling.k8s.io", Version: "v1", Resource: "priorityclasses",
    Kind: "PriorityClass",
    ClusterScoped: true,
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 300},
        {Name: "Value", Expr: "value", RenderType: RenderText, Width: 120},
        {Name: "Global Default", Expr: "status.globalDefaultDisplay", RenderType: RenderBadge, Width: 120},
        {Name: "Preemption", Expr: "preemptionPolicy", RenderType: RenderBadge, Width: 140},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "labels", "events", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

#### RuntimeClasses

```go
{
    Group: "node.k8s.io", Version: "v1", Resource: "runtimeclasses",
    Kind: "RuntimeClass",
    ClusterScoped: true,
    Columns: []Column{
        {Name: "Name", Expr: "metadata.name", RenderType: RenderText, Width: 250},
        {Name: "Handler", Expr: "handler", RenderType: RenderText, Width: 200},
        {Name: "Age", Expr: "metadata.creationTimestamp", RenderType: RenderAge, Width: 100, Align: AlignRight},
    },
    DetailPanels: []string{"overview", "labels", "events", "yaml"},
    Actions: []Action{{Name: "delete", Label: "Delete"}},
}
```

### Enrichers (`internal/resource/enrichers/`)

#### `networkpolicy.go` (new)

```go
type NetworkPolicyEnricher struct{}

func (e *NetworkPolicyEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
    // status.podSelectorDisplay — flatten spec.podSelector.matchLabels to "key=val, ..."
    //   Empty selector → "<all pods>"

    // status.policyTypesDisplay — join spec.policyTypes with ", "
    //   Missing policyTypes → infer from presence of ingress/egress rules

    // status.ingressRuleCount — string(len(spec.ingress))
    //   nil ingress key (not empty array) → "-"

    // status.egressRuleCount — string(len(spec.egress))
    //   nil egress key (not empty array) → "-"

    return nil
}
```

#### `ingressclass.go` (new)

```go
type IngressClassEnricher struct{}

func (e *IngressClassEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
    // status.isDefault — read metadata.annotations["ingressclass.kubernetes.io/is-default-class"]
    //   "true" → "Yes", otherwise → ""

    return nil
}
```

#### `endpointslice.go` (new)

```go
type EndpointSliceEnricher struct{}

func (e *EndpointSliceEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
    // status.serviceDisplay — read metadata.labels["kubernetes.io/service-name"]
    //   Missing → extract from ownerReferences where kind == "Service"

    // status.portsDisplay — flatten ports[] to "name:port/protocol, ..."
    //   e.g. "http:80/TCP, https:443/TCP"

    // status.endpointCount — string(len(endpoints))

    return nil
}
```

#### `resourcequota.go` (new)

```go
type ResourceQuotaEnricher struct{}

func (e *ResourceQuotaEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
    // status.resourceCount — string(len(spec.hard)) — number of tracked resources

    return nil
}
```

#### `limitrange.go` (new)

```go
type LimitRangeEnricher struct{}

func (e *LimitRangeEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
    // status.limitCount — string(len(spec.limits)) — number of limit entries

    return nil
}
```

#### `hpa.go` (new)

```go
type HPAEnricher struct{}

func (e *HPAEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
    // status.referenceDisplay — format spec.scaleTargetRef as "Kind/Name"
    //   e.g. "Deployment/nginx"

    // status.targetsDisplay — summarize spec.metrics[] + status.currentMetrics[]
    //   For each metric, show "current/target" with type prefix:
    //   Resource: "cpu: 60%/80%", "memory: 512Mi/1Gi"
    //   Pods: "packets-per-second: 1k/2k"
    //   Object: "requests-per-second: 100/200"
    //   External: "queue_messages: 30/50"
    //   Join with ", " — cap at 3, append "..." if more
    //   If status.currentMetrics is missing, show "target/?" for each

    return nil
}
```

#### `pdb.go` (new)

```go
type PDBEnricher struct{}

func (e *PDBEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
    // status.podSelectorDisplay — flatten spec.selector.matchLabels to "key=val, ..."

    return nil
}
```

#### `lease.go` (new)

```go
type LeaseEnricher struct{}

func (e *LeaseEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
    // status.leaseDurationDisplay — format spec.leaseDurationSeconds as "Xs" or "Xm"

    return nil
}
```

#### `webhook.go` (new — shared for both Mutating and Validating)

```go
type WebhookConfigEnricher struct{}

func (e *WebhookConfigEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
    // status.webhookCount — string(len(webhooks))
    //   Field is "webhooks" for both MutatingWebhookConfiguration and ValidatingWebhookConfiguration

    return nil
}
```

#### `priorityclass.go` (new)

```go
type PriorityClassEnricher struct{}

func (e *PriorityClassEnricher) Enrich(_ string, obj *unstructured.Unstructured) error {
    // status.globalDefaultDisplay — read globalDefault field
    //   true → "Yes", false/missing → ""

    return nil
}
```

#### Registration in `RegisterBuiltin()`

```go
// Networking
enricherReg.Register("networking.k8s.io.v1.networkpolicies", &enrichers.NetworkPolicyEnricher{})
enricherReg.Register("networking.k8s.io.v1.ingressclasses", &enrichers.IngressClassEnricher{})
enricherReg.Register("discovery.k8s.io.v1.endpointslices", &enrichers.EndpointSliceEnricher{})

// Config
enricherReg.Register("core.v1.resourcequotas", &enrichers.ResourceQuotaEnricher{})
enricherReg.Register("core.v1.limitranges", &enrichers.LimitRangeEnricher{})
enricherReg.Register("coordination.k8s.io.v1.leases", &enrichers.LeaseEnricher{})

// Workloads
enricherReg.Register("autoscaling.v2.horizontalpodautoscalers", &enrichers.HPAEnricher{})
enricherReg.Register("policy.v1.poddisruptionbudgets", &enrichers.PDBEnricher{})

// Cluster
enricherReg.Register("admissionregistration.k8s.io.v1.mutatingwebhookconfigurations", &enrichers.WebhookConfigEnricher{})
enricherReg.Register("admissionregistration.k8s.io.v1.validatingwebhookconfigurations", &enrichers.WebhookConfigEnricher{})
enricherReg.Register("scheduling.k8s.io.v1.priorityclasses", &enrichers.PriorityClassEnricher{})
// RuntimeClasses — no enricher needed, all columns are direct expressions
```

### Frontend Detail Panels

#### New panel components (`frontend/src/lib/components/panels/`)

**`NetworkPolicyPanel.svelte`** — Rule visualization

```typescript
interface Props { obj: Record<string, any> }
```

Layout:
- **Applies to** section: pod selector labels as badges. Empty selector → "All pods in namespace" warning badge.
- **Policy types** indicator: "Ingress", "Egress", or "Ingress + Egress"
- **Ingress rules** section (if policyTypes includes Ingress):
  - Each rule rendered as a card with:
    - **FROM** column: list of sources (podSelector badges, namespaceSelector badges, ipBlock with CIDR + except)
    - **Arrow** indicator
    - **PORTS** column: port/protocol pairs
  - Empty `ingress: []` → "Deny all ingress" red badge
  - Missing `ingress` key → section not shown
- **Egress rules** section (same pattern with TO instead of FROM)
- **Implicit deny** footer: "All ingress/egress not explicitly allowed is denied"

**`EndpointSlicePanel.svelte`** — Address table

```typescript
interface Props { obj: Record<string, any>; ctxName: string }
```

Layout:
- **Ports** summary row: name, port, protocol for each port in the slice
- **Addresses table**: columns = Address, Node Name, Ready, Serving, Terminating, Target Ref
  - Ready/Serving/Terminating as green/yellow/red condition badges
  - Target Ref as clickable link (navigates to pod detail if `targetRef.kind == "Pod"`)
- **Address type** indicator badge (IPv4/IPv6/FQDN)

**`ResourceQuotaPanel.svelte`** — Usage bars

```typescript
interface Props { obj: Record<string, any> }
```

Layout:
- **Scopes** section (if `spec.scopeSelector` or `spec.scopes` present): scope badges
- **Usage table**: one row per resource in `status.hard`
  - Resource name | Used | Hard | Percentage bar
  - Bar color: green (<70%), yellow (70-90%), red (>90%)
  - Used value from `status.used[resource]`, hard from `status.hard[resource]`
  - Missing used value → "0" with gray bar

**`LimitRangePanel.svelte`** — Matrix table

```typescript
interface Props { obj: Record<string, any> }
```

Layout:
- One section per entry in `spec.limits[]`
- Section header: Type badge (Container / Pod / PersistentVolumeClaim)
- Table: columns = Resource, Default, Default Request, Min, Max, Max Limit/Request Ratio
  - Rows: cpu, memory, storage (only those present)
  - Empty cells for inapplicable combinations (e.g., PVC type only has `min`/`max` for storage, no CPU/memory)

**`HPAPanel.svelte`** — Scaling detail

```typescript
interface Props { obj: Record<string, any>; ctxName: string }
```

Layout:
- **Scale target** header: link to the referenced Deployment/StatefulSet/etc.
- **Replica gauge**: visual bar showing `minReplicas ◄──[currentReplicas]──► maxReplicas`
  - Current position proportionally placed
  - `desiredReplicas` shown as a marker if different from current
- **Metrics table**: one row per `spec.metrics[]` entry
  - Columns: Type, Name, Target, Current
  - Type badge: Resource / Pods / Object / External
  - Target: "80% Utilization" or "1000 AverageValue" etc.
  - Current: matched from `status.currentMetrics[]` by index — `<unknown>` if missing
- **Scaling behavior** section (if `spec.behavior` present):
  - Scale Up: stabilization window, policies (type, value, period)
  - Scale Down: same
  - selectPolicy indicator (Max / Min / Disabled)
- **Conditions** section: AbleToScale, ScalingActive, ScalingLimited as status badges with message

**`PDBPanel.svelte`** — Disruption budget

```typescript
interface Props { obj: Record<string, any> }
```

Layout:
- **Selector** section: label badges (same pattern as NetworkPolicy)
- **Budget config**: "Min Available: X" or "Max Unavailable: X" (only one is set)
- **Status bar**: `[██████░░] currentHealthy / expectedPods healthy, disruptionsAllowed disruptions allowed`
  - Bar fill = currentHealthy / expectedPods
  - Color: green if disruptionsAllowed > 0, red if 0
- **Status fields** grid: expectedPods, currentHealthy, desiredHealthy, disruptionsAllowed
- **Conditions** section (if `status.conditions` present): condition badges with messages

**`WebhookConfigPanel.svelte`** — Webhook rule summary (shared)

```typescript
interface Props { obj: Record<string, any> }
```

Layout:
- One collapsible section per webhook in the `webhooks[]` array
- **Header**: webhook name + failure policy badge (Fail = red, Ignore = yellow warning)
- **Client config**:
  - Service: `namespace/name:port` + path
  - URL: the raw URL
  - CA Bundle: "Present" / "Not set" indicator
- **Match rules table**: one row per `rules[]` entry
  - Columns: Operations, API Groups, Resources, Scope
  - Operations as badges (CREATE, UPDATE, DELETE, CONNECT)
  - Resources as comma-separated list, `*` highlighted
- **Selectors**: namespace selector and object selector as label badges
- **Side effects**: badge (None / NoneOnDryRun / Unknown)
- **Timeout**: value + "s" suffix
- **Match conditions** (if present): CEL expressions listed

#### Panel registration in `ResourceDetail.svelte`

Add to `panelComponents` map:

```typescript
import NetworkPolicyPanel from './panels/NetworkPolicyPanel.svelte'
import EndpointSlicePanel from './panels/EndpointSlicePanel.svelte'
import ResourceQuotaPanel from './panels/ResourceQuotaPanel.svelte'
import LimitRangePanel from './panels/LimitRangePanel.svelte'
import HPAPanel from './panels/HPAPanel.svelte'
import PDBPanel from './panels/PDBPanel.svelte'
import WebhookConfigPanel from './panels/WebhookConfigPanel.svelte'

// Add to panelComponents Map:
'netpol'         → NetworkPolicyPanel
'endpointslice'  → EndpointSlicePanel
'resourcequota'  → ResourceQuotaPanel
'limitrange'     → LimitRangePanel
'hpa'            → HPAPanel
'pdb'            → PDBPanel
'webhooks'       → WebhookConfigPanel
```

Add to `panelLabels`:

```typescript
'netpol': 'Rules',
'endpointslice': 'Addresses',
'resourcequota': 'Usage',
'limitrange': 'Limits',
'hpa': 'Scaling',
'pdb': 'Budget',
'webhooks': 'Webhooks',
```

Panel props routing in the conditional render block — all new panels receive `obj`, and `HPAPanel` and `EndpointSlicePanel` additionally receive `ctxName` (for navigation links to referenced resources).

### File Summary

```
internal/resource/enrichers/
  networkpolicy.go        (new)
  ingressclass.go         (new)
  endpointslice.go        (new)
  resourcequota.go        (new)
  limitrange.go           (new)
  hpa.go                  (new)
  pdb.go                  (new)
  lease.go                (new)
  webhook.go              (new)
  priorityclass.go        (new)

internal/resource/builtin.go              (modified — 12 new descriptors + enricher registrations)

frontend/src/lib/components/panels/
  NetworkPolicyPanel.svelte    (new)
  EndpointSlicePanel.svelte    (new)
  ResourceQuotaPanel.svelte    (new)
  LimitRangePanel.svelte       (new)
  HPAPanel.svelte              (new)
  PDBPanel.svelte              (new)
  WebhookConfigPanel.svelte    (new)

frontend/src/lib/components/
  ResourceDetail.svelte        (modified — 7 new panel registrations)
  Sidebar.svelte               (modified — 12 new GVR entries)
```

## Definition of Done

### List Views (all 12 types)
- [ ] Descriptor registered in `builtin.go` with columns, detail panels, and actions
- [ ] Enricher registered (where applicable) with unit tests
- [ ] GVR appears in correct sidebar category
- [ ] List view renders with correct columns and sort
- [ ] Generic detail panels (overview, labels, events, yaml) work

### Custom Detail Panels (7 types)
- [ ] NetworkPolicyPanel: rule cards render for ingress/egress, pod selector badges, empty vs nil rule distinction works, implicit deny shown
- [ ] EndpointSlicePanel: address table with ready/serving/terminating badges, target ref links navigate to pods
- [ ] ResourceQuotaPanel: usage bars with color coding, scopes displayed when present, handles missing `status.used` gracefully
- [ ] LimitRangePanel: matrix table renders per-type section, sparse cells handled (PVC has no cpu/memory rows)
- [ ] HPAPanel: replica gauge renders, all 4 metric source types handled, scaling behavior shown, conditions displayed, scale target links to resource
- [ ] PDBPanel: status bar renders, selector badges, conditions displayed
- [ ] WebhookConfigPanel: works for both Mutating and Validating, rule table renders, failure policy badge, service vs URL client config

### Tests
- [ ] All new enrichers have Go unit tests (`go test ./internal/resource/enrichers/... -v`)
- [ ] Frontend type-checks cleanly (`pnpm check`)
- [ ] Existing tests pass
