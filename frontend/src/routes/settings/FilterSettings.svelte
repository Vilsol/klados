<script lang="ts">
  import { onMount } from 'svelte'
  import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'

  interface FilterEntry {
    name: string
    labels?: Record<string, string>
    annotations?: Record<string, string>
    search?: string
  }

  let filtersByGVR = $state<Record<string, FilterEntry[]>>({})
  let newGVR = $state<string>('')

  let editingGVR = $state<string | null>(null)
  let editingIndex = $state<number>(-1)
  let editName = $state<string>('')
  let editLabels = $state<string>('')
  let editAnnotations = $state<string>('')
  let editSearch = $state<string>('')
  let showModal = $state<boolean>(false)

  onMount(() => {
    ;(async () => {
      const config = await ConfigService.GetConfig()
      if (config && (config as any).savedFilters) {
        filtersByGVR = (config as any).savedFilters
      }
    })()
  })

  function parseKV(str: string): Record<string, string> | undefined {
    if (!str.trim()) return undefined
    const result: Record<string, string> = {}
    for (const pair of str.split(',')) {
      const [key, ...rest] = pair.split('=')
      if (key?.trim() && rest.length > 0) {
        result[key.trim()] = rest.join('=').trim()
      }
    }
    return Object.keys(result).length > 0 ? result : undefined
  }

  function formatKV(obj?: Record<string, string>): string {
    if (!obj) return ''
    return Object.entries(obj).map(([k, v]) => `${k}=${v}`).join(', ')
  }

  function openAdd(gvr: string) {
    editingGVR = gvr
    editingIndex = -1
    editName = ''
    editLabels = ''
    editAnnotations = ''
    editSearch = ''
    showModal = true
  }

  function openEdit(gvr: string, index: number) {
    const filter = filtersByGVR[gvr]?.[index]
    if (!filter) return
    editingGVR = gvr
    editingIndex = index
    editName = filter.name
    editLabels = formatKV(filter.labels)
    editAnnotations = formatKV(filter.annotations)
    editSearch = filter.search ?? ''
    showModal = true
  }

  function closeModal() {
    showModal = false
    editingGVR = null
  }

  async function saveFilter() {
    if (!editingGVR || !editName.trim()) return
    const filter: FilterEntry = {
      name: editName.trim(),
      labels: parseKV(editLabels),
      annotations: parseKV(editAnnotations),
      search: editSearch.trim() || undefined,
    }

    const gvr = editingGVR
    const existing = [...(filtersByGVR[gvr] ?? [])]
    if (editingIndex >= 0) {
      existing[editingIndex] = filter
    } else {
      existing.push(filter)
    }
    filtersByGVR = { ...filtersByGVR, [gvr]: existing }
    await ConfigService.SetSavedFilters(gvr, existing as any[])
    closeModal()
  }

  async function deleteFilter(gvr: string, index: number) {
    const existing = [...(filtersByGVR[gvr] ?? [])]
    existing.splice(index, 1)
    if (existing.length === 0) {
      const { [gvr]: _, ...rest } = filtersByGVR
      filtersByGVR = rest
    } else {
      filtersByGVR = { ...filtersByGVR, [gvr]: existing }
    }
    await ConfigService.SetSavedFilters(gvr, existing as any[])
  }

  function addGVR() {
    const gvr = newGVR.trim()
    if (gvr && !(gvr in filtersByGVR)) {
      filtersByGVR = { ...filtersByGVR, [gvr]: [] }
      newGVR = ''
    }
  }

  let gvrKeys = $derived(Object.keys(filtersByGVR).sort())
</script>

{#if showModal}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onclick={closeModal}>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="bg-bg border border-border rounded-lg p-6 w-96 space-y-4" onclick={(e) => e.stopPropagation()}>
      <h3 class="text-base font-medium text-fg">{editingIndex >= 0 ? 'Edit' : 'Add'} Filter</h3>

      <div>
        <label class="block text-sm font-medium text-fg mb-1">Name</label>
        <input
          type="text"
          bind:value={editName}
          class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-fg mb-1">Labels</label>
        <input
          type="text"
          bind:value={editLabels}
          placeholder="key=value, key2=value2"
          class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
        />
        <p class="text-xs text-muted-foreground mt-1">Comma-separated key=value pairs</p>
      </div>

      <div>
        <label class="block text-sm font-medium text-fg mb-1">Annotations</label>
        <input
          type="text"
          bind:value={editAnnotations}
          placeholder="key=value, key2=value2"
          class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-fg mb-1">Search Text</label>
        <input
          type="text"
          bind:value={editSearch}
          class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
        />
      </div>

      <div class="flex justify-end gap-2 pt-2">
        <button class="px-3 py-1.5 rounded border border-border text-fg text-sm hover:bg-surface-hover" onclick={closeModal}>
          Cancel
        </button>
        <button class="px-3 py-1.5 rounded bg-accent text-accent-foreground text-sm hover:opacity-90" onclick={saveFilter}>
          Save
        </button>
      </div>
    </div>
  </div>
{/if}

<div class="max-w-3xl space-y-6">
  <h2 class="text-base font-medium text-fg">Saved Filters</h2>

  {#each gvrKeys as gvr}
    <div class="border border-border rounded">
      <div class="flex items-center justify-between px-4 py-2 bg-surface border-b border-border">
        <span class="text-sm font-mono text-fg">{gvr}</span>
        <button
          class="text-sm text-accent hover:underline"
          onclick={() => openAdd(gvr)}
        >
          + Add filter
        </button>
      </div>
      {#if (filtersByGVR[gvr] ?? []).length === 0}
        <div class="px-4 py-3 text-sm text-muted-foreground">No filters for this resource type.</div>
      {:else}
        {#each filtersByGVR[gvr] ?? [] as filter, i}
          <div class="flex items-center justify-between px-4 py-2 border-b border-border last:border-0">
            <div>
              <span class="text-sm text-fg font-medium">{filter.name}</span>
              {#if filter.search}
                <span class="text-xs text-muted-foreground ml-2">search: {filter.search}</span>
              {/if}
            </div>
            <div class="flex gap-2">
              <button class="text-xs text-muted-foreground hover:text-fg" onclick={() => openEdit(gvr, i)}>Edit</button>
              <button class="text-xs text-destructive hover:underline" onclick={() => deleteFilter(gvr, i)}>Delete</button>
            </div>
          </div>
        {/each}
      {/if}
    </div>
  {/each}

  {#if gvrKeys.length === 0}
    <p class="text-sm text-muted-foreground">No saved filters. Add a resource type to get started.</p>
  {/if}

  <div>
    <h3 class="text-sm font-medium text-fg mb-2">Add resource type</h3>
    <div class="flex gap-2">
      <input
        type="text"
        bind:value={newGVR}
        placeholder="e.g. apps.v1.deployments"
        class="flex-1 px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
        onkeydown={(e) => e.key === 'Enter' && addGVR()}
      />
      <button
        class="px-3 py-1.5 rounded bg-accent text-accent-foreground text-sm hover:opacity-90"
        onclick={addGVR}
      >
        Add
      </button>
    </div>
  </div>
</div>
