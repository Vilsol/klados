# Generic GVR Phase 6 — Owner Chain & Related Resources Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Two related features that use `ownerReferences` and object UIDs:
1. **Owner chain** — walk `metadata.ownerReferences` upward (up to 5 levels) and render a breadcrumb in the overview.
2. **Related Resources tab** — reverse lookup across currently-cached/watched resources; show objects whose `ownerReferences[].uid` equals the current object's UID.

**Architecture:** Owner chain fetches each owner via `GetResource` (existing Wails binding). Related lookup reads from a new `ResourceCache` — a thin global cache hydrated by every active `ResourceStore` watch. No new backend.

**Tech Stack:** Svelte 5 runes, TypeScript, Wails bindings, vitest.

**Depends on:** Phases 1-2 (auto-generated descriptors include `"related"` panel).

**Reference spec:** `docs/superpowers/specs/2026-04-16-generic-gvr-capabilities-design.md` §3b, §6.

---

## File Structure

- Create: `frontend/src/lib/stores/resourceCache.svelte.ts` — global cache mapping `(ctx, gvr) → Map<uid, object>`
- Modify: `frontend/src/lib/stores/resource.svelte.ts` — populate the cache on ADD/MODIFY, evict on DELETE
- Create: `frontend/src/lib/kubernetes/owners.ts` — `getOwnerReferences`, `findRelated` helpers
- Create: `frontend/src/lib/kubernetes/__tests__/owners.test.ts`
- Create: `frontend/src/lib/components/panels/OwnerChain.svelte` — renders breadcrumb inside overview panel
- Create: `frontend/src/lib/components/panels/RelatedResourcesPanel.svelte`
- Modify: `frontend/src/lib/components/panels/OverviewPanel.svelte` — embed `OwnerChain`
- Modify: `frontend/src/lib/components/ResourceDetail.svelte` — register `RelatedResourcesPanel`

---

## Task 1: Owner helpers + resourceCache infrastructure

**Files:**
- Create: `frontend/src/lib/kubernetes/owners.ts`
- Create: `frontend/src/lib/kubernetes/__tests__/owners.test.ts`
- Create: `frontend/src/lib/stores/resourceCache.svelte.ts`
- Create: `frontend/src/lib/stores/__tests__/resourceCache.test.ts`
- Modify: `frontend/src/lib/stores/resource.svelte.ts`

### Owner helpers

- [ ] **Step 1: Write failing tests**

```typescript
import { describe, it, expect } from "vitest";
import { getOwnerReferences, gvrFromAPIVersion } from "../owners";

describe("getOwnerReferences", () => {
  it("returns [] when missing", () => {
    expect(getOwnerReferences({})).toEqual([]);
    expect(getOwnerReferences({ metadata: {} })).toEqual([]);
  });

  it("returns owner references list", () => {
    const obj = {
      metadata: {
        ownerReferences: [
          { apiVersion: "apps/v1", kind: "ReplicaSet", name: "rs-1", uid: "uid-1", controller: true },
        ],
      },
    };
    const o = getOwnerReferences(obj);
    expect(o.length).toBe(1);
    expect(o[0].kind).toBe("ReplicaSet");
  });
});

describe("gvrFromAPIVersion", () => {
  it("handles core group", () => {
    expect(gvrFromAPIVersion("v1", "Pod")).toBe("core.v1.pods");
  });
  it("handles named group", () => {
    expect(gvrFromAPIVersion("apps/v1", "Deployment")).toBe("apps.v1.deployments");
  });
  it("lowercases and plural-izes simple kinds", () => {
    // naive pluralization for common patterns
    expect(gvrFromAPIVersion("v1", "Service")).toBe("core.v1.services");
    expect(gvrFromAPIVersion("v1", "ConfigMap")).toBe("core.v1.configmaps");
    expect(gvrFromAPIVersion("policy/v1", "PodDisruptionBudget")).toBe("policy.v1.poddisruptionbudgets");
  });
});
```

Save as `frontend/src/lib/kubernetes/__tests__/owners.test.ts`.

- [ ] **Step 2: Run to verify failure**

Run: `cd frontend && npx vitest run src/lib/kubernetes/__tests__/owners.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement owners.ts**

Create `frontend/src/lib/kubernetes/owners.ts`:

```typescript
export interface OwnerReference {
  apiVersion: string;
  kind: string;
  name: string;
  uid: string;
  controller?: boolean;
  blockOwnerDeletion?: boolean;
}

export function getOwnerReferences(obj: unknown): OwnerReference[] {
  if (!obj || typeof obj !== "object") return [];
  const m = (obj as any).metadata;
  const refs = m?.ownerReferences;
  if (!Array.isArray(refs)) return [];
  return refs.filter((r) => r && r.kind && r.name && r.uid && r.apiVersion);
}

/**
 * Convert an ownerReference's (apiVersion, kind) to our dot-separated GVR
 * string. This is a best-effort naive pluralization; for GVRs whose plural
 * doesn't follow the standard rules (e.g. "endpoints" not "endpointses"),
 * the caller should fall back to discovery metadata lookup when possible.
 */
export function gvrFromAPIVersion(apiVersion: string, kind: string): string {
  const slash = apiVersion.indexOf("/");
  const group = slash >= 0 ? apiVersion.slice(0, slash) : "core";
  const version = slash >= 0 ? apiVersion.slice(slash + 1) : apiVersion;
  return `${group}.${version}.${pluralize(kind.toLowerCase())}`;
}

function pluralize(s: string): string {
  if (s.endsWith("s")) return s;           // already plural
  if (s.endsWith("y") && !/[aeiou]y$/.test(s)) return s.slice(0, -1) + "ies";
  if (s.endsWith("x") || s.endsWith("ch") || s.endsWith("sh")) return s + "es";
  return s + "s";
}
```

- [ ] **Step 4: Run owners tests**

Run: `cd frontend && npx vitest run src/lib/kubernetes/__tests__/owners.test.ts`
Expected: all PASS.

### Resource cache store

- [ ] **Step 5: Implement resourceCache.svelte.ts**

```typescript
/**
 * Global cache of watched resources, indexed by (contextName, gvr) → (uid → object).
 * Populated by ResourceStore instances as they receive watch events. Used by
 * RelatedResourcesPanel for reverse-lookup by ownerReferences.uid.
 */
class ResourceCache {
  private cache = $state<Map<string, Map<string, Record<string, unknown>>>>(new Map());

  private keyFor(ctx: string, gvr: string): string { return `${ctx}::${gvr}`; }

  upsert(ctx: string, gvr: string, obj: Record<string, unknown>): void {
    const uid = (obj as any)?.metadata?.uid as string | undefined;
    if (!uid) return;
    const key = this.keyFor(ctx, gvr);
    let m = this.cache.get(key);
    if (!m) {
      m = new Map();
      this.cache.set(key, m);
    }
    m.set(uid, obj);
  }

  remove(ctx: string, gvr: string, uid: string): void {
    const m = this.cache.get(this.keyFor(ctx, gvr));
    m?.delete(uid);
  }

  /**
   * Scan all watched GVRs (in the given context) for objects whose
   * ownerReferences include the given ownerUid. Returns results grouped by
   * GVR for display.
   */
  findByOwnerUID(ctx: string, ownerUid: string): Array<{ gvr: string; items: Record<string, unknown>[] }> {
    const results: Array<{ gvr: string; items: Record<string, unknown>[] }> = [];
    for (const [key, byUid] of this.cache.entries()) {
      if (!key.startsWith(`${ctx}::`)) continue;
      const gvr = key.slice(ctx.length + 2);
      const matches: Record<string, unknown>[] = [];
      for (const obj of byUid.values()) {
        const refs = (obj as any)?.metadata?.ownerReferences;
        if (!Array.isArray(refs)) continue;
        if (refs.some((r: any) => r?.uid === ownerUid)) {
          matches.push(obj);
        }
      }
      if (matches.length > 0) results.push({ gvr, items: matches });
    }
    return results;
  }
}

export const resourceCache = new ResourceCache();
```

- [ ] **Step 6: Unit test resourceCache**

Create `frontend/src/lib/stores/__tests__/resourceCache.test.ts`:

```typescript
import { describe, it, expect, beforeEach } from "vitest";
import { resourceCache } from "../resourceCache.svelte";

describe("resourceCache", () => {
  beforeEach(() => {
    // reset cache between tests (access private via any)
    (resourceCache as any).cache = new Map();
  });

  it("upsert + findByOwnerUID", () => {
    resourceCache.upsert("c", "apps.v1.replicasets", {
      metadata: { uid: "rs-1", ownerReferences: [{ uid: "deploy-1", kind: "Deployment" }] },
    });
    resourceCache.upsert("c", "core.v1.pods", {
      metadata: { uid: "pod-1", ownerReferences: [{ uid: "rs-1", kind: "ReplicaSet" }] },
    });
    resourceCache.upsert("c", "core.v1.pods", {
      metadata: { uid: "pod-2", ownerReferences: [{ uid: "other", kind: "ReplicaSet" }] },
    });

    const byDeploy = resourceCache.findByOwnerUID("c", "deploy-1");
    expect(byDeploy).toEqual([
      { gvr: "apps.v1.replicasets", items: [expect.objectContaining({ metadata: expect.objectContaining({ uid: "rs-1" }) })] },
    ]);

    const byRS = resourceCache.findByOwnerUID("c", "rs-1");
    expect(byRS.length).toBe(1);
    expect(byRS[0].items.length).toBe(1);
    expect((byRS[0].items[0] as any).metadata.uid).toBe("pod-1");
  });

  it("remove evicts object", () => {
    resourceCache.upsert("c", "core.v1.pods", {
      metadata: { uid: "pod-1", ownerReferences: [{ uid: "rs-1" }] },
    });
    resourceCache.remove("c", "core.v1.pods", "pod-1");
    expect(resourceCache.findByOwnerUID("c", "rs-1")).toEqual([]);
  });
});
```

- [ ] **Step 7: Run resourceCache tests**

Run: `cd frontend && npx vitest run src/lib/stores/__tests__/resourceCache.test.ts`
Expected: PASS.

### Hydrate cache from ResourceStore

- [ ] **Step 8: Add cache hooks to resource.svelte.ts**

Open `frontend/src/lib/stores/resource.svelte.ts`. Find:
- Initial list population (where `items` is first set from `ListResources`).
- Watch event handler (`ADDED` / `MODIFIED` / `DELETED`).

Import and call:

```typescript
import { resourceCache } from "./resourceCache.svelte";

// where items are first set after ListResources:
for (const it of initialItems) {
  resourceCache.upsert(this.contextName, this.gvr, it);
}

// inside the watch handler:
if (type === "ADDED" || type === "MODIFIED") {
  resourceCache.upsert(this.contextName, this.gvr, object);
} else if (type === "DELETED") {
  const uid = object?.metadata?.uid;
  if (uid) resourceCache.remove(this.contextName, this.gvr, uid);
}
```

If `contextName` / `gvr` are passed to `start()` but not stored as fields, store them. Also ensure the existing `items` state update still happens — cache updates are additive.

- [ ] **Step 9: Run frontend tests**

Run: `cd frontend && pnpm test`
Expected: existing tests PASS.

The controller prepared a fresh working-copy commit for Task 1. Do NOT run `jj new` or `jj desc` — snapshot captures your changes automatically.

---

## Task 2: Build OwnerChain and RelatedResourcesPanel

**Files:**
- Create: `frontend/src/lib/components/panels/OwnerChain.svelte`
- Modify: `frontend/src/lib/components/panels/OverviewPanel.svelte` — embed `OwnerChain`
- Create: `frontend/src/lib/components/panels/RelatedResourcesPanel.svelte`
- Create: `frontend/src/lib/__tests__/RelatedResourcesPanel.svelte.test.ts`
- Modify: `frontend/src/lib/components/ResourceDetail.svelte` — register `RelatedResourcesPanel`

### OwnerChain component

- [ ] **Step 1: Implement OwnerChain**

```svelte
<script lang="ts">
  import { goto } from "$app/navigation";
  import { getOwnerReferences, gvrFromAPIVersion } from "../../kubernetes/owners";
  import { GetResource } from "../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js";

  interface Props {
    contextName: string;
    obj: Record<string, unknown>;
  }
  let { contextName, obj }: Props = $props();

  interface ChainNode { gvr: string; namespace: string; name: string; kind: string; }

  let chain = $state<ChainNode[]>([]);

  $effect(() => {
    (async () => {
      const result: ChainNode[] = [];
      let current: Record<string, unknown> | null = obj;
      for (let depth = 0; depth < 5; depth++) {
        const refs = getOwnerReferences(current);
        if (refs.length === 0) break;
        const controller = refs.find((r) => r.controller) ?? refs[0];
        const parentGvr = gvrFromAPIVersion(controller.apiVersion, controller.kind);
        const parentNs = (current as any)?.metadata?.namespace ?? "";
        result.push({ gvr: parentGvr, namespace: parentNs, name: controller.name, kind: controller.kind });
        try {
          current = await GetResource(contextName, parentGvr, parentNs, controller.name);
        } catch {
          break;
        }
      }
      chain = result;
    })();
  });

  function navigate(n: ChainNode) {
    goto(`/c/${encodeURIComponent(contextName)}/${n.gvr}/${encodeURIComponent(n.namespace)}/${encodeURIComponent(n.name)}`);
  }
</script>

{#if chain.length > 0}
  <div class="text-xs text-muted flex flex-wrap items-center gap-1">
    <span>Owned by:</span>
    {#each chain as n, i}
      {#if i > 0}<span>→</span>{/if}
      <button class="underline text-accent hover:text-accent/80" onclick={() => navigate(n)}>
        {n.kind}/{n.name}
      </button>
    {/each}
  </div>
{/if}
```

**Notes:**
- Route pattern uses `/c/:ctx/:gvr/:ns/:name` per the CLAUDE.md routes description.
- If `$app/navigation` isn't how this project navigates (check `routes.ts`), swap `goto` for the project's router helper.

- [ ] **Step 2: Embed in OverviewPanel**

Open the Overview panel file (located in Phase 5 Task 2). Near the top of its template, add:

```svelte
<script lang="ts">
  // …existing imports…
  import OwnerChain from "./OwnerChain.svelte";

  interface Props {
    contextName: string;
    // …existing props…
  }
  // make sure contextName is in props
</script>

<OwnerChain {contextName} {obj} />

<!-- existing overview field rendering -->
```

Ensure `ResourceDetail.svelte` passes `contextName` to the overview panel (it should already, since other panels need it).

- [ ] **Step 3: Type-check**

Run: `cd frontend && pnpm check`
Expected: exits 0.

### RelatedResourcesPanel component

- [ ] **Step 4: Implement RelatedResourcesPanel**

```svelte
<script lang="ts">
  import { goto } from "$app/navigation";
  import { resourceCache } from "../../stores/resourceCache.svelte";

  interface Props {
    contextName: string;
    obj: Record<string, unknown>;
  }
  let { contextName, obj }: Props = $props();

  let uid = $derived(((obj as any)?.metadata?.uid as string) ?? "");
  let groups = $derived(uid ? resourceCache.findByOwnerUID(contextName, uid) : []);

  function nav(gvr: string, item: Record<string, unknown>) {
    const ns = (item as any)?.metadata?.namespace ?? "";
    const name = (item as any)?.metadata?.name ?? "";
    goto(`/c/${encodeURIComponent(contextName)}/${gvr}/${encodeURIComponent(ns)}/${encodeURIComponent(name)}`);
  }
</script>

{#if groups.length === 0}
  <div class="p-4 text-muted text-sm">
    No related resources found in active watches.
    <div class="text-xs mt-1">Klados only shows related resources that are currently being watched.</div>
  </div>
{:else}
  <div class="p-4 space-y-4">
    {#each groups as g (g.gvr)}
      <section>
        <h3 class="text-xs font-semibold uppercase text-muted mb-2">
          {g.gvr} ({g.items.length})
        </h3>
        <ul class="space-y-1">
          {#each g.items as it}
            {@const name = (it as any)?.metadata?.name ?? ""}
            {@const ns = (it as any)?.metadata?.namespace ?? ""}
            <li>
              <button class="text-accent underline hover:text-accent/80 text-sm" onclick={() => nav(g.gvr, it)}>
                {ns ? `${ns}/` : ""}{name}
              </button>
            </li>
          {/each}
        </ul>
      </section>
    {/each}
  </div>
{/if}
```

- [ ] **Step 5: Register in ResourceDetail**

Open `frontend/src/lib/components/ResourceDetail.svelte`, add import and entry to panel map:

```typescript
import RelatedResourcesPanel from "./panels/RelatedResourcesPanel.svelte";

// …in the panel map…
["related", RelatedResourcesPanel as PanelComponent],
```

Ensure the `contextName` prop is passed to this panel.

- [ ] **Step 6: Tests**

Create `frontend/src/lib/__tests__/RelatedResourcesPanel.svelte.test.ts`:

```typescript
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/svelte";
import RelatedResourcesPanel from "../components/panels/RelatedResourcesPanel.svelte";
import { resourceCache } from "../stores/resourceCache.svelte";

describe("RelatedResourcesPanel", () => {
  beforeEach(() => {
    (resourceCache as any).cache = new Map();
  });

  it("shows empty state when no related resources in cache", () => {
    render(RelatedResourcesPanel, {
      props: {
        contextName: "c",
        obj: { metadata: { uid: "x" } },
      },
    });
    expect(screen.getByText(/No related resources/)).toBeInTheDocument();
  });

  it("groups related items by GVR", () => {
    resourceCache.upsert("c", "core.v1.pods", {
      metadata: { uid: "p1", name: "p1", namespace: "default", ownerReferences: [{ uid: "owner", kind: "ReplicaSet" }] },
    });
    resourceCache.upsert("c", "core.v1.pods", {
      metadata: { uid: "p2", name: "p2", namespace: "default", ownerReferences: [{ uid: "owner", kind: "ReplicaSet" }] },
    });
    render(RelatedResourcesPanel, {
      props: {
        contextName: "c",
        obj: { metadata: { uid: "owner" } },
      },
    });
    expect(screen.getByText(/core\.v1\.pods \(2\)/)).toBeInTheDocument();
    expect(screen.getByText("default/p1")).toBeInTheDocument();
    expect(screen.getByText("default/p2")).toBeInTheDocument();
  });
});
```

- [ ] **Step 7: Run tests**

Run: `cd frontend && npx vitest run src/lib/__tests__/RelatedResourcesPanel.svelte.test.ts`
Expected: PASS.

The controller prepared a fresh working-copy commit for Task 2. Do NOT run `jj new` or `jj desc` — snapshot captures your changes automatically.

---

## Task 3: Manual verification

- [ ] **Step 1: Launch dev mode**

Run: `task dev`

- [ ] **Step 2: Verify owner chain**

Navigate to a Pod detail page. Confirm:
- The overview shows `Owned by: ReplicaSet/xxx → Deployment/yyy`.
- Clicking each breadcrumb navigates to that owner.

- [ ] **Step 3: Verify related resources**

Navigate to the Deployment that owns the Pod. Open the Related tab. Confirm the Pod(s) and ReplicaSet(s) appear, grouped by GVR.

No additional commit needed — the controller will close out Phase 6 after Task 3 passes.

---

## Self-Review Checklist

- [x] Owner chain respects 5-level cap (loop guard) (3 tasks not 6).
- [x] Owner chain prefers controller reference when multiple owners.
- [x] Related lookup only uses cached/watched resources (per spec).
- [x] Cache hydrated from both initial List and Watch events.
- [x] Naive pluralization documented as best-effort — note about edge cases.
- [x] Commits per task.
