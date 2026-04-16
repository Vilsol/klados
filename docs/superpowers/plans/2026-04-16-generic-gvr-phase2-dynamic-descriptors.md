# Generic GVR Phase 2 — Dynamic Descriptor Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When the frontend receives the enriched discovery payload from Phase 1, auto-generate Descriptors for every GVR that has no built-in or plugin descriptor. Each generated descriptor contains columns derived from `additionalPrinterColumns`, actions/panels conditioned on subresources, and the universal panels (Events, Conditions, Metadata, Drift, Related, YAML, Overview).

**Architecture:** A new pure function `generateDescriptor(APIResource): Descriptor` lives alongside the registry. `DescriptorRegistry.get(gvr)` resolves in the order: built-in → plugin → auto-generated (from cached discovery payload) → static fallback. The static fallback is preserved as a last-resort for GVRs with no discovery entry (e.g. lookups during brief windows before discovery completes).

**Tech Stack:** TypeScript, Svelte 5 runes, Vitest.

**Depends on:** Phase 1 complete (discovery payload includes `printerColumns`, `subresources`, `scaleSpec`).

**Reference spec:** `docs/superpowers/specs/2026-04-16-generic-gvr-capabilities-design.md` §2.

---

## File Structure

- Create: `frontend/src/lib/registry/generator.ts` — pure `generateDescriptor` function + helpers
- Create: `frontend/src/lib/registry/__tests__/generator.test.ts` — unit tests
- Modify: `frontend/src/lib/registry/index.ts` — integrate generator into `DescriptorRegistry.get()`; cache the latest discovery payload as the generator's input
- Modify: `frontend/src/lib/stores/cluster.svelte.ts` — after discovery event handler, call `registry.updateDiscovery(resources)` so the generator has fresh metadata

Keeping the generator in its own file means `index.ts` stays focused on registry lifecycle and the generator is independently testable.

---

## Task 1: Build generator.ts (helpers + `generateDescriptor`)

Single module combining three concerns: JSONPath→CEL conversion, CRD-type→renderType mapping, and full descriptor construction from an `APIResource`. Each helper is trivially testable in isolation, so they share one file and one test suite.

**Files:**
- Create: `frontend/src/lib/registry/generator.ts`
- Create: `frontend/src/lib/registry/__tests__/generator.test.ts`

- [ ] **Step 1: Write the full test suite (all failing)**

Create `frontend/src/lib/registry/__tests__/generator.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import {
  jsonPathToCEL,
  crdTypeToRenderType,
  generateDescriptor,
} from "../generator";
import type { APIResource } from "../../../../bindings/github.com/Vilsol/klados/internal/cluster/index.js";

describe("jsonPathToCEL", () => {
  it("strips leading dot", () => {
    expect(jsonPathToCEL(".spec.replicas")).toBe("spec.replicas");
  });

  it("preserves deep dotted paths", () => {
    expect(jsonPathToCEL(".status.loadBalancer.ingress")).toBe("status.loadBalancer.ingress");
  });

  it("passes through root $", () => {
    expect(jsonPathToCEL("$.metadata.name")).toBe("metadata.name");
  });

  it("returns empty for empty input", () => {
    expect(jsonPathToCEL("")).toBe("");
  });

  it("returns empty for unsupported filter expressions", () => {
    // We deliberately don't try to support JSONPath filter predicates.
    expect(jsonPathToCEL('.status.conditions[?(@.type=="Ready")].status')).toBe("");
  });
});

describe("crdTypeToRenderType", () => {
  it("maps date → age", () => {
    expect(crdTypeToRenderType("date")).toBe("age");
  });
  it("maps boolean → badge", () => {
    expect(crdTypeToRenderType("boolean")).toBe("badge");
  });
  it("maps string/integer/number → text", () => {
    expect(crdTypeToRenderType("string")).toBe("text");
    expect(crdTypeToRenderType("integer")).toBe("text");
    expect(crdTypeToRenderType("number")).toBe("text");
  });
  it("defaults unknown → text", () => {
    expect(crdTypeToRenderType("")).toBe("text");
    expect(crdTypeToRenderType("gibberish")).toBe("text");
  });
});

const baseResource = (over: Partial<APIResource> = {}): APIResource => ({
  GVR: "example.com.v1.widgets",
  Kind: "Widget",
  Namespaced: true,
  Subresources: { Scale: false, Status: false },
  PrinterColumns: [],
  ScaleSpec: undefined,
  ...over,
}) as APIResource;

describe("generateDescriptor", () => {
  it("prepends Name/Namespace/Age and appends printer columns", () => {
    const d = generateDescriptor(baseResource({
      PrinterColumns: [
        { Name: "Replicas", Type: "integer", JSONPath: ".spec.replicas", Priority: 0 },
        { Name: "Ready", Type: "string", JSONPath: ".status.ready", Priority: 1 },
      ] as APIResource["PrinterColumns"],
    }));

    expect(d.columns.map((c) => c.name)).toEqual(["Name", "Namespace", "Age", "Replicas", "Ready"]);
    expect(d.columns[3].expr).toBe("spec.replicas");
    expect(d.columns[3].renderType).toBe("text");
    expect(d.columns[4].hidden).toBe(true); // priority > 0
  });

  it("omits Namespace column for cluster-scoped resources", () => {
    const d = generateDescriptor(baseResource({ Namespaced: false }));
    expect(d.columns.map((c) => c.name)).toEqual(["Name", "Age"]);
    expect(d.clusterScoped).toBe(true);
  });

  it("skips printer columns with unsupported JSONPath", () => {
    const d = generateDescriptor(baseResource({
      PrinterColumns: [
        { Name: "Filtered", Type: "string", JSONPath: '.status.conditions[?(@.type=="Ready")].status', Priority: 0 },
      ] as APIResource["PrinterColumns"],
    }));
    expect(d.columns.map((c) => c.name)).toEqual(["Name", "Namespace", "Age"]);
  });

  it("adds Scale action when scale subresource present", () => {
    const d = generateDescriptor(baseResource({
      Subresources: { Scale: true, Status: false },
      ScaleSpec: { SpecReplicasPath: ".spec.replicas", StatusReplicasPath: ".status.replicas" },
    }));
    expect(d.actions.some((a) => a.name === "scale")).toBe(true);
    expect(d.columns.some((c) => c.name === "Replicas")).toBe(true);
  });

  it("does not duplicate Replicas column when printer columns already include one", () => {
    const d = generateDescriptor(baseResource({
      Subresources: { Scale: true, Status: false },
      ScaleSpec: { SpecReplicasPath: ".spec.replicas", StatusReplicasPath: ".status.replicas" },
      PrinterColumns: [
        { Name: "Replicas", Type: "integer", JSONPath: ".spec.replicas", Priority: 0 },
      ] as APIResource["PrinterColumns"],
    }));
    const replicaCols = d.columns.filter((c) => c.name === "Replicas");
    expect(replicaCols.length).toBe(1);
  });

  it("always includes universal detail panels", () => {
    const d = generateDescriptor(baseResource());
    expect(d.detailPanels).toEqual(expect.arrayContaining([
      "overview", "yaml", "events", "conditions", "metadata", "related", "drift",
    ]));
  });

  it("adds 'status' panel when status subresource present", () => {
    const d = generateDescriptor(baseResource({
      Subresources: { Scale: false, Status: true },
    }));
    expect(d.detailPanels).toContain("status");
  });

  it("always includes delete action and edit-yaml action", () => {
    const d = generateDescriptor(baseResource());
    const names = d.actions.map((a) => a.name);
    expect(names).toContain("delete");
    expect(names).toContain("edit-yaml");
  });
});
```

- [ ] **Step 2: Run tests to verify they all fail**

Run: `cd frontend && npx vitest run src/lib/registry/__tests__/generator.test.ts`
Expected: FAIL — module or exports not found.

- [ ] **Step 3: Implement all three helpers in `generator.ts`**

Create `frontend/src/lib/registry/generator.ts`:

```typescript
import type { APIResource } from "../../../bindings/github.com/Vilsol/klados/internal/cluster/index.js";
import type { resource } from "../../../bindings/github.com/Vilsol/klados/internal/services/index.js";

type Descriptor = resource.Descriptor;
type Column = resource.Column;
type Action = resource.Action;

export type RenderType = "text" | "badge" | "age" | "progress";

const UNIVERSAL_PANELS = [
  "overview",
  "yaml",
  "events",
  "conditions",
  "metadata",
  "related",
  "drift",
];

/**
 * Convert a JSONPath expression (as used in CRD additionalPrinterColumns)
 * to a CEL expression for our renderer. Only the simple dotted-path subset
 * is supported — JSONPath filter predicates return "" so callers can skip
 * those columns.
 */
export function jsonPathToCEL(jsonPath: string): string {
  if (!jsonPath) return "";
  let p = jsonPath.startsWith("$.") ? jsonPath.slice(2) : jsonPath;
  if (p.startsWith(".")) p = p.slice(1);
  if (p.includes("[") || p.includes("?") || p.includes("@")) return "";
  return p;
}

export function crdTypeToRenderType(t: string): RenderType {
  switch (t) {
    case "date":
      return "age";
    case "boolean":
      return "badge";
    default:
      return "text";
  }
}

/**
 * Build a Descriptor from an enriched APIResource (discovery payload). This
 * is called when no built-in or plugin descriptor exists for the GVR.
 */
export function generateDescriptor(r: APIResource): Descriptor {
  const [group, version, resourceName] = r.GVR.split(".").length >= 3
    ? splitGVR(r.GVR)
    : ["", "", r.GVR];

  const columns: Column[] = [];
  columns.push({ name: "Name", expr: "metadata.name", renderType: "text" });
  if (r.Namespaced) {
    columns.push({ name: "Namespace", expr: "metadata.namespace", renderType: "text" });
  }
  columns.push({ name: "Age", expr: "metadata.creationTimestamp", renderType: "age" });

  const existingNames = new Set(columns.map((c) => c.name));
  for (const pc of r.PrinterColumns ?? []) {
    const expr = jsonPathToCEL(pc.JSONPath);
    if (!expr) continue;
    if (existingNames.has(pc.Name)) continue;
    columns.push({
      name: pc.Name,
      expr,
      renderType: crdTypeToRenderType(pc.Type),
      hidden: (pc.Priority ?? 0) > 0,
    });
    existingNames.add(pc.Name);
  }

  // Scale subresource → ensure Replicas column + action
  if (r.Subresources?.Scale) {
    const specPath = jsonPathToCEL(r.ScaleSpec?.SpecReplicasPath ?? ".spec.replicas");
    if (!existingNames.has("Replicas") && specPath) {
      columns.push({ name: "Replicas", expr: specPath, renderType: "text" });
      existingNames.add("Replicas");
    }
  }

  const panels = [...UNIVERSAL_PANELS];
  if (r.Subresources?.Status) panels.push("status");

  const actions: Action[] = [
    { name: "edit-yaml", label: "Edit YAML" },
    { name: "delete", label: "Delete" },
  ];
  if (r.Subresources?.Scale) {
    actions.unshift({ name: "scale", label: "Scale" });
  }

  return {
    group,
    version,
    resource: resourceName,
    kind: r.Kind,
    columns,
    overviewFields: [
      { label: "Namespace", expr: "metadata.namespace", renderType: "text" },
      { label: "Age", expr: "metadata.creationTimestamp", renderType: "age" },
    ],
    detailPanels: panels,
    actions,
    clusterScoped: !r.Namespaced,
  } as Descriptor;
}

function splitGVR(gvr: string): [string, string, string] {
  // Format: "<group parts>.<version>.<resource>". Split from the right by 2.
  const lastDot = gvr.lastIndexOf(".");
  const secondLast = gvr.lastIndexOf(".", lastDot - 1);
  const group = gvr.slice(0, secondLast);
  const version = gvr.slice(secondLast + 1, lastDot);
  const resourceName = gvr.slice(lastDot + 1);
  return [group, version, resourceName];
}
```

If `resource.Descriptor`/`resource.Column`/`resource.Action` aren't exported under that namespace, open `frontend/bindings/github.com/Vilsol/klados/internal/services/index.js` and find the correct import path. The type names remain identical.

- [ ] **Step 4: Run tests — all should pass**

Run: `cd frontend && npx vitest run src/lib/registry/__tests__/generator.test.ts`
Expected: all 17 tests PASS.

- [ ] **Step 5: Commit**

The working copy is an empty commit prepared by the controller. Do NOT run `jj new` or `jj desc` — snapshot will capture the changes and the controller will handle the commit boundary.

---

## Task 2: Wire generator into DescriptorRegistry

**Files:**
- Modify: `frontend/src/lib/registry/index.ts`

- [ ] **Step 1: Read current `DescriptorRegistry.get()` and surrounding context**

Open `frontend/src/lib/registry/index.ts`. Locate:
- The `builtins` and `plugins` Maps
- The `fallbackDescriptor` constructor (~lines 214-237)
- The `get(gvr)` method
- The `setAvailableGVRs` / `isGVRAvailable` helpers

- [ ] **Step 2: Add discovery cache + integrate generator**

Inside the `DescriptorRegistry` class, add a field and a method:

```typescript
import type { APIResource } from "../../../bindings/github.com/Vilsol/klados/internal/cluster/index.js";
import { generateDescriptor } from "./generator";

// …inside the class…
private discovery: Map<string, APIResource> = new Map();

updateDiscovery(resources: APIResource[]): void {
  this.discovery = new Map(resources.map((r) => [r.GVR, r]));
}
```

Modify `get(gvr)` to consult the generator in the correct priority order. The existing method likely reads:

```typescript
get(gvr: string): Descriptor {
  const d = this.plugins.get(gvr) ?? this.builtins.get(gvr) ?? this.fallbackFor(gvr);
  return this.ensureControlledBy(d);
}
```

Change the fallback branch to:

```typescript
get(gvr: string): Descriptor {
  let d = this.plugins.get(gvr) ?? this.builtins.get(gvr);
  if (!d) {
    const discovered = this.discovery.get(gvr);
    d = discovered ? generateDescriptor(discovered) : this.fallbackFor(gvr);
  }
  return this.ensureControlledBy(d);
}
```

(The exact existing structure may differ — adapt the resolution order: built-in → plugin → generated → static. The spec dictates built-in takes highest priority, so check the existing ordering and preserve it.)

- [ ] **Step 3: Hook `updateDiscovery` from the cluster store**

Open `frontend/src/lib/stores/cluster.svelte.ts`. Find the `discovery:${ctxName}:resources` event handler (from Phase 1). Inside the handler, after the local store update, call:

```typescript
import { descriptorRegistry } from "../registry";
// …inside the handler, after setting availableResources…
descriptorRegistry.updateDiscovery(resources);
```

If `descriptorRegistry` is not the current export name, check `frontend/src/lib/registry/index.ts` for the singleton export (pattern: `export const registry = new DescriptorRegistry()` or similar) and use that name.

- [ ] **Step 4: Add a registry unit test covering resolution order**

Create `frontend/src/lib/registry/__tests__/resolution.test.ts`:

```typescript
import { describe, it, expect, beforeEach } from "vitest";
import { DescriptorRegistry } from "../index";

describe("DescriptorRegistry resolution order", () => {
  let reg: DescriptorRegistry;
  beforeEach(() => {
    reg = new DescriptorRegistry();
  });

  it("uses generated descriptor when built-in and plugin absent", () => {
    reg.updateDiscovery([{
      GVR: "example.com.v1.widgets",
      Kind: "Widget",
      Namespaced: true,
      Subresources: { Scale: false, Status: false },
      PrinterColumns: [],
      ScaleSpec: undefined,
    } as any]);
    const d = reg.get("example.com.v1.widgets");
    expect(d.kind).toBe("Widget");
    expect(d.detailPanels).toContain("conditions");
  });

  it("built-in takes priority over generated", () => {
    // Inject a fake built-in
    (reg as any).builtins.set("example.com.v1.widgets", { kind: "BUILTIN", columns: [], detailPanels: [], actions: [], overviewFields: [] });
    reg.updateDiscovery([{
      GVR: "example.com.v1.widgets", Kind: "Widget",
      Namespaced: true, Subresources: { Scale: false, Status: false },
      PrinterColumns: [], ScaleSpec: undefined,
    } as any]);
    const d = reg.get("example.com.v1.widgets");
    expect(d.kind).toBe("BUILTIN");
  });
});
```

Note: the `DescriptorRegistry` constructor may not be exported — if so, use the singleton. Adjust imports accordingly. If `builtins` is private, inject via the existing load path or expose a test helper `__setBuiltinForTest(gvr, d)` in the registry module.

- [ ] **Step 5: Run tests**

Run: `cd frontend && pnpm test`
Expected: all PASS.

- [ ] **Step 6: Type-check**

Run: `cd frontend && pnpm check`
Expected: exits 0.

- [ ] **Step 7: Commit**

The controller prepared a fresh working-copy commit for Task 2. Do NOT run `jj new` or `jj desc` — snapshot captures your changes automatically.

---

## Task 3: End-to-end manual verification

- [ ] **Step 1: Run dev mode**

Run: `task dev`

- [ ] **Step 2: Connect to a cluster with a CRD that has `additionalPrinterColumns`**

Navigate to `/c/<ctx>/<crd-gvr>`. Verify:
- The list page shows the printer columns after Name/Namespace/Age.
- Columns with `priority > 0` are hidden (accessible via column chooser if one exists).
- The detail page shows the universal tabs (Events will be implemented in Phase 3 — it may currently be blank/placeholder; that's expected).

- [ ] **Step 3: Phase marker**

No additional commit needed — the controller will close out Phase 2 after Task 3 passes.

---

## Self-Review Checklist

- [x] Every code block contains executable code.
- [x] Tests precede implementation (each task opens with a failing test suite).
- [x] Resolution priority matches spec: built-in → plugin → generated → static fallback.
- [x] `clusterScoped` flag correctly mirrors `!Namespaced`.
- [x] 3 tasks (not 5) — helpers consolidated into one generator module, followed by registry wiring, followed by manual verification.
