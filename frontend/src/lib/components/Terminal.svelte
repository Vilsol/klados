<script lang="ts">
  import { onMount } from 'svelte'
  import { Terminal } from '@xterm/xterm'
  import { FitAddon } from '@xterm/addon-fit'
  import { WebglAddon } from '@xterm/addon-webgl'
  import { ClipboardAddon } from '@xterm/addon-clipboard'
  import * as ConfigService from '../../../bindings/github.com/Vilsol/klados/internal/services/configservice.js'
  import type { StreamingConfig } from '$lib/stores/streaming.svelte'
  import { shortcutStore } from '$lib/stores/shortcuts.svelte'

  let { sessionID, streamingConfig, ondisconnect }: {
    sessionID: string
    streamingConfig: StreamingConfig
    ondisconnect?: () => void
  } = $props()

  let container: HTMLDivElement

  function buildURL() {
    return `ws://127.0.0.1:${streamingConfig.port}/${streamingConfig.token}/ws/exec/${sessionID}`
  }

  onMount(async () => {
    const term = new Terminal({
      scrollback: 10000,
      fontFamily: 'monospace',
      fontSize: 13,
      cursorBlink: true,
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)

    term.textarea?.addEventListener('focus', () => shortcutStore.setMode('terminal'))
    term.textarea?.addEventListener('blur', () => shortcutStore.setMode('normal'))

    if (await ConfigService.GetTerminalWebGL()) {
      try {
        term.loadAddon(new WebglAddon())
      } catch {
        // unsupported
      }
    }

    try {
      term.loadAddon(new ClipboardAddon())
    } catch {
      // optional
    }

    const ws = new WebSocket(buildURL())
    ws.binaryType = 'arraybuffer'

    function sendResize() {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
    }

    ws.onopen = () => sendResize()

    ws.onmessage = (e) => {
      if (e.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(e.data))
      }
    }

    ws.onclose = () => {
      term.writeln('\r\n[session closed]')
      ondisconnect?.()
    }

    ws.onerror = () => term.writeln('\r\n[connection error]')

    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(new TextEncoder().encode(data))
      }
    })

    const ro = new ResizeObserver((entries) => {
      const { width, height } = entries[0].contentRect
      if (width === 0 || height === 0) return
      if (!term.element) {
        term.open(container)
      }
      fitAddon.fit()
      sendResize()
    })
    ro.observe(container)

    return () => {
      ro.disconnect()
      ws.close()
      term.dispose()
    }
  })
</script>

<div bind:this={container} class="h-full w-full bg-[#1a1a1a] overflow-hidden"></div>
