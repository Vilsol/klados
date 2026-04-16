# Generic GVR Capabilities Design

**Date:** 2026-04-16
**Status:** Design approved, pending implementation planning

## Problem

Klados handles unknown GVRs (CRDs and custom resources) via a minimal fallback descriptor with Name, Namespace, and Age columns. This makes CRDs feel like second-class citizens — users must write a plugin to get a first-class experience. Built-in resources also lack some generic capabilities that would universally apply (drift detection, structured metadata view, conditions health indicators).

## Goal

Extend Klados's generic GVR support so CRDs get a useful out-of-the-box experience without requiring a plugin, and bring universally-applicable capabilities to all resources (built-in and custom).

## Scope

In scope:
- Discovery of CRD `additionalPrinterColumns` and subresources
- Universal Events tab for every resource type
- Conditions display, health badges, validation warnings
- Labels/Annotations panel, Finalizers, Last-applied drift diff
- Owner chain and related-resources lookups
- Interactive scaling generalized to any resource with a scale subresource

Out of scope:
- Plugin system changes (plugins continue to take priority over auto-generated descriptors)
- New resource-editing capabilities beyond what already exists
- Cross-cluster features
- Resource creation forms (YAML editing remains the creation path)

## Approach: Hybrid Discovery + Runtime Inference

Two complementary layers:

1. **Discovery layer (authoritative):** What the API server knows — printer columns, subresources, scale paths. Fetched once at connect time.
2. **Runtime layer (adaptive):** What varies per object — conditions presence, owner refs, finalizers, last-applied annotation. Inspected from fetched objects.

This mirrors how `kubectl describe` operates: CRD metadata for columns, object inspection for conditions/events.

## Architecture

### Backend (`internal/`)

| Package | Changes |
|---|---|
| `cluster/` | `DiscoverResources()` extended to collect printer columns (via CRD list), subresources (from API resource list), scale subresource path (from CRD spec). Discovery payload gains new fields. |
| `resource/engine.go` | New `Scale(ctx, gvr, ns, name, replicas)` method using dynamic client `Resource(gvr).Namespace(ns).UpdateScale()` — respects CRD scale subresource paths. Falls back to `spec.replicas` MergePatch only when scale subresource is absent (preserves today's Deployment/StatefulSet behavior without server-side scale). |
| `resource/descriptor.go` | Descriptor struct unchanged structurally. No breaking changes. |
| `services/resource.go` | `ScaleResource` delegates to `engine.Scale`. Existing signature preserved. |

### Frontend (`frontend/src/`)

| Area | Changes |
|---|---|
| `lib/registry/index.ts` | Fallback logic replaced by a descriptor generator that consumes discovery metadata. Priority: built-in → plugin → auto-generated → static fallback. |
| `lib/registry/loaded.svelte.ts` | Existing reactive gate already works — generator runs after discovery event arrives. |
| `lib/components/panels/` | New: `ConditionsPanel`, `RelatedResourcesPanel`, `MetadataPanel`, `DriftPanel`. Existing `EventsPanel` (currently tied to Pod/Deployment) generalized into a universal panel. |
| `lib/components/` | New `HealthBadge` for list page, `ValidationWarningBanner` for detail page. |

## Feature Details

### 1. Enhanced Discovery

`DiscoverResources()` emits a payload that includes per-resource:
- `additionalPrinterColumns[]` — name, type, jsonPath, description, priority (only populated for CRD-backed resources)
- `subresources[]` — set of supported subresource names (`scale`, `status`, etc.)
- `scaleSpecReplicasPath` — from CRD spec scale subresource (defaults to `.spec.replicas`)
- `scaleStatusReplicasPath` — from CRD spec scale subresource (defaults to `.status.replicas`)

CRD fetches are batched: one `List` call for all CRDs, not per-resource `Get`. Subresource detection uses the existing API resource list (entries like `deployments/scale` become subresources of `deployments`).

### 2. Dynamic Descriptor Generation

When discovery metadata arrives, for each GVR without a built-in or plugin descriptor:

**Columns:**
- Standard columns (Name, Namespace, Age) are always prepended
- Each `additionalPrinterColumn` maps to a Descriptor column:
  - `jsonPath` → CEL expression (e.g. `.spec.replicas` → `spec.replicas`)
  - `type` → render type: `string`/`integer`→`text`, `date`→`age`, `boolean`→`badge`
  - `priority > 0` → hidden by default (matches `kubectl` semantics)
- Duplicate columns (same name as built-in) are merged, not duplicated

**Subresource-driven additions:**
- `scale` subresource → "Scale" action + replicas column (if not already provided by printer columns)
- `status` subresource → "Status" detail panel that renders `status` separately from `spec`

**Universal panels (on every auto-generated descriptor):**
- Overview, YAML, Events, Conditions, Related, Metadata, Drift (conditional)

**Universal actions:**
- Delete, Edit YAML, Scale (conditional on scale subresource)

**Priority order:** Built-in → Plugin → Auto-generated → Static fallback. Built-in descriptors are never overridden.

### 3. Runtime-Inferred Features

All apply to every GVR (built-in and custom) via object inspection.

**3a. Conditions**
- Detail page: `ConditionsPanel` rendered when `status.conditions[]` is present. Shows Type, Status badge (True green / False red / Unknown yellow), Reason, Message, Last Transition Time.
- List page: `HealthBadge` column synthesized from conditions. Green when positive types (`Ready`, `Available`) are True; red when negative types (`Degraded`, `MemoryPressure`, etc.) are True or positive types are False; yellow during `Progressing`. Falls back to True/False ratio badge (e.g. `3/4`) when no recognized types present.
- Detection is structural: array of objects with `type` and `status` string fields qualifies.

**3b. Owner References**
- Overview panel: owner chain walked up to 5 levels, shown as breadcrumb (`ReplicaSet/my-rs → Deployment/my-deploy`). Each entry links to the owner's detail page.
- "Related" tab: reverse lookup against cached/watched resources — finds objects whose `ownerReferences[].uid` matches current object's UID. Grouped by GVR. Limited to active watches (no cluster-wide scan). When empty: "No related resources found in active watches".

**3c. Finalizers**
- Overview panel: list of badges below standard overview fields. Hidden when empty.
- List page: hidden-by-default "Finalizers" column (priority > 0) showing count.

**3d. Metadata Panel (Labels & Annotations)**
- New detail tab. Labels: two-column key/value table with copiable badges. Annotations: two-column table; values > 120 chars collapsible. The `kubectl.kubernetes.io/last-applied-configuration` annotation is excluded (has its own Drift tab).

**3e. Drift Tab (Last-Applied Diff)**
- Tab shown only when `kubectl.kubernetes.io/last-applied-configuration` annotation exists.
- Parses annotation as JSON, converts both it and the current object to YAML, renders via existing `DiffView`.
- Strips server-managed fields before diffing: `managedFields`, `resourceVersion`, `uid`, `creationTimestamp`, `generation`, `selfLink`, `status`.

**3f. Validation Warnings**
- Detail page: `ValidationWarningBanner` at top when conditions indicate problems:
  - `status: "False"` on positive types (`Ready`, `Available`, `Initialized`)
  - `status: "True"` on negative types (`Degraded`, `MemoryPressure`, `DiskPressure`, `PIDPressure`, `NetworkUnavailable`)
- Shows condition Reason and Message. Multiple warnings stack.
- List page: subsumed into `HealthBadge` — no separate column.

### 4. Universal Events Tab

- Every GVR's auto-generated descriptor includes an Events tab. Built-in descriptors that lack one receive it as well.
- On open: `ResourceService.List` for `core.v1.events` in the resource's namespace (all namespaces for cluster-scoped resources). Client-side filter by `involvedObject.uid` + `kind` + `name`.
- Columns: Type, Reason, Object, Message, Count, Age. Warning events get an amber severity indicator.
- Watch subscription active while tab is open; unsubscribes on tab close/navigation (existing watch lifecycle with 30s grace).
- Existing Pod/Deployment events tabs migrate to the shared implementation.

### 5. Interactive Scaling

**Existing state (preserved):**
- `ScaleResource` RPC, `ActionsToolbar.svelte` Scale dialog (single resource), `BulkScaleDialog.svelte` (bulk) — all reused.

**Changes:**
- Backend: `ScaleResource` switches from direct `spec.replicas` MergePatch to `UpdateScale()` via the dynamic client. This respects each CRD's declared scale subresource path. MergePatch remains only for resources without a scale subresource (rare edge case for legacy behavior).
- Frontend: auto-generated descriptors for GVRs with scale subresource include the `"scale"` action — making the existing dialogs available for CRDs without code changes.
- Replicas column auto-added for scalable resources when not provided by printer columns, with CEL expression from the scale subresource `specReplicasPath`.
- RBAC handling: attempt the scale and surface 403 as a notification. No pre-flight SelfSubjectAccessReview (avoids latency cost).

## Data Flow

```
K8s API ──→ cluster.Manager.DiscoverResources()
              │  fetches: API resource list, CRD list
              ▼
           discovery payload (+ printer columns, subresources, scale paths)
              │
              ▼
      Events("discovery:{ctx}:resources")
              │
              ▼
      DescriptorRegistry (frontend)
              │  for each GVR:
              │    built-in? → use built-in
              │    plugin?   → use plugin
              │    else      → generate from discovery metadata
              ▼
      Descriptor consumed by ResourceList / ResourceDetailPage
              │
              ▼
      Per-object rendering + runtime inference:
        - HealthBadge reads status.conditions
        - ValidationWarningBanner scans conditions
        - RelatedResourcesPanel queries ResourceStore cache
        - DriftPanel reads last-applied annotation
```

## Build Phases

Each phase is independently shippable.

1. **Discovery metadata** (backend only). Tests: unit tests against fake CRDs confirming printer columns and subresources are extracted.
2. **Dynamic descriptor generation** (frontend). Test: verify CRDs render with their printer columns automatically.
3. **Universal Events tab**. Refactor existing Pod/Deployment Events panel into the shared implementation; wire into auto-generated descriptors.
4. **Conditions + validation warnings**. `ConditionsPanel`, `HealthBadge`, `ValidationWarningBanner`.
5. **Metadata panel + Finalizers + Drift tab**. Three related per-object features shipped together.
6. **Owner chain + Related resources tab**. Owner walking, reverse-lookup rendering.
7. **Interactive scaling for CRDs**. Backend switch to `UpdateScale`, replicas column for scalable CRDs.

## Testing Strategy

- Go: unit tests for discovery metadata extraction, `Scale` method with fake dynamic client. Existing `testza` patterns.
- Frontend: Vitest for descriptor generation (given mock discovery payload, assert descriptor shape), for runtime inference components (HealthBadge, ConditionsPanel snapshot tests).
- Integration: spin up a CRD in a test cluster, confirm columns auto-populate.

## Open Questions / Trade-offs

- **Related Resources scope:** Limited to currently-watched resources. An alternative of targeted child-type queries (Pods, Events, ReplicaSets) was deferred to a follow-up if users request it.
- **Drift tab applicability:** The `last-applied-configuration` annotation is only set by `kubectl apply`. Resources managed by Helm, operators, or `kubectl create` won't have it, and the tab is simply absent. No attempt is made to reconstruct drift by other means.
- **RBAC pre-check on Scale:** Attempting the operation and handling 403 is chosen over pre-flight SAR checks to avoid per-list API load.
