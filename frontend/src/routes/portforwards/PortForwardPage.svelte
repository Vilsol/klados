<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { Events } from '@wailsio/runtime'
  import { Plus } from 'lucide-svelte'
  import ResourceList from '$lib/components/ResourceList.svelte'
  import PortForwardDialog from '$lib/components/PortForwardDialog.svelte'
  import { descriptorRegistry, type DescriptorDef } from '$lib/registry/index'
  import { columnStore } from '$lib/stores/columns.svelte'
  import { clusterStore } from '$lib/stores/cluster.svelte'
  import { notificationStore } from '$lib/stores/notification.svelte'
  import { Browser } from '@wailsio/runtime'
  import * as PortForwardService from '../../../bindings/github.com/Vilsol/klados/internal/services/portforwardservice.js'
  import { SavedPortForward } from '../../../bindings/github.com/Vilsol/klados/internal/config/models.js'

  const PF_GVR = '_internal.v1.portforwards'

  const pfDescriptor: DescriptorDef = {
    group: '_internal',
    version: 'v1',
    resource: 'portforwards',
    kind: 'PortForward',
    gvr: PF_GVR,
    columns: [
      { name: 'Resource',    expr: 'resource',    renderType: 'text' },
      { name: 'Namespace',   expr: 'namespace',   renderType: 'text' },
      { name: 'Local Port',  expr: 'localPort',   renderType: 'text' },
      { name: 'Remote Port', expr: 'remotePort',  renderType: 'text' },
      { name: 'Status',      expr: 'status',      renderType: 'badge' },
      { name: 'Enabled',     expr: 'enabled',     renderType: 'text' },
    ],
    overviewFields: [],
    detailPanels: [],
    actions: [],
  }

  let { params = {} }: { params?: Record<string, string> } = $props()

  const ctxName = $derived(params.ctx ?? '')

  $effect(() => { if (ctxName) clusterStore.setActiveContext(ctxName) })

  let items = $state<Record<string, any>[]>([])
  let loading = $state(false)
  let dialogOpen = $state(false)

  async function refresh() {
    if (!ctxName) return
    loading = true
    try {
      const [saved, active] = await Promise.all([
        PortForwardService.ListSavedPortForwards(ctxName),
        PortForwardService.ListForwards(ctxName),
      ])
      const activeMap = new Map<string, any>()
      for (const f of active ?? []) {
        if (f?.id) activeMap.set(f.id, f)
      }
      items = (saved ?? []).map((s: any) => {
        const live = activeMap.get(s?.id)
        return {
          id: s?.id ?? '',
          resource: s?.resource ?? '',
          namespace: s?.namespace ?? '',
          localPort: s?.localPort ?? 0,
          remotePort: s?.remotePort ?? 0,
          enabled: s?.enabled ?? false,
          targetKind: s?.targetKind ?? '',
          targetName: s?.targetName ?? '',
          targetGVR: s?.targetGVR ?? '',
          status: live?.status ?? 'stopped',
          error: live?.error ?? '',
          podName: live?.podName ?? '',
          metadata: { name: s?.id ?? '', namespace: s?.namespace ?? '' },
        }
      })
    } finally {
      loading = false
    }
  }

  async function handleCreated(spec: any) {
    if (!spec || !ctxName) return
    const fwd = new SavedPortForward({
      id: spec.id,
      namespace: spec.namespace,
      resource: `${spec.targetKind}s/${spec.targetName}`,
      targetKind: spec.targetKind,
      targetName: spec.targetName,
      targetGVR: spec.targetGVR ?? '',
      localPort: spec.localPort,
      remotePort: spec.remotePort,
      enabled: true,
    })
    try {
      await PortForwardService.SavePortForward(ctxName, fwd)
    } catch (e: any) {
      notificationStore.error(`Failed to save port-forward: ${e?.message ?? e}`)
    }
    await refresh()
  }

  function rowActions(item: Record<string, any>) {
    const isActive = item.status === 'active' || item.status === 'reconnecting'
    return [
      {
        label: isActive ? 'Disconnect' : 'Connect',
        onClick: async () => {
          try {
            if (isActive) {
              await PortForwardService.StopForward(item.id)
            } else {
              await PortForwardService.StartForward(
                ctxName, item.namespace, item.targetKind, item.targetName, item.targetGVR, item.localPort, item.remotePort
              )
            }
          } catch (e: any) {
            notificationStore.error(e?.message ?? String(e))
          }
          await refresh()
        },
      },
      {
        label: item.enabled ? 'Disable' : 'Enable',
        onClick: async () => {
          try {
            await PortForwardService.SetPortForwardEnabled(ctxName, item.id, !item.enabled)
          } catch (e: any) {
            notificationStore.error(e?.message ?? String(e))
          }
          await refresh()
        },
      },
      {
        label: 'Copy URL',
        onClick: () => {
          navigator.clipboard.writeText(`http://localhost:${item.localPort}`)
          notificationStore.push('URL copied', 'success')
        },
      },
      {
        label: 'Remove',
        variant: 'destructive' as const,
        onClick: async () => {
          try {
            await PortForwardService.RemoveSavedPortForward(ctxName, item.id)
          } catch (e: any) {
            notificationStore.error(e?.message ?? String(e))
          }
          items = items.filter((i) => i.id !== item.id)
        },
      },
    ]
  }

  let unsub: (() => void) | undefined

  onMount(async () => {
    descriptorRegistry.registerVirtual(PF_GVR, pfDescriptor)
    await columnStore.loadForGVR(PF_GVR)
    await refresh()
  })

  $effect(() => {
    if (!ctxName) return
    unsub?.()
    unsub = Events.On(`portforward:${ctxName}:updated`, () => { refresh() })
    return () => { unsub?.(); unsub = undefined }
  })

  onDestroy(() => unsub?.())
</script>

<div class="flex flex-col h-full">
  <div class="flex items-center justify-between px-4 py-2 border-b border-border shrink-0">
    <h1 class="text-sm font-semibold">Port Forwards</h1>
    <button
      onclick={() => { dialogOpen = true }}
      class="flex items-center gap-1 px-2 py-1 text-xs rounded bg-accent text-white hover:bg-accent/80 transition-colors"
    >
      <Plus size={12} />
      New Port Forward
    </button>
  </div>

  <div class="flex-1 overflow-hidden">
    <ResourceList
      {items}
      contextName={ctxName}
      gvr={PF_GVR}
      {loading}
      {rowActions}
    />
  </div>
</div>

{#if dialogOpen}
  <PortForwardDialog
    onclose={() => { dialogOpen = false }}
    oncreated={handleCreated}
    prefillContext={ctxName}
  />
{/if}
