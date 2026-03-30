<script lang="ts">
  import { onDestroy, untrack } from 'svelte'
  import { ChevronDown } from 'lucide-svelte'
  import * as LogService from '../../../../bindings/github.com/Vilsol/klados/internal/services/logservice.js'
  import { LogOptions } from '../../../../bindings/github.com/Vilsol/klados/internal/logs/models.js'
  import { streamingStore } from '$lib/stores/streaming.svelte'
  import LogViewer from '$lib/components/LogViewer.svelte'

  let { obj, ctxName, namespace, name }: {
    obj: Record<string, any>
    ctxName: string
    namespace: string
    name: string
  } = $props()

  const containers = $derived<any[]>([
    ...(obj.spec?.containers ?? []).map((c: any) => ({ name: c.name, init: false })),
    ...(obj.spec?.initContainers ?? []).map((c: any) => ({ name: c.name, init: true })),
  ])

  let selectedContainer = $state('')
  let containerDropdownOpen = $state(false)
  let timestamps = $state(false)
  let previous = $state(false)

  const showTimestamps = $derived(timestamps)
  let streamID = $state<string | null>(null)
  let starting = $state(false)
  let logViewer: ReturnType<typeof LogViewer>
  let downloadDropdownOpen = $state(false)

  // Validated container: always valid for the current pod's container list
  const effectContainer = $derived(
    containers.some(c => c.name === selectedContainer)
      ? selectedContainer
      : (containers[0]?.name ?? '')
  )

  let tailLines = $state<number | undefined>(200)
  let scrollToTopOnLoad = $state(false)

  // Keep selectedContainer UI state in sync when pod changes; reset load options
  $effect(() => {
    const c = effectContainer
    untrack(() => {
      selectedContainer = c
      tailLines = 200
      scrollToTopOnLoad = false
    })
  })

  $effect(() => {
    const container = effectContainer
    const prev = previous
    const _ctx = ctxName
    const _ns = namespace
    const _name = name
    const _tail = tailLines
    if (!container || !streamingStore.config) return

    let cancelled = false
    let myID: string | null = null
    starting = true

    LogService.StartLogStream(_ctx, _ns, _name, new LogOptions({
      container,
      follow: true,
      tailLines: _tail,
      timestamps: true,
      previous: prev,
    })).then(id => {
      if (cancelled) { LogService.StopLogStream(id); return }
      myID = id
      streamID = id
      starting = false
    }).catch(() => { starting = false })

    return () => {
      cancelled = true
      starting = false
      streamID = null
      if (myID) LogService.StopLogStream(myID)
    }
  })

  function selectContainer(n: string) {
    selectedContainer = n
    containerDropdownOpen = false
  }

  function handleClickOutside(e: MouseEvent) {
    const t = e.target as HTMLElement
    if (!t.closest('[data-container-dropdown]')) containerDropdownOpen = false
    if (!t.closest('[data-download-dropdown]')) downloadDropdownOpen = false
  }

  const filename = $derived(`${namespace}-${name}-${selectedContainer || 'all'}`)

  let downloading = $state(false)

  async function downloadAll() {
    if (downloading || !streamingStore.config) return
    downloading = true
    try {
      const id = await LogService.StartLogStream(ctxName, namespace, name, new LogOptions({
        container: selectedContainer,
        follow: false,
        timestamps: true,
      }))
      const allLines: string[] = []
      let buf = ''
      await new Promise<void>((resolve) => {
        const socket = new WebSocket(`ws://127.0.0.1:${streamingStore.config!.port}/${streamingStore.config!.token}/ws/logs/${id}`)
        socket.onmessage = (e) => {
          if (typeof e.data !== 'string') return
          try {
            const msg = JSON.parse(e.data)
            if (msg.type === 'eof' || msg.type === 'error') { socket.close(); resolve(); return }
          } catch {}
          const parts = (buf + e.data).split('\n')
          buf = parts.pop() ?? ''
          allLines.push(...parts)
        }
        socket.onerror = () => resolve()
        socket.onclose = () => resolve()
      })
      if (buf) allLines.push(buf)
      const blob = new Blob([allLines.join('\n')], { type: 'text/plain' })
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = `${filename}.log`
      a.click()
      URL.revokeObjectURL(a.href)
      LogService.StopLogStream(id)
    } finally {
      downloading = false
    }
  }

  const containerLabel = $derived(() => {
    if (!selectedContainer) return 'All'
    const c = containers.find(c => c.name === selectedContainer)
    return c ? `${c.name}${c.init ? ' (init)' : ''}` : selectedContainer
  })

  onDestroy(() => {
    if (streamID) LogService.StopLogStream(streamID)
  })
</script>

<svelte:document onclick={handleClickOutside} />

{#if !streamingStore.config}
  <div class="flex items-center justify-center h-full text-sm text-muted">
    Waiting for streaming server…
  </div>
{:else}
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Control bar -->
    <div class="flex items-center gap-2 px-3 py-1.5 border-b border-border bg-surface shrink-0 text-xs flex-wrap">
      <!-- Container dropdown -->
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
            <button
              onclick={() => selectContainer('')}
              class="w-full text-left px-3 py-1.5 text-xs hover:bg-surface-hover transition-colors
                {selectedContainer === '' ? 'font-medium text-fg' : 'text-muted'}"
            >All</button>
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

      <label class="flex items-center gap-1 text-xs text-muted select-none cursor-pointer">
        <input type="checkbox" bind:checked={timestamps} class="accent-accent" />
        Timestamps
      </label>
      <label class="flex items-center gap-1 text-xs text-muted select-none cursor-pointer">
        <input type="checkbox" bind:checked={previous} class="accent-accent" />
        Previous
      </label>

      <button
        onclick={() => { tailLines = undefined; scrollToTopOnLoad = true }}
        class="text-xs text-muted hover:text-fg border border-border rounded px-2 py-1 transition-colors"
        title="Load full history and jump to beginning"
      >Full history</button>

      <div class="relative ml-auto" data-download-dropdown>
        <button
          onclick={() => (downloadDropdownOpen = !downloadDropdownOpen)}
          class="flex items-center gap-1 text-xs text-muted hover:text-fg border border-border rounded px-2 py-1 transition-colors"
        >Download ↓</button>
        {#if downloadDropdownOpen}
          <div class="absolute top-full right-0 mt-1 min-w-[5rem] rounded border border-border bg-bg shadow-lg z-50">
            <button
              onclick={() => { logViewer?.downloadVisible(); downloadDropdownOpen = false }}
              class="w-full text-left px-3 py-1.5 text-xs text-muted hover:bg-surface-hover transition-colors"
            >Visible</button>
            <button
              onclick={() => { downloadAll(); downloadDropdownOpen = false }}
              class="w-full text-left px-3 py-1.5 text-xs text-muted hover:bg-surface-hover transition-colors"
            >All</button>
          </div>
        {/if}
      </div>

      {#if starting}
        <span class="text-xs text-muted italic">Connecting…</span>
      {/if}
    </div>

    <!-- Log viewer -->
    <div class="flex-1 overflow-hidden">
      {#if streamID}
        <LogViewer bind:this={logViewer} {streamID} streamingConfig={streamingStore.config} {showTimestamps} {filename} {scrollToTopOnLoad} />
      {:else}
        <div class="flex items-center justify-center h-full text-sm text-muted">Loading…</div>
      {/if}
    </div>
  </div>
{/if}
