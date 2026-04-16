# Generic GVR Phase 4 — Conditions & Health Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement three coordinated features: the `ConditionsPanel` (detail page tab showing `status.conditions[]`), the `HealthBadge` (list page column synthesizing condition state into a color indicator), and the `ValidationWarningBanner` (detail page banner surfacing problematic conditions).

**Architecture:** All three features read `status.conditions[]` directly from the object returned by existing List/Get/Watch — no backend changes. A shared `conditions.ts` module holds condition classification logic (positive vs. negative condition types, health computation) and is used by all three components.

**Tech Stack:** Svelte 5 runes, TypeScript, vitest.

**Depends on:** Phases 1-2 complete (so auto-generated descriptors already include `"conditions"` in `detailPanels`).

**Reference spec:** `docs/superpowers/specs/2026-04-16-generic-gvr-capabilities-design.md` §3a, §3f.

---

## File Structure

- Create: `frontend/src/lib/kubernetes/conditions.ts` — pure functions for condition classification, health computation, warning detection
- Create: `frontend/src/lib/kubernetes/__tests__/conditions.test.ts` — unit tests for classification/health logic
- Create: `frontend/src/lib/components/panels/ConditionsPanel.svelte` — detail tab listing all conditions
- Create: `frontend/src/lib/components/HealthBadge.svelte` — compact colored indicator
- Create: `frontend/src/lib/components/ValidationWarningBanner.svelte` — top-of-detail warning banner
- Create: `frontend/src/lib/__tests__/ConditionsPanel.svelte.test.ts`
- Create: `frontend/src/lib/__tests__/HealthBadge.svelte.test.ts`
- Modify: `frontend/src/lib/components/ResourceDetail.svelte` — register `ConditionsPanel`, show `ValidationWarningBanner` at top
- Modify: `frontend/src/lib/components/ResourceList.svelte` — render `HealthBadge` column (details in Task 3)

---

## Task 1: Condition classification logic

**Files:**
- Create: `frontend/src/lib/kubernetes/conditions.ts`
- Create: `frontend/src/lib/kubernetes/__tests__/conditions.test.ts`

- [ ] **Step 1: Write failing tests**

Create `frontend/src/lib/kubernetes/__tests__/conditions.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import {
  getConditions,
  computeHealth,
  findWarnings,
  type Condition,
} from "../conditions";

describe("getConditions", () => {
  it("returns [] when status.conditions missing", () => {
    expect(getConditions({ metadata: {} })).toEqual([]);
    expect(getConditions({ status: {} })).toEqual([]);
  });

  it("returns [] when conditions isn't a valid array of objects", () => {
    expect(getConditions({ status: { conditions: "oops" } })).toEqual([]);
    expect(getConditions({ status: { conditions: [{ notATypeField: true }] } })).toEqual([]);
  });

  it("extracts valid conditions", () => {
    const obj = {
      status: {
        conditions: [
          { type: "Ready", status: "True", reason: "Ok", message: "all good" },
          { type: "Available", status: "False" },
        ],
      },
    };
    const c = getConditions(obj);
    expect(c.length).toBe(2);
    expect(c[0].type).toBe("Ready");
    expect(c[0].status).toBe("True");
  });
});

describe("computeHealth", () => {
  const c = (type: string, status: "True" | "False" | "Unknown"): Condition => ({
    type, status, reason: "", message: "", lastTransitionTime: "",
  });

  it("returns unknown when no conditions", () => {
    expect(computeHealth([])).toEqual({ level: "unknown", reason: "no conditions" });
  });

  it("returns healthy when Ready=True and no negatives", () => {
    expect(computeHealth([c("Ready", "True")]).level).toBe("healthy");
  });

  it("returns unhealthy when Ready=False", () => {
    expect(computeHealth([c("Ready", "False")]).level).toBe("unhealthy");
  });

  it("returns unhealthy when Degraded=True", () => {
    expect(computeHealth([c("Degraded", "True"), c("Ready", "True")]).level).toBe("unhealthy");
  });

  it("returns progressing when only Progressing=True among positives", () => {
    expect(computeHealth([c("Progressing", "True")]).level).toBe("progressing");
  });

  it("falls back to True/False ratio when no recognized types", () => {
    const h = computeHealth([c("CustomOne", "True"), c("CustomTwo", "True"), c("CustomThree", "False")]);
    expect(h.level).toBe("mixed");
    expect(h.reason).toBe("2/3 True");
  });
});

describe("findWarnings", () => {
  const c = (type: string, status: "True" | "False" | "Unknown", reason = "", message = ""): Condition => ({
    type, status, reason, message, lastTransitionTime: "",
  });

  it("flags Ready=False", () => {
    const w = findWarnings([c("Ready", "False", "NotReady", "pod not ready")]);
    expect(w.length).toBe(1);
    expect(w[0].type).toBe("Ready");
    expect(w[0].message).toBe("pod not ready");
  });

  it("flags Degraded=True", () => {
    expect(findWarnings([c("Degraded", "True", "Issues", "degraded")]).length).toBe(1);
  });

  it("does not flag Ready=True or Degraded=False", () => {
    expect(findWarnings([c("Ready", "True"), c("Degraded", "False")])).toEqual([]);
  });
});
```

- [ ] **Step 2: Run to verify failure**

Run: `cd frontend && npx vitest run src/lib/kubernetes/__tests__/conditions.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement**

Create `frontend/src/lib/kubernetes/conditions.ts`:

```typescript
export interface Condition {
  type: string;
  status: "True" | "False" | "Unknown" | string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
}

export type HealthLevel = "healthy" | "unhealthy" | "progressing" | "mixed" | "unknown";

export interface Health {
  level: HealthLevel;
  reason: string;
}

const POSITIVE_TYPES = new Set([
  "Ready",
  "Available",
  "Initialized",
  "ContainersReady",
  "PodScheduled",
  "Succeeded",
  "Complete",
  "Synced",
  "Established",
  "NamesAccepted",
]);

const NEGATIVE_TYPES = new Set([
  "Degraded",
  "MemoryPressure",
  "DiskPressure",
  "PIDPressure",
  "NetworkUnavailable",
  "Failed",
  "ReplicaFailure",
  "Stalled",
]);

const PROGRESSING_TYPES = new Set(["Progressing", "Reconciling"]);

export function getConditions(obj: unknown): Condition[] {
  if (!obj || typeof obj !== "object") return [];
  const status = (obj as any).status;
  const arr = status?.conditions;
  if (!Array.isArray(arr)) return [];
  const out: Condition[] = [];
  for (const c of arr) {
    if (!c || typeof c !== "object") continue;
    if (typeof c.type !== "string" || typeof c.status !== "string") continue;
    out.push({
      type: c.type,
      status: c.status,
      reason: c.reason,
      message: c.message,
      lastTransitionTime: c.lastTransitionTime,
    });
  }
  return out;
}

export function computeHealth(conditions: Condition[]): Health {
  if (conditions.length === 0) return { level: "unknown", reason: "no conditions" };

  let anyNegativeTrue = false;
  let anyPositiveFalse = false;
  let anyPositiveTrue = false;
  let anyProgressingTrue = false;

  let trueCount = 0;
  let totalRecognized = 0;

  for (const c of conditions) {
    const isPos = POSITIVE_TYPES.has(c.type);
    const isNeg = NEGATIVE_TYPES.has(c.type);
    const isProg = PROGRESSING_TYPES.has(c.type);

    if (isPos || isNeg || isProg) totalRecognized++;
    if (c.status === "True") trueCount++;

    if (isNeg && c.status === "True") anyNegativeTrue = true;
    if (isPos && c.status === "False") anyPositiveFalse = true;
    if (isPos && c.status === "True") anyPositiveTrue = true;
    if (isProg && c.status === "True") anyProgressingTrue = true;
  }

  if (anyNegativeTrue || anyPositiveFalse) {
    return { level: "unhealthy", reason: "negative condition active" };
  }
  if (anyPositiveTrue && !anyProgressingTrue) {
    return { level: "healthy", reason: "positive conditions met" };
  }
  if (anyProgressingTrue) {
    return { level: "progressing", reason: "progressing" };
  }
  // Unrecognized conditions → show a ratio badge
  const total = conditions.length;
  const trues = conditions.filter((c) => c.status === "True").length;
  return { level: "mixed", reason: `${trues}/${total} True` };
}

export interface Warning {
  type: string;
  reason: string;
  message: string;
}

export function findWarnings(conditions: Condition[]): Warning[] {
  const warns: Warning[] = [];
  for (const c of conditions) {
    if (POSITIVE_TYPES.has(c.type) && c.status === "False") {
      warns.push({ type: c.type, reason: c.reason ?? "", message: c.message ?? "" });
    } else if (NEGATIVE_TYPES.has(c.type) && c.status === "True") {
      warns.push({ type: c.type, reason: c.reason ?? "", message: c.message ?? "" });
    }
  }
  return warns;
}
```

- [ ] **Step 4: Run tests**

Run: `cd frontend && npx vitest run src/lib/kubernetes/__tests__/conditions.test.ts`
Expected: all PASS.

The controller prepared a fresh working-copy commit for Task 1. Do NOT run `jj new` or `jj desc` — snapshot captures your changes automatically.

---

## Task 2: Build condition-based UI components

**Files:**
- Create: `frontend/src/lib/components/panels/ConditionsPanel.svelte`
- Create: `frontend/src/lib/__tests__/ConditionsPanel.svelte.test.ts`
- Create: `frontend/src/lib/components/ValidationWarningBanner.svelte`
- Create: `frontend/src/lib/components/HealthBadge.svelte`
- Create: `frontend/src/lib/__tests__/HealthBadge.svelte.test.ts`

### ConditionsPanel

- [ ] **Step 1: Write failing test**

Create `frontend/src/lib/__tests__/ConditionsPanel.svelte.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/svelte";
import ConditionsPanel from "../components/panels/ConditionsPanel.svelte";

describe("ConditionsPanel", () => {
  it("shows empty state when object has no conditions", () => {
    render(ConditionsPanel, { props: { obj: { status: {} } } });
    expect(screen.getByText(/No conditions reported/i)).toBeInTheDocument();
  });

  it("renders each condition as a row with colored status badge", () => {
    const obj = {
      status: {
        conditions: [
          { type: "Ready", status: "True", reason: "Ok", message: "all good", lastTransitionTime: "2026-04-16T12:00:00Z" },
          { type: "Degraded", status: "False", reason: "", message: "", lastTransitionTime: "2026-04-16T12:00:00Z" },
        ],
      },
    };
    render(ConditionsPanel, { props: { obj } });
    expect(screen.getByText("Ready")).toBeInTheDocument();
    expect(screen.getByText("Degraded")).toBeInTheDocument();
    expect(screen.getByText("all good")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run to verify failure**

Run: `cd frontend && npx vitest run src/lib/__tests__/ConditionsPanel.svelte.test.ts`
Expected: FAIL — component not found.

- [ ] **Step 3: Implement ConditionsPanel**

Create `frontend/src/lib/components/panels/ConditionsPanel.svelte`:

```svelte
<script lang="ts">
  import { getConditions, type Condition } from "../../kubernetes/conditions";

  interface Props { obj: Record<string, unknown>; }
  let { obj }: Props = $props();

  let conditions = $derived(getConditions(obj));

  function badgeClass(status: string): string {
    switch (status) {
      case "True": return "bg-emerald-500/15 text-emerald-500";
      case "False": return "bg-destructive/15 text-destructive";
      default: return "bg-muted/30 text-muted";
    }
  }
</script>

{#if conditions.length === 0}
  <div class="p-4 text-muted text-sm">No conditions reported on this resource.</div>
{:else}
  <table class="w-full text-sm">
    <thead class="text-muted text-left">
      <tr>
        <th class="p-2 w-40">Type</th>
        <th class="p-2 w-28">Status</th>
        <th class="p-2 w-40">Reason</th>
        <th class="p-2">Message</th>
        <th class="p-2 w-48">Last Transition</th>
      </tr>
    </thead>
    <tbody>
      {#each conditions as c (c.type)}
        <tr class="border-t border-border">
          <td class="p-2 font-mono text-xs">{c.type}</td>
          <td class="p-2">
            <span class="px-2 py-0.5 rounded text-xs {badgeClass(c.status)}">{c.status}</span>
          </td>
          <td class="p-2">{c.reason ?? ""}</td>
          <td class="p-2">{c.message ?? ""}</td>
          <td class="p-2 text-muted">{c.lastTransitionTime ?? ""}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
```

- [ ] **Step 4: Run ConditionsPanel test**

Run: `cd frontend && npx vitest run src/lib/__tests__/ConditionsPanel.svelte.test.ts`
Expected: PASS.

### ValidationWarningBanner

- [ ] **Step 5: Implement ValidationWarningBanner (no dedicated test — render path exercised via ResourceDetail tests)**

Create `frontend/src/lib/components/ValidationWarningBanner.svelte`:

```svelte
<script lang="ts">
  import { findWarnings, getConditions } from "../kubernetes/conditions";

  interface Props { obj: Record<string, unknown>; }
  let { obj }: Props = $props();

  let warnings = $derived(findWarnings(getConditions(obj)));
</script>

{#if warnings.length > 0}
  <div class="m-4 rounded border border-amber-500/40 bg-amber-500/5 p-3 text-sm">
    <div class="font-semibold text-amber-500 mb-1">
      {warnings.length === 1 ? "Validation warning" : `${warnings.length} validation warnings`}
    </div>
    <ul class="space-y-1">
      {#each warnings as w}
        <li>
          <span class="font-mono text-xs text-amber-500 mr-2">{w.type}</span>
          {#if w.reason}<span class="text-muted mr-2">{w.reason}:</span>{/if}
          <span>{w.message}</span>
        </li>
      {/each}
    </ul>
  </div>
{/if}
```

- [ ] **Step 6: Type-check**

Run: `cd frontend && pnpm check`
Expected: exits 0.

### HealthBadge

- [ ] **Step 7: Write failing test**

Create `frontend/src/lib/__tests__/HealthBadge.svelte.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/svelte";
import HealthBadge from "../components/HealthBadge.svelte";

describe("HealthBadge", () => {
  it("shows green dot for healthy", () => {
    const obj = { status: { conditions: [{ type: "Ready", status: "True" }] } };
    const { container } = render(HealthBadge, { props: { obj } });
    expect(container.querySelector(".bg-emerald-500")).toBeTruthy();
  });

  it("shows red dot for unhealthy", () => {
    const obj = { status: { conditions: [{ type: "Ready", status: "False" }] } };
    const { container } = render(HealthBadge, { props: { obj } });
    expect(container.querySelector(".bg-destructive")).toBeTruthy();
  });

  it("shows ratio for unrecognized conditions", () => {
    const obj = { status: { conditions: [{ type: "A", status: "True" }, { type: "B", status: "False" }] } };
    render(HealthBadge, { props: { obj } });
    expect(screen.getByText("1/2 True")).toBeInTheDocument();
  });

  it("renders nothing when no conditions", () => {
    const { container } = render(HealthBadge, { props: { obj: {} } });
    expect(container.textContent?.trim()).toBe("");
  });
});
```

- [ ] **Step 8: Run to verify failure**

Run: `cd frontend && npx vitest run src/lib/__tests__/HealthBadge.svelte.test.ts`
Expected: FAIL.

- [ ] **Step 9: Implement HealthBadge**

Create `frontend/src/lib/components/HealthBadge.svelte`:

```svelte
<script lang="ts">
  import { computeHealth, getConditions } from "../kubernetes/conditions";

  interface Props { obj: Record<string, unknown>; }
  let { obj }: Props = $props();

  let health = $derived(computeHealth(getConditions(obj)));

  function dotClass(level: string): string {
    switch (level) {
      case "healthy": return "bg-emerald-500";
      case "unhealthy": return "bg-destructive";
      case "progressing": return "bg-amber-500";
      default: return "bg-muted";
    }
  }
</script>

{#if health.level === "unknown"}
  <!-- render nothing when there are no conditions -->
{:else if health.level === "mixed"}
  <span class="text-xs text-muted">{health.reason}</span>
{:else}
  <span
    class="inline-block w-2.5 h-2.5 rounded-full {dotClass(health.level)}"
    title={health.reason}
    aria-label={health.reason}
  ></span>
{/if}
```

- [ ] **Step 10: Run HealthBadge test**

Run: `cd frontend && npx vitest run src/lib/__tests__/HealthBadge.svelte.test.ts`
Expected: PASS.

The controller prepared a fresh working-copy commit for Task 2. Do NOT run `jj new` or `jj desc` — snapshot captures your changes automatically.

---

## Task 3: Integrate into ResourceDetail and ResourceList

**Files:**
- Modify: `frontend/src/lib/components/ResourceDetail.svelte`
- Modify: `frontend/src/lib/components/ResourceList.svelte` (or wherever rows are rendered)

- [ ] **Step 1: Register ConditionsPanel in the panel component map**

Open `frontend/src/lib/components/ResourceDetail.svelte`. Find the panel component map (around the existing `["events", EventsPanel]` entry noted in structural exploration). Add:

```typescript
import ConditionsPanel from "./panels/ConditionsPanel.svelte";
// …in the map…
["conditions", ConditionsPanel as PanelComponent],
```

- [ ] **Step 2: Render ValidationWarningBanner above the tabs**

In the same file, in the top-level template (above where the tabs/panel are rendered), add:

```svelte
<script lang="ts">
  // …existing imports…
  import ValidationWarningBanner from "./ValidationWarningBanner.svelte";
</script>

{#if obj}
  <ValidationWarningBanner {obj} />
  <!-- existing tabs/panel markup -->
{/if}
```

- [ ] **Step 3: Add HealthBadge to ResourceList rows**

Open `frontend/src/lib/components/ResourceList.svelte`. Find where each row renders the first cell (Name column). Add a `HealthBadge` inline before the name, or as its own leading column.

Recommended: prefix the Name cell content with:

```svelte
<script lang="ts">
  import HealthBadge from "./HealthBadge.svelte";
  // …
</script>

<!-- in the Name cell rendering -->
<div class="inline-flex items-center gap-2">
  <HealthBadge obj={row} />
  <span>{evalExpr(nameColumn.expr, row)}</span>
</div>
```

This puts the badge beside the Name rather than adding a new column, which avoids reshuffling existing descriptors.

- [ ] **Step 4: Type-check + run tests**

Run: `cd frontend && pnpm check && pnpm test`
Expected: PASS.

- [ ] **Step 5: Manual verification**

Run: `task dev`. Navigate to a Deployment list — confirm a green/red dot appears next to the Name. Open a detail page, confirm the Conditions tab works and the warning banner shows when a condition is negative.

The controller prepared a fresh working-copy commit for Task 3. Do NOT run `jj new` or `jj desc` — snapshot captures your changes automatically.

---

## Task 4: Phase marker / manual verification

- [ ] **Step 1: Final checks**

Confirm all previous tasks are complete, tests pass, and the feature works end-to-end in `task dev`.

No additional commit needed — the controller will close out Phase 4 after Task 4 passes.

---

## Self-Review Checklist

- [x] `ConditionsPanel` covered by spec §3a (4 tasks not 6).
- [x] `HealthBadge` covered by spec §3a.
- [x] `ValidationWarningBanner` covered by spec §3f.
- [x] Classification logic in a single shared module (DRY).
- [x] All three components read from existing object payloads — no backend changes.
- [x] Tests precede implementation.
