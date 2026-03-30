<script lang="ts">
  import { Dialog } from 'bits-ui'
  import * as ResourceService from '../../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import ConfirmDialog from '../ConfirmDialog.svelte'
  import { notificationStore } from '$lib/stores/notification.svelte'
  import { push } from 'svelte-spa-router'

  let {
    obj,
    ctxName,
    gvr,
    namespace,
    name,
    actions,
    onrefresh,
  }: {
    obj: Record<string, any>
    ctxName: string
    gvr: string
    namespace: string
    name: string
    actions: string[]
    onrefresh: () => void
  } = $props()

  let deleteOpen = $state(false)
  let forceDeleteOpen = $state(false)
  let scaleOpen = $state(false)
  let restartOpen = $state(false)
  let scaleReplicas = $state<number>(1)
  $effect(() => { scaleReplicas = obj.spec?.replicas ?? 1 })
  let busy = $state(false)

  async function doDelete() {
    busy = true
    try {
      await ResourceService.DeleteResource(ctxName, gvr, namespace, name)
      notificationStore.push(`Deleted ${name}`, 'success')
      push(`/c/${ctxName}/${gvr}`)
    } catch (e: any) {
      notificationStore.push(e?.message ?? 'Delete failed', 'error')
    } finally {
      busy = false
    }
  }

  async function doForceDelete() {
    busy = true
    try {
      await ResourceService.ForceDeleteResource(ctxName, gvr, namespace, name)
      notificationStore.push(`Force deleted ${name}`, 'success')
      push(`/c/${ctxName}/${gvr}`)
    } catch (e: any) {
      notificationStore.push(e?.message ?? 'Force delete failed', 'error')
    } finally {
      busy = false
    }
  }

  async function doScale() {
    busy = true
    try {
      await ResourceService.ScaleResource(ctxName, gvr, namespace, name, scaleReplicas)
      notificationStore.push(`Scaled ${name} to ${scaleReplicas}`, 'success')
      scaleOpen = false
      onrefresh()
    } catch (e: any) {
      notificationStore.push(e?.message ?? 'Scale failed', 'error')
    } finally {
      busy = false
    }
  }

  async function doRestart() {
    busy = true
    try {
      await ResourceService.RestartResource(ctxName, gvr, namespace, name)
      notificationStore.push(`Restarted ${name}`, 'success')
      onrefresh()
    } catch (e: any) {
      notificationStore.push(e?.message ?? 'Restart failed', 'error')
    } finally {
      busy = false
    }
  }


</script>

<div class="flex items-center gap-1.5 px-4 py-2 border-b border-border bg-surface flex-wrap">
  {#if actions.includes('scale')}
    <button
      onclick={() => scaleOpen = true}
      class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
    >Scale</button>
  {/if}

  {#if actions.includes('restart')}
    <button
      onclick={() => restartOpen = true}
      class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
    >Restart</button>
  {/if}

  <div class="flex-1"></div>

  {#if actions.includes('force-delete')}
    <button
      onclick={() => forceDeleteOpen = true}
      class="text-xs px-2.5 py-1 rounded border border-destructive text-destructive hover:bg-destructive/10 transition-colors"
    >Force Delete</button>
  {/if}

  {#if actions.includes('delete')}
    <button
      onclick={() => deleteOpen = true}
      class="text-xs px-2.5 py-1 rounded bg-destructive text-destructive-fg hover:opacity-90 transition-opacity"
    >Delete</button>
  {/if}
</div>

<ConfirmDialog
  bind:open={deleteOpen}
  title="Delete {name}"
  message="This action cannot be undone."
  confirmLabel="Delete"
  onconfirm={doDelete}
/>

<ConfirmDialog
  bind:open={forceDeleteOpen}
  title="Force Delete {name}"
  message="Bypasses graceful termination. The pod will be removed immediately from the API server regardless of node state."
  confirmLabel="Force Delete"
  onconfirm={doForceDelete}
/>

<ConfirmDialog
  bind:open={restartOpen}
  title="Restart {name}"
  message="A rolling restart will be triggered by patching the pod template annotation."
  confirmLabel="Restart"
  onconfirm={doRestart}
/>

<!-- Scale dialog -->
<Dialog.Root bind:open={scaleOpen}>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content
      class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl p-6 w-80"
    >
      <Dialog.Title class="text-base font-semibold mb-4">Scale {name}</Dialog.Title>
      <label for="scale-replicas" class="block text-sm mb-1 text-muted">Replicas</label>
      <input
        id="scale-replicas"
        type="number"
        min="0"
        bind:value={scaleReplicas}
        class="w-full text-sm bg-bg border border-border rounded px-3 py-1.5 focus:outline-none focus:ring-1 focus:ring-accent"
      />
      <div class="flex justify-end gap-2 mt-4">
        <Dialog.Close
          class="px-3 py-1.5 text-sm rounded border border-border hover:bg-surface-hover transition-colors"
        >Cancel</Dialog.Close>
        <button
          onclick={doScale}
          disabled={busy}
          class="px-3 py-1.5 text-sm rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity disabled:opacity-50"
        >{busy ? 'Scaling…' : 'Scale'}</button>
      </div>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
