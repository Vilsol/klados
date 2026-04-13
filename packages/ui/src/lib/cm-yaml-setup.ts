import {
  EditorView,
  keymap,
  lineNumbers,
  highlightActiveLine,
  highlightActiveLineGutter,
  tooltips,
} from '@codemirror/view'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { yaml as yamlLang } from '@codemirror/lang-yaml'
import { searchKeymap, search } from '@codemirror/search'
import { syntaxHighlighting, foldGutter, foldKeymap } from '@codemirror/language'
import { oneDarkHighlightStyle } from '@codemirror/theme-one-dark'
import { autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import type { Extension } from '@codemirror/state'
import { rainbowIndent, rainbowIndentTheme } from './cm-rainbow-indent'

export const cmEditorTheme = EditorView.theme({
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
  '.cm-tooltip.cm-tooltip-autocomplete': { backgroundColor: 'var(--color-surface)', border: '1px solid var(--color-border)', borderRadius: '4px', zIndex: '9999' },
  '.cm-tooltip-autocomplete ul li': { padding: '2px 8px' },
  '.cm-tooltip-autocomplete ul li[aria-selected]': { backgroundColor: 'color-mix(in srgb, var(--color-accent) 25%, transparent)', color: 'var(--color-fg)' },
  '.cm-completionLabel': { fontSize: '12.5px' },
  '.cm-completionDetail': { fontSize: '11px', color: 'var(--color-muted)', fontStyle: 'normal', marginLeft: '8px' },
})

export function cmYamlExtensions(opts?: {
  lang?: Extension
  lineWrapping?: boolean
  rainbowIndent?: boolean
}): Extension[] {
  const exts: Extension[] = [
    lineNumbers(),
    highlightActiveLine(),
    highlightActiveLineGutter(),
    foldGutter(),
    history(),
    closeBrackets(),
    autocompletion(),
    syntaxHighlighting(oneDarkHighlightStyle),
    opts?.lang ?? yamlLang(),
    search({ top: true }),
    keymap.of([...closeBracketsKeymap, ...completionKeymap, indentWithTab, ...defaultKeymap, ...historyKeymap, ...searchKeymap, ...foldKeymap]),
    tooltips({ parent: document.body }),
    cmEditorTheme,
  ]
  if (opts?.lineWrapping !== false) {
    exts.push(EditorView.lineWrapping)
  }
  if (opts?.rainbowIndent !== false) {
    exts.push(rainbowIndentTheme, rainbowIndent)
  }
  return exts
}
