# Phase 2 — Simple Custom Panels

Implement 4 custom detail panels that render structured Kubernetes data as tables, usage bars, and status badges: ResourceQuota, LimitRange, PDB, and EndpointSlice.

## First Action

Read `frontend/src/lib/components/panels/DeploymentPanel.svelte` to see the established panel pattern — how props are received, how `SectionHeader` is used for layout, how conditions are rendered as badges, and how grid layouts structure the detail view. All 4 panels in this phase follow this same structure.

## Context

Phase 1 added descriptors, enrichers, and sidebar entries for 12 new resource types. All types are now browsable in list view with correct columns, and their detail views show generic panels (overview, labels, events, yaml) plus placeholder tabs for custom panels. This phase replaces 4 of those placeholders with real panel components that render structured resource data in a more useful format than raw YAML.

## Files to Read

- `frontend/src/lib/components/panels/DeploymentPanel.svelte` — **what to look for**: props interface (`{ obj: Record<string, any> }`), how sections are structured with `SectionHeader`, how conditions are rendered as status badges, grid layout patterns
- `frontend/src/lib/components/panels/ServicePanel.svelte` — **what to look for**: how `ctxName` prop is used alongside `obj` for panels that need cluster context (EndpointSlice panel needs this for target ref navigation)
- `frontend/src/lib/components/ResourceDetail.svelte` — **what to look for**: the conditional render block (lines ~220-290) showing how panel props are routed — some panels get only `obj`, others get `ctxName`, `namespace`, `name` etc.
- `frontend/src/routes/routes.ts` — **what to look for**: URL structure `/c/:ctx/:gvr/:ns/:name` for constructing navigation links from EndpointSlice target refs

## Source Documents

- `RESOURCE_TYPES_SPEC.md` — panel layout specifications, props interfaces, and rendering details for ResourceQuotaPanel, LimitRangePanel, PDBPanel, and EndpointSlicePanel
- `RESOURCE_TYPES_PHASES.md` — Phase 2 section with deliverables, acceptance criteria, and handoff notes (especially the Kubernetes quantity parsing note and PDB percentage values)

## What Exists

- All 12 resource type descriptors registered in `builtin.go`
- All enrichers implemented and passing tests
- Sidebar entries for all 12 types under correct categories
- Placeholder panel components registered in `ResourceDetail.svelte` for `resourcequota`, `limitrange`, `pdb`, `endpointslice` panel keys
- Panel labels already set: "Usage", "Limits", "Budget", "Addresses"
- Generic detail panels (overview, labels, events, yaml) working for all types

## Deliverables

1. **`ResourceQuotaPanel.svelte`** — Scopes section with badges when `spec.scopeSelector` or `spec.scopes` present. Usage table with one row per resource in `status.hard`: resource name, used value (from `status.used`), hard value, percentage bar. Bar colors: green (<70%), yellow (70-90%), red (>90%). Missing `status.used` value → "0" with gray bar.

2. **`LimitRangePanel.svelte`** — One section per `spec.limits[]` entry. Section header with Type badge (Container / Pod / PersistentVolumeClaim). Table columns: Resource, Default, Default Request, Min, Max, Max Limit/Request Ratio. Rows for cpu, memory, storage — only those present in the entry. Empty cells for inapplicable combinations (PVC has no cpu/memory rows).

3. **`PDBPanel.svelte`** — Selector section with label badges from `spec.selector.matchLabels`. Budget config showing "Min Available: X" or "Max Unavailable: X" (only one is set). Status bar with `currentHealthy / expectedPods` proportional fill — green if `disruptionsAllowed > 0`, red if 0. Status fields grid: expectedPods, currentHealthy, desiredHealthy, disruptionsAllowed. Conditions section with status badges and messages when `status.conditions` present.

4. **`EndpointSlicePanel.svelte`** — Address type indicator badge (IPv4/IPv6/FQDN). Ports summary row (name, port, protocol). Addresses table: Address, Node Name, Ready, Serving, Terminating, Target Ref. Ready/Serving/Terminating as green/yellow/red condition badges. Target Ref as clickable link navigating to pod detail when `targetRef.kind == "Pod"`. Requires `ctxName` prop for navigation.

5. **Replace placeholder components** in `ResourceDetail.svelte` panel map with real implementations. Update props routing in conditional render block.

## Tests

- **Frontend type-check (`pnpm check`)**
  - All 4 new panel components type-check cleanly with their props interfaces

- **Manual verification**
  - ResourceQuotaPanel: create a quota with >70% and >90% usage to verify bar color transitions; verify missing `status.used` shows gray bar with "0"
  - LimitRangePanel: create a limit range with Container and PVC types to verify sparse cells (PVC shows only storage rows)
  - PDBPanel: verify status bar turns red when `disruptionsAllowed` is 0; verify "Min Available" vs "Max Unavailable" renders correctly based on which field is set
  - EndpointSlicePanel: verify target ref links navigate to correct pod detail page; verify Ready/Serving/Terminating badges show correct colors

## Acceptance Criteria

- [ ] ResourceQuotaPanel renders usage bars with correct colors at green/yellow/red thresholds
- [ ] ResourceQuotaPanel displays scopes section when present, omits when absent
- [ ] ResourceQuotaPanel handles missing `status.used` gracefully (shows "0" with gray bar)
- [ ] LimitRangePanel renders matrix table with per-type sections
- [ ] LimitRangePanel handles sparse cells (PVC without cpu/memory rows)
- [ ] PDBPanel status bar fill is proportional to `currentHealthy / expectedPods`
- [ ] PDBPanel shows conditions with status badges when present
- [ ] EndpointSlicePanel renders address table with Ready/Serving/Terminating badges
- [ ] EndpointSlicePanel target ref links navigate to pod detail view
- [ ] `pnpm check` passes
- [ ] Existing tests unaffected

## Definition of Done

Opening a ResourceQuota detail view shows a "Usage" tab with colored percentage bars for each tracked resource. A LimitRange shows a "Limits" tab with a structured matrix table grouped by type. A PDB shows a "Budget" tab with a visual health bar and disruption status. An EndpointSlice shows an "Addresses" tab with a table of endpoints where pod target refs are clickable links. No placeholder components remain for these 4 panel types.

## Known Gotchas

- **Kubernetes quantity strings need parsing for percentage bars**: `status.hard` and `status.used` values are quantity strings like "10", "1Gi", "500m". To compute the percentage for ResourceQuota bars, both must be parsed to a common numeric value. Handle at minimum: plain integers, `m` suffix (millicores/milli-units, divide by 1000), `Ki`/`Mi`/`Gi` suffixes (binary multiples). Don't try to compare incompatible units — if parsing fails, skip the percentage bar and just show raw values.

- **PDB `minAvailable` and `maxUnavailable` can be integers or percentage strings**: A value like `"50%"` is valid. Display the raw value as-is — don't try to resolve percentage to absolute number (that requires knowing the replica count of the target workload, which is out of scope).

- **EndpointSlice target ref navigation URL construction**: Build as `/c/${ctxName}/core.v1.pods/${targetRef.namespace ?? endpoint's namespace}/${targetRef.name}`. The `targetRef` may not include a namespace — fall back to the EndpointSlice's own namespace from `obj.metadata.namespace`.

- **EndpointSlice conditions are per-endpoint, not per-slice**: Each entry in the `endpoints[]` array has its own `conditions` object with `ready`, `serving`, `terminating` booleans. These are not on the slice itself. Iterate `endpoints[]` and read `conditions` from each.
