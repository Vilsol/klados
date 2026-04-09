<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { Events } from '@wailsio/runtime'
  import { Plus, Play, Square, ToggleLeft, ToggleRight, Copy, Trash2 } from 'lucide-svelte'
  import ResourceList from '$lib/components/ResourceList.svelte'
  import PortForwardDialog from '$lib/components/PortForwardDialog.svelte'
  import { descriptorRegistry, type DescriptorDef } from '$lib/registry/index'
  import { columnStore } from '$lib/stores/columns.svelte'
  import { clusterStore } from '$lib/stores/cluster.svelte'
  import { notificationStore } from '$lib/stores/notification.svelte'
  import * as PortForwardService from '../../../bindings/github.com/Vilsol/klados/internal/services/portforwardservice.js'

  const PF_GVR = '_internal.v1.portforwards'

  const pfDescriptor: DescriptorDef = {
    group: '_internal',
    version: 'v1',
    resource: 'portforwards',
    kind: 'PortForward',
    gvr: PF_GVR,
    columns: [
      { name: 'Resource',       expr: 'resource',    renderType: 'text' },
      { name: 'Namespace',      expr: 'namespace',   renderType: 'text' },
      { name: 'Local Port',     expr: 'localPort',   renderType: 'text' },
      { name: 'Remote Port',    expr: 'remotePort',  renderType: 'text' },
      { name: 'Status',         expr: 'status',      renderType: 'badge' },
      { name: 'Enabled',        expr: 'enabled',     renderType: 'text' },
      // Hidden sentinel so withControlledBy() skips adding its column
      { name: 'Controlled By',  expr: '',            renderType: 'controlledBy', hidden: true },
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

      const savedIds = new Set((saved ?? []).map((s: any) => s?.id).filter(Boolean))
      const activeMap = new Map<string, any>()
      for (const f of active ?? []) {
        if (f?.id) activeMap.set(f.id, f)
      }

      const savedItems = (saved ?? []).map((s: any) => {
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
          isSaved: true,
          metadata: { name: s?.id ?? '', namespace: s?.namespace ?? '' },
        }
      })

      // Include active forwards not yet in saved list (e.g. created before auto-save)
      const unsavedActive = (active ?? [])
        .filter((f: any) => f?.id && !savedIds.has(f.id))
        .map((f: any) => ({
          id: f.id,
          resource: f.targetName ?? '',
          namespace: f.namespace ?? '',
          localPort: f.localPort ?? 0,
          remotePort: f.remotePort ?? 0,
          enabled: false,
          targetKind: f.targetKind ?? '',
          targetName: f.targetName ?? '',
          targetGVR: f.targetGVR ?? '',
          status: f.status ?? 'active',
          error: f.error ?? '',
          podName: f.podName ?? '',
          isSaved: false,
          metadata: { name: f.id, namespace: f.namespace ?? '' },
        }))

      items = [...savedItems, ...unsavedActive]
    } finally {
      loading = false
    }
  }

  async function handleCreated(_spec: any) {
    // StartForward auto-saves; just refresh
    await refresh()
  }

  function rowActions(item: Record<string, any>) {
    const isActive = item.status === 'active' || item.status === 'reconnecting'
    return [
      {
        label: isActive ? 'Disconnect' : 'Connect',
        icon: isActive ? Square : Play,
        onClick: async () => {
          try {
            if (isActive) {
              await PortForwardService.StopForward(item.id)
            } else if (item.isSaved) {
              // Saved forward — reconnect using existing ID to avoid duplicate entry
              await PortForwardService.ConnectSavedForward(ctxName, item.id)
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
        icon: item.enabled ? ToggleRight : ToggleLeft,
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
        icon: Copy,
        onClick: () => {
          navigator.clipboard.writeText(`http://localhost:${item.localPort}`)
          notificationStore.push('URL copied', 'success')
        },
      },
      {
        label: 'Remove',
        icon: Trash2,
        variant: 'destructive' as const,
        onClick: async () => {
          try {
            // Stop the tunnel first if running, then remove from config
            if (isActive) await PortForwardService.StopForward(item.id).catch(() => {})
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
