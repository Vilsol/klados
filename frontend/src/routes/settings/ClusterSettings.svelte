<script lang="ts">
  import { onMount } from 'svelte'
  import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'

  interface Props {
    ctxName: string
  }

  let { ctxName }: Props = $props()

  let displayName = $state<string>('')
  let accentColor = $state<string>('')
  let readOnlyOverride = $state<boolean>(false)
  let readOnlyValue = $state<boolean>(false)
  let compactOverride = $state<boolean>(false)
  let compactValue = $state<boolean>(false)
  let favoriteNamespaces = $state<string[]>([])
  let newNamespace = $state<string>('')

  onMount(() => {
    ;(async () => {
      const prefs = await ConfigService.GetClusterPrefs(ctxName)
      if (prefs) {
        const p = prefs as any
        displayName = p.displayName ?? ''
        accentColor = p.accentColor ?? ''
        readOnlyOverride = p.readOnly != null
        readOnlyValue = p.readOnly ?? false
        compactOverride = p.compactRows != null
        compactValue = p.compactRows ?? false
        favoriteNamespaces = p.favoriteNamespaces ?? []
      }
    })()
  })

  function save() {
    ConfigService.SetClusterPrefs(ctxName, {
      displayName: displayName || undefined,
      accentColor: accentColor || undefined,
      readOnly: readOnlyOverride ? readOnlyValue : undefined,
      compactRows: compactOverride ? compactValue : undefined,
      favoriteNamespaces: favoriteNamespaces.length > 0 ? favoriteNamespaces : undefined,
    } as any)
  }

  function setDisplayName(value: string) {
    displayName = value
    save()
  }

  function setAccent(value: string) {
    accentColor = value
    save()
  }

  function toggleReadOnlyOverride(enabled: boolean) {
    readOnlyOverride = enabled
    if (!enabled) readOnlyValue = false
    save()
  }

  function setReadOnly(value: boolean) {
    readOnlyValue = value
    save()
  }

  function toggleCompactOverride(enabled: boolean) {
    compactOverride = enabled
    if (!enabled) compactValue = false
    save()
  }

  function setCompact(value: boolean) {
    compactValue = value
    save()
  }

  function addNamespace() {
    const ns = newNamespace.trim()
    if (ns && !favoriteNamespaces.includes(ns)) {
      favoriteNamespaces = [...favoriteNamespaces, ns]
      newNamespace = ''
      save()
    }
  }

  function removeNamespace(ns: string) {
    favoriteNamespaces = favoriteNamespaces.filter((n) => n !== ns)
    save()
  }
</script>

<div class="max-w-2xl space-y-8">
  <h2 class="text-base font-medium text-fg mb-4">Cluster: {ctxName}</h2>

  <div>
    <label class="block text-sm font-medium text-fg mb-1">Display Name</label>
    <input
      type="text"
      value={displayName}
      oninput={(e) => setDisplayName((e.target as HTMLInputElement).value)}
      placeholder={ctxName}
      class="w-full px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
    />
  </div>

  <div>
    <label class="block text-sm font-medium text-fg mb-1">Accent Color</label>
    <div class="flex items-center gap-3">
      <input
        type="color"
        value={accentColor || '#6366f1'}
        oninput={(e) => setAccent((e.target as HTMLInputElement).value)}
        class="w-8 h-8 rounded cursor-pointer border border-border"
      />
      {#if accentColor}
        <button class="text-sm text-muted-foreground hover:text-fg underline" onclick={() => setAccent('')}>
          Reset
        </button>
      {/if}
    </div>
  </div>

  <div>
    <label class="block text-sm font-medium text-fg mb-2">Read-Only</label>
    <div class="space-y-2">
      <label class="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          checked={readOnlyOverride}
          onchange={(e) => toggleReadOnlyOverride((e.target as HTMLInputElement).checked)}
          class="accent-accent"
        />
        <span class="text-sm text-fg">Override global default</span>
      </label>
      {#if readOnlyOverride}
        <label class="flex items-center gap-2 cursor-pointer ml-6">
          <input
            type="checkbox"
            checked={readOnlyValue}
            onchange={(e) => setReadOnly((e.target as HTMLInputElement).checked)}
            class="accent-accent"
          />
          <span class="text-sm text-fg">Enable read-only mode for this cluster</span>
        </label>
      {:else}
        <p class="text-sm text-muted-foreground ml-6">Using global default</p>
      {/if}
    </div>
  </div>

  <div>
    <label class="block text-sm font-medium text-fg mb-2">Compact Rows</label>
    <div class="space-y-2">
      <label class="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          checked={compactOverride}
          onchange={(e) => toggleCompactOverride((e.target as HTMLInputElement).checked)}
          class="accent-accent"
        />
        <span class="text-sm text-fg">Override global default</span>
      </label>
      {#if compactOverride}
        <label class="flex items-center gap-2 cursor-pointer ml-6">
          <input
            type="checkbox"
            checked={compactValue}
            onchange={(e) => setCompact((e.target as HTMLInputElement).checked)}
            class="accent-accent"
          />
          <span class="text-sm text-fg">Enable compact rows for this cluster</span>
        </label>
      {:else}
        <p class="text-sm text-muted-foreground ml-6">Using global default</p>
      {/if}
    </div>
  </div>

  <div>
    <label class="block text-sm font-medium text-fg mb-2">Favorite Namespaces</label>
    <div class="flex gap-2 mb-2">
      <input
        type="text"
        bind:value={newNamespace}
        placeholder="Namespace name"
        class="flex-1 px-3 py-1.5 rounded border border-border bg-surface text-fg text-sm"
        onkeydown={(e) => e.key === 'Enter' && addNamespace()}
      />
      <button
        class="px-3 py-1.5 rounded bg-accent text-accent-foreground text-sm hover:opacity-90"
        onclick={addNamespace}
      >
        Add
      </button>
    </div>
    {#if favoriteNamespaces.length > 0}
      <div class="flex flex-wrap gap-2">
        {#each favoriteNamespaces as ns}
          <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-surface border border-border text-sm text-fg">
            {ns}
            <button class="text-muted-foreground hover:text-fg ml-1" onclick={() => removeNamespace(ns)}>&times;</button>
          </span>
        {/each}
      </div>
    {:else}
      <p class="text-sm text-muted-foreground">No favorite namespaces configured.</p>
    {/if}
  </div>
</div>
