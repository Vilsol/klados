<script lang="ts">
  import { onDestroy } from 'svelte'
  import { ChevronDown } from 'lucide-svelte'
  import * as ExecService from '../../../../bindings/github.com/Vilsol/klados/internal/services/execservice.js'
  import { streamingStore } from '$lib/stores/streaming.svelte'
  import Terminal from '$lib/components/Terminal.svelte'

  let { obj, ctxName, namespace, name }: {
    obj: Record<string, any>
    ctxName: string
    namespace: string
    name: string
  } = $props()

  interface TermSession {
    id: string
    container: string
    shell: string
  }

  const containers = $derived<any[]>([
    ...(obj.spec?.containers ?? []).map((c: any) => ({ name: c.name, init: false })),
    ...(obj.spec?.initContainers ?? []).map((c: any) => ({ name: c.name, init: true })),
  ])

  const shells = ['bash', 'sh', 'zsh']

  let selectedContainer = $state('')
  let containerDropdownOpen = $state(false)
  let selectedShell = $state('bash')
  let sessions = $state<TermSession[]>([])
  let activeIdx = $state(0)
  let error = $state<string | null>(null)
  let loading = $state(false)

  $effect(() => {
    if (containers.length > 0 && !selectedContainer) {
      selectedContainer = containers[0].name
    }
  })

  function selectContainer(n: string) {
    selectedContainer = n
    containerDropdownOpen = false
  }

  function handleClickOutside(e: MouseEvent) {
    if (!(e.target as HTMLElement).closest('[data-container-dropdown]')) {
      containerDropdownOpen = false
    }
  }

  const containerLabel = $derived(() => {
    if (!selectedContainer) return 'Select container'
    const c = containers.find(c => c.name === selectedContainer)
    return c ? `${c.name}${c.init ? ' (init)' : ''}` : selectedContainer
  })

  async function connect() {
    error = null
    loading = true
    try {
      const id = await ExecService.OpenExecSession(
        ctxName, namespace, name, selectedContainer, selectedShell,
      )
      sessions = [...sessions, { id, container: selectedContainer, shell: selectedShell }]
      activeIdx = sessions.length - 1
    } catch (e: any) {
      error = e?.message ?? String(e)
    } finally {
      loading = false
    }
  }

  function removeSession(i: number) {
    ExecService.CloseExecSession(sessions[i].id)
    sessions = sessions.filter((_, idx) => idx !== i)
    if (activeIdx >= sessions.length) {
      activeIdx = Math.max(0, sessions.length - 1)
    }
  }

  onDestroy(() => {
    for (const s of sessions) {
      ExecService.CloseExecSession(s.id)
    }
  })
</script>

<svelte:document onclick={handleClickOutside} />

{#if sessions.length > 0 && streamingStore.config}
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Compact header: selectors + tab bar + new button -->
    <div class="flex items-center gap-2 px-2 py-1 border-b border-border bg-surface shrink-0 text-xs flex-wrap">
      <!-- Container dropdown (compact) -->
      <div class="relative" data-container-dropdown>
        <button
          onclick={() => (containerDropdownOpen = !containerDropdownOpen)}
          class="flex items-center gap-1 text-xs bg-bg text-fg border border-border rounded px-2 py-1 hover:bg-surface-hover transition-colors"
        >
          <span class="max-w-[8rem] truncate">{containerLabel()}</span>
          <ChevronDown size={12} class="shrink-0 text-muted" />
        </button>
        {#if containerDropdownOpen}
          <div class="absolute top-full left-0 mt-1 min-w-[8rem] rounded border border-border bg-bg shadow-lg z-50">
            {#each containers as c}
              <button
                onclick={() => selectContainer(c.name)}
                class="w-full text-left px-3 py-1.5 text-xs hover:bg-surface-hover transition-colors
                  {selectedContainer === c.name ? 'font-medium text-fg' : 'text-muted'}"
              >
                {c.name}{c.init ? ' (init)' : ''}
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Shell selector (compact) -->
      <div class="flex gap-1">
        {#each shells as shell}
          <button
            onclick={() => selectedShell = shell}
            class="px-2 py-0.5 text-xs rounded border transition-colors
              {selectedShell === shell
                ? 'border-accent text-accent bg-accent/10'
                : 'border-border text-muted hover:bg-surface-hover'}"
          >{shell}</button>
        {/each}
      </div>

      <!-- Session tabs -->
      <div class="flex items-center gap-1 flex-1 overflow-x-auto">
        {#each sessions as s, i}
          <button
            onclick={() => activeIdx = i}
            class="flex items-center gap-1 px-2 py-0.5 rounded border whitespace-nowrap transition-colors
              {i === activeIdx
                ? 'border-accent text-accent bg-accent/10'
                : 'border-border text-muted hover:bg-surface-hover'}"
          >
            <span>{s.shell}:{s.container}</span>
            <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
            <span
              onclick={(e) => { e.stopPropagation(); removeSession(i) }}
              class="ml-1 hover:text-destructive"
            >×</span>
          </button>
        {/each}
      </div>

      <button
        onclick={connect}
        disabled={loading || !selectedContainer}
        class="shrink-0 px-2 py-0.5 text-xs border border-border rounded hover:bg-surface-hover disabled:opacity-50 transition-colors"
        title="New session"
        aria-label="New terminal session"
      >+</button>
    </div>

    <!-- Terminal layers: all mounted, active one visible -->
    <div class="relative flex-1 overflow-hidden">
      {#each sessions as session, i}
        <div class="absolute inset-0" class:invisible={i !== activeIdx}>
          <Terminal
            sessionID={session.id}
            streamingConfig={streamingStore.config}
            ondisconnect={() => removeSession(i)}
          />
        </div>
      {/each}
    </div>
  </div>
{:else}
  <div class="flex flex-col gap-4 p-4 overflow-auto">
    {#if !streamingStore.config}
      <p class="text-sm text-muted">Waiting for streaming server…</p>
    {:else}
      <!-- Container selector -->
      <div class="flex flex-col gap-1">
        <label class="text-xs font-medium text-muted uppercase tracking-wide">Container</label>
        <div class="relative" data-container-dropdown>
          <button
            onclick={() => (containerDropdownOpen = !containerDropdownOpen)}
            class="flex items-center justify-between gap-1 w-full text-sm bg-bg text-fg border border-border rounded px-2 py-1.5 hover:bg-surface-hover transition-colors"
          >
            <span>{containerLabel()}</span>
            <ChevronDown size={14} class="shrink-0 text-muted" />
          </button>
          {#if containerDropdownOpen}
            <div class="absolute top-full left-0 mt-1 w-full rounded border border-border bg-bg shadow-lg z-50">
              {#each containers as c}
                <button
                  onclick={() => selectContainer(c.name)}
                  class="w-full text-left px-3 py-1.5 text-sm hover:bg-surface-hover transition-colors
                    {selectedContainer === c.name ? 'font-medium text-fg' : 'text-muted'}"
                >
                  {c.name}{c.init ? ' (init)' : ''}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      </div>

      <!-- Shell selector -->
      <div class="flex flex-col gap-1">
        <label class="text-xs font-medium text-muted uppercase tracking-wide">Shell</label>
        <div class="flex gap-2">
          {#each shells as shell}
            <button
              onclick={() => selectedShell = shell}
              class="px-3 py-1.5 text-sm rounded border transition-colors
                {selectedShell === shell
                  ? 'border-accent text-accent bg-accent/10'
                  : 'border-border text-muted hover:bg-surface-hover'}"
            >{shell}</button>
          {/each}
        </div>
      </div>

      {#if error}
        <p class="text-sm text-destructive">{error}</p>
      {/if}

      <button
        onclick={connect}
        disabled={loading || !selectedContainer}
        class="self-start px-4 py-2 text-sm bg-accent text-white rounded hover:opacity-90 disabled:opacity-50 transition-opacity"
      >
        {loading ? 'Connecting…' : 'Connect'}
      </button>
    {/if}
  </div>
{/if}
