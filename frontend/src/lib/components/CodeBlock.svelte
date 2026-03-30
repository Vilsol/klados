<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { EditorView } from '@codemirror/view'
  import { EditorState } from '@codemirror/state'
  import { syntaxHighlighting, StreamLanguage } from '@codemirror/language'
  import { oneDarkHighlightStyle } from '@codemirror/theme-one-dark'
  import { yaml as yamlLang } from '@codemirror/lang-yaml'
  import { json as jsonLang } from '@codemirror/lang-json'
  import { toml } from '@codemirror/legacy-modes/mode/toml'
  import { shell } from '@codemirror/legacy-modes/mode/shell'

  let { value, lang }: { value: string; lang: 'yaml' | 'json' | 'toml' | 'shell' | 'plain' } = $props()

  let container: HTMLDivElement
  let view: EditorView

  const theme = EditorView.theme({
    '&': { background: 'transparent', fontSize: '0.75rem', maxHeight: '16rem' },
    '.cm-scroller': { overflow: 'auto', fontFamily: 'inherit', lineHeight: '1.6' },
    '.cm-content': { padding: '0.5rem 0.75rem', caretColor: 'transparent' },
    '.cm-line': { padding: '0' },
    '.cm-cursor': { display: 'none' },
    '&.cm-focused': { outline: 'none' },
  })

  function langExtension() {
    if (lang === 'yaml') return yamlLang()
    if (lang === 'json') return jsonLang()
    if (lang === 'toml') return StreamLanguage.define(toml)
    if (lang === 'shell') return StreamLanguage.define(shell)
    return []
  }

  function makeState() {
    return EditorState.create({
      doc: value,
      extensions: [
        EditorState.readOnly.of(true),
        EditorView.editable.of(false),
        syntaxHighlighting(oneDarkHighlightStyle),
        langExtension(),
        theme,
      ],
    })
  }

  onMount(() => {
    view = new EditorView({ state: makeState(), parent: container })
  })

  $effect(() => {
    // Re-run whenever value or lang changes; noop until view is mounted
    value; lang
    if (view) view.setState(makeState())
  })

  onDestroy(() => view?.destroy())
</script>

<div bind:this={container} class="font-mono"></div>
