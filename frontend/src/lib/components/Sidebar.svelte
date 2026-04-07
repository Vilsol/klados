<script lang="ts">
  import { PanelLeftClose, PanelLeft, ChevronRight, Plus, X, Circle, Puzzle } from 'lucide-svelte'
  import { sessionStore } from '$lib/stores/session.svelte'
  import { clusterStore } from '$lib/stores/cluster.svelte'
  import { Events } from '@wailsio/runtime'
  import { push } from 'svelte-spa-router'
  import { onDestroy } from 'svelte'
  import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import * as PortForwardService from '../../../bindings/github.com/Vilsol/klados/internal/services/portforwardservice.js'
  import * as PluginService from '../../../bindings/github.com/Vilsol/klados/internal/services/pluginservice.js'
  import { notificationStore } from '$lib/stores/notification.svelte.js'
  import { unwrapError } from '$lib/utils/async.js'
  import PortForwardDialog from './PortForwardDialog.svelte'
  import { descriptorRegistry } from '$lib/registry/index'
  import { registryLoaded } from '$lib/registry/loaded.svelte'
  import { buildCRDTree } from '$lib/utils/crdTree'
  import CRDTreeNode from './CRDTreeNode.svelte'

  interface APIResource {
    gvr: string
    kind: string
    namespaced: boolean
  }

  const gvrGroups: Record<string, string[]> = {
    Workloads: [
      'core.v1.pods',
      'apps.v1.deployments',
      'apps.v1.statefulsets',
      'apps.v1.daemonsets',
      'apps.v1.replicasets',
      'batch.v1.jobs',
      'batch.v1.cronjobs',
    ],
    Networking: [
      'core.v1.services',
      'networking.k8s.io.v1.ingresses',
    ],
    Config: ['core.v1.configmaps', 'core.v1.secrets'],
    Storage: [
      'core.v1.persistentvolumeclaims',
      'core.v1.persistentvolumes',
      'storage.k8s.io.v1.storageclasses',
      'storage.k8s.io.v1.csidrivers',
    ],
    Cluster: ['core.v1.nodes', 'apiextensions.k8s.io.v1.customresourcedefinitions'],
    RBAC: [
      'core.v1.serviceaccounts',
      'rbac.authorization.k8s.io.v1.roles',
      'rbac.authorization.k8s.io.v1.clusterroles',
      'rbac.authorization.k8s.io.v1.rolebindings',
      'rbac.authorization.k8s.io.v1.clusterrolebindings',
    ],
  }

  const kindByGvr: Record<string, string> = {
    'core.v1.pods': 'Pods',
    'apps.v1.deployments': 'Deployments',
    'apps.v1.statefulsets': 'StatefulSets',
    'apps.v1.daemonsets': 'DaemonSets',
    'apps.v1.replicasets': 'ReplicaSets',
    'batch.v1.jobs': 'Jobs',
    'batch.v1.cronjobs': 'CronJobs',
    'core.v1.services': 'Services',
    'networking.k8s.io.v1.ingresses': 'Ingresses',
    'core.v1.configmaps': 'ConfigMaps',
    'core.v1.secrets': 'Secrets',
    'core.v1.persistentvolumeclaims': 'PersistentVolumeClaims',
    'core.v1.persistentvolumes': 'PersistentVolumes',
    'storage.k8s.io.v1.storageclasses': 'StorageClasses',
    'storage.k8s.io.v1.csidrivers': 'CSI Drivers',
    'core.v1.nodes': 'Nodes',
    'apiextensions.k8s.io.v1.customresourcedefinitions': 'CRDs',
    'core.v1.serviceaccounts': 'ServiceAccounts',
    'rbac.authorization.k8s.io.v1.roles': 'Roles',
    'rbac.authorization.k8s.io.v1.clusterroles': 'ClusterRoles',
    'rbac.authorization.k8s.io.v1.rolebindings': 'RoleBindings',
    'rbac.authorization.k8s.io.v1.clusterrolebindings': 'ClusterRoleBindings',
  }

  let discoveredGVRs = $state<Set<string>>(new Set())
  let expandedGroups = $state<Record<string, boolean>>({ Workloads: true })
  let customResources = $state<APIResource[]>([])

  const crdTree = $derived((() => {
    const kindMap = new Map(customResources.map((r) => [r.gvr, r.kind]))
    return buildCRDTree(
      customResources.map((r) => r.gvr),
      (gvr) => kindMap.get(gvr) || gvr.split('.').at(-1)!,
    )
  })())

  let expandedNodes = $state(new Set<string>())
  $effect(() => {
    clusterStore.activeContext // track
    expandedNodes = new Set()
  })

  function toggleExpand(fullSuffix: string) {
    const next = new Set(expandedNodes)
    if (next.has(fullSuffix)) next.delete(fullSuffix)
    else next.add(fullSuffix)
    expandedNodes = next
  }

  interface ForwardSpec {
    id: string
    contextName: string
    targetName: string
    localPort: number
    remotePort: number
    status: string
    podName: string
    error: string
  }

  interface PluginSidebarEntry {
    category: string
    label: string
    gvr: string
    icon: string
    plugin: string
  }

  let forwards = $state<ForwardSpec[]>([])
  let showPortForwardDialog = $state(false)
  let pluginEntries = $state<PluginSidebarEntry[]>([])

  async function loadPluginEntries() {
    try {
      const result = await PluginService.GetPluginSidebarEntries()
      pluginEntries = (result ?? []) as PluginSidebarEntry[]
    } catch {
      // ignore
    }
  }

  async function loadForwards() {
    if (!ctx) return
    try {
      const result = await PortForwardService.ListForwards(ctx)
      forwards = (result ?? []) as ForwardSpec[]
    } catch {
      // ignore
    }
  }

  const ctx = $derived(clusterStore.activeContext)

  function handleDiscovery(resources: APIResource[]) {
    const gvrs = new Set(resources.map((r) => r.gvr))
    discoveredGVRs = gvrs

    const knownGVRs = new Set(Object.values(gvrGroups).flat())
    // Show resources not in any builtin group and not purely internal/meta resources
    const internalPrefixes = ['core.v1.', 'rbac.authorization.k8s.io.', 'authorization.k8s.io.', 'authentication.k8s.io.', 'events.k8s.io.']
    customResources = resources.filter(
      (r) => !knownGVRs.has(r.gvr) && !internalPrefixes.some((p) => r.gvr.startsWith(p)),
    )
  }

  let unsub: (() => void) | null = null
  let unsubPF: (() => void) | null = null
  let unsubPlugins: (() => void) | null = null

  $effect(() => {
    if (unsub) { unsub(); unsub = null }
    if (unsubPF) { unsubPF(); unsubPF = null }
    if (unsubPlugins) { unsubPlugins(); unsubPlugins = null }
    if (ctx) {
      unsub = Events.On(`discovery:${ctx}:resources`, (wailsEvent: any) => {
        handleDiscovery((wailsEvent.data ?? wailsEvent) as APIResource[])
      })
      ResourceService.ListAPIResources(ctx)
        .then((r) => { if (r?.length) handleDiscovery(r as APIResource[]) })
        .catch(() => {})

      loadForwards()
      unsubPF = Events.On(`portforward:${ctx}:updated`, () => { loadForwards() })
    }

    loadPluginEntries()
    unsubPlugins = Events.On('plugins:loaded', () => { loadPluginEntries() })
  })

  onDestroy(() => {
    unsub?.()
    unsubPF?.()
    unsubPlugins?.()
  })

  async function stopForward(id: string) {
    try {
      await PortForwardService.StopForward(id)
      await loadForwards()
      notificationStore.success('Port forward stopped')
    } catch (e: any) {
      notificationStore.error('Failed to stop port forward', unwrapError(e))
    }
  }

  function navigate(gvr: string) {
    if (!ctx) return
    push(`/c/${ctx}/${gvr}`)
  }

  function toggleGroup(name: string) {
    expandedGroups[name] = !expandedGroups[name]
  }
</script>

<aside
  class="border-r border-border bg-surface shrink-0 overflow-hidden transition-all duration-200"
  class:w-60={!sessionStore.sidebarCollapsed}
  class:w-0={sessionStore.sidebarCollapsed}
>
  <div class="w-60 h-full flex flex-col">
    <div class="flex items-center justify-between px-3 py-2 border-b border-border">
      <span class="text-xs font-semibold uppercase tracking-wider text-muted">Resources</span>
      <button onclick={() => sessionStore.toggleSidebar()} class="p-1 rounded hover:bg-surface-hover transition-colors" aria-label="Collapse sidebar">
        <PanelLeftClose size={14} />
      </button>
    </div>

    <nav class="flex-1 overflow-y-auto py-1">
      {#if ctx}
        <button
          onclick={() => push(`/c/${ctx}`)}
          class="w-full text-left px-3 py-1.5 text-sm flex items-center gap-2 rounded-none hover:bg-surface-hover transition-colors border-b border-border mb-1 text-fg"
        >
          Overview
        </button>
      {/if}
      {#each Object.entries(gvrGroups) as [groupName, gvrs]}
        {@const available = gvrs.filter((g) => !ctx || discoveredGVRs.size === 0 || discoveredGVRs.has(g))}
        {#if available.length > 0}
          <div>
            <button
              onclick={() => toggleGroup(groupName)}
              class="w-full flex items-center gap-1 px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-muted hover:bg-surface-hover transition-colors"
            >
              <ChevronRight
                size={12}
                class="transition-transform {expandedGroups[groupName] ? 'rotate-90' : ''}"
              />
              {groupName}
            </button>
            {#if expandedGroups[groupName]}
              <div class="ml-4">
                {#each available as gvr}
                  <button
                    onclick={() => navigate(gvr)}
                    class="w-full text-left px-3 py-1 text-sm hover:bg-surface-hover transition-colors rounded-sm"
                  >
                    {kindByGvr[gvr] ?? gvr.split('.').at(-1)}
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
      {/each}

      {#if crdTree.length > 0}
        <div>
          <button
            onclick={() => toggleGroup('Custom Resources')}
            class="w-full flex items-center gap-1 px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-muted hover:bg-surface-hover transition-colors"
          >
            <ChevronRight
              size={12}
              class="transition-transform {expandedGroups['Custom Resources'] ? 'rotate-90' : ''}"
            />
            Custom Resources
          </button>
          {#if expandedGroups['Custom Resources']}
            <div class="ml-4">
              {#each crdTree as node}
                <CRDTreeNode {node} expanded={expandedNodes} onToggle={toggleExpand} ctxName={ctx ?? ''} />
              {/each}
            </div>
          {/if}
        </div>
      {/if}
      {#if pluginEntries.length > 0}
        {@const pluginCategories = [...new Set(pluginEntries.map((e) => e.category))]}
        {#each pluginCategories as category}
          <div>
            <button
              onclick={() => toggleGroup(`plugin:${category}`)}
              class="w-full flex items-center gap-1 px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-muted hover:bg-surface-hover transition-colors"
            >
              <ChevronRight
                size={12}
                class="transition-transform {expandedGroups[`plugin:${category}`] ? 'rotate-90' : ''}"
              />
              {category}
            </button>
            {#if expandedGroups[`plugin:${category}`]}
              <div class="ml-4">
                {#each pluginEntries.filter((e) => e.category === category) as entry}
                  <button
                    onclick={() => navigate(entry.gvr)}
                    class="w-full text-left px-3 py-1 text-sm hover:bg-surface-hover transition-colors rounded-sm"
                  >
                    {entry.label}
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      {/if}

      {#if ctx}
        <div class="border-t border-border mt-1 pt-1">
          <button
            onclick={() => push(`/c/${ctx}/events`)}
            class="w-full flex items-center gap-2 px-3 py-1.5 text-xs font-medium text-muted hover:bg-surface-hover transition-colors"
          >
            Event Stream
          </button>
        </div>
      {/if}
    </nav>

    <!-- Plugins link -->
    <div class="border-t border-border">
      <button
        onclick={() => push('/plugins')}
        class="w-full flex items-center gap-2 px-3 py-1.5 text-xs font-medium text-muted hover:bg-surface-hover transition-colors"
      >
        <Puzzle size={12} />
        Plugins
      </button>
    </div>

    <!-- Port Forwards -->
    <div class="border-t border-border">
      <div class="flex items-center justify-between px-3 py-2">
        <span class="text-xs font-semibold uppercase tracking-wider text-muted">Port Forwards</span>
        {#if ctx}
          <button
            onclick={() => showPortForwardDialog = true}
            class="p-1 rounded hover:bg-surface-hover transition-colors text-muted hover:text-fg"
            title="New port forward"
            aria-label="New port forward"
          >
            <Plus size={12} />
          </button>
        {/if}
      </div>

      {#if forwards.length > 0}
        <div class="flex flex-col pb-1">
          {#each forwards as fwd}
            <div class="flex items-center gap-2 px-3 py-1 group">
              <Circle
                size={8}
                class="shrink-0 fill-current
                  {fwd.status === 'active' ? 'text-green-500' :
                   fwd.status === 'reconnecting' ? 'text-yellow-500' :
                   'text-red-500'}"
              />
              <div class="flex-1 min-w-0">
                <div class="text-xs font-mono truncate">{fwd.localPort}→{fwd.targetName}</div>
                {#if fwd.podName && fwd.podName !== fwd.targetName}
                  <div class="text-xs text-muted truncate">{fwd.podName}</div>
                {/if}
              </div>
              <button
                onclick={() => stopForward(fwd.id)}
                class="p-0.5 rounded text-muted hover:text-fg opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                title="Stop"
                aria-label="Stop port forward {fwd.id}"
              >
                <X size={10} />
              </button>
            </div>
          {/each}
        </div>
      {:else if ctx}
        <p class="text-xs text-muted px-3 pb-2">No active forwards</p>
      {/if}
    </div>
  </div>
</aside>

{#if showPortForwardDialog}
  <PortForwardDialog onclose={() => { showPortForwardDialog = false; loadForwards() }} />
{/if}

{#if sessionStore.sidebarCollapsed}
  <button
    onclick={() => sessionStore.toggleSidebar()}
    class="absolute left-0 top-16 p-1.5 bg-surface border border-border border-l-0 rounded-r hover:bg-surface-hover transition-colors z-10"
    aria-label="Expand sidebar"
  >
    <PanelLeft size={14} />
  </button>
{/if}
