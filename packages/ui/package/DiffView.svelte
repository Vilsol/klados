<script lang="ts">
  import { onDestroy } from 'svelte'
  import { MergeView, unifiedMergeView } from '@codemirror/merge'
  import { EditorView } from '@codemirror/view'
  import { EditorState } from '@codemirror/state'
  import { syntaxHighlighting } from '@codemirror/language'
  import { yaml as yamlLang } from '@codemirror/lang-yaml'
  import { oneDarkHighlightStyle } from '@codemirror/theme-one-dark'

  let {
    original,
    modified,
    mode,
  }: {
    original: string
    modified: string
    mode: 'split' | 'unified'
  } = $props()

  let container: HTMLDivElement
  let view: MergeView | EditorView | undefined

  const sharedExts = [
    EditorView.editable.of(false),
    EditorView.lineWrapping,
    yamlLang(),
    syntaxHighlighting(oneDarkHighlightStyle),
  ]

  $effect(() => {
    if (!container) return
    const _original = original
    const _modified = modified
    const _mode = mode

    if (view instanceof MergeView) {
      view.destroy()
    } else {
      view?.destroy()
    }
    view = undefined

    if (_mode === 'split') {
      view = new MergeView({
        parent: container,
        a: { doc: _original, extensions: sharedExts },
        b: { doc: _modified, extensions: sharedExts },
      })
    } else {
      view = new EditorView({
        parent: container,
        state: EditorState.create({
          doc: _modified,
          extensions: [
            ...sharedExts,
            unifiedMergeView({ original: _original, mergeControls: false }),
          ],
        }),
      })
    }
  })

  onDestroy(() => {
    if (view instanceof MergeView) {
      view.destroy()
    } else {
      view?.destroy()
    }
  })
</script>

<div
  bind:this={container}
  class={mode === 'split' ? 'w-[88vw] h-[70vh] overflow-auto' : 'w-[60vw] h-[70vh] overflow-auto'}
></div>
