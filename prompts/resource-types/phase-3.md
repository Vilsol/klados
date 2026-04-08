# Phase 3 — Complex Custom Panels

Implement 3 custom detail panels with rich visualization: NetworkPolicy rule cards, HPA scaling detail with replica gauge and metrics, and WebhookConfig rule summary with collapsible sections.

## First Action

Read `frontend/src/lib/components/panels/RulesPanel.svelte` to see how RBAC rules are rendered as a structured table with badges — the WebhookConfig panel follows a very similar pattern (operations as badges, resources in columns, collapsible sections). This is the closest existing reference for the visualization complexity in this phase.

## Context

Phase 1 added descriptors, enrichers, and sidebar entries for 12 new resource types. All types are browsable with correct list columns and generic detail panels. This phase replaces the remaining 3 placeholder panel components with rich visualizations: NetworkPolicy gets rule cards showing ingress/egress flows, HPA gets a scaling gauge with metrics and behavior details, and WebhookConfig gets a collapsible rule summary. These are the most rendering-intensive panels in the batch.

## Files to Read

- `frontend/src/lib/components/panels/RulesPanel.svelte` — **what to look for**: how RBAC rules are rendered as a table with expandable details, badge rendering for verbs/resources — similar pattern needed for webhook operations and network policy ports
- `frontend/src/lib/components/panels/DeploymentPanel.svelte` — **what to look for**: conditions rendering as status badges with messages (needed for HPA conditions: AbleToScale, ScalingActive, ScalingLimited), grid layout for status fields
- `frontend/src/lib/components/ResourceDetail.svelte` — **what to look for**: the conditional render block for props routing — HPAPanel needs `ctxName` for scale target navigation links
- `frontend/src/routes/routes.ts` — **what to look for**: URL structure `/c/:ctx/:gvr/:ns/:name` for constructing navigation links (HPA scale target, NetworkPolicy pod selector links)

## Source Documents

- `RESOURCE_TYPES_SPEC.md` — detailed panel layouts for NetworkPolicyPanel (rule card wireframe), HPAPanel (gauge, metrics table, scaling behavior, conditions), and WebhookConfigPanel (collapsible webhook sections, match rules table, client config variants)
- `RESOURCE_TYPES_PHASES.md` — Phase 3 section with deliverables, acceptance criteria, and handoff notes (especially nil vs empty ingress/egress distinction, HPA scale target URL construction, shared webhook panel)

## What Exists

- All 12 resource type descriptors, enrichers, and sidebar entries (Phase 1)
- Placeholder panel components registered in `ResourceDetail.svelte` for `netpol`, `hpa`, `webhooks` panel keys
- Panel labels already set: "Rules", "Scaling", "Webhooks"
- NetworkPolicy enricher already distinguishes nil vs empty ingress/egress (stored as "-" vs "0" in count columns)
- HPA enricher already computes `referenceDisplay` and `targetsDisplay`
- Generic detail panels (overview, labels, events, yaml) working for all types

## Deliverables

1. **`NetworkPolicyPanel.svelte`** — Rule visualization:
   - "Applies to" section with pod selector labels as badges; empty selector → "All pods in namespace" warning badge
   - Policy types indicator: "Ingress", "Egress", or "Ingress + Egress"
   - Ingress rules section (shown only if `policyTypes` includes Ingress):
     - Each rule rendered as a card with FROM column (podSelector badges, namespaceSelector badges, ipBlock with CIDR + except ranges) → arrow → PORTS column (port/protocol pairs)
     - Empty `ingress: []` → "Deny all ingress" red badge
     - Missing `ingress` key (nil) → section not shown at all
   - Egress rules section with same card pattern using TO instead of FROM
   - Implicit deny footer: "All ingress/egress not explicitly allowed is denied"

2. **`HPAPanel.svelte`** — Scaling detail (requires `ctxName` prop):
   - Scale target header with clickable link to referenced Deployment/StatefulSet/etc.
   - Replica gauge: visual bar `minReplicas ◄──[currentReplicas]──► maxReplicas` with proportional positioning; `desiredReplicas` marker shown if different from current
   - Metrics table with columns: Type (badge), Name, Target, Current — one row per `spec.metrics[]` entry; all 4 source types handled (Resource, Pods, Object, External); missing `status.currentMetrics` → `<unknown>`
   - Scaling behavior section (only when `spec.behavior` present): scaleUp and scaleDown subsections showing stabilization window, policies (type, value, period), and selectPolicy indicator (Max / Min / Disabled)
   - Conditions section: AbleToScale, ScalingActive, ScalingLimited as status badges with messages

3. **`WebhookConfigPanel.svelte`** — Shared for both Mutating and Validating:
   - One collapsible section per webhook in `webhooks[]` array
   - Section header: webhook name + failure policy badge (Fail = red, Ignore = yellow warning)
   - Client config: Service rendered as `namespace/name:port` + path, OR raw URL; CA Bundle shown as "Present" / "Not set" indicator
   - Match rules table: Operations (as badges: CREATE, UPDATE, DELETE, CONNECT), API Groups, Resources (`*` highlighted), Scope — one row per `rules[]` entry
   - Namespace selector and object selector rendered as label badges
   - Side effects badge (None / NoneOnDryRun / Unknown)
   - Timeout value with "s" suffix
   - Match conditions listed as CEL expressions (if present)

4. **Replace placeholder components** in `ResourceDetail.svelte` panel map with real implementations. Update props routing — HPAPanel receives `ctxName` in addition to `obj`.

## Tests

- **Frontend type-check (`pnpm check`)**
  - All 3 new panel components type-check cleanly with their props interfaces

- **Manual verification**
  - NetworkPolicyPanel: create a policy with `ingress: []` and verify "Deny all ingress" red badge; create a policy without `ingress` key and verify section is hidden; create a policy with mixed FROM sources (podSelector + ipBlock in same rule) and verify both render in one card; create a policy with empty podSelector and verify "All pods in namespace" warning
  - HPAPanel: verify gauge positions current replicas proportionally between min and max; test with Resource, Pods, and Object metric types; verify missing `spec.behavior` doesn't break layout; verify scale target link navigates to correct workload detail page
  - WebhookConfigPanel: verify collapsible sections expand and collapse; verify Ignore failure policy shows yellow warning badge; test with service-based and URL-based client configs; verify `*` in resources list is highlighted; open both a MutatingWebhookConfiguration and ValidatingWebhookConfiguration detail to confirm the same panel works for both

## Acceptance Criteria

- [ ] NetworkPolicyPanel distinguishes empty `ingress: []` (deny all, red badge) from missing `ingress` key (section hidden)
- [ ] NetworkPolicyPanel renders mixed FROM sources (podSelector + namespaceSelector + ipBlock) in same rule card
- [ ] NetworkPolicyPanel shows "All pods in namespace" warning badge for empty podSelector
- [ ] HPAPanel replica gauge positions current replicas proportionally between min and max
- [ ] HPAPanel handles all 4 metric source types (Resource, Pods, Object, External)
- [ ] HPAPanel shows scaling behavior section only when `spec.behavior` is present
- [ ] HPAPanel scale target link navigates to the referenced workload detail page
- [ ] WebhookConfigPanel renders identically for both MutatingWebhookConfiguration and ValidatingWebhookConfiguration
- [ ] WebhookConfigPanel collapsible sections expand and collapse
- [ ] WebhookConfigPanel shows failure policy badges with correct colors (Fail = red, Ignore = yellow)
- [ ] WebhookConfigPanel renders both service and URL client config variants correctly
- [ ] `pnpm check` passes
- [ ] Existing tests unaffected

## Definition of Done

Opening a NetworkPolicy detail view shows a "Rules" tab with structured rule cards showing FROM/TO sources with label badges and port lists, clearly distinguishing deny-all from no-rules. An HPA detail view shows a "Scaling" tab with a replica gauge, metrics table covering all metric types, and optional scaling behavior details. A MutatingWebhookConfiguration or ValidatingWebhookConfiguration detail view shows a "Webhooks" tab with collapsible sections per webhook, each showing failure policy, client config, match rules, and selectors. No placeholder components remain.

## Known Gotchas

- **NetworkPolicy nil vs empty is a rendering decision, not just an enricher concern**: The Phase 1 enricher stores "-" vs "0" for the list column counts, but the panel must also inspect the raw object. Use `obj.spec?.ingress` — if the key is `undefined`, don't show the ingress section. If it's an empty array `[]`, show the "Deny all ingress" badge. Do NOT rely on the enricher's count column for this — read the raw spec in the panel component.

- **HPA scale target URL requires GVR conversion from apiVersion + kind**: `scaleTargetRef` has `apiVersion: "apps/v1"` and `kind: "Deployment"`. Convert to Klados GVR format: split apiVersion into group + version, pluralize kind to resource name (Deployment → deployments, StatefulSet → statefulsets). Build URL as `/c/${ctxName}/apps.v1.deployments/${namespace}/${name}`. Pluralization is lowercase + "s" for standard workload types — handle `Deployment`, `StatefulSet`, `ReplicaSet` explicitly rather than building a general pluralizer.

- **WebhookConfig panel is one component for both resource types**: Register a single `WebhookConfigPanel` under the `webhooks` key. Both `MutatingWebhookConfiguration` and `ValidatingWebhookConfiguration` have an identical `webhooks[]` array structure. No conditional logic based on resource kind is needed inside the component.

- **NetworkPolicy rule `from`/`to` arrays contain mixed source types**: A single rule's `from` array can contain entries with `podSelector`, `namespaceSelector`, `ipBlock`, or combinations thereof. Each entry in the `from` array is a separate source (OR relationship). Within a single entry, `podSelector` + `namespaceSelector` together is an AND relationship. The card must visually distinguish these — show each `from[]` entry as a row, and within a row show AND-combined selectors together.

- **Collapsible sections in WebhookConfigPanel**: Use a simple `$state` boolean per webhook (e.g., `let expanded = $state(webhooks.map(() => true))` to start expanded). Toggle on header click. No animation library needed — a CSS `hidden` class or conditional `{#if}` block is sufficient.
