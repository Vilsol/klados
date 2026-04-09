<script lang="ts">
  import type { Component } from 'svelte'
  import type { DescriptorDef } from '$lib/registry/index'
  import { slotRegistry } from '$lib/plugins/slots.svelte.js'
  import { loadPluginComponent } from '$lib/plugins/loader.js'
  import { createPluginContext } from '$lib/plugins/context.js'
  import { clusterStore } from '$lib/stores/cluster.svelte.js'
  import { streamingStore } from '$lib/stores/streaming.svelte.js'
  import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import * as SchemaService from '../../../bindings/github.com/Vilsol/klados/internal/services/schemaservice.js'
  import { notificationStore } from '$lib/stores/notification.svelte.js'
  import { unwrapError } from '$lib/utils/async.js'
  import type { ControllerRef } from '$lib/utils/relationships'

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
  import NodeDrainTab from './panels/NodeDrainTab.svelte'
  import RulesPanel from './panels/RulesPanel.svelte'
  import BindingPanel from './panels/BindingPanel.svelte'
  import ServiceAccountPanel from './panels/ServiceAccountPanel.svelte'
  import StorageClassParametersPanel from './panels/StorageClassParametersPanel.svelte'
  import CSICapabilitiesPanel from './panels/CSICapabilitiesPanel.svelte'
  import CRDPanel from './panels/CRDPanel.svelte'
  import CRDSchemaPanel from './panels/CRDSchemaPanel.svelte'
  import ResourceQuotaPanel from './panels/ResourceQuotaPanel.svelte'
  import LimitRangePanel from './panels/LimitRangePanel.svelte'
  import PDBPanel from './panels/PDBPanel.svelte'
  import EndpointSlicePanel from './panels/EndpointSlicePanel.svelte'
  import NetworkPolicyPanel from './panels/NetworkPolicyPanel.svelte'
  import HPAPanel from './panels/HPAPanel.svelte'
  import WebhookConfigPanel from './panels/WebhookConfigPanel.svelte'
  import ActionsToolbar from './panels/ActionsToolbar.svelte'
  import MetricsTab from './charts/MetricsTab.svelte'
  import { YAMLEditor } from '@klados/ui'

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
    ['drain', NodeDrainTab as PanelComponent],
    ['rules', RulesPanel as PanelComponent],
    ['binding', BindingPanel as PanelComponent],
    ['serviceaccount', ServiceAccountPanel as PanelComponent],
    ['sc-parameters', StorageClassParametersPanel as PanelComponent],
    ['csi-capabilities', CSICapabilitiesPanel as PanelComponent],
    ['crd', CRDPanel as PanelComponent],
    ['crd-schema', CRDSchemaPanel as PanelComponent],
    ['metrics', MetricsTab as PanelComponent],
    ['netpol', NetworkPolicyPanel as PanelComponent],
    ['endpointslice', EndpointSlicePanel as PanelComponent],
    ['resourcequota', ResourceQuotaPanel as PanelComponent],
    ['limitrange', LimitRangePanel as PanelComponent],
    ['hpa', HPAPanel as PanelComponent],
    ['pdb', PDBPanel as PanelComponent],
    ['webhooks', WebhookConfigPanel as PanelComponent],
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
    drain: 'Drain',
    rules: 'Rules',
    binding: 'Binding',
    serviceaccount: 'Token & Secrets',
    'sc-parameters': 'Parameters',
    'csi-capabilities': 'Capabilities',
    crd: 'CRD',
    'crd-schema': 'Schema',
    metrics: 'Metrics',
    netpol: 'Rules',
    endpointslice: 'Addresses',
    resourcequota: 'Usage',
    limitrange: 'Limits',
    hpa: 'Scaling',
    pdb: 'Budget',
    webhooks: 'Webhooks',
  }

  let {
    obj = $bindable(),
    descriptor,
    ctxName,
    gvr,
    namespace,
    name,
    onrefresh,
    onupdate,
    onopenowner,
  }: {
    obj: Record<string, any>
    descriptor: DescriptorDef
    ctxName: string
    gvr: string
    namespace: string
    name: string
    onrefresh: () => void
    onupdate?: (updated: Record<string, any>) => void
    onopenowner?: (ref: ControllerRef, namespace: string) => void
  } = $props()

  const foldedIntoOverview = new Set(['labels', 'containers'])
  const visiblePanels = $derived(descriptor.detailPanels.filter((p) => panelComponents.has(p) && !foldedIntoOverview.has(p)))
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
    const ns = clusterStore.getSelectedNamespaces(ctxName)[0] ?? namespace
    const manifest = {
      schemaVersion: 1 as const,
      name: tab.pluginName,
      version: '',
      displayName: '',
      minHostVersion: '',
      permissions: {
        resources: tab.perms.resources?.map((p) => ({
          group: p.group,
          version: p.version,
          resource: p.resource,
          verbs: p.verbs as any,
        })),
        logs: tab.perms.logs || undefined,
        exec: tab.perms.exec || undefined,
        storage: tab.perms.storage || undefined,
        events: tab.perms.events || undefined,
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
          {#await loadPluginComponent(pt.pluginName, pt.component, basePluginURL) then Cmp}
            {#if Cmp}
              {@const ptCtx = makePluginCtx(pt)}
              <Cmp resource={obj} ctx={ptCtx} />
            {:else}
              <div class="flex items-center justify-center h-full text-sm text-muted">
                Plugin component failed to load
              </div>
            {/if}
          {/await}
        {:else}
          <div class="h-full bg-surface animate-pulse rounded"></div>
        {/if}
      {/if}
    {/each}
    {#each visiblePanels as panel}
      {#if activePanel === panel}
        {@const PanelCmp = panelComponents.get(panel)!}
        {#if panel === 'overview'}
          <PanelCmp obj={obj} onupdate={(updated: Record<string, any>) => { obj = updated; onupdate?.(updated) }} {descriptor} {gvr} {ctxName} {namespace} {name} {onopenowner} />
        {:else if panel === 'yaml'}
          {#key uid}
            <PanelCmp
              obj={obj}
              onupdate={(updated: Record<string, any>) => { obj = updated; onupdate?.(updated) }}
              {ctxName} {gvr} {namespace} {name}
              kind={descriptor.kind ?? ''}
              {onrefresh}
              onSave={(ctx: string, g: string, ns: string, parsed: Record<string, any>) => ResourceService.UpdateResource(ctx, g, ns, parsed)}
              onGetResource={(ctx: string, g: string, ns: string, n: string) => ResourceService.GetResource(ctx, g, ns, n)}
              onGetSchema={(ctx: string, g: string, k: string) => SchemaService.GetSchema(ctx, g, k)}
              onNotify={(msg: string, type: 'info' | 'success' | 'error') => {
                if (type === 'success') notificationStore.success(msg)
                else if (type === 'error') notificationStore.error(unwrapError(msg))
                else notificationStore.push(msg, type)
              }}
            />
          {/key}
        {:else if panel === 'events'}
          <PanelCmp {ctxName} {namespace} {uid} />
        {:else if panel === 'labels'}
          <div class="overflow-auto h-full">
            <PanelCmp obj={obj} onupdate={(updated: Record<string, any>) => { obj = updated; onupdate?.(updated) }} {ctxName} {gvr} {namespace} {name} />
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
        {:else if panel === 'ingress' || panel === 'configmap' || panel === 'secret' || panel === 'node' || panel === 'rules'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} />
          </div>
        {:else if panel === 'serviceaccount' || panel === 'binding'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} {ctxName} />
          </div>
        {:else if panel === 'sc-parameters'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} />
          </div>
        {:else if panel === 'csi-capabilities'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} {ctxName} />
          </div>
        {:else if panel === 'crd'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} {ctxName} />
          </div>
        {:else if panel === 'crd-schema'}
          <div class="h-full">
            <PanelCmp {obj} />
          </div>
        {:else if panel === 'metrics'}
          <PanelCmp {obj} {ctxName} {gvr} {namespace} {name} />
        {:else if panel === 'resourcequota' || panel === 'limitrange' || panel === 'pdb'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} />
          </div>
        {:else if panel === 'endpointslice'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} {ctxName} />
          </div>
        {:else if panel === 'netpol' || panel === 'webhooks'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} />
          </div>
        {:else if panel === 'hpa'}
          <div class="overflow-auto h-full">
            <PanelCmp {obj} {ctxName} />
          </div>
        {/if}
      {/if}
    {/each}
  </div>
</div>
