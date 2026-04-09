# Resource Relationships — Controlled By

Add a generic "Controlled By" mechanism driven by `metadata.ownerReferences` so users can click through ownership chains (Pod → ReplicaSet → Deployment) without leaving the current resource list page.

## First Action

Read `frontend/src/routes/ResourceListPage.svelte` lines 60–190 to understand how `selectedItem` drives the DetailDrawer — your click handler will replace `selectedItem` (and introduce a `selectedGVR`) to swap drawer content in-place.

## Context

Klados has a working resource list with detail drawers, a descriptor registry with column render types, and discovery data that includes `(gvr, kind, namespaced)` per API resource. What's missing is any awareness of resource ownership. This phase adds a "Controlled By" column and detail field that lets users navigate the owner chain by replacing drawer content on click.

## Files to Read

- `frontend/src/routes/ResourceListPage.svelte` — **what to look for**: how `selectedItem` state controls the DetailDrawer (lines 62–65, 165–187); the drawer currently receives the page-level `gvr` — you'll need to make this dynamic via a `selectedGVR` state variable
- `frontend/src/lib/components/ResourceList.svelte` — **what to look for**: `renderCell()` and `renderValue()` (lines 153–161, 347–357) — this is where you add the `controlledBy` render type branch; also the `onselect` callback pattern (line 329)
- `frontend/src/lib/registry/index.ts` — **what to look for**: the fallback descriptor (lines 188–210) where you'll inject the "Controlled By" column; the `DescriptorDef` type (line 36–47) including `kind` field; the `RenderType` union where `controlledBy` must be added
- `frontend/src/lib/components/Sidebar.svelte` — **what to look for**: `APIResource` interface `{gvr, kind, namespaced}` (lines 19–23) and `handleDiscovery()` (lines 182–192) — this is the data source for building the kind→GVR map
- `frontend/src/lib/stores/cluster.svelte.ts` — **what to look for**: where to add the `kindGVRMap` and its rebuild trigger; currently has no discovery data storage (kind mapping lives only in Sidebar)
- `frontend/src/lib/components/panels/OverviewPanel.svelte` — **what to look for**: how `overviewFields` render in the 3-column grid (lines 121–132) and `renderValue()` (lines 39–46) — the "Controlled By" detail field plugs in here
- `frontend/src/lib/registry/gvr.ts` (or equivalent) — **what to look for**: existing `parseGVR()` utility that splits `apps.v1.replicasets` into group/version/resource — you'll need the inverse (`gvrToApiVersion`) for the kind→GVR map

## Source Documents

- `RELATIONSHIP_SPEC.md` — full design spec; covers all decisions, rejected alternatives, implementation details including type signatures and data flow
- `CLAUDE.md` — architecture overview, GVR format, Wails event conventions, Svelte 5 patterns, rendering pipeline

## What Exists

- Resource list with column rendering pipeline (`text`, `badge`, `age`, `progress` render types)
- Detail drawer that accepts `item`, `ctxName`, `gvr` props and renders tabs (Overview, YAML, etc.)
- OverviewPanel rendering `descriptor.overviewFields` with CEL expression evaluation
- Descriptor registry with fallback descriptor for unknown GVRs
- Discovery data flow: backend emits `discovery:{ctx}:resources` with `APIResource[]`, consumed in Sidebar
- `ResourceService.GetResource(ctx, gvr, namespace, name)` RPC for fetching individual resources
- `parseGVR()` utility for splitting dot-separated GVR strings

## Deliverables

1. **`frontend/src/lib/utils/relationships.ts`** — new file containing:
   - `getControllerRef(obj)` — extracts the `controller: true` ownerReference, returns `{apiVersion, kind, name, uid}` or `null`
   - `gvrToApiVersion(gvr)` — converts `apps.v1.replicasets` to `apps/v1`, handles `core` → empty group
   - `buildKindGVRMap(resources: APIResource[])` — builds `Map<"apiVersion:kind", gvr>` from discovery data
   - `resolveGVR(map, apiVersion, kind)` — lookup wrapper returning `string | undefined`

2. **Kind→GVR map in `cluster.svelte.ts`** — store the map, rebuild on discovery event; export a `resolveGVR(apiVersion, kind)` method on `clusterStore`

3. **`controlledBy` render type** — add to `RenderType` union in `registry/index.ts`; implement in `ResourceList.svelte`'s `renderValue()` as a clickable button showing the kind, with `title` attribute showing `Kind/Name`

4. **"Controlled By" column on fallback descriptor** — inject into the fallback descriptor's columns array (priority ~90, after Namespace, before Age); also add to all built-in descriptors or apply universally via the registry

5. **"Controlled By" field in OverviewPanel** — add a field to the Overview tab showing `Kind/Name` as a clickable link; clicking calls the same `openOwnerDrawer` handler

6. **Drawer replacement on click** — in `ResourceListPage.svelte`, introduce `selectedGVR` state alongside `selectedItem`; `openOwnerDrawer(ref)` calls `resolveGVR` → `GetResource` → sets both `selectedItem` and `selectedGVR`; DetailDrawer re-renders for the new resource type

## Tests

### Unit

- `getControllerRef()` returns the `controller: true` ref, ignoring non-controller refs
- `getControllerRef()` returns `null` when no ownerReferences exist
- `getControllerRef()` returns `null` when ownerReferences exist but none has `controller: true`
- `gvrToApiVersion("apps.v1.replicasets")` → `"apps/v1"`
- `gvrToApiVersion("core.v1.pods")` → `"v1"` (core maps to empty group)
- `gvrToApiVersion("networking.k8s.io.v1.ingresses")` → `"networking.k8s.io/v1"` (dotted group)
- `buildKindGVRMap` builds correct map from APIResource array
- `resolveGVR` returns GVR for known kinds, `undefined` for unknown

### Manual Verification

- Open Pods list → "Controlled By" column shows `ReplicaSet` for managed pods, empty for standalone pods
- Hover the column value → tooltip shows `ReplicaSet/my-rs-abc123`
- Click `ReplicaSet` in column → drawer opens showing the ReplicaSet detail
- In ReplicaSet drawer, "Controlled By" in Overview shows `Deployment/my-deploy` → click opens Deployment drawer
- Close drawer → still on the Pods resource list page (no navigation occurred)
- Pod with no ownerReferences → column is empty, no "Controlled By" in Overview
- Delete the owner resource → click shows a notification, drawer doesn't open/crash

## Acceptance Criteria

- [ ] Every resource with a `controller: true` ownerReference shows the controller kind in the "Controlled By" column
- [ ] Hovering the column value shows a tooltip with `Kind/Name`
- [ ] Clicking the column value opens the detail drawer for the owner resource without page navigation
- [ ] The detail drawer Overview tab shows a "Controlled By" field with `Kind/Name`, clickable
- [ ] Clicking through the chain works across multiple levels (Pod → RS → Deployment) by replacing drawer content
- [ ] Resources without ownerReferences show nothing in the column (no placeholder text)
- [ ] Unknown kinds (not in discovery) render as plain non-clickable text
- [ ] All unit tests pass for the relationships utility functions
- [ ] Kind→GVR map rebuilds correctly when switching cluster contexts

## Definition of Done

A user viewing any resource list can see which controller owns each resource, click through the ownership chain in the detail drawer (each click replacing the drawer content with the parent resource), and walk all the way up to the top-level controller — all without leaving the current page. The kind→GVR resolver handles core group mapping and dotted groups correctly, and gracefully degrades to plain text for unknown or unresolvable kinds.

## Known Gotchas

- **The trap**: `gvrToApiVersion` must handle groups with dots (e.g. `networking.k8s.io.v1.ingresses`). **Why**: `parseGVR` splits from the right — group can contain dots. **What to do**: Reuse the existing `parseGVR()` logic, don't write a naive split-on-dot.

- **The trap**: The DetailDrawer currently receives the page-level `gvr` which determines which descriptor (and thus which panels/tabs) to render. If you only update `selectedItem` without also updating the GVR, the drawer will render the owner with the wrong descriptor. **Why**: A Pod's descriptor has different panels than a ReplicaSet's. **What to do**: Introduce `selectedGVR` state and pass it to the drawer; update both atomically when navigating to an owner.

- **The trap**: `ownerReferences` don't specify whether the owner is namespaced or cluster-scoped. **Why**: Cross-namespace ownership doesn't exist in Kubernetes, but cluster-scoped resources can own namespaced ones. **What to do**: Use the child's namespace for the `GetResource` call — if the owner is cluster-scoped, the namespace parameter is ignored by the API.

- **The trap**: Discovery data may not be loaded yet when the first resource list renders (race on initial connect). **Why**: Discovery is async after cluster connect. **What to do**: Check if `kindGVRMap` is populated before making links clickable; render as plain text until discovery completes.

- **The trap**: Adding the "Controlled By" column only to the fallback descriptor means explicitly registered descriptors (Pods, Deployments, etc.) won't have it. **Why**: Built-in descriptors define their own column lists, overriding the fallback. **What to do**: Inject the column universally in the registry — either by appending it to every descriptor's columns during registration, or by handling it as a special always-present column in the rendering pipeline.
