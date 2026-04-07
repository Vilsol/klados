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
  import { parse } from 'yaml'
  import * as ResourceService from '../../../bindings/github.com/Vilsol/klados/internal/services/resourceservice.js'
  import { notificationStore } from '$lib/stores/notification.svelte'

  let {
    open = $bindable(false),
    ctxName,
    gvr,
    defaultNamespace = 'default',
    onsuccess,
  }: {
    open: boolean
    ctxName: string
    gvr: string
    defaultNamespace?: string
    onsuccess?: () => void
  } = $props()

  let container: HTMLDivElement | undefined = $state()
  let view: EditorView | undefined
  let saving = $state(false)

  // svelte-ignore state_referenced_locally
  const TEMPLATE = `apiVersion: ""
kind: ""
metadata:
  name: ""
  namespace: "${defaultNamespace}"
spec: {}
`

  const editorTheme = EditorView.theme({
    '&': { height: '100%', fontSize: '12.5px', backgroundColor: 'var(--color-bg)', color: 'var(--color-fg)' },
    '.cm-content': { padding: '4px 0', fontFamily: '"JetBrains Mono", "Fira Code", ui-monospace, monospace', caretColor: 'var(--color-accent)' },
    '.cm-gutters': { backgroundColor: 'var(--color-surface)', color: 'var(--color-muted)', borderRight: '1px solid var(--color-border)', minWidth: '3rem' },
    '.cm-scroller': { overflow: 'auto', lineHeight: '1.6' },
  })

  function initEditor(doc: string) {
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
        ],
      }),
      parent: container,
    })
  }

  onDestroy(() => view?.destroy())

  $effect(() => {
    if (open && container && !view) initEditor(TEMPLATE)
  })

  async function importFromClipboard() {
    try {
      const text = await navigator.clipboard.readText()
      if (text.trim()) {
        view?.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } })
      }
    } catch {
      notificationStore.push('Could not read clipboard', 'error')
    }
  }

  async function apply() {
    if (!view) return
    saving = true
    try {
      const yamlText = view.state.doc.toString()
      const parsed = parse(yamlText) as Record<string, any>
      if (!parsed) {
        notificationStore.push('Invalid YAML', 'error')
        return
      }
      const ns = parsed.metadata?.namespace || defaultNamespace
      await ResourceService.CreateResource(ctxName, gvr, ns, parsed)
      notificationStore.push(`Created ${parsed.metadata?.name ?? 'resource'}`, 'success')
      open = false
      onsuccess?.()
    } catch (e: any) {
      notificationStore.push(e?.message ?? 'Create failed', 'error')
    } finally {
      saving = false
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="fixed inset-0 bg-black/50 z-40" />
    <Dialog.Content
      class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 bg-surface border border-border rounded-lg shadow-xl flex flex-col"
      style="width: min(800px, 92vw); height: min(600px, 85vh);"
    >
      <div class="flex items-center gap-2 px-4 py-3 border-b border-border shrink-0">
        <Dialog.Title class="text-sm font-semibold flex-1">Create Resource</Dialog.Title>
        <button
          onclick={importFromClipboard}
          class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
        >Import from Clipboard</button>
        <Dialog.Close
          class="text-xs px-2.5 py-1 rounded border border-border hover:bg-surface-hover transition-colors"
        >Cancel</Dialog.Close>
        <button
          onclick={apply}
          disabled={saving}
          class="text-xs px-2.5 py-1 rounded bg-accent text-accent-fg hover:opacity-90 transition-opacity disabled:opacity-50"
        >{saving ? 'Creating…' : 'Create'}</button>
      </div>
      <div bind:this={container} class="flex-1 overflow-hidden"></div>
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
