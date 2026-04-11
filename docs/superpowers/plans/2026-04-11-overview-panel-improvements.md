# Overview Panel Improvements — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve the OverviewPanel detail view with richer fields, click-to-copy values, tolerations section, compact conditions, and structured container accordion.

**Architecture:** Hybrid approach — simple scalar fields added via Go descriptors (no frontend changes), complex sections (tolerations, container accordion) hardcoded in OverviewPanel. New `CopyableValue` component in `@klados/ui` wraps all detail field values.

**Tech Stack:** Go (builtin.go descriptors), Svelte 5 (OverviewPanel, CopyableValue), Tailwind v4, lucide-svelte icons

---

### Task 1: CopyableValue Component

Create a reusable component in `@klados/ui` that wraps a value with click-to-copy, hover icon, and optional tooltip for raw values.

**Files:**
- Create: `packages/ui/src/lib/CopyableValue.svelte`
- Modify: `packages/ui/src/lib/index.ts` (add export)

- [ ] **Step 1: Create CopyableValue.svelte**

```svelte
<script lang="ts">
  import { Copy, Check } from 'lucide-svelte'

  let {
    value,
    rawValue,
    class: className = '',
  }: {
    value: string
    rawValue?: string
    class?: string
  } = $props()

  let copied = $state(false)
  let flashing = $state(false)

  async function copy() {
    await navigator.clipboard.writeText(rawValue ?? value)
    copied = true
    flashing = true
    setTimeout(() => flashing = false, 300)
    setTimeout(() => copied = false, 1500)
  }
</script>

<button
  onclick={copy}
  title={rawValue ?? value}
  class="group relative inline-flex items-center gap-1 cursor-pointer max-w-full
    hover:underline hover:decoration-dotted hover:decoration-muted
    transition-colors {flashing ? 'bg-accent/10' : ''} rounded px-0.5 -mx-0.5 {className}"
>
  <span class="truncate"><slot>{value}</slot></span>
  <span class="shrink-0 opacity-0 group-hover:opacity-60 transition-opacity">
    {#if copied}
      <Check size={12} />
    {:else}
      <Copy size={12} />
    {/if}
  </span>
</button>
```

- [ ] **Step 2: Export from index.ts**

Add to `packages/ui/src/lib/index.ts`:
```ts
export { default as CopyableValue } from './CopyableValue.svelte'
```

- [ ] **Step 3: Verify it builds**

Run: `cd frontend && pnpm check`
Expected: No type errors

- [ ] **Step 4: Commit**

```bash
jj desc -m "feat(ui): add CopyableValue component with click-to-copy, hover icon, and raw value tooltip"
jj new
```

---

### Task 2: Add New Overview Fields in Go Descriptors

Add the new `overviewFields` entries to pods, deployments, statefulsets, and daemonsets in `builtin.go`.

**Files:**
- Modify: `internal/resource/builtin.go`

- [ ] **Step 1: Add pod overview fields**

After the existing pod `OverviewFields` (line 29, after `Age`), add:
```go
{Label: "Service Account", Expr: "spec.serviceAccountName", RenderType: RenderText},
{Label: "QoS Class", Expr: "status.qosClass", RenderType: RenderBadge},
{Label: "Priority", Expr: "spec.priority", RenderType: RenderText},
{Label: "Restart Policy", Expr: "spec.restartPolicy", RenderType: RenderBadge},
{Label: "DNS Policy", Expr: "spec.dnsPolicy", RenderType: RenderText},
```

- [ ] **Step 2: Add deployment overview fields**

After the existing deployment `OverviewFields` (after `Age`), add:
```go
{Label: "Service Account", Expr: "spec.template.spec.serviceAccountName", RenderType: RenderText},
{Label: "Revision", Expr: "metadata.annotations['deployment.kubernetes.io/revision']", RenderType: RenderText},
```

Note: Strategy is already present in deployments.

- [ ] **Step 3: Add statefulset overview fields**

After the existing statefulset `OverviewFields` (after `Age`), add:
```go
{Label: "Update Strategy", Expr: "spec.updateStrategy.type", RenderType: RenderBadge},
{Label: "Service Account", Expr: "spec.template.spec.serviceAccountName", RenderType: RenderText},
{Label: "Service Name", Expr: "spec.serviceName", RenderType: RenderText},
```

- [ ] **Step 4: Add daemonset overview fields**

After the existing daemonset `OverviewFields` (after `Age`), add:
```go
{Label: "Update Strategy", Expr: "spec.updateStrategy.type", RenderType: RenderBadge},
{Label: "Service Account", Expr: "spec.template.spec.serviceAccountName", RenderType: RenderText},
```

- [ ] **Step 5: Verify CEL expressions parse**

Run: `go test ./internal/resource/ -v`
Expected: All existing tests pass (the registry validates CEL expressions on register)

- [ ] **Step 6: Commit**

```bash
jj desc -m "feat(resource): add service account, QoS, strategy, and other overview fields to pod/deployment/statefulset/daemonset descriptors"
jj new
```

---

### Task 3: Integrate CopyableValue into OverviewPanel + Conditions 3-col

Wire up CopyableValue in the overview fields grid, add raw value tooltips for age fields, and change conditions to 3 columns.

**Files:**
- Modify: `frontend/src/lib/components/panels/OverviewPanel.svelte`

- [ ] **Step 1: Update imports**

Add `CopyableValue` to the `@klados/ui` import:
```ts
import { SectionHeader, KeyValueBadge, EmptyState, StatusBadge, KeyValuePairEditor, CopyableValue } from '@klados/ui'
```

- [ ] **Step 2: Update renderValue to return both display and raw values**

Replace the existing `renderValue` function with one that returns both:
```ts
function getRawValue(expr: string): string {
  const raw = evalExpr(expr, obj)
  if (raw === null || raw === undefined) return ''
  return String(raw)
}

function renderValue(expr: string, renderType: string): string {
  const raw = evalExpr(expr, obj)
  if (renderType === 'age' && raw) {
    return formatAge(String(raw))
  }
  if (raw === null || raw === undefined) return '—'
  return String(raw)
}
```

- [ ] **Step 3: Wrap overview field values with CopyableValue**

Replace the overview fields `{#each}` block (lines 126–137) with:
```svelte
{#each descriptor.overviewFields as field}
  <div class="min-w-0">
    <div class="text-xs text-muted mb-0.5">{field.label}</div>
    {#if field.renderType === 'badge'}
      <CopyableValue
        value={renderValue(field.expr, field.renderType)}
        rawValue={getRawValue(field.expr)}
        class="text-xs font-mono"
      >
        <span class="bg-bg border border-border rounded px-2 py-0.5 inline-block">
          {renderValue(field.expr, field.renderType)}
        </span>
      </CopyableValue>
    {:else}
      <CopyableValue
        value={renderValue(field.expr, field.renderType)}
        rawValue={field.renderType === 'age' ? getRawValue(field.expr) : undefined}
        class="text-xs font-mono"
      />
    {/if}
  </div>
{/each}
```

- [ ] **Step 4: Change conditions grid to 3 columns**

Change line 225:
```svelte
<!-- Before -->
<div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
<!-- After -->
<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
```

- [ ] **Step 5: Verify it builds**

Run: `cd frontend && pnpm check`
Expected: No type errors

- [ ] **Step 6: Commit**

```bash
jj desc -m "feat(overview): integrate CopyableValue for click-to-copy fields, add raw value tooltips, 3-column conditions grid"
jj new
```

---

### Task 4: Tolerations Section

Add a dedicated collapsible tolerations card below Details, with a clickable count in the Details grid that scrolls to it.

**Files:**
- Modify: `frontend/src/lib/components/panels/OverviewPanel.svelte`

- [ ] **Step 1: Add tolerations state and helpers**

Add to the `<script>` section:
```ts
const tolerationsPath = $derived(
  obj.spec?.tolerations ?? obj.spec?.template?.spec?.tolerations ?? []
)
const tolerations = $derived<any[]>(tolerationsPath)
let tolerationsExpanded = $state(true)
let tolerationsEl: HTMLElement | undefined = $state()

function formatToleration(t: any): string {
  const key = t.key || '*'
  const op = t.operator === 'Exists' ? 'Exists' : `=${t.value ?? ''}`
  const effect = t.effect || 'All'
  const seconds = t.tolerationSeconds != null ? ` (${t.tolerationSeconds}s)` : ''
  return `${key} ${op} — ${effect}${seconds}`
}

function scrollToTolerations() {
  tolerationsExpanded = true
  tolerationsEl?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
}
```

- [ ] **Step 2: Add tolerations count to the Details grid**

After the `controllerRef` block (after line 150), before the closing `</div>` of the grid, add:
```svelte
{#if tolerations.length > 0}
  <div class="min-w-0">
    <div class="text-xs text-muted mb-0.5">Tolerations</div>
    <button
      onclick={scrollToTolerations}
      class="text-xs font-mono text-accent hover:underline"
    >{tolerations.length}</button>
  </div>
{/if}
```

- [ ] **Step 3: Add tolerations card section**

After the Details `</section>` (line 161), before the Labels section, add:
```svelte
{#if tolerations.length > 0}
  <section bind:this={tolerationsEl} class="bg-surface border border-border rounded-lg p-4">
    <button
      onclick={() => tolerationsExpanded = !tolerationsExpanded}
      class="flex items-center gap-1 w-full text-left"
    >
      <SectionHeader class="">{tolerationsExpanded ? '▾' : '▸'} Tolerations ({tolerations.length})</SectionHeader>
    </button>
    {#if tolerationsExpanded}
      <div class="flex flex-col gap-1 mt-3">
        {#each tolerations as t}
          <CopyableValue value={formatToleration(t)} class="text-xs font-mono" />
        {/each}
      </div>
    {/if}
  </section>
{/if}
```

- [ ] **Step 4: Verify it builds**

Run: `cd frontend && pnpm check`
Expected: No type errors

- [ ] **Step 5: Commit**

```bash
jj desc -m "feat(overview): add collapsible tolerations section with clickable count in details grid"
jj new
```

---

### Task 5: Container Card Accordion

Restructure container cards into accordion sections: Resources (expanded), Ports (expanded), Environment (collapsed), Mounts (collapsed). Hide sections with no data.

**Files:**
- Modify: `frontend/src/lib/components/panels/OverviewPanel.svelte`

- [ ] **Step 1: Replace expandedEnv/expandedMounts with accordion state**

Remove `expandedEnv` and `expandedMounts` state variables. Replace with a simple override map — Resources and Ports default to expanded, Env and Mounts default to collapsed, and user clicks override the default:

```ts
let sectionOverrides = $state<Record<string, boolean>>({})

function isSectionOpen(cname: string, section: string): boolean {
  const key = `${cname}:${section}`
  if (key in sectionOverrides) return sectionOverrides[key]
  return section === 'resources' || section === 'ports'
}

function toggleSection(cname: string, section: string) {
  const key = `${cname}:${section}`
  sectionOverrides = { ...sectionOverrides, [key]: !isSectionOpen(cname, section) }
}
```

- [ ] **Step 2: Replace container card body with accordion sections**

Replace the container card body (everything inside the `<div class="bg-bg border ...">` after the header/image, starting from the resources block) with this structure:

```svelte
{#each containers as c}
  {@const status = containerStatus(c.name)}
  <div class="bg-bg border border-border rounded-lg p-3">
    <!-- Header: always visible -->
    <div class="flex items-center justify-between mb-1">
      <span class="text-sm font-medium">{c.name}</span>
      <div class="flex items-center gap-1.5">
        {#if status?.restartCount > 0}
          <span class="text-xs px-2 py-0.5 rounded-full bg-yellow-500/15 text-yellow-600 dark:text-yellow-400">
            {status.restartCount} restart{status.restartCount !== 1 ? 's' : ''}
          </span>
        {/if}
        <StatusBadge status={!!status?.ready} mode="pill">{stateLabel(status)}</StatusBadge>
      </div>
    </div>
    <p class="text-xs font-mono text-muted break-all mb-3">{c.image}</p>

    <!-- Accordion sections -->
    <div class="flex flex-col gap-0.5">
      <!-- Resources -->
      {#if c.resources?.requests || c.resources?.limits}
        <div>
          <button
            onclick={() => toggleSection(c.name, 'resources')}
            class="flex items-center gap-1 w-full text-left py-1.5 text-xs font-semibold text-muted uppercase tracking-wide hover:text-fg transition-colors"
          >
            {isSectionOpen(c.name, 'resources') ? '▾' : '▸'} Resources
          </button>
          {#if isSectionOpen(c.name, 'resources')}
            <div class="pl-4 pb-2">
              <div class="flex flex-wrap gap-2">
                {#if c.resources?.requests?.cpu || c.resources?.limits?.cpu}
                  <div class="flex items-center gap-1.5 text-xs bg-surface border border-border rounded px-2 py-1">
                    <span class="text-muted">CPU</span>
                    <span class="font-mono">{c.resources?.requests?.cpu ?? '—'}</span>
                    <span class="text-muted">/</span>
                    <span class="font-mono">{c.resources?.limits?.cpu ?? '—'}</span>
                  </div>
                {/if}
                {#if c.resources?.requests?.memory || c.resources?.limits?.memory}
                  <div class="flex items-center gap-1.5 text-xs bg-surface border border-border rounded px-2 py-1">
                    <span class="text-muted">Mem</span>
                    <span class="font-mono">{c.resources?.requests?.memory ?? '—'}</span>
                    <span class="text-muted">/</span>
                    <span class="font-mono">{c.resources?.limits?.memory ?? '—'}</span>
                  </div>
                {/if}
                {#if c.resources?.requests?.['ephemeral-storage'] || c.resources?.limits?.['ephemeral-storage']}
                  <div class="flex items-center gap-1.5 text-xs bg-surface border border-border rounded px-2 py-1">
                    <span class="text-muted">Disk</span>
                    <span class="font-mono">{c.resources?.requests?.['ephemeral-storage'] ?? '—'}</span>
                    <span class="text-muted">/</span>
                    <span class="font-mono">{c.resources?.limits?.['ephemeral-storage'] ?? '—'}</span>
                  </div>
                {/if}
              </div>
              <div class="text-[10px] text-muted mt-1">req / limit</div>
            </div>
          {/if}
        </div>
      {/if}

      <!-- Ports -->
      {#if c.ports?.length}
        <div>
          <button
            onclick={() => toggleSection(c.name, 'ports')}
            class="flex items-center gap-1 w-full text-left py-1.5 text-xs font-semibold text-muted uppercase tracking-wide hover:text-fg transition-colors"
          >
            {isSectionOpen(c.name, 'ports') ? '▾' : '▸'} Ports ({c.ports.length})
          </button>
          {#if isSectionOpen(c.name, 'ports')}
            <div class="pl-4 pb-2 flex flex-wrap gap-1">
              {#each c.ports as p}
                <PortButton port={p.containerPort} protocol={p.protocol ?? 'TCP'} onclick={() => pfPort = p.containerPort} />
              {/each}
            </div>
          {/if}
        </div>
      {/if}

      <!-- Environment -->
      {#if c.env?.length}
        <div>
          <button
            onclick={() => toggleSection(c.name, 'env')}
            class="flex items-center gap-1 w-full text-left py-1.5 text-xs font-semibold text-muted uppercase tracking-wide hover:text-fg transition-colors"
          >
            {isSectionOpen(c.name, 'env') ? '▾' : '▸'} Environment ({c.env.length})
          </button>
          {#if isSectionOpen(c.name, 'env')}
            <div class="pl-4 pb-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
              {#each c.env as e}
                <span class="text-xs font-mono text-accent">{e.name}</span>
                <span class="text-xs font-mono text-muted truncate">
                  {e.value ?? (e.valueFrom ? '(from secret/configmap)' : '—')}
                </span>
              {/each}
            </div>
          {/if}
        </div>
      {/if}

      <!-- Mounts -->
      {#if c.volumeMounts?.length}
        <div>
          <button
            onclick={() => toggleSection(c.name, 'mounts')}
            class="flex items-center gap-1 w-full text-left py-1.5 text-xs font-semibold text-muted uppercase tracking-wide hover:text-fg transition-colors"
          >
            {isSectionOpen(c.name, 'mounts') ? '▾' : '▸'} Mounts ({c.volumeMounts.length})
          </button>
          {#if isSectionOpen(c.name, 'mounts')}
            <div class="pl-4 pb-2 flex flex-col gap-1">
              {#each c.volumeMounts as m}
                <div class="flex items-center gap-2 text-xs">
                  <span class="font-mono text-accent">{m.mountPath}</span>
                  {#if m.name}
                    <span class="text-muted">← {m.name}</span>
                  {/if}
                  {#if m.subPath}
                    <span class="font-mono text-muted">/{m.subPath}</span>
                  {/if}
                  {#if m.readOnly}
                    <span class="px-1.5 py-0.5 rounded bg-surface border border-border text-muted text-[10px]">RO</span>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
{/each}
```

- [ ] **Step 3: Verify it builds**

Run: `cd frontend && pnpm check`
Expected: No type errors

- [ ] **Step 4: Commit**

```bash
jj desc -m "feat(overview): restructure container cards into accordion sections (resources, ports, env, mounts)"
jj new
```
