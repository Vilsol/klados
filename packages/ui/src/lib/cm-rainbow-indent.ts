import { EditorView, ViewPlugin, Decoration, type DecorationSet, type ViewUpdate } from '@codemirror/view'
import { RangeSetBuilder } from '@codemirror/state'

const INDENT_UNIT = 2

const COLORS = [
  'rgba(45,212,191,0.3)',
  'rgba(96,165,250,0.3)',
  'rgba(167,139,250,0.3)',
  'rgba(244,114,182,0.3)',
  'rgba(251,146,60,0.3)',
  'rgba(250,204,21,0.3)',
]

const BG_COLORS = [
  'rgba(45,212,191,0.06)',
  'rgba(96,165,250,0.06)',
  'rgba(167,139,250,0.06)',
  'rgba(244,114,182,0.06)',
  'rgba(251,146,60,0.06)',
  'rgba(250,204,21,0.06)',
]

// Width of each guide line, in px
const WIDTH_PX = 2

export const rainbowIndentTheme = EditorView.baseTheme({
  '.cm-line': { position: 'relative' },
  '.cm-line.cm-ril::before': {
    content: '""',
    position: 'absolute',
    left: '2px',
    top: '0',
    bottom: '0',
    width: '100%',
    backgroundImage: 'var(--ril-bg), var(--ril-fill)',
    backgroundRepeat: 'no-repeat',
    pointerEvents: 'none',
    zIndex: '0',
  },
})

// Build a non-repeating gradient for a line with the given indent level.
function buildBgGradient(level: number): string {
  const stops: string[] = []
  for (let k = 0; k < level; k++) {
    const bg = BG_COLORS[k % BG_COLORS.length]
    const start = k * INDENT_UNIT
    const end = (k + 1) * INDENT_UNIT
    stops.push(`${bg} ${start}ch`, `${bg} ${end}ch`)
  }
  stops.push(`transparent ${level * INDENT_UNIT}ch`)
  return `linear-gradient(to right, ${stops.join(', ')})`
}

function buildLineGradient(level: number): string {
  const stops: string[] = []
  for (let k = 0; k < level; k++) {
    const color = COLORS[k % COLORS.length]
    const pos = (k * INDENT_UNIT + 0.9).toFixed(1)
    stops.push(
      `transparent ${pos}ch`,
      `${color} ${pos}ch`,
      `${color} calc(${pos}ch + ${WIDTH_PX}px)`,
      `transparent calc(${pos}ch + ${WIDTH_PX}px)`,
    )
  }
  return `linear-gradient(to right, ${stops.join(', ')})`
}

// Pre-compute gradients for common indent depths
const gradientCache = new Map<number, { lines: string; bg: string }>()
function getGradients(level: number): { lines: string; bg: string } {
  let g = gradientCache.get(level)
  if (!g) {
    g = { lines: buildLineGradient(level), bg: buildBgGradient(level) }
    gradientCache.set(level, g)
  }
  return g
}

class RainbowIndentPlugin {
  decorations: DecorationSet

  constructor(view: EditorView) {
    this.decorations = this.build(view)
  }

  update(update: ViewUpdate) {
    if (update.docChanged || update.viewportChanged) {
      this.decorations = this.build(update.view)
    }
  }

  build(view: EditorView): DecorationSet {
    const builder = new RangeSetBuilder<Decoration>()
    const { from, to } = view.viewport

    const lines: { lineFrom: number; level: number; empty: boolean }[] = []
    let pos = view.state.doc.lineAt(from).from
    const end = view.state.doc.lineAt(to).to

    while (pos <= end) {
      const line = view.state.doc.lineAt(pos)
      const text = line.text
      const empty = text.trim().length === 0
      const leading = empty ? 0 : text.length - text.trimStart().length
      lines.push({ lineFrom: line.from, level: Math.floor(leading / INDENT_UNIT), empty })
      if (line.to >= view.state.doc.length) break
      pos = line.to + 1
    }

    // Two-pass empty-line resolution: min of nearest non-empty above and below
    const resolved = lines.map((l) => l.level)
    for (let i = 0; i < lines.length; i++) {
      if (!lines[i].empty) continue
      let above = 0
      for (let a = i - 1; a >= 0; a--) {
        if (!lines[a].empty) { above = lines[a].level; break }
      }
      let below = 0
      for (let b = i + 1; b < lines.length; b++) {
        if (!lines[b].empty) { below = lines[b].level; break }
      }
      resolved[i] = Math.min(above, below)
    }

    for (let i = 0; i < lines.length; i++) {
      const level = resolved[i]
      if (level <= 0) continue
      builder.add(
        lines[i].lineFrom,
        lines[i].lineFrom,
        Decoration.line({ class: 'cm-ril', attributes: { style: `--ril-bg: ${getGradients(level).lines}; --ril-fill: ${getGradients(level).bg}` } }),
      )
    }

    return builder.finish()
  }
}

export const rainbowIndent = ViewPlugin.fromClass(RainbowIndentPlugin, {
  decorations: (v) => v.decorations,
})
