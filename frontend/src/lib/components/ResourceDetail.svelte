<script lang="ts">
  import type { Component } from 'svelte'
  import type { DescriptorDef } from '$lib/registry/index'
  import { slotRegistry } from '$lib/plugins/slots.svelte.js'
  import { loadPluginComponent } from '$lib/plugins/loader.js'
  import { createPluginContext } from '$lib/plugins/context.js'
  import { clusterStore } from '$lib/stores/cluster.svelte.js'
  import { streamingStore } from '$lib/stores/streaming.svelte.js'
  import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'

  import OverviewPanel from './panels/OverviewPanel.svelte'
  import EventsPanel from './panels/EventsPanel.svelte'
  import LabelsAnnotationsPanel from './panels/LabelsAnnotationsPanel.svelte'
  import ContainersPanel from './panels/ContainersPanel.svelte'
  import DeploymentPanel from './panels/DeploymentPanel.svelte'
  import LogsPanel from './panels/LogsPanel.svelte'
  import TerminalPanel from './panels/TerminalPanel.svelte'
  import ServicePanel from './panels/ServicePanel.svelte'
  import IngressPanel from './panels/IngressPanel.svelte'
  import ConfigMapPanel from './panels/ConfigMapPanel.svelte'
  import SecretPanel from './panels/SecretPanel.svelte'
  import NodePanel from './panels/NodePanel.svelte'
  import ActionsToolbar from './panels/ActionsToolbar.svelte'
  import YAMLEditor from './YAMLEditor.svelte'

  type PanelComponent = Component<any>

  const panelComponents: Map<string, PanelComponent> = new Map([
    ['overview', OverviewPanel as PanelComponent],
    ['yaml', YAMLEditor as PanelComponent],
    ['events', EventsPanel as PanelComponent],
    ['labels', LabelsAnnotationsPanel as PanelComponent],
    ['containers', ContainersPanel as PanelComponent],
    ['deployment-detail', DeploymentPanel as PanelComponent],
    ['logs', LogsPanel as PanelComponent],
    ['terminal', TerminalPanel as PanelComponent],
    ['service', ServicePanel as PanelComponent],
    ['ingress', IngressPanel as PanelComponent],
    ['configmap', ConfigMapPanel as PanelComponent],
    ['secret', SecretPanel as PanelComponent],
    ['node', NodePanel as PanelComponent],
  ])

  const panelLabels: Record<string, string> = {
    overview: 'Overview',
    yaml: 'YAML',
    events: 'Events',
    labels: 'Labels',
    containers: 'Containers',
    'deployment-detail': 'Details',
    logs: 'Logs',
    terminal: 'Terminal',
    service: 'Endpoints',
    ingress: 'Rules',
    configmap: 'Data',
    secret: 'Data',
    node: 'Conditions',
  }

  let {
    obj = $bindable(),
    descriptor,
    ctxName,
    gvr,
    namespace,
    name,
    onrefresh,
  }: {
    obj: Record<string, any>
    descriptor: DescriptorDef
    ctxName: string
    gvr: string
    namespace: string
    name: string
    onrefresh: () => void
  } = $props()

  const visiblePanels = $derived(descriptor.detailPanels.filter((p) => panelComponents.has(p)))
  const pluginTabs = $derived(slotRegistry.getDetailTabs(gvr))
  let activePanel = $state('')
  $effect(() => {
    const allPanels = [...visiblePanels, ...pluginTabs.map((t) => t.id)]
    if (allPanels.length > 0 && !allPanels.includes(activePanel)) {
      activePanel = allPanels[0]
    }
  })

  const basePluginURL = $derived(
    streamingStore.config
      ? `http://127.0.0.1:${streamingStore.config.port}/${streamingStore.config.token}/plugins`
      : null
  )

  function makePluginCtx(tab: import('$lib/plugins/slots.svelte.js').RegisteredDetailTab) {
    const ns = clusterStore.selectedNamespaces[0] ?? namespace
    const manifest = {
      schemaVersion: 1 as const,
      name: tab.pluginName,
      version: '',
      displayName: '',
      minHostVersion: '',
      permissions: {
        resources: tab.resourcePerms.map((p) => ({
          group: p.group,
          version: p.version,
          resource: p.resource,
          verbs: p.verbs as any,
        })),
      },
    }
    return createPluginContext(manifest, {
      clusterName: ctxName,
      clusterVersion: '',
      namespace: ns,
      listResources: (g, n) => ResourceService.ListResources(ctxName, g, n ?? ''),
      getResource: (g, n, name) => ResourceService.GetResource(ctxName, g, n, name),
    })
  }

  const uid = $derived<string>(obj.metadata?.uid ?? '')
</script>

<div class="flex flex-col h-full overflow-hidden">
  <!-- Actions toolbar -->
  {#if descriptor.actions.length > 0}
    <ActionsToolbar
      {obj}
      {ctxName}
      {gvr}
      {namespace}
      {name}
      actions={descriptor.actions}
      {onrefresh}
    />
  {/if}

  <!-- Panel tab bar -->
  <div class="flex items-center border-b border-border bg-surface shrink-0 overflow-x-auto">
    {#each visiblePanels as panel}
      <button
        onclick={() => activePanel = panel}
        class="px-4 py-2 text-xs font-medium whitespace-nowrap transition-colors border-b-2
          {activePanel === panel
            ? 'border-accent text-accent'
            : 'border-transparent text-muted hover:text-fg hover:bg-surface-hover'}"
      >
        {panelLabels[panel] ?? panel}
      </button>
    {/each}
    {#each pluginTabs as pt}
      <button
        onclick={() => activePanel = pt.id}
        class="px-4 py-2 text-xs font-medium whitespace-nowrap transition-colors border-b-2
          {activePanel === pt.id
            ? 'border-accent text-accent'
            : 'border-transparent text-muted hover:text-fg hover:bg-surface-hover'}"
      >
        {pt.label}
      </button>
    {/each}
  </div>

  <!-- Active panel -->
  <div class="flex-1 overflow-hidden">
    {#each pluginTabs as pt}
      {#if activePanel === pt.id}
        {#if basePluginURL}
          {#await loadPluginComponent(pt.pluginName, pt.component, basePluginURL) then cmp}
            {#if cmp}
              {@const ptCtx = makePluginCtx(pt)}
              <svelte:component this={cmp} resource={obj} ctx={ptCtx} />
            {:else}
              <div class="flex items-center justify-center h-full text-sm text-muted">
                Plugin component failed to load
              </div>
            {/if}
          {/await}
        {:else}
          <div class="h-full bg-surface animate-pulse rounded" />
        {/if}
      {/if}
    {/each}
    {#each visiblePanels as panel}
      {#if activePanel === panel}
        {@const PanelCmp = panelComponents.get(panel)!}
        {#if panel === 'overview'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} {descriptor} {gvr} />
          </div>
        {:else if panel === 'yaml'}
          {#key uid}
            <PanelCmp bind:obj {ctxName} {gvr} {namespace} {name} kind={descriptor.kind ?? ''} onrefresh={onrefresh} />
          {/key}
        {:else if panel === 'events'}
          <PanelCmp {ctxName} {namespace} {uid} />
        {:else if panel === 'labels'}
          <div class="overflow-auto h-full">
            <PanelCmp bind:obj {ctxName} {gvr} {namespace} {name} />
          </div>
        {:else if panel === 'containers'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} {ctxName} />
          </div>
        {:else if panel === 'deployment-detail'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} />
          </div>
        {:else if panel === 'logs' || panel === 'terminal'}
          <PanelCmp {obj} {ctxName} namespace={namespace} {name} />
        {:else if panel === 'service'}
          <PanelCmp {obj} ctxName={ctxName} />
        {:else if panel === 'ingress' || panel === 'configmap' || panel === 'secret' || panel === 'node'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} />
          </div>
        {/if}
      {/if}
    {/each}
  </div>
</div>
