# Resource Relationships

## Context

Users navigating resources in Klados need to understand ownership chains — e.g., which ReplicaSet controls a Pod, which Deployment controls that ReplicaSet. This feature adds a generic "Controlled By" mechanism driven entirely by `metadata.ownerReferences`, with no per-resource-type special casing.

Two surfaces:
1. A **"Controlled By" column** in every resource list, showing the controller kind (clickable, with name tooltip).
2. A **"Controlled By" field** in the detail drawer's Overview tab.

Clicking the controller opens the detail drawer for that owner resource **in-place** (replacing current drawer content), allowing the user to walk up the ownership chain without leaving the current resource list page.

## Decisions

**Generic ownerReferences-only approach**
No hardcoded relationship logic per resource type. Every resource that has an `ownerReference` with `controller: true` gets the column and detail field automatically. This covers Pods→ReplicaSets→Deployments, Jobs→CronJobs, ReplicaSets→Deployments, etc. without maintenance burden.

**Single controller only**
When multiple `ownerReferences` exist, display only the one with `controller: true`. If none has `controller: true`, display nothing. Keeps the UI clean and covers 99% of real-world cases.

**Drawer replacement, not stacking**
Clicking an owner replaces the current drawer content with the owner's detail view. No breadcrumbs, no back button, no drawer stacking. Simple and avoids z-index/layout complexity. The user can close the drawer and click a different row to reset.

**Frontend-only kind→GVR resolution**
The discovery data (`APIResource[]`) already contains `gvr`, `kind`, and the GVR string encodes the group+version. Build a reverse map `(apiVersion, kind) → gvr` from existing discovery data. No backend changes needed.

**Upward-only traversal**
Only "Controlled By" (child→parent). No "Controls" reverse direction. Reverse would require scanning all watched resources by ownerReference UID, which is expensive and not needed for the primary use case.

## Rejected Alternatives

**Graph visualization**
A visual node-edge graph of resource relationships. Rejected because clickable drill-through in the existing list+drawer UI provides the same navigability with less complexity and better integration with the current UX.

**Selector-based relationships (Service→Pods, etc.)**
Resolving label selectors to find related resources. Rejected for v1 — requires cross-resource queries and introduces complexity around which resources are currently watched. ownerReferences are already on the object and cover the most common navigation patterns.

## Priorities & Tradeoffs

Optimized for **simplicity and generality** — one mechanism that works for every resource type with zero per-type configuration. Sacrificing completeness (selector-based relationships like Service→Pods are not covered) in favor of a clean, maintainable implementation that covers the ownership hierarchy.

## Potential Gotchas

- **CRDs with custom controllers**: ownerReferences from CRD instances may point to kinds not in the descriptor registry. The kind→GVR resolver must handle this gracefully (non-clickable fallback showing just the kind text).
- **Cross-namespace owners**: ownerReferences are namespace-scoped (owner must be in the same namespace) except for cluster-scoped owners of namespaced resources. `GetResource` already handles both — just pass the correct namespace (from the child's namespace for namespaced owners, empty for cluster-scoped).
- **Discovery timing**: The kind→GVR map depends on discovery data being loaded. If discovery hasn't completed for a context, owner links should render as plain text (non-clickable).
- **Owner resource deleted**: The owner referenced may no longer exist (e.g., orphaned resources). `GetResource` will return an error — the drawer should show an appropriate message or simply not open.

## Implementation Details

### Kind→GVR Resolver

Add to the discovery data flow (currently in `Sidebar.svelte`, should be promoted to a shared store or the descriptor registry):

```typescript
// Map key: "apiVersion:kind" e.g. "apps/v1:ReplicaSet"
// Map value: GVR string e.g. "apps.v1.replicasets"
type KindGVRMap = Map<string, string>;

function buildKindGVRMap(resources: APIResource[]): KindGVRMap {
  const map = new Map<string, string>();
  for (const r of resources) {
    // Derive apiVersion from GVR: "apps.v1.replicasets" → "apps/v1"
    // "core.v1.pods" → "v1" (core group = empty group in apiVersion)
    const apiVersion = gvrToApiVersion(r.gvr);
    map.set(`${apiVersion}:${r.kind}`, r.gvr);
  }
  return map;
}

function gvrToApiVersion(gvr: string): string {
  // ParseGVR splits from right: group.version.resource
  // "apps.v1.replicasets" → group="apps", version="v1"
  // "core.v1.pods" → group="core" (maps to ""), version="v1"
  const { group, version } = parseGVR(gvr);
  if (group === 'core' || group === '') return version;
  return `${group}/${version}`;
}

function resolveGVR(apiVersion: string, kind: string): string | undefined {
  return kindGVRMap.get(`${apiVersion}:${kind}`);
}
```

This map should be built/rebuilt whenever discovery data is received (same event: `discovery:{ctx}:resources`). Store it in `clusterStore` or `DescriptorRegistry` — wherever the discovery data is already consumed.

### Extracting the Controller Reference

Utility function used by both the column and the detail field:

```typescript
interface ControllerRef {
  apiVersion: string;   // e.g. "apps/v1"
  kind: string;         // e.g. "ReplicaSet"
  name: string;         // e.g. "my-rs-abc123"
  uid: string;
}

function getControllerRef(obj: any): ControllerRef | null {
  const refs = obj?.metadata?.ownerReferences;
  if (!Array.isArray(refs)) return null;
  const controller = refs.find((r: any) => r.controller === true);
  if (!controller) return null;
  return {
    apiVersion: controller.apiVersion,
    kind: controller.kind,
    name: controller.name,
    uid: controller.uid,
  };
}
```

### "Controlled By" Column

Add a built-in column to the fallback descriptor (and optionally to all descriptors) in `DescriptorRegistry`:

```typescript
{
  name: 'Controlled By',
  expr: 'metadata.ownerReferences',
  renderType: 'controlledBy',   // new render type
  visible: true,
  priority: 90,                 // after name, namespace, before age
}
```

The `controlledBy` renderer in `ResourceList.svelte`:

```svelte
<!-- Renders the controller kind as a clickable link with tooltip -->
{#if controllerRef}
  <button
    class="text-accent hover:underline cursor-pointer"
    title="{controllerRef.kind}/{controllerRef.name}"
    onclick={() => openOwnerDrawer(controllerRef)}
  >
    {controllerRef.kind}
  </button>
{/if}
```

### "Controlled By" in Detail Overview

Add a field to the Overview section of the detail drawer/tab. Same data extraction, same click behavior:

```svelte
{#if controllerRef}
  <dt>Controlled By</dt>
  <dd>
    <button
      class="text-accent hover:underline cursor-pointer"
      title="{controllerRef.kind}/{controllerRef.name}"
      onclick={() => openOwnerDrawer(controllerRef)}
    >
      {controllerRef.kind}/{controllerRef.name}
    </button>
  </dd>
{/if}
```

In the detail view, show `Kind/Name` (not just Kind) since there's more space and the user is already looking at a specific resource.

### Drawer Navigation

When the user clicks a controller link (from either column or detail view):

```typescript
async function openOwnerDrawer(ref: ControllerRef) {
  const gvr = resolveGVR(ref.apiVersion, ref.kind);
  if (!gvr) return; // unknown kind, no-op

  const namespace = currentItem.metadata?.namespace ?? '';

  try {
    const owner = await ResourceService.GetResource(ctxName, gvr, namespace, ref.name);
    // Replace drawer content: update selectedItem and selectedGVR
    selectedItem = owner;
    selectedGVR = gvr;
  } catch (e) {
    // Owner doesn't exist (deleted/orphaned) — show notification or no-op
    notificationStore.add('Owner resource not found', 'warning');
  }
}
```

The `DetailDrawer` component needs to accept a reactive `gvr` prop (it may currently be fixed to the page's GVR). When `selectedGVR` changes, the drawer re-renders its tabs/panels for the new resource type.

### Data Flow

```
User clicks "Controlled By" link (column or detail overview)
  → getControllerRef(item) extracts apiVersion, kind, name
  → resolveGVR(apiVersion, kind) looks up GVR from discovery map
  → ResourceService.GetResource(ctx, gvr, namespace, name) fetches owner
  → selectedItem = owner, selectedGVR = gvr
  → DetailDrawer re-renders with new resource type and data
  → User sees owner's detail, can click its owner to continue up the chain
```

### Files to Create/Modify

| File | Change |
|------|--------|
| `frontend/src/lib/utils/relationships.ts` | New: `getControllerRef()`, `resolveGVR()`, `buildKindGVRMap()`, `gvrToApiVersion()` |
| `frontend/src/lib/stores/cluster.svelte.ts` | Store `kindGVRMap`, rebuild on discovery event |
| `frontend/src/lib/registry/index.ts` | Add "Controlled By" column to fallback descriptor |
| `frontend/src/lib/components/ResourceList.svelte` | Add `controlledBy` render type |
| `frontend/src/routes/ResourceListPage.svelte` | Support `selectedGVR` state alongside `selectedItem`; pass to drawer; implement `openOwnerDrawer` |
| Detail overview component (wherever the Overview tab fields are rendered) | Add "Controlled By" field with click handler |

## Definition of Done

- Every resource with a `controller: true` ownerReference shows the controller's kind in the "Controlled By" column.
- Hovering the column value shows a tooltip with `Kind/Name`.
- Clicking the column value opens the detail drawer for the owner resource without navigating away from the current page.
- The detail drawer's Overview tab shows a "Controlled By" field with `Kind/Name`, also clickable.
- Clicking through the chain works (Pod → ReplicaSet → Deployment) by replacing drawer content each time.
- Resources without ownerReferences show nothing in the column (no "N/A" or placeholder).
- Unknown kinds (no GVR in discovery) render as plain text, not clickable.
- Deleted owners show a notification rather than crashing.
