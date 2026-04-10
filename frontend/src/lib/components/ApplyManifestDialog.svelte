<script lang="ts">
  import { Dialog } from 'bits-ui'
  import { onDestroy } from 'svelte'
  import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter } from '@codemirror/view'
  import { EditorState } from '@codemirror/state'
  import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
  import { yaml as yamlLang } from '@codemirror/lang-yaml'
  import { search, searchKeymap } from '@codemirror/search'
  import { syntaxHighlighting, foldGutter, foldKeymap } from '@codemirror/language'
  import { oneDarkHighlightStyle } from '@codemirror/theme-one-dark'
  import * as AppService from '../../../bindings/github.com/Vilsol/klados/internal/services/appservice.js'
  import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import { notificationStore } from '$lib/stores/notification.svelte'

  type ApplyResult = {
    gvr: string
    namespace: string
    name: string
    action: string
    error: string
  }

  let {
    open = $bindable(false),
    ctxName,
  }: {
    open: boolean
    ctxName: string
  } = $props()

  let container: HTMLDivElement | undefined = $state()
  let view: EditorView | undefined
  let applying = $state(false)
  let results = $state<ApplyResult[]>([])
  let hasApplied = $state(false)
  let editorContent = $state('')

  const editorTheme = EditorView.theme({
    '&': { height: '100%', fontSize: '12.5px', backgroundColor: 'var(--color-bg)', color: 'var(--color-fg)' },
    '.cm-content': { padding: '4px 0', fontFamily: '"JetBrains Mono", "Fira Code", ui-monospace, monospace', caretColor: 'var(--color-accent)' },
    '.cm-gutters': { backgroundColor: 'var(--color-surface)', color: 'var(--color-muted)', borderRight: '1px solid var(--color-border)', minWidth: '3rem' },
    '.cm-scroller': { overflow: 'auto', lineHeight: '1.6' },
  })

  function initEditor(doc: string = '') {
    view?.destroy()
    view = new EditorView({
      state: EditorState.create({
        doc,
        extensions: [
          lineNumbers(), highlightActiveLine(), highlightActiveLineGutter(), foldGutter(),
          history(), syntaxHighlighting(oneDarkHighlightStyle), yamlLang(),
          search({ top: true }),
          keymap.of([...defaultKeymap, ...historyKeymap, ...searchKeymap, ...foldKeymap]),
          EditorView.lineWrapping,
          editorTheme,
          EditorView.updateListener.of((update) => {
            if (update.docChanged) editorContent = update.state.doc.toString()
          }),
        ],
      }),
      parent: container!,
    })
  }

  onDestroy(() => view?.destroy())

  $effect(() => {
    if (open && container && !view) {
      initEditor()
    }
    if (!open) {
      view?.destroy()
      view = undefined
      results = []
      hasApplied = false
      editorContent = ''
    }
  })

  function loadContent(text: string) {
    editorContent = text
    if (view) {
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } })
    } else if (container) {
      initEditor(text)
    }
  }

  async function openFile() {
    try {
      const content = await AppService.BrowseManifestFile()
      if (content) loadContent(content)
    } catch (e: any) {
      notificationStore.push(e?.message ?? 'Could not open file', 'error')
    }
  }

  async function pasteFromClipboard() {
    try {
      const text = await navigator.clipboard.readText()
      if (text.trim()) loadContent(text)
    } catch {
      notificationStore.push('Could not read clipboard', 'error')
    }
  }

  const docCount = $derived(
    editorContent.trim()
      ? editorContent.split('---').filter((s) => s.trim() && !s.trim().startsWith('#')).length
      : 0
  )

  const editorEmpty = $derived(!editorContent.trim())

  async function applyManifest() {
    if (!view) return
    applying = true
    hasApplied = false
    try {
      const yaml = view.state.doc.toString()
      const res = await ResourceService.ApplyManifest(ctxName, yaml)
      results = (res ?? []) as ApplyResult[]
      hasApplied = true
    } catch (e: any) {
      notificationStore.push(e?.message ?? 'Apply failed', 'error')
    } finally {
      applying = false
    }
  }

  function actionClass(action: string, error: string): string {
    if (error) return 'text-destructive'
    if (action === 'created') return 'text-green-400'
    if (action === 'configured') return 'text-blue-400'
    return 'text-muted'
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content
      class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl flex flex-col"
      style="width: min(800px, 92vw); height: min(700px, 90vh);"
    >
      <div class="flex items-center gap-2 px-4 py-3 border-b border-border shrink-0">
        <Dialog.Title class="text-sm font-semibold flex-1">Apply Manifest</Dialog.Title>
        <button
          onclick={openFile}
          class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
        >Open File…</button>
        <button
          onclick={pasteFromClipboard}
          class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
        >Paste from Clipboard</button>
        <Dialog.Close
          class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
        >Cancel</Dialog.Close>
        <button
          onclick={applyManifest}
          disabled={applying || editorEmpty}
          class="text-xs px-2.5 py-1 rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity disabled:opacity-50"
        >{applying ? 'Applying…' : docCount > 0 ? `Apply (${docCount} resource${docCount === 1 ? '' : 's'})` : 'Apply'}</button>
      </div>

      <div bind:this={container} class="flex-1 overflow-hidden min-h-0"></div>

      {#if hasApplied}
        <div class="border-t border-border shrink-0 max-h-48 overflow-y-auto px-4 py-2 flex flex-col gap-1">
          {#each results as r}
            <div class="flex items-start gap-2 text-xs font-mono">
              <span class="text-muted shrink-0">{r.gvr}/{r.namespace || '—'}/{r.name || '—'}</span>
              {#if r.error}
                <span class="text-destructive break-all">{r.error}</span>
              {:else}
                <span class={actionClass(r.action, r.error)}>{r.action}</span>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
