<script lang="ts">
  import * as ResourceService from '../../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import { notificationStore } from '$lib/stores/notification.svelte'

  let {
    obj = $bindable(),
    ctxName,
    gvr,
    namespace,
    name,
  }: {
    obj: Record<string, any>
    ctxName: string
    gvr: string
    namespace: string
    name: string
  } = $props()

  let editing = $state(false)
  let saving = $state(false)

  let editLabels = $state<[string, string][]>([])
  let editAnnotations = $state<[string, string][]>([])

  function startEdit() {
    editLabels = Object.entries(obj.metadata?.labels ?? {}).map(([k, v]) => [k, String(v)])
    editAnnotations = Object.entries(obj.metadata?.annotations ?? {}).map(([k, v]) => [k, String(v)])
    editing = true
  }

  function cancelEdit() {
    editing = false
  }

  async function save() {
    saving = true
    try {
      const updated = JSON.parse(JSON.stringify(obj))
      updated.metadata.labels = Object.fromEntries(editLabels.filter(([k]) => k.trim()))
      updated.metadata.annotations = Object.fromEntries(editAnnotations.filter(([k]) => k.trim()))
      const result = await ResourceService.UpdateResource(ctxName, gvr, namespace, updated)
      if (result) obj = result
      editing = false
      notificationStore.push('Labels and annotations saved.', 'success')
    } catch (e: any) {
      notificationStore.push(e?.message ?? 'Save failed', 'error')
    } finally {
      saving = false
    }
  }

  const labels = $derived(Object.entries(obj.metadata?.labels ?? {}))
  const annotations = $derived(Object.entries(obj.metadata?.annotations ?? {}))
</script>

<div class="p-4 flex flex-col gap-6 overflow-auto">
  <div class="flex items-center justify-between">
    <span class="text-xs font-semibold text-muted uppercase tracking-wide">Labels & Annotations</span>
    {#if !editing}
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
          onclick={save}
          disabled={saving}
          class="text-xs px-2.5 py-1 rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity disabled:opacity-50"
        >{saving ? 'Saving…' : 'Save'}</button>
      </div>
    {/if}
  </div>

  {#if !editing}
    <section>
      <h3 class="text-xs font-medium mb-2">Labels</h3>
      {#if labels.length === 0}
        <p class="text-xs text-muted">None</p>
      {:else}
        <div class="flex flex-wrap gap-1.5">
          {#each labels.sort(([a], [b]) => a.localeCompare(b)) as [k, v]}
            <span class="text-xs font-mono bg-surface border border-border rounded px-2 py-0.5">
              <span class="text-accent">{k}</span><span class="text-muted">=</span>{v}
            </span>
          {/each}
        </div>
      {/if}
    </section>

    <section>
      <h3 class="text-xs font-medium mb-2">Annotations</h3>
      {#if annotations.length === 0}
        <p class="text-xs text-muted">None</p>
      {:else}
        <div class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5">
          {#each annotations.sort(([a], [b]) => a.localeCompare(b)) as [k, v]}
            <span class="text-xs font-mono text-muted">{k}</span>
            <span class="text-xs font-mono break-all">{v}</span>
          {/each}
        </div>
      {/if}
    </section>
  {:else}
    <section>
      <h3 class="text-xs font-medium mb-2">Labels</h3>
      <div class="flex flex-col gap-1.5">
        {#each editLabels as pair, i}
          <div class="flex gap-2 items-center">
            <input
              bind:value={editLabels[i][0]}
              placeholder="key"
              class="text-xs font-mono flex-1 bg-surface border border-border rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-accent"
            />
            <span class="text-muted text-xs">=</span>
            <input
              bind:value={editLabels[i][1]}
              placeholder="value"
              class="text-xs font-mono flex-1 bg-surface border border-border rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-accent"
            />
            <button
              onclick={() => editLabels.splice(i, 1)}
              class="text-xs text-muted hover:text-destructive transition-colors"
            >✕</button>
          </div>
        {/each}
        <button
          onclick={() => editLabels.push(['', ''])}
          class="text-xs text-accent hover:underline self-start"
        >+ Add label</button>
      </div>
    </section>

    <section>
      <h3 class="text-xs font-medium mb-2">Annotations</h3>
      <div class="flex flex-col gap-1.5">
        {#each editAnnotations as pair, i}
          <div class="flex gap-2 items-center">
            <input
              bind:value={editAnnotations[i][0]}
              placeholder="key"
              class="text-xs font-mono flex-1 bg-surface border border-border rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-accent"
            />
            <span class="text-muted text-xs">=</span>
            <input
              bind:value={editAnnotations[i][1]}
              placeholder="value"
              class="text-xs font-mono flex-1 bg-surface border border-border rounded px-2 py-1 focus:outline-none focus:ring-1 focus:ring-accent"
            />
            <button
              onclick={() => editAnnotations.splice(i, 1)}
              class="text-xs text-muted hover:text-destructive transition-colors"
            >✕</button>
          </div>
        {/each}
        <button
          onclick={() => editAnnotations.push(['', ''])}
          class="text-xs text-accent hover:underline self-start"
        >+ Add annotation</button>
      </div>
    </section>
  {/if}
</div>
