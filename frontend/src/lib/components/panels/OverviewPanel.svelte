<script lang="ts">
  import { evalExpr } from '$lib/registry/index'
  import type { DescriptorDef } from '$lib/registry/index'
  import { formatAge } from '$lib/utils/age'
  import { getControllerRef, type ControllerRef } from '$lib/utils/relationships'
  import { clusterStore } from '$lib/stores/cluster.svelte'
  import { toggleSet } from '$lib/utils/collections'
  import { SectionHeader, KeyValueBadge, EmptyState, StatusBadge, KeyValuePairEditor } from '@klados/ui'
  import { slotRegistry } from '$lib/plugins/slots.svelte.js'
  import { loadPluginComponent } from '$lib/plugins/loader.js'
  import { streamingStore } from '$lib/stores/streaming.svelte.js'
  import * as ResourceService from '../../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import { notificationStore } from '$lib/stores/notification.svelte'
  import PortForwardDialog from '$lib/components/PortForwardDialog.svelte'
  import PortButton from '$lib/components/PortButton.svelte'

  let {
    obj,
    onupdate,
    descriptor,
    gvr = '',
    ctxName = '',
    namespace = '',
    name = '',
    onopenowner,
  }: {
    obj: Record<string, any>
    onupdate?: (updated: Record<string, any>) => void
    descriptor: DescriptorDef
    gvr?: string
    ctxName?: string
    namespace?: string
    name?: string
    onopenowner?: (ref: ControllerRef) => void
  } = $props()

  const basePluginURL = $derived(
    streamingStore.config
      ? `http://127.0.0.1:${streamingStore.config.port}/${streamingStore.config.token}/plugins`
      : null
  )

  function renderValue(expr: string, renderType: string): string {
    const raw = evalExpr(expr, obj)
    if (renderType === 'age' && raw) {
      return formatAge(String(raw))
    }
    if (raw === null || raw === undefined) return '—'
    return String(raw)
  }

  // Labels/annotations state
  const hasLabelsPanel = $derived(descriptor.detailPanels.includes('labels'))
  let editingLabels = $state(false)
  let saving = $state(false)
  let editLabels = $state<[string, string][]>([])
  let editAnnotations = $state<[string, string][]>([])

  function startEdit() {
    editLabels = Object.entries(obj.metadata?.labels ?? {}).map(([k, v]) => [k, String(v)])
    editAnnotations = Object.entries(obj.metadata?.annotations ?? {}).map(([k, v]) => [k, String(v)])
    editingLabels = true
  }

  function cancelEdit() {
    editingLabels = false
  }

  async function saveLabels() {
    saving = true
    try {
      const updated = JSON.parse(JSON.stringify(obj))
      updated.metadata.labels = Object.fromEntries(editLabels.filter(([k]) => k.trim()))
      updated.metadata.annotations = Object.fromEntries(editAnnotations.filter(([k]) => k.trim()))
      const result = await ResourceService.UpdateResource(ctxName, gvr, namespace, updated)
      if (result) onupdate?.(result)
      editingLabels = false
      notificationStore.push('Labels and annotations saved.', 'success')
    } catch (e: any) {
      notificationStore.push(e?.message ?? 'Save failed', 'error')
    } finally {
      saving = false
    }
  }

  const controllerRef = $derived(getControllerRef(obj))
  const labels = $derived(Object.entries(obj.metadata?.labels ?? {}))
  const annotations = $derived(Object.entries(obj.metadata?.annotations ?? {}))

  // Containers state
  const hasContainersPanel = $derived(descriptor.detailPanels.includes('containers'))
  const containers = $derived<any[]>(obj.spec?.containers ?? [])
  const initContainers = $derived<any[]>(obj.spec?.initContainers ?? [])
  const conditions = $derived<any[]>(obj.status?.conditions ?? [])

  let pfPort = $state<number | null>(null)

  function containerStatus(cname: string): any {
    return (obj.status?.containerStatuses ?? []).find((s: any) => s.name === cname)
  }

  function initContainerStatus(cname: string): any {
    return (obj.status?.initContainerStatuses ?? []).find((s: any) => s.name === cname)
  }

  function stateLabel(status: any): string {
    if (!status) return 'Unknown'
    if (status.state?.running) return 'Running'
    if (status.state?.waiting) return `Waiting: ${status.state.waiting.reason ?? ''}`
    if (status.state?.terminated) return `Terminated: ${status.state.terminated.reason ?? ''}`
    return 'Unknown'
  }

  let expandedEnv = $state<Set<string>>(new Set())
  let expandedMounts = $state<Set<string>>(new Set())


  let showInitContainers = $state(false)
</script>

<div class="overflow-auto h-full p-4 flex flex-col gap-4">
  <!-- Overview fields card -->
  <section class="bg-surface border border-border rounded-lg p-4">
    <SectionHeader class="mb-3">Details</SectionHeader>
    <div class="grid grid-cols-3 gap-x-6 gap-y-3">
      {#each descriptor.overviewFields as field}
        <div class="min-w-0">
          <div class="text-xs text-muted mb-0.5">{field.label}</div>
          {#if field.renderType === 'badge'}
            <span class="text-xs font-mono bg-bg border border-border rounded px-2 py-0.5 inline-block">
              {renderValue(field.expr, field.renderType)}
            </span>
          {:else}
            <div class="text-xs font-mono truncate">{renderValue(field.expr, field.renderType)}</div>
          {/if}
        </div>
      {/each}
      {#if controllerRef}
        <div class="min-w-0">
          <div class="text-xs text-muted mb-0.5">Controlled By</div>
          {#if onopenowner && clusterStore.resolveOwnerGVR(controllerRef.apiVersion, controllerRef.kind)}
            <button
              class="text-xs font-mono text-accent hover:underline"
              onclick={() => onopenowner!(controllerRef)}
            >{controllerRef.kind}/{controllerRef.name}</button>
          {:else}
            <div class="text-xs font-mono truncate">{controllerRef.kind}/{controllerRef.name}</div>
          {/if}
        </div>
      {/if}
    </div>
    {#if basePluginURL}
      {#each slotRegistry.getOverviewFields(gvr) as field (field.id)}
        {#await loadPluginComponent(field.pluginName, field.component, basePluginURL) then Cmp}
          {#if Cmp}
            <Cmp resource={obj} />
          {/if}
        {/await}
      {/each}
    {/if}
  </section>

  <!-- Labels & Annotations card -->
  {#if hasLabelsPanel}
    <section class="bg-surface border border-border rounded-lg p-4">
      <div class="flex items-center justify-between mb-3">
        <SectionHeader class="">Labels & Annotations</SectionHeader>
        {#if !editingLabels}
          <button
            onclick={startEdit}
            class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
          >Edit</button>
        {:else}
          <div class="flex gap-2">
            <button
              onclick={cancelEdit}
              class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
            >Cancel</button>
            <button
              onclick={saveLabels}
              disabled={saving}
              class="text-xs px-2.5 py-1 rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity disabled:opacity-50"
            >{saving ? 'Saving…' : 'Save'}</button>
          </div>
        {/if}
      </div>

      {#if !editingLabels}
        <div class="mb-3">
          <h4 class="text-xs font-medium mb-1.5">Labels</h4>
          <KeyValueBadge entries={labels} />
        </div>

        <div>
          <h4 class="text-xs font-medium mb-1.5">Annotations</h4>
          {#if annotations.length === 0}
            <EmptyState />
          {:else}
            <div class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5">
              {#each annotations.sort(([a], [b]) => a.localeCompare(b)) as [k, v]}
                <span class="text-xs font-mono text-muted">{k}</span>
                <span class="text-xs font-mono break-all">{v}</span>
              {/each}
            </div>
          {/if}
        </div>
      {:else}
        <div class="mb-3">
          <h4 class="text-xs font-medium mb-1.5">Labels</h4>
          <KeyValuePairEditor bind:pairs={editLabels} addLabel="+ Add label" />
        </div>

        <div>
          <h4 class="text-xs font-medium mb-1.5">Annotations</h4>
          <KeyValuePairEditor bind:pairs={editAnnotations} addLabel="+ Add annotation" />
        </div>
      {/if}
    </section>
  {/if}

  <!-- Conditions card -->
  {#if hasContainersPanel && conditions.length > 0}
    <section class="bg-surface border border-border rounded-lg p-4">
      <SectionHeader class="mb-3">Conditions</SectionHeader>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
        {#each conditions as cond}
          <div class="flex items-center gap-2 px-3 py-2 rounded-md bg-bg border border-border" title={cond.message ?? ''}>
            <span class="w-2 h-2 rounded-full shrink-0
              {cond.status === 'True' ? 'bg-green-500' : 'bg-muted'}"></span>
            <span class="text-xs font-mono flex-1 truncate">{cond.type}</span>
            {#if cond.reason}
              <span class="text-xs text-muted truncate">{cond.reason}</span>
            {/if}
            {#if cond.lastTransitionTime}
              <span class="text-xs text-muted shrink-0 tabular-nums" title={new Date(cond.lastTransitionTime).toLocaleString()}>{formatAge(cond.lastTransitionTime)}</span>
            {/if}
          </div>
        {/each}
      </div>
    </section>
  {/if}

  <!-- Containers card -->
  {#if hasContainersPanel && containers.length > 0}
    <section class="bg-surface border border-border rounded-lg p-4">
      <SectionHeader class="mb-3">Containers</SectionHeader>
      <div class="flex flex-col gap-3">
        {#each containers as c}
          {@const status = containerStatus(c.name)}
          <div class="bg-bg border border-border rounded-lg p-3">
            <div class="flex items-center justify-between mb-2">
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
            <p class="text-xs font-mono text-muted break-all mb-2">{c.image}</p>

            {#if c.resources?.requests || c.resources?.limits}
              <div class="flex flex-wrap gap-2 mb-2">
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
              <div class="text-[10px] text-muted mb-2">req / limit</div>
            {/if}

            {#if c.ports?.length}
              <div class="flex flex-wrap gap-1 mt-1">
                {#each c.ports as p}
                  <PortButton port={p.containerPort} protocol={p.protocol ?? 'TCP'} onclick={() => pfPort = p.containerPort} />
                {/each}
              </div>
            {/if}

            {#if c.env?.length}
              <button
                onclick={() => expandedEnv = toggleSet(expandedEnv, c.name)}
                class="text-xs text-accent hover:underline mt-1"
              >
                {expandedEnv.has(c.name) ? '▾' : '▸'} {c.env.length} env var{c.env.length !== 1 ? 's' : ''}
              </button>
              {#if expandedEnv.has(c.name)}
                <div class="mt-1.5 grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 pl-3">
                  {#each c.env as e}
                    <span class="text-xs font-mono text-accent">{e.name}</span>
                    <span class="text-xs font-mono text-muted truncate">
                      {e.value ?? (e.valueFrom ? '(from secret/configmap)' : '—')}
                    </span>
                  {/each}
                </div>
              {/if}
            {/if}

            {#if c.volumeMounts?.length}
              <button
                onclick={() => expandedMounts = toggleSet(expandedMounts, c.name)}
                class="text-xs text-accent hover:underline mt-1"
              >
                {expandedMounts.has(c.name) ? '▾' : '▸'} {c.volumeMounts.length} mount{c.volumeMounts.length !== 1 ? 's' : ''}
              </button>
              {#if expandedMounts.has(c.name)}
                <div class="mt-1.5 flex flex-col gap-1 pl-3">
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
            {/if}
          </div>
        {/each}
      </div>
    </section>
  {/if}

  <!-- Init containers -->
  {#if hasContainersPanel && initContainers.length > 0}
    <section class="bg-surface border border-border rounded-lg p-4">
      <button
        onclick={() => showInitContainers = !showInitContainers}
        class="text-xs font-semibold text-muted uppercase tracking-wide flex items-center gap-1"
      >
        {showInitContainers ? '▾' : '▸'} Init Containers ({initContainers.length})
      </button>
      {#if showInitContainers}
        <div class="flex flex-col gap-2 mt-3">
          {#each initContainers as c}
            {@const status = initContainerStatus(c.name)}
            <div class="bg-bg border border-border rounded-lg p-3">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium">{c.name}</span>
                <span class="text-xs px-2 py-0.5 rounded-full bg-surface-hover text-muted">{stateLabel(status)}</span>
              </div>
              <p class="text-xs font-mono text-muted truncate mt-1">{c.image}</p>
            </div>
          {/each}
        </div>
      {/if}
    </section>
  {/if}
</div>

{#if pfPort !== null}
  <PortForwardDialog
    prefillContext={ctxName}
    prefillNamespace={obj.metadata?.namespace ?? ''}
    prefillTargetKind="pod"
    prefillTarget={obj.metadata?.name ?? ''}
    prefillGVR=""
    prefillRemotePort={pfPort}
    onclose={() => pfPort = null}
  />
{/if}
