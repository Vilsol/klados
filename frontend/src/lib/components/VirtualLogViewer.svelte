<script lang="ts">
  import { createVirtualizer } from '@tanstack/svelte-virtual'
  import type { SvelteVirtualizer } from '@tanstack/svelte-virtual'
  import { untrack } from 'svelte'
  import { AnsiUp } from 'ansi_up'
  import stripAnsi from 'strip-ansi'

  let { lines, eofReached = false, eofHistory = false, historyLoading = false, showTimestamps = false, onLoadHistory, filename = 'logs' }: {
    lines: string[]
    eofReached?: boolean
    eofHistory?: boolean
    historyLoading?: boolean
    showTimestamps?: boolean
    onLoadHistory?: () => void
    filename?: string
  } = $props()

  const TS_RE = /^(\d{4}-\d{2}-\d{2}T[\d:.]+(?:Z|[+-]\d{2}:\d{2})) /

  function esc(s: string): string {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
  }

  function detectFormat(plain: string): 'json' | 'logfmt' | 'klog' | 'clf' | null {
    const t = plain.trimStart()
    if (t.startsWith('{') || t.startsWith('[')) return 'json'
    if (/^[IWEF]\d{4} \d{2}:\d{2}:\d{2}/.test(t)) return 'klog'
    if (/^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3} /.test(t)) return 'clf'
    if ((t.match(/\w[\w.]*=(?:"[^"]*"|\S+)/g)?.length ?? 0) >= 2) return 'logfmt'
    return null
  }

  function highlightJSON(text: string): string {
    let out = ''
    let i = 0
    const depth: Array<'obj' | 'arr'> = []
    while (i < text.length) {
      const ch = text[i]
      if (ch === '{' || ch === '[') {
        depth.push(ch === '{' ? 'obj' : 'arr')
        out += `<span class="hl-p">${ch}</span>`; i++
      } else if (ch === '}' || ch === ']') {
        depth.pop()
        out += `<span class="hl-p">${ch}</span>`; i++
      } else if (ch === ':') {
        out += `<span class="hl-p">:</span>`; i++
      } else if (ch === ',') {
        out += `<span class="hl-p">,</span>`; i++
      } else if (ch === '"') {
        let j = i + 1
        while (j < text.length) {
          if (text[j] === '\\') { j += 2; continue }
          if (text[j] === '"') { j++; break }
          j++
        }
        let k = j; while (k < text.length && (text[k] === ' ' || text[k] === '\t')) k++
        const isKey = depth[depth.length - 1] === 'obj' && text[k] === ':'
        out += isKey
          ? `<span class="hl-k">${esc(text.slice(i, j))}</span>`
          : `<span class="hl-s">${esc(text.slice(i, j))}</span>`
        i = j
      } else if (text.slice(i, i + 4) === 'true' || text.slice(i, i + 5) === 'false') {
        const w = text[i + 4] === 'e' ? 4 : 5
        out += `<span class="hl-b">${text.slice(i, i + w)}</span>`; i += w
      } else if (text.slice(i, i + 4) === 'null') {
        out += `<span class="hl-n">null</span>`; i += 4
      } else if (ch === '-' || (ch >= '0' && ch <= '9')) {
        let j = i; if (text[j] === '-') j++
        while (j < text.length && /[\d.eE+\-]/.test(text[j])) j++
        out += `<span class="hl-num">${text.slice(i, j)}</span>`; i = j
      } else {
        out += ch === '<' ? '&lt;' : ch === '>' ? '&gt;' : ch === '&' ? '&amp;' : ch
        i++
      }
    }
    return out
  }

  function highlightLogfmt(text: string): string {
    const RE = /(\w[\w.]*)=("(?:[^"\\]|\\.)*"|\S*)/g
    let out = ''; let last = 0; let m: RegExpExecArray | null
    while ((m = RE.exec(text)) !== null) {
      out += esc(text.slice(last, m.index))
      out += `<span class="hl-k">${esc(m[1])}</span><span class="hl-p">=</span><span class="hl-s">${esc(m[2])}</span>`
      last = m.index + m[0].length
    }
    return out + esc(text.slice(last))
  }

  function highlightKlog(text: string): string {
    const m = /^([IWEF])(\d{4} \d{2}:\d{2}:\d{2}\.\d+)\s+(\d+)\s+(\S+)\]\s*(.*)/.exec(text)
    if (!m) return esc(text)
    const [, lvl, dt, pid, src, rest] = m
    const lc = lvl === 'E' || lvl === 'F' ? 'log-error' : lvl === 'W' ? 'log-warn' : 'log-info'
    return `<span class="${lc} font-bold">${lvl}</span><span class="hl-p">${esc(dt)}</span> <span class="hl-p">${esc(pid)}</span> <span class="hl-p">${esc(src)}]</span> ${highlightLogfmt(rest)}`
  }

  function highlightCLF(text: string): string {
    const m = /^(\S+) (\S+) (\S+) (\[[^\]]*\]) "([^"]*)" (\d{3}) (\S+)(.*)/.exec(text)
    if (!m) return esc(text)
    const [, host, ident, user, ts, req, status, bytes, rest] = m
    const sc = parseInt(status)
    const sc2 = sc >= 500 ? 'log-error' : sc >= 400 ? 'log-warn' : sc >= 300 ? 'log-info' : 'hl-s'
    const rp = req.split(' ')
    const reqHtml = rp.length >= 2
      ? `<span class="hl-b">${esc(rp[0])}</span> <span class="hl-k">${esc(rp.slice(1, -1).join(' '))}</span> <span class="hl-p">${esc(rp[rp.length - 1])}</span>`
      : esc(req)
    return `<span class="hl-k">${esc(host)}</span> <span class="hl-p">${esc(ident)} ${esc(user)}</span> <span class="hl-p">${esc(ts)}</span> &quot;${reqHtml}&quot; <span class="${sc2}">${status}</span> <span class="hl-num">${esc(bytes)}</span>${esc(rest)}`
  }

  function renderLine(content: string, plain: string): string {
    if (content !== plain) return ansiUp.ansi_to_html(content)
    const fmt = detectFormat(plain)
    if (fmt === 'json') return highlightJSON(plain)
    if (fmt === 'logfmt') return highlightLogfmt(plain)
    if (fmt === 'klog') return highlightKlog(plain)
    if (fmt === 'clf') return highlightCLF(plain)
    return ansiUp.ansi_to_html(content)
  }

  interface ProcessedLine { html: string; plain: string; ts?: string }
  let processedLines: ProcessedLine[] = $state([])
  let ansiUp = new AnsiUp()

  let sticky = $state(true)

  function setSticky(val: boolean) {
    sticky = val
  }

  $effect(() => {
    if (lines.length < processedLines.length) {
      processedLines = []
      ansiUp = new AnsiUp()
      ansiUp.use_classes = true
      setSticky(true)
    }
    while (processedLines.length < lines.length) {
      const raw = lines[processedLines.length]
      const m = TS_RE.exec(raw)
      const content = m ? raw.slice(m[0].length) : raw
      const plain = stripAnsi(content)
      processedLines.push({ html: renderLine(content, plain), plain, ts: m?.[1] })
    }
  })

  let searchQuery = $state('')
  let regexSearch = $state(false)
  let highlight = $state(false)
  let wrap = $state(false)

  let scrollEl = $state<HTMLDivElement | undefined>(undefined)
  let matchCursor = $state(-1)
  // Non-reactive flags for scroll event coordination
  let wheelScrollCount = 0
  let programmaticScroll = false
  let pendingScrollDelta = 0

  const searchPattern = $derived((() => {
    if (!searchQuery) return null
    try {
      return new RegExp(regexSearch ? searchQuery : searchQuery.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i')
    } catch {
      return null
    }
  })())

  const matchIndices = $derived(
    searchPattern
      ? processedLines.reduce<number[]>((acc, l, i) => {
          if (searchPattern.test(l.plain)) acc.push(i)
          return acc
        }, [])
      : []
  )

  // Stable virtualizer store — created once, updated via setOptions
  const virtualizerStore = createVirtualizer<HTMLDivElement, HTMLDivElement>({
    count: 0,
    getScrollElement: () => scrollEl ?? null,
    estimateSize: () => 20,
    overscan: 15,
  })

  // Imperative reference for scrollToIndex etc.
  let virt: SvelteVirtualizer<HTMLDivElement, HTMLDivElement>
  virtualizerStore.subscribe(v => { virt = v })

  function updateVirtOptions(count: number) {
    virt?.setOptions({
      count,
      getScrollElement: () => scrollEl ?? null,
      estimateSize: () => 20,
      overscan: 15,
      measureElement: wrap
        ? (el: Element) => el.getBoundingClientRect().height
        : undefined,
    })
  }

  // Keep virtualizer in sync with count + wrap mode
  $effect(() => {
    const count = processedLines.length
    const _wrap = wrap
    untrack(() => updateVirtOptions(count))
  })

  // Svelte action: triggers measureElement on each row (needed for variable-height wrap mode)
  function measureEl(el: Element) {
    virt?.measureElement(el)
  }

  export function prependLines(batch: string[]) {
    const batchUp = new AnsiUp()
    batchUp.use_classes = true
    const processed: ProcessedLine[] = batch.map(raw => {
      const m = TS_RE.exec(raw)
      const content = m ? raw.slice(m[0].length) : raw
      const plain = stripAnsi(content)
      return { html: renderLine(content, plain), plain, ts: m?.[1] }
    })

    const firstVisibleIndex = $virtualizerStore.getVirtualItems()[0]?.index ?? 0
    wheelScrollCount = 0

    processedLines = [...processed, ...processedLines]

    // Synchronously update virtualizer count and restore scroll position before
    // the browser paints — eliminates the flash from layout/paint at wrong offset
    programmaticScroll = true
    updateVirtOptions(processedLines.length)
    virt?.scrollToIndex(firstVisibleIndex + batch.length, { align: 'start' })
  }

  export function scrollToLine(index: number, _align: 'start' | 'end' = 'start') {
    setSticky(false)
    virt?.scrollToIndex(index, { align: 'start' })
  }

  export function scrollToTop() {
    setSticky(false)
    virt?.scrollToIndex(0, { align: 'start' })
  }

  // Auto-tail: batched via rAF
  let tailRaf: number | null = null
  $effect(() => {
    const count = processedLines.length
    if (!sticky || count === 0 || !scrollEl) return
    untrack(() => {
      if (tailRaf !== null) cancelAnimationFrame(tailRaf)
      tailRaf = requestAnimationFrame(() => {
        tailRaf = null
        if (!sticky || !scrollEl) return
        programmaticScroll = true
        scrollEl.scrollTop = scrollEl.scrollHeight
      })
    })
    return () => { if (tailRaf !== null) { cancelAnimationFrame(tailRaf); tailRaf = null } }
  })

  function onScroll(e: Event) {
    if (programmaticScroll) { programmaticScroll = false; return }
    if (wheelScrollCount > 0) { wheelScrollCount--; return }
    const el = e.target as HTMLElement
    const dist = el.scrollHeight - el.scrollTop - el.clientHeight
    const next = dist < 40
    setSticky(next)
  }

  function onWheel(e: WheelEvent) {
    if (e.deltaY < 0) {
      setSticky(false)
      wheelScrollCount++
      if (scrollEl && scrollEl.scrollTop === 0 && !eofHistory) {
        pendingScrollDelta += e.deltaY
        onLoadHistory?.()
      }
    } else if (e.deltaY > 0) {
      wheelScrollCount = 0
    }
  }

  function levelClass(plain: string): string {
    if (/error|erro|fatal/i.test(plain)) return 'log-error'
    if (/warn|wrn/i.test(plain))         return 'log-warn'
    if (/info/i.test(plain))             return 'log-info'
    if (/debug|dbg/i.test(plain))        return 'log-debug'
    return ''
  }

  function findNext() {
    if (!matchIndices.length) return
    const next = matchIndices.findIndex(i => i > matchCursor)
    const idx = next === -1 ? 0 : next
    matchCursor = matchIndices[idx]
    setSticky(false)
    scrollToLine(matchCursor, 'start')
  }

  function findPrev() {
    if (!matchIndices.length) return
    const prev = [...matchIndices].reverse().findIndex(i => i < matchCursor)
    const idx = prev === -1 ? matchIndices.length - 1 : matchIndices.length - 1 - prev
    matchCursor = matchIndices[idx]
    setSticky(false)
    scrollToLine(matchCursor, 'start')
  }

  function onCopy(e: ClipboardEvent) {
    const sel = window.getSelection()
    if (!sel || sel.isCollapsed || !e.clipboardData) return
    const rows = Array.from(scrollEl?.querySelectorAll('[data-index]') ?? [])
    const selected = rows.filter(r => sel.containsNode(r, true))
    if (selected.length === 0) return
    const text = selected.map(r => {
      const idx = parseInt(r.getAttribute('data-index') ?? '0', 10)
      const pl = processedLines[idx]
      if (!pl) return ''
      return (showTimestamps && pl.ts ? pl.ts + ' ' : '') + pl.plain
    }).join('\n')
    e.clipboardData.setData('text/plain', text)
    e.preventDefault()
  }

  export function downloadVisible() {
    const text = processedLines.map(l => (l.ts ? l.ts + ' ' : '') + l.plain).join('\n')
    const blob = new Blob([text], { type: 'text/plain' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = `${filename}.log`
    a.click()
    URL.revokeObjectURL(a.href)
  }
</script>

<div class="flex flex-col h-full overflow-hidden">
  <!-- Toolbar -->
  <div class="flex items-center gap-1 px-3 py-1.5 border-b border-border bg-surface shrink-0 flex-wrap">
    <input
      type="text"
      bind:value={searchQuery}
      onkeydown={(e) => e.key === 'Enter' && findNext()}
      placeholder="Search…"
      class="flex-1 min-w-0 text-xs bg-surface-hover border border-border rounded px-2 py-1 focus:outline-none focus:border-accent"
    />
    <button
      onclick={() => (regexSearch = !regexSearch)}
      title="Toggle regex search"
      aria-label="Toggle regex search"
      class="text-xs px-2 py-1 rounded border transition-colors
        {regexSearch ? 'border-accent text-accent bg-accent/10' : 'border-border text-muted hover:text-fg'}"
    >.*</button>
    <button onclick={findPrev} class="text-xs text-muted hover:text-fg px-1.5 py-1" title="Previous match" aria-label="Previous match">↑</button>
    <button onclick={findNext} class="text-xs text-muted hover:text-fg px-1.5 py-1" title="Next match" aria-label="Next match">↓</button>

    <div class="w-px h-4 bg-border mx-0.5"></div>

    <label class="flex items-center gap-1 text-xs text-muted select-none cursor-pointer" title="Highlight error/warn/info/debug lines">
      <input type="checkbox" bind:checked={highlight} class="accent-accent" />
      HL
    </label>
    <label class="flex items-center gap-1 text-xs text-muted select-none cursor-pointer">
      <input type="checkbox" bind:checked={wrap} class="accent-accent" />
      Wrap
    </label>

    {#if eofReached}
      <span class="text-xs text-muted italic">EOF</span>
    {/if}
  </div>

  <!-- Virtual scroll container -->
  <div
    bind:this={scrollEl}
    onscroll={onScroll}
    onwheel={onWheel}
    oncopy={onCopy}
    class="flex-1 overflow-y-auto overflow-x-auto bg-[#1a1a1a] font-mono text-[13px] relative"
    class:overflow-x-hidden={wrap}
  >
    <div class="flex items-center gap-2 px-3 py-1.5 text-xs text-muted border-b border-border">
      {#if eofHistory}
        <span class="italic">Beginning of log</span>
      {:else if historyLoading}
        <svg class="animate-spin h-3 w-3 shrink-0" viewBox="0 0 24 24" fill="none">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"/>
        </svg>
        Loading more logs…
      {:else}
        <span class="italic">Scroll up to load more</span>
      {/if}
    </div>

    <div style:height="{$virtualizerStore.getTotalSize()}px" class="relative">
      {#each $virtualizerStore.getVirtualItems() as row (row.index)}
        <div
          data-index={row.index}
          use:measureEl
          style:position="absolute"
          style:top="0"
          style:left="0"
          style:width="100%"
          style:transform="translateY({row.start}px)"
          class="log-row px-3 py-0 leading-5 {highlight ? levelClass(processedLines[row.index].plain) : ''} {matchIndices.includes(row.index) ? 'search-match' : ''}"
          class:whitespace-pre={!wrap}
          class:whitespace-pre-wrap={wrap}
          class:break-all={wrap}
        >
          {#if showTimestamps && processedLines[row.index].ts}<span class="text-muted mr-2 select-none">{processedLines[row.index].ts}</span>{/if}{@html processedLines[row.index].html}
        </div>
      {/each}
    </div>
  </div>

  <!-- Footer -->
  <div class="flex items-center px-3 py-1 border-t border-border bg-surface shrink-0">
    <span class="text-xs text-muted">{processedLines.length.toLocaleString()} lines{eofReached ? '' : ' (live)'}</span>
  </div>
</div>

<style>
  :global(.ansi-bold) { font-weight: bold; }
  :global(.ansi-italic) { font-style: italic; }
  :global(.ansi-underline) { text-decoration: underline; }

  /* ANSI 16-color classes */
  :global(.ansi-black-fg)   { color: #4e4e4e; }
  :global(.ansi-red-fg)     { color: #e06c75; }
  :global(.ansi-green-fg)   { color: #98c379; }
  :global(.ansi-yellow-fg)  { color: #e5c07b; }
  :global(.ansi-blue-fg)    { color: #61afef; }
  :global(.ansi-magenta-fg) { color: #c678dd; }
  :global(.ansi-cyan-fg)    { color: #56b6c2; }
  :global(.ansi-white-fg)   { color: #abb2bf; }
  :global(.ansi-bright-black-fg)   { color: #5c6370; }
  :global(.ansi-bright-red-fg)     { color: #e06c75; }
  :global(.ansi-bright-green-fg)   { color: #98c379; }
  :global(.ansi-bright-yellow-fg)  { color: #e5c07b; }
  :global(.ansi-bright-blue-fg)    { color: #61afef; }
  :global(.ansi-bright-magenta-fg) { color: #c678dd; }
  :global(.ansi-bright-cyan-fg)    { color: #56b6c2; }
  :global(.ansi-bright-white-fg)   { color: #ffffff; }

  :global(.ansi-black-bg)   { background: #4e4e4e; }
  :global(.ansi-red-bg)     { background: #e06c75; }
  :global(.ansi-green-bg)   { background: #98c379; }
  :global(.ansi-yellow-bg)  { background: #e5c07b; }
  :global(.ansi-blue-bg)    { background: #61afef; }
  :global(.ansi-magenta-bg) { background: #c678dd; }
  :global(.ansi-cyan-bg)    { background: #56b6c2; }
  :global(.ansi-white-bg)   { background: #abb2bf; }

  .log-error { color: #ef4444; }
  .log-warn  { color: #f59e0b; }
  .log-info  { color: #38bdf8; }
  .log-debug { color: #6b7280; }

  .search-match { background: #854d0e55; outline: 1px solid #854d0e; }

  :global(.hl-k)   { color: #61afef; }
  :global(.hl-s)   { color: #98c379; }
  :global(.hl-num) { color: #d19a66; }
  :global(.hl-b)   { color: #c678dd; }
  :global(.hl-n)   { color: #c678dd; }
  :global(.hl-p)   { color: #5c6370; }

</style>
