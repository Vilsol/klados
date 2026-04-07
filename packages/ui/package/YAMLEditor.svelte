<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { Dialog } from 'bits-ui'
  import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter, tooltips } from '@codemirror/view'
  import { EditorState, StateEffect, Compartment } from '@codemirror/state'
  import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
  import { yaml as yamlLang } from '@codemirror/lang-yaml'
  import { searchKeymap, search } from '@codemirror/search'
  import { syntaxHighlighting, foldGutter, foldKeymap } from '@codemirror/language'
  import { oneDarkHighlightStyle } from '@codemirror/theme-one-dark'
  import { yamlSchema } from 'codemirror-json-schema/yaml'
  import { stringify, parse } from 'yaml'
  import DiffView from './DiffView.svelte'
  import { rainbowIndent, rainbowIndentTheme } from './cm-rainbow-indent'

  let {
    obj = $bindable(),
    ctxName,
    gvr,
    namespace,
    name,
    kind = '',
    onrefresh,
    onSave,
    onGetResource,
    onGetSchema,
    onNotify,
    onSetEditorMode,
    onOpenUrl,
  }: {
    obj: Record<string, any>
    ctxName: string
    gvr: string
    namespace: string
    name: string
    kind?: string
    onrefresh?: () => void
    onSave?: (ctx: string, gvr: string, ns: string, parsed: Record<string, any>) => Promise<Record<string, any> | null>
    onGetResource?: (ctx: string, gvr: string, ns: string, name: string) => Promise<Record<string, any> | null>
    onGetSchema?: (ctx: string, gvr: string, kind: string) => Promise<Record<string, any>>
    onNotify?: (msg: string, type: 'info' | 'success' | 'error') => void
    onSetEditorMode?: (mode: string) => void
    onOpenUrl?: (url: string) => void
  } = $props()

  let container: HTMLDivElement
  let view = $state.raw<EditorView | undefined>(undefined)
  let editing = $state(false)
  let saving = $state(false)
  let diffOpen = $state(false)
  let diffMode = $state<'split' | 'unified'>('split')
  let originalYaml = $state.raw('')
  let showIndentGuides = $state(true)
  let lineWrapping = $state(false)
  const rainbowCompartment = new Compartment()
  const wrapCompartment = new Compartment()
  let conflictError = $state<string | null>(null)
  let schemaLoaded = $state(false)
  let schema: Record<string, any> | null = null
  let hideManagedFields = $state(true)

  function makeYaml(o: Record<string, any>): string {
    const stripped = JSON.parse(JSON.stringify(o))
    if (hideManagedFields && stripped.metadata?.managedFields) delete stripped.metadata.managedFields
    return stringify(stripped, { indent: 2, indentSeq: true })
  }

  // yamlContent tracks both obj (live updates) and hideManagedFields (toggle)
  const yamlContent = $derived(makeYaml(obj))

  // Sync editor document whenever yamlContent changes.
  // yamlContent is read unconditionally so it's always tracked — the dispatch
  // only fires after onMount has set view.
  $effect(() => {
    const content = yamlContent
    if (!editing && view) view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: content } })
  })

  $effect(() => {
    const exts = showIndentGuides ? [rainbowIndentTheme, rainbowIndent] : []
    view?.dispatch({ effects: rainbowCompartment.reconfigure(exts) })
  })

  $effect(() => {
    const ext = lineWrapping ? [EditorView.lineWrapping] : []
    view?.dispatch({ effects: wrapCompartment.reconfigure(ext) })
  })


  const editorTheme = EditorView.theme({
    '&': { height: '100%', fontSize: '12.5px', backgroundColor: 'var(--color-bg)', color: 'var(--color-fg)' },
    '.cm-content': { padding: '4px 0', fontFamily: '"JetBrains Mono", "Fira Code", ui-monospace, monospace', caretColor: 'var(--color-accent)' },
    '.cm-gutters': { backgroundColor: 'var(--color-surface)', color: 'var(--color-muted)', borderRight: '1px solid var(--color-border)', minWidth: '3rem' },
    '.cm-lineNumbers .cm-gutterElement': { padding: '0 8px', minWidth: '2.5rem' },
    '.cm-foldGutter .cm-gutterElement': { padding: '0 2px', cursor: 'pointer' },
    '.cm-activeLineGutter': { backgroundColor: 'var(--color-surface-hover)', color: 'var(--color-fg)' },
    '.cm-activeLine': { backgroundColor: 'color-mix(in srgb, var(--color-surface-hover) 60%, transparent)' },
    '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--color-accent)' },
    '.cm-selectionBackground': { backgroundColor: 'color-mix(in srgb, var(--color-accent) 20%, transparent)' },
    '&.cm-focused .cm-selectionBackground': { backgroundColor: 'color-mix(in srgb, var(--color-accent) 30%, transparent)' },
    '.cm-foldPlaceholder': { backgroundColor: 'var(--color-surface)', border: '1px solid var(--color-border)', color: 'var(--color-muted)', borderRadius: '3px', padding: '0 4px' },
    '.cm-scroller': { overflow: 'auto', lineHeight: '1.6' },
    '.cm-searchMatch': { backgroundColor: 'color-mix(in srgb, var(--color-accent) 25%, transparent)', outline: '1px solid color-mix(in srgb, var(--color-accent) 50%, transparent)' },
    '.cm-searchMatch.cm-searchMatch-selected': { backgroundColor: 'color-mix(in srgb, var(--color-accent) 50%, transparent)' },
    '.cm-panels': { backgroundColor: 'var(--color-surface)', borderTop: '1px solid var(--color-border)', color: 'var(--color-fg)' },
    '.cm-panel input': { backgroundColor: 'var(--color-bg)', border: '1px solid var(--color-border)', color: 'var(--color-fg)', borderRadius: '3px', padding: '1px 4px' },
    '.cm-panel button': { backgroundColor: 'var(--color-surface)', border: '1px solid var(--color-border)', color: 'var(--color-fg)', borderRadius: '3px', cursor: 'pointer' },
    '.cm-tooltip': { backgroundColor: 'var(--color-surface)', border: '1px solid var(--color-border)', color: 'var(--color-fg)', borderRadius: '4px', zIndex: '9999' },
    '.cm-tooltip-hover': { backgroundColor: 'var(--color-surface)', border: '1px solid var(--color-border)', color: 'var(--color-fg)', borderRadius: '4px', zIndex: '9999' },
    '.cm-tooltip .cm-tooltip-section:not(:first-child)': { borderTop: '1px solid var(--color-border)' },
  })

  function baseExtensions(withSchema: Record<string, any> | null) {
    return [
      lineNumbers(),
      highlightActiveLine(),
      highlightActiveLineGutter(),
      foldGutter(),
      history(),
      syntaxHighlighting(oneDarkHighlightStyle),
      withSchema ? safeSchemaExtensions(withSchema) : yamlLang(),
      search({ top: true }),
      keymap.of([...defaultKeymap, ...historyKeymap, ...searchKeymap, ...foldKeymap]),
      wrapCompartment.of(lineWrapping ? [EditorView.lineWrapping] : []),
      tooltips({ parent: document.body }),
      editorTheme,
      rainbowCompartment.of([rainbowIndentTheme, rainbowIndent]),
    ]
  }

  function safeSchemaExtensions(s: Record<string, any>) {
    try {
      return yamlSchema(s as any)
    } catch {
      return yamlLang()
    }
  }

  function initEditor(content: string, readonly: boolean) {
    const extensions = [
      ...baseExtensions(schema ?? null),
      EditorView.editable.of(!readonly),
    ]
    // Create new editor before destroying old one so a failed init leaves content intact
    try {
      const newView = new EditorView({
        state: EditorState.create({ doc: content, extensions }),
        parent: container,
      })
      view?.destroy()
      view = newView
    } catch (e) {
      console.error('YAMLEditor init failed:', e)
    }
  }

  function handleDocClick(e: MouseEvent) {
    const a = (e.target as Element).closest('a')
    if (a?.href) {
      e.preventDefault()
      onOpenUrl?.(a.href)
    }
  }

  onMount(async () => {
    container.addEventListener('focusin', () => onSetEditorMode?.('editor'))
    container.addEventListener('focusout', (e) => {
      if (!container.contains(e.relatedTarget as Node)) onSetEditorMode?.('normal')
    })

    initEditor(makeYaml(obj), true)

    // Load schema in background and append extensions to the live editor without rebuilding
    if (ctxName && gvr && kind) {
      try {
        const s = await onGetSchema?.(ctxName, gvr, kind)
        if (s && Object.keys(s).length > 0 && view) {
          schema = s
          const exts = safeSchemaExtensions(s)
          if (!Array.isArray(exts) || exts.length > 0) {
            try {
              view.dispatch({ effects: StateEffect.appendConfig.of(exts) })
              schemaLoaded = true
            } catch {
              // Schema extensions incompatible — editor still works
            }
          }
        }
      } catch {
        // Schema is optional
      }
    }
  })

  onMount(() => document.addEventListener('click', handleDocClick))
  onDestroy(() => { view?.destroy(); document.removeEventListener('click', handleDocClick) })

  function startEdit() {
    conflictError = null
    editing = true
    initEditor(makeYaml(obj), false)
  }

  function cancelEdit() {
    editing = false
    conflictError = null
    initEditor(makeYaml(obj), true)
  }

  async function save() {
    if (!view) return
    saving = true
    conflictError = null
    try {
      const yamlText = view.state.doc.toString()
      const parsed = parse(yamlText) as Record<string, any>
      const result = await onSave?.(ctxName, gvr, namespace, parsed)
      if (result) {
        obj = result
        editing = false
        initEditor(makeYaml(result), true)
        onNotify?.('Changes applied.', 'success')
      }
    } catch (e: any) {
      const msg: string = e?.message ?? String(e)
      if (msg.includes('409') || msg.toLowerCase().includes('conflict')) {
        await handleConflict()
      } else {
        onNotify?.(msg, 'error')
      }
    } finally {
      saving = false
    }
  }

  async function handleConflict() {
    try {
      const latest = await onGetResource?.(ctxName, gvr, namespace, name)
      if (latest) {
        const changed = (['spec', 'metadata', 'status'] as const).filter(
          (k) => JSON.stringify(obj[k]) !== JSON.stringify(latest[k])
        )
        const serverRV = latest.metadata?.resourceVersion ?? '?'
        const ourRV = obj.metadata?.resourceVersion ?? '?'
        conflictError = `Conflict: resourceVersion ${ourRV} → ${serverRV}.`
        if (changed.length > 0) conflictError += ` Changed: ${changed.join(', ')}.`
      }
    } catch {
      conflictError = 'Conflict: resource was modified on the server.'
    }
  }

  function refresh() {
    conflictError = null
    cancelEdit()
    onrefresh?.()
  }

  function formatYaml() {
    if (!view) return
    try {
      const parsed = parse(view.state.doc.toString())
      const formatted = stringify(parsed, { indent: 2, indentSeq: true })
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: formatted } })
    } catch {
      onNotify?.('Invalid YAML — cannot format', 'error')
    }
  }

  function openDiff() {
    originalYaml = makeYaml(obj)
    diffOpen = true
  }

  function exportYaml() {
    const blob = new Blob([makeYaml(obj)], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${name}.yaml`
    a.click()
    URL.revokeObjectURL(url)
  }

  function copyYaml() {
    navigator.clipboard.writeText(makeYaml(obj)).then(() => {
      onNotify?.('Copied YAML to clipboard', 'success')
    })
  }
</script>

<div class="flex flex-col h-full">
  <div class="flex items-center gap-2 px-3 py-1.5 border-b border-border bg-surface shrink-0 flex-wrap">
    {#if !editing}
      <button
        onclick={startEdit}
        class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
      >Edit</button>
    {:else}
      <button
        onclick={save}
        disabled={saving}
        class="text-xs px-2.5 py-1 rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity disabled:opacity-50"
      >{saving ? 'Saving…' : 'Save'}</button>
      <button
        onclick={cancelEdit}
        class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
      >Cancel</button>
      <button
        onclick={formatYaml}
        class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
      >Format</button>
      <button
        onclick={openDiff}
        class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
      >Review Changes</button>
    {/if}

    {#if conflictError}
      <span class="text-xs text-destructive flex-1 min-w-0 truncate">{conflictError}</span>
      <button
        onclick={refresh}
        class="text-xs px-2.5 py-1 rounded border border-destructive text-destructive hover:bg-destructive/10 transition-colors shrink-0"
      >Refresh</button>
    {/if}

    <label class="flex items-center gap-1.5 text-xs text-muted cursor-pointer select-none shrink-0 ml-auto">
      <input type="checkbox" bind:checked={hideManagedFields} class="accent-accent" />
      Hide managedFields
    </label>
    <label class="flex items-center gap-1.5 text-xs text-muted cursor-pointer select-none shrink-0">
      <input type="checkbox" bind:checked={showIndentGuides} class="accent-accent" />
      Indent guides
    </label>
    <label class="flex items-center gap-1.5 text-xs text-muted cursor-pointer select-none shrink-0">
      <input type="checkbox" bind:checked={lineWrapping} class="accent-accent" />
      Word wrap
    </label>

    <button
      onclick={exportYaml}
      class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors shrink-0"
    >Export</button>
    <button
      onclick={copyYaml}
      class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors shrink-0"
    >Copy</button>

    {#if schemaLoaded && !conflictError}
      <span class="text-xs text-muted">Schema active</span>
    {/if}
  </div>

  <div bind:this={container} class="flex-1 overflow-hidden"></div>
</div>

<Dialog.Root bind:open={diffOpen}>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl p-4 w-auto max-w-[95vw]">
      <div class="flex items-center gap-2 mb-3">
        <span class="text-sm font-semibold flex-1">Review Changes</span>
        <button
          onclick={() => { diffMode = 'split' }}
          class="text-xs px-2.5 py-1 rounded border transition-colors {diffMode === 'split' ? 'bg-accent text-accent-fg border-accent' : 'border-border hover:bg-surface-hover'}"
        >Side by side</button>
        <button
          onclick={() => { diffMode = 'unified' }}
          class="text-xs px-2.5 py-1 rounded border transition-colors {diffMode === 'unified' ? 'bg-accent text-accent-fg border-accent' : 'border-border hover:bg-surface-hover'}"
        >Unified</button>
      </div>
      {#if diffOpen}
        <DiffView original={originalYaml} modified={view?.state.doc.toString() ?? ''} mode={diffMode} />
      {/if}
      <div class="flex items-center gap-2 mt-3 justify-end">
        <button
          onclick={() => { diffOpen = false }}
          class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
        >Back to Edit</button>
        <button
          onclick={async () => { await save(); diffOpen = false }}
          class="text-xs px-2.5 py-1 rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity"
        >Apply</button>
      </div>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
