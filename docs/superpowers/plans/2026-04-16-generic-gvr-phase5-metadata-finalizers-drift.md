# Generic GVR Phase 5 — Metadata, Finalizers, Drift Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Three per-object features that apply to all GVRs:
1. **MetadataPanel** — structured labels/annotations view on the detail page.
2. **Finalizers display** — list in overview panel + optional hidden list column.
3. **DriftPanel** — diff of current object against `kubectl.kubernetes.io/last-applied-configuration` annotation, shown only when that annotation exists.

**Architecture:** All three read from the existing object payload. The drift diff reuses `DiffView` (already in the codebase per Storybook). A small `metadata.ts` helper normalizes label/annotation maps and strips server-managed fields before diffing.

**Tech Stack:** Svelte 5, TypeScript, vitest, js-yaml (already a dep for YAML editors — verify).

**Depends on:** Phases 1-2 (auto-generated descriptors include `"metadata"` and `"drift"` panels).

**Reference spec:** `docs/superpowers/specs/2026-04-16-generic-gvr-capabilities-design.md` §3b (finalizers partial), §3d, §3e.

---

## File Structure

- Create: `frontend/src/lib/kubernetes/metadata.ts` — helpers: `getLabels`, `getAnnotations`, `getFinalizers`, `stripServerFields`, `getLastAppliedConfig`
- Create: `frontend/src/lib/kubernetes/__tests__/metadata.test.ts`
- Create: `frontend/src/lib/components/panels/MetadataPanel.svelte`
- Create: `frontend/src/lib/components/panels/DriftPanel.svelte`
- Modify: `frontend/src/lib/components/panels/OverviewPanel.svelte` (or equivalent — find the built-in overview) — show finalizers list
- Modify: `frontend/src/lib/components/ResourceDetail.svelte` — register new panels, conditionally show `DriftPanel` only when last-applied annotation exists
- Modify: `frontend/src/lib/registry/index.ts` — mark `"drift"` panel as conditional (see Task 5)

---

## Task 1: Metadata helpers

**Files:**
- Create: `frontend/src/lib/kubernetes/metadata.ts`
- Create: `frontend/src/lib/kubernetes/__tests__/metadata.test.ts`

- [ ] **Step 1: Write failing tests**

```typescript
import { describe, it, expect } from "vitest";
import {
  getLabels,
  getAnnotations,
  getFinalizers,
  stripServerFields,
  getLastAppliedConfig,
  LAST_APPLIED_ANNOTATION,
} from "../metadata";

describe("getLabels / getAnnotations", () => {
  it("returns empty objects when missing", () => {
    expect(getLabels({})).toEqual({});
    expect(getAnnotations({ metadata: {} })).toEqual({});
  });
  it("extracts label and annotation maps", () => {
    const obj = { metadata: { labels: { app: "x" }, annotations: { a: "1" } } };
    expect(getLabels(obj)).toEqual({ app: "x" });
    expect(getAnnotations(obj)).toEqual({ a: "1" });
  });
});

describe("getFinalizers", () => {
  it("returns [] when missing", () => {
    expect(getFinalizers({})).toEqual([]);
    expect(getFinalizers({ metadata: {} })).toEqual([]);
  });
  it("returns the finalizers list", () => {
    const obj = { metadata: { finalizers: ["foregroundDeletion", "example.com/cleanup"] } };
    expect(getFinalizers(obj)).toEqual(["foregroundDeletion", "example.com/cleanup"]);
  });
});

describe("stripServerFields", () => {
  it("removes server-managed fields", () => {
    const obj = {
      apiVersion: "v1",
      kind: "Pod",
      metadata: {
        name: "p",
        namespace: "default",
        resourceVersion: "123",
        uid: "abc",
        creationTimestamp: "2026-04-16T00:00:00Z",
        generation: 1,
        selfLink: "/api/v1/pods/p",
        managedFields: [{ manager: "kubectl" }],
      },
      spec: { x: 1 },
      status: { ready: true },
    };
    const out = stripServerFields(obj);
    expect(out.metadata.resourceVersion).toBeUndefined();
    expect(out.metadata.uid).toBeUndefined();
    expect(out.metadata.creationTimestamp).toBeUndefined();
    expect(out.metadata.generation).toBeUndefined();
    expect(out.metadata.selfLink).toBeUndefined();
    expect(out.metadata.managedFields).toBeUndefined();
    expect(out.status).toBeUndefined();
    expect(out.metadata.name).toBe("p");
    expect(out.spec).toEqual({ x: 1 });
  });
  it("does not mutate the input", () => {
    const obj = { metadata: { uid: "x" } };
    stripServerFields(obj);
    expect(obj.metadata.uid).toBe("x");
  });
});

describe("getLastAppliedConfig", () => {
  it("returns null when annotation missing", () => {
    expect(getLastAppliedConfig({})).toBeNull();
  });
  it("parses the annotation as JSON", () => {
    const obj = { metadata: { annotations: { [LAST_APPLIED_ANNOTATION]: '{"spec":{"x":1}}' } } };
    expect(getLastAppliedConfig(obj)).toEqual({ spec: { x: 1 } });
  });
  it("returns null when JSON is malformed", () => {
    const obj = { metadata: { annotations: { [LAST_APPLIED_ANNOTATION]: "{not json" } } };
    expect(getLastAppliedConfig(obj)).toBeNull();
  });
});
```

Save as `frontend/src/lib/kubernetes/__tests__/metadata.test.ts`.

- [ ] **Step 2: Run to verify failure**

Run: `cd frontend && npx vitest run src/lib/kubernetes/__tests__/metadata.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement**

Create `frontend/src/lib/kubernetes/metadata.ts`:

```typescript
export const LAST_APPLIED_ANNOTATION = "kubectl.kubernetes.io/last-applied-configuration";

const SERVER_META_FIELDS = [
  "resourceVersion",
  "uid",
  "creationTimestamp",
  "generation",
  "selfLink",
  "managedFields",
];

function getMetadata(obj: unknown): Record<string, any> {
  if (!obj || typeof obj !== "object") return {};
  const m = (obj as any).metadata;
  return m && typeof m === "object" ? m : {};
}

export function getLabels(obj: unknown): Record<string, string> {
  const l = getMetadata(obj).labels;
  return l && typeof l === "object" ? { ...l } : {};
}

export function getAnnotations(obj: unknown): Record<string, string> {
  const a = getMetadata(obj).annotations;
  return a && typeof a === "object" ? { ...a } : {};
}

export function getFinalizers(obj: unknown): string[] {
  const f = getMetadata(obj).finalizers;
  return Array.isArray(f) ? [...f] : [];
}

export function stripServerFields(obj: Record<string, any>): Record<string, any> {
  const copy: Record<string, any> = JSON.parse(JSON.stringify(obj ?? {}));
  if (copy.metadata && typeof copy.metadata === "object") {
    for (const f of SERVER_META_FIELDS) delete copy.metadata[f];
  }
  delete copy.status;
  return copy;
}

export function getLastAppliedConfig(obj: unknown): Record<string, any> | null {
  const ann = getAnnotations(obj);
  const raw = ann[LAST_APPLIED_ANNOTATION];
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch {
    return null;
  }
}
```

- [ ] **Step 4: Run tests**

Run: `cd frontend && npx vitest run src/lib/kubernetes/__tests__/metadata.test.ts`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
jj desc -m "kubernetes: metadata helpers for labels, annotations, finalizers, drift"
```

---

## Task 2: MetadataPanel component

**Files:**
- Create: `frontend/src/lib/components/panels/MetadataPanel.svelte`

- [ ] **Step 1: Implement**

```svelte
<script lang="ts">
  import {
    getLabels,
    getAnnotations,
    LAST_APPLIED_ANNOTATION,
  } from "../../kubernetes/metadata";

  interface Props { obj: Record<string, unknown>; }
  let { obj }: Props = $props();

  let labels = $derived(getLabels(obj));
  let annotations = $derived(() => {
    const all = getAnnotations(obj);
    // Exclude last-applied — handled by DriftPanel
    const { [LAST_APPLIED_ANNOTATION]: _, ...rest } = all;
    return rest;
  });
  let expanded = $state<Record<string, boolean>>({});

  function toggle(key: string) { expanded[key] = !expanded[key]; }
  function isLong(v: string) { return v.length > 120; }
</script>

<div class="p-4 space-y-6 text-sm">
  <section>
    <h3 class="font-semibold mb-2">Labels</h3>
    {#if Object.keys(labels).length === 0}
      <div class="text-muted">No labels.</div>
    {:else}
      <div class="flex flex-wrap gap-2">
        {#each Object.entries(labels) as [k, v]}
          <span class="rounded bg-surface border border-border px-2 py-0.5 font-mono text-xs">
            {k}={v}
          </span>
        {/each}
      </div>
    {/if}
  </section>

  <section>
    <h3 class="font-semibold mb-2">Annotations</h3>
    {#if Object.keys(annotations()).length === 0}
      <div class="text-muted">No annotations.</div>
    {:else}
      <table class="w-full text-xs font-mono">
        <tbody>
          {#each Object.entries(annotations()) as [k, v]}
            <tr class="border-b border-border">
              <td class="p-2 align-top w-64 break-all">{k}</td>
              <td class="p-2 align-top break-all">
                {#if isLong(v)}
                  {#if expanded[k]}
                    <span class="whitespace-pre-wrap">{v}</span>
                    <button class="ml-2 text-accent text-xs" onclick={() => toggle(k)}>collapse</button>
                  {:else}
                    <span>{v.slice(0, 120)}…</span>
                    <button class="ml-2 text-accent text-xs" onclick={() => toggle(k)}>expand</button>
                  {/if}
                {:else}
                  <span>{v}</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </section>
</div>
```

- [ ] **Step 2: Quick render test**

Create `frontend/src/lib/__tests__/MetadataPanel.svelte.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/svelte";
import MetadataPanel from "../components/panels/MetadataPanel.svelte";
import { LAST_APPLIED_ANNOTATION } from "../kubernetes/metadata";

describe("MetadataPanel", () => {
  it("renders labels and annotations", () => {
    const obj = {
      metadata: {
        labels: { app: "x" },
        annotations: { foo: "bar" },
      },
    };
    render(MetadataPanel, { props: { obj } });
    expect(screen.getByText("app=x")).toBeInTheDocument();
    expect(screen.getByText("foo")).toBeInTheDocument();
    expect(screen.getByText("bar")).toBeInTheDocument();
  });

  it("excludes last-applied annotation", () => {
    const obj = {
      metadata: {
        annotations: { [LAST_APPLIED_ANNOTATION]: "{}", other: "visible" },
      },
    };
    render(MetadataPanel, { props: { obj } });
    expect(screen.queryByText(LAST_APPLIED_ANNOTATION)).toBeNull();
    expect(screen.getByText("other")).toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Run tests**

Run: `cd frontend && npx vitest run src/lib/__tests__/MetadataPanel.svelte.test.ts`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
jj new && jj desc -m "MetadataPanel: structured labels/annotations view"
```

---

## Task 3: Finalizers in Overview

**Files:**
- Modify: whichever `OverviewPanel.svelte` / built-in overview component exists. Locate via `grep -rn "overviewFields" frontend/src | head`

- [ ] **Step 1: Locate the overview component**

Run: `grep -rln 'overviewFields\|OverviewPanel' frontend/src`
Expected output names the file rendering the overview panel (likely `frontend/src/lib/components/panels/OverviewPanel.svelte` or similar). Call it `OverviewPanel.svelte` below.

- [ ] **Step 2: Add finalizers section**

Near the end of the overview's template (after the overview fields loop), add:

```svelte
<script lang="ts">
  // …existing imports…
  import { getFinalizers } from "../../kubernetes/metadata";

  // …existing props/logic…
  let finalizers = $derived(getFinalizers(obj));
</script>

<!-- after overview fields loop -->
{#if finalizers.length > 0}
  <section class="px-4 pb-4">
    <h3 class="text-xs font-semibold text-muted uppercase mb-2">Finalizers</h3>
    <div class="flex flex-wrap gap-2">
      {#each finalizers as f}
        <span class="rounded bg-surface border border-border px-2 py-0.5 font-mono text-xs">{f}</span>
      {/each}
    </div>
  </section>
{/if}
```

- [ ] **Step 3: Type-check + run tests**

Run: `cd frontend && pnpm check && pnpm test`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
jj new && jj desc -m "OverviewPanel: show finalizers when present"
```

---

## Task 4: DriftPanel component

**Files:**
- Create: `frontend/src/lib/components/panels/DriftPanel.svelte`
- Locate: the existing `DiffView` component — search `grep -rn 'DiffView' frontend/src` to find import path

- [ ] **Step 1: Check for js-yaml dependency**

Run: `grep '"js-yaml"' frontend/package.json`
Expected: present. If not, install: `cd frontend && pnpm add js-yaml && pnpm add -D @types/js-yaml`.

- [ ] **Step 2: Implement**

```svelte
<script lang="ts">
  import { dump as yamlDump } from "js-yaml";
  import {
    getLastAppliedConfig,
    stripServerFields,
  } from "../../kubernetes/metadata";
  import DiffView from "@klados/ui"; // adjust to actual import path — verify via Storybook import

  interface Props { obj: Record<string, unknown>; }
  let { obj }: Props = $props();

  let lastApplied = $derived(getLastAppliedConfig(obj));

  let currentYaml = $derived(yamlDump(stripServerFields(obj as Record<string, any>)));
  let lastAppliedYaml = $derived(
    lastApplied ? yamlDump(stripServerFields(lastApplied)) : ""
  );
</script>

{#if !lastApplied}
  <div class="p-4 text-muted text-sm">
    No <code>last-applied-configuration</code> annotation on this resource.
    Drift detection is only available for resources managed with <code>kubectl apply</code>.
  </div>
{:else}
  <div class="h-full">
    <DiffView
      left={lastAppliedYaml}
      right={currentYaml}
      leftTitle="Last Applied"
      rightTitle="Current"
    />
  </div>
{/if}
```

**Note:** The `DiffView` import path depends on the project's actual setup. Check `apps/docs/src/stories/DiffView.stories.ts` for the correct import and props. If it lives in `frontend/src/lib/components/DiffView.svelte` or `@klados/ui`, adjust.

- [ ] **Step 3: Test**

Create `frontend/src/lib/__tests__/DriftPanel.svelte.test.ts`:

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/svelte";

vi.mock("@klados/ui", () => ({ default: vi.fn() }));

import DriftPanel from "../components/panels/DriftPanel.svelte";
import { LAST_APPLIED_ANNOTATION } from "../kubernetes/metadata";

describe("DriftPanel", () => {
  it("shows empty state when annotation missing", () => {
    render(DriftPanel, { props: { obj: {} } });
    expect(screen.getByText(/No.*last-applied/)).toBeInTheDocument();
  });

  it("does not show empty state when annotation present", () => {
    const obj = {
      metadata: {
        annotations: { [LAST_APPLIED_ANNOTATION]: '{"spec":{"x":1}}' },
      },
      spec: { x: 1 },
    };
    render(DriftPanel, { props: { obj } });
    expect(screen.queryByText(/No.*last-applied/)).toBeNull();
  });
});
```

If the DiffView mock path differs, adjust accordingly.

- [ ] **Step 4: Run tests**

Run: `cd frontend && npx vitest run src/lib/__tests__/DriftPanel.svelte.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
jj new && jj desc -m "DriftPanel: diff current object against last-applied-configuration"
```

---

## Task 5: Conditional DriftPanel rendering + wiring

**Files:**
- Modify: `frontend/src/lib/components/ResourceDetail.svelte`
- Modify: `frontend/src/lib/registry/generator.ts` — `"drift"` stays in panels but should only render when applicable (handled at the `ResourceDetail` level since tabs are string-identified)

- [ ] **Step 1: Register panels and conditionally hide Drift tab**

Open `frontend/src/lib/components/ResourceDetail.svelte`. Add imports:

```typescript
import MetadataPanel from "./panels/MetadataPanel.svelte";
import DriftPanel from "./panels/DriftPanel.svelte";
import { getLastAppliedConfig } from "../kubernetes/metadata";
```

Add to the panel component map:

```typescript
["metadata", MetadataPanel as PanelComponent],
["drift", DriftPanel as PanelComponent],
```

Before rendering the tab list, filter out `"drift"` if the resource has no last-applied annotation:

```typescript
let visiblePanels = $derived(() => {
  const all = descriptor.detailPanels ?? [];
  return all.filter((p) => {
    if (p === "drift" && !getLastAppliedConfig(obj)) return false;
    return true;
  });
});
```

Use `visiblePanels()` when rendering the tab strip (instead of `descriptor.detailPanels`).

- [ ] **Step 2: Type-check + tests**

Run: `cd frontend && pnpm check && pnpm test`
Expected: PASS.

- [ ] **Step 3: Manual verification**

Run: `task dev`. Open a resource that has a last-applied annotation (e.g. create a Deployment via `kubectl apply`). Confirm the Drift tab appears; on a resource created without `kubectl apply` (e.g. `kubectl create`), the Drift tab is absent.

- [ ] **Step 4: Commit**

```bash
jj new && jj desc -m "ResourceDetail: register Metadata/Drift panels, hide Drift when N/A"
```

---

## Task 6: Phase marker

- [ ] **Step 1: Commit phase marker**

```bash
jj new && jj desc -m "docs: phase 5 metadata/finalizers/drift complete"
```

---

## Self-Review Checklist

- [x] Metadata helpers unit-tested, no mutation of input.
- [x] MetadataPanel excludes last-applied annotation.
- [x] Finalizers show only when non-empty.
- [x] Drift tab conditional on annotation presence.
- [x] DriftPanel strips server fields before diffing.
- [x] Commits per task.
